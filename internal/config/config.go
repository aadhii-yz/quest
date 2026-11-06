// Package config resolves user-visible paths used by quest.
//
// Resolution is pure: no directories are created here. Creating the cache
// directory (0700) is the cache store's job, so resolution stays testable and
// side-effect free.
package config

import (
	"os"
	"path/filepath"
)

// Environment variables that override resolved paths.
const (
	EnvCacheDir = "QUEST_CACHE_DIR"
	EnvDefsDir  = "QUEST_DEFS_DIR"
)

// Paths are the filesystem locations quest uses.
type Paths struct {
	// CacheDir holds downloaded tester assets and user state. It is created
	// on demand with 0700 permissions and is safe to delete.
	CacheDir string
	// DefsDir overrides the content cache with local course-definition.yml
	// files for development. Empty means "use verified quest-source content".
	DefsDir string
}

// Resolve determines paths from the environment, falling back to the user
// cache directory. DefsDir is an explicit development override and is empty
// by default: course content comes from the pinned quest-source release.
func Resolve() Paths {
	return Paths{
		CacheDir: resolveCacheDir(),
		DefsDir:  resolveDefsDir(),
	}
}

func resolveCacheDir() string {
	if dir := os.Getenv(EnvCacheDir); dir != "" {
		return dir
	}
	// Honor XDG_CACHE_HOME explicitly so the cache location is the same on
	// every platform when the variable is set; os.UserCacheDir only consults
	// it on Unix.
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "quest")
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(".", ".quest-cache")
	}
	return filepath.Join(base, "quest")
}

func resolveDefsDir() string {
	return os.Getenv(EnvDefsDir)
}
