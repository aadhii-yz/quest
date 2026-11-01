package config

import (
	"path/filepath"
	"testing"
)

func TestResolveOverrides(t *testing.T) {
	t.Setenv(EnvCacheDir, "/tmp/quest-cache")
	t.Setenv(EnvDefsDir, "/tmp/quest-defs")
	paths := Resolve()
	if paths.CacheDir != "/tmp/quest-cache" {
		t.Fatalf("cache dir = %q", paths.CacheDir)
	}
	if paths.DefsDir != "/tmp/quest-defs" {
		t.Fatalf("defs dir = %q", paths.DefsDir)
	}
}

func TestResolveDefaults(t *testing.T) {
	t.Setenv(EnvCacheDir, "")
	t.Setenv(EnvDefsDir, "")
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg")
	paths := Resolve()
	if paths.DefsDir != "temp" {
		t.Fatalf("defs dir = %q", paths.DefsDir)
	}
	if want := filepath.Join("/tmp/xdg", "quest"); paths.CacheDir != want {
		t.Fatalf("cache dir = %q, want %q", paths.CacheDir, want)
	}
}
