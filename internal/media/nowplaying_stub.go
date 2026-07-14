//go:build !windows || !cgo

package media

// NowPlaying is unavailable without Windows + CGO.
func NowPlaying() Track {
	return Track{}
}

// NowPlayingFiltered is unavailable without Windows + CGO.
func NowPlayingFiltered(Filter) Track {
	return Track{}
}
