package content

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aadhii-yz/quest/internal/cache"
	"github.com/aadhii-yz/quest/internal/provider"
)

// Bundle is one verified, extracted course bundle.
type Bundle struct {
	// Dir holds the extracted files: course-definition.yml, LICENSE,
	// stage_descriptions/ and starter_templates/.
	Dir string
	// Manifest is the decoded manifest.json that was verified against the
	// index and quest's pins.
	Manifest Manifest
}

// Manifest describes a bundle's provenance. Files maps every bundled file to
// its SHA256 digest; manifest.json is not listed.
type Manifest struct {
	Format         int               `json:"format"`
	ContentRelease string            `json:"content_release"`
	Provider       string            `json:"provider"`
	Course         string            `json:"course"`
	Name           string            `json:"name"`
	Spec           Spec              `json:"spec"`
	Builder        Builder           `json:"builder"`
	Files          map[string]string `json:"files"`
}

// Spec pins the upstream course specification repository.
type Spec struct {
	Repo        string `json:"repo"`
	Commit      string `json:"commit"`
	License     string `json:"license"`
	LicenseFile string `json:"license_file"`
}

// Builder records who built the bundle. It is provenance, not trust.
type Builder struct {
	Repo   string `json:"repo"`
	Commit string `json:"commit"`
}

// DefinitionPath is the course-definition.yml inside the bundle.
func (b *Bundle) DefinitionPath() string {
	return filepath.Join(b.Dir, "course-definition.yml")
}

// Course fetches, extracts and verifies one course bundle. A verified cached
// extraction is returned without network access. A bundle that fails any check
// is removed and never used.
func (s *Store) Course(ctx context.Context, slug string, progress cache.ProgressFunc) (*Bundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, err := s.indexLocked(ctx)
	if err != nil {
		return nil, err
	}
	entry, ok := idx.Course(slug)
	if !ok {
		return nil, fmt.Errorf("content: course %q is not in release %s", slug, s.release)
	}

	assets := s.assets()
	dir := filepath.Join(assets.Dir, slug)
	if b, err := s.loadBundle(dir, entry); err == nil {
		return b, nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("content: reset %s: %w", dir, err)
	}

	archive, err := assets.Fetch(ctx, providerArtifact(entry, s.base()), progress)
	if err != nil {
		return nil, err
	}
	if err := cache.ExtractTarGz(ctx, archive, dir); err != nil {
		return nil, err
	}

	b, err := s.loadBundle(dir, entry)
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return b, nil
}

// providerArtifact builds the download description for one bundle entry. It
// lives here so the index remains the only source of bundle URLs and digests.
func providerArtifact(entry IndexCourse, base string) provider.Artifact {
	return provider.Artifact{
		URL:      base + "/" + entry.Bundle.Asset,
		SHA256:   entry.Bundle.SHA256,
		Filename: entry.Bundle.Asset,
		Size:     entry.Bundle.Size,
	}
}

func (s *Store) loadBundle(dir string, entry IndexCourse) (*Bundle, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := cache.VerifyFile(manifestPath, entry.ManifestSHA256); err != nil {
		return nil, fmt.Errorf("content: verify manifest for %s: %w", entry.Slug, err)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("content: read manifest for %s: %w", entry.Slug, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("content: parse manifest for %s: %w", entry.Slug, err)
	}
	if err := s.validateManifest(dir, entry, &m); err != nil {
		return nil, err
	}
	return &Bundle{Dir: dir, Manifest: m}, nil
}

func (s *Store) validateManifest(dir string, entry IndexCourse, m *Manifest) error {
	if m.Format != Format {
		return fmt.Errorf("content: %s has unsupported manifest format %d", entry.Slug, m.Format)
	}
	if m.ContentRelease != s.release {
		return fmt.Errorf("content: %s manifest release %q, want %q", entry.Slug, m.ContentRelease, s.release)
	}
	if m.Provider != Provider || m.Course != entry.Slug {
		return fmt.Errorf("content: %s manifest is for %q/%q", entry.Slug, m.Provider, m.Course)
	}

	pin, ok := s.pins[entry.Slug]
	if !ok {
		return fmt.Errorf("content: no spec pin for course %q", entry.Slug)
	}
	if entry.SpecCommit != pin.Commit || m.Spec.Commit != pin.Commit {
		return fmt.Errorf("content: %s spec commit %q, want pinned %q", entry.Slug, m.Spec.Commit, pin.Commit)
	}
	if entry.License != pin.License || m.Spec.License != pin.License {
		return fmt.Errorf("content: %s license %q, want pinned %q", entry.Slug, m.Spec.License, pin.License)
	}
	if m.Spec.LicenseFile == "" {
		return fmt.Errorf("content: %s manifest has no license file", entry.Slug)
	}

	if _, ok := m.Files["course-definition.yml"]; !ok {
		return fmt.Errorf("content: %s manifest is missing course-definition.yml", entry.Slug)
	}
	if _, ok := m.Files[m.Spec.LicenseFile]; !ok {
		return fmt.Errorf("content: %s manifest does not list %s", entry.Slug, m.Spec.LicenseFile)
	}

	for name, digest := range m.Files {
		rel, err := safeRelPath(name)
		if err != nil {
			return err
		}
		if err := cache.VerifyFile(filepath.Join(dir, rel), digest); err != nil {
			return fmt.Errorf("content: verify %s in %s: %w", name, entry.Slug, err)
		}
	}
	return nil
}
