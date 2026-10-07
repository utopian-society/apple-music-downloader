package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const TempDirPrefix = "apple-music-downloader-"

// TempManager manages a per-run temporary directory, tracks created files and directories,
// and ensures idempotent cleanup on exit or cancellation.
type TempManager struct {
	mu          sync.Mutex
	baseDir     string
	rootDir     string
	keepFiles   bool
	activeFiles map[string]struct{}
	activeDirs  map[string]struct{}
	cleaned     bool
	sigOnce     sync.Once
}

var (
	activeTempMu    sync.Mutex
	activeTempFiles = make(map[string]struct{})
	signalInitOnce  sync.Once
)

// NewTempManager creates a new TempManager with its own per-run temp root directory.
func NewTempManager(baseDir string, keepFiles bool) (*TempManager, error) {
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("generate random temp name: %w", err)
	}
	rootDir := filepath.Join(baseDir, TempDirPrefix+hex.EncodeToString(randBytes))
	if err := os.MkdirAll(rootDir, 0700); err != nil {
		return nil, fmt.Errorf("create temp root dir: %w", err)
	}

	tm := &TempManager{
		baseDir:     baseDir,
		rootDir:     rootDir,
		keepFiles:   keepFiles,
		activeFiles: make(map[string]struct{}),
		activeDirs:  make(map[string]struct{}),
	}
	tm.initSignalHandler()
	return tm, nil
}

// Configure updates settings on the TempManager (e.g. after configuration is parsed).
func (tm *TempManager) Configure(baseDir string, keepFiles bool) {
	if tm == nil {
		return
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.keepFiles = keepFiles
	if baseDir != "" && baseDir != tm.baseDir {
		randBytes := make([]byte, 8)
		_, _ = rand.Read(randBytes)
		newRoot := filepath.Join(baseDir, TempDirPrefix+hex.EncodeToString(randBytes))
		if err := os.MkdirAll(newRoot, 0700); err == nil {
			if !tm.keepFiles && tm.rootDir != "" {
				_ = os.RemoveAll(tm.rootDir)
			}
			tm.baseDir = baseDir
			tm.rootDir = newRoot
		}
	}
}

// RootDir returns the absolute path to the temp root, recreating it if removed.
func (tm *TempManager) RootDir() string {
	if tm == nil {
		return os.TempDir()
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	_ = os.MkdirAll(tm.rootDir, 0700)
	tm.cleaned = false
	return tm.rootDir
}

func (tm *TempManager) initSignalHandler() {
	tm.sigOnce.Do(func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			_ = tm.Cleanup()
			cleanupActiveTempFiles()
			fmt.Println()
			os.Exit(130)
		}()
	})
}

// CreateTempFile creates a tracked temporary file inside the temp root.
func (tm *TempManager) CreateTempFile(prefix, suffix string) (*os.File, error) {
	if tm == nil {
		return os.CreateTemp("", prefix+"*"+suffix)
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if err := os.MkdirAll(tm.rootDir, 0700); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(tm.rootDir, prefix+"*"+suffix)
	if err != nil {
		return nil, err
	}
	tm.activeFiles[f.Name()] = struct{}{}
	return f, nil
}

// CreateTempDir creates a tracked temporary directory inside the temp root.
func (tm *TempManager) CreateTempDir(prefix string) (string, error) {
	if tm == nil {
		return os.MkdirTemp("", prefix+"*")
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if err := os.MkdirAll(tm.rootDir, 0700); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(tm.rootDir, prefix+"*")
	if err != nil {
		return "", err
	}
	tm.activeDirs[dir] = struct{}{}
	return dir, nil
}

// NewFilePath returns a path for a tracked temporary file inside the temp root.
func (tm *TempManager) NewFilePath(filename string) (string, error) {
	if tm == nil {
		return filepath.Join(os.TempDir(), filename), nil
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if err := os.MkdirAll(tm.rootDir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(tm.rootDir, filename)
	tm.activeFiles[path] = struct{}{}
	return path, nil
}

// isInside returns true if path is strictly inside root.
func isInside(root, path string) bool {
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "."
}

// RemoveFile safely removes a file if it is located inside the temp root.
func (tm *TempManager) RemoveFile(path string) error {
	if tm == nil || path == "" {
		return nil
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.activeFiles, path)
	if tm.keepFiles {
		return nil
	}
	if !isInside(tm.rootDir, path) {
		return fmt.Errorf("refusing to remove file outside temp root: %s", path)
	}
	return removeWithRetry(path)
}

// Cleanup removes all tracked temp files/dirs and the temp root. Idempotent.
func (tm *TempManager) Cleanup() error {
	if tm == nil {
		return nil
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.cleaned {
		return nil
	}
	if tm.keepFiles {
		fmt.Printf("Keeping temporary files at: %s\n", tm.rootDir)
		tm.cleaned = true
		return nil
	}
	var lastErr error
	if tm.rootDir != "" && strings.HasPrefix(filepath.Base(tm.rootDir), TempDirPrefix) {
		if err := os.RemoveAll(tm.rootDir); err != nil {
			lastErr = err
		}
	}
	tm.activeFiles = make(map[string]struct{})
	tm.activeDirs = make(map[string]struct{})
	tm.cleaned = true
	return lastErr
}

// SweepStaleTempRoots removes any leftover temp directories matching TempDirPrefix older than maxAge.
func SweepStaleTempRoots(baseDir string, maxAge time.Duration) (int, error) {
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-maxAge)
	swept := 0
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !strings.HasPrefix(name, TempDirPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			target := filepath.Join(baseDir, name)
			if err := os.RemoveAll(target); err == nil {
				swept++
			}
		}
	}
	return swept, nil
}

// Backward compatibility helpers below

func initTempSignalHandler() {
	signalInitOnce.Do(func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			cleanupActiveTempFiles()
			fmt.Println()
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

// copyFile copies the contents of src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// replaceFile atomically renames src to dst, falling back to copy+remove across filesystems.
func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	_ = os.Remove(src)
	return nil
}
