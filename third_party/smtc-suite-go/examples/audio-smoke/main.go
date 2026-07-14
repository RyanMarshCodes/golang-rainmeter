//go:build windows && cgo

package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio/loopback"
)

func main() {
	duration := flag.Duration("duration", 5*time.Second, "capture duration")
	flag.Parse()

	var packets int
	var bytes int
	var silent int

	capturer, err := loopback.New(nil)
	if err != nil {
		panic(err)
	}
	defer capturer.Close()

	format := capturer.Format()
	fmt.Printf("format: %d Hz, %d ch, %d bits, blockAlign=%d, sampleFormat=%v\n",
		format.SampleRate,
		format.Channels,
		format.BitsPerSample,
		format.BlockAlign,
		format.SampleFormat,
	)

	if err := capturer.Start(); err != nil {
		panic(err)
	}
	defer capturer.Stop()

	deadline := time.After(*duration)
	for {
		select {
		case frame, ok := <-capturer.Frames():
			if !ok {
				fmt.Printf("packets=%d bytes=%d silent=%d\n", packets, bytes, silent)
				return
			}
			packets++
			bytes += len(frame.Data)
			if frame.Silent {
				silent++
			}
		case err, ok := <-capturer.Errors():
			if ok && err != nil {
				panic(err)
			}
		case <-deadline:
			fmt.Printf("packets=%d bytes=%d silent=%d\n", packets, bytes, silent)
			return
		}
	}
}
