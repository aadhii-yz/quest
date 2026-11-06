// Package ui implements the QuesTUI terminal interface.
//
// The root model owns the chrome (header, help footer, window size) and routes
// messages to one of two screens: the catalog and the active course.
package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/runner"
)

type viewMode int

const (
	viewCatalog viewMode = iota
	viewCourse
)

// app is the root tea.Model. Each view owns its own component tree; the app
// routes messages and owns the chrome.
type app struct {
	width, height int
	view          viewMode

	theme  *theme
	cat    *catalog
	course *courseModel
	runner runner.Runner
	loader ContentLoader

	help       help.Model
	catKeys    catalogKeys
	courseKeys courseKeys
}

// Option configures the root model.
type Option func(*app)

// WithContentLoader lets the UI request verified content for a course. The
// loader is wired in cmd/quest; a nil loader disables content loading.
func WithContentLoader(l ContentLoader) Option {
	return func(a *app) { a.loader = l }
}

// New returns the root model for tea.NewProgram.
func New(providers []provider.Provider, r runner.Runner, opts ...Option) tea.Model {
	return newApp(providers, r, opts...)
}

func newApp(providers []provider.Provider, r runner.Runner, opts ...Option) app {
	t := newTheme(true)
	m := app{
		view:       viewCatalog,
		theme:      t,
		cat:        newCatalog(t, providers),
		runner:     r,
		help:       t.newHelp(),
		catKeys:    defaultCatalogKeys(),
		courseKeys: defaultCourseKeys(),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
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
	case runEventMsg:
		if m.view == viewCourse && m.course != nil {
			return m, m.course.handleEvent(msg)
		}
		return m, nil
	case contentLoadedMsg:
		return m.handleContentLoaded(msg), nil
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
	c, ok := m.cat.course()
	if !ok {
		return m, nil
	}
	m.course = newCourseModel(m.theme, c, m.cat.languageIndex(), m.runner)
	m.view = viewCourse
	m.layout()
	if m.loader == nil {
		return m, nil
	}
	if !c.Ready() {
		m.course.beginLoading()
	}
	return m, loadContentCmd(m.loader, c.ID)
}

// handleContentLoaded refreshes the catalog and the open course after a bundle
// was fetched and verified.
func (m app) handleContentLoaded(msg contentLoadedMsg) app {
	if len(msg.providers) > 0 {
		m.cat.setProviders(msg.providers)
	}
	if m.course == nil || m.course.course.ID != msg.slug {
		return m
	}
	if msg.err != nil {
		m.course.finishLoading()
		m.course.appendLog("content  " + msg.err.Error())
		return m
	}
	if c, ok := findCourse(msg.providers, msg.slug); ok {
		m.course.setCourse(c)
		m.layout()
	}
	return m
}

func (m app) handleCourseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	c := m.course
	switch {
	case key.Matches(msg, m.courseKeys.Back):
		c.stopRun()
		m.view = viewCatalog
		m.course = nil
		m.layout()
		return m, nil
	case key.Matches(msg, m.courseKeys.Quit):
		c.stopRun()
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
			courses += len(p.Courses())
		}
		left += m.theme.bold("catalog")
		right = m.theme.muted(fmt.Sprintf("%d resources · %d courses", len(m.cat.providers), courses))
	}
	return bar(left, right, m.width)
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
