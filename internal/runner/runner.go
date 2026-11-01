// Package runner executes testers and streams their output.
//
// The contract is deliberately small: a Request describes what to run and a
// channel of Events carries plain output lines followed by a done event. Only
// Simulated is implemented today; real execution will fetch a pinned asset
// through internal/cache, verify its SHA256 and stream the process output.
package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/aadhii-yz/quest/internal/licensing"
	"github.com/aadhii-yz/quest/internal/provider"
)

// Event is one line of tester output, or the terminal done event.
type Event struct {
	Line string
	Done bool
}

// Request describes a single tester invocation.
type Request struct {
	CourseID string
	Stage    provider.Stage
	Language string
	Tester   provider.TesterPlan
}

// Runner starts a tester and returns its event stream.
//
// Implementations must stop producing events and close the channel when the
// context is cancelled.
type Runner interface {
	Run(ctx context.Context, req Request) (<-chan Event, error)
}

// Simulated emits the shape of a real run without downloading or executing
// anything. Every step is labelled so the UI cannot imply real verification.
type Simulated struct {
	// Delay between events. Zero streams as fast as the event loop drains.
	Delay time.Duration
}

// Run implements Runner.
func (s Simulated) Run(ctx context.Context, req Request) (<-chan Event, error) {
	lines := s.script(req)
	events := make(chan Event)
	go func() {
		defer close(events)
		for _, line := range lines {
			if !s.wait(ctx) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case events <- Event{Line: line}:
			}
		}
		select {
		case <-ctx.Done():
		case events <- Event{Done: true}:
		}
	}()
	return events, nil
}

func (s Simulated) wait(ctx context.Context) bool {
	if s.Delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(s.Delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s Simulated) script(req Request) []string {
	stageID, stageName := req.Stage.ID, req.Stage.Name
	if stageID == "" {
		stageID = "unknown"
	}
	if stageName == "" {
		stageName = "current stage"
	}

	lines := []string{
		fmt.Sprintf("$ quest-tester run --stage %s --language %s", stageID, req.Language),
	}
	if !req.Tester.Available {
		return append(lines,
			fmt.Sprintf("no pinned tester build for %s", req.Tester.Platform),
			"FAIL  nothing to run on this platform",
		)
	}

	lines = append(lines,
		"note: simulated runner — no download, verification or execution happens",
		fmt.Sprintf("[sim] pinned release %s", req.Tester.Release),
		fmt.Sprintf("[sim] sha256 check for %s", req.Tester.AssetName),
		"[sim] extract into the user cache directory",
		fmt.Sprintf("[sim] run checks for %q", stageName),
		"ok    "+stageName,
		"PASS  0 failures, 2 checks",
	)
	if notice := licensing.Notice(req.Tester); notice != "" {
		lines = append(lines, "warning: "+notice)
	}
	return lines
}
