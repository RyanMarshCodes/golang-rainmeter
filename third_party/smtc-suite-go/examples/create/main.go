//go:build windows && cgo

package main

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/create"
)

func main() {
	fmt.Println("Creating SMTC session...")
	thumbnail := ""
	if len(os.Args) > 1 {
		thumbnail = os.Args[1]
	}

	tracks := []smtc.MediaInfo{
		{
			Title:        "First Track",
			Artist:       "smtc-suite-go",
			AlbumTitle:   "Create API Demo",
			TrackNumber:  1,
			PlaybackType: smtc.PlaybackTypeMusic,
		},
		{
			Title:        "Second Track",
			Artist:       "smtc-suite-go",
			AlbumTitle:   "Create API Demo",
			TrackNumber:  2,
			PlaybackType: smtc.PlaybackTypeMusic,
		},
		{
			Title:        "Third Track",
			Artist:       "smtc-suite-go",
			AlbumTitle:   "Create API Demo",
			TrackNumber:  3,
			PlaybackType: smtc.PlaybackTypeMusic,
		},
	}

	c, err := create.New(&create.Config{
		MediaInfo:      tracks[0],
		AppMediaID:     "examples/create",
		PlaybackStatus: smtc.PlaybackStatusPlaying,
		TimelineInfo: smtc.TimelineInfo{
			StartTime:   0,
			EndTime:     3 * time.Minute,
			MinSeekTime: 0,
			MaxSeekTime: 3 * time.Minute,
			Position:    0,
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer c.Close()

	fmt.Println("SMTC session is active. Open the Windows media overlay to inspect it.")
	fmt.Println("Use Play/Pause/Next/Previous media keys or overlay buttons.")
	if thumbnail == "" {
		fmt.Println("Pass a cover image URL as argv[1] to test artwork reliably.")
		fmt.Println("Local image paths are accepted, but Windows Shell may ignore them.")
	} else if isURI(thumbnail) {
		if err := c.SetThumbnailFromURI(thumbnail); err != nil {
			fmt.Printf("SetThumbnailFromURI error: %v\n", err)
		} else {
			fmt.Printf("Cover artwork URI: %s\n", thumbnail)
		}
	} else {
		if err := c.SetThumbnailFromFile(thumbnail); err != nil {
			fmt.Printf("SetThumbnailFromFile error: %v\n", err)
		} else {
			fmt.Printf("Cover artwork file: %s\n", thumbnail)
			fmt.Println("Note: Windows Shell may ignore local file thumbnails; try an https URL if it does not appear.")
		}
	}

	trackIndex := 0
	status := smtc.PlaybackStatusPlaying
	position := time.Duration(0)
	duration := 3 * time.Minute
	var stateMu sync.Mutex
	eventsDone := make(chan struct{})

	go func() {
		defer close(eventsDone)
		deadline := time.After(45 * time.Second)
		for {
			select {
			case button := <-c.ButtonEvents():
				fmt.Printf("ButtonPressed: %s\n", button)

				stateMu.Lock()
				switch button {
				case smtc.ButtonPlay:
					status = smtc.PlaybackStatusPlaying
					setStatus(c, status)
				case smtc.ButtonPause:
					status = smtc.PlaybackStatusPaused
					setStatus(c, status)
				case smtc.ButtonStop:
					status = smtc.PlaybackStatusStopped
					setStatus(c, status)
				case smtc.ButtonNext:
					trackIndex = (trackIndex + 1) % len(tracks)
					status = smtc.PlaybackStatusPlaying
					position = 0
					setTrack(c, tracks[trackIndex], status, position, duration)
				case smtc.ButtonPrevious:
					trackIndex = (trackIndex + len(tracks) - 1) % len(tracks)
					status = smtc.PlaybackStatusPlaying
					position = 0
					setTrack(c, tracks[trackIndex], status, position, duration)
				}
				stateMu.Unlock()
			case <-deadline:
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for i := 0; i < 30; i++ {
		<-ticker.C
		stateMu.Lock()
		if status == smtc.PlaybackStatusPlaying {
			position += time.Second
			if position >= duration {
				trackIndex = (trackIndex + 1) % len(tracks)
				position = 0
				setTrack(c, tracks[trackIndex], status, position, duration)
			} else {
				setTimeline(c, position, duration)
			}
		}
		stateMu.Unlock()
	}

	fmt.Println("Waiting for media button events for 15s...")
	<-eventsDone
}

func isURI(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme != ""
}

func setTrack(c *create.Creator, info smtc.MediaInfo, status smtc.PlaybackStatus, position, duration time.Duration) {
	if err := c.SetMediaInfo(info); err != nil {
		fmt.Printf("SetMediaInfo error: %v\n", err)
		return
	}
	if err := c.SetPlaybackStatus(status); err != nil {
		fmt.Printf("SetPlaybackStatus error: %v\n", err)
		return
	}
	setTimeline(c, position, duration)
	fmt.Printf("Now playing: %s (%s)\n", info.Title, status)
}

func setStatus(c *create.Creator, status smtc.PlaybackStatus) {
	if err := c.SetPlaybackStatus(status); err != nil {
		fmt.Printf("SetPlaybackStatus error: %v\n", err)
		return
	}
	fmt.Printf("Playback status: %s\n", status)
}

func setTimeline(c *create.Creator, position, duration time.Duration) {
	err := c.SetTimelineInfo(smtc.TimelineInfo{
		StartTime:   0,
		EndTime:     duration,
		MinSeekTime: 0,
		MaxSeekTime: duration,
		Position:    position,
	})
	if err != nil {
		fmt.Printf("SetTimelineInfo error: %v\n", err)
		return
	}
	fmt.Printf("Timeline: %s / %s\n", position.Round(time.Second), duration.Round(time.Second))
}
