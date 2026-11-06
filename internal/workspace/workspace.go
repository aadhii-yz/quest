// Package workspace manages the directory context of a quest project.
//
// A project is a starter directory containing .quest/state.json. Starting
// quest inside one opens its course at the saved stage instead of the catalog;
// detecting a project walks up from the working directory, like lazygit's git
// discovery.
package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DirName is the project marker directory inside a starter directory.
	DirName = ".quest"
	// StateFile is the marker filename inside DirName.
	StateFile = "state.json"
	// Format is the marker format this package understands.
	Format = 1
)

// ErrNotFound means no project marker was found from the starting directory
// upwards.
var ErrNotFound = errors.New("workspace: no project marker found")

// State is the persisted project context. It is small on purpose: the course
// definition, starter files and testers are never copied into it.
type State struct {
	Format   int    `json:"format"`
	Provider string `json:"provider"`
	Course   string `json:"course"`
	Language string `json:"language"`
	Stage    string `json:"stage"`
}

// Detect walks up from start looking for <dir>/.quest/state.json and returns
// the project root and decoded state. A missing marker returns ErrNotFound; a
// malformed or unsupported marker returns an error naming the file.
func Detect(start string) (string, State, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", State{}, fmt.Errorf("workspace: resolve %s: %w", start, err)
	}

	for {
		path := filepath.Join(dir, DirName, StateFile)
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			var st State
			if err := json.Unmarshal(data, &st); err != nil {
				return "", State{}, fmt.Errorf("workspace: parse %s: %w", path, err)
			}
			if err := validate(st); err != nil {
				return "", State{}, fmt.Errorf("workspace: %s: %w", path, err)
			}
			return dir, st, nil
		case !errors.Is(err, os.ErrNotExist):
			return "", State{}, fmt.Errorf("workspace: read %s: %w", path, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", State{}, ErrNotFound
		}
		dir = parent
	}
}

// Save writes the project state atomically with 0600 permissions, creating the
// marker directory if needed.
func Save(root string, st State) error {
	if root == "" {
		return errors.New("workspace: empty project directory")
	}
	st.Format = Format
	if err := validate(st); err != nil {
		return fmt.Errorf("workspace: %w", err)
	}

	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("workspace: create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("workspace: encode state: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return fmt.Errorf("workspace: temp state: %w", err)
	}
	tmpPath := tmp.Name()
	done := false
	defer func() {
		if !done {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("workspace: write state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("workspace: close state: %w", err)
	}
	if err := os.Rename(tmpPath, filepath.Join(dir, StateFile)); err != nil {
		return fmt.Errorf("workspace: save state: %w", err)
	}
	done = true
	return nil
}

func validate(st State) error {
	if st.Format != Format {
		return fmt.Errorf("unsupported format %d, want %d", st.Format, Format)
	}
	if st.Provider == "" || st.Course == "" {
		return errors.New("missing provider or course")
	}
	return nil
}
