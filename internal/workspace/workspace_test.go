package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testState() State {
	return State{Provider: "codecrafters", Course: "grep", Language: "go", Stage: "cq2"}
}

func TestSaveAndDetect(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, testState()); err != nil {
		t.Fatalf("save: %v", err)
	}

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	found, st, err := Detect(nested)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if found != root {
		t.Fatalf("root = %q, want %q", found, root)
	}
	if st.Format != Format || st.Course != "grep" || st.Stage != "cq2" {
		t.Fatalf("state = %+v", st)
	}

	data, err := os.ReadFile(filepath.Join(root, DirName, StateFile))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk State
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("state file is not json: %v", err)
	}
	if onDisk != st {
		t.Fatalf("on disk = %+v, want %+v", onDisk, st)
	}
}

func TestDetectNotFound(t *testing.T) {
	_, _, err := Detect(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestDetectRejectsMalformedState(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, StateFile), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Detect(root); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error = %v, want parse failure", err)
	}
}

func TestDetectRejectsUnknownFormat(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"format":2,"provider":"codecrafters","course":"grep"}`)
	if err := os.WriteFile(filepath.Join(dir, StateFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Detect(root); err == nil || !strings.Contains(err.Error(), "format") {
		t.Fatalf("error = %v, want format failure", err)
	}
}

func TestSaveRequiresCourse(t *testing.T) {
	if err := Save(t.TempDir(), State{Provider: "codecrafters"}); err == nil {
		t.Fatal("save accepted a state without a course")
	}
}

func writeStarter(t *testing.T, bundle string) {
	t.Helper()
	files := map[string]struct {
		body string
		mode os.FileMode
	}{
		"starter_templates/all/code/README.md":            {"starter\n", 0o644},
		"starter_templates/all/code/codecrafters.yml":     {"debug: false\n", 0o644},
		"starter_templates/all/code/.codecrafters/run.sh": {"#!/bin/sh\n", 0o755},
		"starter_templates/go/code/app/main.go":           {"package main\n", 0o644},
		"starter_templates/go/config.yml":                 {"attributes: {}\n", 0o644},
	}
	for name, f := range files {
		path := filepath.Join(bundle, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(f.body), f.mode); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCreate(t *testing.T) {
	bundle := t.TempDir()
	writeStarter(t, bundle)
	dir := filepath.Join(t.TempDir(), "projects", "grep")

	err := Create(CreateOptions{
		Dir:       dir,
		BundleDir: bundle,
		Provider:  "codecrafters",
		Course:    "grep",
		Language:  "go",
		Stage:     "cq2",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(dir, "app", "main.go"))
	if err != nil {
		t.Fatalf("language overlay missing: %v", err)
	}
	if string(body) != "package main\n" {
		t.Fatalf("main.go = %q", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Fatalf("shared template missing: %v", err)
	}

	run := filepath.Join(dir, ".codecrafters", "run.sh")
	info, err := os.Stat(run)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("run.sh mode = %o, want executable", info.Mode().Perm())
	}

	_, st, err := Detect(dir)
	if err != nil {
		t.Fatalf("detect created project: %v", err)
	}
	if st.Course != "grep" || st.Language != "go" || st.Stage != "cq2" {
		t.Fatalf("state = %+v", st)
	}

	ignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ignore), ".quest/") {
		t.Fatalf(".gitignore = %q", ignore)
	}
}

func TestCreateRefusesNonEmptyDirectory(t *testing.T) {
	bundle := t.TempDir()
	writeStarter(t, bundle)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Create(CreateOptions{Dir: dir, BundleDir: bundle, Provider: "codecrafters", Course: "grep", Language: "go"})
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("error = %v, want not empty", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "keep.txt")); statErr != nil {
		t.Fatalf("existing file was touched: %v", statErr)
	}
}

func TestCreateCleansUpOnFailure(t *testing.T) {
	bundle := t.TempDir() // no starter_templates at all
	parent := t.TempDir()
	dir := filepath.Join(parent, "grep")

	err := Create(CreateOptions{Dir: dir, BundleDir: bundle, Provider: "codecrafters", Course: "grep", Language: "go"})
	if err == nil {
		t.Fatal("create succeeded without starter templates")
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("destination exists after failed create: %v", statErr)
	}
	entries, readErr := os.ReadDir(parent)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary project left behind: %v", entries[0].Name())
	}
}
