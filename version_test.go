package main

import "testing"

func TestParseVersion(t *testing.T) {
	if v := parseVersion([]byte(" 1.2.3\r\n")); v != "1.2.3" {
		t.Errorf("got %q", v)
	}
	if v := parseVersion(nil); v != "dev" {
		t.Errorf("got %q", v)
	}
	if appVersion == "dev" {
		t.Error("VERSION not embedded")
	}
}
