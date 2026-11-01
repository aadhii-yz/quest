// Command quest starts QuesTUI, a terminal interface for working through
// guided programming projects.
//
// The interface has two views:
//
//   - catalog: pick a resource provider (CodeCrafters, Advent of Code, ...),
//     then a course and the implementation language.
//   - course: read the selected stage, browse stages in a toggleable side
//     panel, and watch a toggleable test runner log at the bottom.
//
// Course definitions are read from local course-definition.yml files (see
// QUEST_DEFS_DIR, default "temp"). Nothing is downloaded: the test runner is a
// clearly-labelled simulation until the pinned-release download, sha256
// verification and execution pipeline exists.
package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func main() {
	p := tea.NewProgram(newApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "quest: %v\n", err)
		os.Exit(1)
	}
}

type viewMode int

const (
	viewCatalog viewMode = iota
	viewCourse
)

// app is the root tea.Model. Each view owns its own component tree; the app
// routes messages and owns the chrome (header, help footer, window size).
type app struct {
	width, height int
	view          viewMode

	theme  *theme
	cat    *catalog
	course *courseModel

	help       help.Model
	catKeys    catalogKeys
	courseKeys courseKeys
}

func newApp() app {
	t := newTheme(true)
	return app{
		view:       viewCatalog,
		theme:      t,
		cat:        newCatalog(t),
		help:       t.newHelp(),
		catKeys:    defaultCatalogKeys(),
		courseKeys: defaultCourseKeys(),
	}
}

func (m app) Init() tea.Cmd {
	return tea.Batch(tea.RequestWindowSize, tea.RequestBackgroundColor)
}

func (m app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case tea.BackgroundColorMsg:
		m.theme.isDark = msg.IsDark()
		m.help = m.theme.newHelp()
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case runTickMsg:
		if m.view == viewCourse && m.course != nil {
			return m, m.course.advance(msg)
		}
		return m, nil
	default:
		// Component messages (list filter matches, cursor blinks, ...) must
		// reach the focused component, not be swallowed by the root model.
		if m.view == viewCourse && m.course != nil {
			return m, m.course.updateFocused(msg)
		}
		return m, m.cat.updateFocused(msg)
	}
}

func (m app) View() tea.View {
	var v tea.View
	v.SetContent(m.viewString())
	v.AltScreen = true
	v.WindowTitle = "quest"
	return v
}

func (m app) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.view == viewCourse && m.course != nil {
		return m.handleCourseKey(msg)
	}
	return m.handleCatalogKey(msg)
}

func (m app) handleCatalogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// While filtering, the list owns all keys so typing "q" or "?" works.
	if m.cat.filtering() {
		return m, m.cat.updateFocused(msg)
	}
	switch {
	case key.Matches(msg, m.catKeys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.catKeys.Help):
		m.toggleHelp()
	case key.Matches(msg, m.catKeys.NextColumn):
		m.cat.moveFocus(1)
	case key.Matches(msg, m.catKeys.PrevColumn):
		m.cat.moveFocus(-1)
	case key.Matches(msg, m.catKeys.Open):
		return m.openSelection()
	case key.Matches(msg, m.catKeys.Filter):
		if len(m.cat.courses.Items()) > 0 {
			m.cat.focus = areaCourses
			return m, m.cat.updateFocused(msg)
		}
	}
	return m, m.cat.updateFocused(msg)
}

// openSelection drills into the next column, or opens the course once the
// language column is confirmed.
func (m app) openSelection() (tea.Model, tea.Cmd) {
	if m.cat.focus != areaLanguages {
		m.cat.moveFocus(1)
		return m, nil
	}
	c := m.cat.course()
	if c == nil {
		return m, nil
	}
	m.course = newCourseModel(m.theme, c, m.cat.languageIndex())
	m.view = viewCourse
	m.layout()
	return m, nil
}

func (m app) handleCourseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	c := m.course
	switch {
	case key.Matches(msg, m.courseKeys.Back):
		m.view = viewCatalog
		m.course = nil
		m.layout()
		return m, nil
	case key.Matches(msg, m.courseKeys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.courseKeys.Help):
		m.toggleHelp()
	case key.Matches(msg, m.courseKeys.NextPane):
		c.moveFocus(1)
	case key.Matches(msg, m.courseKeys.PrevPane):
		c.moveFocus(-1)
	case key.Matches(msg, m.courseKeys.ToggleStages):
		c.showStages = !c.showStages
		m.layout()
	case key.Matches(msg, m.courseKeys.ToggleTests):
		c.showTests = !c.showTests
		m.layout()
	case key.Matches(msg, m.courseKeys.NextStage):
		c.selectStage(1)
	case key.Matches(msg, m.courseKeys.PrevStage):
		c.selectStage(-1)
	case key.Matches(msg, m.courseKeys.NextLanguage):
		c.nextLanguage()
	case key.Matches(msg, m.courseKeys.Run):
		return m, c.startRun()
	default:
		return m, c.updateFocused(msg)
	}
	return m, nil
}

func (m *app) toggleHelp() {
	m.help.ShowAll = !m.help.ShowAll
	m.layout()
}

// layout recomputes the body height (which depends on the help footer size)
// and resizes the active view's components.
func (m *app) layout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	m.help.SetWidth(m.width)
	helpH := lipgloss.Height(m.help.View(m.activeKeys()))
	bodyH := max(m.height-1-helpH, 1)
	if m.view == viewCourse && m.course != nil {
		m.course.layout(m.width, bodyH)
		return
	}
	m.cat.layout(m.width, bodyH)
}

func (m app) activeKeys() help.KeyMap {
	if m.view == viewCourse {
		return m.courseKeys
	}
	return m.catKeys
}

func (m app) viewString() string {
	if m.width <= 0 || m.height <= 0 {
		return "loading…"
	}
	body := m.cat.view()
	if m.view == viewCourse && m.course != nil {
		body = m.course.view()
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.header(), body, m.help.View(m.activeKeys()))
	return fitScreen(content, m.width, m.height)
}

func (m app) header() string {
	brand := lipgloss.NewStyle().Bold(true).Foreground(m.theme.accentColor()).Render("quest")
	left := brand + m.theme.muted(" · ")
	right := ""
	if m.view == viewCourse && m.course != nil {
		left += m.course.title()
		right = m.course.meta()
	} else {
		courses := 0
		for _, p := range m.cat.providers {
			courses += len(p.courses)
		}
		left += m.theme.bold("catalog")
		right = m.theme.muted(fmt.Sprintf("%d resources · %d courses", len(m.cat.providers), courses))
	}
	return bar(left, right, m.width)
}

// -----------------------------------------------------------------------------
// key maps
// -----------------------------------------------------------------------------

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
		{k.Help, k.Quit},
	}
}

// -----------------------------------------------------------------------------
// small text helpers
// -----------------------------------------------------------------------------

// plain strips the light Markdown used in course definitions and collapses
// whitespace, for one-line summaries.
func plain(s string) string {
	replacer := strings.NewReplacer("**", "", "`", "", "__", "")
	return strings.Join(strings.Fields(replacer.Replace(stripLinks(s))), " ")
}

// stripLinks rewrites [text](url) as text, keeping prose readable where URLs
// would otherwise dominate a one-line summary.
func stripLinks(s string) string {
	if !strings.ContainsRune(s, '[') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '[' {
			mid := strings.IndexByte(s[i:], ']')
			if mid > 0 {
				close := i + mid
				if close+1 < len(s) && s[close+1] == '(' {
					if end := strings.IndexByte(s[close:], ')'); end > 0 {
						b.WriteString(s[i+1 : close])
						i = close + end + 1
						continue
					}
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return min(max(v, lo), hi)
}

// fitScreen forces the rendered frame to exactly w x h cells.
func fitScreen(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = fit(lines[i], w)
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", max(w, 0)))
	}
	return strings.Join(lines, "\n")
}
