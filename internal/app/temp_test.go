package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
	r.Config.CoverFormat = "jpg"
	r.Config.CoverSize = "600x600"

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
