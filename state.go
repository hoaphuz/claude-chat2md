package main

// File trạng thái trong thư mục đầu ra: nhớ file chat nào đã xuất, xuất ra file .md nào,
// để lần sau chỉ chuyển các file mới hoặc đã thay đổi (Options.Incremental).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const stateFileName = ".claude-chat2md-state.json"

// stateEntry là thông tin lần xuất gần nhất của một file .jsonl nguồn.
type stateEntry struct {
	Size    int64  `json:"size"`               // kích thước file nguồn
	ModTime int64  `json:"mtime"`              // thời điểm sửa file nguồn (UnixNano)
	Format  string `json:"format"`             // phiên bản + tùy chọn đã dùng (formatKey)
	Out     string `json:"out,omitempty"`      // file .md, tương đối so với thư mục đầu ra; "" = chat trống
	OutSize int64  `json:"outSize,omitempty"`  // kích thước file .md lúc ghi
	OutMod  int64  `json:"outMtime,omitempty"` // thời điểm ghi file .md (UnixNano)
}

type stateDoc struct {
	Files map[string]stateEntry `json:"files"` // khóa: đường dẫn tuyệt đối của file nguồn
}

// formatKey: đổi phiên bản hoặc tùy chọn ảnh hưởng nội dung/tên file thì phải xuất lại.
func formatKey(opt Options) string {
	return fmt.Sprintf("%s tools=%t thinking=%t title=%t", appVersion, opt.Tools, opt.Thinking, opt.TitleName)
}

// stateKey là khóa ổn định của một file nguồn (không phân biệt hoa thường trên Windows).
func stateKey(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

// loadState đọc file trạng thái; thiếu hoặc hỏng thì trả về rỗng (xuất lại toàn bộ).
func loadState(outRoot string) map[string]stateEntry {
	files := map[string]stateEntry{}
	data, err := os.ReadFile(filepath.Join(outRoot, stateFileName))
	if err != nil {
		if !os.IsNotExist(err) {
			logDebug("Read state failed: %v", err)
		}
		return files
	}
	var doc stateDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		logDebug("State file corrupt, ignoring: %v", err)
		return files
	}
	for k, v := range doc.Files {
		files[k] = v
	}
	return files
}

// saveState ghi file trạng thái qua file tạm rồi đổi tên, tránh để lại file ghi dở.
func saveState(outRoot string, files map[string]stateEntry) error {
	data, err := json.MarshalIndent(stateDoc{Files: files}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outRoot, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(outRoot, stateFileName)
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// fileSig trả về kích thước và thời điểm sửa; ok=false nếu không đọc được.
func fileSig(p string) (size, mod int64, ok bool) {
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return 0, 0, false
	}
	return fi.Size(), fi.ModTime().UnixNano(), true
}

// removeStale xóa file .md cũ khi phiên được xuất ra tên khác (vd. đổi tên phiên),
// nhưng chỉ khi file cũ vẫn y nguyên như lúc công cụ ghi (người dùng chưa sửa).
func removeStale(outRoot string, prev stateEntry) {
	p := filepath.Join(outRoot, prev.Out)
	size, mod, ok := fileSig(p)
	if !ok {
		return
	}
	if size != prev.OutSize || mod != prev.OutMod {
		logInfo("Giữ lại file cũ vì đã bị sửa: %s", prev.Out)
		return
	}
	if err := os.Remove(p); err != nil {
		logDebug("Remove stale %s failed: %v", p, err)
		return
	}
	logInfo("Đã xóa file cũ (tên phiên đã đổi): %s", prev.Out)
}
