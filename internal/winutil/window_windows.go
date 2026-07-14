//go:build windows

package winutil

import (
	"image/color"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
)

const (
	wsExLayered     = 0x00080000
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExAppWindow   = 0x00040000

	wsPopup      = 0x80000000
	wsCaption    = 0x00C00000 // WS_BORDER | WS_DLGFRAME
	wsThickFrame = 0x00040000
	wsSysMenu    = 0x00080000
	wsMinimizeBox = 0x00020000
	wsMaximizeBox = 0x00010000

	hwndTopmost    = ^uintptr(0) // -1
	hwndNotTopmost = ^uintptr(1) // -2

	swpNoSize        = 0x0001
	swpNoMove        = 0x0002
	swpNoZOrder      = 0x0004
	swpNoActivate    = 0x0010
	swpFrameChanged  = 0x0020
	swpShowWindow    = 0x0040

	lwaAlpha = 0x00000002
)

var (
	gwlStyle   = ^uintptr(0) - 15 // GWL_STYLE = -16
	gwlExStyle = ^uintptr(0) - 19 // GWL_EXSTYLE = -20
)

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procClientToScreen       = user32.NewProc("ClientToScreen")
	procAdjustWindowRectEx   = user32.NewProc("AdjustWindowRectEx")
	procGetWindowLongPtr     = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr     = user32.NewProc("SetWindowLongPtrW")
	procSetLayeredAttr       = user32.NewProc("SetLayeredWindowAttributes")
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procIsWindowVisible      = user32.NewProc("IsWindowVisible")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

// ClearColor is the widget/theme clear fill.
// With GLFW_TRANSPARENT_FRAMEBUFFER it must be fully transparent (A=0) so
// glyph AA keeps a real alpha channel instead of blending onto opaque black
// (which color-key then leaves as dark fringing on thin fonts).
func ClearColor() color.Color {
	return color.NRGBA{R: 0, G: 0, B: 0, A: 0}
}

// ChromaKeyColor is a legacy alias for ClearColor.
func ChromaKeyColor() color.Color { return ClearColor() }

// ApplyDesktopProps sets position, optional size, z-order, click-through, and
// optional whole-window opacity. Pass width/height <= 0 to leave size unchanged.
// Returns false if the native HWND is not ready yet (common on first Show).
// Desktop show-through comes from the transparent framebuffer + A=0 clear color
// — not from LWA_COLORKEY.
func ApplyDesktopProps(w fyne.Window, x, y int, width, height int, alwaysOnTop, transparent, clickThrough bool, opacity float32) bool {
	applied := false
	withHWND(w, func(hwnd windows.HWND) {
		applied = true
		insertAfter := hwndNotTopmost
		if alwaysOnTop {
			insertAfter = hwndTopmost
		}
		flags := uintptr(swpNoActivate | swpShowWindow)
		cw, ch := uintptr(0), uintptr(0)
		if width > 0 && height > 0 {
			cw, ch = uintptr(int32(width)), uintptr(int32(height))
		} else {
			flags |= swpNoSize
		}
		_, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			insertAfter,
			uintptr(int32(x)),
			uintptr(int32(y)),
			cw,
			ch,
			flags,
		)

		alpha := opacityToByte(opacity)
		useAlpha := alpha < 255
		layered := clickThrough || useAlpha

		ex, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlExStyle)
		ex &^= wsExAppWindow
		ex |= wsExToolWindow
		if layered {
			ex |= wsExLayered
		} else if !transparent {
			ex &^= wsExLayered
		}
		if clickThrough {
			ex |= wsExTransparent
		} else {
			ex &^= wsExTransparent
		}
		_, _, _ = procSetWindowLongPtr.Call(uintptr(hwnd), gwlExStyle, ex)

		if useAlpha {
			_, _, _ = procSetLayeredAttr.Call(uintptr(hwnd), 0, uintptr(alpha), lwaAlpha)
		}
	})
	return applied
}

func opacityToByte(opacity float32) byte {
	if opacity <= 0 {
		return 255 // unset / default → fully opaque content
	}
	if opacity >= 1 {
		return 255
	}
	return byte(opacity*255 + 0.5)
}

// Bounds returns the window's screen rectangle in physical pixels.
func Bounds(w fyne.Window) (x, y, width, height int, ok bool) {
	var r rect
	var got bool
	withHWND(w, func(hwnd windows.HWND) {
		ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
		got = ret != 0
	})
	if !got {
		return 0, 0, 0, 0, false
	}
	return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top), true
}

// ClientBounds returns the client area's screen origin and size in physical pixels.
func ClientBounds(w fyne.Window) (x, y, width, height int, ok bool) {
	var got bool
	var ox, oy, cw, ch int32
	withHWND(w, func(hwnd windows.HWND) {
		var client rect
		if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&client))); ret == 0 {
			return
		}
		origin := struct{ X, Y int32 }{}
		if ret, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&origin))); ret == 0 {
			return
		}
		ox, oy = origin.X, origin.Y
		cw, ch = client.Right-client.Left, client.Bottom-client.Top
		got = cw > 0 && ch > 0
	})
	if !got {
		return 0, 0, 0, 0, false
	}
	return int(ox), int(oy), int(cw), int(ch), true
}

// CursorPos returns the current screen cursor position in physical pixels.
func CursorPos() (x, y int, ok bool) {
	var p struct{ X, Y int32 }
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(p.X), int(p.Y), true
}

// IsVisible reports whether the native window is currently shown.
func IsVisible(w fyne.Window) bool {
	var visible bool
	withHWND(w, func(hwnd windows.HWND) {
		r, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		visible = r != 0
	})
	return visible
}

// SetPosition moves the window without resizing.
func SetPosition(w fyne.Window, x, y int) {
	withHWND(w, func(hwnd windows.HWND) {
		_, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			0,
			uintptr(int32(x)),
			uintptr(int32(y)),
			0,
			0,
			uintptr(swpNoSize|swpNoActivate),
		)
	})
}

// SetBounds moves and resizes the window in physical pixels.
func SetBounds(w fyne.Window, x, y, width, height int) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	withHWND(w, func(hwnd windows.HWND) {
		_, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			0,
			uintptr(int32(x)),
			uintptr(int32(y)),
			uintptr(int32(width)),
			uintptr(int32(height)),
			uintptr(swpNoActivate),
		)
	})
}

// SetClientBounds places the window so its client area occupies the given
// screen rectangle (physical pixels), accounting for the current frame style.
func SetClientBounds(w fyne.Window, x, y, width, height int) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	withHWND(w, func(hwnd windows.HWND) {
		style, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlStyle)
		exStyle, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlExStyle)
		outer := rect{Left: 0, Top: 0, Right: int32(width), Bottom: int32(height)}
		_, _, _ = procAdjustWindowRectEx.Call(
			uintptr(unsafe.Pointer(&outer)),
			style,
			0,
			exStyle,
		)
		wx := int32(x) + outer.Left
		wy := int32(y) + outer.Top
		ww := outer.Right - outer.Left
		wh := outer.Bottom - outer.Top
		if ww < 1 {
			ww = 1
		}
		if wh < 1 {
			wh = 1
		}
		_, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			0,
			uintptr(wx),
			uintptr(wy),
			uintptr(ww),
			uintptr(wh),
			uintptr(swpNoActivate),
		)
	})
}

// SetNativeChrome toggles OS title bar + resize frame for edit mode.
// Client content size and screen position are preserved across the toggle.
func SetNativeChrome(w fyne.Window, enabled bool) {
	withHWND(w, func(hwnd windows.HWND) {
		var client rect
		if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&client))); ret == 0 {
			return
		}
		cw := client.Right - client.Left
		ch := client.Bottom - client.Top
		if cw < 1 || ch < 1 {
			return
		}

		origin := struct{ X, Y int32 }{}
		if ret, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&origin))); ret == 0 {
			return
		}

		style, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlStyle)
		exStyle, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlExStyle)

		wantChrome := enabled
		hasChrome := style&wsCaption != 0 && style&wsThickFrame != 0 && style&wsPopup == 0
		if wantChrome == hasChrome {
			return
		}

		if wantChrome {
			style &^= wsPopup
			style |= wsCaption | wsThickFrame | wsSysMenu | wsMinimizeBox
		} else {
			style &^= wsCaption | wsThickFrame | wsSysMenu | wsMinimizeBox | wsMaximizeBox
			style |= wsPopup
		}
		_, _, _ = procSetWindowLongPtr.Call(uintptr(hwnd), gwlStyle, style)

		outer := rect{Left: 0, Top: 0, Right: cw, Bottom: ch}
		_, _, _ = procAdjustWindowRectEx.Call(
			uintptr(unsafe.Pointer(&outer)),
			style,
			0,
			exStyle,
		)
		wx := origin.X + outer.Left
		wy := origin.Y + outer.Top
		ww := outer.Right - outer.Left
		wh := outer.Bottom - outer.Top
		if ww < 1 {
			ww = 1
		}
		if wh < 1 {
			wh = 1
		}
		_, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			0,
			uintptr(wx),
			uintptr(wy),
			uintptr(ww),
			uintptr(wh),
			uintptr(swpNoZOrder|swpNoActivate|swpFrameChanged),
		)
	})
}

func withHWND(w fyne.Window, fn func(hwnd windows.HWND)) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		winCtx, ok := ctx.(driver.WindowsWindowContext)
		if !ok || winCtx.HWND == 0 {
			return
		}
		fn(windows.HWND(winCtx.HWND))
	})
}
