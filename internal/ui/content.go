package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/aadhii-yz/quest/internal/provider"
)

// ContentLoader fetches and verifies the content bundle for one course and
// returns providers rebuilt from the updated content. It is implemented in
// cmd/quest; the UI never downloads anything itself.
type ContentLoader interface {
	Load(ctx context.Context, slug string) ([]provider.Provider, error)
}

type contentLoadedMsg struct {
	slug      string
	providers []provider.Provider
	err       error
}

// loadContentCmd runs a loader off the update loop and delivers its result as
// a message.
func loadContentCmd(loader ContentLoader, slug string) tea.Cmd {
	return func() tea.Msg {
		providers, err := loader.Load(context.Background(), slug)
		return contentLoadedMsg{slug: slug, providers: providers, err: err}
	}
}

// findCourse returns the course with the given ID across all providers.
func findCourse(providers []provider.Provider, id string) (provider.Course, bool) {
	for _, p := range providers {
		for _, c := range p.Courses() {
			if c.ID == id {
				return c, true
			}
		}
	}
	return provider.Course{}, false
}
