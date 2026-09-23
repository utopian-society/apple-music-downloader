package app

import (
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	activeTempMu    sync.Mutex
	activeTempFiles = make(map[string]struct{})
	signalInitOnce  sync.Once
)

func initTempSignalHandler() {
	signalInitOnce.Do(func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			cleanupActiveTempFiles()
			os.Exit(130)
		}()
	})
}

func registerTempFile(path string) {
	initTempSignalHandler()
	activeTempMu.Lock()
	defer activeTempMu.Unlock()
	activeTempFiles[path] = struct{}{}
}

func unregisterTempFile(path string) {
	activeTempMu.Lock()
	defer activeTempMu.Unlock()
	delete(activeTempFiles, path)
}

func cleanupActiveTempFiles() {
	activeTempMu.Lock()
	defer activeTempMu.Unlock()
	for path := range activeTempFiles {
		_ = removeWithRetry(path)
	}
}

// cleanStaleTempFiles removes any existing files in dir matching pattern.
func cleanStaleTempFiles(dir, pattern string) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}
	for _, match := range matches {
		_ = removeWithRetry(match)
	}
}

// removeWithRetry attempts to remove a file, retrying briefly on Windows/OneDrive locks.
func removeWithRetry(path string) error {
	var err error
	for i := 0; i < 5; i++ {
		err = os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return err
}
