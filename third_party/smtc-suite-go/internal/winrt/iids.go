//go:build windows && cgo

package winrt

import (
	"unsafe"
)

// GUID represents a COM/WinRT interface identifier (16 bytes).
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// IID_IUnknown: {00000000-0000-0000-C000-000000000046}
var IID_IUnknown = &GUID{
	0x00000000, 0x0000, 0x0000,
	[8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// IID_IInspectable: {AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}
var IID_IInspectable = &GUID{
	0xAF86E2E0, 0xB12D, 0x4C6A,
	[8]byte{0x9C, 0x5A, 0xD7, 0xAA, 0x65, 0x10, 0x1E, 0x90},
}

// IID_IAsyncInfo: {00000036-0000-0000-C000-000000000046}
var IID_IAsyncInfo = &GUID{
	0x00000036, 0x0000, 0x0000,
	[8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// IID_IAgileObject: {94EA2B94-E9CC-49E0-C0FF-EE64CA8F5B90}
var IID_IAgileObject = &GUID{
	0x94EA2B94, 0xE9CC, 0x49E0,
	[8]byte{0xC0, 0xFF, 0xEE, 0x64, 0xCA, 0x8F, 0x5B, 0x90},
}

// ---- Windows.Media.Control IIDs (from windows-rs authoritative source) ----

// IID_IGSMTCSessionManagerStatics: {2050C4EE-11A0-57DE-AED7-C97C70338245}
var IID_IGSMTCSessionManagerStatics = &GUID{
	0x2050C4EE, 0x11A0, 0x57DE,
	[8]byte{0xAE, 0xD7, 0xC9, 0x7C, 0x70, 0x33, 0x82, 0x45},
}

// IID_IGSMTCSessionManager: {CACE8EAC-E86E-504A-AB31-5FF8FF1BCE49}
var IID_IGSMTCSessionManager = &GUID{
	0xCACE8EAC, 0xE86E, 0x504A,
	[8]byte{0xAB, 0x31, 0x5F, 0xF8, 0xFF, 0x1B, 0xCE, 0x49},
}

// IID_IGSMTCSession: {7148C835-9B14-5AE2-AB85-DC9B1C14E1A8}
var IID_IGSMTCSession = &GUID{
	0x7148C835, 0x9B14, 0x5AE2,
	[8]byte{0xAB, 0x85, 0xDC, 0x9B, 0x1C, 0x14, 0xE1, 0xA8},
}

// IID_IGSMTCSessionMediaProperties: {68856CF6-ADB4-54B2-AC16-05837907ACB6}
var IID_IGSMTCSessionMediaProperties = &GUID{
	0x68856CF6, 0xADB4, 0x54B2,
	[8]byte{0xAC, 0x16, 0x05, 0x83, 0x79, 0x07, 0xAC, 0xB6},
}

// IID_IGSMTCSessionPlaybackInfo: {94B4B6CF-E8BA-51AD-87A7-C10ADE106127}
var IID_IGSMTCSessionPlaybackInfo = &GUID{
	0x94B4B6CF, 0xE8BA, 0x51AD,
	[8]byte{0x87, 0xA7, 0xC1, 0x0A, 0xDE, 0x10, 0x61, 0x27},
}

// IID_IGSMTCSessionPlaybackControls: {6501A3E6-BC7A-503A-BB1B-68F158F3FB03}
var IID_IGSMTCSessionPlaybackControls = &GUID{
	0x6501A3E6, 0xBC7A, 0x503A,
	[8]byte{0xBB, 0x1B, 0x68, 0xF1, 0x58, 0xF3, 0xFB, 0x03},
}

// IID_IGSMTCSessionTimelineProperties: {EDE34136-6F25-588D-8ECF-EA5B6735AAA5}
var IID_IGSMTCSessionTimelineProperties = &GUID{
	0xEDE34136, 0x6F25, 0x588D,
	[8]byte{0x8E, 0xCF, 0xEA, 0x5B, 0x67, 0x35, 0xAA, 0xA5},
}

// IID_ITypedEventHandler_GSMTCSessionManager_CurrentSessionChangedEventArgs: {228BD0ED-1FA2-5E9B-A6EC-42566173103B}
var IID_ITypedEventHandler_GSMTCSessionManager_CurrentSessionChangedEventArgs = &GUID{
	0x228BD0ED, 0x1FA2, 0x5E9B,
	[8]byte{0xA6, 0xEC, 0x42, 0x56, 0x61, 0x73, 0x10, 0x3B},
}

// IID_ITypedEventHandler_GSMTCSessionManager_SessionsChangedEventArgs: {2E2A8630-DC8C-530A-9746-BC984D4B029E}
var IID_ITypedEventHandler_GSMTCSessionManager_SessionsChangedEventArgs = &GUID{
	0x2E2A8630, 0xDC8C, 0x530A,
	[8]byte{0x97, 0x46, 0xBC, 0x98, 0x4D, 0x4B, 0x02, 0x9E},
}

// IID_ITypedEventHandler_GSMTCSession_TimelinePropertiesChangedEventArgs: {E8BF62AF-FAC1-5FFF-9053-0BF191AE777E}
var IID_ITypedEventHandler_GSMTCSession_TimelinePropertiesChangedEventArgs = &GUID{
	0xE8BF62AF, 0xFAC1, 0x5FFF,
	[8]byte{0x90, 0x53, 0x0B, 0xF1, 0x91, 0xAE, 0x77, 0x7E},
}

// IID_ITypedEventHandler_GSMTCSession_PlaybackInfoChangedEventArgs: {2BDF1426-D41F-5896-897F-EFC0B0FA7392}
var IID_ITypedEventHandler_GSMTCSession_PlaybackInfoChangedEventArgs = &GUID{
	0x2BDF1426, 0xD41F, 0x5896,
	[8]byte{0x89, 0x7F, 0xEF, 0xC0, 0xB0, 0xFA, 0x73, 0x92},
}

// IID_ITypedEventHandler_GSMTCSession_MediaPropertiesChangedEventArgs: {0F2CE2B7-AFA7-5ED0-8CB6-8C40CF9B3A5F}
var IID_ITypedEventHandler_GSMTCSession_MediaPropertiesChangedEventArgs = &GUID{
	0x0F2CE2B7, 0xAFA7, 0x5ED0,
	[8]byte{0x8C, 0xB6, 0x8C, 0x40, 0xCF, 0x9B, 0x3A, 0x5F},
}

// ---- Windows.Media IIDs (Create module) ----

// IID_IMediaPlayer2: {3c841218-2123-4fc5-9082-2f883f77bdf5}
var IID_IMediaPlayer2 = &GUID{
	0x3c841218, 0x2123, 0x4fc5,
	[8]byte{0x90, 0x82, 0x2f, 0x88, 0x3f, 0x77, 0xbd, 0xf5},
}

// IID_IMediaPlayerSource2: {82449b9f-7322-4c0b-b03b-3e69a48260c5}
var IID_IMediaPlayerSource2 = &GUID{
	0x82449b9f, 0x7322, 0x4c0b,
	[8]byte{0xb0, 0x3b, 0x3e, 0x69, 0xa4, 0x82, 0x60, 0xc5},
}

// IID_IMediaPlaybackSource: {ef9dc2bc-9317-4696-b051-2bad643177b5}
var IID_IMediaPlaybackSource = &GUID{
	0xef9dc2bc, 0x9317, 0x4696,
	[8]byte{0xb0, 0x51, 0x2b, 0xad, 0x64, 0x31, 0x77, 0xb5},
}

// IID_IMediaPlaybackItemFactory: {7133fce1-1769-4ff9-a7c1-38d2c4d42360}
var IID_IMediaPlaybackItemFactory = &GUID{
	0x7133fce1, 0x1769, 0x4ff9,
	[8]byte{0xa7, 0xc1, 0x38, 0xd2, 0xc4, 0xd4, 0x23, 0x60},
}

// IID_IMediaPlaybackItem2: {d859d171-d7ef-4b81-ac1f-f40493cbb091}
var IID_IMediaPlaybackItem2 = &GUID{
	0xd859d171, 0xd7ef, 0x4b81,
	[8]byte{0xac, 0x1f, 0xf4, 0x04, 0x93, 0xcb, 0xb0, 0x91},
}

// IID_IMediaSourceStatics: {f77d6fa4-4652-410e-b1d8-e9a5e245a45c}
var IID_IMediaSourceStatics = &GUID{
	0xf77d6fa4, 0x4652, 0x410e,
	[8]byte{0xb1, 0xd8, 0xe9, 0xa5, 0xe2, 0x45, 0xa4, 0x5c},
}

// IID_ISystemMediaTransportControlsInterop: {ddb0472d-c911-4a1f-86d9-dc3d71a95f5a}
var IID_ISystemMediaTransportControlsInterop = &GUID{
	0xddb0472d, 0xc911, 0x4a1f,
	[8]byte{0x86, 0xd9, 0xdc, 0x3d, 0x71, 0xa9, 0x5f, 0x5a},
}

// IID_ISystemMediaTransportControls: {99FA3FF4-1742-42A6-902E-087D41F965EC}
var IID_ISystemMediaTransportControls = &GUID{
	0x99FA3FF4, 0x1742, 0x42A6,
	[8]byte{0x90, 0x2E, 0x08, 0x7D, 0x41, 0xF9, 0x65, 0xEC},
}

// IID_ISystemMediaTransportControls2: {ea98d2f6-7f3c-4af2-a586-72889808efb1}
var IID_ISystemMediaTransportControls2 = &GUID{
	0xea98d2f6, 0x7f3c, 0x4af2,
	[8]byte{0xa5, 0x86, 0x72, 0x88, 0x98, 0x08, 0xef, 0xb1},
}

// IID_ISystemMediaTransportControlsTimelineProperties: {5125316a-c3a2-475b-8507-93534dc88f15}
var IID_ISystemMediaTransportControlsTimelineProperties = &GUID{
	0x5125316a, 0xc3a2, 0x475b,
	[8]byte{0x85, 0x07, 0x93, 0x53, 0x4d, 0xc8, 0x8f, 0x15},
}

// IID_IStorageFileStatics: {5984c710-daf2-43c8-8bb4-a4d3eacfd03f}
var IID_IStorageFileStatics = &GUID{
	0x5984c710, 0xdaf2, 0x43c8,
	[8]byte{0x8b, 0xb4, 0xa4, 0xd3, 0xea, 0xcf, 0xd0, 0x3f},
}

// IID_IRandomAccessStreamReferenceStatics: {857309dc-3fbf-4e7d-986f-ef3b1a07a964}
var IID_IRandomAccessStreamReferenceStatics = &GUID{
	0x857309dc, 0x3fbf, 0x4e7d,
	[8]byte{0x98, 0x6f, 0xef, 0x3b, 0x1a, 0x07, 0xa9, 0x64},
}

// IID_IRandomAccessStreamReference: {33EE3134-1DD6-4E3A-8067-D1C162E8642B}
var IID_IRandomAccessStreamReference = &GUID{
	0x33EE3134, 0x1DD6, 0x4E3A,
	[8]byte{0x80, 0x67, 0xD1, 0xC1, 0x62, 0xE8, 0x64, 0x2B},
}

// IID_IRandomAccessStream: {905A0FE1-BC53-11DF-8C49-001E4FC686DA}
var IID_IRandomAccessStream = &GUID{
	0x905A0FE1, 0xBC53, 0x11DF,
	[8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
}

// IID_IInputStream: {905A0FE2-BC53-11DF-8C49-001E4FC686DA}
var IID_IInputStream = &GUID{
	0x905A0FE2, 0xBC53, 0x11DF,
	[8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
}

// IID_IBuffer: {905A0FE0-BC53-11DF-8C49-001E4FC686DA}
var IID_IBuffer = &GUID{
	0x905A0FE0, 0xBC53, 0x11DF,
	[8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
}

// IID_IBufferFactory: {71AF914D-C10F-484B-BC50-14BC623B3A27}
var IID_IBufferFactory = &GUID{
	0x71AF914D, 0xC10F, 0x484B,
	[8]byte{0xBC, 0x50, 0x14, 0xBC, 0x62, 0x3B, 0x3A, 0x27},
}

// IID_IBufferByteAccess: {905A0FEF-BC53-11DF-8C49-001E4FC686DA}
var IID_IBufferByteAccess = &GUID{
	0x905A0FEF, 0xBC53, 0x11DF,
	[8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
}

// IID_IUriRuntimeClassFactory: {44a9796f-723e-4fdf-a218-033e75b0c084}
var IID_IUriRuntimeClassFactory = &GUID{
	0x44a9796f, 0x723e, 0x4fdf,
	[8]byte{0xa2, 0x18, 0x03, 0x3e, 0x75, 0xb0, 0xc0, 0x84},
}

// IID_ISystemMediaTransportControlsDisplayUpdater
var IID_ISystemMediaTransportControlsDisplayUpdater *GUID

// ptr returns the GUID as an unsafe.Pointer for passing to COM methods.
func (g *GUID) ptr() unsafe.Pointer {
	if g == nil {
		return nil
	}
	return unsafe.Pointer(g)
}
