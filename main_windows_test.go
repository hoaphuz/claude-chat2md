//go:build windows

package main

import "testing"

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
		for _, name := range []string{"chat", "markdown-file", "export-all", "folder-output", "history"} {
			if buttonIcon(name) == 0 {
				t.Errorf("buttonIcon(%q) at %d DPI failed", name, d)
			}
		}
	}
}
