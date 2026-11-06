package codecrafters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const grepDefinition = `slug: "grep"
name: "Build your own grep"
short_name: "grep"
release_status: "live"
short_description_md: |-
  Learn **regex** syntax with [the docs](https://example.com/regex).
completion_percentage: 30
languages:
  - slug: "go"
  - slug: "rust"
stages:
  - slug: "cq2"
    name: "Match a literal character"
    difficulty: very_easy
    marketing_md: |-
      In this stage, we'll handle the simplest regex possible.
  - slug: "oq2"
    name: "Match digits"
    difficulty: easy
    primary_extension_slug: "printing-matches"
    marketing_md: |-
      Implement digit matching.
`

func writeDefinition(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, "course", "course-definition.yml")
	if name != "" {
		path = filepath.Join(dir, name, "course-definition.yml")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefinitions(t *testing.T) {
	dir := t.TempDir()
	writeDefinition(t, filepath.Join(dir, "one"), "build-your-own-grep", grepDefinition)
	writeDefinition(t, filepath.Join(dir, "two"), "build-your-own-grep", strings.Replace(grepDefinition, "Match a literal character", "Duplicate", 1))
	// A definition without a slug is ignored.
	writeDefinition(t, filepath.Join(dir, "three"), "broken", "name: no slug\n")

	defs := loadDefinitions(dir)
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
	if _, ok := defs["grep"]; !ok {
		t.Fatal("grep definition not loaded")
	}
}

func TestAdapterCourse(t *testing.T) {
	dir := t.TempDir()
	writeDefinition(t, dir, "build-your-own-grep", grepDefinition)

	a := newAdapter(dir, "linux_amd64")
	var found bool
	for _, c := range a.Courses() {
		if c.ID != "grep" {
			continue
		}
		found = true
		if c.ProviderID != "codecrafters" {
			t.Fatalf("provider id = %q", c.ProviderID)
		}
		if c.Summary != "Learn regex syntax with the docs." {
			t.Fatalf("summary = %q", c.Summary)
		}
		if len(c.Languages) != 2 || c.Languages[0] != "go" || c.Languages[1] != "rust" {
			t.Fatalf("languages = %v", c.Languages)
		}
		if len(c.Stages) != 2 {
			t.Fatalf("stages = %d", len(c.Stages))
		}
		if c.Stages[1].Number != 2 || c.Stages[1].ID != "oq2" {
			t.Fatalf("stage = %+v", c.Stages[1])
		}
		if c.Stages[1].Extension != "printing-matches" {
			t.Fatalf("extension = %q", c.Stages[1].Extension)
		}
		if !c.Ready() {
			t.Fatal("course with stages should be ready")
		}
		if !c.Tester.Available || c.Tester.License == "" {
			t.Fatalf("tester plan = %+v", c.Tester)
		}
	}
	if !found {
		t.Fatal("grep course missing")
	}
}

func TestAdapterCourseWithoutDefinition(t *testing.T) {
	a := newAdapter(t.TempDir(), "linux_amd64")
	for _, c := range a.Courses() {
		if c.ID != "shell" {
			continue
		}
		if c.Ready() {
			t.Fatal("shell should have no stages without a definition")
		}
		if !strings.Contains(c.Note, "course-definition.yml") {
			t.Fatalf("note = %q", c.Note)
		}
		if len(c.Languages) == 0 {
			t.Fatal("languages should fall back to defaults")
		}
		return
	}
	t.Fatal("shell course missing")
}

func TestTesterPlanMissingPlatform(t *testing.T) {
	src := sourceBySlug(t, "grep")
	plan := testerPlan(src, "darwin_arm64") // a known platform gap
	if plan.Available {
		t.Fatal("grep publishes no darwin_arm64 asset")
	}
	if plan.Platform != "darwin_arm64" {
		t.Fatalf("platform = %q", plan.Platform)
	}
	if plan.License == "" {
		t.Fatal("license posture should still be reported")
	}
}

func TestTesterPlanAssetName(t *testing.T) {
	src := sourceBySlug(t, "grep")
	plan := testerPlan(src, "linux_amd64")
	if !plan.Available {
		t.Fatal("grep should publish linux_amd64")
	}
	if !strings.HasSuffix(plan.AssetName, "_linux_amd64.tar.gz") {
		t.Fatalf("asset name = %q", plan.AssetName)
	}
	if plan.Artifact == nil {
		t.Fatal("artifact is nil for an available platform")
	}
	if plan.Artifact.Filename != plan.AssetName {
		t.Fatalf("artifact filename = %q, want %q", plan.Artifact.Filename, plan.AssetName)
	}
	if !strings.HasSuffix(plan.Artifact.URL, plan.AssetName) {
		t.Fatalf("artifact url = %q, want suffix %q", plan.Artifact.URL, plan.AssetName)
	}
	if len(plan.Artifact.SHA256) != 64 {
		t.Fatalf("artifact sha256 = %q", plan.Artifact.SHA256)
	}
	if plan.Release == "" {
		t.Fatal("release should be pinned")
	}
}

func TestSlugFromRepo(t *testing.T) {
	cases := map[string]string{
		"https://github.com/codecrafters-io/build-your-own-grep":   "grep",
		"https://github.com/codecrafters-io/build-your-own-shell/": "shell",
		"https://github.com/example/other-repo":                    "other-repo",
	}
	for in, want := range cases {
		if got := slugFromRepo(in); got != want {
			t.Errorf("slugFromRepo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlainSummary(t *testing.T) {
	got := plain("Learn **regex** with [docs](https://example.com).")
	if got != "Learn regex with docs." {
		t.Fatalf("plain = %q", got)
	}
}

func sourceBySlug(t *testing.T, slug string) *Source {
	t.Helper()
	for i := range Sources {
		if slugFromRepo(Sources[i].RepoURL) == slug {
			return &Sources[i]
		}
	}
	t.Fatalf("source %q not found", slug)
	return nil
}

func TestEmptyDefsDirLoadsNothing(t *testing.T) {
	adapter := newAdapter("", "linux_amd64")
	for _, c := range adapter.Courses() {
		if c.Ready() {
			t.Fatalf("course %q loaded stages from an empty definitions dir", c.ID)
		}
	}
}
