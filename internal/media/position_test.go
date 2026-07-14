package media

import (
	"testing"
	"time"
)

func TestPlaybackClockExtrapolatesWhilePlaying(t *testing.T) {
	var c playbackClock
	pos := c.position("a|t", 10*time.Second, 100*time.Second, 1, true)
	if pos != 10*time.Second {
		t.Fatalf("initial: %v", pos)
	}
	c.mu.Lock()
	c.baseAt = time.Now().Add(-2 * time.Second)
	c.mu.Unlock()
	pos = c.position("a|t", 10*time.Second, 100*time.Second, 1, true)
	if pos < 11*time.Second+900*time.Millisecond || pos > 12*time.Second+200*time.Millisecond {
		t.Fatalf("extrapolated: %v", pos)
	}
}

func TestPlaybackClockResyncsOnSeek(t *testing.T) {
	var c playbackClock
	_ = c.position("a|t", 10*time.Second, 100*time.Second, 1, true)
	c.mu.Lock()
	c.baseAt = time.Now().Add(-5 * time.Second)
	c.mu.Unlock()
	pos := c.position("a|t", 40*time.Second, 100*time.Second, 1, true)
	if pos < 39*time.Second || pos > 41*time.Second {
		t.Fatalf("seek resync: %v", pos)
	}
}

func TestPlaybackClockFreezesWhenPaused(t *testing.T) {
	var c playbackClock
	_ = c.position("a|t", 10*time.Second, 100*time.Second, 1, true)
	c.mu.Lock()
	c.baseAt = time.Now().Add(-3 * time.Second)
	c.mu.Unlock()
	pos := c.position("a|t", 10*time.Second, 100*time.Second, 1, false)
	if pos < 12*time.Second+500*time.Millisecond || pos > 14*time.Second {
		t.Fatalf("fold on pause: %v", pos)
	}
	c.mu.Lock()
	c.baseAt = time.Now().Add(-2 * time.Second)
	c.mu.Unlock()
	pos2 := c.position("a|t", 10*time.Second, 100*time.Second, 1, false)
	if pos2 != pos {
		t.Fatalf("paused drifted: %v -> %v", pos, pos2)
	}
}
