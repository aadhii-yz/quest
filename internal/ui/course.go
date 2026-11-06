package ui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aadhii-yz/quest/internal/provider"
	"github.com/aadhii-yz/quest/internal/runner"
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
	stage provider.Stage
}

func (i stageItem) FilterValue() string { return i.stage.Name }
func (i stageItem) row() (string, string) {
	return fmt.Sprintf("%2d %s", i.stage.Number, i.stage.Name), difficultyLabel(i.stage.Difficulty)
}

// courseModel is the course screen: a stage list, a description viewport and a
// test output viewport, with the panes independently toggleable.
type courseModel struct {
	theme   *theme
	course  provider.Course
	runner  runner.Runner
	langIdx int

	stages list.Model
	desc   viewport.Model
	log    viewport.Model

	logLines []string
	events   <-chan runner.Event
	cancel   context.CancelFunc
	run      runState

	showStages bool
	showTests  bool
	focus      int

	width, height int
	stageW        int
	descH, testH  int

	descKey  string
	descSlug string

	loading bool
	// pendingStage is the stage to select once a content bundle arrives.
	pendingStage string
}

func newCourseModel(t *theme, c provider.Course, langIdx int, r runner.Runner) *courseModel {
	if len(c.Languages) > 0 {
		langIdx = clampInt(langIdx, 0, len(c.Languages)-1)
	}
	items := make([]list.Item, 0, len(c.Stages))
	for _, stage := range c.Stages {
		items = append(items, stageItem{stage: stage})
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
		runner:     r,
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

// beginLoading marks the course as waiting for a content bundle.
func (c *courseModel) beginLoading() {
	c.loading = true
	c.descKey = ""
	c.refreshDescription(true)
}

// finishLoading clears the loading state after a failed load.
func (c *courseModel) finishLoading() {
	c.loading = false
	c.descKey = ""
	c.refreshDescription(true)
}

// setCourse swaps in a provider.Course after its content was loaded, keeping
// the selected stage and language where they still exist.
func (c *courseModel) setCourse(cr provider.Course) {
	stageID := c.pendingStage
	if stage, ok := c.stage(); ok {
		stageID = stage.ID
	}
	c.pendingStage = ""
	lang := c.language()

	c.course = cr
	c.loading = false
	for i, l := range cr.Languages {
		if l == lang {
			c.langIdx = i
			break
		}
	}

	items := make([]list.Item, 0, len(cr.Stages))
	for _, stage := range cr.Stages {
		items = append(items, stageItem{stage: stage})
	}
	c.stages.SetItems(items)
	idx := 0
	for i, stage := range cr.Stages {
		if stage.ID == stageID {
			idx = i
			break
		}
	}
	if len(items) > 0 {
		c.stages.Select(idx)
	}
	c.descKey, c.descSlug = "", ""
	c.refreshDescription(true)
	c.appendLog(fmt.Sprintf("content loaded: %d stages", len(cr.Stages)))
	c.layout(c.width, c.height)
}

// selectStageByID moves the stage cursor to the given stage when present.
func (c *courseModel) selectStageByID(id string) {
	if id == "" {
		return
	}
	for i, item := range c.stages.Items() {
		stage, ok := item.(stageItem)
		if ok && stage.stage.ID == id {
			c.stages.Select(i)
			c.syncStage()
			return
		}
	}
}

func (c *courseModel) language() string {
	if len(c.course.Languages) == 0 {
		return ""
	}
	return c.course.Languages[clampInt(c.langIdx, 0, len(c.course.Languages)-1)]
}

func (c *courseModel) stage() (provider.Stage, bool) {
	if item, ok := c.stages.SelectedItem().(stageItem); ok {
		return item.stage, true
	}
	return provider.Stage{}, false
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
	if n := len(c.course.Languages); n > 0 {
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

	key := fmt.Sprintf("%d|%s|%t", c.desc.Width(), stage.ID, c.theme.isDark)
	if key == c.descKey {
		return
	}
	stageChanged := stage.ID != c.descSlug
	c.descKey, c.descSlug = key, stage.ID

	var b strings.Builder
	meta := make([]string, 0, 3)
	if d := difficultyLabel(stage.Difficulty); d != "" {
		meta = append(meta, d)
	}
	if stage.Extension != "" {
		meta = append(meta, "extension "+stage.Extension)
	}
	meta = append(meta, "slug "+stage.ID)
	b.WriteString(c.theme.muted(strings.Join(meta, " · ")))
	b.WriteString("\n\n")
	b.WriteString(c.theme.markdown(stage.Description, c.desc.Width()))

	c.desc.SetContent(b.String())
	if stageChanged || force {
		c.desc.GotoTop()
	}
}

func (c *courseModel) emptyDescription() string {
	var b strings.Builder
	if c.loading {
		b.WriteString(c.theme.bold("Loading content…"))
		b.WriteString("\n\n")
		b.WriteString(c.theme.muted("Fetching and verifying the pinned quest-source bundle."))
		return b.String()
	}
	b.WriteString(c.theme.bold("No stage list loaded"))
	b.WriteString("\n\n")
	if c.course.Note != "" {
		b.WriteString(c.theme.muted(c.course.Note))
	} else {
		b.WriteString(c.theme.muted("This course has no stage list attached yet."))
	}
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

// startRun asks the runner for an event stream. The simulated runner is the
// only implementation today; real execution must go through cache verification
// and the licensing gate first.
func (c *courseModel) startRun() tea.Cmd {
	if c.run == runRunning || c.runner == nil {
		return nil
	}
	stage, ok := c.stage()
	if !ok {
		c.appendLog("FAIL  no stage selected")
		return nil
	}

	c.stopRun()
	ctx, cancel := context.WithCancel(context.Background())
	events, err := c.runner.Run(ctx, runner.Request{
		CourseID: c.course.ID,
		Stage:    stage,
		Language: c.language(),
		Tester:   c.course.Tester,
	})
	if err != nil {
		cancel()
		c.run = runDone
		c.appendLog("FAIL  " + err.Error())
		return nil
	}
	c.cancel = cancel
	c.events = events
	c.run = runRunning
	c.appendLog("")
	return waitForRunEvent(events)
}

type runEventMsg runner.Event

func waitForRunEvent(events <-chan runner.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return runEventMsg(runner.Event{Done: true})
		}
		return runEventMsg(ev)
	}
}

// handleEvent appends runner output and re-arms the wait until the stream ends.
func (c *courseModel) handleEvent(msg runEventMsg) tea.Cmd {
	ev := runner.Event(msg)
	if ev.Line != "" {
		c.appendLog(ev.Line)
	}
	if ev.Done {
		c.run = runDone
		c.stopRun()
		return nil
	}
	if c.events == nil {
		c.run = runDone
		return nil
	}
	return waitForRunEvent(c.events)
}

// stopRun cancels an in-flight run and drops the event stream.
func (c *courseModel) stopRun() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.events = nil
}

func (c *courseModel) view() string {
	rightW := c.width
	stagePanel := ""
	if c.stageW > 0 {
		rightW = c.width - c.stageW - 1
		stagePanel = c.theme.panel(
			c.theme.accentBold(fmt.Sprintf("stages · %d", len(c.course.Stages))),
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
		if c.loading {
			return c.theme.accentBold("stage") + c.theme.warn(" · loading content…")
		}
		return c.theme.accentBold("stage")
	}
	title := fmt.Sprintf("stage %d/%d", stage.Number, len(c.course.Stages))
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
	return c.theme.accentBold(c.course.ProviderName) + c.theme.muted(" · ") + c.theme.bold(c.course.Name)
}

func (c *courseModel) meta() string {
	meta := c.theme.muted(c.language())
	if stage, ok := c.stage(); ok {
		meta += c.theme.muted(fmt.Sprintf(" · stage %d/%d", stage.Number, len(c.course.Stages)))
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
