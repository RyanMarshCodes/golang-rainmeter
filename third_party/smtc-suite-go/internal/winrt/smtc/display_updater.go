//go:build windows && cgo

package smtc

// ISystemMediaTransportControlsDisplayUpdater vtable layout.
//
// Vtable slots after IInspectable (0-5):
//
//	[6]  get_Type(out *MediaPlaybackType) HRESULT
//	[7]  put_Type(in MediaPlaybackType) HRESULT
//	[8]  get_AppMediaId(out *HSTRING) HRESULT
//	[9]  put_AppMediaId(in HSTRING) HRESULT
//	[10] get_Thumbnail(out *IRandomAccessStreamReference*) HRESULT
//	[11] put_Thumbnail(in IRandomAccessStreamReference*) HRESULT
//	[12] get_MusicProperties(out *IMusicDisplayProperties*) HRESULT
//	[13] get_VideoProperties(out *IVideoDisplayProperties*) HRESULT
//	[14] get_ImageProperties(out *IImageDisplayProperties*) HRESULT
//	[15] CopyFromFileAsync(...) HRESULT
//	[16] ClearAll() HRESULT
//	[17] Update() HRESULT

const (
	Slot_DisplayUpdater_put_AppMediaId = 9
	Slot_DisplayUpdater_get_Thumbnail  = 10
	Slot_DisplayUpdater_put_Thumbnail  = 11
	Slot_DisplayUpdater_put_Type       = 7
	Slot_DisplayUpdater_get_MusicProps = 12
	Slot_DisplayUpdater_get_VideoProps = 13
	Slot_DisplayUpdater_get_ImageProps = 14
	Slot_DisplayUpdater_Update         = 17
	Slot_DisplayUpdater_ClearAll       = 16
)

// IMusicDisplayProperties vtable slots.
//
//	[6]  get_Title(out *HSTRING) HRESULT
//	[7]  put_Title(in HSTRING) HRESULT
//	[8]  get_AlbumArtist(out *HSTRING) HRESULT
//	[9]  put_AlbumArtist(in HSTRING) HRESULT
//	[10] get_Artist(out *HSTRING) HRESULT
//	[11] put_Artist(in HSTRING) HRESULT
//	[12] get_AlbumTitle(out *HSTRING) HRESULT
//	[13] put_AlbumTitle(in HSTRING) HRESULT
//	[14] get_TrackNumber(out *uint32) HRESULT
//	[15] put_TrackNumber(in uint32) HRESULT
//	[16] get_Genres(out *IVector<HSTRING>*) HRESULT
//	[17] get_AlbumTrackCount(out *uint32) HRESULT
//	[18] put_AlbumTrackCount(in uint32) HRESULT

const (
	Slot_MusicProps_put_Title  = 7
	Slot_MusicProps_put_Artist = 11
)

// IVideoDisplayProperties vtable slots (subset).
const (
	Slot_VideoProps_put_Title    = 6
	Slot_VideoProps_put_Subtitle = 8
)
