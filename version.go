package main

import (
	_ "embed"
	"strings"
)

// File VERSION là nguồn duy nhất của số phiên bản
//
//go:embed VERSION
var versionFile []byte

// appVersion đọc từ VERSION lúc build; "dev" nếu file trống
var appVersion = parseVersion(versionFile)

func parseVersion(b []byte) string {
	if v := strings.TrimSpace(string(b)); v != "" {
		return v
	}
	return "dev"
}
