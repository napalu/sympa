package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// StoreMetadata holds optional metadata stored in the marker file.
type StoreMetadata struct {
	KeyfileFingerprint string `json:"keyfile_fingerprint,omitempty"`
}

const (
	markerFile = ".sympa-store"
	extension  = ".age"
	envDir     = "SYMPA_DIR"
)

// Store represents a sympa secret store on disk.
type Store struct {
	dir string
}

// New returns a Store rooted at the configured directory.
func New() *Store {
	return &Store{dir: storeDir()}
}

// Dir returns the store root directory.
func (s *Store) Dir() string {
	return s.dir
}

// Init creates the store directory and marker file.
func (s *Store) Init() error {
	return s.InitWithMetadata(StoreMetadata{})
}

// InitWithMetadata creates the store directory and marker file with optional metadata.
func (s *Store) InitWithMetadata(meta StoreMetadata) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("creating store directory: %w", err)
	}
	marker := filepath.Join(s.dir, markerFile)
	if _, err := os.Stat(marker); err == nil {
		return fmt.Errorf("store already initialized at %s", s.dir)
	}
	data, err := marshalMetadata(meta)
	if err != nil {
		return err
	}
	if err := os.WriteFile(marker, data, 0600); err != nil {
		return fmt.Errorf("creating marker file: %w", err)
	}
	return nil
}

// ReadMetadata reads store metadata from the marker file.
// Returns zero-value StoreMetadata for empty/missing marker files (backward compat).
func (s *Store) ReadMetadata() (StoreMetadata, error) {
	marker := filepath.Join(s.dir, markerFile)
	raw, err := os.ReadFile(marker)
	if err != nil {
		return StoreMetadata{}, fmt.Errorf("reading marker file: %w", err)
	}
	if len(raw) == 0 {
		return StoreMetadata{}, nil
	}
	var meta StoreMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return StoreMetadata{}, fmt.Errorf("parsing marker file: %w", err)
	}
	return meta, nil
}

// WriteMetadata writes store metadata to the marker file.
func (s *Store) WriteMetadata(meta StoreMetadata) error {
	data, err := marshalMetadata(meta)
	if err != nil {
		return err
	}
	marker := filepath.Join(s.dir, markerFile)
	if err := os.WriteFile(marker, data, 0600); err != nil {
		return fmt.Errorf("writing marker file: %w", err)
	}
	return nil
}

// marshalMetadata returns nil for empty metadata (backward compat), JSON otherwise.
func marshalMetadata(meta StoreMetadata) ([]byte, error) {
	if meta.KeyfileFingerprint == "" {
		return nil, nil
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}
	return data, nil
}

// IsInitialized checks whether the store has been initialized.
func (s *Store) IsInitialized() bool {
	_, err := os.Stat(filepath.Join(s.dir, markerFile))
	return err == nil
}

// Exists checks whether a secret exists in the store.
func (s *Store) Exists(name string) bool {
	path, err := s.safePath(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Read returns the raw encrypted bytes for a secret.
func (s *Store) Read(name string) ([]byte, error) {
	path, err := s.safePath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading secret %q: %w", name, err)
	}
	return data, nil
}

// Write stores encrypted data for a secret, creating parent directories as needed.
func (s *Store) Write(name string, data []byte) error {
	path, err := s.safePath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating parent directories: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing secret %q: %w", name, err)
	}
	return nil
}

// Remove deletes a secret file.
func (s *Store) Remove(name string) error {
	path, err := s.safePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing secret %q: %w", name, err)
	}
	s.cleanEmptyDirs(filepath.Dir(path))
	return nil
}

// RemoveDir removes a directory and all its contents.
func (s *Store) RemoveDir(name string) error {
	path, err := s.safeDirPath(name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("removing directory %q: %w", name, err)
	}
	s.cleanEmptyDirs(filepath.Dir(path))
	return nil
}

// IsDir checks whether a name refers to a directory in the store.
func (s *Store) IsDir(name string) bool {
	path, err := s.safeDirPath(name)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Rename moves a secret from one name to another.
func (s *Store) Rename(oldName, newName string) error {
	oldPath, err := s.safePath(oldName)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	newPath, err := s.safePath(newName)
	if err != nil {
		return fmt.Errorf("destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0700); err != nil {
		return fmt.Errorf("creating parent directories: %w", err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("moving %q to %q: %w", oldName, newName, err)
	}
	s.cleanEmptyDirs(filepath.Dir(oldPath))
	return nil
}

// List prints a tree of secrets under the given subfolder.
func (s *Store) List(subfolder string) error {
	root, err := s.safeDirPath(subfolder)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("cannot access %q: %w", subfolder, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", subfolder)
	}

	label := "sympa"
	if subfolder != "" {
		label = subfolder
	}
	fmt.Println(label)
	return printTree(root, "")
}

// Find returns secret names matching a glob pattern.
func (s *Store) Find(pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(s.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != s.dir && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, extension) {
			return nil
		}
		rel, _ := filepath.Rel(s.dir, path)
		name := strings.TrimSuffix(rel, extension)
		lowerName := strings.ToLower(name)
		lowerPattern := strings.ToLower(pattern)
		matched, _ := filepath.Match(lowerPattern, strings.ToLower(filepath.Base(name)))
		if matched || strings.Contains(lowerName, lowerPattern) {
			matches = append(matches, name)
		}
		return nil
	})
	return matches, err
}

// AllSecrets returns the names of all secrets in the store.
func (s *Store) AllSecrets() ([]string, error) {
	var names []string
	err := filepath.Walk(s.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != s.dir && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(path, extension) {
			rel, _ := filepath.Rel(s.dir, path)
			names = append(names, strings.TrimSuffix(rel, extension))
		}
		return nil
	})
	return names, err
}

// errInvalidName is returned when a secret name would escape the store root.
var errInvalidName = fmt.Errorf("invalid secret name: path escapes store")

// safePath validates name and returns the resolved .age file path under s.dir.
func (s *Store) safePath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("secret name cannot be empty")
	}
	if filepath.IsAbs(name) {
		return "", errInvalidName
	}
	cleaned := filepath.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", errInvalidName
	}
	full := filepath.Join(s.dir, cleaned+extension)
	if !strings.HasPrefix(full, s.dir+string(filepath.Separator)) {
		return "", errInvalidName
	}
	return full, nil
}

// safeDirPath validates name and returns the resolved directory path under s.dir.
func (s *Store) safeDirPath(name string) (string, error) {
	if name == "" {
		return s.dir, nil
	}
	if filepath.IsAbs(name) {
		return "", errInvalidName
	}
	cleaned := filepath.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", errInvalidName
	}
	full := filepath.Join(s.dir, cleaned)
	if !strings.HasPrefix(full, s.dir+string(filepath.Separator)) {
		return "", errInvalidName
	}
	return full, nil
}

// cleanEmptyDirs removes empty parent directories up to the store root.
func (s *Store) cleanEmptyDirs(dir string) {
	for dir != s.dir {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

func storeDir() string {
	if d := os.Getenv(envDir); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sympa"
	}
	return filepath.Join(home, ".sympa")
}

func printTree(dir, prefix string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Filter out hidden files/dirs except .age files
	var visible []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		visible = append(visible, e)
	}

	sort.Slice(visible, func(i, j int) bool {
		return visible[i].Name() < visible[j].Name()
	})

	for i, entry := range visible {
		isLast := i == len(visible)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		name := entry.Name()
		if entry.IsDir() {
			fmt.Printf("%s%s%s\n", prefix, connector, name)
			if err := printTree(filepath.Join(dir, name), prefix+childPrefix); err != nil {
				return err
			}
		} else if strings.HasSuffix(name, extension) {
			fmt.Printf("%s%s%s\n", prefix, connector, strings.TrimSuffix(name, extension))
		}
	}
	return nil
}
