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
	title := flag.Bool("title", false, "đặt tên file theo tên phiên chat thay vì mã session")
	update := flag.Bool("update", false, "chỉ chuyển chat mới hoặc đã thay đổi kể từ lần xuất trước")
	outFlag := flag.String("o", "", "thư mục đầu ra; khi có cờ này, mọi tham số còn lại là nguồn")
	version := flag.Bool("version", false, "in số phiên bản")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Cách dùng: claude-chat2md [-tools] [-thinking] [-title] [-update] [file.jsonl | thư-mục] [thư-mục-đầu-ra]")
		fmt.Fprintln(os.Stderr, "      hoặc: claude-chat2md [cờ…] -o thư-mục-đầu-ra file.jsonl|thư-mục …")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *version {
		fmt.Println("claude-chat2md " + appVersion)
		return
	}

	// Có -o: mọi tham số là nguồn (nhiều file/thư mục). Không có -o: [nguồn] [đầu-ra] như cũ.
	inputs := []string{DefaultSourceDir()}
	out := *outFlag
	switch {
	case out != "":
		if flag.NArg() > 0 {
			inputs = flag.Args()
		}
	default:
		out = "claude-export"
		if flag.NArg() > 0 {
			inputs = SplitInputs(flag.Arg(0))
		}
		if flag.NArg() > 1 {
			out = flag.Arg(1)
		}
		if flag.NArg() > 2 {
			fmt.Fprintln(os.Stderr, "Nhiều nguồn thì dùng -o <thư-mục-đầu-ra>, ví dụ: claude-chat2md -o out a.jsonl b.jsonl")
			os.Exit(2)
		}
	}

	dir := initLogger()
	logDebug("%s %s started (CLI), log dir %q", appName, appVersion, dir)
	setLogSink(func(s string) { fmt.Println(s) })

	res, err := ConvertAll(inputs, out, Options{Tools: *tools, Thinking: *thinking, TitleName: *title, Incremental: *update})
	if err != nil {
		logDebug("ConvertAll failed: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Xong: %d file Markdown mới/cập nhật → %s (không đổi %d, trống %d, lỗi %d)\n", res.OK, out, res.Unchanged, res.Skipped, res.Failed)
}
