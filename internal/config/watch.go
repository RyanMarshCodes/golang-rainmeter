package config

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch calls onChange (debounced) when the config file changes on disk.
// It returns a stop function.
func Watch(store *Store, onChange func()) (stop func(), err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(store.Path())
	if err := watcher.Add(dir); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		var timer *time.Timer
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()

		base := filepath.Base(store.Path())
		for {
			select {
			case <-done:
				return
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("config watch: %v", err)
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Base(ev.Name) != base {
					continue
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
					continue
				}
				if store.ShouldIgnoreReload() {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(200*time.Millisecond, func() {
					if store.ShouldIgnoreReload() {
						return
					}
					onChange()
				})
			}
		}
	}()

	return func() {
		close(done)
		_ = watcher.Close()
	}, nil
}
