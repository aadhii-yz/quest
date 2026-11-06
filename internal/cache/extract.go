package cache

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// MaxExtractBytes caps the total uncompressed size of one archive. Tester
// assets embed toolchains, so this is generous; content bundles are far below
// it.
const MaxExtractBytes int64 = 2 << 30

// ExtractTarGz extracts archive into dest, which must not exist yet.
//
// Extraction fails closed: absolute paths, parent traversal, backslashes,
// links, duplicate file entries, non-regular entries and archives larger than
// MaxExtractBytes are rejected. The archive is unpacked into a temporary
// directory next to dest and renamed only on success, so a failed extraction
// leaves nothing behind. Directories are created 0700 and files 0600, or 0700
// when the archive marks them executable.
func ExtractTarGz(ctx context.Context, archive, dest string) error {
	return extractTarGz(ctx, archive, dest, MaxExtractBytes)
}

func extractTarGz(ctx context.Context, archive, dest string, maxBytes int64) error {
	if archive == "" || dest == "" {
		return errors.New("cache: empty archive or destination")
	}
	if maxBytes <= 0 {
		return errors.New("cache: invalid extraction cap")
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("cache: extract destination exists: %s", dest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cache: stat %s: %w", dest, err)
	}

	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, dirPerm); err != nil {
		return fmt.Errorf("cache: create %s: %w", parent, err)
	}
	tmp, err := os.MkdirTemp(parent, "."+filepath.Base(dest)+".tmp-*")
	if err != nil {
		return fmt.Errorf("cache: extract temp dir: %w", err)
	}
	done := false
	defer func() {
		if !done {
			os.RemoveAll(tmp)
		}
	}()

	if err := unpackTarGz(ctx, archive, tmp, maxBytes); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("cache: extract to %s: %w", dest, err)
	}
	done = true
	return nil
}

func unpackTarGz(ctx context.Context, archive, dir string, maxBytes int64) error {
	f, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("cache: open %s: %w", archive, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("cache: read %s: %w", archive, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	seen := make(map[string]struct{})
	var total int64

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cache: read %s: %w", archive, err)
		}

		name, err := safeArchivePath(hdr.Name)
		if err != nil {
			return err
		}
		if name == "" {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(dir, name), dirPerm); err != nil {
				return fmt.Errorf("cache: create %q: %w", name, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if _, dup := seen[name]; dup {
				return fmt.Errorf("cache: duplicate archive entry %q", name)
			}
			seen[name] = struct{}{}
			if hdr.Size < 0 || hdr.Size > maxBytes-total {
				return fmt.Errorf("cache: archive exceeds %d bytes", maxBytes)
			}
			total += hdr.Size

			target := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
				return fmt.Errorf("cache: create parent of %q: %w", name, err)
			}
			mode := os.FileMode(0o600)
			if hdr.Mode&0o111 != 0 {
				mode = 0o700
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return fmt.Errorf("cache: create %q: %w", name, err)
			}
			_, copyErr := io.CopyN(out, tr, hdr.Size)
			closeErr := out.Close()
			if copyErr != nil {
				return fmt.Errorf("cache: extract %q: %w", name, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("cache: write %q: %w", name, closeErr)
			}
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			// PAX metadata is consumed by archive/tar where it matters.
			continue
		default:
			return fmt.Errorf("cache: unsupported archive entry %q (type %d)", hdr.Name, hdr.Typeflag)
		}
	}
}

// safeArchivePath validates one tar entry name. It returns "" for the archive
// root, which callers skip.
func safeArchivePath(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, '\\') {
		return "", fmt.Errorf("cache: unsafe archive path %q", name)
	}
	if strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return "", fmt.Errorf("cache: absolute archive path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("cache: archive path escapes destination %q", name)
	}
	native := filepath.FromSlash(clean)
	if !filepath.IsLocal(native) {
		return "", fmt.Errorf("cache: archive path escapes destination %q", name)
	}
	return native, nil
}
