// Command quest starts QuesTUI, a terminal interface for working through
// guided programming projects.
//
// The entry point only resolves user paths, wires the available providers and
// the runner, and starts Bubble Tea. All UI code lives in internal/ui.
package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aadhii-yz/quest/internal/config"
	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/provider/aoc"
	"github.com/aadhii-yz/quest/internal/provider/codecrafters"
	"github.com/aadhii-yz/quest/internal/runner"
	"github.com/aadhii-yz/quest/internal/ui"
)

func main() {
	cfg := config.Resolve()
	providers := []provider.Provider{
		codecrafters.New(cfg.DefsDir),
		aoc.New(),
		provider.NewPlaceholder("exercism", "Exercism",
			"Mentored exercises and concept tracks. Not wired up yet."),
		provider.NewPlaceholder("open-source", "Open-source courses",
			"Other openly licensed guided projects will be added here."),
	}

	program := tea.NewProgram(ui.New(providers, runner.Simulated{Delay: 180 * time.Millisecond}))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "quest: %v\n", err)
		os.Exit(1)
	}
}
