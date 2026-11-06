package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aadhii-yz/quest/internal/provider"
)

// maxDownloadBytes is a sanity cap for assets with no pinned size. Pinned
// sizes are enforced exactly instead.
const maxDownloadBytes int64 = 4 << 30

// ProgressFunc receives download progress. The first call reports the total
// size (or -1 when the server does not publish one) before the body is read.
// Later calls are throttled, and the final call has downloaded equal to the
// bytes stored.
type ProgressFunc func(downloaded, total int64)

// Fetch downloads art into the store, verifies its SHA256 and returns the
// stored path. A cached file that already verifies is returned without any
// network access. A cached file that fails verification is removed and
// fetched again; a freshly downloaded file that fails verification is
// discarded and reported as an error. There is no unverified result.
func (s *Store) Fetch(ctx context.Context, art provider.Artifact, progress ProgressFunc) (string, error) {
	if s.Dir == "" {
		return "", errors.New("cache: empty directory")
	}
	dest, err := s.Path(art.Filename)
	if err != nil {
		return "", err
	}
	want, err := validateDigest(art.SHA256)
	if err != nil {
		return "", err
	}
	if err := validateURL(art.URL); err != nil {
		return "", err
	}

	switch err := VerifyFile(dest, want); {
	case err == nil:
		return dest, nil
	case !errors.Is(err, os.ErrNotExist):
		// Corrupt or unreadable: never trust it, replace it.
		if rmErr := os.Remove(dest); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return "", fmt.Errorf("cache: remove corrupt %s: %w", dest, rmErr)
		}
	}

	if err := s.Ensure(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, art.URL, nil)
	if err != nil {
		return "", fmt.Errorf("cache: request %s: %w", art.Filename, err)
	}
	req.Header.Set("User-Agent", "quest-cache/1")
	// Ask for the file as stored so Content-Length and the digest describe the
	// same bytes as the published checksum.
	req.Header.Set("Accept-Encoding", "identity")

	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cache: fetch %s: %w", art.Filename, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cache: fetch %s: http %s", art.Filename, resp.Status)
	}

	total := resp.ContentLength
	if art.Size > 0 {
		if total > 0 && total != art.Size {
			return "", fmt.Errorf("cache: %s is %d bytes, want %d", art.Filename, total, art.Size)
		}
		total = art.Size
	}
	limit := maxDownloadBytes
	if total > 0 {
		if total > maxDownloadBytes {
			return "", fmt.Errorf("cache: %s is %d bytes, over the %d byte cap", art.Filename, total, maxDownloadBytes)
		}
		limit = total
	}
	if progress != nil {
		progress(0, total)
	}

	tmp, err := os.CreateTemp(s.Dir, "."+art.Filename+".part-*")
	if err != nil {
		return "", fmt.Errorf("cache: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	stored := false
	defer func() {
		if !stored {
			os.Remove(tmpPath)
		}
	}()

	h := sha256.New()
	pw := &progressWriter{progress: progress, total: total}
	written, copyErr := io.Copy(io.MultiWriter(tmp, h, pw), io.LimitReader(resp.Body, limit+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return "", fmt.Errorf("cache: download %s: %w", art.Filename, copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("cache: write %s: %w", art.Filename, closeErr)
	}
	if written > limit {
		return "", fmt.Errorf("cache: %s exceeds the expected %d bytes", art.Filename, limit)
	}
	if art.Size > 0 && written != art.Size {
		return "", fmt.Errorf("cache: %s is %d bytes, want %d", art.Filename, written, art.Size)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return "", fmt.Errorf("cache: sha256 mismatch for %s: got %s want %s", art.Filename, got, want)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return "", fmt.Errorf("cache: store %s: %w", dest, err)
	}
	stored = true
	if progress != nil {
		progress(written, total)
	}
	return dest, nil
}

// validateURL accepts only absolute http(s) URLs, so a provider mistake cannot
// turn into a local file read.
func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("cache: invalid artifact url %q: %w", raw, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("cache: invalid artifact url %q", raw)
	}
	return nil
}

// progressWriter reports progress at most every 100ms so callers do not have
// to throttle a fast local server.
type progressWriter struct {
	progress     ProgressFunc
	total, done  int64
	lastReported time.Time
	reportEvery  time.Duration
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if p.progress == nil {
		return len(b), nil
	}
	every := p.reportEvery
	if every == 0 {
		every = 100 * time.Millisecond
	}
	if time.Since(p.lastReported) >= every {
		p.progress(p.done, p.total)
		p.lastReported = time.Now()
	}
	return len(b), nil
}
