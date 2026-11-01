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
	// DefsDir is where local course-definition.yml files are discovered.
	DefsDir string
}

// Resolve determines paths from the environment, falling back to the user
// cache directory and the repository's gitignored temp/ directory.
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
	base, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(".", ".quest-cache")
	}
	return filepath.Join(base, "quest")
}

func resolveDefsDir() string {
	if dir := os.Getenv(EnvDefsDir); dir != "" {
		return dir
	}
	return "temp"
}
