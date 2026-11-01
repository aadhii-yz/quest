package main

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// theme is the single palette shared by every component. It is a pointer so
// that a background-colour change (tea.BackgroundColorMsg) updates all views
// at once.
type theme struct {
	isDark bool
}

func newTheme(isDark bool) *theme { return &theme{isDark: isDark} }

func (t *theme) pick(light, dark string) color.Color {
	return lipgloss.LightDark(t.isDark)(lipgloss.Color(light), lipgloss.Color(dark))
}

func (t *theme) accentColor() color.Color   { return t.pick("#0969da", "#7dc4ff") }
func (t *theme) mutedColor() color.Color    { return t.pick("#57606a", "#8b93a7") }
func (t *theme) goodColor() color.Color     { return t.pick("#1a7f37", "#7ee787") }
func (t *theme) warnColor() color.Color     { return t.pick("#9a6700", "#e3b341") }
func (t *theme) badColor() color.Color      { return t.pick("#cf222e", "#ff7b72") }
func (t *theme) hairlineColor() color.Color { return t.pick("#d0d7de", "#30363d") }
func (t *theme) selectedBg() color.Color    { return t.pick("#ddf4ff", "#1f2b3d") }
func (t *theme) selectedFg() color.Color    { return t.pick("#0a3069", "#cfe3ff") }

func styleWith(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

func (t *theme) accent(s string) string { return styleWith(t.accentColor()).Render(s) }
func (t *theme) muted(s string) string  { return styleWith(t.mutedColor()).Render(s) }
func (t *theme) good(s string) string   { return styleWith(t.goodColor()).Render(s) }
func (t *theme) warn(s string) string   { return styleWith(t.warnColor()).Render(s) }
func (t *theme) bad(s string) string    { return styleWith(t.badColor()).Render(s) }

func (t *theme) bold(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

func (t *theme) accentBold(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(t.accentColor()).Render(s)
}

// fit truncates s to at most w cells, then pads it to exactly w. It is meant
// for single lines: MaxWidth alone truncates, whereas Width would wrap.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = lipgloss.NewStyle().MaxWidth(w).Render(s)
	if pad := w - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// panelInner returns the content width and body height available inside a
// bordered panel of w x h: the frame takes two rows/columns and one title line
// is reserved above the body.
func panelInner(w, h int) (contentW, bodyH int) {
	return max(w-4, 1), max(h-3, 0)
}

// panel renders a bordered pane with a title line. The body must already be
// rendered at contentW x bodyH (use panelInner); it is clipped if it overflows.
func (t *theme) panel(title, body string, w, h int, focused bool) string {
	if w < 8 || h < 4 {
		return lipgloss.NewStyle().MaxWidth(max(w, 0)).MaxHeight(max(h, 0)).Render(body)
	}
	border := t.hairlineColor()
	if focused {
		border = t.accentColor()
	}
	// In Lip Gloss v2 Width/Height are the total block size, including border
	// and padding. The inner content is therefore w-4 cells wide and h-2 rows
	// tall, one row of which is the title line.
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(w).
		Height(h).
		MaxHeight(h).
		Render(fit(title, w-4) + "\n" + body)
}

// bar lays out a left and right label on one full width line.
func bar(left, right string, w int) string {
	if w <= 0 {
		return ""
	}
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return fit(left+" "+right, w)
	}
	return left + strings.Repeat(" ", gap) + right
}

// markdown renders course-definition Markdown with syntax highlighting and
// wrapping for the current background.
func (t *theme) markdown(src string, width int) string {
	if strings.TrimSpace(src) == "" {
		return ""
	}
	style := "light"
	if t.isDark {
		style = "dark"
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(max(width-4, 20)),
	)
	if err != nil {
		return src
	}
	out, err := renderer.Render(src)
	if err != nil {
		return src
	}
	return strings.Trim(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
}

// newHelp returns a help model styled with the current palette.
func (t *theme) newHelp() help.Model {
	h := help.New()
	h.Styles = help.DefaultStyles(t.isDark)
	h.Styles.ShortKey = styleWith(t.accentColor())
	h.Styles.ShortDesc = styleWith(t.mutedColor())
	h.Styles.ShortSeparator = styleWith(t.hairlineColor())
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.FullSeparator = h.Styles.ShortSeparator
	h.Styles.Ellipsis = h.Styles.ShortSeparator
	return h
}

// rowItem is the small interface every list in the UI implements so the same
// one-line delegate can render them all.
type rowItem interface {
	list.Item
	// row returns the label and an optional right-aligned badge.
	row() (label, badge string)
}

// rowDelegate renders compact single-line rows. List chrome (title, status,
// pagination, help) is switched off by the owning model.
type rowDelegate struct {
	theme *theme
}

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(rowItem)
	if !ok {
		return
	}
	label, badge := it.row()
	width := max(m.Width(), 1)
	selected := index == m.Index()

	marker := "  "
	if selected {
		marker = "▸ "
	}
	line := marker + label
	if badge != "" {
		if gap := width - lipgloss.Width(line) - lipgloss.Width(badge); gap > 0 {
			line += strings.Repeat(" ", gap) + d.theme.muted(badge)
		}
	}
	line = fit(line, width)
	if selected {
		// Strip nested styles so the selection background covers the whole row.
		line = lipgloss.NewStyle().
			Foreground(d.theme.selectedFg()).
			Background(d.theme.selectedBg()).
			Render(ansi.Strip(line))
	}
	fmt.Fprint(w, line)
}

// newRowList builds a chrome-free list of single-line rows.
func newRowList(items []list.Item, t *theme, filtering bool, noun string) list.Model {
	l := list.New(items, rowDelegate{theme: t}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(filtering)
	l.SetStatusBarItemName(noun, noun)
	l.Styles.NoItems = styleWith(t.mutedColor())
	l.Styles.Filter.Focused.Prompt = styleWith(t.accentColor())
	l.Styles.Filter.Focused.Text = lipgloss.NewStyle()
	l.Styles.Filter.Blurred.Prompt = styleWith(t.mutedColor())
	l.Styles.DefaultFilterCharacterMatch = lipgloss.NewStyle().
		Foreground(t.accentColor()).
		Underline(true)
	return l
}

// logLine colours a test-runner log line by its prefix.
func (t *theme) logLine(line string) string {
	switch {
	case strings.HasPrefix(line, "$ "):
		return t.accent(line)
	case strings.HasPrefix(line, "PASS"), strings.HasPrefix(line, "ok "):
		return t.good(line)
	case strings.HasPrefix(line, "FAIL"):
		return t.bad(line)
	case strings.HasPrefix(line, "warning:"):
		return t.warn(line)
	case strings.HasPrefix(line, "note:"), strings.HasPrefix(line, "["):
		return t.muted(line)
	}
	return line
}
