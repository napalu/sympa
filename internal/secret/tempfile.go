package secret

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// TempFile holds a path to a secure temporary file and its cleanup function.
type TempFile struct {
	Path    string
	cleanup func()
}

// Create creates a secure temporary file, preferring RAM-backed storage.
//   - Linux: uses /dev/shm (tmpfs)
//   - macOS: creates a RAM disk via hdiutil
//   - Fallback: os.TempDir() with a warning
func Create() (*TempFile, error) {
	dir, detach := ramDir()
	if dir == "" {
		dir = os.TempDir()
		fmt.Fprintln(os.Stderr, "warning: RAM-backed storage not available, using disk-backed temp directory")
	}

	sympaDir := filepath.Join(dir, "sympa")
	if err := os.MkdirAll(sympaDir, 0700); err != nil {
		if detach != nil {
			detach()
		}
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}

	f, err := os.CreateTemp(sympaDir, "secret-*")
	if err != nil {
		if detach != nil {
			detach()
		}
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	if err := f.Chmod(0600); err != nil {
		f.Close()
		os.Remove(f.Name())
		if detach != nil {
			detach()
		}
		return nil, fmt.Errorf("setting temp file permissions: %w", err)
	}
	path := f.Name()
	f.Close()

	return &TempFile{Path: path, cleanup: detach}, nil
}

// Cleanup overwrites the temp file contents with random data, removes it,
// and detaches any RAM disk that was created.
func (t *TempFile) Cleanup() {
	info, err := os.Stat(t.Path)
	if err == nil && info.Size() > 0 {
		if f, err := os.OpenFile(t.Path, os.O_WRONLY, 0); err == nil {
			noise := make([]byte, info.Size())
			rand.Read(noise)
			f.Write(noise)
			f.Sync()
			f.Close()
		}
	}
	os.Remove(t.Path)

	if t.cleanup != nil {
		t.cleanup()
	}
}
