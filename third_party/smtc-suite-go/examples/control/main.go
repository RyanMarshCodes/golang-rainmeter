//go:build windows && cgo

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/control"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/monitor"
)

func main() {
	sessionID, command, args := parseArgs(os.Args[1:])
	if command == "help" || command == "-h" || command == "--help" {
		fmt.Println(usage())
		return
	}
	if command == "sessions" {
		listSessions()
		return
	}

	ctrl, err := control.New(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	if command == "info" {
		printInfo(ctrl)
		return
	}

	if err := runCommand(ctrl, command, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", command, err)
		os.Exit(1)
	}
	fmt.Printf("%s OK\n", command)
}

func parseArgs(args []string) (sessionID string, command string, rest []string) {
	command = "toggle"
	for len(args) > 0 {
		if args[0] == "-session" && len(args) >= 2 {
			sessionID = args[1]
			args = args[2:]
			continue
		}
		command = args[0]
		rest = args[1:]
		return sessionID, command, rest
	}
	return sessionID, command, nil
}

func runCommand(ctrl *control.Controller, command string, args []string) error {
	switch command {
	case "play":
		return ctrl.Play()
	case "pause":
		return ctrl.Pause()
	case "toggle", "toggle-play-pause":
		return ctrl.TogglePlayPause()
	case "stop":
		return ctrl.Stop()
	case "next":
		return ctrl.Next()
	case "prev", "previous":
		return ctrl.Previous()
	case "ff", "fast-forward":
		return ctrl.FastForward()
	case "rewind":
		return ctrl.Rewind()
	case "seek":
		if len(args) == 0 {
			return fmt.Errorf("usage: seek <duration>, for example seek 30s or seek 1m15s")
		}
		pos, err := time.ParseDuration(args[0])
		if err != nil {
			return err
		}
		return ctrl.Seek(pos)
	case "rate":
		if len(args) == 0 {
			return fmt.Errorf("usage: rate <float>, for example rate 1.25")
		}
		rate, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return err
		}
		return ctrl.SetPlaybackRate(rate)
	case "shuffle":
		if len(args) == 0 {
			return fmt.Errorf("usage: shuffle on|off")
		}
		active, err := parseBoolArg(args[0])
		if err != nil {
			return err
		}
		return ctrl.SetShuffle(active)
	case "repeat":
		if len(args) == 0 {
			return fmt.Errorf("usage: repeat none|track|list")
		}
		mode, err := parseRepeatArg(args[0])
		if err != nil {
			return err
		}
		return ctrl.SetRepeatMode(mode)
	default:
		return fmt.Errorf("unknown command %q\n%s", command, usage())
	}
}

func printInfo(ctrl *control.Controller) {
	info, err := ctrl.MediaInfo()
	if err != nil {
		fmt.Printf("MediaInfo failed: %v\n", err)
		return
	}
	fmt.Printf("Current media: %s - %s\n", info.Title, info.Artist)
	fmt.Printf("Album: %s\n", info.AlbumTitle)
	fmt.Printf("Cover: %t, bytes=%d, sha256=%s\n", info.ThumbnailAvailable, len(info.ThumbnailData), shortHash(info.ThumbnailHash))
}

func listSessions() {
	m, err := monitor.New(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	sessions := m.Sessions()
	if len(sessions) == 0 {
		fmt.Println("No active SMTC sessions found.")
		return
	}
	for i, s := range sessions {
		fmt.Printf("[%d] %s\n", i+1, s.SourceAppUserModelID)
		fmt.Printf("    %s - %s (%s)\n", s.MediaInfo.Title, s.MediaInfo.Artist, s.PlaybackStatus)
	}
}

func parseBoolArg(value string) (bool, error) {
	switch value {
	case "1", "true", "on", "yes":
		return true, nil
	case "0", "false", "off", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected on|off, got %q", value)
	}
}

func parseRepeatArg(value string) (smtc.AutoRepeatMode, error) {
	switch value {
	case "none", "off":
		return smtc.AutoRepeatNone, nil
	case "track", "one":
		return smtc.AutoRepeatTrack, nil
	case "list", "all":
		return smtc.AutoRepeatList, nil
	default:
		return smtc.AutoRepeatNone, fmt.Errorf("expected none|track|list, got %q", value)
	}
}

func usage() string {
	return `Usage:
  go run ./examples/control [command]
  go run ./examples/control -session <app-user-model-id> [command]

Commands:
  help
  sessions
  info
  play | pause | toggle | stop
  next | prev | ff | rewind
  seek <duration>        example: seek 30s
  rate <float>           example: rate 1.25
  shuffle on|off
  repeat none|track|list`
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
