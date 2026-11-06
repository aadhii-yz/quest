package content

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
)

const testRelease = "content-test"

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func testPins(commit string) map[string]Pin {
	return map[string]Pin{"grep": {Commit: commit, License: "MIT"}}
}

type release struct {
	index      Index
	indexBytes []byte
	bundle     []byte
	asset      string
}

// buildRelease creates an index and one bundle. mutate runs after the file
// digests are computed, so tests can corrupt or change manifest fields.
func buildRelease(t *testing.T, indexCommit, manifestCommit string, mutate func(*Manifest)) release {
	t.Helper()

	files := map[string]string{
		"course-definition.yml":                 "slug: grep\nname: Build your own grep\n",
		"LICENSE":                               "MIT License\n",
		"stage_descriptions/base-01-cq2.md":     "match a character\n",
		"starter_templates/all/code/run.sh":     "#!/bin/sh\n",
		"starter_templates/go/code/app/main.go": "package main\n",
	}

	m := Manifest{
		Format:         Format,
		ContentRelease: testRelease,
		Provider:       Provider,
		Course:         "grep",
		Name:           "Build your own grep",
		Spec: Spec{
			Repo:        "https://github.com/codecrafters-io/build-your-own-grep",
			Commit:      manifestCommit,
			License:     "MIT",
			LicenseFile: "LICENSE",
		},
		Files: map[string]string{},
	}
	for name, body := range files {
		m.Files[name] = digest([]byte(body))
	}
	if mutate != nil {
		mutate(&m)
	}

	manifestBytes, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	entries := map[string][]byte{"manifest.json": manifestBytes}
	for name, body := range files {
		entries[name] = []byte(body)
	}
	bundle := tarGz(t, entries)

	asset := "codecrafters-grep-" + testRelease + ".tar.gz"
	idx := Index{
		Format:         Format,
		ContentRelease: testRelease,
		Provider:       Provider,
		Courses: []IndexCourse{{
			Slug:           "grep",
			Name:           "Build your own grep",
			License:        "MIT",
			SpecCommit:     indexCommit,
			Stages:         1,
			Languages:      []string{"go"},
			Bundle:         BundleRef{Asset: asset, SHA256: digest(bundle), Size: int64(len(bundle))},
			ManifestSHA256: digest(manifestBytes),
		}},
	}
	indexBytes, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	return release{index: idx, indexBytes: indexBytes, bundle: bundle, asset: asset}
}

func tarGz(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		body := entries[name]
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testServer(t *testing.T, indexBytes, bundleBytes []byte, asset string, indexHits, bundleHits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			if indexHits != nil {
				indexHits.Add(1)
			}
			w.Write(indexBytes)
		case "/" + asset:
			if bundleHits != nil {
				bundleHits.Add(1)
			}
			w.Write(bundleBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testStore(t *testing.T, cacheDir, base string, pins map[string]Pin, indexSHA string) *Store {
	t.Helper()
	st := New(cacheDir, pins)
	st.release = testRelease
	st.indexSHA256 = indexSHA
	st.baseURL = base
	return st
}

func TestIndexAndCourse(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, nil)

	var indexHits, bundleHits atomic.Int32
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, &indexHits, &bundleHits)

	cacheDir := t.TempDir()
	st := testStore(t, cacheDir, srv.URL, testPins(commit), digest(rel.indexBytes))

	idx, err := st.Index(context.Background())
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(idx.Courses) != 1 || idx.Courses[0].Slug != "grep" {
		t.Fatalf("index courses = %+v", idx.Courses)
	}

	b, err := st.Course(context.Background(), "grep", nil)
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	if b.Manifest.Spec.Commit != commit {
		t.Fatalf("manifest commit = %q", b.Manifest.Spec.Commit)
	}
	body, err := os.ReadFile(b.DefinitionPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "slug: grep") {
		t.Fatalf("definition = %q", body)
	}
	if want := filepath.Join(cacheDir, "content", testRelease, "grep"); b.Dir != want {
		t.Fatalf("bundle dir = %q, want %q", b.Dir, want)
	}

	// A second call is served from the verified extraction.
	if _, err := st.Course(context.Background(), "grep", nil); err != nil {
		t.Fatalf("cached course: %v", err)
	}
	if got := bundleHits.Load(); got != 1 {
		t.Fatalf("bundle downloads = %d, want 1", got)
	}
	if got := indexHits.Load(); got != 1 {
		t.Fatalf("index downloads = %d, want 1", got)
	}
}

func TestIndexRejectsDigestMismatch(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, nil)
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	st := testStore(t, t.TempDir(), srv.URL, testPins(commit), strings.Repeat("0", 64))

	if _, err := st.Index(context.Background()); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("error = %v, want sha256 mismatch", err)
	}
}

func TestIndexRejectsWrongRelease(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, nil)
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	st := testStore(t, t.TempDir(), srv.URL, testPins(commit), digest(rel.indexBytes))
	st.release = "content-other"

	if _, err := st.Index(context.Background()); err == nil {
		t.Fatal("index accepted a different release")
	}
}

func TestCourseRejectsCommitDrift(t *testing.T) {
	pin := strings.Repeat("a", 40)
	rel := buildRelease(t, pin, strings.Repeat("b", 40), nil)
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	cacheDir := t.TempDir()
	st := testStore(t, cacheDir, srv.URL, testPins(pin), digest(rel.indexBytes))

	_, err := st.Course(context.Background(), "grep", nil)
	if err == nil || !strings.Contains(err.Error(), "spec commit") {
		t.Fatalf("error = %v, want spec commit mismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(cacheDir, "content", testRelease, "grep")); !os.IsNotExist(statErr) {
		t.Fatalf("failed bundle was left on disk: %v", statErr)
	}
}

func TestCourseRejectsBadFileDigest(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, func(m *Manifest) {
		m.Files["LICENSE"] = strings.Repeat("0", 64)
	})
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	st := testStore(t, t.TempDir(), srv.URL, testPins(commit), digest(rel.indexBytes))

	if _, err := st.Course(context.Background(), "grep", nil); err == nil {
		t.Fatal("course accepted a bundle with a bad file digest")
	}
}

func TestCourseRejectsUnknownSlug(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, nil)
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	st := testStore(t, t.TempDir(), srv.URL, testPins(commit), digest(rel.indexBytes))

	if _, err := st.Course(context.Background(), "../../etc", nil); err == nil {
		t.Fatal("course accepted an unknown slug")
	}
}

func TestCourseWorksFromCacheOffline(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, nil)
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	cacheDir := t.TempDir()
	first := testStore(t, cacheDir, srv.URL, testPins(commit), digest(rel.indexBytes))
	if _, err := first.Course(context.Background(), "grep", nil); err != nil {
		t.Fatalf("first course: %v", err)
	}
	srv.Close()

	second := testStore(t, cacheDir, srv.URL, testPins(commit), digest(rel.indexBytes))
	if _, err := second.Index(context.Background()); err != nil {
		t.Fatalf("cached index: %v", err)
	}
	if _, err := second.Course(context.Background(), "grep", nil); err != nil {
		t.Fatalf("cached course: %v", err)
	}
}

func TestManifestRejectsUnsafePath(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel := buildRelease(t, commit, commit, func(m *Manifest) {
		m.Files["../escape"] = strings.Repeat("0", 64)
	})
	srv := testServer(t, rel.indexBytes, rel.bundle, rel.asset, nil, nil)

	st := testStore(t, t.TempDir(), srv.URL, testPins(commit), digest(rel.indexBytes))

	if _, err := st.Course(context.Background(), "grep", nil); err == nil {
		t.Fatal("course accepted a manifest with a traversing path")
	}
}
