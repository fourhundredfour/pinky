package config

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounceWindow absorbs the burst of WRITE/CHMOD/RENAME events a single
// "save" in a text editor typically produces.
const debounceWindow = 250 * time.Millisecond

// Watcher watches a config file's containing directory (not the file
// itself - editors commonly save by renaming a temp file over the
// original, which stops inode-based watches from firing again) and reloads
// it whenever it changes, invoking onChange with the freshly loaded config.
//
// Reload errors (e.g. a transiently invalid file mid-write) are logged and
// otherwise ignored; the previous good config keeps being used until the
// next successful reload.
type Watcher struct {
	path     string
	onChange func(*Config)

	fs *fsnotify.Watcher

	mu    sync.Mutex
	timer *time.Timer

	done chan struct{}
}

// NewWatcher starts watching path's directory and returns the ready
// Watcher. Call Close to stop watching.
func NewWatcher(path string, onChange func(*Config)) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	if err := fs.Add(dir); err != nil {
		fs.Close()
		return nil, err
	}

	w := &Watcher{
		path:     path,
		onChange: onChange,
		fs:       fs,
		done:     make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	target := filepath.Clean(w.path)
	for {
		select {
		case event, ok := <-w.fs.Events:
			if !ok {
				return
			}
			if filepath.Clean(event.Name) != target {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			w.scheduleReload()
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			log.Printf("config: watcher error: %v", err)
		case <-w.done:
			return
		}
	}
}

func (w *Watcher) scheduleReload() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(debounceWindow, w.reload)
}

func (w *Watcher) reload() {
	cfg, err := Load(w.path)
	if err != nil {
		log.Printf("config: reload failed, keeping previous settings: %v", err)
		return
	}
	w.onChange(cfg)
}

// Close stops watching and releases the underlying OS resources.
func (w *Watcher) Close() error {
	close(w.done)
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	return w.fs.Close()
}
