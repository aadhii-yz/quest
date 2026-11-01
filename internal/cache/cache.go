// Package cache owns the user-visible cache directory for downloaded tester
// assets and verifies their digests.
//
// The cache is safe to delete. It is created with 0700 permissions and never
// lives inside the repository tree. Verification is not optional: callers
// receive a verified digest or an error, and there is no "skip" path.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const dirPerm = 0o700

// Store is a directory of downloaded assets.
type Store struct {
	Dir string
}

// NewStore returns a store rooted at dir.
func NewStore(dir string) *Store { return &Store{Dir: dir} }

// Ensure creates the cache directory with 0700 permissions. An existing
// directory is tightened to 0700 as well.
func (s *Store) Ensure() error {
	if s.Dir == "" {
		return errors.New("cache: empty directory")
	}
	if err := os.MkdirAll(s.Dir, dirPerm); err != nil {
		return fmt.Errorf("cache: create %s: %w", s.Dir, err)
	}
	if err := os.Chmod(s.Dir, dirPerm); err != nil {
		return fmt.Errorf("cache: permissions for %s: %w", s.Dir, err)
	}
	return nil
}

// Path returns the absolute path for an asset filename. It rejects names that
// are empty or would escape the cache directory.
func (s *Store) Path(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", fmt.Errorf("cache: invalid asset name %q", name)
	}
	return filepath.Join(s.Dir, name), nil
}

// VerifyFile checks that path matches the expected hex SHA256 digest.
func VerifyFile(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return VerifyReader(f, want)
}

// VerifyReader checks that r hashes to the expected hex SHA256 digest. It
// fails closed for an empty or malformed digest.
func VerifyReader(r io.Reader, want string) error {
	want = strings.ToLower(strings.TrimSpace(want))
	if len(want) != sha256.Size*2 {
		return fmt.Errorf("cache: invalid sha256 digest %q", want)
	}
	if _, err := hex.DecodeString(want); err != nil {
		return fmt.Errorf("cache: invalid sha256 digest %q: %w", want, err)
	}

	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return fmt.Errorf("cache: hash: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("cache: sha256 mismatch: got %s want %s", got, want)
	}
	return nil
}
