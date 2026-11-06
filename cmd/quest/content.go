package main

import (
	"context"

	"github.com/aadhii-yz/quest/internal/content"
	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/provider/codecrafters"
)

// contentLoader adapts the verified content store to ui.ContentLoader.
type contentLoader struct {
	store *content.Store
	build func() []provider.Provider
}

func (l *contentLoader) Load(ctx context.Context, slug string) ([]provider.Provider, error) {
	if _, err := l.store.Course(ctx, slug, nil); err != nil {
		return nil, err
	}
	return l.build(), nil
}

// contentPins maps every pinned course slug to the spec commit and license the
// content bundle must match.
func contentPins() map[string]content.Pin {
	pins := make(map[string]content.Pin, len(codecrafters.Sources))
	for _, src := range codecrafters.Sources {
		pins[src.Slug()] = content.Pin{
			Commit:  src.RepoCommit,
			License: string(src.RepoLicense),
		}
	}
	return pins
}
