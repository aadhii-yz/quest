package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aadhii-yz/quest/internal/codecrafters"
	"gopkg.in/yaml.v3"
)

// -----------------------------------------------------------------------------
// catalog data
// -----------------------------------------------------------------------------

type course struct {
	name      string
	slug      string
	provider  string
	summary   string
	languages []string
	stages    []codecrafters.Stage

	// note explains why a course is not fully ready, if it is not.
	note string

	// tester release details, so the UI can show what would be executed and
	// whether it is licensed and available for this platform.
	testerTag  string
	testerName string
	license    codecrafters.LicenseStatus
	assetKey   string
	assetOK    bool
}

func (c *course) languageAt(i int) string {
	if len(c.languages) == 0 {
		return ""
	}
	return c.languages[clampInt(i, 0, len(c.languages)-1)]
}

func (c *course) ready() bool { return len(c.stages) > 0 }

type provider struct {
	name    string
	note    string
	courses []*course
	planned bool
}

// defaultLanguages is used when no course definition is available locally, so
// the language selector is never empty.
var defaultLanguages = []string{
	"go", "rust", "python", "javascript", "typescript", "c", "cpp",
	"java", "kotlin", "zig",
}

// aocYears are pinned rather than derived from the clock: the current year's
// puzzles only exist after December starts.
var aocYears = []int{2024, 2023, 2022, 2021, 2020, 2019, 2018, 2017, 2016, 2015}

func defsDir() string {
	if d := os.Getenv("QUEST_DEFS_DIR"); d != "" {
		return d
	}
	return "temp"
}

// loadCatalog assembles the list of available resources. CodeCrafters courses
// come from the pinned Sources; their languages and stages are filled in from
// any course-definition.yml found under the local definitions directory.
func loadCatalog() []*provider {
	defs := loadDefinitions(defsDir())
	osArch := runtime.GOOS + "_" + runtime.GOARCH

	cc := &provider{
		name: "CodeCrafters",
		note: "Build real tools from scratch. Tester binaries are downloaded per pinned release, verified by sha256 and run locally.",
	}
	for i := range codecrafters.Sources {
		src := &codecrafters.Sources[i]
		slug := slugFromRepo(src.RepoURL)
		c := &course{
			name:      strings.TrimPrefix(src.CourseName, "Build your own "),
			slug:      slug,
			provider:  cc.name,
			license:   src.TesterLicense,
			testerTag: src.TesterRelease,
		}
		if asset, ok := src.AssetFor(osArch); ok {
			c.assetOK = true
			c.assetKey = osArch
			c.testerName = filepath.Base(asset.URL)
		}
		if def, ok := defs[slug]; ok {
			c.summary = plain(def.ShortDescriptionMD)
			for _, l := range def.Languages {
				c.languages = append(c.languages, l.Slug)
			}
			c.stages = def.Stages
		}
		if len(c.languages) == 0 {
			c.languages = defaultLanguages
		}
		if !c.ready() {
			c.note = "no course-definition.yml found locally"
		}
		cc.courses = append(cc.courses, c)
	}

	return []*provider{
		cc,
		aocProvider(),
		{
			name:    "Exercism",
			note:    "Mentored exercises and concept tracks. Not wired up yet.",
			planned: true,
		},
		{
			name:    "Open-source courses",
			note:    "Other openly licensed guided projects will be added here.",
			planned: true,
		},
	}
}

func aocProvider() *provider {
	p := &provider{
		name: "Advent of Code",
		note: "Annual puzzles. Quest fetches puzzle text politely, one request at a time, and caches it locally.",
	}
	for _, year := range aocYears {
		c := &course{
			name:      fmt.Sprintf("%d", year),
			slug:      fmt.Sprintf("aoc-%d", year),
			provider:  p.name,
			summary:   fmt.Sprintf("25 two-part puzzles from the %d Advent of Code.", year),
			languages: defaultLanguages,
			note:      "puzzle fetching is not implemented yet",
		}
		for day := 1; day <= 25; day++ {
			c.stages = append(c.stages, codecrafters.Stage{
				Slug: fmt.Sprintf("day-%02d", day),
				Name: fmt.Sprintf("Day %d", day),
				MarketingMD: fmt.Sprintf(
					"Puzzle text for %d day %d will appear here once polite fetching and local caching are implemented.",
					year, day),
			})
		}
		p.courses = append(p.courses, c)
	}
	return p
}

func slugFromRepo(repoURL string) string {
	slug := filepath.Base(strings.TrimSuffix(repoURL, "/"))
	return strings.TrimPrefix(slug, "build-your-own-")
}

// loadDefinitions reads course-definition.yml files from root itself and one
// or two directory levels below it (temp/<course>/, temp/<provider>/<course>/).
func loadDefinitions(root string) map[string]codecrafters.CourseDefinition {
	out := map[string]codecrafters.CourseDefinition{}
	patterns := []string{
		filepath.Join(root, "course-definition.yml"),
		filepath.Join(root, "*", "course-definition.yml"),
		filepath.Join(root, "*", "*", "course-definition.yml"),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var def codecrafters.CourseDefinition
			if err := yaml.Unmarshal(data, &def); err != nil || def.Slug == "" {
				continue
			}
			if _, exists := out[def.Slug]; exists {
				continue
			}
			out[def.Slug] = def
		}
	}
	return out
}

// -----------------------------------------------------------------------------
// catalog model
// -----------------------------------------------------------------------------

const (
	areaResources = iota
	areaCourses
	areaLanguages
)

type resourceItem struct{ p *provider }

func (i resourceItem) FilterValue() string { return i.p.name }
func (i resourceItem) row() (string, string) {
	badge := fmt.Sprintf("%d", len(i.p.courses))
	if i.p.planned {
		badge = "soon"
	}
	return i.p.name, badge
}

type courseItem struct{ c *course }

func (i courseItem) FilterValue() string {
	return strings.Join([]string{i.c.name, i.c.provider, i.c.summary}, " ")
}
func (i courseItem) row() (string, string) {
	if !i.c.ready() {
		return i.c.name, "no spec"
	}
	return i.c.name, ""
}

type languageItem struct{ slug string }

func (i languageItem) FilterValue() string { return i.slug }
func (i languageItem) row() (string, string) {
	return i.slug, ""
}

type catalog struct {
	theme     *theme
	providers []*provider

	resources list.Model
	courses   list.Model
	languages list.Model

	focus         int
	width, height int

	lastProvider *provider
	lastCourse   *course
}

func newCatalog(t *theme) *catalog {
	c := &catalog{theme: t, providers: loadCatalog()}
	items := make([]list.Item, 0, len(c.providers))
	for _, p := range c.providers {
		items = append(items, resourceItem{p})
	}
	c.resources = newRowList(items, t, false, "resources")
	c.courses = newRowList(nil, t, true, "courses")
	c.languages = newRowList(nil, t, false, "languages")
	c.resources.Select(0)
	c.refresh()
	return c
}

// provider returns the resource selected in the resources list.
func (c *catalog) provider() *provider {
	item, ok := c.resources.SelectedItem().(resourceItem)
	if !ok {
		return nil
	}
	return item.p
}

// course returns the course selected in the courses list.
func (c *catalog) course() *course {
	item, ok := c.courses.SelectedItem().(courseItem)
	if !ok {
		return nil
	}
	return item.c
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

// refresh rebuilds dependent lists after a selection changed.
func (c *catalog) refresh() {
	if p := c.provider(); p != c.lastProvider {
		c.lastProvider = p
		items := make([]list.Item, 0, len(p.courses))
		for _, cr := range p.courses {
			items = append(items, courseItem{cr})
		}
		c.courses.SetItems(items)
		c.courses.ResetFilter()
		c.courses.Select(0)
		c.lastCourse = nil
	}
	if cr := c.course(); cr != c.lastCourse {
		c.lastCourse = cr
		var items []list.Item
		if cr != nil {
			for _, lang := range cr.languages {
				items = append(items, languageItem{lang})
			}
		}
		c.languages.SetItems(items)
		c.languages.ResetFilter()
		c.languages.Select(0)
	}
	if !c.focusValid() {
		c.focus = areaResources
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
	resW := clampInt(c.width/4, 20, 30)
	langW := clampInt(c.width/6, 14, 22)
	courseW := c.width - resW - langW - 2
	if courseW < 24 {
		langW = max(langW-(24-courseW), 12)
		courseW = c.width - resW - langW - 2
	}
	cw, ch := panelInner(resW, c.height)
	c.resources.SetSize(cw, ch)
	cw, ch = panelInner(courseW, c.height)
	c.courses.SetSize(cw, ch)
	cw, ch = panelInner(langW, c.height)
	c.languages.SetSize(cw, ch)
}

func (c *catalog) view() string {
	if c.width < 64 {
		title := "resources"
		switch c.focus {
		case areaCourses:
			title = "courses · " + c.provider().name
		case areaLanguages:
			if cr := c.course(); cr != nil {
				title = "language · " + cr.name
			}
		}
		return c.theme.panel(c.theme.accentBold(title), c.list(c.focus).View(), c.width, c.height, true)
	}

	resW := clampInt(c.width/4, 20, 30)
	langW := clampInt(c.width/6, 14, 22)
	courseW := c.width - resW - langW - 2
	if courseW < 24 {
		langW = max(langW-(24-courseW), 12)
		courseW = c.width - resW - langW - 2
	}

	courseTitle := c.theme.accentBold("courses")
	if p := c.provider(); p != nil {
		courseTitle += c.theme.muted(" · " + p.name)
	}
	langTitle := c.theme.accentBold("language")
	if cr := c.course(); cr != nil {
		langTitle += c.theme.muted(" · " + cr.name)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		c.theme.panel(c.theme.accentBold("resources"), c.resources.View(), resW, c.height, c.focus == areaResources),
		" ",
		c.theme.panel(courseTitle, c.courses.View(), courseW, c.height, c.focus == areaCourses),
		" ",
		c.theme.panel(langTitle, c.languages.View(), langW, c.height, c.focus == areaLanguages),
	)
}
