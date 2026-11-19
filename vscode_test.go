package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- bộ ghi SQLite tối giản, chỉ để dựng file mẫu cho test ----------

type testDB struct {
	ps    int
	pages [][]byte
}

func (d *testDB) alloc() uint32 {
	d.pages = append(d.pages, make([]byte, d.ps))
	return uint32(len(d.pages))
}

func (d *testDB) pg(n uint32) []byte { return d.pages[n-1] }

func putVarint(v uint64) []byte {
	if v < 0x80 {
		return []byte{byte(v)}
	}
	var groups []byte
	for v > 0 {
		groups = append([]byte{byte(v & 0x7f)}, groups...)
		v >>= 7
	}
	for i := 0; i < len(groups)-1; i++ {
		groups[i] |= 0x80
	}
	return groups
}

// record mã hóa các cột (string → TEXT, []byte → BLOB, int → số nguyên 1 hoặc 4 byte, nil → NULL).
func record(cols ...any) []byte {
	var types, body []byte
	for _, c := range cols {
		switch v := c.(type) {
		case nil:
			types = append(types, putVarint(0)...)
		case int:
			if v >= -128 && v < 128 {
				types = append(types, putVarint(1)...)
				body = append(body, byte(int8(v)))
			} else {
				types = append(types, putVarint(4)...)
				body = binary.BigEndian.AppendUint32(body, uint32(int32(v)))
			}
		case string:
			types = append(types, putVarint(uint64(len(v)*2+13))...)
			body = append(body, v...)
		case []byte:
			types = append(types, putVarint(uint64(len(v)*2+12))...)
			body = append(body, v...)
		}
	}
	hlen := len(types) + 1 // header ngắn hơn 128 byte trong các test
	return append(append(putVarint(uint64(hlen)), types...), body...)
}

// leafCell tạo ô trang lá; payload dài thì phần dư đi vào chuỗi trang tràn.
func (d *testDB) leafCell(rowid int64, payload []byte) []byte {
	cell := append(putVarint(uint64(len(payload))), putVarint(uint64(rowid))...)
	u := d.ps
	maxLocal := u - 35
	if len(payload) <= maxLocal {
		return append(cell, payload...)
	}
	minLocal := (u-12)*32/255 - 23
	local := minLocal + (len(payload)-minLocal)%(u-4)
	if local > maxLocal {
		local = minLocal
	}
	cell = append(cell, payload[:local]...)
	rest := payload[local:]
	first := d.alloc()
	cell = binary.BigEndian.AppendUint32(cell, first)
	for cur := first; len(rest) > 0; {
		n := min(len(rest), u-4)
		copy(d.pg(cur)[4:], rest[:n])
		rest = rest[n:]
		if len(rest) > 0 {
			next := d.alloc()
			binary.BigEndian.PutUint32(d.pg(cur), next)
			cur = next
		}
	}
	return cell
}

// writePage ghi một trang B-tree bảng (0x0D lá, 0x05 trong).
func (d *testDB) writePage(n uint32, kind byte, cells [][]byte, right uint32) {
	p := d.pg(n)
	hdr := 0
	if n == 1 {
		hdr = 100
	}
	p[hdr] = kind
	binary.BigEndian.PutUint16(p[hdr+3:], uint16(len(cells)))
	ptrs := hdr + 8
	if kind == 0x05 {
		binary.BigEndian.PutUint32(p[hdr+8:], right)
		ptrs = hdr + 12
	}
	end := d.ps
	for i, c := range cells {
		end -= len(c)
		copy(p[end:], c)
		binary.BigEndian.PutUint16(p[ptrs+2*i:], uint16(end))
	}
	binary.BigEndian.PutUint16(p[hdr+5:], uint16(end))
}

// table ghi các dòng (rowid 1..n) vào cây B-tree: nhiều trang lá thì thêm một trang gốc trong.
func (d *testDB) table(rows [][]byte) uint32 {
	type leaf struct {
		n     uint32
		cells [][]byte
		last  int64
	}
	var leaves []*leaf
	space := 0
	for i, r := range rows {
		c := d.leafCell(int64(i+1), r)
		if len(leaves) == 0 || space < len(c)+2 {
			leaves = append(leaves, &leaf{n: d.alloc()})
			space = d.ps - 8
		}
		l := leaves[len(leaves)-1]
		l.cells = append(l.cells, c)
		l.last = int64(i + 1)
		space -= len(c) + 2
	}
	for _, l := range leaves {
		d.writePage(l.n, 0x0D, l.cells, 0)
	}
	if len(leaves) == 1 {
		return leaves[0].n
	}
	root := d.alloc()
	var cells [][]byte
	for _, l := range leaves[:len(leaves)-1] {
		cells = append(cells, append(binary.BigEndian.AppendUint32(nil, l.n), putVarint(uint64(l.last))...))
	}
	d.writePage(root, 0x05, cells, leaves[len(leaves)-1].n)
	return root
}

// buildItemTable tạo file SQLite có bảng ItemTable(key, value) với các dòng cho trước.
func buildItemTable(t *testing.T, path string, ps int, kv [][2]string) {
	t.Helper()
	d := &testDB{ps: ps}
	d.alloc() // trang 1: sqlite_schema
	var rows [][]byte
	for _, r := range kv {
		rows = append(rows, record(r[0], []byte(r[1])))
	}
	root := d.table(rows)
	d.writePage(1, 0x0D, [][]byte{
		d.leafCell(1, record("table", "ItemTable", "ItemTable", int(root), "CREATE TABLE ItemTable (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)")),
	}, 0)
	h := d.pg(1)
	copy(h, "SQLite format 3\x00")
	binary.BigEndian.PutUint16(h[16:], uint16(ps))
	h[18], h[19], h[21], h[22], h[23] = 1, 1, 64, 32, 32
	binary.BigEndian.PutUint32(h[28:], uint32(len(d.pages)))
	binary.BigEndian.PutUint32(h[56:], 1)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var all []byte
	for _, p := range d.pages {
		all = append(all, p...)
	}
	if err := os.WriteFile(path, all, 0o644); err != nil {
		t.Fatal(err)
	}
}

// groupsJSON tạo giá trị globalState của extension có nhóm cho một dự án.
func groupsJSON(project string, groups map[string][]string) string {
	var parts []string
	for name, ids := range groups {
		parts = append(parts, fmt.Sprintf(`{"id":"x","name":%q,"collapsed":false,"sessionIds":["%s"]}`, name, strings.Join(ids, `","`)))
	}
	return fmt.Sprintf(`{"other":1,"sessionGroups:%s":[%s]}`, strings.ReplaceAll(project, `\`, `\\`), strings.Join(parts, ","))
}

// ---------- test ----------

func TestSQLiteReadItemTable(t *testing.T) {
	p := filepath.Join(tempDir(t), "state.vscdb")
	var kv [][2]string
	for i := 0; i < 300; i++ { // đủ nhiều để cần nhiều trang lá
		kv = append(kv, [2]string{fmt.Sprintf("key%03d", i), fmt.Sprintf("giá trị %d", i)})
	}
	big := strings.Repeat("dữ liệu dài ", 400) // vượt một trang: cần trang tràn
	kv = append(kv, [2]string{"big", big})
	buildItemTable(t, p, 512, kv)

	db, err := openSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	root, err := db.tableRoot("ItemTable")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	if err := db.scanTable(root, func(c []any) error {
		got[sqliteText(c[0])] = sqliteText(c[1])
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(kv) {
		t.Fatalf("read %d rows, want %d", len(got), len(kv))
	}
	for _, r := range kv {
		if got[r[0]] != r[1] {
			t.Fatalf("row %s: got %d bytes, want %d", r[0], len(got[r[0]]), len(r[1]))
		}
	}
}

// Chế độ WAL: trang đã commit trong file -wal thay cho trang trong file chính; khung
// chưa commit bị bỏ qua.
func TestSQLiteWAL(t *testing.T) {
	dir := tempDir(t)
	oldP, newP := filepath.Join(dir, "old.vscdb"), filepath.Join(dir, "new.vscdb")
	buildItemTable(t, oldP, 512, [][2]string{{"k", "giá trị cũ"}})
	buildItemTable(t, newP, 512, [][2]string{{"k", "giá trị MỚI"}})
	main, _ := os.ReadFile(oldP)
	main[18], main[19] = 2, 2
	newData, _ := os.ReadFile(newP)
	nPages := len(newData) / 512

	w := make([]byte, 32)
	binary.BigEndian.PutUint32(w[0:], 0x377f0682)
	binary.BigEndian.PutUint32(w[8:], 512)
	binary.BigEndian.PutUint32(w[16:], 7) // salt
	binary.BigEndian.PutUint32(w[20:], 9)
	frame := func(pg int, commit int, data []byte) {
		h := make([]byte, 24)
		binary.BigEndian.PutUint32(h[0:], uint32(pg))
		binary.BigEndian.PutUint32(h[4:], uint32(commit))
		binary.BigEndian.PutUint32(h[8:], 7)
		binary.BigEndian.PutUint32(h[12:], 9)
		w = append(append(w, h...), data...)
	}
	for pg := 1; pg <= nPages; pg++ {
		commit := 0
		if pg == nPages {
			commit = nPages
		}
		frame(pg, commit, newData[(pg-1)*512:pg*512])
	}
	frame(2, 0, make([]byte, 512)) // chưa commit: phải bị bỏ qua

	p := filepath.Join(dir, "state.vscdb")
	os.WriteFile(p, main, 0o644)
	os.WriteFile(p+"-wal", w, 0o644)
	db, err := openSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	root, err := db.tableRoot("ItemTable")
	if err != nil {
		t.Fatal(err)
	}
	var got string
	if err := db.scanTable(root, func(c []any) error { got = sqliteText(c[1]); return nil }); err != nil {
		t.Fatal(err)
	}
	if got != "giá trị MỚI" {
		t.Fatalf("got %q", got)
	}
}

func TestSQLiteRejectsGarbage(t *testing.T) {
	p := filepath.Join(tempDir(t), "x.vscdb")
	os.WriteFile(p, []byte(strings.Repeat("x", 4096)), 0o644)
	if _, err := openSQLite(p); err == nil {
		t.Fatal("expected error for non-SQLite file")
	}
	// File SQLite bị cắt cụt: không panic, chỉ báo lỗi.
	full := filepath.Join(tempDir(t), "state.vscdb")
	buildItemTable(t, full, 512, [][2]string{{"a", strings.Repeat("b", 3000)}})
	data, _ := os.ReadFile(full)
	os.WriteFile(p, data[:1024], 0o644)
	if _, err := readGroupsFromDB(p, map[string]string{}); err == nil {
		t.Fatal("expected error for truncated file")
	}
}

// TestReadRealStateDB đọc nhóm từ một state.vscdb thật khi đặt CHAT2MD_VSCDB (mặc định bỏ qua).
func TestReadRealStateDB(t *testing.T) {
	p := os.Getenv("CHAT2MD_VSCDB")
	if p == "" {
		t.Skip("set CHAT2MD_VSCDB to a state.vscdb to run")
	}
	out := map[string]string{}
	n, err := readGroupsFromDB(p, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d groups, %d grouped sessions", n, len(out))
}

func TestParseGroupsValue(t *testing.T) {
	out := map[string]string{}
	v := `{"sessionGroups:D:\\Projects\\my-app":[{"name":"Nhóm A","sessionIds":["AAA","bbb"]},{"name":"  ","sessionIds":["ccc"]}],"x":[]}`
	if n := parseGroupsValue(v, out); n != 1 {
		t.Fatalf("groups %d, want 1", n)
	}
	if out["aaa"] != "Nhóm A" || out["bbb"] != "Nhóm A" || out["ccc"] != "" {
		t.Fatalf("bySession %v", out)
	}
}

func TestVSCodeUserDataDir(t *testing.T) {
	root := tempDir(t) // giả lập VS Code portable: <root>\data\user-data\User\globalStorage
	ud := filepath.Join(root, "data", "user-data")
	gs := filepath.Join(ud, "User", "globalStorage")
	os.MkdirAll(gs, 0o755)
	for _, in := range []string{root, filepath.Join(root, "data"), ud, filepath.Join(ud, "User"), gs} {
		if got := vscodeUserDataDir(in); got != ud {
			t.Errorf("vscodeUserDataDir(%q) = %q, want %q", in, got, ud)
		}
	}
	if got := vscodeUserDataDir(tempDir(t)); got != "" {
		t.Errorf("empty dir: got %q", got)
	}
}

// isolateVSCode trỏ các vị trí VS Code chuẩn vào thư mục tạm để test không đọc dữ liệu thật.
func isolateVSCode(t *testing.T) {
	t.Helper()
	tmp := tempDir(t)
	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME", "HOME"} {
		t.Setenv(k, tmp)
	}
	t.Setenv("VSCODE_PORTABLE", "")
}

func TestConvertAllGroups(t *testing.T) {
	isolateVSCode(t)
	src, out, vsc := tempDir(t), tempDir(t), tempDir(t)
	a := writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Phiên A"}`, testUserLine)
	b := writeJSONL(t, src, "bbb.jsonl", `{"type":"ai-title","aiTitle":"Phiên B"}`, testUserLine)
	touch(t, a, 1000)
	touch(t, b, 1000)
	// Bản portable: <thư mục cài>\data\user-data\User\globalStorage\state.vscdb
	db := filepath.Join(vsc, "data", "user-data", "User", "globalStorage", "state.vscdb")
	setGroups := func(groups map[string][]string) {
		buildItemTable(t, db, 512, [][2]string{
			{"other.ext", "{}"},
			{"Anthropic.claude-code", groupsJSON(`D:\Projects\my-app`, groups)},
		})
	}
	setGroups(map[string][]string{"Nhóm: ví dụ": {"aaa"}})

	opt := Options{TitleName: true, Incremental: true, GroupHeader: true, GroupFolders: true, VSCodeDirs: []string{vsc}}
	run := func() Result {
		t.Helper()
		res, err := ConvertAll([]string{src}, out, opt)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if res := run(); res.OK != 2 {
		t.Fatalf("first run %+v", res)
	}
	groupDir := filepath.Join(out, "Nhóm ví dụ") // ":" không hợp lệ trong tên thư mục
	md, err := os.ReadFile(filepath.Join(groupDir, "Phiên A.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "- **Nhóm:** Nhóm: ví dụ\n") {
		t.Errorf("missing group line:\n%s", md)
	}
	mdB, _ := os.ReadFile(filepath.Join(out, "Phiên B.md"))
	if strings.Contains(string(mdB), "**Nhóm:**") {
		t.Errorf("ungrouped chat should have no group line")
	}

	// Không đổi gì: bỏ qua hết.
	if res := run(); res.OK != 0 || res.Unchanged != 2 {
		t.Fatalf("second run %+v", res)
	}

	// Chuyển A sang nhóm khác: xuất lại A, file cũ và thư mục nhóm trống bị dọn.
	setGroups(map[string][]string{"Nhóm mới": {"aaa"}})
	if res := run(); res.OK != 1 || res.Unchanged != 1 {
		t.Fatalf("after regroup %+v", res)
	}
	if _, err := os.Stat(filepath.Join(out, "Nhóm mới", "Phiên A.md")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(groupDir); !os.IsNotExist(err) {
		t.Errorf("old group dir should be removed, err %v", err)
	}

	// Tắt thư mục nhóm: A về thư mục dự án, vẫn có dòng nhóm.
	opt.GroupFolders = false
	if res := run(); res.OK != 2 {
		t.Fatalf("folders off %+v", res)
	}
	if names := mdFiles(t, out); strings.Join(names, "|") != "Phiên A.md|Phiên B.md" {
		t.Errorf("files %v", names)
	}
	if _, err := os.Stat(filepath.Join(out, "Nhóm mới")); !os.IsNotExist(err) {
		t.Errorf("group dir should be removed, err %v", err)
	}
}

// Đã xuất khi chưa bật thư mục nhóm, sau đó bật: file được dời vào thư mục nhóm,
// file cũ chưa sửa bị dọn, file cũ đã sửa được giữ.
func TestGroupFoldersTurnedOnLater(t *testing.T) {
	isolateVSCode(t)
	src, out, vsc := tempDir(t), tempDir(t), tempDir(t)
	writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"A"}`, testUserLine)
	writeJSONL(t, src, "bbb.jsonl", `{"type":"ai-title","aiTitle":"B"}`, testUserLine)
	writeJSONL(t, src, "ccc.jsonl", `{"type":"ai-title","aiTitle":"C"}`, testUserLine)
	buildItemTable(t, filepath.Join(vsc, "User", "globalStorage", "state.vscdb"), 512, [][2]string{
		{"Anthropic.claude-code", groupsJSON(`D:\Projects\my-app`, map[string][]string{"Nhóm X": {"aaa", "bbb"}})},
	})
	opt := Options{TitleName: true, Incremental: true, VSCodeDirs: []string{vsc}}
	if _, err := ConvertAll([]string{src}, out, opt); err != nil {
		t.Fatal(err)
	}
	// Người dùng sửa B.md sau lần xuất đầu.
	edited := filepath.Join(out, "B.md")
	os.WriteFile(edited, []byte("ghi chú thêm\n"), 0o644)
	touch(t, edited, 5000)

	opt.GroupFolders = true
	res, err := ConvertAll([]string{src}, out, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK != 3 { // đổi tùy chọn: chuyển lại tất cả dù đang bật "chỉ chuyển thay đổi"
		t.Fatalf("res %+v", res)
	}
	if names := mdFiles(t, out); strings.Join(names, "|") != "B.md|C.md" {
		t.Errorf("project dir: %v (A.md should be moved, edited B.md kept)", names)
	}
	if names := mdFiles(t, filepath.Join(out, "Nhóm X")); strings.Join(names, "|") != "A.md|B.md" {
		t.Errorf("group dir: %v", names)
	}
}

// Xuất riêng một dự án vào thư mục đã có bản xuất cả projects: file của lần trước
// (ở thư mục dự án) không bị coi là thư mục nhóm và không bị xóa.
func TestGroupsKeepOtherBaseExport(t *testing.T) {
	isolateVSCode(t)
	root, out := tempDir(t), tempDir(t)
	proj := filepath.Join(root, "my-app")
	os.MkdirAll(proj, 0o755)
	writeJSONL(t, proj, "aaa.jsonl", `{"type":"ai-title","aiTitle":"A"}`, testUserLine)
	if _, err := ConvertAll([]string{root}, out, Options{TitleName: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := ConvertAll([]string{proj}, out, Options{TitleName: true, GroupFolders: true}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(out, "my-app", "A.md"), filepath.Join(out, "A.md")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s: %v", p, err)
		}
	}
}
