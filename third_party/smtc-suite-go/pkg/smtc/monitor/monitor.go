//go:build windows && cgo

// Package monitor watches system-wide Windows SMTC media sessions.
package monitor

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/xiaowumin-mark/smtc-suite-go/internal/winrt"
	smtcvt "github.com/xiaowumin-mark/smtc-suite-go/internal/winrt/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
)

// Monitor watches system-wide SMTC sessions.
type Monitor struct {
	mu      sync.Mutex
	closed  bool
	manager unsafe.Pointer

	managerEvents        chan ManagerEvent
	sessions             map[string]*sessionInfo
	sessionsChangedEvent *winrt.EventHandler
	currentChangedEvent  *winrt.EventHandler
	workCh               chan monitorWork
	doneCh               chan struct{}
	wg                   sync.WaitGroup
}

// Config configures the Monitor.
type Config struct {
	ManagerEventBuffer int
	WorkBuffer         int
}

// New creates and starts a Monitor.
func New(cfg *Config) (*Monitor, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	buf := cfg.ManagerEventBuffer
	if buf <= 0 {
		buf = 16
	}
	workBuf := cfg.WorkBuffer
	if workBuf <= 0 {
		workBuf = 64
	}

	if err := winrt.InitMTA(); err != nil {
		return nil, fmt.Errorf("monitor: %w", err)
	}

	factory, err := winrt.GetActivationFactory(
		smtcvt.RuntimeClass_GlobalSystemMediaTransportControlsSessionManager,
		winrt.IID_IGSMTCSessionManagerStatics,
	)
	if err != nil {
		winrt.UninitMTA()
		return nil, fmt.Errorf("monitor: get activation factory: %w", err)
	}
	defer winrt.Release(factory)

	asyncPtr, err := winrt.VtableGetPtr(factory, smtcvt.Slot_ManagerStatics_RequestAsync)
	if err != nil {
		winrt.UninitMTA()
		return nil, fmt.Errorf("monitor: RequestAsync: %w", err)
	}

	asyncOp := winrt.NewAsyncOperation(asyncPtr)
	managerPtr, err := asyncOp.Wait()
	if err != nil {
		asyncOp.Release()
		winrt.UninitMTA()
		return nil, fmt.Errorf("monitor: wait for session manager: %w", err)
	}
	asyncOp.Release()

	m := &Monitor{
		manager:       managerPtr,
		managerEvents: make(chan ManagerEvent, buf),
		sessions:      make(map[string]*sessionInfo),
		workCh:        make(chan monitorWork, workBuf),
		doneCh:        make(chan struct{}),
	}

	if _, err := m.refreshSessions(true); err != nil {
		m.Close()
		return nil, err
	}
	if err := m.registerManagerEvents(); err != nil {
		m.Close()
		return nil, err
	}

	m.wg.Add(1)
	go m.eventLoop()

	return m, nil
}

// Sessions returns a snapshot of all current sessions.
func (m *Monitor) Sessions() []smtc.SessionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotSessionsLocked()
}

// CurrentSession returns the current active session, or nil if none.
func (m *Monitor) CurrentSession() *smtc.SessionInfo {
	m.mu.Lock()
	if m.closed || m.manager == nil {
		m.mu.Unlock()
		return nil
	}
	manager := m.manager
	winrt.AddRef(manager)
	m.mu.Unlock()
	defer winrt.Release(manager)

	currentPtr, err := winrt.VtableGetPtr(manager, smtcvt.Slot_Manager_GetCurrentSession)
	if err != nil || currentPtr == nil {
		return nil
	}
	defer winrt.Release(currentPtr)

	id, _ := getSourceAppUserModelID(currentPtr)
	if id == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		info := cloneSessionInfo(s.info)
		return &info
	}
	return nil
}

// Events returns a read-only channel of manager and session events.
func (m *Monitor) Events() <-chan ManagerEvent {
	return m.managerEvents
}

// Close releases all resources.
func (m *Monitor) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	close(m.doneCh)

	manager := m.manager
	m.manager = nil
	sessionsChangedEvent := m.sessionsChangedEvent
	m.sessionsChangedEvent = nil
	currentChangedEvent := m.currentChangedEvent
	m.currentChangedEvent = nil
	sessions := m.sessions
	m.sessions = make(map[string]*sessionInfo)
	m.mu.Unlock()

	if sessionsChangedEvent != nil {
		_ = sessionsChangedEvent.Close()
	}
	if currentChangedEvent != nil {
		_ = currentChangedEvent.Close()
	}
	for _, s := range sessions {
		s.closeEvents()
	}
	m.wg.Wait()
	for _, s := range sessions {
		if s.ptr != nil {
			winrt.Release(s.ptr)
		}
	}
	if manager != nil {
		winrt.Release(manager)
	}
	close(m.managerEvents)
	winrt.UninitMTA()
	return nil
}

func (m *Monitor) registerManagerEvents() error {
	m.currentChangedEvent = winrt.NewTypedEventHandler([]*winrt.GUID{winrt.IID_ITypedEventHandler_GSMTCSessionManager_CurrentSessionChangedEventArgs}, func(sender, args unsafe.Pointer) {
		m.enqueue(monitorWork{typ: workCurrentSessionChanged})
	})
	if err := m.currentChangedEvent.Register(m.manager, smtcvt.Slot_Manager_add_CurrentSessionChanged, smtcvt.Slot_Manager_remove_CurrentSessionChanged); err != nil {
		m.currentChangedEvent.Close()
		m.currentChangedEvent = nil
		return fmt.Errorf("monitor: register CurrentSessionChanged: %w", err)
	}

	m.sessionsChangedEvent = winrt.NewTypedEventHandler([]*winrt.GUID{winrt.IID_ITypedEventHandler_GSMTCSessionManager_SessionsChangedEventArgs}, func(sender, args unsafe.Pointer) {
		m.enqueue(monitorWork{typ: workRefreshSessions})
	})
	if err := m.sessionsChangedEvent.Register(m.manager, smtcvt.Slot_Manager_add_SessionsChanged, smtcvt.Slot_Manager_remove_SessionsChanged); err != nil {
		m.sessionsChangedEvent.Close()
		m.sessionsChangedEvent = nil
		return fmt.Errorf("monitor: register SessionsChanged: %w", err)
	}
	return nil
}

func (m *Monitor) enqueue(work monitorWork) {
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return
	}
	select {
	case m.workCh <- work:
	default:
	}
}

func (m *Monitor) eventLoop() {
	defer m.wg.Done()
	for {
		select {
		case <-m.doneCh:
			return
		case work := <-m.workCh:
			m.handleWork(work)
		}
	}
}

func (m *Monitor) handleWork(work monitorWork) {
	switch work.typ {
	case workRefreshSessions:
		if sessions, err := m.refreshSessions(true); err == nil {
			m.emit(ManagerEvent{Type: ManagerEventSessionsChanged, Sessions: sessions})
		}
	case workCurrentSessionChanged:
		m.emitCurrentSessionChanged()
	case workSessionPlaybackChanged:
		m.handleSessionUpdate(work.sessionID, ManagerEventSessionPlaybackChanged, fetchPlaybackUpdate)
	case workSessionTimelineChanged:
		m.handleSessionUpdate(work.sessionID, ManagerEventSessionTimelineChanged, fetchTimelineUpdate)
	case workSessionMediaChanged:
		m.handleSessionUpdate(work.sessionID, ManagerEventSessionMediaChanged, fetchMediaUpdate)
	}
}

func (m *Monitor) emitCurrentSessionChanged() {
	m.mu.Lock()
	if m.closed || m.manager == nil {
		m.mu.Unlock()
		return
	}
	manager := m.manager
	winrt.AddRef(manager)
	m.mu.Unlock()
	defer winrt.Release(manager)

	currentPtr, err := winrt.VtableGetPtr(manager, smtcvt.Slot_Manager_GetCurrentSession)
	if err != nil || currentPtr == nil {
		m.emit(ManagerEvent{Type: ManagerEventCurrentSessionChanged})
		return
	}
	defer winrt.Release(currentPtr)

	id, _ := getSourceAppUserModelID(currentPtr)
	m.emit(ManagerEvent{Type: ManagerEventCurrentSessionChanged, CurrentSessionID: id})
}

// refreshSessions enumerates all sessions and updates event subscriptions.
func (m *Monitor) refreshSessions(registerEvents bool) ([]smtc.SessionInfo, error) {
	m.mu.Lock()
	if m.closed || m.manager == nil {
		m.mu.Unlock()
		return nil, nil
	}
	manager := m.manager
	winrt.AddRef(manager)
	m.mu.Unlock()
	defer winrt.Release(manager)

	sessionsPtr, err := winrt.VtableGetPtr(manager, smtcvt.Slot_Manager_GetSessions)
	if err != nil || sessionsPtr == nil {
		return nil, err
	}
	defer winrt.Release(sessionsPtr)

	count32, err := winrt.VtableGetU32(sessionsPtr, 7)
	if err != nil {
		return nil, err
	}
	count := min(int32(count32), 50)
	seen := make(map[string]bool, count)
	type enumeratedSession struct {
		id   string
		ptr  unsafe.Pointer
		info smtc.SessionInfo
	}
	enumerated := make([]enumeratedSession, 0, count)

	for i := int32(0); i < count; i++ {
		sessionPtr, err := winrt.VtableGetPtrWithArg(sessionsPtr, 6, uintptr(i))
		if err != nil || sessionPtr == nil {
			continue
		}

		id, _ := getSourceAppUserModelID(sessionPtr)
		if id == "" {
			winrt.Release(sessionPtr)
			continue
		}

		seen[id] = true
		enumerated = append(enumerated, enumeratedSession{
			id:   id,
			ptr:  sessionPtr,
			info: fetchSessionInfo(sessionPtr, id),
		})
	}

	var removed []*sessionInfo
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		for _, e := range enumerated {
			winrt.Release(e.ptr)
		}
		return nil, nil
	}

	for _, e := range enumerated {
		if existing, ok := m.sessions[e.id]; ok {
			existing.info = e.info
			winrt.Release(e.ptr)
			continue
		}
		si := &sessionInfo{ptr: e.ptr, info: e.info}
		m.sessions[e.id] = si
		if registerEvents {
			m.mu.Unlock()
			if err := m.registerSessionEvents(e.id, si); err != nil {
				si.closeEvents()
			}
			m.mu.Lock()
		}
	}
	for id, s := range m.sessions {
		if !seen[id] {
			removed = append(removed, s)
			delete(m.sessions, id)
		}
	}
	snapshot := m.snapshotSessionsLocked()
	m.mu.Unlock()

	for _, s := range removed {
		s.closeEvents()
		if s.ptr != nil {
			winrt.Release(s.ptr)
		}
	}
	return snapshot, nil
}

func (m *Monitor) registerSessionEvents(id string, s *sessionInfo) error {
	s.timelineEvent = winrt.NewTypedEventHandler([]*winrt.GUID{winrt.IID_ITypedEventHandler_GSMTCSession_TimelinePropertiesChangedEventArgs}, func(sender, args unsafe.Pointer) {
		m.enqueue(monitorWork{typ: workSessionTimelineChanged, sessionID: id})
	})
	if err := s.timelineEvent.Register(s.ptr, smtcvt.Slot_Session_add_TimelinePropertiesChanged, smtcvt.Slot_Session_remove_TimelinePropertiesChanged); err != nil {
		return fmt.Errorf("monitor: register TimelinePropertiesChanged: %w", err)
	}

	s.playbackEvent = winrt.NewTypedEventHandler([]*winrt.GUID{winrt.IID_ITypedEventHandler_GSMTCSession_PlaybackInfoChangedEventArgs}, func(sender, args unsafe.Pointer) {
		m.enqueue(monitorWork{typ: workSessionPlaybackChanged, sessionID: id})
	})
	if err := s.playbackEvent.Register(s.ptr, smtcvt.Slot_Session_add_PlaybackInfoChanged, smtcvt.Slot_Session_remove_PlaybackInfoChanged); err != nil {
		return fmt.Errorf("monitor: register PlaybackInfoChanged: %w", err)
	}

	s.mediaEvent = winrt.NewTypedEventHandler([]*winrt.GUID{winrt.IID_ITypedEventHandler_GSMTCSession_MediaPropertiesChangedEventArgs}, func(sender, args unsafe.Pointer) {
		m.enqueue(monitorWork{typ: workSessionMediaChanged, sessionID: id})
	})
	if err := s.mediaEvent.Register(s.ptr, smtcvt.Slot_Session_add_MediaPropertiesChanged, smtcvt.Slot_Session_remove_MediaPropertiesChanged); err != nil {
		return fmt.Errorf("monitor: register MediaPropertiesChanged: %w", err)
	}
	return nil
}

func (m *Monitor) handleSessionUpdate(id string, typ ManagerEventType, update func(unsafe.Pointer, *smtc.SessionInfo)) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	s, ok := m.sessions[id]
	if !ok || s.ptr == nil {
		m.mu.Unlock()
		return
	}
	ptr := s.ptr
	winrt.AddRef(ptr)
	m.mu.Unlock()

	defer winrt.Release(ptr)
	var eventInfo smtc.SessionInfo
	m.mu.Lock()
	if current, ok := m.sessions[id]; ok {
		eventInfo = cloneSessionInfo(current.info)
	}
	m.mu.Unlock()
	update(ptr, &eventInfo)

	m.mu.Lock()
	if current, ok := m.sessions[id]; ok && !m.closed {
		current.info = cloneSessionInfo(eventInfo)
		snapshot := cloneSessionInfo(current.info)
		m.mu.Unlock()
		m.emit(ManagerEvent{Type: typ, SessionID: id, Session: &snapshot})
		return
	}
	m.mu.Unlock()
}

func (m *Monitor) emit(evt ManagerEvent) {
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return
	}
	select {
	case m.managerEvents <- evt:
	default:
	}
}

func (m *Monitor) snapshotSessionsLocked() []smtc.SessionInfo {
	result := make([]smtc.SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, cloneSessionInfo(s.info))
	}
	return result
}

func getSourceAppUserModelID(sessionPtr unsafe.Pointer) (string, error) {
	hstr, err := winrt.VtableGetHSTRING(sessionPtr, smtcvt.Slot_Session_GetSourceAppUserModelId)
	if err != nil {
		return "", err
	}
	defer hstr.Delete()
	return hstr.String(), nil
}

func fetchSessionInfo(sessionPtr unsafe.Pointer, id string) smtc.SessionInfo {
	info := smtc.SessionInfo{
		SessionID:            id,
		SourceAppUserModelID: id,
	}
	fetchPlaybackUpdate(sessionPtr, &info)
	fetchTimelineUpdate(sessionPtr, &info)
	fetchMediaUpdate(sessionPtr, &info)
	return info
}

func fetchPlaybackUpdate(sessionPtr unsafe.Pointer, info *smtc.SessionInfo) {
	playbackPtr, err := winrt.VtableGetPtr(sessionPtr, smtcvt.Slot_Session_GetPlaybackInfo)
	if err != nil || playbackPtr == nil {
		return
	}
	defer winrt.Release(playbackPtr)
	if status, err := winrt.VtableGetI32(playbackPtr, smtcvt.Slot_PlaybackInfo_PlaybackStatus); err == nil {
		info.PlaybackStatus = smtc.PlaybackStatus(status)
	}
	if playbackType, ok := getOptionalI32(playbackPtr, smtcvt.Slot_PlaybackInfo_PlaybackType); ok {
		info.PlaybackType = smtc.PlaybackType(playbackType)
	}
	if repeat, ok := getOptionalI32(playbackPtr, smtcvt.Slot_PlaybackInfo_AutoRepeatMode); ok {
		info.AutoRepeatMode = smtc.AutoRepeatMode(repeat)
	}
	if rate, ok := getOptionalF64(playbackPtr, smtcvt.Slot_PlaybackInfo_PlaybackRate); ok {
		info.PlaybackRate = rate
		info.TimelineInfo.PlaybackRate = rate
	}
	if shuffle, ok := getOptionalBool(playbackPtr, smtcvt.Slot_PlaybackInfo_IsShuffleActive); ok {
		info.IsShuffleActive = shuffle
	}
	if controlsPtr, err := winrt.VtableGetPtr(playbackPtr, smtcvt.Slot_PlaybackInfo_Controls); err == nil && controlsPtr != nil {
		info.PlaybackControls = fetchPlaybackControls(controlsPtr)
		winrt.Release(controlsPtr)
	}
}

func getOptionalI32(obj unsafe.Pointer, slot int) (int32, bool) {
	ref, err := winrt.VtableGetPtr(obj, slot)
	if err != nil || ref == nil {
		return 0, false
	}
	defer winrt.Release(ref)
	v, err := winrt.ReferenceGetI32(ref)
	return v, err == nil
}

func getOptionalBool(obj unsafe.Pointer, slot int) (bool, bool) {
	ref, err := winrt.VtableGetPtr(obj, slot)
	if err != nil || ref == nil {
		return false, false
	}
	defer winrt.Release(ref)
	v, err := winrt.ReferenceGetBool(ref)
	return v, err == nil
}

func getOptionalF64(obj unsafe.Pointer, slot int) (float64, bool) {
	ref, err := winrt.VtableGetPtr(obj, slot)
	if err != nil || ref == nil {
		return 0, false
	}
	defer winrt.Release(ref)
	v, err := winrt.ReferenceGetF64(ref)
	return v, err == nil
}

func fetchPlaybackControls(controlsPtr unsafe.Pointer) smtc.PlaybackControls {
	return smtc.PlaybackControls{
		Play:             getControlBool(controlsPtr, smtcvt.Slot_Controls_IsPlayEnabled),
		Pause:            getControlBool(controlsPtr, smtcvt.Slot_Controls_IsPauseEnabled),
		Stop:             getControlBool(controlsPtr, smtcvt.Slot_Controls_IsStopEnabled),
		Record:           getControlBool(controlsPtr, smtcvt.Slot_Controls_IsRecordEnabled),
		FastForward:      getControlBool(controlsPtr, smtcvt.Slot_Controls_IsFastForwardEnabled),
		Rewind:           getControlBool(controlsPtr, smtcvt.Slot_Controls_IsRewindEnabled),
		Next:             getControlBool(controlsPtr, smtcvt.Slot_Controls_IsNextEnabled),
		Previous:         getControlBool(controlsPtr, smtcvt.Slot_Controls_IsPreviousEnabled),
		ChannelUp:        getControlBool(controlsPtr, smtcvt.Slot_Controls_IsChannelUpEnabled),
		ChannelDown:      getControlBool(controlsPtr, smtcvt.Slot_Controls_IsChannelDownEnabled),
		PlayPauseToggle:  getControlBool(controlsPtr, smtcvt.Slot_Controls_IsPlayPauseToggleEnabled),
		Shuffle:          getControlBool(controlsPtr, smtcvt.Slot_Controls_IsShuffleEnabled),
		Repeat:           getControlBool(controlsPtr, smtcvt.Slot_Controls_IsRepeatEnabled),
		PlaybackRate:     getControlBool(controlsPtr, smtcvt.Slot_Controls_IsPlaybackRateEnabled),
		PlaybackPosition: getControlBool(controlsPtr, smtcvt.Slot_Controls_IsPlaybackPositionEnabled),
	}
}

func getControlBool(controlsPtr unsafe.Pointer, slot int) bool {
	v, err := winrt.VtableGetBool(controlsPtr, slot)
	return err == nil && v
}

func fetchTimelineUpdate(sessionPtr unsafe.Pointer, info *smtc.SessionInfo) {
	timelinePtr, err := winrt.VtableGetPtr(sessionPtr, smtcvt.Slot_Session_GetTimelineProperties)
	if err != nil || timelinePtr == nil {
		return
	}
	defer winrt.Release(timelinePtr)
	info.TimelineInfo = fetchTimelineInfo(timelinePtr)
}

func fetchMediaUpdate(sessionPtr unsafe.Pointer, info *smtc.SessionInfo) {
	mediaAsyncPtr, err := winrt.VtableGetPtr(sessionPtr, smtcvt.Slot_Session_TryGetMediaPropertiesAsync)
	if err != nil || mediaAsyncPtr == nil {
		return
	}
	asyncOp := winrt.NewAsyncOperation(mediaAsyncPtr)
	if mediaPtr, err := asyncOp.Wait(); err == nil && mediaPtr != nil {
		defer winrt.Release(mediaPtr)
		info.MediaInfo = fetchMediaInfo(mediaPtr)
		if info.MediaInfo.PlaybackType != smtc.PlaybackTypeUnknown {
			info.PlaybackType = info.MediaInfo.PlaybackType
		}
	}
	asyncOp.Release()
}

func fetchTimelineInfo(timelinePtr unsafe.Pointer) smtc.TimelineInfo {
	var tl smtc.TimelineInfo

	if ticks, err := winrt.VtableGetTicks(timelinePtr, smtcvt.Slot_Timeline_Position); err == nil {
		tl.Position = smtc.TicksToDuration(ticks)
	}
	if ticks, err := winrt.VtableGetTicks(timelinePtr, smtcvt.Slot_Timeline_StartTime); err == nil {
		tl.StartTime = smtc.TicksToDuration(ticks)
	}
	if ticks, err := winrt.VtableGetTicks(timelinePtr, smtcvt.Slot_Timeline_EndTime); err == nil {
		tl.EndTime = smtc.TicksToDuration(ticks)
	}
	if ticks, err := winrt.VtableGetTicks(timelinePtr, smtcvt.Slot_Timeline_MinSeekTime); err == nil {
		tl.MinSeekTime = smtc.TicksToDuration(ticks)
	}
	if ticks, err := winrt.VtableGetTicks(timelinePtr, smtcvt.Slot_Timeline_MaxSeekTime); err == nil {
		tl.MaxSeekTime = smtc.TicksToDuration(ticks)
	}

	return tl
}

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
	if playbackType, ok := getOptionalI32(mediaPtr, smtcvt.Slot_MediaProps_PlaybackType); ok {
		mi.PlaybackType = smtc.PlaybackType(playbackType)
	}
	// Skip thumbnail stream reads. Downloading album art on every
	// MediaPropertiesChanged (e.g. YouTube Music track skip) uses fragile
	// WinRT async COM and has crashed host processes.
	if thumbnailPtr, err := winrt.VtableGetPtr(mediaPtr, smtcvt.Slot_MediaProps_Thumbnail); err == nil && thumbnailPtr != nil {
		mi.ThumbnailAvailable = true
		winrt.Release(thumbnailPtr)
	}

	return mi
}

func cloneSessionInfo(in smtc.SessionInfo) smtc.SessionInfo {
	out := in
	if in.MediaInfo.Genres != nil {
		out.MediaInfo.Genres = append([]string(nil), in.MediaInfo.Genres...)
	}
	if in.MediaInfo.ThumbnailData != nil {
		out.MediaInfo.ThumbnailData = append([]byte(nil), in.MediaInfo.ThumbnailData...)
	}
	return out
}

// ---- Event types ----

// ManagerEventType classifies monitor events.
type ManagerEventType int

const (
	ManagerEventSessionsChanged ManagerEventType = iota
	ManagerEventCurrentSessionChanged
	ManagerEventSessionPlaybackChanged
	ManagerEventSessionTimelineChanged
	ManagerEventSessionMediaChanged
)

// ManagerEvent is a union type for manager and session events.
type ManagerEvent struct {
	Type             ManagerEventType
	Sessions         []smtc.SessionInfo
	CurrentSessionID string
	SessionID        string
	Session          *smtc.SessionInfo
}

type sessionInfo struct {
	ptr           unsafe.Pointer
	info          smtc.SessionInfo
	timelineEvent *winrt.EventHandler
	playbackEvent *winrt.EventHandler
	mediaEvent    *winrt.EventHandler
}

func (s *sessionInfo) closeEvents() {
	if s.timelineEvent != nil {
		_ = s.timelineEvent.Close()
		s.timelineEvent = nil
	}
	if s.playbackEvent != nil {
		_ = s.playbackEvent.Close()
		s.playbackEvent = nil
	}
	if s.mediaEvent != nil {
		_ = s.mediaEvent.Close()
		s.mediaEvent = nil
	}
}

type monitorWorkType int

const (
	workRefreshSessions monitorWorkType = iota
	workCurrentSessionChanged
	workSessionPlaybackChanged
	workSessionTimelineChanged
	workSessionMediaChanged
)

type monitorWork struct {
	typ       monitorWorkType
	sessionID string
}
