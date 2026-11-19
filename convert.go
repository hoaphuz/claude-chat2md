package main

// Phần chuyển đổi transcript Claude Code (.jsonl) sang Markdown.
// Không phụ thuộc hệ điều hành; giao diện Win32 nằm trong main_windows.go.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
	"unicode/utf8"
)

// Options là các tùy chọn khi xuất.
type Options struct {
	Tools     bool // kèm lệnh gọi công cụ và kết quả
	Thinking  bool // kèm phần suy luận của Claude
	TitleName bool // đặt tên file theo tiêu đề phiên thay vì mã session

	Incremental bool // bỏ qua file nguồn không đổi kể từ lần xuất trước (xem state.go)

	// Nhóm chat tạo trong VS Code (xem vscode.go).
	GroupHeader  bool     // ghi tên nhóm vào phần đầu file .md
	GroupFolders bool     // xếp file .md vào thư mục con theo nhóm
	VSCodeDirs   []string // chỗ tìm thêm dữ liệu VS Code (bản portable, file state.vscdb)

	groups *sessionGroups // ConvertAll nạp khi bật một trong hai tùy chọn nhóm
}

func (o Options) useGroups() bool { return o.GroupHeader || o.GroupFolders }

// Result là thống kê sau khi chuyển.
type Result struct {
	OK, Skipped, Failed int
	Unchanged           int // bỏ qua vì không đổi kể từ lần xuất trước
}

const maxToolChars = 3000 // cắt bớt kết quả công cụ quá dài

// convStats là số liệu của một file, chỉ dùng cho log chi tiết.
type convStats struct {
	lines, malformed, firstBadLine int
	firstBadErr                    error
	sidechain, meta, kept          int
	unknown                        map[string]int // loại block chưa hỗ trợ
}

type block struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	Content  json.RawMessage `json:"content"`
	IsError  bool            `json:"is_error"`
}

type message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type entry struct {
	Type        string   `json:"type"`
	Summary     string   `json:"summary"`
	CustomTitle string   `json:"customTitle"`
	AITitle     string   `json:"aiTitle"`
	Cwd         string   `json:"cwd"`
	Timestamp   string   `json:"timestamp"`
	IsSidechain bool     `json:"isSidechain"`
	IsMeta      bool     `json:"isMeta"`
	Message     *message `json:"message"`
}

// ---------- tiện ích ----------

func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxToolChars {
		return s
	}
	r := []rune(s)
	return string(r[:maxToolChars]) + fmt.Sprintf("\n… (đã cắt %d ký tự)", len(r)-maxToolChars)
}

var reBackticks = regexp.MustCompile("`+")

// fence chọn số dấu ` đủ dài để nội dung bên trong không phá khối code.
func fence(text, lang string) string {
	longest := 2
	for _, m := range reBackticks.FindAllString(text, -1) {
		if len(m) > longest {
			longest = len(m)
		}
	}
	f := strings.Repeat("`", longest+1)
	return f + lang + "\n" + text + "\n" + f
}

func formatTime(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ""
	}
	return t.Local().Format("15:04:05 02/01/2006")
}

// parseContent: nội dung tin nhắn có thể là chuỗi hoặc mảng block.
func parseContent(raw json.RawMessage) (str string, blocks []block, isStr bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil, true
	}
	if err := json.Unmarshal(raw, &str); err == nil {
		return str, nil, true
	}
	_ = json.Unmarshal(raw, &blocks)
	return "", blocks, false
}

func toolResultText(raw json.RawMessage) string {
	s, blocks, isStr := parseContent(raw)
	if isStr {
		return s
	}
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		case "image":
			parts = append(parts, "[hình ảnh]")
		}
	}
	return strings.Join(parts, "\n")
}

func prettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "{}"
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

func details(summary, body string) string {
	return "<details><summary>" + summary + "</summary>\n\n" + body + "\n\n</details>"
}

func renderContent(raw json.RawMessage, opt Options, st *convStats) string {
	s, blocks, isStr := parseContent(raw)
	if isStr {
		return strings.TrimSpace(s)
	}
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if t := strings.TrimSpace(b.Text); t != "" {
				parts = append(parts, t)
			}
		case "thinking":
			if opt.Thinking && strings.TrimSpace(b.Thinking) != "" {
				parts = append(parts, details("💭 Suy luận", strings.TrimSpace(b.Thinking)))
			}
		case "tool_use":
			if opt.Tools {
				parts = append(parts, details("🔧 "+b.Name, fence(truncate(prettyJSON(b.Input)), "json")))
			}
		case "tool_result":
			if opt.Tools {
				txt := strings.TrimSpace(toolResultText(b.Content))
				if txt == "" {
					txt = "(trống)"
				}
				label := "📄 Kết quả"
				if b.IsError {
					label = "⚠️ Lỗi từ công cụ"
				}
				parts = append(parts, details(label, fence(truncate(txt), "")))
			}
		case "image":
			parts = append(parts, "_[hình ảnh đính kèm]_")
		default:
			if st != nil {
				st.unknown[b.Type]++
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

// Bỏ các đoạn hệ thống tự chèn vào tin nhắn người dùng.
var (
	reReminder = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)
	reCmdName  = regexp.MustCompile(`(?s)<command-name>(.*?)</command-name>`)
	reCmdMsg   = regexp.MustCompile(`(?s)<command-message>.*?</command-message>`)
	reCmdArgs  = regexp.MustCompile(`(?s)<command-args>(.*?)</command-args>`)
	reStdout   = regexp.MustCompile(`(?s)<local-command-stdout>.*?</local-command-stdout>`)
)

func cleanUserText(t string) string {
	t = reReminder.ReplaceAllString(t, "")
	t = reCmdName.ReplaceAllStringFunc(t, func(m string) string {
		return strings.TrimSpace(reCmdName.FindStringSubmatch(m)[1]) + " "
	})
	t = reCmdMsg.ReplaceAllString(t, "")
	t = reCmdArgs.ReplaceAllStringFunc(t, func(m string) string {
		return strings.TrimSpace(reCmdArgs.FindStringSubmatch(m)[1])
	})
	t = reStdout.ReplaceAllString(t, "")
	return strings.TrimSpace(t)
}

func allToolResults(blocks []block) bool {
	for _, b := range blocks {
		if b.Type != "tool_result" {
			return false
		}
	}
	return true
}

const (
	roleUser   = "👤 Bạn"
	roleClaude = "🤖 Claude"
)

// ConvertFile chuyển một file .jsonl; trả về md "" nếu không có nội dung.
// title là tiêu đề phiên (dùng cho tiêu đề Markdown và tên file khi bật TitleName).
func ConvertFile(file string, opt Options) (md, title string, err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", "", err
	}

	var customTitle, aiTitle, firstTime, cwd, lastRole string
	var body []string
	st := &convStats{unknown: map[string]int{}}
	defer func() {
		logDebug("Parsed %s: %d bytes, %d lines, %d malformed (first: line %d, %v), %d sidechain, %d meta, %d messages kept, unknown blocks %v",
			file, len(data), st.lines, st.malformed, st.firstBadLine, st.firstBadErr, st.sidechain, st.meta, st.kept, st.unknown)
	}()

	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		st.lines++
		var e entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			st.malformed++ // bỏ qua dòng hỏng
			if st.firstBadErr == nil {
				st.firstBadLine, st.firstBadErr = i+1, err
			}
			continue
		}
		if e.Type == "summary" && e.Summary != "" && title == "" {
			title = e.Summary
		}
		if e.Type == "custom-title" && e.CustomTitle != "" {
			customTitle = e.CustomTitle
		}
		if e.Type == "ai-title" && e.AITitle != "" {
			aiTitle = e.AITitle // Claude Code tự đặt tên phiên; lấy bản mới nhất
		}
		if cwd == "" && e.Cwd != "" {
			cwd = e.Cwd
		}
		if e.Type != "user" && e.Type != "assistant" {
			continue
		}
		// bỏ hội thoại của subagent và tin nhắn hệ thống
		if e.IsSidechain || e.IsMeta || e.Message == nil {
			if e.IsSidechain {
				st.sidechain++
			} else if e.IsMeta {
				st.meta++
			}
			continue
		}
		if firstTime == "" && e.Timestamp != "" {
			firstTime = e.Timestamp
		}
		st.kept++

		text := renderContent(e.Message.Content, opt, st)
		if e.Type == "user" {
			// tin nhắn "user" chỉ chứa kết quả công cụ thì gộp vào lượt của Claude
			if _, blocks, isStr := parseContent(e.Message.Content); !isStr && allToolResults(blocks) {
				if text != "" {
					body = append(body, text)
				}
				continue
			}
			text = cleanUserText(text)
		}
		if text == "" {
			continue
		}

		role := roleClaude
		if e.Type == "user" {
			role = roleUser
		}
		if role != lastRole {
			h := "---\n\n### " + role
			if t := formatTime(e.Timestamp); t != "" {
				h += "  ·  _" + t + "_"
			}
			body = append(body, h)
			lastRole = role
		}
		body = append(body, text)
	}

	if len(body) == 0 {
		return "", "", nil
	}

	session := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	switch {
	case customTitle != "":
		title = customTitle
	case aiTitle != "":
		title = aiTitle
	}
	if title == "" {
		title = session
		for i := 1; i < len(body); i++ {
			if strings.Contains(body[i-1], roleUser) {
				first := strings.SplitN(body[i], "\n", 2)[0]
				if r := []rune(first); len(r) > 80 {
					first = string(r[:80])
				}
				title = first
				break
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("# " + title + "\n\n")
	sb.WriteString("- **Session:** `" + session + "`\n")
	if cwd != "" {
		sb.WriteString("- **Thư mục dự án:** `" + cwd + "`\n")
	}
	if opt.GroupHeader {
		if g := opt.groups.of(session); g != "" {
			sb.WriteString("- **Nhóm:** " + g + "\n")
		}
	}
	if t := formatTime(firstTime); t != "" {
		sb.WriteString("- **Bắt đầu:** " + t + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Join(body, "\n\n"))
	sb.WriteString("\n")
	return sb.String(), title, nil
}

// Ký tự không được phép trong tên file Windows (và "/" trên mọi hệ điều hành).
var reBadName = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]+`)

const maxNameRunes = 100

// safeFileName biến tiêu đề phiên thành tên file hợp lệ (không gồm phần mở rộng);
// trả về "" nếu không còn gì dùng được.
func safeFileName(title string) string {
	s := reBadName.ReplaceAllString(title, " ")
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maxNameRunes {
		s = string(r[:maxNameRunes])
	}
	s = strings.TrimRight(s, ". ")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Tên thiết bị dành riêng của Windows (CON, NUL, COM1…), kể cả khi có đuôi.
	stem := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
	switch stem {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		s = "_" + s
	}
	return s
}

// uniqueName trả về tên file (kèm .md) chưa bị file nguồn khác giữ trong used
// (khóa: đường dẫn .md viết thường, giá trị: khóa file nguồn sở hữu), thêm " (2)",
// " (3)"… khi trùng, rồi giữ tên đó cho owner. Tên chưa ai giữ nhưng foreign(đường dẫn
// tương đối) trả về true (file có sẵn trên đĩa không phải của chat này) cũng bị bỏ qua.
func uniqueName(dir, name, owner string, used map[string]string, foreign func(rel string) bool) string {
	rel := func(c string) string { return filepath.Join(dir, c+".md") }
	cand := name
	for i := 2; ; i++ {
		o, taken := used[strings.ToLower(rel(cand))]
		if o == owner && taken {
			break
		}
		if !taken && (foreign == nil || !foreign(rel(cand))) {
			break
		}
		cand = fmt.Sprintf("%s (%d)", name, i)
	}
	used[strings.ToLower(rel(cand))] = owner
	return cand + ".md"
}

var reSessionLine = regexp.MustCompile("(?m)^- \\*\\*Session:\\*\\* `([^`]+)`")

// mdSession đọc mã phiên trong phần đầu một file .md do ứng dụng xuất;
// "" nếu không đọc được hoặc không có (file không do ứng dụng tạo).
func mdSession(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 4096) // dòng Session nằm ngay dưới tiêu đề
	n, _ := io.ReadFull(f, buf)
	if m := reSessionLine.FindSubmatch(buf[:n]); m != nil {
		return string(m[1])
	}
	return ""
}

// foreignFile: tại outRoot/rel đã có file mà không phải bản xuất của phiên session
// (file người dùng tự tạo, của công cụ khác, hoặc của chat khác) — không được ghi đè.
func foreignFile(outRoot, rel, session string) bool {
	p := filepath.Join(outRoot, rel)
	if _, err := os.Lstat(p); err != nil {
		return false
	}
	return !strings.EqualFold(mdSession(p), session)
}

// safeConvert chuyển một file; panic chỉ làm hỏng file đó, không dừng cả lượt.
func safeConvert(file string, opt Options) (md, title string, err error) {
	defer func() {
		if r := recover(); r != nil {
			logDebug("Panic converting %s: %v\n%s", file, r, debug.Stack())
			err = fmt.Errorf("lỗi nội bộ: %v", r)
		}
	}()
	return ConvertFile(file, opt)
}

// SplitInputs tách ô nguồn thành danh sách đường dẫn: nhiều đường dẫn cách nhau
// bằng ";", có thể bọc trong dấu ngoặc kép.
func SplitInputs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ";") {
		p = strings.Trim(strings.TrimSpace(p), `"`)
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// JoinInputs là chiều ngược của SplitInputs, dùng để hiện danh sách file đã chọn.
func JoinInputs(paths []string) string { return strings.Join(paths, "; ") }

// fileBase là thư mục gốc để tính đường dẫn đầu ra của một file chọn lẻ: file nằm
// trong thư mục projects của Claude Code thì giữ thư mục con theo dự án (giống khi
// chuyển cả thư mục projects), còn lại thì đặt thẳng vào thư mục đầu ra.
func fileBase(file string) string {
	root := DefaultSourceDir()
	if abs, err := filepath.Abs(file); err == nil {
		if rel, err := filepath.Rel(root, abs); err == nil && rel != "." &&
			!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return root
		}
	}
	return filepath.Dir(file)
}

// srcFile là một file .jsonl cần xét, kèm thư mục gốc để tính đường dẫn đầu ra.
type srcFile struct {
	path, base string
	explicit   bool // người dùng chọn đích danh file này (luôn chuyển, kể cả khi không đổi)
}

// collectInputs gom các file .jsonl từ danh sách file/thư mục, bỏ trùng.
func collectInputs(inputs []string) ([]srcFile, error) {
	var files []srcFile
	seen := map[string]bool{}
	add := func(f srcFile) {
		if k := stateKey(f.path); !seen[k] {
			seen[k] = true
			files = append(files, f)
		}
	}
	for _, input := range inputs {
		st, err := os.Stat(input)
		if err != nil {
			logDebug("Stat input failed: %v", err)
			return nil, fmt.Errorf("không tìm thấy: %s", input)
		}
		if !st.IsDir() {
			if !strings.EqualFold(filepath.Ext(input), ".jsonl") {
				return nil, fmt.Errorf("không phải file chat .jsonl: %s", input)
			}
			add(srcFile{path: input, base: fileBase(input), explicit: true})
			continue
		}
		_ = filepath.WalkDir(input, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				logDebug("Walk skipped %s: %v", p, err)
				return nil // bỏ qua thư mục không đọc được
			}
			if d.IsDir() {
				switch d.Name() {
				case "subagents", "memory", "tool-results":
					return filepath.SkipDir
				}
				return nil
			}
			if strings.EqualFold(filepath.Ext(d.Name()), ".jsonl") {
				add(srcFile{path: p, base: input})
			}
			return nil
		})
	}
	return files, nil
}

// ConvertAll chuyển các file và/hoặc thư mục trong inputs; nhật ký đi qua logInfo/logError/logDebug.
func ConvertAll(inputs []string, outRoot string, opt Options) (Result, error) {
	var res Result
	start := time.Now()
	logDebug("ConvertAll inputs=%q out=%q options=%+v", inputs, outRoot, opt)
	if len(inputs) == 0 {
		return res, fmt.Errorf("chưa chọn thư mục hoặc file nguồn")
	}
	files, err := collectInputs(inputs)
	if err != nil {
		return res, err
	}
	if len(files) == 0 {
		return res, fmt.Errorf("không có file .jsonl nào trong: %s", JoinInputs(inputs))
	}
	logInfo("Tìm thấy %d file .jsonl", len(files))

	if opt.useGroups() {
		opt.groups = loadSessionGroups(opt.VSCodeDirs)
		if opt.groups.groups == 0 {
			logWarn("Không tìm thấy nhóm chat nào trong dữ liệu VS Code; các chat được xuất như chưa phân nhóm")
		} else {
			logInfo("Đọc được %d nhóm chat từ VS Code (%d chat đã phân nhóm)", opt.groups.groups, len(opt.groups.bySession))
		}
	}

	state := loadState(outRoot)
	fk := formatKey(opt)

	// Lượt 1: xác định file cần chuyển. Tên .md đã xuất trước đó được giữ cho đúng
	// file nguồn của nó, để file khác (mới, trùng tiêu đề) không ghi đè lên.
	type job struct {
		src, key, dir string // dir: thư mục con (tương đối) trong đầu ra
		baseDir       string // dir khi chưa thêm thư mục nhóm
		groupDir      string // thư mục nhóm đã thêm vào dir ("" nếu không)
		fk            string // formatKey, kèm tên nhóm khi dùng nhóm
		size, mod     int64
		prev          stateEntry
		hasPrev       bool
	}
	used := map[string]string{}
	var jobs []job
	for _, f := range files {
		j := job{src: f.path, key: stateKey(f.path), fk: fk}
		rel, err := filepath.Rel(f.base, f.path)
		if err != nil {
			rel = filepath.Base(f.path)
		}
		j.dir = filepath.Dir(rel)
		j.baseDir = j.dir
		if opt.useGroups() {
			// Đổi nhóm không làm file .jsonl thay đổi, nên đưa tên nhóm vào khóa định dạng.
			g := opt.groups.of(strings.TrimSuffix(filepath.Base(f.path), filepath.Ext(f.path)))
			j.fk += " group=" + g
			if opt.GroupFolders && g != "" {
				j.groupDir = groupDirName(g)
				j.dir = filepath.Join(j.dir, j.groupDir)
			}
		}
		j.size, j.mod, _ = fileSig(f.path)
		j.prev, j.hasPrev = state[j.key]
		if j.hasPrev && j.prev.Out != "" {
			used[strings.ToLower(j.prev.Out)] = j.key
		}
		if opt.Incremental && !f.explicit && j.hasPrev && j.prev.Size == j.size && j.prev.ModTime == j.mod && j.prev.Format == j.fk {
			if j.prev.Out == "" {
				res.Unchanged++ // chat trống, không đổi
				continue
			}
			if _, _, ok := fileSig(filepath.Join(outRoot, j.prev.Out)); ok && strings.EqualFold(filepath.Dir(j.prev.Out), j.dir) {
				res.Unchanged++
				continue
			}
		}
		jobs = append(jobs, j)
	}
	if opt.Incremental {
		logInfo("Cần chuyển %d file mới hoặc đã thay đổi, bỏ qua %d file không đổi", len(jobs), res.Unchanged)
	}

	// Lượt 2: chuyển các file còn lại.
	for _, j := range jobs {
		t0 := time.Now()
		md, title, err := safeConvert(j.src, opt)
		if err != nil {
			res.Failed++
			logError("Lỗi: %s: %v", j.src, err)
			continue
		}
		if md == "" {
			res.Skipped++
			state[j.key] = stateEntry{Size: j.size, ModTime: j.mod, Format: j.fk}
			logDebug("Skipped (no messages): %s", j.src)
			continue
		}
		session := strings.TrimSuffix(filepath.Base(j.src), filepath.Ext(j.src))
		name := session
		if opt.TitleName {
			if n := safeFileName(title); n != "" {
				name = n
			}
		}
		rel := filepath.Join(j.dir, uniqueName(j.dir, name, j.key, used, func(r string) bool {
			if foreignFile(outRoot, r, session) {
				logInfo("Giữ nguyên tệp có sẵn không phải bản xuất của chat này: %s", r)
				return true
			}
			return false
		}))
		dest := filepath.Join(outRoot, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			res.Failed++
			logError("Lỗi: %v", err)
			continue
		}
		if err := os.WriteFile(dest, []byte(md), 0o644); err != nil {
			res.Failed++
			logError("Lỗi: %v", err)
			continue
		}
		res.OK++
		logInfo("Đã xuất: %s", rel)
		logDebug("Wrote %s (%d bytes) in %s", dest, len(md), time.Since(t0))

		e := stateEntry{Size: j.size, ModTime: j.mod, Format: j.fk, Out: rel, GroupDir: j.groupDir}
		e.OutSize, e.OutMod, _ = fileSig(dest)
		state[j.key] = e
		// Cùng thư mục dự án mà ra tên khác (đổi tên phiên, bật/tắt đặt tên theo tiêu đề,
		// đổi nhóm hoặc bật/tắt thư mục nhóm): dọn file cũ.
		if j.hasPrev && j.prev.Out != "" && !strings.EqualFold(j.prev.Out, rel) &&
			strings.EqualFold(j.prev.baseDir(), j.baseDir) {
			removeStale(outRoot, j.prev)
		}
	}

	if err := saveState(outRoot, state); err != nil {
		logWarn("Không lưu được trạng thái xuất (lần sau sẽ chuyển lại toàn bộ): %v", err)
	}
	logDebug("ConvertAll done: %+v in %s", res, time.Since(start))
	return res, nil
}

// DefaultSourceDir trả về thư mục projects của Claude Code.
func DefaultSourceDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "projects")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// DefaultOutputDir trả về thư mục đầu ra mặc định.
func DefaultOutputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "claude-export")
}
