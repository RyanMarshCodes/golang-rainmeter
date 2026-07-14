//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSessionMediaProperties vtable layout.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6]  get_Title(out *HSTRING) HRESULT
//	[7]  get_Subtitle(out *HSTRING) HRESULT
//	[8]  get_AlbumArtist(out *HSTRING) HRESULT
//	[9]  get_Artist(out *HSTRING) HRESULT
//	[10] get_AlbumTitle(out *HSTRING) HRESULT
//	[11] get_TrackNumber(out *i32) HRESULT
//	[12] get_Genres(out *IVectorView<HSTRING>*) HRESULT
//	[13] get_AlbumTrackCount(out *i32) HRESULT
//	[14] get_PlaybackType(out *IReference<MediaPlaybackType>*) HRESULT (nullable)
//	[15] get_Thumbnail(out *IRandomAccessStreamReference*) HRESULT

const (
	Slot_MediaProps_Title           = 6
	Slot_MediaProps_Subtitle        = 7
	Slot_MediaProps_AlbumArtist     = 8
	Slot_MediaProps_Artist          = 9
	Slot_MediaProps_AlbumTitle      = 10
	Slot_MediaProps_TrackNumber     = 11
	Slot_MediaProps_Genres          = 12
	Slot_MediaProps_AlbumTrackCount = 13
	Slot_MediaProps_PlaybackType    = 14
	Slot_MediaProps_Thumbnail       = 15
)
