package main

// Nhóm chat (session group) mà extension Claude Code tạo trong VS Code. Nhóm không nằm
// trong file .jsonl mà trong globalState của extension, tức bảng ItemTable của file
// state.vscdb (SQLite) trong thư mục dữ liệu VS Code, dưới các khóa
// "sessionGroups:<thư mục dự án>" = [{id, name, collapsed, sessionIds}].

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Thư mục dữ liệu của VS Code và các bản dựng từ VS Code.
var vscodeAppNames = []string{"Code", "Code - Insiders", "VSCodium", "Code - OSS", "Cursor", "Windsurf"}

const groupsKeyPrefix = "sessionGroups:"

// vscodeUserDataDir chuẩn hóa một đường dẫn người dùng đưa vào (thư mục cài VS Code
// portable, thư mục data, user-data, hoặc User / User\globalStorage bên trong) thành
// thư mục user-data (chứa User\globalStorage); "" nếu không phải thư mục dữ liệu VS Code.
func vscodeUserDataDir(p string) string {
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	cands := []string{p, filepath.Join(p, "user-data"), filepath.Join(p, "data", "user-data")}
	switch strings.ToLower(filepath.Base(p)) {
	case "user":
		cands = append(cands, filepath.Dir(p))
	case "globalstorage":
		cands = append(cands, filepath.Dir(filepath.Dir(p)))
	}
	for _, c := range cands {
		if st, err := os.Stat(filepath.Join(c, "User", "globalStorage")); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

// vscodeStateDBs liệt kê các file state.vscdb toàn cục (mọi profile) của VS Code ở vị trí
// chuẩn, ở bản portable (biến VSCODE_PORTABLE) và ở các đường dẫn thêm trong extra
// (file state.vscdb hoặc thư mục, xem vscodeUserDataDir). Không trùng lặp.
func vscodeStateDBs(extra []string) []string {
	var userDirs []string
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, "Library", "Application Support")
		}
	default:
		base, _ = os.UserConfigDir()
	}
	if base != "" {
		for _, n := range vscodeAppNames {
			userDirs = append(userDirs, filepath.Join(base, n))
		}
	}
	if p := os.Getenv("VSCODE_PORTABLE"); p != "" {
		userDirs = append(userDirs, filepath.Join(p, "user-data"))
	}

	var dbs []string
	seen := map[string]bool{}
	add := func(p string) {
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			return
		}
		if k := stateKey(p); !seen[k] {
			seen[k] = true
			dbs = append(dbs, p)
		}
	}
	for _, e := range extra {
		if strings.EqualFold(filepath.Ext(e), ".vscdb") {
			add(e)
		} else if d := vscodeUserDataDir(e); d != "" {
			userDirs = append(userDirs, d)
		}
	}
	for _, d := range userDirs {
		add(filepath.Join(d, "User", "globalStorage", "state.vscdb"))
		profiles, _ := filepath.Glob(filepath.Join(d, "User", "profiles", "*", "globalStorage", "state.vscdb"))
		for _, p := range profiles {
			add(p)
		}
	}
	return dbs
}

// sessionGroups là kết quả đọc nhóm: mã session (viết thường) → tên nhóm.
type sessionGroups struct {
	bySession map[string]string
	groups    int // số nhóm đọc được
	dbs       int // số file state.vscdb có dữ liệu nhóm
}

func (g *sessionGroups) of(session string) string {
	if g == nil {
		return ""
	}
	return g.bySession[strings.ToLower(session)]
}

// loadSessionGroups đọc nhóm từ mọi state.vscdb tìm được. File sửa gần đây hơn được ưu
// tiên khi một chat nằm trong nhóm ở nhiều profile. File lỗi chỉ bị bỏ qua.
func loadSessionGroups(extra []string) *sessionGroups {
	dbs := vscodeStateDBs(extra)
	mod := map[string]int64{}
	for _, p := range dbs {
		_, mod[p], _ = fileSig(p)
	}
	sort.SliceStable(dbs, func(i, j int) bool { return mod[dbs[i]] < mod[dbs[j]] })

	g := &sessionGroups{bySession: map[string]string{}}
	for _, p := range dbs {
		n, err := readGroupsFromDB(p, g.bySession)
		if err != nil {
			logDebug("Read session groups from %s failed: %v", p, err)
			continue
		}
		logDebug("Read %d session groups from %s", n, p)
		if n > 0 {
			g.groups += n
			g.dbs++
		}
	}
	return g
}

// readGroupsFromDB đọc các khóa sessionGroups:* trong ItemTable của một state.vscdb,
// ghi vào out; trả về số nhóm có tên.
func readGroupsFromDB(path string, out map[string]string) (int, error) {
	db, err := openSQLite(path)
	if err != nil {
		return 0, err
	}
	root, err := db.tableRoot("ItemTable")
	if err != nil {
		return 0, err
	}
	count := 0
	err = db.scanTable(root, func(c []any) error {
		if len(c) < 2 {
			return nil
		}
		v := sqliteText(c[1])
		if !strings.Contains(v, groupsKeyPrefix) {
			return nil
		}
		count += parseGroupsValue(v, out)
		return nil
	})
	return count, err
}

// parseGroupsValue đọc một giá trị globalState (JSON) của extension.
func parseGroupsValue(v string, out map[string]string) int {
	var state map[string]json.RawMessage
	if err := json.Unmarshal([]byte(v), &state); err != nil {
		logDebug("Parse globalState value failed: %v", err)
		return 0
	}
	count := 0
	for k, raw := range state {
		if !strings.HasPrefix(k, groupsKeyPrefix) {
			continue
		}
		var groups []struct {
			Name       string   `json:"name"`
			SessionIDs []string `json:"sessionIds"`
		}
		if err := json.Unmarshal(raw, &groups); err != nil {
			logDebug("Parse %s failed: %v", k, err)
			continue
		}
		for _, gr := range groups {
			name := strings.TrimSpace(gr.Name)
			if name == "" {
				continue
			}
			count++
			for _, id := range gr.SessionIDs {
				out[strings.ToLower(id)] = name
			}
		}
	}
	return count
}

// groupDirName là tên thư mục con cho một nhóm.
func groupDirName(group string) string {
	if n := safeFileName(group); n != "" {
		return n
	}
	return "Nhóm không tên"
}
