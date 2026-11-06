package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/runner"
)

type fakeWorkspace struct {
	created []string
	saves   []string
	err     error
}

func (w *fakeWorkspace) Create(_ context.Context, dir, courseID, language, stageID string) error {
	w.created = append(w.created, dir+"|"+courseID+"|"+language+"|"+stageID)
	return w.err
}

func (w *fakeWorkspace) Save(_ context.Context, dir, courseID, language, stageID string) error {
	w.saves = append(w.saves, dir+"|"+courseID+"|"+language+"|"+stageID)
	return w.err
}

func TestStartOpensCourseAtSavedStage(t *testing.T) {
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{fakeCourse()}}},
		runner.Simulated{},
		WithWorkspace(&fakeWorkspace{}),
		WithStart("/tmp/project", "grep", "rust", "s07"),
	)
	if m.view != viewCourse || m.course == nil {
		t.Fatal("start did not open the course")
	}
	stage, ok := m.course.stage()
	if !ok || stage.ID != "s07" {
		t.Fatalf("stage = %+v ok=%t, want s07", stage, ok)
	}
	if got := m.course.language(); got != "rust" {
		t.Fatalf("language = %q, want rust", got)
	}
	if m.workspaceDir != "/tmp/project" {
		t.Fatalf("workspace dir = %q", m.workspaceDir)
	}
}

func TestStartLoadsMissingContentAndKeepsStage(t *testing.T) {
	loader := &fakeLoader{providers: []provider.Provider{
		fakeProvider{courses: []provider.Course{fakeCourse()}},
	}}
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{unreadyCourse()}}},
		runner.Simulated{},
		WithContentLoader(loader),
		WithStart("/tmp/project", "grep", "go", "s07"),
	)
	if m.initCmd == nil {
		t.Fatal("start did not request content for an unready course")
	}
	next, _ := m.Update(m.initCmd())
	m = next.(app)

	stage, ok := m.course.stage()
	if !ok || stage.ID != "s07" {
		t.Fatalf("stage = %+v ok=%t, want s07 after load", stage, ok)
	}
	if !m.course.course.Ready() {
		t.Fatal("course not ready after loading")
	}
}

func TestCreatePromptStartsCreation(t *testing.T) {
	ws := &fakeWorkspace{}
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{fakeCourse()}}},
		runner.Simulated{},
		WithWorkspace(ws),
		WithStart("/tmp/project", "grep", "go", "s01"),
	)
	m = setSize(m, 100, 30)

	m = press(t, m, 'c')
	if m.prompt == nil {
		t.Fatal("create did not open a prompt")
	}
	assertFrame(t, m, "create prompt")

	m.prompt.SetValue("/tmp/newproject")
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(app)
	if cmd == nil {
		t.Fatal("prompt did not start creation")
	}
	if m.prompt != nil {
		t.Fatal("prompt stayed open after submit")
	}
	next, _ = m.Update(cmd())
	m = next.(app)

	if len(ws.created) != 1 || !strings.Contains(ws.created[0], "/tmp/newproject|grep|go|s01") {
		t.Fatalf("create calls = %v", ws.created)
	}
	if m.workspaceDir != "/tmp/newproject" {
		t.Fatalf("workspace dir = %q", m.workspaceDir)
	}
	if joined := strings.Join(m.course.logLines, "\n"); !strings.Contains(joined, "created project") {
		t.Fatalf("log = %v", m.course.logLines)
	}
}

func TestCreatePromptEscapeCancels(t *testing.T) {
	ws := &fakeWorkspace{}
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{fakeCourse()}}},
		runner.Simulated{},
		WithWorkspace(ws),
		WithStart("/tmp/project", "grep", "go", "s01"),
	)
	m = setSize(m, 100, 30)
	m = press(t, m, 'c')
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(app)
	if cmd != nil || m.prompt != nil {
		t.Fatal("escape did not close the prompt")
	}
	if len(ws.created) != 0 {
		t.Fatalf("create ran after cancel: %v", ws.created)
	}
	assertFrame(t, m, "closed prompt")
}

func TestStageChangePersistsState(t *testing.T) {
	ws := &fakeWorkspace{}
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{fakeCourse()}}},
		runner.Simulated{},
		WithWorkspace(ws),
		WithStart("/tmp/project", "grep", "go", "s01"),
	)
	m = setSize(m, 100, 30)

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
	m = next.(app)
	if cmd == nil {
		t.Fatal("stage change did not schedule a state save")
	}
	next, _ = m.Update(cmd())
	m = next.(app)

	if len(ws.saves) != 1 || !strings.Contains(ws.saves[0], "/tmp/project|grep|go|s02") {
		t.Fatalf("save calls = %v", ws.saves)
	}
}

func TestNoWorkspaceDisablesCreateAndSave(t *testing.T) {
	m := newApp(
		[]provider.Provider{fakeProvider{courses: []provider.Course{fakeCourse()}}},
		runner.Simulated{},
		WithStart("/tmp/project", "grep", "go", "s01"),
	)
	m = setSize(m, 100, 30)

	m = press(t, m, 'c')
	if m.prompt != nil {
		t.Fatal("create opened without a workspace")
	}
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
	m = next.(app)
	if cmd != nil {
		t.Fatal("stage change scheduled a save without a workspace")
	}
}
