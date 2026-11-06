package ui

import "charm.land/bubbles/v2/key"

// catalogKeys are the catalog screen's bindings. Navigation keys are forwarded
// to the focused list; these bindings exist for app-level actions and help.
type catalogKeys struct {
	Move       key.Binding
	NextColumn key.Binding
	PrevColumn key.Binding
	Open       key.Binding
	Filter     key.Binding
	Help       key.Binding
	Quit       key.Binding
}

func defaultCatalogKeys() catalogKeys {
	return catalogKeys{
		Move: key.NewBinding(
			key.WithKeys("up", "down", "k", "j"),
			key.WithHelp("↑/↓", "move"),
		),
		NextColumn: key.NewBinding(
			key.WithKeys("right", "l", "tab"),
			key.WithHelp("→/tab", "next column"),
		),
		PrevColumn: key.NewBinding(
			key.WithKeys("left", "h", "shift+tab"),
			key.WithHelp("←", "previous column"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k catalogKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Move, k.NextColumn, k.Open, k.Filter, k.Help, k.Quit}
}

func (k catalogKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Move, k.PrevColumn, k.NextColumn},
		{k.Open, k.Filter},
		{k.Help, k.Quit},
	}
}

// courseKeys are the course screen's bindings.
type courseKeys struct {
	Back         key.Binding
	NextPane     key.Binding
	PrevPane     key.Binding
	NextStage    key.Binding
	PrevStage    key.Binding
	NextLanguage key.Binding
	ToggleStages key.Binding
	ToggleTests  key.Binding
	Run          key.Binding
	Create       key.Binding
	Help         key.Binding
	Quit         key.Binding
}

func defaultCourseKeys() courseKeys {
	return courseKeys{
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "catalog"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "focus"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "focus back"),
		),
		NextStage: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n/p", "stage"),
		),
		PrevStage: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "previous stage"),
		),
		NextLanguage: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "language"),
		),
		ToggleStages: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stages"),
		),
		ToggleTests: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tests"),
		),
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create project"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k courseKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Back, k.NextPane, k.NextStage, k.ToggleStages, k.ToggleTests, k.Run, k.Help, k.Quit,
	}
}

func (k courseKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.NextPane, k.PrevPane},
		{k.NextStage, k.PrevStage, k.NextLanguage},
		{k.ToggleStages, k.ToggleTests, k.Run},
		{k.Create, k.Help, k.Quit},
	}
}
