package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

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

// clampInt constrains v to [lo, hi], tolerating an empty range.
func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return min(max(v, lo), hi)
}
