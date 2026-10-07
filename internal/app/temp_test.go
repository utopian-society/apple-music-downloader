package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"amdl/internal/config"
	"amdl/internal/download"
)

func TestWriteCoverCleansStaleTempFiles(t *testing.T) {
	dir := t.TempDir()
	staleTemp := filepath.Join(dir, "cover.tmp-120699116")
	if err := os.WriteFile(staleTemp, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-jpeg-image-data"))
	}))
	defer server.Close()

	originalClient := download.Client
	t.Cleanup(func() {
		download.Client = originalClient
	})
	download.Client = server.Client()

	r := NewRunner(config.ConfigSet{})
	r.Config.Metadata.Artwork.Format = "jpg"
	r.Config.Metadata.Artwork.Size = "600x600"

	covPath, err := r.writeCover(dir, "cover", server.URL+"/image/{w}x{h}.jpg")
	if err != nil {
		t.Fatalf("writeCover() failed: %v", err)
	}

	if _, err := os.Stat(covPath); err != nil {
		t.Fatalf("expected cover file %s to exist: %v", covPath, err)
	}

	// Stale temp file must have been cleaned up
	if _, err := os.Stat(staleTemp); !os.IsNotExist(err) {
		t.Fatalf("stale temp file %s was not cleaned up", staleTemp)
	}

	// No .tmp-* files should remain in dir
	matches, err := filepath.Glob(filepath.Join(dir, "cover.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) > 0 {
		t.Fatalf("unexpected temp files remaining: %v", matches)
	}
}

func TestPrepareCollectionFolderCleansStaleTemp(t *testing.T) {
	dir := t.TempDir()
	albumDir := filepath.Join(dir, "Test Album")
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		t.Fatal(err)
	}
	staleTemp := filepath.Join(albumDir, "cover.tmp-120699116")
	if err := os.WriteFile(staleTemp, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	r := NewRunner(config.ConfigSet{})
	folder, err := r.prepareCollectionFolder(dir, "Test Album")
	if err != nil {
		t.Fatalf("prepareCollectionFolder() failed: %v", err)
	}
	if folder != albumDir {
		t.Fatalf("got folder %s, want %s", folder, albumDir)
	}

	if _, err := os.Stat(staleTemp); !os.IsNotExist(err) {
		t.Fatalf("stale temp file %s was not cleaned up by prepareCollectionFolder", staleTemp)
	}
}

func TestActiveTempFileCleanup(t *testing.T) {
	dir := t.TempDir()
	tempFile, err := os.CreateTemp(dir, "active.tmp-*")
	if err != nil {
		t.Fatal(err)
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()

	registerTempFile(tempPath)
	cleanupActiveTempFiles()

	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("expected registered temp file %s to be deleted by cleanupActiveTempFiles", tempPath)
	}
}

func TestTempManagerCreateAndCleanup(t *testing.T) {
	baseDir := t.TempDir()
	tm, err := NewTempManager(baseDir, false)
	if err != nil {
		t.Fatalf("NewTempManager failed: %v", err)
	}

	root := tm.RootDir()
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected root dir %s to exist: %v", root, err)
	}

	// Create temp file
	f, err := tm.CreateTempFile("test-", ".tmp")
	if err != nil {
		t.Fatalf("CreateTempFile failed: %v", err)
	}
	filePath := f.Name()
	_, _ = f.WriteString("hello temp")
	_ = f.Close()

	// Create temp dir
	subDir, err := tm.CreateTempDir("sub-")
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}

	// New file path
	trackedPath, err := tm.NewFilePath("tracked.txt")
	if err != nil {
		t.Fatalf("NewFilePath failed: %v", err)
	}
	if err := os.WriteFile(trackedPath, []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify they all exist
	for _, p := range []string{filePath, subDir, trackedPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
	}

	// Remove single file safely
	if err := tm.RemoveFile(filePath); err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed", filePath)
	}

	// Cleanup
	if err := tm.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected root %s to be deleted after Cleanup", root)
	}

	// Idempotent cleanup: calling again must not error
	if err := tm.Cleanup(); err != nil {
		t.Fatalf("second Cleanup call failed: %v", err)
	}
}

func TestTempManagerKeepTempFiles(t *testing.T) {
	baseDir := t.TempDir()
	tm, err := NewTempManager(baseDir, true)
	if err != nil {
		t.Fatalf("NewTempManager failed: %v", err)
	}
	root := tm.RootDir()

	trackedPath, err := tm.NewFilePath("keep-me.txt")
	if err != nil {
		t.Fatalf("NewFilePath failed: %v", err)
	}
	_ = os.WriteFile(trackedPath, []byte("keep"), 0644)

	if err := tm.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// With keepFiles=true, root and file must still exist
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected root %s to be preserved when keepFiles=true", root)
	}
	if _, err := os.Stat(trackedPath); err != nil {
		t.Fatalf("expected file %s to be preserved when keepFiles=true", trackedPath)
	}
}

func TestTempManagerRefusesOutsideDelete(t *testing.T) {
	baseDir := t.TempDir()
	tm, err := NewTempManager(baseDir, false)
	if err != nil {
		t.Fatalf("NewTempManager failed: %v", err)
	}
	defer func() { _ = tm.Cleanup() }()

	outsideFile := filepath.Join(baseDir, "outside.txt")
	_ = os.WriteFile(outsideFile, []byte("outside"), 0644)

	err = tm.RemoveFile(outsideFile)
	if err == nil {
		t.Fatalf("expected RemoveFile to fail for path outside temp root")
	}
	if !strings.Contains(err.Error(), "outside temp root") {
		t.Fatalf("unexpected error message: %v", err)
	}

	if _, err := os.Stat(outsideFile); err != nil {
		t.Fatalf("outside file was unexpectedly deleted: %v", err)
	}
}

func TestSweepStaleTempRoots(t *testing.T) {
	baseDir := t.TempDir()

	// 1. Stale temp root (> 24h old)
	staleDir := filepath.Join(baseDir, TempDirPrefix+"stale123")
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-25 * time.Hour)
	_ = os.Chtimes(staleDir, oldTime, oldTime)

	// 2. Fresh temp root (< 24h old)
	freshDir := filepath.Join(baseDir, TempDirPrefix+"fresh456")
	if err := os.MkdirAll(freshDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 3. Unrelated directory with different prefix (> 24h old)
	otherDir := filepath.Join(baseDir, "other-prefix-789")
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(otherDir, oldTime, oldTime)

	swept, err := SweepStaleTempRoots(baseDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("SweepStaleTempRoots failed: %v", err)
	}
	if swept != 1 {
		t.Fatalf("expected 1 directory swept, got %d", swept)
	}

	// Stale dir must be deleted
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("expected stale dir %s to be deleted", staleDir)
	}

	// Fresh dir must be preserved
	if _, err := os.Stat(freshDir); err != nil {
		t.Fatalf("expected fresh dir %s to be preserved", freshDir)
	}

	// Other prefix dir must be preserved
	if _, err := os.Stat(otherDir); err != nil {
		t.Fatalf("expected other dir %s to be preserved", otherDir)
	}
}
