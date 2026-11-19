package main

import (
	"os"
	"testing"
	"time"
)

// tempDir thay cho t.TempDir(): trên Windows, antivirus/Windows Search có thể
// còn giữ file vừa ghi khi test dọn dẹp ("The directory is not empty"), khiến
// t.TempDir() đánh fail cả test dù logic đúng. Ở đây xóa có thử lại, và nếu
// vẫn không xóa được thì chỉ ghi log.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "claude-chat2md-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; ; i++ {
			err := os.RemoveAll(dir)
			if err == nil {
				return
			}
			if i == 20 {
				t.Logf("không xóa được thư mục tạm %s: %v", dir, err)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
	return dir
}
