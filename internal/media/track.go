package media

import "time"

// Track is the currently active media session metadata.
type Track struct {
	Title    string
	Artist   string
	AppID    string
	Playing  bool
	OK       bool
	Position time.Duration
	Duration time.Duration
}

// Progress returns 0–1 playback fraction, or 0 if unknown.
func (t Track) Progress() float64 {
	if t.Duration <= 0 {
		return 0
	}
	p := float64(t.Position) / float64(t.Duration)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// Filter selects which SMTC sessions count as now-playing sources.
//
// Allow entries are substring matches (case-insensitive) against the session's
// SourceAppUserModelID and common media metadata fields (title/artist/album…).
// Empty Allow = accept any app (still subject to Deny).
//
// Note: Chromium browser *tabs* share one SMTC app id — you cannot filter by
// URL/tab. Install a site as a PWA (e.g. YouTube Music → Install app) to get a
// distinct AppUserModelID that can be allowlisted.
type Filter struct {
	Allow []string // substring match; empty = allow all (minus Deny)
	Deny  []string // substring match; always excluded
}
