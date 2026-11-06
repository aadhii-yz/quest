package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/runner"
)

type fakeLoader struct {
	providers []provider.Provider
	err       error
	calls     []string
}

func (l *fakeLoader) Load(_ context.Context, slug string) ([]provider.Provider, error) {
	l.calls = append(l.calls, slug)
	if l.err != nil {
		return nil, l.err
	}
	return l.providers, nil
}

func unreadyCourse() provider.Course {
	c := fakeCourse()
	c.Stages = nil
	c.Note = "no local definition"
	return c
}

func press(t *testing.T, m app, code rune) app {
	t.Helper()
	next, _ := m.Update(tea.KeyPressMsg{Code: code})
	return next.(app)
}

// openCourse moves the catalog selection from resources to courses to
// languages and opens the course, returning the loader command.
func openCourse(t *testing.T, m app) (app, tea.Cmd) {
	t.Helper()
	m = press(t, m, tea.KeyEnter)
	m = press(t, m, tea.KeyEnter)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return next.(app), cmd
}

func TestContentLoaderFillsCourse(t *testing.T) {
	loader := &fakeLoader{providers: []provider.Provider{
		fakeProvider{courses: []provider.Course{fakeCourse()}},
	}}
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{unreadyCourse()}}},
		runner.Simulated{},
		WithContentLoader(loader),
	)
	m = setSize(m, 100, 30)

	m, cmd := openCourse(t, m)
	if cmd == nil {
		t.Fatal("opening a course did not request content")
	}
	if m.view != viewCourse || m.course == nil {
		t.Fatal("course screen did not open")
	}
	if len(loader.calls) != 0 {
		t.Fatal("loader ran on the update loop instead of a command")
	}

	next, _ := m.Update(cmd())
	m = next.(app)

	if len(loader.calls) != 1 || loader.calls[0] != "grep" {
		t.Fatalf("loader calls = %v", loader.calls)
	}
	if !m.course.course.Ready() {
		t.Fatal("loaded course has no stages")
	}
	if got := len(m.course.course.Stages); got != 30 {
		t.Fatalf("stages = %d, want 30", got)
	}
	if joined := strings.Join(m.course.logLines, "\n"); !strings.Contains(joined, "content loaded") {
		t.Fatalf("log = %v", m.course.logLines)
	}
	if c, ok := m.cat.course(); !ok || c.ID != "grep" {
		t.Fatalf("catalog lost the selected course: %+v ok=%t", c, ok)
	}
	assertFrame(t, m, "loaded course")
}

func TestContentLoaderErrorIsLogged(t *testing.T) {
	loader := &fakeLoader{err: errors.New("network down")}
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{unreadyCourse()}}},
		runner.Simulated{},
		WithContentLoader(loader),
	)
	m = setSize(m, 100, 30)

	m, cmd := openCourse(t, m)
	if cmd == nil {
		t.Fatal("opening a course did not request content")
	}
	next, _ := m.Update(cmd())
	m = next.(app)

	if m.course.course.Ready() {
		t.Fatal("course became ready after a loader error")
	}
	if m.course.loading {
		t.Fatal("loading state was not cleared")
	}
	if joined := strings.Join(m.course.logLines, "\n"); !strings.Contains(joined, "network down") {
		t.Fatalf("log = %v", m.course.logLines)
	}
	assertFrame(t, m, "content error")
}

func TestNoLoaderOpensImmediately(t *testing.T) {
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{fakeCourse()}}},
		runner.Simulated{},
	)
	m = setSize(m, 100, 30)
	m, cmd := openCourse(t, m)
	if cmd != nil {
		t.Fatal("unexpected command without a content loader")
	}
	if m.view != viewCourse || !m.course.course.Ready() {
		t.Fatal("course did not open from existing definitions")
	}
}
