package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoggerWritesFilesAndSink(t *testing.T) {
	dir := t.TempDir()
	var ui []string
	lg.mu.Lock()
	lg.dir, lg.sink = dir, func(s string) { ui = append(ui, s) }
	lg.mu.Unlock()
	defer func() {
		lg.mu.Lock()
		lg.dir, lg.sink = "", nil
		lg.mu.Unlock()
	}()

	logInfo("hiện %d", 1)
	logDebug("chỉ debug")

	if len(ui) != 1 || ui[0] != "hiện 1" {
		t.Fatalf("sink got %q", ui)
	}
	day := time.Now().Format("2006-01-02")
	mainLog, _ := os.ReadFile(filepath.Join(dir, day+".log"))
	dbg, _ := os.ReadFile(filepath.Join(dir, "debug-"+day+".log"))
	if !strings.Contains(string(mainLog), "hiện 1") || strings.Contains(string(mainLog), "chỉ debug") {
		t.Errorf("main log: %q", mainLog)
	}
	if !strings.Contains(string(dbg), "hiện 1") || !strings.Contains(string(dbg), "chỉ debug") {
		t.Errorf("debug log: %q", dbg)
	}
}

func TestCleanOldLogsOnlyOwnFiles(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().AddDate(0, 0, -30)
	for _, n := range []string{"2026-01-01.log", "debug-2026-01-01.log", "ghichu.txt", "khac.log"} {
		p := filepath.Join(dir, n)
		_ = os.WriteFile(p, []byte("x"), 0o644)
		_ = os.Chtimes(p, old, old)
	}
	cleanOldLogs(dir, 7)
	for n, want := range map[string]bool{"2026-01-01.log": false, "debug-2026-01-01.log": false, "ghichu.txt": true, "khac.log": true} {
		if _, err := os.Stat(filepath.Join(dir, n)); (err == nil) != want {
			t.Errorf("%s exists=%v, want %v", n, err == nil, want)
		}
	}
}
