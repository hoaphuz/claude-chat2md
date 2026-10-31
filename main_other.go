//go:build !windows

package main

// Bản dòng lệnh cho macOS/Linux (Windows dùng giao diện trong main_windows.go).

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	tools := flag.Bool("tools", false, "kèm lệnh gọi công cụ và kết quả")
	thinking := flag.Bool("thinking", false, "kèm phần suy luận của Claude")
	version := flag.Bool("version", false, "in số phiên bản")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Cách dùng: claude-chat2md [-tools] [-thinking] [file.jsonl | thư-mục] [thư-mục-đầu-ra]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *version {
		fmt.Println("claude-chat2md " + appVersion)
		return
	}

	input, out := DefaultSourceDir(), "claude-export"
	if flag.NArg() > 0 {
		input = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		out = flag.Arg(1)
	}

	dir := initLogger()
	logDebug("%s %s started (CLI), log dir %q", appName, appVersion, dir)
	setLogSink(func(s string) { fmt.Println(s) })

	res, err := ConvertAll(input, out, Options{Tools: *tools, Thinking: *thinking})
	if err != nil {
		logDebug("ConvertAll failed: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Xong: %d file Markdown → %s (bỏ qua %d, lỗi %d)\n", res.OK, out, res.Skipped, res.Failed)
}
