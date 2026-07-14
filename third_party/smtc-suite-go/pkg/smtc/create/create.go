//go:build windows && cgo

package create

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/xiaowumin-mark/smtc-suite-go/internal/winrt"
	smtcvt "github.com/xiaowumin-mark/smtc-suite-go/internal/winrt/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
)

// Creator owns a media session published to Windows System Media Transport
// Controls (SMTC).
//
// A Creator is backed by a WinRT Windows.Media.Playback.MediaPlayer instance.
// Keeping the Creator alive keeps the MediaPlayer and its
// SystemMediaTransportControls object alive; closing it releases the WinRT
// objects and removes the session from the Windows media UI.
//
// Methods on Creator are serialized with an internal mutex. The underlying
// WinRT objects are initialized in an MTA apartment, so this type is intended
// for normal Go use from application goroutines. Button presses are exposed
// through ButtonEvents.
type Creator struct {
	mu          sync.Mutex
	closed      bool
	mediaPlayer unsafe.Pointer
	smtc        unsafe.Pointer
	smtc2       unsafe.Pointer
	display     unsafe.Pointer
	mediaInfo   smtc.MediaInfo
	buttons     chan smtc.Button
	buttonEvent *winrt.EventHandler
}

// Config describes the initial state of a Creator.
//
// Zero values are accepted. New fills in conservative defaults so the session
// is visible during manual testing: a title, artist, app media id, Playing
// status, and common transport buttons enabled.
type Config struct {
	// MediaInfo is the initial metadata shown in the Windows media UI.
	//
	// Only the verified music metadata fields are currently written: Title,
	// Artist, and PlaybackType. If PlaybackType is zero, it is treated as
	// smtc.PlaybackTypeMusic.
	MediaInfo smtc.MediaInfo

	// AppMediaID is an application-defined media identifier exposed through
	// SystemMediaTransportControlsDisplayUpdater.AppMediaId. Windows does not
	// require it to be globally unique, but a stable value is useful when
	// debugging multiple sessions.
	AppMediaID string

	// PlaybackStatus is the initial state published to SMTC. If it is zero,
	// New uses smtc.PlaybackStatusPlaying instead of the monitor-side
	// "Closed" value so the example session appears active.
	PlaybackStatus smtc.PlaybackStatus

	// Buttons controls which transport buttons are enabled initially. If nil,
	// DefaultButtons is used.
	Buttons *Buttons

	// TimelineInfo is the initial timeline shown by Windows media controls. If
	// EndTime is zero, no initial timeline is published.
	TimelineInfo smtc.TimelineInfo

	// ThumbnailPath is an optional local image file path used as the initial
	// cover artwork. Windows Shell may ignore local file thumbnails for
	// MediaPlayer-backed SMTC sessions; prefer SetThumbnailFromURI for reliable
	// artwork display.
	ThumbnailPath string

	// ButtonEventBuffer controls the capacity of the ButtonEvents channel. If
	// it is zero or negative, New uses a small default buffer. Events are
	// dropped when the buffer is full so the WinRT callback thread is never
	// blocked by Go application code.
	ButtonEventBuffer int
}

// Buttons describes which Windows media transport buttons are enabled for a
// Creator.
//
// Enabled buttons may be displayed by the Windows media UI and may later
// produce ButtonPressed events on the ButtonEvents channel. Disabled buttons
// are hidden or shown as unavailable depending on the shell surface.
type Buttons struct {
	Play        bool
	Pause       bool
	Stop        bool
	Next        bool
	Previous    bool
	FastForward bool
	Rewind      bool
	Record      bool
	ChannelUp   bool
	ChannelDown bool
}

// DefaultConfig returns a Config suitable for a visible test session.
//
// Callers usually pass nil to New unless they need custom initial metadata.
func DefaultConfig() *Config { return &Config{} }

// DefaultButtons returns the common media-player buttons enabled.
//
// Fast-forward, rewind, record, and channel buttons default to disabled. The
// current implementation only writes Play, Pause, Stop, Next, and Previous;
// advanced buttons are left for the event/capability pass once their runtime
// behavior is verified on the MediaPlayer-backed SMTC object.
func DefaultButtons() Buttons {
	return Buttons{
		Play:     true,
		Pause:    true,
		Stop:     true,
		Next:     true,
		Previous: true,
	}
}

// New creates a Windows SMTC session and publishes its initial state.
//
// The returned Creator must be closed when the application no longer wants the
// session to appear in Windows media controls. New uses the modern
// MediaPlayer.SystemMediaTransportControls path; it intentionally does not use
// the deprecated ISystemMediaTransportControlsInterop.GetForWindow path.
func New(cfg *Config) (*Creator, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	normalizeConfig(cfg)

	if err := winrt.InitMTA(); err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	c := &Creator{
		buttons: make(chan smtc.Button, cfg.ButtonEventBuffer),
	}
	if err := c.initWithMediaPlayer(cfg); err != nil {
		_ = c.Close()
		return nil, err
	}

	return c, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.MediaInfo.Title == "" {
		cfg.MediaInfo.Title = "smtc-suite-go"
	}
	if cfg.MediaInfo.Artist == "" {
		cfg.MediaInfo.Artist = "Go CGo WinRT"
	}
	if cfg.MediaInfo.PlaybackType == smtc.PlaybackTypeUnknown {
		cfg.MediaInfo.PlaybackType = smtc.PlaybackTypeMusic
	}
	if cfg.AppMediaID == "" {
		cfg.AppMediaID = "github.com/xiaowumin-mark/smtc-suite-go"
	}
	if cfg.PlaybackStatus == smtc.PlaybackStatusClosed {
		cfg.PlaybackStatus = smtc.PlaybackStatusPlaying
	}
	if cfg.Buttons == nil {
		buttons := DefaultButtons()
		cfg.Buttons = &buttons
	}
	if cfg.ButtonEventBuffer <= 0 {
		cfg.ButtonEventBuffer = 16
	}
}

func (c *Creator) initWithMediaPlayer(cfg *Config) error {
	mediaPlayer, err := winrt.ActivateInstance("Windows.Media.Playback.MediaPlayer")
	if err != nil {
		return fmt.Errorf("create: activate MediaPlayer: %w", err)
	}
	c.mediaPlayer = mediaPlayer

	mediaPlayer2, err := winrt.QueryInterface(mediaPlayer, winrt.IID_IMediaPlayer2)
	if err != nil {
		return fmt.Errorf("create: query IMediaPlayer2: %w", err)
	}
	defer winrt.Release(mediaPlayer2)

	smtcPtr, err := winrt.VtableGetPtr(mediaPlayer2, smtcvt.Slot_MediaPlayer2_get_SystemMediaTransportControls)
	if err != nil {
		return fmt.Errorf("create: get SystemMediaTransportControls: %w", err)
	}
	c.smtc = smtcPtr

	smtc2Ptr, err := winrt.QueryInterface(c.smtc, winrt.IID_ISystemMediaTransportControls2)
	if err != nil {
		return fmt.Errorf("create: query ISystemMediaTransportControls2: %w", err)
	}
	c.smtc2 = smtc2Ptr

	if err := c.setEnabledLocked(true); err != nil {
		return err
	}
	if err := c.setEnabledButtonsLocked(*cfg.Buttons); err != nil {
		return err
	}
	if err := c.setPlaybackStatusLocked(cfg.PlaybackStatus); err != nil {
		return err
	}

	displayPtr, err := winrt.VtableGetPtr(c.smtc, smtcvt.Slot_Controls_get_DisplayUpdater)
	if err != nil {
		return fmt.Errorf("create: get DisplayUpdater: %w", err)
	}
	c.display = displayPtr

	if err := c.setMediaInfoLocked(cfg.MediaInfo, cfg.AppMediaID); err != nil {
		return err
	}
	if cfg.ThumbnailPath != "" {
		if err := c.setThumbnailFromFileLocked(cfg.ThumbnailPath); err != nil {
			return err
		}
	}
	if cfg.TimelineInfo.EndTime > 0 {
		if err := c.setTimelineInfoLocked(cfg.TimelineInfo); err != nil {
			return err
		}
	}

	if err := c.registerButtonEventsLocked(); err != nil {
		return err
	}

	return nil
}

// SetThumbnailFromFile updates the cover artwork shown in Windows media
// controls from a local image file.
//
// The file is resolved by Windows.Storage.StorageFile.GetFileFromPathAsync and
// wrapped with RandomAccessStreamReference.CreateFromFile. Windows Shell may
// ignore this for MediaPlayer-backed SMTC sessions even when the call succeeds;
// prefer SetThumbnailFromURI when possible.
func (c *Creator) SetThumbnailFromFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setThumbnailFromFileLocked(path)
}

// SetThumbnailFromURI updates the cover artwork from an absolute URI.
//
// This matches Windows.Media.Storage.Streams.RandomAccessStreamReference.CreateFromUri
// and supports https:// and file:/// URIs.
func (c *Creator) SetThumbnailFromURI(uri string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setThumbnailFromURILocked(uri)
}

func (c *Creator) setThumbnailFromFileLocked(path string) error {
	ref, err := winrt.RandomAccessStreamReferenceFromFile(path)
	if err != nil {
		return fmt.Errorf("create: create thumbnail stream reference from file: %w", err)
	}
	return c.setThumbnailRefLocked(ref)
}

func (c *Creator) setThumbnailFromURILocked(uri string) error {
	ref, err := winrt.RandomAccessStreamReferenceFromURI(uri)
	if err != nil {
		return fmt.Errorf("create: create thumbnail stream reference from URI: %w", err)
	}
	return c.setThumbnailRefLocked(ref)
}

func (c *Creator) setThumbnailRefLocked(ref unsafe.Pointer) error {
	defer winrt.Release(ref)

	if err := setDisplayThumbnail(c.display, ref); err != nil {
		return fmt.Errorf("create: put Thumbnail: %w", err)
	}
	if err := callNoArgs(c.display, smtcvt.Slot_DisplayUpdater_Update); err != nil {
		return fmt.Errorf("create: display Update: %w", err)
	}
	return nil
}

// SetTimelineInfo updates the timeline shown in Windows media controls.
//
// StartTime, EndTime, MinSeekTime, MaxSeekTime, and Position are published as
// WinRT TimeSpan values. If MinSeekTime or MaxSeekTime is zero, this method
// uses StartTime and EndTime respectively. PlaybackRate is published when it is
// non-zero.
func (c *Creator) SetTimelineInfo(info smtc.TimelineInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setTimelineInfoLocked(info)
}

func (c *Creator) setTimelineInfoLocked(info smtc.TimelineInfo) error {
	if c.smtc2 == nil {
		return fmt.Errorf("create: timeline interface not initialized")
	}
	if info.MinSeekTime == 0 {
		info.MinSeekTime = info.StartTime
	}
	if info.MaxSeekTime == 0 {
		info.MaxSeekTime = info.EndTime
	}

	timeline, err := winrt.ActivateInstance(smtcvt.RuntimeClass_SystemMediaTransportControlsTimelineProperties)
	if err != nil {
		return fmt.Errorf("create: activate TimelineProperties: %w", err)
	}
	defer winrt.Release(timeline)

	if err := winrt.VtablePutTicks(timeline, smtcvt.Slot_TimelineProps_put_StartTime, smtc.DurationToTicks(info.StartTime)); err != nil {
		return fmt.Errorf("create: put timeline StartTime: %w", err)
	}
	if err := winrt.VtablePutTicks(timeline, smtcvt.Slot_TimelineProps_put_EndTime, smtc.DurationToTicks(info.EndTime)); err != nil {
		return fmt.Errorf("create: put timeline EndTime: %w", err)
	}
	if err := winrt.VtablePutTicks(timeline, smtcvt.Slot_TimelineProps_put_MinSeekTime, smtc.DurationToTicks(info.MinSeekTime)); err != nil {
		return fmt.Errorf("create: put timeline MinSeekTime: %w", err)
	}
	if err := winrt.VtablePutTicks(timeline, smtcvt.Slot_TimelineProps_put_MaxSeekTime, smtc.DurationToTicks(info.MaxSeekTime)); err != nil {
		return fmt.Errorf("create: put timeline MaxSeekTime: %w", err)
	}
	if err := winrt.VtablePutTicks(timeline, smtcvt.Slot_TimelineProps_put_Position, smtc.DurationToTicks(info.Position)); err != nil {
		return fmt.Errorf("create: put timeline Position: %w", err)
	}
	if info.PlaybackRate != 0 {
		if err := winrt.VtablePutF64(c.smtc2, smtcvt.Slot_Controls2_put_PlaybackRate, info.PlaybackRate); err != nil {
			return fmt.Errorf("create: put PlaybackRate: %w", err)
		}
	}
	return callWithPtr(c.smtc2, smtcvt.Slot_Controls2_UpdateTimelineProperties, timeline)
}

// ButtonEvents returns a channel of system media button presses for this
// session.
//
// Windows raises these events when the user activates media controls that are
// enabled for this Creator, for example by pressing keyboard media keys or
// clicking the Windows media overlay. The channel is buffered. If the receiver
// falls behind and the buffer fills, new button events may be dropped to avoid
// blocking WinRT callback threads.
//
// The channel is not closed by Close. This avoids a send-on-closed-channel race
// with callbacks already in flight from Windows. Applications should stop
// reading it when their own context is canceled or after Close returns.
func (c *Creator) ButtonEvents() <-chan smtc.Button {
	return c.buttons
}

func (c *Creator) registerButtonEventsLocked() error {
	c.buttonEvent = winrt.NewEventHandler(func(sender, args unsafe.Pointer) {
		if args == nil {
			return
		}
		raw, err := winrt.VtableGetI32(args, smtcvt.Slot_ButtonPressedArgs_get_Button)
		if err != nil {
			return
		}
		select {
		case c.buttons <- smtc.Button(raw):
		default:
		}
	})
	if err := c.buttonEvent.Register(c.smtc, smtcvt.Slot_Controls_add_ButtonPressed, smtcvt.Slot_Controls_remove_ButtonPressed); err != nil {
		_ = c.buttonEvent.Close()
		c.buttonEvent = nil
		return fmt.Errorf("create: add ButtonPressed: %w", err)
	}
	return nil
}

// SetEnabled enables or disables this SMTC session as a whole.
//
// Disabling the session leaves the Creator alive but tells Windows that the
// application is not currently publishing active media controls. Callers can
// re-enable it later without creating a new Creator.
func (c *Creator) SetEnabled(enabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setEnabledLocked(enabled)
}

func (c *Creator) setEnabledLocked(enabled bool) error {
	if err := winrt.VtablePutBool(c.smtc, smtcvt.Slot_Controls_put_IsEnabled, enabled); err != nil {
		return fmt.Errorf("create: put IsEnabled: %w", err)
	}
	return nil
}

// SetEnabledButtons updates the transport buttons Windows may show for this
// session.
//
// This changes button availability in the system UI. Enabled buttons can
// produce events on ButtonEvents when the user activates them.
//
// At the moment, only Play, Pause, Stop, Next, and Previous are written. Passing
// true for FastForward, Rewind, Record, ChannelUp, or ChannelDown returns an
// error instead of touching unverified vtable slots.
func (c *Creator) SetEnabledButtons(buttons Buttons) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setEnabledButtonsLocked(buttons)
}

func (c *Creator) setEnabledButtonsLocked(buttons Buttons) error {
	if buttons.FastForward || buttons.Rewind || buttons.Record || buttons.ChannelUp || buttons.ChannelDown {
		return fmt.Errorf("create: advanced transport buttons are not implemented yet")
	}
	for _, call := range []struct {
		name  string
		slot  int
		value bool
	}{
		{"IsPlayEnabled", smtcvt.Slot_Controls_put_IsPlayEnabled, buttons.Play},
		{"IsPauseEnabled", smtcvt.Slot_Controls_put_IsPauseEnabled, buttons.Pause},
		{"IsStopEnabled", smtcvt.Slot_Controls_put_IsStopEnabled, buttons.Stop},
		{"IsNextEnabled", smtcvt.Slot_Controls_put_IsNextEnabled, buttons.Next},
		{"IsPreviousEnabled", smtcvt.Slot_Controls_put_IsPreviousEnabled, buttons.Previous},
	} {
		if err := winrt.VtablePutBool(c.smtc, call.slot, call.value); err != nil {
			return fmt.Errorf("create: put %s: %w", call.name, err)
		}
	}
	return nil
}

// SetPlaybackStatus publishes the current playback state to Windows.
//
// The public smtc.PlaybackStatus type is shared with the monitor package,
// whose Windows.Media.Control enum has six values. Create publishes through
// Windows.Media.MediaPlaybackStatus, which has five values. The mapping is:
// Closed -> Closed, Opened/Stopped -> Stopped, Changing -> Changing,
// Playing -> Playing, and Paused -> Paused.
func (c *Creator) SetPlaybackStatus(status smtc.PlaybackStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setPlaybackStatusLocked(status)
}

func (c *Creator) setPlaybackStatusLocked(status smtc.PlaybackStatus) error {
	if err := winrt.VtablePutI32(c.smtc, smtcvt.Slot_Controls_put_PlaybackStatus, int32(toMediaPlaybackStatus(status))); err != nil {
		return fmt.Errorf("create: put PlaybackStatus: %w", err)
	}
	return nil
}

// SetMediaInfo updates the metadata shown in Windows media controls.
//
// The current implementation writes music-style metadata through
// SystemMediaTransportControlsDisplayUpdater.MusicProperties. Title and Artist
// are the verified fields currently written. Cover artwork can be updated with
// SetThumbnailFromFile.
func (c *Creator) SetMediaInfo(info smtc.MediaInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkOpenLocked(); err != nil {
		return err
	}
	return c.setMediaInfoLocked(info, "")
}

func (c *Creator) setMediaInfoLocked(info smtc.MediaInfo, appMediaID string) error {
	c.mediaInfo = info
	playbackType := info.PlaybackType
	if playbackType == smtc.PlaybackTypeUnknown {
		playbackType = smtc.PlaybackTypeMusic
	}
	if err := winrt.VtablePutI32(c.display, smtcvt.Slot_DisplayUpdater_put_Type, int32(toMediaPlaybackType(playbackType))); err != nil {
		return fmt.Errorf("create: put display Type: %w", err)
	}
	if appMediaID != "" {
		if err := putDisplayHSTRING(c.display, smtcvt.Slot_DisplayUpdater_put_AppMediaId, appMediaID); err != nil {
			return fmt.Errorf("create: put AppMediaId: %w", err)
		}
	}

	musicProps, err := winrt.VtableGetPtr(c.display, smtcvt.Slot_DisplayUpdater_get_MusicProps)
	if err != nil {
		return fmt.Errorf("create: get MusicProperties: %w", err)
	}
	defer winrt.Release(musicProps)

	if err := putDisplayHSTRING(musicProps, smtcvt.Slot_MusicProps_put_Title, info.Title); err != nil {
		return fmt.Errorf("create: put Title: %w", err)
	}
	if err := putDisplayHSTRING(musicProps, smtcvt.Slot_MusicProps_put_Artist, info.Artist); err != nil {
		return fmt.Errorf("create: put Artist: %w", err)
	}

	if err := callNoArgs(c.display, smtcvt.Slot_DisplayUpdater_Update); err != nil {
		return fmt.Errorf("create: display Update: %w", err)
	}
	return nil
}

func toMediaPlaybackStatus(status smtc.PlaybackStatus) smtcvt.MediaPlaybackStatus {
	switch status {
	case smtc.PlaybackStatusClosed:
		return smtcvt.MediaPlaybackStatusClosed
	case smtc.PlaybackStatusChanging:
		return smtcvt.MediaPlaybackStatusChanging
	case smtc.PlaybackStatusPlaying:
		return smtcvt.MediaPlaybackStatusPlaying
	case smtc.PlaybackStatusPaused:
		return smtcvt.MediaPlaybackStatusPaused
	case smtc.PlaybackStatusOpened, smtc.PlaybackStatusStopped:
		return smtcvt.MediaPlaybackStatusStopped
	default:
		return smtcvt.MediaPlaybackStatusStopped
	}
}

func toMediaPlaybackType(playbackType smtc.PlaybackType) smtcvt.MediaPlaybackType {
	switch playbackType {
	case smtc.PlaybackTypeVideo:
		return smtcvt.MediaPlaybackTypeVideo
	case smtc.PlaybackTypeImage:
		return smtcvt.MediaPlaybackTypeImage
	case smtc.PlaybackTypeMusic:
		return smtcvt.MediaPlaybackTypeMusic
	default:
		return smtcvt.MediaPlaybackTypeUnknown
	}
}

func putDisplayHSTRING(obj unsafe.Pointer, slot int, value string) error {
	hstr, err := winrt.NewHSTRING(value)
	if err != nil {
		return err
	}
	defer hstr.Delete()
	return winrt.VtablePutHSTRING(obj, slot, hstr)
}

func callNoArgs(obj unsafe.Pointer, slot int) error {
	fn := winrt.VtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj))
	if int32(r1) < 0 {
		return fmt.Errorf("HRESULT 0x%08X", uint32(r1))
	}
	return nil
}

func callWithPtr(obj unsafe.Pointer, slot int, arg unsafe.Pointer) error {
	fn := winrt.VtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), uintptr(arg))
	if int32(r1) < 0 {
		return fmt.Errorf("HRESULT 0x%08X", uint32(r1))
	}
	return nil
}

func setDisplayThumbnail(display unsafe.Pointer, ref unsafe.Pointer) error {
	fn := winrt.VtableFn(display, smtcvt.Slot_DisplayUpdater_put_Thumbnail)
	r1, _, _ := syscall.SyscallN(fn, uintptr(display), uintptr(ref))
	if int32(r1) < 0 {
		return fmt.Errorf("HRESULT 0x%08X", uint32(r1))
	}
	return nil
}

func (c *Creator) checkOpenLocked() error {
	if c.closed {
		return fmt.Errorf("create: closed")
	}
	if c.smtc == nil || c.display == nil {
		return fmt.Errorf("create: not initialized")
	}
	return nil
}

// Close releases all WinRT objects owned by the Creator.
//
// Close is idempotent. After Close returns, the Creator must not be used again.
// The Windows media UI may remove the session asynchronously after the final
// COM references are released.
func (c *Creator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	var errs []error
	if c.buttonEvent != nil {
		if err := c.buttonEvent.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close ButtonPressed handler: %w", err))
		}
		c.buttonEvent = nil
	}
	if c.display != nil {
		winrt.Release(c.display)
		c.display = nil
	}
	if c.smtc2 != nil {
		winrt.Release(c.smtc2)
		c.smtc2 = nil
	}
	if c.smtc != nil {
		winrt.Release(c.smtc)
		c.smtc = nil
	}
	if c.mediaPlayer != nil {
		winrt.Release(c.mediaPlayer)
		c.mediaPlayer = nil
	}
	winrt.UninitMTA()
	if len(errs) > 0 {
		return fmt.Errorf("create: close: %w", errors.Join(errs...))
	}
	return nil
}
