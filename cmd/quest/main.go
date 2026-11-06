// Command quest starts QuesTUI, a terminal interface for working through
// guided programming projects.
//
// The entry point only resolves user paths, wires the available providers and
// the runner, and starts Bubble Tea. All UI code lives in internal/ui.
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aadhii-yz/quest/internal/config"
	"github.com/aadhii-yz/quest/internal/content"
	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/provider/aoc"
	"github.com/aadhii-yz/quest/internal/provider/codecrafters"
	"github.com/aadhii-yz/quest/internal/runner"
	"github.com/aadhii-yz/quest/internal/ui"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println(version())
		return
	}

	cfg := config.Resolve()

	buildProviders := func(defsDir string) []provider.Provider {
		return []provider.Provider{
			codecrafters.New(defsDir),
			aoc.New(),
			provider.NewPlaceholder("exercism", "Exercism",
				"Mentored exercises and concept tracks. Not wired up yet."),
			provider.NewPlaceholder("open-source", "Open-source courses",
				"Other openly licensed guided projects will be added here."),
		}
	}

	defsDir := cfg.DefsDir
	var loader ui.ContentLoader
	if defsDir == "" {
		// No local override: read verified bundles from the pinned release.
		// The cache root doubles as the adapter's definitions directory.
		store := content.New(cfg.CacheDir, contentPins())
		defsDir = store.Root()
		loader = &contentLoader{
			store: store,
			build: func() []provider.Provider { return buildProviders(store.Root()) },
		}
	}

	program := tea.NewProgram(ui.New(
		buildProviders(defsDir),
		runner.Simulated{Delay: 180 * time.Millisecond},
		ui.WithContentLoader(loader),
	))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "quest: %v\n", err)
		os.Exit(1)
	}
}

// buildVersion is injected for release builds with
// -ldflags "-X main.buildVersion=<tag>". Local builds leave it empty and fall
// back to module and VCS information.
var buildVersion string

// version reports the build version and revision. Release builds carry the tag
// through buildVersion; `go install` carries a module version; local builds
// fall back to "devel" plus the VCS revision recorded in the binary.
func version() string {
	v := buildVersion

	var rev, dirty string
	if info, ok := debug.ReadBuildInfo(); ok {
		if v == "" {
			v = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				rev = setting.Value
			case "vcs.modified":
				if setting.Value == "true" {
					dirty = "-dirty"
				}
			}
		}
	}

	return formatVersion(v, rev, dirty)
}

func formatVersion(v, rev, dirty string) string {
	v = strings.TrimSuffix(v, "+dirty")
	if v == "" || v == "(devel)" {
		v = "devel"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if rev == "" {
		return "quest " + v
	}
	return fmt.Sprintf("quest %s (%s)", v, rev+dirty)
}
