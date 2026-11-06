package ui

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procGetDlgItem       = user32.NewProc("GetDlgItem")
	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procLoadCursorW      = user32.NewProc("LoadCursorW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	WS_GROUP            = 0x00020000

	WS_EX_CLIENTEDGE = 0x00000200

	CW_USEDEFAULT = 0x80000000

	SW_SHOW      = 5
	SW_SHOWDEFAULT = 10

	IDC_ARROW = 32512

	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_COMMAND       = 0x0111
	WM_SETFONT       = 0x0030
	WM_CTLCOLORSTATIC = 0x0138

	BN_CLICKED = 0

	CBS_DROPDOWNLIST = 0x0003
	CBS_HASSTRINGS   = 0x0200
	CB_ADDSTRING     = 0x0143
	CB_GETCURSEL     = 0x0147
	CB_SETCURSEL     = 0x014E
	CB_RESETCONTENT  = 0x014B
	CBN_SELCHANGE    = 1

	BS_PUSHBUTTON = 0x00000000
	BS_AUTOCHECKBOX = 0x00000003
	BS_GROUPBOX   = 0x00000007

	ES_AUTOHSCROLL = 0x0080
	ES_LEFT        = 0x0000

	SS_LEFT = 0x0000

	BM_GETCHECK = 0x00F0
	BM_SETCHECK = 0x00F1
	BST_CHECKED = 1

	MB_OK              = 0
	MB_ICONINFORMATION = 0x40
	MB_ICONERROR       = 0x10
	MB_ICONWARNING     = 0x30
	MB_YESNO           = 0x04
	IDYES              = 6

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	FW_NORMAL = 400
	DEFAULT_CHARSET = 1
	OUT_DEFAULT_PRECIS = 0
	CLIP_DEFAULT_PRECIS = 0
	DEFAULT_QUALITY = 0
	DEFAULT_PITCH = 0
)

// Control IDs
const (
	ID_SYSCOMBO = 1001
	ID_IFCOMBO  = 1002
	ID_IP       = 1003
	ID_MASK     = 1004
	ID_GW       = 1005
	ID_DNS      = 1006
	ID_APPLY    = 1007
	ID_REFRESH  = 1008
	ID_EXIT     = 1009
	ID_BOTHSETS = 1010
	ID_PEIP     = 1011
	ID_PEIFACE  = 1012
	ID_STATUS   = 1013
	ID_CURIP    = 1014
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type msg struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// App holds UI callbacks implemented by main.
type App struct {
	OnRefresh func()
	OnSysChange func(index int)
	OnIfChange  func(index int)
	OnApply     func()
	hwnd        windows.Handle
	font        windows.Handle
	instance    windows.Handle
}

var appRef *App

func utf16Ptr(s string) *uint16 {
	p, _ := windows.UTF16PtrFromString(s)
	return p
}

func className() *uint16 { return utf16Ptr("PEIPToolWnd") }

func (a *App) wndProc(hwnd windows.Handle, msgU uint32, wParam, lParam uintptr) uintptr {
	switch msgU {
	case WM_COMMAND:
		id := int(wParam & 0xFFFF)
		notify := int((wParam >> 16) & 0xFFFF)
		switch id {
		case ID_APPLY:
			if notify == BN_CLICKED && a.OnApply != nil {
				a.OnApply()
			}
		case ID_REFRESH:
			if notify == BN_CLICKED && a.OnRefresh != nil {
				a.OnRefresh()
			}
		case ID_EXIT:
			if notify == BN_CLICKED {
				procDestroyWindow.Call(uintptr(hwnd))
			}
		case ID_SYSCOMBO:
			if notify == CBN_SELCHANGE && a.OnSysChange != nil {
				a.OnSysChange(a.ComboSel(ID_SYSCOMBO))
			}
		case ID_IFCOMBO:
			if notify == CBN_SELCHANGE && a.OnIfChange != nil {
				a.OnIfChange(a.ComboSel(ID_IFCOMBO))
			}
		}
		return 0
	case WM_CLOSE:
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msgU), wParam, lParam)
	return ret
}

func wndProc(hwnd windows.Handle, msgU uint32, wParam, lParam uintptr) uintptr {
	if appRef != nil {
		return appRef.wndProc(hwnd, msgU, wParam, lParam)
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msgU), wParam, lParam)
	return ret
}

func getModuleHandle() windows.Handle {
	h, _, _ := procGetModuleHandleW.Call(0)
	return windows.Handle(h)
}

func (a *App) createFont() {
	h, _, _ := procCreateFontW.Call(
		uintptr(16), 0, 0, 0, uintptr(FW_NORMAL),
		0, 0, 0, uintptr(DEFAULT_CHARSET),
		uintptr(OUT_DEFAULT_PRECIS), uintptr(CLIP_DEFAULT_PRECIS),
		uintptr(DEFAULT_QUALITY), uintptr(DEFAULT_PITCH),
		uintptr(unsafe.Pointer(utf16Ptr("Microsoft YaHei UI"))),
	)
	if h == 0 {
		h, _, _ = procCreateFontW.Call(
			uintptr(16), 0, 0, 0, uintptr(FW_NORMAL),
			0, 0, 0, uintptr(DEFAULT_CHARSET),
			uintptr(OUT_DEFAULT_PRECIS), uintptr(CLIP_DEFAULT_PRECIS),
			uintptr(DEFAULT_QUALITY), uintptr(DEFAULT_PITCH),
			uintptr(unsafe.Pointer(utf16Ptr("SimSun"))),
		)
	}
	a.font = windows.Handle(h)
}

func (a *App) applyFont(hwnd windows.Handle) {
	if a.font != 0 {
		procSendMessageW.Call(uintptr(hwnd), WM_SETFONT, uintptr(a.font), 1)
	}
}

func (a *App) createChild(class, title string, style, x, y, width, height int32, id int) windows.Handle {
	ex := 0
	if class == "EDIT" || class == "COMBOBOX" {
		ex = WS_EX_CLIENTEDGE
	}
	raw, _, _ := procCreateWindowExW.Call(
		uintptr(ex),
		uintptr(unsafe.Pointer(utf16Ptr(class))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(uint32(WS_CHILD|WS_VISIBLE|style)),
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		uintptr(a.hwnd),
		uintptr(id),
		uintptr(a.instance),
		0,
	)
	child := windows.Handle(raw)
	a.applyFont(child)
	return child
}

func (a *App) label(text string, x, y, w, h int32) {
	a.createChild("STATIC", text, SS_LEFT, x, y, w, h, 0)
}

// Run creates the main window and message loop. Blocks.
func (a *App) Run() error {
	runtime.LockOSThread()
	appRef = a
	a.instance = getModuleHandle()
	a.createFont()

	cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	cls := wndClassEx{
		WndProc:    syscall.NewCallback(wndProc),
		Instance:   a.instance,
		Cursor:     windows.Handle(cursor),
		Background: windows.Handle(16), // COLOR_BTNFACE+1 = 16? COLOR_BTNFACE is 15, brush = 15+1
		ClassName:  className(),
	}
	cls.Size = uint32(unsafe.Sizeof(cls))
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))
	if atom == 0 {
		return err
	}

	screenW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	screenH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	winW, winH := int32(520), int32(460)
	x := (int32(screenW) - winW) / 2
	y := (int32(screenH) - winH) / 2

	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className())),
		uintptr(unsafe.Pointer(utf16Ptr("PE 离线改 IP 工具"))),
		uintptr(WS_OVERLAPPEDWINDOW|WS_VISIBLE),
		uintptr(x), uintptr(y), uintptr(winW), uintptr(winH),
		0, 0, uintptr(a.instance), 0,
	)
	if hwnd == 0 {
		return err
	}
	a.hwnd = windows.Handle(hwnd)

	// Layout
	a.label("目标系统:", 16, 16, 80, 22)
	a.createChild("COMBOBOX", "", CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_VSCROLL|WS_TABSTOP, 100, 12, 380, 200, ID_SYSCOMBO)

	a.label("网卡接口:", 16, 48, 80, 22)
	a.createChild("COMBOBOX", "", CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_VSCROLL|WS_TABSTOP, 100, 44, 380, 200, ID_IFCOMBO)

	a.label("当前地址:", 16, 80, 80, 22)
	a.createChild("STATIC", "-", SS_LEFT, 100, 80, 380, 22, ID_CURIP)

	a.createChild("BUTTON", "目标系统静态 IP（写入离线注册表，重启进系统后生效）", BS_GROUPBOX|WS_GROUP, 12, 110, 480, 160, 0)

	a.label("IP 地址:", 28, 138, 70, 22)
	a.createChild("EDIT", "", ES_LEFT|ES_AUTOHSCROLL|WS_TABSTOP|WS_BORDER, 110, 134, 180, 24, ID_IP)

	a.label("子网掩码:", 300, 138, 70, 22)
	a.createChild("EDIT", "255.255.255.0", ES_LEFT|ES_AUTOHSCROLL|WS_TABSTOP|WS_BORDER, 375, 134, 100, 24, ID_MASK)

	a.label("默认网关:", 28, 172, 70, 22)
	a.createChild("EDIT", "", ES_LEFT|ES_AUTOHSCROLL|WS_TABSTOP|WS_BORDER, 110, 168, 180, 24, ID_GW)

	a.label("DNS:", 300, 172, 70, 22)
	a.createChild("EDIT", "", ES_LEFT|ES_AUTOHSCROLL|WS_TABSTOP|WS_BORDER, 375, 168, 100, 24, ID_DNS)

	a.createChild("BUTTON", "同时写入 ControlSet001/002（推荐）", BS_AUTOCHECKBOX|WS_TABSTOP, 28, 208, 280, 22, ID_BOTHSETS)
	a.SetCheck(ID_BOTHSETS, true)

	a.label("DNS 多个用逗号分隔，例如 223.5.5.5,8.8.8.8", 28, 236, 440, 22)

	a.createChild("BUTTON", "可选：同时改 PE 本机临时 IP（仅当前 PE 会话）", BS_GROUPBOX|WS_GROUP, 12, 280, 480, 70, 0)
	a.createChild("BUTTON", "启用", BS_AUTOCHECKBOX|WS_TABSTOP, 28, 308, 60, 22, ID_PEIP)
	a.label("PE 网卡名:", 100, 308, 70, 22)
	a.createChild("EDIT", "以太网", ES_LEFT|ES_AUTOHSCROLL|WS_TABSTOP|WS_BORDER, 175, 304, 140, 24, ID_PEIFACE)

	a.createChild("BUTTON", "刷新", BS_PUSHBUTTON|WS_TABSTOP, 16, 365, 90, 30, ID_REFRESH)
	a.createChild("BUTTON", "应用写入", BS_PUSHBUTTON|WS_TABSTOP, 120, 365, 120, 30, ID_APPLY)
	a.createChild("BUTTON", "退出", BS_PUSHBUTTON|WS_TABSTOP, 260, 365, 90, 30, ID_EXIT)

	a.createChild("STATIC", "就绪。请选择目标系统与网卡。", SS_LEFT, 16, 405, 470, 22, ID_STATUS)

	procShowWindow.Call(uintptr(a.hwnd), SW_SHOW)
	procUpdateWindow.Call(uintptr(a.hwnd))

	if a.OnRefresh != nil {
		a.OnRefresh()
	}

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func (a *App) dlg(id int) windows.Handle {
	h, _, _ := procGetDlgItem.Call(uintptr(a.hwnd), uintptr(id))
	return windows.Handle(h)
}

func (a *App) SetStatus(s string) {
	procSetWindowTextW.Call(uintptr(a.dlg(ID_STATUS)), uintptr(unsafe.Pointer(utf16Ptr(s))))
}

func (a *App) SetCurIP(s string) {
	procSetWindowTextW.Call(uintptr(a.dlg(ID_CURIP)), uintptr(unsafe.Pointer(utf16Ptr(s))))
}

func (a *App) GetText(id int) string {
	h := a.dlg(id)
	length, _, _ := procGetWindowTextLength.Call(uintptr(h))
	if length == 0 {
		buf := make([]uint16, 8)
		procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), 8)
		return windows.UTF16ToString(buf)
	}
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
	return windows.UTF16ToString(buf)
}

func (a *App) SetText(id int, s string) {
	procSetWindowTextW.Call(uintptr(a.dlg(id)), uintptr(unsafe.Pointer(utf16Ptr(s))))
}

func (a *App) ComboReset(id int) {
	procSendMessageW.Call(uintptr(a.dlg(id)), CB_RESETCONTENT, 0, 0)
}

func (a *App) ComboAdd(id int, s string) {
	procSendMessageW.Call(uintptr(a.dlg(id)), CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(s))))
}

func (a *App) ComboSel(id int) int {
	r, _, _ := procSendMessageW.Call(uintptr(a.dlg(id)), CB_GETCURSEL, 0, 0)
	return int(int32(r))
}

func (a *App) ComboSetSel(id int, index int) {
	procSendMessageW.Call(uintptr(a.dlg(id)), CB_SETCURSEL, uintptr(index), 0)
}

func (a *App) Checked(id int) bool {
	r, _, _ := procSendMessageW.Call(uintptr(a.dlg(id)), BM_GETCHECK, 0, 0)
	return r == BST_CHECKED
}

func (a *App) SetCheck(id int, on bool) {
	v := uintptr(0)
	if on {
		v = BST_CHECKED
	}
	procSendMessageW.Call(uintptr(a.dlg(id)), BM_SETCHECK, v, 0)
}

func Info(title, text string) {
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), MB_OK|MB_ICONINFORMATION)
}

func Error(title, text string) {
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), MB_OK|MB_ICONERROR)
}

func WarnYesNo(title, text string) bool {
	r, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), MB_YESNO|MB_ICONWARNING)
	return r == IDYES
}
