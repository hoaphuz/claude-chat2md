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
	"unicode/utf16"
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
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")

	pGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	pCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")

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

	cbsDropDown       = 0x0002
	cbsAutoHScroll    = 0x0040
	cbAddString       = 0x0143
	cbResetContent    = 0x014B
	cbSetItemHeight   = 0x0153
	cbSetDroppedWidth = 0x0160

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

	ofnHideReadOnly     = 0x00000004
	ofnNoChangeDir      = 0x00000008
	ofnAllowMultiSelect = 0x00000200
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnExplorer         = 0x00080000
	fnerrBufferTooSmall = 0x3003
	openFileBufLen      = 1 << 16 // ký tự UTF-16, đủ cho vài trăm file

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
	idChkTitle
	idChkOpen
	idChkIncremental
	idRun
	idOpenOut
	idOpenLog
	idLog
	idSrcFileBtn
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

// openFileName là OPENFILENAMEW.
type openFileName struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
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
	hTitle, hIncr, hOpen, hRun    uintptr
	hOpenOut, hLog                uintptr
	dpi                           = 96
	events                        = make(chan event, 512)
	running                       bool
	lastOutDir                    string
	runLabel                      string // chữ gốc của nút Chuyển đổi, khôi phục khi xong
	settings                      Settings
)

// ---------- nhớ nguồn và thư mục đầu ra ----------

// fillHistory nạp lịch sử vào danh sách sổ xuống của combobox, giữ chữ đang có trong ô.
func fillHistory(h uintptr, list []string) {
	cur := getText(h)
	sendMsg(h, cbResetContent, 0, 0)
	for _, d := range list {
		sendMsg(h, cbAddString, 0, ptr(wstr(d)))
	}
	setText(h, cur)
}

// remember ghi v làm giá trị cuối cùng của ô h (hSrc hoặc hOut); addHistory thì đưa vào lịch sử.
func remember(h uintptr, v string, addHistory bool) {
	if v == "" {
		return
	}
	last, recent := &settings.LastOut, &settings.RecentOut
	if h == hSrc {
		last, recent = &settings.LastSrc, &settings.RecentSrc
	}
	*last = v
	if addHistory {
		*recent = addRecent(*recent, v)
		fillHistory(h, *recent)
	}
	saveSettings(settingsPath(), settings)
}

// historyCombo tạo ô nhập có nút sổ xuống chứa lịch sử, hiện sẵn giá trị cuối (hoặc def).
func historyCombo(x, y, w, id int, recent []string, last, def string) uintptr {
	h := control("COMBOBOX", "", wsTabStop|wsVScroll|cbsDropDown|cbsAutoHScroll, 0, x, y, w, 220, id)
	sendMsg(h, cbSetItemHeight, ^uintptr(0), scale(20)) // ô chọn cao ngang các nút bên cạnh
	sendMsg(h, cbSetDroppedWidth, scale(528), 0)
	fillHistory(h, recent)
	if last == "" {
		last = def
	}
	setText(h, last)
	return h
}

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

// sourceDirHint trả về thư mục để mở sẵn trong hộp thoại chọn, suy từ ô nguồn
// (thư mục, hoặc thư mục chứa file đầu tiên trong danh sách).
func sourceDirHint() string {
	inputs := SplitInputs(getText(hSrc))
	if len(inputs) == 0 {
		return DefaultSourceDir()
	}
	p := inputs[0]
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	if st, err := os.Stat(filepath.Dir(p)); err == nil && st.IsDir() {
		return filepath.Dir(p)
	}
	return DefaultSourceDir()
}

// parseMultiSelect đọc kết quả OFN_ALLOWMULTISELECT|OFN_EXPLORER: một file thì là
// đường dẫn đầy đủ; nhiều file thì "thư mục\0tên1\0tên2\0\0".
func parseMultiSelect(buf []uint16) []string {
	var parts []string
	for start, i := 0, 0; i < len(buf); i++ {
		if buf[i] != 0 {
			continue
		}
		if i == start {
			break // hai số 0 liền nhau: hết danh sách
		}
		parts = append(parts, syscall.UTF16ToString(buf[start:i]))
		start = i + 1
	}
	if len(parts) <= 1 {
		return parts
	}
	files := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		files = append(files, filepath.Join(parts[0], name))
	}
	return files
}

// pickFiles mở hộp thoại chọn một hoặc nhiều file .jsonl.
func pickFiles(initialDir string) ([]string, bool) {
	filter := utf16.Encode([]rune("File chat Claude Code (*.jsonl)\x00*.jsonl\x00Tất cả file (*.*)\x00*.*\x00\x00"))
	buf := make([]uint16, openFileBufLen)
	ofn := openFileName{
		hwndOwner:       hwndMain,
		lpstrFilter:     &filter[0],
		nFilterIndex:    1,
		lpstrFile:       &buf[0],
		nMaxFile:        uint32(len(buf)),
		lpstrInitialDir: wstr(initialDir),
		lpstrTitle:      wstr("Chọn một hoặc nhiều file chat (giữ Ctrl hoặc Shift để chọn nhiều)"),
		flags:           ofnExplorer | ofnAllowMultiSelect | ofnFileMustExist | ofnPathMustExist | ofnHideReadOnly | ofnNoChangeDir,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	ok := call(pGetOpenFileNameW, uintptr(unsafe.Pointer(&ofn))) != 0
	runtime.KeepAlive(filter)
	if !ok {
		if code := call(pCommDlgExtendedError); code != 0 {
			logDebug("GetOpenFileNameW failed: 0x%x", code)
			if code == fnerrBufferTooSmall {
				msgBox(hwndMain, "Chọn quá nhiều file cùng lúc. Hãy chọn ít file hơn, hoặc chọn cả thư mục.", mbIconWarn)
			}
		}
		return nil, false
	}
	files := parseMultiSelect(buf)
	return files, len(files) > 0
}

func openFolder(path string) {
	call(pShellExecuteW, hwndMain, ptr(wstr("open")), ptr(wstr(path)), 0, 0, swShowNormal)
}

// ---------- chuyển đổi ----------

func startConvert() {
	if running {
		return
	}
	inputs, out := SplitInputs(getText(hSrc)), getText(hOut)
	if len(inputs) == 0 || out == "" {
		msgBox(hwndMain, "Vui lòng nhập đủ nguồn (thư mục hoặc file chat) và thư mục đầu ra.", mbIconWarn)
		return
	}
	for _, in := range inputs {
		if _, err := os.Stat(in); err != nil {
			msgBox(hwndMain, "Không tìm thấy nguồn:\n"+in, mbIconError)
			return
		}
	}
	opt := Options{Tools: isChecked(hTools), Thinking: isChecked(hThinking), TitleName: isChecked(hTitle), Incremental: isChecked(hIncr)}

	running = true
	lastOutDir = out
	remember(hSrc, JoinInputs(inputs), true)
	remember(hOut, out, true)
	call(pEnableWindow, hRun, 0)
	runLabel = getText(hRun)
	setText(hRun, " Đang chuyển…")
	setText(hLog, "")
	if len(inputs) == 1 {
		logInfo("Nguồn: %s", inputs[0])
	} else {
		logInfo("Nguồn: %d mục đã chọn", len(inputs))
	}
	logInfo("Đầu ra: %s", out)

	go func() {
		res, err := ConvertAll(inputs, out, opt)
		if err != nil {
			logError("Lỗi: %v", err)
		} else {
			logInfo("")
			logInfo("Xong: %d file Markdown mới/cập nhật, %d không đổi, bỏ qua %d file trống, %d lỗi.", res.OK, res.Unchanged, res.Skipped, res.Failed)
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
				if p, ok := pickFolder("Chọn thư mục dữ liệu chat (thường là .claude\\projects):", sourceDirHint()); ok {
					setText(hSrc, p)
					remember(hSrc, p, true)
				}
			case idSrcFileBtn:
				if files, ok := pickFiles(sourceDirHint()); ok {
					setText(hSrc, JoinInputs(files))
					remember(hSrc, JoinInputs(files), true)
				}
			case idOutBtn:
				if p, ok := pickFolder("Chọn thư mục lưu tệp Markdown:", getText(hOut)); ok {
					setText(hOut, p)
					remember(hOut, p, true)
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
		// Nhớ cả đường dẫn gõ tay hoặc chọn từ lịch sử mà chưa chuyển đổi.
		if src := getText(hSrc); src != "" && src != settings.LastSrc {
			remember(hSrc, src, false)
		}
		if out := getText(hOut); out != "" && out != settings.LastOut {
			remember(hOut, out, false)
		}
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
	const clientW, clientH = 560, 528
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
	// Ô nguồn và ô thư mục đầu ra là combobox: gõ tay được, nút sổ xuống hiện các mục dùng gần đây.
	settings = loadSettings(settingsPath())
	control("STATIC", "Dữ liệu chat của Claude Code (cả thư mục, hoặc một/nhiều tệp):", 0, 0, 16, 16, 528, 18, 0)
	hSrc = historyCombo(16, 38, 316, idSrcEdit, settings.RecentSrc, settings.LastSrc, DefaultSourceDir())
	setButtonIcon(control("BUTTON", "Thư mục…", wsTabStop|bsPushButton, 0, 340, 37, 100, 28, idSrcBtn), "chat")
	setButtonIcon(control("BUTTON", "Chọn tệp…", wsTabStop|bsPushButton, 0, 444, 37, 100, 28, idSrcFileBtn), "export-chat")

	control("STATIC", "Thư mục lưu tệp Markdown:", 0, 0, 16, 76, 528, 18, 0)
	hOut = historyCombo(16, 98, 424, idOutEdit, settings.RecentOut, settings.LastOut, DefaultOutputDir())
	setButtonIcon(control("BUTTON", "Chọn…", wsTabStop|bsPushButton, 0, 448, 97, 96, 28, idOutBtn), "markdown-file")

	control("BUTTON", "Tùy chọn", bsGroupBox, 0, 16, 138, 528, 148, 0)
	hTools = control("BUTTON", "Kèm các lệnh Claude đã chạy và kết quả (thu gọn)", wsTabStop|bsAutoCheckBox, 0, 32, 160, 500, 22, idChkTools)
	hThinking = control("BUTTON", "Kèm phần suy luận (thinking) của Claude", wsTabStop|bsAutoCheckBox, 0, 32, 184, 500, 22, idChkThinking)
	hTitle = control("BUTTON", "Đặt tên tệp theo tên phiên chat (thay vì mã session)", wsTabStop|bsAutoCheckBox, 0, 32, 208, 500, 22, idChkTitle)
	sendMsg(hTitle, bmSetCheck, bstChecked, 0)
	hIncr = control("BUTTON", "Chỉ chuyển chat mới hoặc đã thay đổi (bỏ qua chat đã xuất)", wsTabStop|bsAutoCheckBox, 0, 32, 232, 500, 22, idChkIncremental)
	sendMsg(hIncr, bmSetCheck, bstChecked, 0)
	hOpen = control("BUTTON", "Mở thư mục đầu ra khi xong", wsTabStop|bsAutoCheckBox, 0, 32, 256, 500, 22, idChkOpen)
	sendMsg(hOpen, bmSetCheck, bstChecked, 0)

	hRun = control("BUTTON", "Chuyển đổi", wsTabStop|bsDefPushButton, 0, 16, 300, 150, 34, idRun)
	hOpenOut = control("BUTTON", "Mở thư mục đầu ra", wsTabStop|bsPushButton, 0, 176, 300, 150, 34, idOpenOut)
	hOpenLog := control("BUTTON", "Mở thư mục log", wsTabStop|bsPushButton, 0, 336, 300, 150, 34, idOpenLog)
	setButtonIcon(hRun, "export-all")
	setButtonIcon(hOpenOut, "folder-output")
	setButtonIcon(hOpenLog, "history")

	control("STATIC", "Nhật ký:", 0, 0, 16, 348, 528, 18, 0)
	hLog = control("EDIT", "", wsTabStop|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, 16, 368, 528, 144, idLog)
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
