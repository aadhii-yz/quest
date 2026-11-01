package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aadhii-yz/quest/internal/codecrafters"
)

const (
	paneStages = iota
	paneDescription
	paneTests
)

type runState int

const (
	runIdle runState = iota
	runRunning
	runDone
)

const maxLogLines = 300

type stageItem struct {
	stage codecrafters.Stage
	num   int
}

func (i stageItem) FilterValue() string { return i.stage.Name }
func (i stageItem) row() (string, string) {
	return fmt.Sprintf("%2d %s", i.num, i.stage.Name), difficultyLabel(i.stage.Difficulty)
}

type courseModel struct {
	theme   *theme
	course  *course
	langIdx int

	stages list.Model
	desc   viewport.Model
	log    viewport.Model

	logLines []string
	steps    []string
	run      runState
	seq      int

	showStages bool
	showTests  bool
	focus      int

	width, height int
	stageW        int
	descH, testH  int

	stageIdx int
	descKey  string
	descSlug string
}

type runTickMsg struct{ seq int }

func newCourseModel(t *theme, c *course, langIdx int) *courseModel {
	if len(c.languages) > 0 {
		langIdx = clampInt(langIdx, 0, len(c.languages)-1)
	}
	items := make([]list.Item, 0, len(c.stages))
	for i, stage := range c.stages {
		items = append(items, stageItem{stage: stage, num: i + 1})
	}
	stages := newRowList(items, t, false, "stages")
	stages.Select(0)

	desc := viewport.New()
	desc.SoftWrap = true
	desc.FillHeight = true

	log := viewport.New()
	log.SoftWrap = true
	log.FillHeight = true

	m := &courseModel{
		theme:      t,
		course:     c,
		langIdx:    langIdx,
		stages:     stages,
		desc:       desc,
		log:        log,
		showStages: true,
		showTests:  true,
		focus:      paneDescription,
		logLines: []string{
			"test runner ready — press r to run the current stage",
			"note: this log is simulated; tester download and execution are not wired up yet",
		},
	}
	m.refreshLog()
	return m
}

func (c *courseModel) language() string { return c.course.languageAt(c.langIdx) }

func (c *courseModel) stage() (codecrafters.Stage, bool) {
	if item, ok := c.stages.SelectedItem().(stageItem); ok {
		return item.stage, true
	}
	return codecrafters.Stage{}, false
}

func (c *courseModel) visiblePanes() []int {
	panes := make([]int, 0, 3)
	if c.stageW > 0 {
		panes = append(panes, paneStages)
	}
	panes = append(panes, paneDescription)
	if c.testH > 0 {
		panes = append(panes, paneTests)
	}
	return panes
}

func (c *courseModel) moveFocus(delta int) {
	panes := c.visiblePanes()
	idx := 0
	for i, p := range panes {
		if p == c.focus {
			idx = i
		}
	}
	idx = (idx + delta + len(panes)) % len(panes)
	c.focus = panes[idx]
}

func (c *courseModel) ensureFocus() {
	for _, p := range c.visiblePanes() {
		if p == c.focus {
			return
		}
	}
	c.focus = paneDescription
}

func (c *courseModel) nextLanguage() {
	if n := len(c.course.languages); n > 0 {
		c.langIdx = (c.langIdx + 1) % n
	}
}

// selectStage moves the stage cursor, which keeps the description in sync.
func (c *courseModel) selectStage(delta int) {
	switch {
	case delta > 0:
		c.stages.CursorDown()
	case delta < 0:
		c.stages.CursorUp()
	}
	c.syncStage()
}

func (c *courseModel) syncStage() {
	c.stageIdx = c.stages.Index()
	c.refreshDescription(false)
}

// updateFocused forwards a message to the focused pane.
func (c *courseModel) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch c.focus {
	case paneStages:
		c.stages, cmd = c.stages.Update(msg)
		c.syncStage()
	case paneDescription:
		c.desc, cmd = c.desc.Update(msg)
	case paneTests:
		c.log, cmd = c.log.Update(msg)
	}
	return cmd
}

func (c *courseModel) layout(w, h int) {
	c.width, c.height = w, h
	c.stageW = 0
	if c.showStages && w >= 50 {
		c.stageW = clampInt(w/3, 22, 36)
	}
	rightW := w
	if c.stageW > 0 {
		rightW = w - c.stageW - 1
	}

	c.testH = 0
	if c.showTests && h >= 10 {
		c.testH = clampInt(h/3, 6, h-5)
	}
	c.descH = h
	if c.testH > 0 {
		c.descH = h - c.testH - 1
	}

	if c.stageW > 0 {
		cw, ch := panelInner(c.stageW, h)
		c.stages.SetSize(cw, ch)
	}
	cw, ch := panelInner(rightW, c.descH)
	c.desc.SetWidth(cw)
	c.desc.SetHeight(ch)
	if c.testH > 0 {
		tw, th := panelInner(rightW, c.testH)
		c.log.SetWidth(tw)
		c.log.SetHeight(th)
	}

	c.ensureFocus()
	c.refreshDescription(false)
	c.refreshLog()
}

// refreshDescription re-renders the stage Markdown with glamour. The cache key
// covers the rendered width and the background style, so resizing and theme
// changes are picked up without re-rendering on every frame.
func (c *courseModel) refreshDescription(force bool) {
	if c.desc.Width() <= 0 {
		return
	}
	stage, ok := c.stage()
	if !ok {
		key := fmt.Sprintf("%d|empty|%t", c.desc.Width(), c.theme.isDark)
		if key == c.descKey {
			return
		}
		c.descKey, c.descSlug = key, ""
		c.desc.SetContent(c.emptyDescription())
		c.desc.GotoTop()
		return
	}

	key := fmt.Sprintf("%d|%s|%t", c.desc.Width(), stage.Slug, c.theme.isDark)
	if key == c.descKey {
		return
	}
	stageChanged := stage.Slug != c.descSlug
	c.descKey, c.descSlug = key, stage.Slug

	var b strings.Builder
	meta := make([]string, 0, 3)
	if d := difficultyLabel(stage.Difficulty); d != "" {
		meta = append(meta, d)
	}
	if stage.PrimaryExtensionSlug != "" {
		meta = append(meta, "extension "+stage.PrimaryExtensionSlug)
	}
	meta = append(meta, "slug "+stage.Slug)
	b.WriteString(c.theme.muted(strings.Join(meta, " · ")))
	b.WriteString("\n\n")
	b.WriteString(c.theme.markdown(stage.MarketingMD, c.desc.Width()))

	c.desc.SetContent(b.String())
	if stageChanged || force {
		c.desc.GotoTop()
	}
}

func (c *courseModel) emptyDescription() string {
	var b strings.Builder
	b.WriteString(c.theme.bold("No stage list loaded"))
	b.WriteString("\n\n")
	b.WriteString(c.theme.muted("This course has no course-definition.yml attached yet."))
	b.WriteString("\n\n")
	if c.course.note != "" {
		b.WriteString(c.theme.muted(c.course.note))
		b.WriteString("\n\n")
	}
	b.WriteString(c.theme.muted("Place course-definition.yml under " + defsDir() + "/<course>/ and reopen."))
	return b.String()
}

func (c *courseModel) refreshLog() {
	if c.log.Width() <= 0 {
		return
	}
	atBottom := c.log.AtBottom()
	lines := make([]string, 0, len(c.logLines))
	for _, line := range c.logLines {
		lines = append(lines, c.theme.logLine(line))
	}
	c.log.SetContent(strings.Join(lines, "\n"))
	if atBottom {
		c.log.GotoBottom()
	}
}

func (c *courseModel) appendLog(lines ...string) {
	c.logLines = append(c.logLines, lines...)
	if len(c.logLines) > maxLogLines {
		c.logLines = c.logLines[len(c.logLines)-maxLogLines:]
	}
	c.refreshLog()
}

// startRun begins a simulated test run. Real execution must go through the
// pinned-release download, sha256 verification and consent gate first.
func (c *courseModel) startRun() tea.Cmd {
	if c.run == runRunning {
		return nil
	}
	c.seq++
	c.run = runRunning
	c.steps = c.runScript()
	c.appendLog("")
	if len(c.steps) == 0 {
		c.run = runDone
		return nil
	}
	c.appendLog(c.steps[0])
	c.steps = c.steps[1:]
	if len(c.steps) == 0 {
		c.run = runDone
		return nil
	}
	return c.tick()
}

func (c *courseModel) tick() tea.Cmd {
	seq := c.seq
	return tea.Tick(220*time.Millisecond, func(time.Time) tea.Msg {
		return runTickMsg{seq: seq}
	})
}

func (c *courseModel) advance(msg runTickMsg) tea.Cmd {
	if msg.seq != c.seq || c.run != runRunning {
		return nil
	}
	if len(c.steps) == 0 {
		c.run = runDone
		return nil
	}
	c.appendLog(c.steps[0])
	c.steps = c.steps[1:]
	if len(c.steps) == 0 {
		c.run = runDone
		return nil
	}
	return c.tick()
}

func (c *courseModel) runScript() []string {
	stage, ok := c.stage()
	stageSlug, stageName := "unknown", "current stage"
	if ok {
		stageSlug, stageName = stage.Slug, stage.Name
	}
	lines := []string{
		fmt.Sprintf("$ quest-tester run --stage %s --language %s", stageSlug, c.language()),
	}
	if !c.course.assetOK {
		return append(lines,
			fmt.Sprintf("no pinned tester build for %s/%s", runtime.GOOS, runtime.GOARCH),
			"FAIL  nothing to run on this platform",
		)
	}
	lines = append(lines,
		"note: simulated run — real tester execution is not wired up yet",
		fmt.Sprintf("[1/4] resolved pinned release %s", c.course.testerTag),
		fmt.Sprintf("[2/4] verified sha256 of %s", c.course.testerName),
		"[3/4] extracted into the user cache directory",
		fmt.Sprintf("[4/4] running checks for %q", stageName),
		"ok    "+stageName,
		"PASS  0 failures, 2 checks",
	)
	if c.course.license == codecrafters.LicenseNone {
		lines = append(lines, "warning: this tester has no license; running it needs explicit consent")
	}
	return lines
}

func (c *courseModel) view() string {
	rightW := c.width
	stagePanel := ""
	if c.stageW > 0 {
		rightW = c.width - c.stageW - 1
		stagePanel = c.theme.panel(
			c.theme.accentBold(fmt.Sprintf("stages · %d", len(c.course.stages))),
			c.stages.View(), c.stageW, c.height, c.focus == paneStages,
		)
	}

	descPanel := c.theme.panel(
		c.descTitle(), c.desc.View(), rightW, c.descH, c.focus == paneDescription,
	)
	right := descPanel
	if c.testH > 0 {
		tests := c.theme.panel(c.testTitle(), c.log.View(), rightW, c.testH, c.focus == paneTests)
		right = lipgloss.JoinVertical(lipgloss.Left, descPanel, "", tests)
	}
	if c.stageW > 0 {
		return lipgloss.JoinHorizontal(lipgloss.Top, stagePanel, " ", right)
	}
	return right
}

func (c *courseModel) descTitle() string {
	stage, ok := c.stage()
	if !ok {
		return c.theme.accentBold("stage")
	}
	title := fmt.Sprintf("stage %d/%d", c.stages.Index()+1, len(c.course.stages))
	return c.theme.accentBold(title) + c.theme.muted(" · "+stage.Name)
}

func (c *courseModel) testTitle() string {
	switch c.run {
	case runRunning:
		return c.theme.accentBold("tests") + c.theme.warn(" · running…")
	case runDone:
		return c.theme.accentBold("tests") + c.theme.good(" · done")
	default:
		return c.theme.accentBold("tests") + c.theme.muted(" · idle")
	}
}

func (c *courseModel) title() string {
	return c.theme.accentBold(c.course.provider) + c.theme.muted(" · ") + c.theme.bold(c.course.name)
}

func (c *courseModel) meta() string {
	meta := c.theme.muted(c.language())
	if _, ok := c.stage(); ok {
		meta += c.theme.muted(fmt.Sprintf(" · stage %d/%d", c.stages.Index()+1, len(c.course.stages)))
	}
	return meta
}

func difficultyLabel(d string) string {
	switch d {
	case "very_easy":
		return "very easy"
	case "easy", "medium", "hard":
		return d
	}
	return ""
}
