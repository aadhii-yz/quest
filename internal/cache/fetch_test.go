package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aadhii-yz/quest/internal/provider"
)

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func assetServer(t *testing.T, payload []byte, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.Write(payload)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchStoresVerifiedAsset(t *testing.T) {
	payload := []byte("prebuilt tester binary")
	srv := assetServer(t, payload, nil)
	store := NewStore(t.TempDir())
	store.Client = srv.Client()

	art := provider.Artifact{
		URL:      srv.URL + "/shell-tester_133_linux_amd64.tar.gz",
		SHA256:   digest(payload),
		Filename: "shell-tester_133_linux_amd64.tar.gz",
		Size:     int64(len(payload)),
	}

	type sample struct{ downloaded, total int64 }
	var samples []sample
	path, err := store.Fetch(context.Background(), art, func(downloaded, total int64) {
		samples = append(samples, sample{downloaded, total})
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if want := filepath.Join(store.Dir, art.Filename); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("content = %q, want %q", got, payload)
	}
	if len(samples) == 0 {
		t.Fatal("progress was never reported")
	}
	if first := samples[0]; first.downloaded != 0 || first.total != int64(len(payload)) {
		t.Fatalf("first progress = %+v, want 0/%d", first, len(payload))
	}
	if last := samples[len(samples)-1]; last.downloaded != int64(len(payload)) {
		t.Fatalf("last progress = %+v, want %d downloaded", last, len(payload))
	}
}

func TestFetchRejectsDigestMismatch(t *testing.T) {
	payload := []byte("asset")
	srv := assetServer(t, payload, nil)
	store := NewStore(t.TempDir())
	store.Client = srv.Client()

	art := provider.Artifact{
		URL:      srv.URL + "/asset.tar.gz",
		SHA256:   strings.Repeat("0", 64),
		Filename: "asset.tar.gz",
	}
	_, err := store.Fetch(context.Background(), art, nil)
	if err == nil {
		t.Fatal("fetch succeeded with a wrong digest")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("error = %v, want sha256 mismatch", err)
	}
	assertEmptyDir(t, store.Dir)
}

func TestFetchReusesVerifiedFile(t *testing.T) {
	payload := []byte("cached")
	var hits atomic.Int32
	srv := assetServer(t, payload, &hits)
	store := NewStore(t.TempDir())
	store.Client = srv.Client()

	dest := filepath.Join(store.Dir, "asset.tar.gz")
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	art := provider.Artifact{URL: srv.URL + "/asset.tar.gz", SHA256: digest(payload), Filename: "asset.tar.gz"}
	path, err := store.Fetch(context.Background(), art, nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if path != dest {
		t.Fatalf("path = %q, want %q", path, dest)
	}
	if hits.Load() != 0 {
		t.Fatalf("server hits = %d, want 0 for a verified cache hit", hits.Load())
	}
}

func TestFetchReplacesCorruptFile(t *testing.T) {
	payload := []byte("good bytes")
	var hits atomic.Int32
	srv := assetServer(t, payload, &hits)
	store := NewStore(t.TempDir())
	store.Client = srv.Client()

	dest := filepath.Join(store.Dir, "asset.tar.gz")
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}

	art := provider.Artifact{URL: srv.URL + "/asset.tar.gz", SHA256: digest(payload), Filename: "asset.tar.gz"}
	if _, err := store.Fetch(context.Background(), art, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("server hits = %d, want 1", hits.Load())
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("content = %q, want %q", got, payload)
	}
}

func TestFetchEnforcesPinnedSize(t *testing.T) {
	payload := []byte("asset")
	srv := assetServer(t, payload, nil)
	store := NewStore(t.TempDir())
	store.Client = srv.Client()

	art := provider.Artifact{
		URL:      srv.URL + "/asset.tar.gz",
		SHA256:   digest(payload),
		Filename: "asset.tar.gz",
		Size:     int64(len(payload) + 10),
	}
	_, err := store.Fetch(context.Background(), art, nil)
	if err == nil {
		t.Fatal("fetch succeeded with the wrong pinned size")
	}
	if !strings.Contains(err.Error(), "want") {
		t.Fatalf("error = %v, want a size mismatch", err)
	}
	assertEmptyDir(t, store.Dir)
}

func TestFetchRejectsNonHTTPURL(t *testing.T) {
	store := NewStore(t.TempDir())
	art := provider.Artifact{URL: "file:///etc/passwd", SHA256: strings.Repeat("a", 64), Filename: "asset"}
	if _, err := store.Fetch(context.Background(), art, nil); err == nil {
		t.Fatal("fetch accepted a non-http url")
	}
}

func TestFetchCanceledContext(t *testing.T) {
	payload := []byte("asset")
	srv := assetServer(t, payload, nil)
	store := NewStore(t.TempDir())
	store.Client = srv.Client()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	art := provider.Artifact{URL: srv.URL + "/asset.tar.gz", SHA256: digest(payload), Filename: "asset.tar.gz"}
	if _, err := store.Fetch(ctx, art, nil); err == nil {
		t.Fatal("fetch succeeded with a canceled context")
	}
	if _, err := os.Stat(filepath.Join(store.Dir, art.Filename)); !os.IsNotExist(err) {
		t.Fatalf("destination exists after canceled fetch: %v", err)
	}
}

func TestFetchRejectsInvalidFilename(t *testing.T) {
	store := NewStore(t.TempDir())
	for _, name := range []string{"", "../escape", "a/b", "."} {
		art := provider.Artifact{URL: "https://example.com/a", SHA256: strings.Repeat("a", 64), Filename: name}
		if _, err := store.Fetch(context.Background(), art, nil); err == nil {
			t.Fatalf("fetch accepted filename %q", name)
		}
	}
}

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("cache dir is not empty: %v", names)
	}
}
