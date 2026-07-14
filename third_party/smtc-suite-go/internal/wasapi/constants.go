//go:build windows && cgo

package wasapi

const (
	clsctxAll = 0x17

	eRender  = 0
	eConsole = 0

	audioClientShareModeShared = 0

	audioClientStreamFlagsLoopback = 0x00020000

	audioClientBufferFlagsDataDiscontinuity = 0x00000001
	audioClientBufferFlagsSilent            = 0x00000002
	audioClientBufferFlagsTimestampError    = 0x00000004

	waveFormatPCM        = 0x0001
	waveFormatIEEEFloat  = 0x0003
	waveFormatExtensible = 0xFFFE
)

const defaultBufferDuration100ns = 1 * 1000 * 1000 // 100 ms.

const (
	slotIMMDeviceEnumeratorEnumAudioEndpoints             = 3
	slotIMMDeviceEnumeratorGetDefaultAudioEndpoint        = 4
	slotIMMDeviceEnumeratorGetDevice                      = 5
	slotIMMDeviceEnumeratorRegisterEndpointNotification   = 6
	slotIMMDeviceEnumeratorUnregisterEndpointNotification = 7

	slotIMMDeviceActivate          = 3
	slotIMMDeviceOpenPropertyStore = 4
	slotIMMDeviceGetID             = 5
	slotIMMDeviceGetState          = 6

	slotIAudioClientInitialize        = 3
	slotIAudioClientGetBufferSize     = 4
	slotIAudioClientGetStreamLatency  = 5
	slotIAudioClientGetCurrentPadding = 6
	slotIAudioClientIsFormatSupported = 7
	slotIAudioClientGetMixFormat      = 8
	slotIAudioClientGetDevicePeriod   = 9
	slotIAudioClientStart             = 10
	slotIAudioClientStop              = 11
	slotIAudioClientReset             = 12
	slotIAudioClientSetEventHandle    = 13
	slotIAudioClientGetService        = 14

	slotIAudioCaptureClientGetBuffer         = 3
	slotIAudioCaptureClientReleaseBuffer     = 4
	slotIAudioCaptureClientGetNextPacketSize = 5
)
