package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const testUserLine = `{"type":"user","message":{"role":"user","content":"xin chào"},"timestamp":"2026-01-01T00:00:00Z"}`

func writeJSONL(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConvertFileTitlePriority(t *testing.T) {
	dir := tempDir(t)
	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{"first-message", []string{testUserLine}, "xin chào"},
		{"summary", []string{`{"type":"summary","summary":"Tóm tắt"}`, testUserLine}, "Tóm tắt"},
		{"ai-title-latest", []string{`{"type":"summary","summary":"Tóm tắt"}`, `{"type":"ai-title","aiTitle":"Tên cũ"}`, testUserLine, `{"type":"ai-title","aiTitle":"Tên mới"}`}, "Tên mới"},
		{"custom-wins", []string{`{"type":"custom-title","customTitle":"Tên riêng"}`, `{"type":"ai-title","aiTitle":"Tên AI"}`, testUserLine}, "Tên riêng"},
	}
	for _, c := range cases {
		p := writeJSONL(t, dir, c.name+".jsonl", c.lines...)
		md, title, err := ConvertFile(p, Options{})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if title != c.want || !strings.HasPrefix(md, "# "+c.want+"\n") {
			t.Errorf("%s: title %q, want %q", c.name, title, c.want)
		}
	}
}

func TestSafeFileName(t *testing.T) {
	cases := map[string]string{
		`Sửa lỗi: a/b\c?*`:          "Sửa lỗi a b c",
		"  nhiều   khoảng trắng.. ": "nhiều khoảng trắng",
		"con":                       "_con",
		"NUL.txt":                   "_NUL.txt",
		"???":                       "",
		strings.Repeat("ả", 150):    strings.Repeat("ả", maxNameRunes),
	}
	for in, want := range cases {
		if got := safeFileName(in); got != want {
			t.Errorf("safeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConvertAllTitleName(t *testing.T) {
	src, out := tempDir(t), tempDir(t)
	writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Cùng tên"}`, testUserLine)
	writeJSONL(t, src, "bbb.jsonl", `{"type":"ai-title","aiTitle":"Cùng tên"}`, testUserLine)
	writeJSONL(t, src, "ccc.jsonl", `{"type":"ai-title","aiTitle":"???"}`, testUserLine)

	res, err := ConvertAll([]string{src}, out, Options{TitleName: true})
	if err != nil || res.OK != 3 {
		t.Fatalf("res %+v, err %v", res, err)
	}
	names := mdFiles(t, out)
	want := []string{"Cùng tên (2).md", "Cùng tên.md", "ccc.md"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Errorf("files %v, want %v", names, want)
	}

	// Tắt tùy chọn: giữ tên theo mã session.
	out2 := tempDir(t)
	if _, err := ConvertAll([]string{src}, out2, Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out2, "aaa.md")); err != nil {
		t.Errorf("expected aaa.md: %v", err)
	}
}

// File .md có sẵn không do ứng dụng xuất cho chat đó thì không bị ghi đè.
func TestConvertAllKeepsForeignFile(t *testing.T) {
	src, out := tempDir(t), tempDir(t)
	writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Ghi chú"}`, testUserLine)
	note := filepath.Join(out, "Ghi chú.md")
	os.WriteFile(note, []byte("# Ghi chú riêng của tôi\n"), 0o644)

	if _, err := ConvertAll([]string{src}, out, Options{TitleName: true}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(note); string(data) != "# Ghi chú riêng của tôi\n" {
		t.Errorf("user file overwritten: %q", data)
	}
	if names := mdFiles(t, out); strings.Join(names, "|") != "Ghi chú (2).md|Ghi chú.md" {
		t.Errorf("files %v", names)
	}
}

// Mất file trạng thái: các bản xuất cũ (nhận ra nhờ dòng Session) được ghi đè đúng chỗ,
// không sinh thêm bản trùng " (3)".
func TestConvertAllLostStateNoDuplicates(t *testing.T) {
	src, out := tempDir(t), tempDir(t)
	writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Cùng tên"}`, testUserLine)
	writeJSONL(t, src, "bbb.jsonl", `{"type":"ai-title","aiTitle":"Cùng tên"}`, testUserLine)
	for i := 0; i < 2; i++ {
		if _, err := ConvertAll([]string{src}, out, Options{TitleName: true}); err != nil {
			t.Fatal(err)
		}
		os.Remove(filepath.Join(out, stateFileName))
	}
	if names := mdFiles(t, out); strings.Join(names, "|") != "Cùng tên (2).md|Cùng tên.md" {
		t.Errorf("files %v", names)
	}
	if s := mdSession(filepath.Join(out, "Cùng tên.md")); s != "aaa" {
		t.Errorf("Cùng tên.md belongs to %q, want aaa", s)
	}
}

func mdFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// touch đặt lại thời điểm sửa để test không phụ thuộc độ phân giải đồng hồ.
func touch(t *testing.T, p string, sec int64) {
	t.Helper()
	tm := time.Unix(sec, 0)
	if err := os.Chtimes(p, tm, tm); err != nil {
		t.Fatal(err)
	}
}

func TestConvertAllIncremental(t *testing.T) {
	src, out := tempDir(t), tempDir(t)
	a := writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Phiên A"}`, testUserLine)
	b := writeJSONL(t, src, "bbb.jsonl", `{"type":"ai-title","aiTitle":"Phiên B"}`, testUserLine)
	writeJSONL(t, src, "empty.jsonl", `{"type":"summary","summary":"x"}`)
	touch(t, a, 1000)
	touch(t, b, 1000)
	opt := Options{TitleName: true, Incremental: true}

	run := func() Result {
		t.Helper()
		res, err := ConvertAll([]string{src}, out, opt)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	if res := run(); res.OK != 2 || res.Skipped != 1 || res.Unchanged != 0 {
		t.Fatalf("first run: %+v", res)
	}
	if res := run(); res.OK != 0 || res.Unchanged != 3 {
		t.Fatalf("second run should skip everything: %+v", res)
	}

	// Phiên A có thêm tin nhắn và Claude Code đổi tên phiên: xuất lại, xóa file tên cũ.
	writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Phiên A mới"}`, testUserLine, testUserLine)
	touch(t, a, 2000)
	if res := run(); res.OK != 1 || res.Unchanged != 2 {
		t.Fatalf("after change: %+v", res)
	}
	if got := strings.Join(mdFiles(t, out), "|"); got != "Phiên A mới.md|Phiên B.md" {
		t.Errorf("files after rename: %s", got)
	}

	// File .md bị xóa thì xuất lại dù nguồn không đổi.
	os.Remove(filepath.Join(out, "Phiên B.md"))
	if res := run(); res.OK != 1 {
		t.Fatalf("after deleting output: %+v", res)
	}

	// Đổi tùy chọn nội dung thì xuất lại toàn bộ.
	opt.Tools = true
	if res := run(); res.OK != 2 {
		t.Fatalf("after option change: %+v", res)
	}

	// Phiên mới trùng tên với file đã xuất (không đổi) không được ghi đè lên nó.
	c := writeJSONL(t, src, "000.jsonl", `{"type":"ai-title","aiTitle":"Phiên B"}`, testUserLine)
	touch(t, c, 1000)
	if res := run(); res.OK != 1 {
		t.Fatalf("new clashing session: %+v", res)
	}
	if got := strings.Join(mdFiles(t, out), "|"); got != "Phiên A mới.md|Phiên B (2).md|Phiên B.md" {
		t.Errorf("files after clash: %s", got)
	}
}

func TestRemoveStaleKeepsEditedFile(t *testing.T) {
	src, out := tempDir(t), tempDir(t)
	a := writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Tên 1"}`, testUserLine)
	touch(t, a, 1000)
	opt := Options{TitleName: true, Incremental: true}
	if _, err := ConvertAll([]string{src}, out, opt); err != nil {
		t.Fatal(err)
	}
	// Người dùng ghi chú vào file đã xuất, rồi phiên được đổi tên.
	old := filepath.Join(out, "Tên 1.md")
	if err := os.WriteFile(old, []byte("ghi chú riêng"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeJSONL(t, src, "aaa.jsonl", `{"type":"ai-title","aiTitle":"Tên 2"}`, testUserLine)
	touch(t, a, 2000)
	if _, err := ConvertAll([]string{src}, out, opt); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(mdFiles(t, out), "|"); got != "Tên 1.md|Tên 2.md" {
		t.Errorf("edited old file must be kept: %s", got)
	}
}

func TestSplitInputs(t *testing.T) {
	got := SplitInputs(` "D:\a b\x.jsonl" ;D:\y.jsonl;; `)
	want := []string{`D:\a b\x.jsonl`, `D:\y.jsonl`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("SplitInputs = %q, want %q", got, want)
	}
	if back := SplitInputs(JoinInputs(want)); strings.Join(back, "|") != strings.Join(want, "|") {
		t.Errorf("round trip = %q", back)
	}
	if len(SplitInputs("  ;  ")) != 0 {
		t.Error("blank input must give no paths")
	}
}

func TestConvertAllSelectedFiles(t *testing.T) {
	// Giả lập thư mục projects của Claude Code với hai dự án.
	cfg := tempDir(t)
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	p1 := filepath.Join(cfg, "projects", "d--proj1")
	p2 := filepath.Join(cfg, "projects", "d--proj2")
	for _, d := range []string{p1, p2} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	a := writeJSONL(t, p1, "aaa.jsonl", `{"type":"ai-title","aiTitle":"A"}`, testUserLine)
	writeJSONL(t, p1, "bbb.jsonl", `{"type":"ai-title","aiTitle":"B"}`, testUserLine)
	c := writeJSONL(t, p2, "ccc.jsonl", `{"type":"ai-title","aiTitle":"C"}`, testUserLine)
	// File nằm ngoài thư mục projects: xuất thẳng vào thư mục đầu ra.
	other := tempDir(t)
	d := writeJSONL(t, other, "ddd.jsonl", `{"type":"ai-title","aiTitle":"D"}`, testUserLine)

	out := tempDir(t)
	opt := Options{TitleName: true, Incremental: true}
	res, err := ConvertAll([]string{a, c, d, a}, out, opt) // a trùng: chỉ chuyển một lần
	if err != nil || res.OK != 3 {
		t.Fatalf("res %+v, err %v", res, err)
	}
	for _, want := range []string{filepath.Join("d--proj1", "A.md"), filepath.Join("d--proj2", "C.md"), "D.md"} {
		if _, err := os.Stat(filepath.Join(out, want)); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "d--proj1", "B.md")); err == nil {
		t.Error("unselected file B must not be converted")
	}

	// File chọn đích danh luôn được chuyển, kể cả khi không đổi.
	if res, _ := ConvertAll([]string{a}, out, opt); res.OK != 1 || res.Unchanged != 0 {
		t.Errorf("explicit file should be reconverted: %+v", res)
	}
	// Chuyển cả thư mục projects sau đó: các file đã xuất ở đúng chỗ được tính là không đổi.
	if res, _ := ConvertAll([]string{filepath.Join(cfg, "projects")}, out, opt); res.OK != 1 || res.Unchanged != 2 {
		t.Errorf("folder run after selected files: %+v", res)
	}

	if _, err := ConvertAll([]string{filepath.Join(other, "nope.jsonl")}, out, opt); err == nil {
		t.Error("missing input must fail")
	}
	txt := filepath.Join(other, "x.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	if _, err := ConvertAll([]string{txt}, out, opt); err == nil {
		t.Error("non-.jsonl file must fail")
	}
}
