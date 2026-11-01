package ui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

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
