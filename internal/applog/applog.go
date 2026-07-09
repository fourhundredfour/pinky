package applog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

const maxLogSize = 5 * 1024 * 1024 // 5 MB

// Init opens the log file at path for appending, rotating to path.old first
// if the existing file is larger than 5 MB. It configures the default logger
// to write to this file with microsecond precision.
func Init(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("applog: failed to create log directory: %w", err)
	}

	// Check if rotation is needed
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
		oldPath := path + ".old"
		_ = os.Remove(oldPath) // Ignore error if it doesn't exist
		if err := os.Rename(path, oldPath); err != nil {
			log.Printf("applog: failed to rotate log file: %v", err)
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("applog: failed to open log file: %w", err)
	}

	// Set output of default logger to the file
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Printf("applog: initialized logging at %s", path)
	return nil
}

// Go runs fn in a new goroutine with panic recovery and logging.
func Go(name string, fn func()) {
	go func() {
		defer RecoverAndLog(name)
		fn()
	}()
}

// RecoverAndLog recovers from any panic, logs the error and stack trace,
// and prevents the process from crashing. It is designed to be deferred.
func RecoverAndLog(name string) {
	if r := recover(); r != nil {
		log.Printf("PANIC RECOVERED [%s]: %v\nStack trace:\n%s", name, r, string(debug.Stack()))
	}
}
