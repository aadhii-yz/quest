package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// Workspace creates starter directories from verified content and persists the
// current course, language and stage. It is implemented in cmd/quest; the UI
// never writes project files by itself.
type Workspace interface {
	Create(ctx context.Context, dir, courseID, language, stageID string) error
	Save(ctx context.Context, dir, courseID, language, stageID string) error
}

type workspaceMsg struct {
	dir     string
	created bool
	err     error
}

func createWorkspaceCmd(w Workspace, dir, courseID, language, stageID string) tea.Cmd {
	return func() tea.Msg {
		err := w.Create(context.Background(), dir, courseID, language, stageID)
		return workspaceMsg{dir: dir, created: true, err: err}
	}
}

func saveWorkspaceCmd(w Workspace, dir, courseID, language, stageID string) tea.Cmd {
	return func() tea.Msg {
		err := w.Save(context.Background(), dir, courseID, language, stageID)
		return workspaceMsg{dir: dir, err: err}
	}
}
