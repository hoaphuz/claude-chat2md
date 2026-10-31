package main

// Phần chuyển đổi transcript Claude Code (.jsonl) sang Markdown.
// Không phụ thuộc hệ điều hành; giao diện Win32 nằm trong main_windows.go.

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	Tools    bool // kèm lệnh gọi công cụ và kết quả
	Thinking bool // kèm phần suy luận của Claude
}

// Result là thống kê sau khi chuyển.
type Result struct {
	OK, Skipped, Failed int
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

// ConvertFile chuyển một file .jsonl; trả về "" nếu không có nội dung.
func ConvertFile(file string, opt Options) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	var title, firstTime, cwd, lastRole string
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
			title = e.CustomTitle
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
		return "", nil
	}

	session := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
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
	if t := formatTime(firstTime); t != "" {
		sb.WriteString("- **Bắt đầu:** " + t + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Join(body, "\n\n"))
	sb.WriteString("\n")
	return sb.String(), nil
}

// safeConvert chuyển một file; panic chỉ làm hỏng file đó, không dừng cả lượt.
func safeConvert(file string, opt Options) (md string, err error) {
	defer func() {
		if r := recover(); r != nil {
			logDebug("Panic converting %s: %v\n%s", file, r, debug.Stack())
			err = fmt.Errorf("lỗi nội bộ: %v", r)
		}
	}()
	return ConvertFile(file, opt)
}

// ConvertAll chuyển một file hoặc cả thư mục; nhật ký đi qua logInfo/logError/logDebug.
func ConvertAll(input, outRoot string, opt Options) (Result, error) {
	var res Result
	start := time.Now()
	logDebug("ConvertAll input=%q out=%q options=%+v", input, outRoot, opt)
	st, err := os.Stat(input)
	if err != nil {
		logDebug("Stat input failed: %v", err)
		return res, fmt.Errorf("không tìm thấy: %s", input)
	}

	base := input
	var files []string
	if !st.IsDir() {
		base = filepath.Dir(input)
		if strings.EqualFold(filepath.Ext(input), ".jsonl") {
			files = []string{input}
		}
	} else {
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
				files = append(files, p)
			}
			return nil
		})
	}

	if len(files) == 0 {
		return res, fmt.Errorf("không có file .jsonl nào trong: %s", input)
	}
	logInfo("Tìm thấy %d file .jsonl", len(files))

	for _, f := range files {
		t0 := time.Now()
		md, err := safeConvert(f, opt)
		if err != nil {
			res.Failed++
			logError("Lỗi: %s: %v", f, err)
			continue
		}
		if md == "" {
			res.Skipped++
			logDebug("Skipped (no messages): %s", f)
			continue
		}
		rel, err := filepath.Rel(base, f)
		if err != nil {
			rel = filepath.Base(f)
		}
		rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + ".md"
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
