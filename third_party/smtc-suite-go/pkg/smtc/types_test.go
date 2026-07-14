package smtc

import (
	"testing"
	"time"
)

func TestPlaybackStatusString(t *testing.T) {
	tests := map[PlaybackStatus]string{
		PlaybackStatusClosed:   "Closed",
		PlaybackStatusOpened:   "Opened",
		PlaybackStatusChanging: "Changing",
		PlaybackStatusStopped:  "Stopped",
		PlaybackStatusPlaying:  "Playing",
		PlaybackStatusPaused:   "Paused",
		PlaybackStatus(99):     "Unknown",
	}
	for status, want := range tests {
		if got := status.String(); got != want {
			t.Fatalf("%d.String() = %q, want %q", status, got, want)
		}
	}
}

func TestAutoRepeatModeString(t *testing.T) {
	tests := map[AutoRepeatMode]string{
		AutoRepeatNone:     "None",
		AutoRepeatTrack:    "Track",
		AutoRepeatList:     "List",
		AutoRepeatMode(99): "Unknown",
	}
	for mode, want := range tests {
		if got := mode.String(); got != want {
			t.Fatalf("%d.String() = %q, want %q", mode, got, want)
		}
	}
}

func TestTimeSpanConversions(t *testing.T) {
	d := 2*time.Minute + 3*time.Second + 456*time.Millisecond + 78900*time.Nanosecond
	ticks := DurationToTicks(d)
	if got := TicksToDuration(ticks); got != d {
		t.Fatalf("TicksToDuration(DurationToTicks(%v)) = %v", d, got)
	}
}

func TestButtonString(t *testing.T) {
	tests := map[Button]string{
		ButtonPlay:        "Play",
		ButtonPause:       "Pause",
		ButtonStop:        "Stop",
		ButtonRecord:      "Record",
		ButtonFastForward: "FastForward",
		ButtonRewind:      "Rewind",
		ButtonNext:        "Next",
		ButtonPrevious:    "Previous",
		ButtonChannelUp:   "ChannelUp",
		ButtonChannelDown: "ChannelDown",
		Button(99):        "Unknown",
	}
	for button, want := range tests {
		if got := button.String(); got != want {
			t.Fatalf("%d.String() = %q, want %q", button, got, want)
		}
	}
}

func TestPlaybackControlsZeroValue(t *testing.T) {
	var controls PlaybackControls
	if controls.Play || controls.Pause || controls.Stop || controls.Next || controls.Previous || controls.PlaybackPosition {
		t.Fatalf("zero PlaybackControls should not enable any controls: %+v", controls)
	}
}

func TestDurationToTicksTruncatesSubTick(t *testing.T) {
	if got := DurationToTicks(99 * time.Nanosecond); got != 0 {
		t.Fatalf("DurationToTicks(99ns) = %d, want 0", got)
	}
	if got := DurationToTicks(100 * time.Nanosecond); got != 1 {
		t.Fatalf("DurationToTicks(100ns) = %d, want 1", got)
	}
}
