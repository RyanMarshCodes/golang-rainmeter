package media

import (
	"sync"
	"time"
)

// playbackClock extrapolates SMTC timeline Position between sparse updates.
// Many players (esp. browsers) only fire TimelinePropertiesChanged on seek/
// pause/track change, so a 100ms UI poll still sees a frozen cached Position.
type playbackClock struct {
	mu      sync.Mutex
	key     string
	smtcPos time.Duration
	basePos time.Duration
	baseAt  time.Time
	rate    float64
	playing bool
}

var trackClock playbackClock

func (c *playbackClock) position(key string, smtcPos, dur time.Duration, rate float64, playing bool) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if rate <= 0 {
		rate = 1
	}

	reset := func() {
		c.key = key
		c.smtcPos = smtcPos
		c.basePos = smtcPos
		c.baseAt = now
		c.rate = rate
		c.playing = playing
	}

	if key == "" || key != c.key {
		reset()
		return clampPos(smtcPos, dur)
	}

	if smtcPos != c.smtcPos {
		c.smtcPos = smtcPos
		c.basePos = smtcPos
		c.baseAt = now
	}

	if c.playing != playing || c.rate != rate {
		if c.playing {
			c.basePos += time.Duration(float64(now.Sub(c.baseAt)) * c.rate)
		}
		c.playing = playing
		c.rate = rate
		c.baseAt = now
	}

	out := c.basePos
	if c.playing {
		out += time.Duration(float64(now.Sub(c.baseAt)) * c.rate)
	}
	return clampPos(out, dur)
}

func clampPos(pos, dur time.Duration) time.Duration {
	if pos < 0 {
		return 0
	}
	if dur > 0 && pos > dur {
		return dur
	}
	return pos
}
