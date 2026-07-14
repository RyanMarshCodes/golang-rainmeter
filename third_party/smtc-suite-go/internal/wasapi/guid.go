//go:build windows && cgo

package wasapi

import "fmt"

// GUID represents a COM interface or class identifier.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g GUID) String() string {
	return fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3],
		g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

var (
	// CLSID_MMDeviceEnumerator: {BCDE0395-E52F-467C-8E3D-C4579291692E}
	CLSID_MMDeviceEnumerator = &GUID{0xBCDE0395, 0xE52F, 0x467C, [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}

	// IID_IMMDeviceEnumerator: {A95664D2-9614-4F35-A746-DE8DB63617E6}
	IID_IMMDeviceEnumerator = &GUID{0xA95664D2, 0x9614, 0x4F35, [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}

	// IID_IAudioClient: {1CB9AD4C-DBFA-4C32-B178-C2F568A703B2}
	IID_IAudioClient = &GUID{0x1CB9AD4C, 0xDBFA, 0x4C32, [8]byte{0xB1, 0x78, 0xC2, 0xF5, 0x68, 0xA7, 0x03, 0xB2}}

	// IID_IAudioCaptureClient: {C8ADBD64-E71E-48A0-A4DE-185C395CD317}
	IID_IAudioCaptureClient = &GUID{0xC8ADBD64, 0xE71E, 0x48A0, [8]byte{0xA4, 0xDE, 0x18, 0x5C, 0x39, 0x5C, 0xD3, 0x17}}

	// KSDATAFORMAT_SUBTYPE_PCM: {00000001-0000-0010-8000-00AA00389B71}
	KSDATAFORMAT_SUBTYPE_PCM = &GUID{0x00000001, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}

	// KSDATAFORMAT_SUBTYPE_IEEE_FLOAT: {00000003-0000-0010-8000-00AA00389B71}
	KSDATAFORMAT_SUBTYPE_IEEE_FLOAT = &GUID{0x00000003, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}
)

func equalGUID(a, b *GUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
