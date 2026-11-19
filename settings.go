package main

// Cài đặt giao diện được nhớ giữa các lần mở: thư mục đầu ra cuối cùng và lịch sử
// các thư mục đầu ra đã dùng gần đây, tương tự cho ô nguồn. Lưu ở settings.json cạnh file exe (bản portable).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxRecent        = 10
	settingsFileName = "settings.json"
)

type Settings struct {
	LastSrc   string   `json:"lastSrc,omitempty"` // ô nguồn: thư mục, hoặc các file cách nhau bằng ;
	RecentSrc []string `json:"recentSrc,omitempty"`
	LastOut   string   `json:"lastOut,omitempty"`
	RecentOut []string `json:"recentOut,omitempty"`

	// Thư mục VS Code portable đã thấy đang chạy, để đọc nhóm chat cả khi VS Code đã tắt.
	VSCodeDirs []string `json:"vscodeDirs,omitempty"`
}

var cachedSettingsPath *string

// settingsPath trả về đường dẫn file cài đặt, chọn một lần mỗi lần chạy, cùng thứ tự
// với thư mục log: dev\ của dự án (khi `go run`) → cạnh file exe → %APPDATA%\claude-chat2md
// (khi thư mục exe không ghi được). "" nếu không có chỗ nào ghi được.
func settingsPath() string {
	if cachedSettingsPath != nil {
		return *cachedSettingsPath
	}
	exe := exeDir()
	var candidates []string
	if root := devRoot(exe); root != "" {
		candidates = append(candidates, filepath.Join(root, "dev"))
	}
	if exe != "" {
		candidates = append(candidates, exe)
	}
	if base, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(base, appName))
	}
	p := ""
	for _, d := range candidates {
		// File cài đặt đã có ở chỗ ưu tiên hơn thì dùng luôn, kể cả khi tạm thời không ghi được.
		if _, err := os.Stat(filepath.Join(d, settingsFileName)); err == nil || isWritableDir(d) {
			p = filepath.Join(d, settingsFileName)
			break
		}
	}
	logDebug("settings file: %q", p)
	cachedSettingsPath = &p
	return p
}

// loadSettings đọc file cài đặt; thiếu hoặc hỏng thì trả về cài đặt trống.
func loadSettings(path string) Settings {
	var s Settings
	if path == "" {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logDebug("read settings %s: %v", path, err)
		}
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		logDebug("parse settings %s: %v", path, err)
		return Settings{}
	}
	return s
}

func saveSettings(path string, s Settings) {
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err == nil {
		if err = os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
			err = os.WriteFile(path, data, 0o644)
		}
	}
	if err != nil {
		logDebug("save settings %s: %v", path, err)
	}
}

// addRecent đưa dir lên đầu danh sách, bỏ bản trùng (không phân biệt hoa thường,
// bỏ dấu \ cuối) và giữ tối đa maxRecent mục.
func addRecent(list []string, dir string) []string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return list
	}
	key := strings.ToLower(filepath.Clean(dir))
	out := []string{dir}
	for _, d := range list {
		if strings.ToLower(filepath.Clean(d)) != key && len(out) < maxRecent {
			out = append(out, d)
		}
	}
	return out
}
