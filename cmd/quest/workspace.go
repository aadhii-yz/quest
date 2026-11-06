package main

import (
	"context"

	"github.com/aadhii-yz/quest/internal/content"
	"github.com/aadhii-yz/quest/internal/workspace"
)

// projectStore implements ui.Workspace using verified content bundles.
type projectStore struct {
	store *content.Store
}

func (p *projectStore) Create(ctx context.Context, dir, courseID, language, stageID string) error {
	bundle, err := p.store.Course(ctx, courseID, nil)
	if err != nil {
		return err
	}
	return workspace.Create(workspace.CreateOptions{
		Dir:       dir,
		BundleDir: bundle.Dir,
		Provider:  content.Provider,
		Course:    courseID,
		Language:  language,
		Stage:     stageID,
	})
}

func (p *projectStore) Save(_ context.Context, dir, courseID, language, stageID string) error {
	return workspace.Save(dir, workspace.State{
		Provider: content.Provider,
		Course:   courseID,
		Language: language,
		Stage:    stageID,
	})
}
