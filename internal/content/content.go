// Package content loads course content from the pinned quest-source release.
//
// The trust chain is fixed: the app pins one release tag and the SHA256 of its
// index.json; the index pins each bundle's digest and size; the bundle
// manifest pins the upstream spec repository, commit and license, which must
// agree with internal/provider/codecrafters pins.
//
// Nothing here resolves "latest", and nothing from a bundle is executed.
// Downloads and verification go through internal/cache, so content and tester
// assets share one verified path.
package content

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aadhii-yz/quest/internal/cache"
)

// Pinned content release. These three values are the only ones that need to
// change when content moves forward.
const (
	// Repo is the content repository in owner/repo form, without a scheme.
	Repo = "aadhii-yz/quest-source"
	// Release is the pinned release tag. Never resolve "latest".
	Release = "content-0.1.0"
	// IndexSHA256 is the digest of that release's index.json.
	IndexSHA256 = "c0f520f9053980eda7759cf43de20618a2f5ec50f9f2353ce45e3af744975dce"

	// Format is the bundle and index format this package understands.
	Format = 1
	// Provider is the provider id content bundles carry.
	Provider = "codecrafters"
)

// Pin is what quest itself pins for one course. It is compared with the bundle
// manifest before any content is used.
type Pin struct {
	Commit  string
	License string // SPDX identifier, e.g. "MIT"
}

// Store loads index and bundle data from a cache directory.
type Store struct {
	cacheDir    string
	pins        map[string]Pin
	client      *http.Client
	release     string
	indexSHA256 string
	baseURL     string

	mu    sync.Mutex
	index *Index
}

// New returns a store for the pinned release. cacheDir is the user cache
// directory; content lives under content/<release> inside it.
func New(cacheDir string, pins map[string]Pin) *Store {
	return &Store{
		cacheDir:    cacheDir,
		pins:        pins,
		release:     Release,
		indexSHA256: IndexSHA256,
	}
}

// Release is the pinned content release tag.
func (s *Store) Release() string { return s.release }

// Root is the directory that holds the index and extracted bundles.
func (s *Store) Root() string {
	return filepath.Join(s.cacheDir, "content", s.release)
}

// base returns the release download URL without a trailing slash.
func (s *Store) base() string {
	if s.baseURL != "" {
		return s.baseURL
	}
	return "https://github.com/" + Repo + "/releases/download/" + s.release
}

// assets returns the cache store rooted at the content directory. It is a new
// value each call, so callers never share mutable cache state.
func (s *Store) assets() *cache.Store {
	st := cache.NewStore(s.Root())
	st.Client = s.client
	return st
}

// validSlug reports whether slug is a single safe path component.
func validSlug(slug string) bool {
	if slug == "" || slug == "." || slug == ".." {
		return false
	}
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

// safeRelPath validates a relative path taken from a manifest.
func safeRelPath(name string) (string, error) {
	if name == "" || path.IsAbs(name) || filepath.IsAbs(name) {
		return "", fmt.Errorf("content: unsafe manifest path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("content: unsafe manifest path %q", name)
	}
	native := filepath.FromSlash(clean)
	if !filepath.IsLocal(native) {
		return "", fmt.Errorf("content: unsafe manifest path %q", name)
	}
	return native, nil
}
