package persist

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchDebounce = 300 * time.Millisecond

// watcher watches the data file's directory (not the file itself: sync
// clients and our atomic save replace the file via rename, which would drop
// a watch on the file) and calls onChange once activity settles.
type watcher struct {
	fsw       *fsnotify.Watcher
	done      chan struct{}
	closeOnce sync.Once
}

func startWatcher(path string, onChange func()) (*watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fsw.Add(filepath.Dir(path)); err != nil {
		fsw.Close()
		return nil, err
	}
	w := &watcher{fsw: fsw, done: make(chan struct{})}
	name := filepath.Base(path)
	go func() {
		var timer *time.Timer
		for {
			select {
			case <-w.done:
				if timer != nil {
					timer.Stop()
				}
				return
			case ev, ok := <-fsw.Events:
				if !ok {
					return
				}
				if !strings.EqualFold(filepath.Base(ev.Name), name) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(watchDebounce, onChange)
			case _, ok := <-fsw.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return w, nil
}

func (w *watcher) close() {
	w.closeOnce.Do(func() {
		close(w.done)
		w.fsw.Close()
	})
}
