//go:build windows && cgo

// Example: List all SMTC sessions and print live Now Playing events.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/monitor"
)

var (
	watchDuration = flag.Duration("duration", 60*time.Second, "how long to watch SMTC events")
	coversDir     = flag.String("covers-dir", "testdata", "directory for saved cover files")
	noCoverSave   = flag.Bool("no-cover-save", false, "disable saving cover artwork")
	savedCovers   = make(map[string]bool)
)

func main() {
	flag.Parse()

	m, err := monitor.New(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	sessions := m.Sessions()
	printSessions(sessions)
	if !*noCoverSave {
		saveSessionCovers(sessions)
	}
	if cur := m.CurrentSession(); cur != nil {
		fmt.Printf("Current session: %s - %s\n", cur.MediaInfo.Title, cur.MediaInfo.Artist)
	}

	fmt.Printf("Watching SMTC events for %s. Start, pause, seek, or change tracks in a media app.\n", *watchDuration)
	timer := time.NewTimer(*watchDuration)
	defer timer.Stop()
	for {
		select {
		case evt, ok := <-m.Events():
			if !ok {
				return
			}
			printEvent(evt)
		case <-timer.C:
			return
		}
	}
}

func printSessions(sessions []smtc.SessionInfo) {
	if len(sessions) == 0 {
		fmt.Println("No active SMTC media sessions found.")
		fmt.Println("Open a media app like Spotify, Apple Music, or a browser playing media.")
		return
	}

	fmt.Printf("Found %d media session(s):\n\n", len(sessions))
	for i, s := range sessions {
		fmt.Printf("[%d] %s\n", i+1, s.SourceAppUserModelID)
		printSession("    ", s)
		fmt.Println()
	}
}

func printEvent(evt monitor.ManagerEvent) {
	switch evt.Type {
	case monitor.ManagerEventSessionsChanged:
		fmt.Printf("SessionsChanged: %d session(s)\n", len(evt.Sessions))
		for _, s := range evt.Sessions {
			fmt.Printf("  - %s: %s - %s (%s)\n", s.SourceAppUserModelID, s.MediaInfo.Title, s.MediaInfo.Artist, s.PlaybackStatus)
		}
	case monitor.ManagerEventCurrentSessionChanged:
		fmt.Printf("CurrentSessionChanged: %s\n", evt.CurrentSessionID)
	case monitor.ManagerEventSessionPlaybackChanged:
		fmt.Printf("SessionPlaybackChanged: %s\n", evt.SessionID)
		printEventSession(evt.Session)
	case monitor.ManagerEventSessionTimelineChanged:
		fmt.Printf("SessionTimelineChanged: %s\n", evt.SessionID)
		printEventSession(evt.Session)
	case monitor.ManagerEventSessionMediaChanged:
		fmt.Printf("SessionMediaChanged: %s\n", evt.SessionID)
		printEventSession(evt.Session)
		if !*noCoverSave {
			saveEventCover(evt.Session)
		}
	}
}

func printEventSession(s *smtc.SessionInfo) {
	if s == nil {
		return
	}
	printSession("  ", *s)
}

func printSession(prefix string, s smtc.SessionInfo) {
	fmt.Printf("%sTitle:    %s\n", prefix, s.MediaInfo.Title)
	fmt.Printf("%sArtist:   %s\n", prefix, s.MediaInfo.Artist)
	fmt.Printf("%sAlbum:    %s\n", prefix, s.MediaInfo.AlbumTitle)
	fmt.Printf("%sCover:    %t (%d bytes, sha256=%s)\n", prefix, s.MediaInfo.ThumbnailAvailable, len(s.MediaInfo.ThumbnailData), shortHash(s.MediaInfo.ThumbnailHash))
	fmt.Printf("%sStatus:   %s\n", prefix, s.PlaybackStatus)
	fmt.Printf("%sMode:     type=%d repeat=%s shuffle=%t rate=%.2f\n", prefix, s.PlaybackType, s.AutoRepeatMode, s.IsShuffleActive, s.PlaybackRate)
	fmt.Printf("%sControls: play=%t pause=%t toggle=%t next=%t prev=%t seek=%t rate=%t\n", prefix, s.PlaybackControls.Play, s.PlaybackControls.Pause, s.PlaybackControls.PlayPauseToggle, s.PlaybackControls.Next, s.PlaybackControls.Previous, s.PlaybackControls.PlaybackPosition, s.PlaybackControls.PlaybackRate)
	fmt.Printf("%sPosition: %v / %v\n", prefix, s.TimelineInfo.Position.Round(time.Second), s.TimelineInfo.EndTime.Round(time.Second))
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func saveSessionCovers(sessions []smtc.SessionInfo) {
	for _, s := range sessions {
		saveCover(s.MediaInfo)
	}
}

func saveEventCover(session *smtc.SessionInfo) {
	if session == nil {
		return
	}
	saveCover(session.MediaInfo)
}

func saveCover(info smtc.MediaInfo) {
	if len(info.ThumbnailData) == 0 || info.ThumbnailHash == "" {
		return
	}
	if savedCovers[info.ThumbnailHash] {
		return
	}

	if err := os.MkdirAll(*coversDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Save cover: create %s failed: %v\n", *coversDir, err)
		return
	}

	name := fmt.Sprintf("cover-%s%s", shortHash(info.ThumbnailHash), imageExt(info.ThumbnailData))
	path := filepath.Join(*coversDir, name)
	if err := os.WriteFile(path, info.ThumbnailData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Save cover: write %s failed: %v\n", path, err)
		return
	}
	savedCovers[info.ThumbnailHash] = true
	fmt.Printf("Saved cover: %s (%d bytes)\n", path, len(info.ThumbnailData))
}

func imageExt(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return ".jpg"
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		return ".png"
	case len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))):
		return ".gif"
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return ".webp"
	default:
		return ".bin"
	}
}
