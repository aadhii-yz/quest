package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/runner"
)

type fakeProvider struct {
	courses []provider.Course
}

func (f fakeProvider) ID() string                 { return "fake" }
func (f fakeProvider) Name() string               { return "Fake" }
func (f fakeProvider) Note() string               { return "fake provider for tests" }
func (f fakeProvider) Planned() bool              { return len(f.courses) == 0 }
func (f fakeProvider) Courses() []provider.Course { return f.courses }

func fakeCourse() provider.Course {
	c := provider.Course{
		ID:           "grep",
		ProviderID:   "fake",
		ProviderName: "Fake",
		Name:         "Grep",
		Summary:      "Build a regex engine and learn about character classes.",
		Languages:    []string{"go", "rust"},
		Tester: provider.TesterPlan{
			Available: true,
			Platform:  "linux_amd64",
			AssetName: "fake_linux_amd64.tar.gz",
			Release:   "v1",
			License:   provider.LicenseNone,
		},
	}
	for i := 1; i <= 30; i++ {
		c.Stages = append(c.Stages, provider.Stage{
			ID:          fmt.Sprintf("s%02d", i),
			Number:      i,
			Name:        fmt.Sprintf("Stage %d", i),
			Difficulty:  []string{"very_easy", "easy", "medium", "hard"}[(i-1)%4],
			Description: fmt.Sprintf("In this stage, implement feature %d.\n\n**Example:**\n\n`x%d` should match.", i, i),
		})
	}
	return c
}

func testApp(helpShown bool) app {
	providers := []provider.Provider{
		fakeProvider{courses: []provider.Course{fakeCourse()}},
		provider.NewPlaceholder("planned", "Planned", "not implemented yet"),
	}
	m := newApp(providers, runner.Simulated{})
	m.help.ShowAll = helpShown
	return m
}

func setSize(m app, w, h int) app {
	m.width, m.height = w, h
	m.layout()
	return m
}

func assertFrame(t *testing.T, m app, label string) {
	t.Helper()
	lines := strings.Split(m.viewString(), "\n")
	if len(lines) != m.height {
		t.Fatalf("%s %dx%d: got %d lines", label, m.width, m.height, len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > m.width {
			t.Fatalf("%s %dx%d: line %d is %d wide: %q", label, m.width, m.height, i, w, line)
		}
	}
}

func TestCatalogFrameInvariants(t *testing.T) {
	for _, helpShown := range []bool{false, true} {
		m := testApp(helpShown)
		for _, w := range []int{20, 30, 40, 50, 63, 64, 80, 100, 140, 200} {
			for _, h := range []int{6, 8, 10, 16, 24, 40} {
				m = setSize(m, w, h)
				assertFrame(t, m, "catalog")
			}
		}
	}
}

func TestCourseFrameInvariants(t *testing.T) {
	m := testApp(false)
	c := fakeCourse()
	for _, w := range []int{20, 30, 45, 49, 50, 64, 100, 160} {
		for _, h := range []int{6, 9, 10, 12, 24, 40} {
			for _, panes := range [][2]bool{{true, true}, {true, false}, {false, true}, {false, false}} {
				cm := newCourseModel(m.theme, c, 0, m.runner)
				cm.showStages, cm.showTests = panes[0], panes[1]
				m.course = cm
				m.view = viewCourse
				m = setSize(m, w, h)
				assertFrame(t, m, "course")
			}
		}
	}

	// Full help must not push the frame out of the window either.
	m = testApp(true)
	m.course = newCourseModel(m.theme, c, 0, m.runner)
	m.view = viewCourse
	for _, size := range [][2]int{{40, 12}, {80, 24}, {120, 40}} {
		m = setSize(m, size[0], size[1])
		assertFrame(t, m, "course help")
	}
}

func TestRunDrivesLog(t *testing.T) {
	m := testApp(false)
	m = setSize(m, 100, 30)
	m.course = newCourseModel(m.theme, fakeCourse(), 0, m.runner)
	m.view = viewCourse
	m.layout()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("run did not start")
	}
	for i := 0; cmd != nil && i < 200; i++ {
		next, nextCmd := m.Update(cmd())
		m = next.(app)
		cmd = nextCmd
	}
	if m.course.run != runDone {
		t.Fatalf("run state = %d, want done", m.course.run)
	}
	joined := strings.Join(m.course.logLines, "\n")
	if !strings.Contains(joined, "PASS") {
		t.Fatalf("log missing result: %v", m.course.logLines)
	}
	if !strings.Contains(joined, "no license") {
		t.Fatal("log should carry the license warning")
	}
}

func TestBackCancelsRun(t *testing.T) {
	m := testApp(false)
	m = setSize(m, 100, 30)
	m.course = newCourseModel(m.theme, fakeCourse(), 0, runner.Simulated{})
	m.view = viewCourse
	m.layout()
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"}); cmd == nil {
		t.Fatal("run did not start")
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(app)
	if m.view != viewCatalog || m.course != nil {
		t.Fatal("escape did not return to the catalog")
	}
}
