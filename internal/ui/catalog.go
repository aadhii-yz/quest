package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aadhii-yz/quest/internal/provider"
)

const (
	areaResources = iota
	areaCourses
	areaLanguages
)

type resourceItem struct{ p provider.Provider }

func (i resourceItem) FilterValue() string { return i.p.Name() }
func (i resourceItem) row() (string, string) {
	badge := fmt.Sprintf("%d", len(i.p.Courses()))
	if i.p.Planned() {
		badge = "soon"
	}
	return i.p.Name(), badge
}

type courseItem struct{ c provider.Course }

func (i courseItem) FilterValue() string {
	return strings.Join([]string{i.c.Name, i.c.ProviderName, i.c.Summary}, " ")
}
func (i courseItem) row() (string, string) {
	if !i.c.Ready() {
		return i.c.Name, "no spec"
	}
	return i.c.Name, ""
}

type languageItem struct{ slug string }

func (i languageItem) FilterValue() string { return i.slug }
func (i languageItem) row() (string, string) {
	return i.slug, ""
}

// catalog is the browse screen: resources, their courses, and the selected
// course's implementation languages.
type catalog struct {
	theme     *theme
	providers []provider.Provider

	resources list.Model
	courses   list.Model
	languages list.Model

	focus         int
	width, height int

	lastProviderID string
	lastCourseID   string
}

func newCatalog(t *theme, providers []provider.Provider) *catalog {
	c := &catalog{theme: t, providers: providers}
	items := make([]list.Item, 0, len(providers))
	for _, p := range providers {
		items = append(items, resourceItem{p})
	}
	c.resources = newRowList(items, t, false, "resources")
	c.courses = newRowList(nil, t, true, "courses")
	c.languages = newRowList(nil, t, false, "languages")
	c.resources.Select(0)
	c.refresh()
	return c
}

// provider returns the selected resource, or nil when none is selected.
func (c *catalog) provider() provider.Provider {
	item, ok := c.resources.SelectedItem().(resourceItem)
	if !ok {
		return nil
	}
	return item.p
}

// course returns the selected course, or ok=false when none is selected.
func (c *catalog) course() (provider.Course, bool) {
	item, ok := c.courses.SelectedItem().(courseItem)
	if !ok {
		return provider.Course{}, false
	}
	return item.c, true
}

func (c *catalog) languageIndex() int { return c.languages.Index() }

func (c *catalog) list(area int) *list.Model {
	switch area {
	case areaResources:
		return &c.resources
	case areaCourses:
		return &c.courses
	case areaLanguages:
		return &c.languages
	}
	return nil
}

// areas lists the columns that currently hold something selectable.
func (c *catalog) areas() []int {
	areas := []int{areaResources}
	if len(c.courses.Items()) > 0 {
		areas = append(areas, areaCourses)
	}
	if len(c.languages.Items()) > 0 {
		areas = append(areas, areaLanguages)
	}
	return areas
}

func (c *catalog) focusValid() bool {
	for _, a := range c.areas() {
		if a == c.focus {
			return true
		}
	}
	return false
}

func (c *catalog) moveFocus(delta int) {
	areas := c.areas()
	idx := 0
	for i, a := range areas {
		if a == c.focus {
			idx = i
		}
	}
	idx = (idx + delta + len(areas)) % len(areas)
	c.focus = areas[idx]
}

// filtering reports whether the focused list is currently capturing keystrokes
// for a filter query.
func (c *catalog) filtering() bool {
	if l := c.list(c.focus); l != nil {
		return l.SettingFilter()
	}
	return false
}

// refresh rebuilds dependent lists after a selection changed. Selections are
// tracked by ID because provider.Course is a value type.
func (c *catalog) refresh() {
	if p := c.provider(); p != nil && p.ID() != c.lastProviderID {
		c.lastProviderID = p.ID()
		courses := p.Courses()
		items := make([]list.Item, 0, len(courses))
		for _, cr := range courses {
			items = append(items, courseItem{cr})
		}
		c.courses.SetItems(items)
		c.courses.ResetFilter()
		c.courses.Select(0)
		c.lastCourseID = ""
	}
	if cr, ok := c.course(); ok && cr.ID != c.lastCourseID {
		c.lastCourseID = cr.ID
		items := make([]list.Item, 0, len(cr.Languages))
		for _, lang := range cr.Languages {
			items = append(items, languageItem{lang})
		}
		c.languages.SetItems(items)
		c.languages.ResetFilter()
		c.languages.Select(0)
	} else if !ok && c.lastCourseID != "" {
		c.lastCourseID = ""
		c.languages.SetItems(nil)
		c.languages.Select(0)
	}
	if !c.focusValid() {
		c.focus = areaResources
	}
	c.resize()
}

// setProviders replaces the provider list after content was loaded elsewhere,
// keeping the selected provider, course and language positions when possible.
func (c *catalog) setProviders(providers []provider.Provider) {
	provIdx, courseIdx, langIdx := c.resources.Index(), c.courses.Index(), c.languages.Index()

	c.providers = providers
	items := make([]list.Item, 0, len(providers))
	for _, p := range providers {
		items = append(items, resourceItem{p})
	}
	c.resources.SetItems(items)
	if n := len(items); n > 0 {
		c.resources.Select(clampInt(provIdx, 0, n-1))
	}
	c.lastProviderID = ""
	c.lastCourseID = ""
	c.refresh()

	if n := len(c.courses.Items()); n > 0 {
		c.courses.Select(clampInt(courseIdx, 0, n-1))
		c.lastCourseID = ""
		c.refresh()
	}
	if n := len(c.languages.Items()); n > 0 {
		c.languages.Select(clampInt(langIdx, 0, n-1))
	}
	c.resize()
}

// updateFocused forwards a message to the focused list and refreshes the
// dependent columns afterwards.
func (c *catalog) updateFocused(msg tea.Msg) tea.Cmd {
	l := c.list(c.focus)
	if l == nil {
		return nil
	}
	var cmd tea.Cmd
	*l, cmd = l.Update(msg)
	c.refresh()
	return cmd
}

func (c *catalog) layout(w, h int) {
	c.width, c.height = w, h
	c.resize()
}

// resize sizes every list to the pane it will be rendered in. In narrow mode
// only the focused column is shown, full width.
func (c *catalog) resize() {
	if c.width <= 0 || c.height <= 0 {
		return
	}
	if c.width < 64 {
		cw, ch := panelInner(c.width, c.height)
		for _, a := range []int{areaResources, areaCourses, areaLanguages} {
			c.list(a).SetSize(cw, ch)
		}
		return
	}
	resW, courseW, langW := c.columnWidths()
	cw, ch := panelInner(resW, c.height)
	c.resources.SetSize(cw, ch)
	cw, ch = panelInner(courseW, c.height)
	c.courses.SetSize(cw, ch)
	cw, ch = panelInner(langW, c.height)
	c.languages.SetSize(cw, ch)
}

// columnWidths splits the catalog width into the three columns.
func (c *catalog) columnWidths() (resW, courseW, langW int) {
	resW = clampInt(c.width/4, 20, 30)
	langW = clampInt(c.width/6, 14, 22)
	courseW = c.width - resW - langW - 2
	if courseW < 24 {
		langW = max(langW-(24-courseW), 12)
		courseW = c.width - resW - langW - 2
	}
	return resW, courseW, langW
}

func (c *catalog) view() string {
	if c.width < 64 {
		title := "resources"
		switch c.focus {
		case areaCourses:
			if p := c.provider(); p != nil {
				title = "courses · " + p.Name()
			}
		case areaLanguages:
			if cr, ok := c.course(); ok {
				title = "language · " + cr.Name
			}
		}
		return c.theme.panel(c.theme.accentBold(title), c.list(c.focus).View(), c.width, c.height, true)
	}

	resW, courseW, langW := c.columnWidths()
	courseTitle := c.theme.accentBold("courses")
	if p := c.provider(); p != nil {
		courseTitle += c.theme.muted(" · " + p.Name())
	}
	langTitle := c.theme.accentBold("language")
	if cr, ok := c.course(); ok {
		langTitle += c.theme.muted(" · " + cr.Name)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		c.theme.panel(c.theme.accentBold("resources"), c.resources.View(), resW, c.height, c.focus == areaResources),
		" ",
		c.theme.panel(courseTitle, c.courses.View(), courseW, c.height, c.focus == areaCourses),
		" ",
		c.theme.panel(langTitle, c.languages.View(), langW, c.height, c.focus == areaLanguages),
	)
}
