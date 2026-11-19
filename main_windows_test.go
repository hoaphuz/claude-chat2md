//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
	"unsafe"
)

func TestParseMultiSelect(t *testing.T) {
	enc := func(s string) []uint16 { return utf16.Encode([]rune(s)) }
	one := parseMultiSelect(enc(`C:\p\a.jsonl` + "\x00\x00"))
	if strings.Join(one, "|") != `C:\p\a.jsonl` {
		t.Errorf("single = %q", one)
	}
	many := parseMultiSelect(enc(`C:\p` + "\x00a.jsonl\x00b.jsonl\x00\x00"))
	want := filepath.Join(`C:\p`, "a.jsonl") + "|" + filepath.Join(`C:\p`, "b.jsonl")
	if strings.Join(many, "|") != want {
		t.Errorf("multi = %q", many)
	}
	if len(parseMultiSelect(make([]uint16, 4))) != 0 {
		t.Error("empty buffer must give no files")
	}
}

// OPENFILENAMEW trên Windows 64-bit dài 152 byte.
func TestOpenFileNameSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) == 8 && unsafe.Sizeof(openFileName{}) != 152 {
		t.Errorf("sizeof(openFileName) = %d, want 152", unsafe.Sizeof(openFileName{}))
	}
}

func TestAppIconResource(t *testing.T) {
	hInst = call(pGetModuleHandleW, 0)
	cx := call(pGetSystemMetrics, smCxIcon)
	if call(pLoadImageW, hInst, appIconID, imageIcon, cx, cx, 0) == 0 {
		t.Fatal("app icon resource missing; run: go run ./tools/mkrsrc")
	}
}

func TestButtonIcons(t *testing.T) {
	defer func(d int) { dpi = d }(dpi)
	for _, d := range []int{96, 144} {
		dpi = d
		for _, name := range []string{"chat", "markdown-file", "export-all", "export-chat", "folder-output", "history"} {
			if buttonIcon(name) == 0 {
				t.Errorf("buttonIcon(%q) at %d DPI failed", name, d)
			}
		}
	}
}
