package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestAddRecent(t *testing.T) {
	var l []string
	l = addRecent(l, `D:\A`)
	l = addRecent(l, `D:\B`)
	l = addRecent(l, `d:\a\`) // trùng D:\A: đưa lên đầu, không nhân đôi
	l = addRecent(l, "  ")
	if want := []string{`d:\a\`, `D:\B`}; !reflect.DeepEqual(l, want) {
		t.Fatalf("got %q, want %q", l, want)
	}
	for i := 0; i < 20; i++ {
		l = addRecent(l, filepath.Join("x", string(rune('a'+i))))
	}
	if len(l) != maxRecent {
		t.Fatalf("len = %d, want %d", len(l), maxRecent)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "settings.json")
	if s := loadSettings(p); s.LastOut != "" || len(s.RecentOut) != 0 {
		t.Fatalf("missing file should give empty settings, got %+v", s)
	}
	want := Settings{
		LastSrc: `D:\a.jsonl;D:\b.jsonl`, RecentSrc: []string{`D:\a.jsonl;D:\b.jsonl`, `C:\projects`},
		LastOut: `D:\Out`, RecentOut: []string{`D:\Out`, `E:\Old`},
	}
	saveSettings(p, want)
	if got := loadSettings(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
