package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateOptions describes a starter directory to create from a verified
// content bundle.
type CreateOptions struct {
	// Dir is the destination directory. A leading ~ is expanded. It must not
	// already contain files.
	Dir string
	// BundleDir is the extracted course bundle that provides starter_templates.
	BundleDir string
	// Provider, Course, Language and Stage are written to the marker.
	Provider string
	Course   string
	Language string
	Stage    string
}

// Create builds a starter directory atomically: it copies the shared
// starter_templates/all/code tree and overlays starter_templates/<language>/code,
// writes .quest/state.json and appends .quest/ to .gitignore. Nothing is left
// behind on failure and an existing non-empty destination is refused.
func Create(opts CreateOptions) error {
	dir, err := expandHome(opts.Dir)
	if err != nil {
		return err
	}
	if dir == "" {
		return errors.New("workspace: empty destination")
	}
	if opts.BundleDir == "" {
		return errors.New("workspace: empty bundle directory")
	}
	if opts.Language == "" {
		return errors.New("workspace: empty language")
	}

	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("workspace: %s is not a directory", dir)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("workspace: read %s: %w", dir, err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("workspace: %s is not empty", dir)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("workspace: stat %s: %w", dir, err)
	}

	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("workspace: create %s: %w", parent, err)
	}
	tmp, err := os.MkdirTemp(parent, "."+filepath.Base(dir)+".tmp-*")
	if err != nil {
		return fmt.Errorf("workspace: temp project: %w", err)
	}
	done := false
	defer func() {
		if !done {
			os.RemoveAll(tmp)
		}
	}()

	shared := filepath.Join(opts.BundleDir, "starter_templates", "all", "code")
	if err := copyTree(shared, tmp); err != nil {
		return err
	}
	language := filepath.Join(opts.BundleDir, "starter_templates", opts.Language, "code")
	if _, err := os.Stat(language); err == nil {
		if err := copyTree(language, tmp); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("workspace: stat %s: %w", language, err)
	}

	st := State{
		Provider: opts.Provider,
		Course:   opts.Course,
		Language: opts.Language,
		Stage:    opts.Stage,
	}
	if err := Save(tmp, st); err != nil {
		return err
	}
	if err := ignoreMarker(filepath.Join(tmp, ".gitignore")); err != nil {
		return err
	}

	if err := os.Rename(tmp, dir); err != nil {
		return fmt.Errorf("workspace: create %s: %w", dir, err)
	}
	done = true
	return nil
}

func expandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("workspace: home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path), nil
}

// copyTree copies regular files and directories, creating directories 0700 and
// preserving file permissions. Links and other non-regular entries are
// rejected: a verified bundle never needs them.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return fmt.Errorf("workspace: create %s: %w", target, err)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("workspace: refusing non-regular starter file %s", path)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("workspace: create %s: %w", target, err)
		}
		in, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("workspace: open %s: %w", path, err)
		}
		defer in.Close()

		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return fmt.Errorf("workspace: create %s: %w", target, err)
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return fmt.Errorf("workspace: copy %s: %w", path, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("workspace: write %s: %w", target, closeErr)
		}
		return nil
	})
}

// ignoreMarker appends .quest/ to a project's .gitignore unless it is already
// listed.
func ignoreMarker(path string) error {
	const line = DirName + "/"
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("workspace: read %s: %w", path, err)
	}
	for _, existing := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(existing) == line {
			return nil
		}
	}

	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, []byte(line+"\n")...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("workspace: write %s: %w", path, err)
	}
	return nil
}
