package cache

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type tarEntry struct {
	name     string
	body     string
	mode     int64
	typeflag byte
	linkname string
}

func writeTarGz(t *testing.T, entries ...tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		if e.typeflag == 0 {
			e.typeflag = tar.TypeReg
		}
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     mode,
			Size:     int64(len(e.body)),
			Typeflag: e.typeflag,
			Linkname: e.linkname,
		}
		if e.typeflag == tar.TypeDir {
			hdr.Size = 0
			if hdr.Mode == 0 {
				hdr.Mode = 0o755
			}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if hdr.Size > 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write body %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractTarGz(t *testing.T) {
	archive := writeTarGz(t,
		tarEntry{name: "docs/", typeflag: tar.TypeDir},
		tarEntry{name: "docs/readme.md", body: "hello"},
		tarEntry{name: "bin/tool", body: "#!/bin/sh\n", mode: 0o755},
	)

	dest := filepath.Join(t.TempDir(), "bundle")
	if err := ExtractTarGz(context.Background(), archive, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(dest, "docs", "readme.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("readme = %q", body)
	}

	info, err := os.Stat(filepath.Join(dest, "bin", "tool"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Fatalf("executable mode = %o, want 700", info.Mode().Perm())
	}
}

func TestSafeArchivePathRoot(t *testing.T) {
	name, err := safeArchivePath(".")
	if err != nil || name != "" {
		t.Fatalf("safeArchivePath(.) = %q, %v; want empty, nil", name, err)
	}
}

func TestExtractTarGzRejectsUnsafeEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []tarEntry
	}{
		{name: "parent traversal", entries: []tarEntry{{name: "../evil", body: "x"}}},
		{name: "nested traversal", entries: []tarEntry{{name: "a/../../evil", body: "x"}}},
		{name: "absolute path", entries: []tarEntry{{name: "/evil", body: "x"}}},
		{name: "backslash", entries: []tarEntry{{name: `a\b`, body: "x"}}},
		{name: "symlink", entries: []tarEntry{{name: "link", typeflag: tar.TypeSymlink, linkname: "../evil"}}},
		{name: "hardlink", entries: []tarEntry{{name: "link", typeflag: tar.TypeLink, linkname: "target"}}},
		{name: "duplicate file", entries: []tarEntry{{name: "same", body: "a"}, {name: "same", body: "b"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := writeTarGz(t, tt.entries...)
			parent := t.TempDir()
			dest := filepath.Join(parent, "out")
			err := ExtractTarGz(context.Background(), archive, dest)
			if err == nil {
				t.Fatalf("extract succeeded for %s", tt.name)
			}
			if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
				t.Fatalf("destination exists after failed extraction: %v", statErr)
			}
			entries, readErr := os.ReadDir(parent)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("temporary extraction left behind: %v", entries[0].Name())
			}
		})
	}
}

func TestExtractTarGzSizeCap(t *testing.T) {
	archive := writeTarGz(t, tarEntry{name: "big", body: strings.Repeat("x", 32)})
	dest := filepath.Join(t.TempDir(), "out")
	if err := extractTarGz(context.Background(), archive, dest, 16); err == nil {
		t.Fatal("extract succeeded past the size cap")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("destination exists after failed extraction: %v", err)
	}
}

func TestExtractTarGzDestinationExists(t *testing.T) {
	archive := writeTarGz(t, tarEntry{name: "file", body: "x"})
	dest := filepath.Join(t.TempDir(), "out")
	if err := os.Mkdir(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ExtractTarGz(context.Background(), archive, dest); err == nil {
		t.Fatal("extract overwrote an existing destination")
	}
}

func TestExtractTarGzCanceled(t *testing.T) {
	archive := writeTarGz(t, tarEntry{name: "file", body: "x"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := filepath.Join(t.TempDir(), "out")
	if err := ExtractTarGz(ctx, archive, dest); err == nil {
		t.Fatal("extract succeeded with a canceled context")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("destination exists after canceled extraction: %v", err)
	}
}
