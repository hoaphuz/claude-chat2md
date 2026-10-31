//go:build windows

package main

// Giao diện Win32 thuần (không thư viện ngoài, không cgo).
// Build: go build -ldflags "-H=windowsgui -s" -o claude-chat2md.exe

import (
	"embed"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

// Toàn bộ giao diện Win32 phải chạy trên cùng một luồng hệ điều hành.
func init() { runtime.LockOSThread() }

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW     = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pSendMessageW         = user32.NewProc("SendMessageW")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pShowWindow           = user32.NewProc("ShowWindow")
	pUpdateWindow         = user32.NewProc("UpdateWindow")
	pEnableWindow         = user32.NewProc("EnableWindow")
	pSetWindowTextW       = user32.NewProc("SetWindowTextW")
	pGetWindowTextW       = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	pMessageBoxW          = user32.NewProc("MessageBoxW")
	pLoadCursorW          = user32.NewProc("LoadCursorW")
	pLoadIconW            = user32.NewProc("LoadIconW")
	pLoadImageW           = user32.NewProc("LoadImageW")
	pCreateIconFromResEx  = user32.NewProc("CreateIconFromResourceEx")
	pGetDC                = user32.NewProc("GetDC")
	pReleaseDC            = user32.NewProc("ReleaseDC")
	pGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	pAdjustWindowRectEx   = user32.NewProc("AdjustWindowRectEx")
	pSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	pSetFocus             = user32.NewProc("SetFocus")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pCreateActCtxW    = kernel32.NewProc("CreateActCtxW")
	pActivateActCtx   = kernel32.NewProc("ActivateActCtx")

	pCreateFontW     = gdi32.NewProc("CreateFontW")
	pGetDeviceCaps   = gdi32.NewProc("GetDeviceCaps")
	pGetSysColorBrsh = user32.NewProc("GetSysColorBrush")

	pSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	pShellExecuteW        = shell32.NewProc("ShellExecuteW")

	pCoInitializeEx = ole32.NewProc("CoInitializeEx")
	pCoTaskMemFree  = ole32.NewProc("CoTaskMemFree")

	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
)

const (
	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsMinimizeBox = 0x00020000
	wsVisible     = 0x10000000
	wsChild       = 0x40000000
	wsTabStop     = 0x00010000
	wsVScroll     = 0x00200000

	wsExClientEdge    = 0x00000200
	wsExControlParent = 0x00010000

	esMultiline   = 0x0004
	esAutoVScroll = 0x0040
	esAutoHScroll = 0x0080
	esReadOnly    = 0x0800

	bsPushButton    = 0x0
	bsDefPushButton = 0x1
	bsAutoCheckBox  = 0x3
	bsGroupBox      = 0x7

	wmDestroy  = 0x0002
	wmSetFont  = 0x0030
	wmCommand  = 0x0111
	wmApp      = 0x8000
	wmAppEvent = wmApp + 1

	bnClicked  = 0
	bmGetCheck = 0x00F0
	bmSetCheck = 0x00F1
	bstChecked = 1
	bmSetImage = 0x00F7

	imageIcon  = 1
	smCxIcon   = 11
	smCyIcon   = 12
	smCxSmIcon = 49
	smCySmIcon = 50
	appIconID  = 1 // RT_GROUP_ICON do tools/mkrsrc sinh
	iconResVer = 0x00030000

	emSetSel     = 0x00B1
	emScrollCrt  = 0x00B7
	emReplaceSel = 0x00C2
	emLimitText  = 0x00C5

	swShow       = 5
	swShowNormal = 1

	colorBtnFace = 15
	idcArrow     = 32512
	idiApp       = 32512
	logPixelsY   = 90
	smCxScreen   = 0
	smCyScreen   = 1

	mbIconError = 0x10
	mbIconWarn  = 0x30
	mbIconInfo  = 0x40

	bifReturnOnlyFsDirs = 0x0001
	bifEditBox          = 0x0010
	bifNewDialogStyle   = 0x0040
	bffmInitialized     = 1
	bffmSetSelectionW   = 0x0400 + 103

	iccStandardClasses = 0x4000
	iccWin95Classes    = 0x00FF
	coinitApartment    = 0x2
)

const (
	idSrcEdit = 101 + iota
	idSrcBtn
	idOutEdit
	idOutBtn
	idChkTools
	idChkThinking
	idChkOpen
	idRun
	idOpenOut
	idOpenLog
	idLog
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct{ x, y int32 }

type msgT struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
	private uint32
}

type rect struct{ left, top, right, bottom int32 }

type actCtx struct {
	cbSize                 uint32
	dwFlags                uint32
	lpSource               *uint16
	wProcessorArchitecture uint16
	wLangID                uint16
	lpAssemblyDirectory    *uint16
	lpResourceName         *uint16
	lpApplicationName      *uint16
	hModule                uintptr
}

type initCommonControlsEx struct {
	dwSize uint32
	dwICC  uint32
}

type browseInfo struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

// ---------- tiện ích ----------

func call(p *syscall.LazyProc, args ...uintptr) uintptr {
	r, _, _ := p.Call(args...)
	return r
}

func wstr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func ptr(p *uint16) uintptr { return uintptr(unsafe.Pointer(p)) }

func sendMsg(hwnd, msg, w, l uintptr) uintptr { return call(pSendMessageW, hwnd, msg, w, l) }

func getText(hwnd uintptr) string {
	n := call(pGetWindowTextLengthW, hwnd)
	buf := make([]uint16, n+1)
	call(pGetWindowTextW, hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func setText(hwnd uintptr, s string) { call(pSetWindowTextW, hwnd, ptr(wstr(s))) }

func isChecked(hwnd uintptr) bool { return sendMsg(hwnd, bmGetCheck, 0, 0) == bstChecked }

func msgBox(owner uintptr, text string, icon uintptr) {
	call(pMessageBoxW, owner, ptr(wstr(text)), ptr(wstr("Claude → Markdown")), icon)
}

// Bật giao diện hiện đại (Common Controls v6) bằng activation context,
// để không cần nhúng manifest vào file .exe.
func enableVisualStyles() {
	const manifest = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0"
        processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
    </dependentAssembly>
  </dependency>
</assembly>`
	path := filepath.Join(os.TempDir(), "claude-chat2md.manifest")
	if os.WriteFile(path, []byte(manifest), 0o644) != nil {
		return
	}
	ac := actCtx{lpSource: wstr(path)}
	ac.cbSize = uint32(unsafe.Sizeof(ac))
	h := call(pCreateActCtxW, uintptr(unsafe.Pointer(&ac)))
	if h == ^uintptr(0) { // INVALID_HANDLE_VALUE
		return
	}
	var cookie uintptr
	call(pActivateActCtx, h, uintptr(unsafe.Pointer(&cookie)))
}

// ---------- icon ----------

// Icon nút lấy từ bộ icon thiết kế (bản cho nền sáng, 16px và @2x 32px).
//
//go:embed assets/icons/png/light/*.png
var iconFS embed.FS

// appIcon nạp icon ứng dụng nhúng trong .exe (rsrc_windows_amd64.syso) theo
// kích thước hệ thống; thiếu tài nguyên thì dùng icon mặc định của Windows.
func appIcon(cxMetric, cyMetric uintptr) uintptr {
	cx, cy := call(pGetSystemMetrics, cxMetric), call(pGetSystemMetrics, cyMetric)
	if h := call(pLoadImageW, hInst, appIconID, imageIcon, cx, cy, 0); h != 0 {
		return h
	}
	logDebug("app icon resource not found, using IDI_APPLICATION")
	return call(pLoadIconW, 0, idiApp)
}

// buttonIcon tạo HICON từ PNG nhúng; màn hình DPI cao (≥144) dùng bản @2x.
// Trả về 0 nếu không tạo được (nút vẫn hiện chữ bình thường).
func buttonIcon(name string) uintptr {
	file, size := name+".png", 16
	if dpi >= 144 {
		file, size = name+"@2x.png", 32
	}
	data, err := iconFS.ReadFile("assets/icons/png/light/" + file)
	if err != nil {
		logDebug("button icon %s: %v", file, err)
		return 0
	}
	h := call(pCreateIconFromResEx, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)),
		1, iconResVer, uintptr(size), uintptr(size), 0)
	if h == 0 {
		logDebug("CreateIconFromResourceEx failed for %s", file)
	}
	return h
}

// setButtonIcon gắn icon bên trái chữ của nút (cần Common Controls v6),
// thêm một khoảng trắng đầu chữ để icon không dính sát chữ.
func setButtonIcon(btn uintptr, name string) {
	if h := buttonIcon(name); h != 0 {
		sendMsg(btn, bmSetImage, imageIcon, h)
		setText(btn, " "+getText(btn))
	}
}

// ---------- trạng thái ứng dụng ----------

type event struct {
	text string
	done bool
	res  Result
	err  error
}

// postEvent gửi sự kiện về luồng giao diện; gọi được từ mọi goroutine.
func postEvent(e event) {
	events <- e
	call(pPostMessageW, hwndMain, wmAppEvent, 0, 0)
}

var (
	hInst, hwndMain, hFont        uintptr
	hSrc, hOut, hTools, hThinking uintptr
	hOpen, hRun, hOpenOut, hLog   uintptr
	dpi                           = 96
	events                        = make(chan event, 512)
	running                       bool
	lastOutDir                    string
	runLabel                      string // chữ gốc của nút Chuyển đổi, khôi phục khi xong
)

func scale(v int) uintptr { return uintptr(v * dpi / 96) }

func control(class, text string, style, exStyle uintptr, x, y, w, h, id int) uintptr {
	hw := call(pCreateWindowExW, exStyle, ptr(wstr(class)), ptr(wstr(text)),
		wsChild|wsVisible|style,
		scale(x), scale(y), scale(w), scale(h),
		hwndMain, uintptr(id), hInst, 0)
	sendMsg(hw, wmSetFont, hFont, 1)
	return hw
}

func appendLog(s string) {
	n := call(pGetWindowTextLengthW, hLog)
	sendMsg(hLog, emSetSel, n, n)
	sendMsg(hLog, emReplaceSel, 0, ptr(wstr(s+"\r\n")))
	sendMsg(hLog, emScrollCrt, 0, 0)
}

// ---------- chọn thư mục ----------

var browseCallback = syscall.NewCallback(func(hwnd, msg, lParam, data uintptr) uintptr {
	if msg == bffmInitialized && data != 0 {
		sendMsg(hwnd, bffmSetSelectionW, 1, data)
	}
	return 0
})

func pickFolder(title, initial string) (string, bool) {
	display := make([]uint16, 260)
	initW := wstr(initial)
	bi := browseInfo{
		hwndOwner:      hwndMain,
		pszDisplayName: &display[0],
		lpszTitle:      wstr(title),
		ulFlags:        bifReturnOnlyFsDirs | bifNewDialogStyle | bifEditBox,
		lpfn:           browseCallback,
		lParam:         ptr(initW),
	}
	pidl := call(pSHBrowseForFolderW, uintptr(unsafe.Pointer(&bi)))
	runtime.KeepAlive(initW)
	runtime.KeepAlive(display)
	if pidl == 0 {
		return "", false
	}
	defer call(pCoTaskMemFree, pidl)
	buf := make([]uint16, 1024)
	if call(pSHGetPathFromIDListW, pidl, uintptr(unsafe.Pointer(&buf[0]))) == 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

func openFolder(path string) {
	call(pShellExecuteW, hwndMain, ptr(wstr("open")), ptr(wstr(path)), 0, 0, swShowNormal)
}

// ---------- chuyển đổi ----------

func startConvert() {
	if running {
		return
	}
	src, out := getText(hSrc), getText(hOut)
	if src == "" || out == "" {
		msgBox(hwndMain, "Vui lòng nhập đủ thư mục nguồn và thư mục đầu ra.", mbIconWarn)
		return
	}
	if _, err := os.Stat(src); err != nil {
		msgBox(hwndMain, "Không tìm thấy thư mục nguồn:\n"+src, mbIconError)
		return
	}
	opt := Options{Tools: isChecked(hTools), Thinking: isChecked(hThinking)}

	running = true
	lastOutDir = out
	call(pEnableWindow, hRun, 0)
	runLabel = getText(hRun)
	setText(hRun, " Đang chuyển…")
	setText(hLog, "")
	logInfo("Nguồn: %s", src)
	logInfo("Đầu ra: %s", out)

	go func() {
		res, err := ConvertAll(src, out, opt)
		if err != nil {
			logError("Lỗi: %v", err)
		} else {
			logInfo("")
			logInfo("Xong: %d file Markdown, bỏ qua %d file trống, %d lỗi.", res.OK, res.Skipped, res.Failed)
		}
		postEvent(event{done: true, res: res, err: err})
	}()
}

// Xử lý sự kiện từ luồng chuyển đổi (chạy trên luồng giao diện).
func drainEvents() {
	for {
		select {
		case e := <-events:
			if !e.done {
				appendLog(e.text)
				continue
			}
			running = false
			call(pEnableWindow, hRun, 1)
			setText(hRun, runLabel)
			if e.err != nil {
				msgBox(hwndMain, e.err.Error(), mbIconError)
				continue
			}
			if e.res.OK > 0 && isChecked(hOpen) {
				openFolder(lastOutDir)
			}
		default:
			return
		}
	}
}

// ---------- cửa sổ ----------

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCommand:
		if (wParam>>16)&0xFFFF == bnClicked {
			switch wParam & 0xFFFF {
			case idSrcBtn:
				if p, ok := pickFolder("Chọn thư mục dữ liệu chat (thường là .claude\\projects):", getText(hSrc)); ok {
					setText(hSrc, p)
				}
			case idOutBtn:
				if p, ok := pickFolder("Chọn thư mục lưu file Markdown:", getText(hOut)); ok {
					setText(hOut, p)
				}
			case idRun:
				startConvert()
			case idOpenOut:
				out := getText(hOut)
				if _, err := os.Stat(out); err != nil {
					msgBox(hwndMain, "Thư mục đầu ra chưa tồn tại. Hãy chuyển đổi trước.", mbIconInfo)
				} else {
					openFolder(out)
				}
			case idOpenLog:
				if d := logDir(); d == "" {
					msgBox(hwndMain, "Không có thư mục ghi được nên nhật ký chưa được lưu ra file.", mbIconInfo)
				} else {
					openFolder(d)
				}
			}
		}
		return 0
	case wmAppEvent:
		drainEvents()
		return 0
	case wmDestroy:
		call(pPostQuitMessage, 0)
		return 0
	}
	return call(pDefWindowProcW, hwnd, msg, wParam, lParam)
}

func main() {
	dir := initLogger()
	logDebug("%s %s started (%s/%s, %s), log dir %q", appName, appVersion, runtime.GOOS, runtime.GOARCH, runtime.Version(), dir)

	call(pSetProcessDPIAware)
	enableVisualStyles()
	icc := initCommonControlsEx{dwICC: iccStandardClasses | iccWin95Classes}
	icc.dwSize = uint32(unsafe.Sizeof(icc))
	call(pInitCommonControlsEx, uintptr(unsafe.Pointer(&icc)))
	call(pCoInitializeEx, 0, coinitApartment)

	hdc := call(pGetDC, 0)
	if d := int(call(pGetDeviceCaps, hdc, logPixelsY)); d > 0 {
		dpi = d
	}
	call(pReleaseDC, 0, hdc)

	hInst = call(pGetModuleHandleW, 0)
	hFont = call(pCreateFontW, uintptr(int32(-9*dpi/72))&0xFFFFFFFF, 0, 0, 0, 400, 0, 0, 0,
		1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, ptr(wstr("Segoe UI")))

	className := wstr("ClaudeToMarkdown")
	wc := wndClassEx{
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInst,
		hIcon:         appIcon(smCxIcon, smCyIcon),
		hCursor:       call(pLoadCursorW, 0, idcArrow),
		hbrBackground: call(pGetSysColorBrsh, colorBtnFace),
		lpszClassName: className,
		hIconSm:       appIcon(smCxSmIcon, smCySmIcon),
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	call(pRegisterClassExW, uintptr(unsafe.Pointer(&wc)))

	// Kích thước vùng nội dung (px ở 96 DPI)
	const clientW, clientH = 560, 480
	style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	r := rect{0, 0, int32(scale(clientW)), int32(scale(clientH))}
	call(pAdjustWindowRectEx, uintptr(unsafe.Pointer(&r)), style, 0, wsExControlParent)
	winW, winH := int(r.right-r.left), int(r.bottom-r.top)
	x := (int(call(pGetSystemMetrics, smCxScreen)) - winW) / 2
	y := (int(call(pGetSystemMetrics, smCyScreen)) - winH) / 2

	hwndMain = call(pCreateWindowExW, wsExControlParent, ptr(className),
		ptr(wstr("Xuất chat Claude Code sang Markdown v"+appVersion)), style,
		uintptr(x), uintptr(y), uintptr(winW), uintptr(winH), 0, 0, hInst, 0)

	// --- các điều khiển ---
	control("STATIC", "Thư mục dữ liệu chat của Claude Code:", 0, 0, 16, 16, 528, 18, 0)
	hSrc = control("EDIT", DefaultSourceDir(), wsTabStop|esAutoHScroll, wsExClientEdge, 16, 38, 424, 26, idSrcEdit)
	setButtonIcon(control("BUTTON", "Chọn…", wsTabStop|bsPushButton, 0, 448, 37, 96, 28, idSrcBtn), "chat")

	control("STATIC", "Thư mục lưu file Markdown:", 0, 0, 16, 76, 528, 18, 0)
	hOut = control("EDIT", DefaultOutputDir(), wsTabStop|esAutoHScroll, wsExClientEdge, 16, 98, 424, 26, idOutEdit)
	setButtonIcon(control("BUTTON", "Chọn…", wsTabStop|bsPushButton, 0, 448, 97, 96, 28, idOutBtn), "markdown-file")

	control("BUTTON", "Tùy chọn", bsGroupBox, 0, 16, 138, 528, 100, 0)
	hTools = control("BUTTON", "Kèm các lệnh Claude đã chạy và kết quả (thu gọn)", wsTabStop|bsAutoCheckBox, 0, 32, 160, 500, 22, idChkTools)
	hThinking = control("BUTTON", "Kèm phần suy luận (thinking) của Claude", wsTabStop|bsAutoCheckBox, 0, 32, 184, 500, 22, idChkThinking)
	hOpen = control("BUTTON", "Mở thư mục đầu ra khi xong", wsTabStop|bsAutoCheckBox, 0, 32, 208, 500, 22, idChkOpen)
	sendMsg(hOpen, bmSetCheck, bstChecked, 0)

	hRun = control("BUTTON", "Chuyển đổi", wsTabStop|bsDefPushButton, 0, 16, 252, 150, 34, idRun)
	hOpenOut = control("BUTTON", "Mở thư mục đầu ra", wsTabStop|bsPushButton, 0, 176, 252, 150, 34, idOpenOut)
	hOpenLog := control("BUTTON", "Mở thư mục log", wsTabStop|bsPushButton, 0, 336, 252, 150, 34, idOpenLog)
	setButtonIcon(hRun, "export-all")
	setButtonIcon(hOpenOut, "folder-output")
	setButtonIcon(hOpenLog, "history")

	control("STATIC", "Nhật ký:", 0, 0, 16, 300, 528, 18, 0)
	hLog = control("EDIT", "", wsTabStop|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, 16, 320, 528, 144, idLog)
	sendMsg(hLog, emLimitText, 0, 0) // bỏ giới hạn độ dài nhật ký
	setLogSink(func(s string) { postEvent(event{text: s}) })
	setText(hLog, "Bấm \"Chuyển đổi\" để bắt đầu. Dữ liệu gốc chỉ được đọc, không bị sửa hay xóa.")

	call(pShowWindow, hwndMain, swShow)
	call(pUpdateWindow, hwndMain)
	call(pSetFocus, hRun)

	var m msgT
	for {
		r := int32(call(pGetMessageW, uintptr(unsafe.Pointer(&m)), 0, 0, 0))
		if r <= 0 {
			break
		}
		// IsDialogMessage giúp phím Tab di chuyển giữa các ô
		if call(pIsDialogMessageW, hwndMain, uintptr(unsafe.Pointer(&m))) != 0 {
			continue
		}
		call(pTranslateMessage, uintptr(unsafe.Pointer(&m)))
		call(pDispatchMessageW, uintptr(unsafe.Pointer(&m)))
	}
}
