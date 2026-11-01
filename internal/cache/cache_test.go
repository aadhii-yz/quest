package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const helloDigest = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" // sha256("hello")

func TestEnsurePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "assets")
	s := NewStore(dir)
	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Fatalf("permissions = %o, want 700", perm)
		}
	}
}

func TestPathRejectsTraversal(t *testing.T) {
	s := NewStore(t.TempDir())
	for _, name := range []string{"", ".", "..", "../escape", "dir/asset.tar.gz"} {
		if _, err := s.Path(name); err == nil {
			t.Errorf("Path(%q) accepted an invalid name", name)
		}
	}
	if _, err := s.Path("tester.tar.gz"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
}

func TestVerifyReader(t *testing.T) {
	if err := VerifyReader(strings.NewReader("hello"), helloDigest); err != nil {
		t.Fatalf("valid digest rejected: %v", err)
	}
	if err := VerifyReader(strings.NewReader("hello"), strings.ToUpper(helloDigest)); err != nil {
		t.Fatalf("uppercase digest should be accepted: %v", err)
	}
	if err := VerifyReader(strings.NewReader("tampered"), helloDigest); err == nil {
		t.Fatal("mismatch must fail closed")
	}
	for _, bad := range []string{"", "deadbeef", strings.Repeat("z", 64)} {
		if err := VerifyReader(strings.NewReader("hello"), bad); err == nil {
			t.Fatalf("malformed digest %q accepted", bad)
		}
	}
}

func TestVerifyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile(path, helloDigest); err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
	if err := VerifyFile(path, strings.Repeat("a", 64)); err == nil {
		t.Fatal("mismatched file accepted")
	}
	if err := VerifyFile(filepath.Join(t.TempDir(), "missing"), helloDigest); err == nil {
		t.Fatal("missing file accepted")
	}
}
