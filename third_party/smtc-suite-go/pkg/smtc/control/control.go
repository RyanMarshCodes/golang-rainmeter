//go:build windows && cgo

package control

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/xiaowumin-mark/smtc-suite-go/internal/winrt"
	smtcvt "github.com/xiaowumin-mark/smtc-suite-go/internal/winrt/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
)

// Controller controls a remote SMTC media session.
type Controller struct {
	mu      sync.Mutex
	closed  bool
	session unsafe.Pointer
}

// New creates a Controller for a specific session, or the current session.
func New(sessionID string) (*Controller, error) {
	if err := winrt.InitMTA(); err != nil {
		return nil, fmt.Errorf("control: %w", err)
	}

	factory, err := winrt.GetActivationFactory(
		smtcvt.RuntimeClass_GlobalSystemMediaTransportControlsSessionManager,
		winrt.IID_IGSMTCSessionManagerStatics,
	)
	if err != nil {
		winrt.UninitMTA()
		return nil, fmt.Errorf("control: get factory: %w", err)
	}
	defer winrt.Release(factory)

	asyncPtr, err := winrt.VtableGetPtr(factory, smtcvt.Slot_ManagerStatics_RequestAsync)
	if err != nil {
		winrt.UninitMTA()
		return nil, fmt.Errorf("control: request async: %w", err)
	}

	asyncOp := winrt.NewAsyncOperation(asyncPtr)
	managerPtr, err := asyncOp.Wait()
	asyncOp.Release()
	if err != nil {
		winrt.UninitMTA()
		return nil, fmt.Errorf("control: wait for manager: %w", err)
	}
	defer winrt.Release(managerPtr)

	var sessionPtr unsafe.Pointer
	if sessionID == "" {
		sessionPtr, err = winrt.VtableGetPtr(managerPtr, smtcvt.Slot_Manager_GetCurrentSession)
	} else {
		sessionPtr, err = findSession(managerPtr, sessionID)
	}
	if err != nil || sessionPtr == nil {
		winrt.UninitMTA()
		return nil, fmt.Errorf("control: session not found")
	}

	return &Controller{session: sessionPtr}, nil
}

func findSession(managerPtr unsafe.Pointer, sessionID string) (unsafe.Pointer, error) {
	sessionsPtr, err := winrt.VtableGetPtr(managerPtr, smtcvt.Slot_Manager_GetSessions)
	if err != nil {
		return nil, err
	}
	defer winrt.Release(sessionsPtr)

	count, _ := winrt.VtableGetU32(sessionsPtr, 7)
	for i := range min(int32(count), 50) {
		sp, err := winrt.VtableGetPtrWithArg(sessionsPtr, 6, uintptr(i))
		if err != nil || sp == nil {
			continue
		}
		hs, err := winrt.VtableGetHSTRING(sp, smtcvt.Slot_Session_GetSourceAppUserModelId)
		if err != nil {
			winrt.Release(sp)
			continue
		}
		id := hs.String()
		hs.Delete()
		if id == sessionID {
			return sp, nil
		}
		winrt.Release(sp)
	}
	return nil, fmt.Errorf("session %q not found", sessionID)
}

// Play sends a play command.
func (c *Controller) Play() error { return c.asyncBool(smtcvt.Slot_Session_TryPlayAsync) }

// Pause sends a pause command.
func (c *Controller) Pause() error { return c.asyncBool(smtcvt.Slot_Session_TryPauseAsync) }

// TogglePlayPause toggles between play and pause.
func (c *Controller) TogglePlayPause() error {
	return c.asyncBool(smtcvt.Slot_Session_TryTogglePlayPauseAsync)
}

// Stop stops playback.
func (c *Controller) Stop() error { return c.asyncBool(smtcvt.Slot_Session_TryStopAsync) }

// Next skips to the next track.
func (c *Controller) Next() error { return c.asyncBool(smtcvt.Slot_Session_TrySkipNextAsync) }

// Previous goes to the previous track.
func (c *Controller) Previous() error { return c.asyncBool(smtcvt.Slot_Session_TrySkipPreviousAsync) }

// FastForward starts fast-forwarding.
func (c *Controller) FastForward() error { return c.asyncBool(smtcvt.Slot_Session_TryFastForwardAsync) }

// Rewind starts rewinding.
func (c *Controller) Rewind() error { return c.asyncBool(smtcvt.Slot_Session_TryRewindAsync) }

// Seek moves the playback position.
func (c *Controller) Seek(position time.Duration) error {
	ticks := smtc.DurationToTicks(position)
	return c.asyncBoolWithArg(smtcvt.Slot_Session_TryChangePlaybackPositionAsync, uintptr(ticks))
}

// SetPlaybackRate changes the playback rate.
func (c *Controller) SetPlaybackRate(rate float64) error {
	return c.asyncBoolWithFloat(smtcvt.Slot_Session_TryChangePlaybackRateAsync, rate)
}

// SetShuffle enables or disables shuffle mode.
func (c *Controller) SetShuffle(active bool) error {
	v := uintptr(0)
	if active {
		v = 1
	}
	return c.asyncBoolWithArg(smtcvt.Slot_Session_TryChangeShuffleActiveAsync, v)
}

// SetRepeatMode sets the repeat mode.
func (c *Controller) SetRepeatMode(mode smtc.AutoRepeatMode) error {
	return c.asyncBoolWithArg(smtcvt.Slot_Session_TryChangeAutoRepeatModeAsync, uintptr(mode))
}

// MediaInfo returns the current media metadata for the controlled session.
func (c *Controller) MediaInfo() (smtc.MediaInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return smtc.MediaInfo{}, fmt.Errorf("control: closed")
	}

	mediaAsyncPtr, err := winrt.VtableGetPtr(c.session, smtcvt.Slot_Session_TryGetMediaPropertiesAsync)
	if err != nil {
		return smtc.MediaInfo{}, fmt.Errorf("control: TryGetMediaPropertiesAsync: %w", err)
	}
	if mediaAsyncPtr == nil {
		return smtc.MediaInfo{}, fmt.Errorf("control: media properties operation is nil")
	}

	asyncOp := winrt.NewAsyncOperation(mediaAsyncPtr)
	defer asyncOp.Release()
	mediaPtr, err := asyncOp.WaitTimeout(5 * time.Second)
	if err != nil {
		return smtc.MediaInfo{}, fmt.Errorf("control: wait for media properties: %w", err)
	}
	if mediaPtr == nil {
		return smtc.MediaInfo{}, fmt.Errorf("control: media properties are nil")
	}
	defer winrt.Release(mediaPtr)

	return fetchMediaInfo(mediaPtr), nil
}

// Close releases the controller.
func (c *Controller) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.session != nil {
		winrt.Release(c.session)
		c.session = nil
	}
	winrt.UninitMTA()
	return nil
}

// ---- internal helpers ----

func fetchMediaInfo(mediaPtr unsafe.Pointer) smtc.MediaInfo {
	var mi smtc.MediaInfo

	if hstr, err := winrt.VtableGetHSTRING(mediaPtr, smtcvt.Slot_MediaProps_Title); err == nil {
		mi.Title = hstr.String()
		hstr.Delete()
	}
	if hstr, err := winrt.VtableGetHSTRING(mediaPtr, smtcvt.Slot_MediaProps_Artist); err == nil {
		mi.Artist = hstr.String()
		hstr.Delete()
	}
	if hstr, err := winrt.VtableGetHSTRING(mediaPtr, smtcvt.Slot_MediaProps_AlbumTitle); err == nil {
		mi.AlbumTitle = hstr.String()
		hstr.Delete()
	}
	if hstr, err := winrt.VtableGetHSTRING(mediaPtr, smtcvt.Slot_MediaProps_AlbumArtist); err == nil {
		mi.AlbumArtist = hstr.String()
		hstr.Delete()
	}
	if n, err := winrt.VtableGetI32(mediaPtr, smtcvt.Slot_MediaProps_TrackNumber); err == nil {
		mi.TrackNumber = n
	}
	if n, err := winrt.VtableGetI32(mediaPtr, smtcvt.Slot_MediaProps_AlbumTrackCount); err == nil {
		mi.AlbumTrackCount = n
	}
	if thumbnailPtr, err := winrt.VtableGetPtr(mediaPtr, smtcvt.Slot_MediaProps_Thumbnail); err == nil && thumbnailPtr != nil {
		mi.ThumbnailAvailable = true
		if data, err := winrt.ReadRandomAccessStreamReference(thumbnailPtr); err == nil && len(data) > 0 {
			mi.ThumbnailData = data
			hash := sha256.Sum256(data)
			mi.ThumbnailHash = fmt.Sprintf("%x", hash[:])
		}
		winrt.Release(thumbnailPtr)
	}

	return mi
}

func (c *Controller) asyncBool(slot int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("control: closed")
	}
	var asyncPtr unsafe.Pointer
	fn := winrt.VtableFn(c.session, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(c.session), uintptr(unsafe.Pointer(&asyncPtr)))
	if int32(r1) < 0 {
		return fmt.Errorf("control: HRESULT 0x%08X", uint32(r1))
	}
	asyncOp := winrt.NewAsyncOperationBool(asyncPtr)
	ok, err := asyncOp.WaitTimeout(5 * time.Second)
	asyncOp.Release()
	if err == nil && !ok {
		return fmt.Errorf("control: operation rejected")
	}
	return err
}

func (c *Controller) asyncBoolWithArg(slot int, arg uintptr) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("control: closed")
	}
	var asyncPtr unsafe.Pointer
	fn := winrt.VtableFn(c.session, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(c.session), arg, uintptr(unsafe.Pointer(&asyncPtr)))
	if int32(r1) < 0 {
		return fmt.Errorf("control: HRESULT 0x%08X", uint32(r1))
	}
	asyncOp := winrt.NewAsyncOperationBool(asyncPtr)
	ok, err := asyncOp.WaitTimeout(5 * time.Second)
	asyncOp.Release()
	if err == nil && !ok {
		return fmt.Errorf("control: operation rejected")
	}
	return err
}

func (c *Controller) asyncBoolWithFloat(slot int, val float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("control: closed")
	}
	asyncPtr, err := winrt.VtableAsyncBoolWithF64(c.session, slot, val)
	if err != nil {
		return fmt.Errorf("control: async float64 call: %w", err)
	}
	asyncOp := winrt.NewAsyncOperationBool(asyncPtr)
	ok, err := asyncOp.WaitTimeout(5 * time.Second)
	asyncOp.Release()
	if err == nil && !ok {
		return fmt.Errorf("control: operation rejected")
	}
	return err
}
