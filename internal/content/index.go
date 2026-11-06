package content

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aadhii-yz/quest/internal/provider"
)

// Index is the release catalog. It is intentionally small: the catalog renders
// from it without downloading any bundle.
type Index struct {
	Format         int           `json:"format"`
	ContentRelease string        `json:"content_release"`
	Provider       string        `json:"provider"`
	Courses        []IndexCourse `json:"courses"`
}

// IndexCourse is one course in the release index.
type IndexCourse struct {
	Slug               string    `json:"slug"`
	Name               string    `json:"name"`
	ShortDescriptionMD string    `json:"short_description_md"`
	License            string    `json:"license"`
	SpecCommit         string    `json:"spec_commit"`
	Stages             int       `json:"stages"`
	Languages          []string  `json:"languages"`
	Bundle             BundleRef `json:"bundle"`
	ManifestSHA256     string    `json:"manifest_sha256"`
}

// BundleRef pins one bundle asset within the release.
type BundleRef struct {
	Asset  string `json:"asset"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Course looks up one course by slug.
func (i *Index) Course(slug string) (IndexCourse, bool) {
	for _, c := range i.Courses {
		if c.Slug == slug {
			return c, true
		}
	}
	return IndexCourse{}, false
}

// Index returns the pinned release index, fetching and verifying it once per
// process. A cached index that matches the pinned digest is used without
// network access.
func (s *Store) Index(ctx context.Context) (*Index, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.indexLocked(ctx)
}

func (s *Store) indexLocked(ctx context.Context) (*Index, error) {
	if s.index != nil {
		return s.index, nil
	}

	assets := s.assets()
	path, err := assets.Fetch(ctx, provider.Artifact{
		URL:      s.base() + "/index.json",
		SHA256:   s.indexSHA256,
		Filename: "index.json",
	}, nil)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("content: read index: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("content: parse index: %w", err)
	}
	if err := s.validateIndex(&idx); err != nil {
		return nil, err
	}
	s.index = &idx
	return s.index, nil
}

func (s *Store) validateIndex(idx *Index) error {
	if idx.Format != Format {
		return fmt.Errorf("content: unsupported index format %d, want %d", idx.Format, Format)
	}
	if idx.ContentRelease != s.release {
		return fmt.Errorf("content: index release %q, want %q", idx.ContentRelease, s.release)
	}
	if idx.Provider != Provider {
		return fmt.Errorf("content: index provider %q, want %q", idx.Provider, Provider)
	}
	if len(idx.Courses) == 0 {
		return fmt.Errorf("content: index has no courses")
	}

	seen := make(map[string]bool, len(idx.Courses))
	for _, c := range idx.Courses {
		if !validSlug(c.Slug) {
			return fmt.Errorf("content: invalid course slug %q", c.Slug)
		}
		if seen[c.Slug] {
			return fmt.Errorf("content: duplicate course %q", c.Slug)
		}
		seen[c.Slug] = true

		if c.SpecCommit == "" || c.License == "" {
			return fmt.Errorf("content: course %q is missing spec metadata", c.Slug)
		}
		if c.Bundle.Asset == "" || filepath.Base(c.Bundle.Asset) != c.Bundle.Asset {
			return fmt.Errorf("content: course %q has invalid bundle asset %q", c.Slug, c.Bundle.Asset)
		}
		if c.Bundle.Size <= 0 || c.Bundle.SHA256 == "" || c.ManifestSHA256 == "" {
			return fmt.Errorf("content: course %q has incomplete bundle metadata", c.Slug)
		}
	}
	return nil
}
