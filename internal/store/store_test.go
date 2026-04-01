package store

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := &Store{dir: dir}
	// Create marker so IsInitialized passes
	os.WriteFile(filepath.Join(dir, markerFile), nil, 0600)
	return s
}

func TestSafePath_Traversal(t *testing.T) {
	s := testStore(t)

	bad := []string{
		"../../etc/passwd",
		"../outside",
		"foo/../../outside",
		"/absolute/path",
		"..",
		"../..",
		"",
	}
	for _, name := range bad {
		_, err := s.safePath(name)
		if err == nil {
			t.Errorf("safePath(%q) should have returned error", name)
		}
	}
}

func TestSafePath_Valid(t *testing.T) {
	s := testStore(t)

	good := []struct {
		name string
		want string // expected suffix relative to store dir
	}{
		{"simple", "simple.age"},
		{"email/gmail", "email/gmail.age"},
		{"a/b/c", "a/b/c.age"},
		{"foo/../bar", "bar.age"}, // cleaned but still under store
	}
	for _, tc := range good {
		path, err := s.safePath(tc.name)
		if err != nil {
			t.Errorf("safePath(%q) unexpected error: %v", tc.name, err)
			continue
		}
		want := filepath.Join(s.dir, tc.want)
		if path != want {
			t.Errorf("safePath(%q) = %q, want %q", tc.name, path, want)
		}
	}
}

func TestSafeDirPath_Traversal(t *testing.T) {
	s := testStore(t)

	bad := []string{
		"../../etc",
		"../outside",
		"/absolute",
		"..",
	}
	for _, name := range bad {
		_, err := s.safeDirPath(name)
		if err == nil {
			t.Errorf("safeDirPath(%q) should have returned error", name)
		}
	}
}

func TestSafeDirPath_Valid(t *testing.T) {
	s := testStore(t)

	// Empty name should return store dir
	path, err := s.safeDirPath("")
	if err != nil {
		t.Fatalf("safeDirPath(\"\") unexpected error: %v", err)
	}
	if path != s.dir {
		t.Errorf("safeDirPath(\"\") = %q, want %q", path, s.dir)
	}

	// Valid subfolder
	path, err = s.safeDirPath("email")
	if err != nil {
		t.Fatalf("safeDirPath(\"email\") unexpected error: %v", err)
	}
	want := filepath.Join(s.dir, "email")
	if path != want {
		t.Errorf("safeDirPath(\"email\") = %q, want %q", path, want)
	}
}

func TestExists_Traversal(t *testing.T) {
	s := testStore(t)
	// Should return false, not panic or escape
	if s.Exists("../../etc/passwd") {
		t.Error("Exists(../../etc/passwd) should return false")
	}
}

func TestRead_Traversal(t *testing.T) {
	s := testStore(t)
	_, err := s.Read("../../etc/passwd")
	if err == nil {
		t.Error("Read(../../etc/passwd) should return error")
	}
}

func TestWrite_Traversal(t *testing.T) {
	s := testStore(t)
	err := s.Write("../../etc/evil", []byte("data"))
	if err == nil {
		t.Error("Write(../../etc/evil) should return error")
	}
}

func TestRemove_Traversal(t *testing.T) {
	s := testStore(t)
	err := s.Remove("../../etc/passwd")
	if err == nil {
		t.Error("Remove(../../etc/passwd) should return error")
	}
}

func TestRemoveDir_Traversal(t *testing.T) {
	s := testStore(t)
	err := s.RemoveDir("../../etc")
	if err == nil {
		t.Error("RemoveDir(../../etc) should return error")
	}
}

func TestRename_Traversal(t *testing.T) {
	s := testStore(t)
	// Create a real secret to rename from
	os.MkdirAll(filepath.Join(s.dir, "test"), 0700)
	os.WriteFile(filepath.Join(s.dir, "test/secret.age"), []byte("data"), 0600)

	err := s.Rename("test/secret", "../../etc/evil")
	if err == nil {
		t.Error("Rename to ../../etc/evil should return error")
	}

	err = s.Rename("../../etc/passwd", "test/new")
	if err == nil {
		t.Error("Rename from ../../etc/passwd should return error")
	}
}
