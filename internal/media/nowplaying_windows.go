//go:build windows && cgo

package media

import (
	"log"
	"strings"
	"sync"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/monitor"
)

var (
	monOnce sync.Once
	mon     *monitor.Monitor
	monErr  error

	loggedSources sync.Map // SourceAppUserModelID → struct{}
)

func getMonitor() (*monitor.Monitor, error) {
	monOnce.Do(func() {
		mon, monErr = monitor.New(nil)
		if monErr != nil {
			log.Printf("media: SMTC monitor: %v", monErr)
			return
		}
		// Drain events so the monitor's buffered channel never backs up.
		go func() {
			for range mon.Events() {
			}
		}()
	})
	return mon, monErr
}

func noteSource(appID, title, artist string) {
	if appID == "" {
		return
	}
	if _, seen := loggedSources.LoadOrStore(appID, struct{}{}); seen {
		return
	}
	meta := strings.TrimSpace(title)
	if artist != "" {
		if meta != "" {
			meta += " — "
		}
		meta += artist
	}
	if meta == "" {
		meta = "(no title)"
	}
	log.Printf("media: SMTC source %q  e.g. %q", appID, meta)
}

func sessionHaystack(s *smtc.SessionInfo) string {
	if s == nil {
		return ""
	}
	parts := []string{
		s.SourceAppUserModelID,
		s.MediaInfo.Title,
		s.MediaInfo.Subtitle,
		s.MediaInfo.Artist,
		s.MediaInfo.AlbumArtist,
		s.MediaInfo.AlbumTitle,
	}
	parts = append(parts, s.MediaInfo.Genres...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

func appAllowed(s *smtc.SessionInfo, f Filter) bool {
	if s == nil {
		return false
	}
	hay := sessionHaystack(s)
	for _, d := range f.Deny {
		if d != "" && strings.Contains(hay, strings.ToLower(d)) {
			return false
		}
	}
	if len(f.Allow) == 0 {
		return true
	}
	for _, a := range f.Allow {
		if a != "" && strings.Contains(hay, strings.ToLower(a)) {
			return true
		}
	}
	return false
}

func trackFromSession(s *smtc.SessionInfo) Track {
	if s == nil {
		return Track{}
	}
	ok := s.PlaybackStatus == smtc.PlaybackStatusPlaying ||
		s.PlaybackStatus == smtc.PlaybackStatusPaused ||
		s.PlaybackStatus == smtc.PlaybackStatusOpened
	title := s.MediaInfo.Title
	artist := s.MediaInfo.Artist
	if artist == "" {
		artist = s.MediaInfo.AlbumArtist
	}
	if title == "" && artist == "" {
		return Track{}
	}
	tl := s.TimelineInfo
	dur := tl.EndTime - tl.StartTime
	if dur <= 0 && tl.MaxSeekTime > tl.MinSeekTime {
		dur = tl.MaxSeekTime - tl.MinSeekTime
	}
	pos := tl.Position - tl.StartTime
	if pos < 0 {
		pos = tl.Position
	}
	if pos > dur && dur > 0 {
		pos = dur
	}
	isPlaying := s.PlaybackStatus == smtc.PlaybackStatusPlaying
	rate := s.PlaybackRate
	if rate <= 0 {
		rate = tl.PlaybackRate
	}
	key := s.SourceAppUserModelID + "|" + title + "|" + artist
	pos = trackClock.position(key, pos, dur, rate, isPlaying)
	return Track{
		Title:    title,
		Artist:   artist,
		AppID:    s.SourceAppUserModelID,
		Playing:  isPlaying,
		OK:       ok,
		Position: pos,
		Duration: dur,
	}
}

// NowPlaying returns the current SMTC session track info (best-effort).
func NowPlaying() Track {
	return NowPlayingFiltered(Filter{})
}

// NowPlayingFiltered returns track info from sessions matching f.
// Uses the cached Sessions() snapshot only (no CurrentSession COM) to avoid
// racing the monitor event loop during media property refreshes.
func NowPlayingFiltered(f Filter) (tr Track) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("media: SMTC panic recovered: %v", r)
			tr = Track{}
		}
	}()

	m, err := getMonitor()
	if err != nil || m == nil {
		return Track{}
	}

	sessions := m.Sessions()
	var fallback *smtc.SessionInfo
	for i := range sessions {
		s := &sessions[i]
		noteSource(s.SourceAppUserModelID, s.MediaInfo.Title, s.MediaInfo.Artist)
		if !appAllowed(s, f) {
			continue
		}
		if s.PlaybackStatus == smtc.PlaybackStatusPlaying {
			if t := trackFromSession(s); t.OK {
				return t
			}
		}
		if fallback == nil {
			fallback = s
		}
	}
	if fallback != nil {
		return trackFromSession(fallback)
	}
	return Track{}
}
