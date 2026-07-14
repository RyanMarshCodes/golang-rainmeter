//go:build windows && cgo

package wasapi

// #cgo LDFLAGS: -lole32
import "C"
