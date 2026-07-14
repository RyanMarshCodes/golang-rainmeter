//go:build windows && cgo

package create

// #include <windows.h>
//
// extern void goStaCallback(void);
//
// static HWND g_smtcHwnd = NULL;
// static const wchar_t* smtcWC = L"SMTCSuiteGoCW";
//
// static LRESULT CALLBACK SMTCWndProc(HWND h, UINT m, WPARAM w, LPARAM l) {
//     if (m == WM_CREATE) return 0;
//     if (m == WM_APP) { goStaCallback(); return 0; }
//     return DefWindowProcW(h, m, w, l);
// }
//
// static HWND smtcCreateWindow(void) {
//     WNDCLASSEXW w = {0};
//     w.cbSize = sizeof(w);
//     w.lpfnWndProc = SMTCWndProc;
//     w.hInstance = GetModuleHandleW(0);
//     w.lpszClassName = smtcWC;
//     RegisterClassExW(&w);
//     HWND h = CreateWindowExW(WS_EX_TOOLWINDOW, smtcWC, L"", WS_POPUP,
//         0, 0, 1, 1, 0, 0, w.hInstance, 0);
//     g_smtcHwnd = h;
//     return h;
// }
//
// static void smtcRunPump(void) {
//     MSG msg;
//     while (GetMessageW(&msg, 0, 0, 0) > 0) {
//         TranslateMessage(&msg);
//         DispatchMessageW(&msg);
//     }
// }
//
// static void smtcQuitPump(void) { PostQuitMessage(0); }
// static void smtcWakeSTA(void) { if (g_smtcHwnd) PostMessageW(g_smtcHwnd, WM_APP, 0, 0); }
// static HWND smtcGetWindow(void) { return g_smtcHwnd; }
import "C"
import (
	"fmt"
	"runtime"
	"sync"
)

type staThread struct {
	ready  chan struct{}
	err    error
	done   chan struct{}
	cmdCh  chan func()
}

var (
	globalSTA     *staThread
	globalSTAMu   sync.Mutex
	globalSTAOnce sync.Once
)

func getSTAThread() (*staThread, error) {
	globalSTAMu.Lock()
	if globalSTA != nil {
		t := globalSTA
		globalSTAMu.Unlock()
		return t, nil
	}
	globalSTAMu.Unlock()

	var onceErr error
	globalSTAOnce.Do(func() {
		t := &staThread{
			ready: make(chan struct{}),
			done:  make(chan struct{}),
			cmdCh: make(chan func()),
		}
		go t.run()
		<-t.ready
		if t.err != nil {
			onceErr = t.err
			return
		}
		globalSTAMu.Lock()
		globalSTA = t
		globalSTAMu.Unlock()
	})
	if onceErr != nil {
		return nil, onceErr
	}
	return globalSTA, nil
}

func (t *staThread) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr := C.CoInitializeEx(nil, 2) // COINIT_APARTMENTTHREADED
	if uint32(hr) != 0 && uint32(hr) != 1 { // S_OK=0, S_FALSE=1
		t.err = fmt.Errorf("CoInitializeEx(STA): 0x%08X", uint32(hr))
		close(t.ready)
		return
	}

	hwnd := C.smtcCreateWindow()
	if hwnd == nil {
		t.err = fmt.Errorf("CreateSMTCWindow failed")
		C.CoUninitialize()
		close(t.ready)
		return
	}
	close(t.ready)

	// Process commands and window messages
	done := false
	for !done {
		select {
		case fn := <-t.cmdCh:
			if fn != nil {
				fn()
			}
		default:
			// Pump Windows messages (non-blocking)
			var msg C.MSG
			for C.PeekMessageW(&msg, nil, 0, 0, 1 /*PM_REMOVE*/) != 0 {
				if msg.message == 0x0012 { // WM_QUIT
					done = true
					break
				}
				C.TranslateMessage(&msg)
				C.DispatchMessageW(&msg)
			}
			// Also check for quit signaled via cmdCh (close on nil)
			if !done {
				// Brief sleep to avoid busy-waiting
				runtime.Gosched()
			}
		}
	}

	C.DestroyWindow(C.smtcGetWindow())
	C.CoUninitialize()
	close(t.done)
}

// Do dispatches a function to the STA thread and blocks until complete.
func (t *staThread) Do(fn func()) {
	done := make(chan struct{})
	t.cmdCh <- func() {
		fn()
		close(done)
	}
	C.smtcWakeSTA() // wake up the pump to process the command
	<-done
}

// Shutdown stops the message pump.
func (t *staThread) Shutdown() {
	t.cmdCh <- func() {
		C.smtcQuitPump()
	}
	<-t.done
}

// HWND returns the window handle.
func (t *staThread) HWND() C.HWND {
	return C.smtcGetWindow()
}

//export goStaCallback
func goStaCallback() {
	// Called from the window proc when WM_APP is received.
	// Just return; the PeekMessage loop will pick up pending commands.
}
