package main

// Nhật ký hai tầng:
//   - logInfo/logWarn/logError: dòng ngắn (tiếng Việt) hiện trên giao diện, ghi vào <ngày>.log
//   - logDebug: chi tiết cho sửa lỗi, chỉ ghi vào debug-<ngày>.log (debug cũng chứa mọi dòng ở trên)
// Nhật ký chỉ ghi đường dẫn và số liệu, không chép nội dung chat.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	appName          = "claude-chat2md"
	logRetentionDays = 7
	maxDebugEntry    = 20000 // giới hạn độ dài 1 dòng log chi tiết
)

type logger struct {
	mu   sync.Mutex
	dir  string       // "" = không ghi file (chưa init, hoặc không có chỗ ghi)
	sink func(string) // nhận dòng hiện trên giao diện; được gọi từ nhiều goroutine
}

var lg = &logger{}

// initLogger chọn thư mục log và dọn log cũ; trả về thư mục log ("" nếu không ghi được).
func initLogger() string {
	dir := findLogDir()
	lg.mu.Lock()
	lg.dir = dir
	lg.mu.Unlock()
	if dir != "" {
		go cleanOldLogs(dir, logRetentionDays)
	}
	return dir
}

func setLogSink(f func(string)) {
	lg.mu.Lock()
	lg.sink = f
	lg.mu.Unlock()
}

func logDir() string {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	return lg.dir
}

func logInfo(format string, args ...any)  { lg.write("INFO", fmt.Sprintf(format, args...), true) }
func logWarn(format string, args ...any)  { lg.write("WARN", fmt.Sprintf(format, args...), true) }
func logError(format string, args ...any) { lg.write("ERROR", fmt.Sprintf(format, args...), true) }

// logDebug ghi CHỈ vào debug-<ngày>.log, không hiện trên giao diện.
func logDebug(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if len(msg) > maxDebugEntry {
		msg = msg[:maxDebugEntry] + fmt.Sprintf("... (truncated, %d chars total)", len(msg))
	}
	lg.write("DEBUG", msg, false)
}

func (l *logger) write(level, msg string, ui bool) {
	now := time.Now()
	l.mu.Lock()
	sink := l.sink
	if l.dir != "" {
		day := now.Format("2006-01-02")
		if ui {
			appendLine(filepath.Join(l.dir, day+".log"),
				fmt.Sprintf("[%s] [%s] %s", now.Format("15:04:05"), level, msg))
		}
		appendLine(filepath.Join(l.dir, "debug-"+day+".log"),
			fmt.Sprintf("[%s] [%s] %s", now.Format("15:04:05.000"), level, msg))
	}
	l.mu.Unlock()
	if ui && sink != nil {
		sink(msg)
	}
}

func appendLine(path, line string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

// Chỉ xóa đúng file log của ứng dụng, không đụng file khác trong thư mục.
var reLogFile = regexp.MustCompile(`^(debug-)?\d{4}-\d{2}-\d{2}\.log$`)

func cleanOldLogs(dir string, days int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, e := range entries {
		if e.IsDir() || !reLogFile.MatchString(e.Name()) {
			continue
		}
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// findLogDir: dev\logs của dự án (khi `go run`) → logs cạnh file exe → thư mục cache người dùng
// (%LOCALAPPDATA%\claude-chat2md\logs trên Windows).
func findLogDir() string {
	exeDir := ""
	if p, err := os.Executable(); err == nil {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
		exeDir = filepath.Dir(p)
	}
	var candidates []string
	if root := devRoot(exeDir); root != "" {
		candidates = append(candidates, filepath.Join(root, "dev", "logs"))
	}
	if exeDir != "" {
		candidates = append(candidates, filepath.Join(exeDir, "logs"))
	}
	if base, err := os.UserCacheDir(); err == nil {
		candidates = append(candidates, filepath.Join(base, appName, "logs"))
	}
	for _, d := range candidates {
		if isWritableDir(d) {
			return d
		}
	}
	return ""
}

// devRoot: exe do `go run` tạo nằm trong thư mục go-build tạm/cache; khi đó tìm thư mục
// dự án (có go.mod) từ thư mục hiện tại đi ngược lên.
func devRoot(exeDir string) string {
	if !strings.Contains(strings.ToLower(exeDir), "go-build") {
		return ""
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func isWritableDir(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".write_test_*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
