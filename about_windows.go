//go:build windows

package main

// Hộp "Giới thiệu": TaskDialog (Common Controls v6) để có link bấm được; thiếu thì dùng MessageBox.

import (
	"encoding/binary"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	projectURL = "https://github.com/hoaphuz/claude-chat2md"
	guideURL   = projectURL + "/blob/main/docs/huongdan-sudung.md"

	tdfEnableHyperlinks     = 0x0001
	tdfUseHIconMain         = 0x0002
	tdfAllowCancel          = 0x0008
	tdfPositionRelativeToWn = 0x1000
	tdcbfOK                 = 0x0001
	tdnHyperlinkClicked     = 3
)

var (
	pLstrlenW      = kernel32.NewProc("lstrlenW")
	pRtlMoveMemory = kernel32.NewProc("RtlMoveMemory")
)

var aboutCallback = syscall.NewCallback(func(hwnd, msg, wParam, lParam, data uintptr) uintptr {
	if msg == tdnHyperlinkClicked && lParam != 0 {
		openURL(wideString(lParam))
	}
	return 0
})

// wideString chép chuỗi UTF-16 (kết thúc bằng 0) ở địa chỉ p do Windows cấp sang chuỗi Go.
func wideString(p uintptr) string {
	n := call(pLstrlenW, p)
	if n == 0 || n > 4096 {
		return ""
	}
	buf := make([]uint16, n)
	call(pRtlMoveMemory, uintptr(unsafe.Pointer(&buf[0])), p, n*2)
	return syscall.UTF16ToString(buf)
}

// openURL chỉ mở hai địa chỉ của dự án (link trong hộp thoại do ta viết, nhưng vẫn kiểm tra).
func openURL(u string) {
	if u != projectURL && u != guideURL {
		logDebug("Ignored hyperlink %q", u)
		return
	}
	call(pShellExecuteW, hwndMain, ptr(wstr("open")), ptr(wstr(u)), 0, 0, swShowNormal)
}

// showAbout hiện thông tin chương trình và link tới tài liệu hướng dẫn.
func showAbout() {
	title := "Giới thiệu"
	head := appName + " v" + appVersion
	content := "Chuyển các đoạn chat với Claude Code (trong VS Code hoặc terminal) thành tệp Markdown " +
		"để đọc lại, lưu trữ, tìm kiếm hoặc chia sẻ.\n\n" +
		"Công cụ chỉ đọc dữ liệu chat, không sửa hay xóa gì, và không gửi dữ liệu đi đâu.\n\n" +
		"<a href=\"" + guideURL + "\">Tài liệu hướng dẫn sử dụng</a>\n" +
		"<a href=\"" + projectURL + "\">Trang dự án trên GitHub</a>  (mã nguồn, bản mới, báo lỗi)"
	footer := "Công cụ cộng đồng, không liên quan tới Anthropic. Giấy phép MIT."

	if pTaskDialogIndirect.Find() != nil {
		msgBox(hwndMain, head+"\n\n"+"Tài liệu hướng dẫn:\n"+guideURL+"\n\nTrang dự án:\n"+projectURL+"\n\n"+footer, mbIconInfo)
		return
	}
	pTitle, pMain, pContent, pFooter := wstr(title), wstr(head), wstr(content), wstr(footer)

	// TASKDIALOGCONFIG được khai báo với #pragma pack(1) nên dựng thủ công từng byte (bản 64-bit: 160 byte).
	cfg := make([]byte, 0, 160)
	u32 := func(v uint32) { cfg = binary.LittleEndian.AppendUint32(cfg, v) }
	u64 := func(v uintptr) { cfg = binary.LittleEndian.AppendUint64(cfg, uint64(v)) }
	u32(160)      // cbSize
	u64(hwndMain) // hwndParent
	u64(hInst)    // hInstance
	u32(tdfEnableHyperlinks | tdfUseHIconMain | tdfAllowCancel | tdfPositionRelativeToWn)
	u32(tdcbfOK)                     // dwCommonButtons
	u64(ptr(pTitle))                 // pszWindowTitle
	u64(appIcon(smCxIcon, smCyIcon)) // hMainIcon
	u64(ptr(pMain))                  // pszMainInstruction
	u64(ptr(pContent))               // pszContent
	u32(0)                           // cButtons
	u64(0)                           // pButtons
	u32(0)                           // nDefaultButton
	u32(0)                           // cRadioButtons
	u64(0)                           // pRadioButtons
	u32(0)                           // nDefaultRadioButton
	u64(0)                           // pszVerificationText
	u64(0)                           // pszExpandedInformation
	u64(0)                           // pszExpandedControlText
	u64(0)                           // pszCollapsedControlText
	u64(0)                           // hFooterIcon
	u64(ptr(pFooter))                // pszFooter
	u64(aboutCallback)               // pfCallback
	u64(0)                           // lpCallbackData
	u32(0)                           // cxWidth (tự động)
	if len(cfg) != 160 || unsafe.Sizeof(uintptr(0)) != 8 {
		msgBox(hwndMain, head+"\n\n"+guideURL, mbIconInfo)
		return
	}
	hr, _, _ := pTaskDialogIndirect.Call(uintptr(unsafe.Pointer(&cfg[0])), 0, 0, 0)
	runtime.KeepAlive(pTitle)
	runtime.KeepAlive(pMain)
	runtime.KeepAlive(pContent)
	runtime.KeepAlive(pFooter)
	if hr != 0 {
		logDebug("TaskDialogIndirect failed: 0x%x", hr)
		msgBox(hwndMain, head+"\n\nTài liệu hướng dẫn:\n"+guideURL, mbIconInfo)
	}
}
