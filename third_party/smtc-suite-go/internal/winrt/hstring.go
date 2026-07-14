//go:build windows && cgo

package winrt

// #include "c/helpers.h"
import "C"
import (
	"unsafe"
)

// HSTRING wraps a Windows Runtime HSTRING handle.
// HSTRING is a reference-counted immutable string type used by WinRT.
type HSTRING struct {
	h C.HSTRING
}

// NewHSTRING creates an HSTRING from a Go string.
// The caller must call Delete() to free the HSTRING.
func NewHSTRING(s string) (*HSTRING, error) {
	hstr := &HSTRING{}
	if len(s) == 0 {
		hr := C.WindowsCreateString(nil, 0, &hstr.h)
		if hr < 0 {
			return nil, hresultError("WindowsCreateString", hr)
		}
		return hstr, nil
	}

	utf16, err := utf16FromString(s)
	if err != nil {
		return nil, err
	}

	hr := C.WindowsCreateString(
		(*C.WCHAR)(unsafe.Pointer(&utf16[0])),
		C.UINT32(len(utf16)),
		&hstr.h,
	)
	if hr < 0 {
		return nil, hresultError("WindowsCreateString", hr)
	}
	return hstr, nil
}

// Delete frees the HSTRING's underlying Windows Runtime string.
func (h *HSTRING) Delete() {
	if h != nil && h.h != nil {
		C.WindowsDeleteString(h.h)
		h.h = nil
	}
}

// String converts the HSTRING to a Go string.
func (h *HSTRING) String() string {
	if h == nil || h.h == nil {
		return ""
	}

	var length C.UINT32
	buf := C.WindowsGetStringRawBuffer(h.h, &length)
	if buf == nil || length == 0 {
		return ""
	}

	utf16Slice := unsafe.Slice((*uint16)(unsafe.Pointer(buf)), int(length))
	return string(utf16Decode(utf16Slice))
}

// Raw returns the underlying C HSTRING handle for passing to COM methods.
func (h *HSTRING) Raw() C.HSTRING {
	if h == nil {
		return nil
	}
	return h.h
}

// utf16FromString converts a Go string to UTF-16 (little-endian, []uint16).
func utf16FromString(s string) ([]uint16, error) {
	runes := []rune(s)
	result := make([]uint16, 0, len(runes)+1)
	for _, r := range runes {
		if r <= 0xFFFF {
			result = append(result, uint16(r))
		} else {
			r -= 0x10000
			result = append(result, uint16(0xD800|(r>>10)))
			result = append(result, uint16(0xDC00|(r&0x3FF)))
		}
	}
	return result, nil
}

// utf16Decode converts a UTF-16 []uint16 slice to a Go string.
func utf16Decode(utf16 []uint16) string {
	runes := make([]rune, 0, len(utf16))
	for i := 0; i < len(utf16); i++ {
		r := rune(utf16[i])
		if r >= 0xD800 && r <= 0xDBFF && i+1 < len(utf16) {
			lo := rune(utf16[i+1])
			if lo >= 0xDC00 && lo <= 0xDFFF {
				r = (r-0xD800)<<10 | (lo - 0xDC00) + 0x10000
				i++
			}
		}
		runes = append(runes, r)
	}
	return string(runes)
}
