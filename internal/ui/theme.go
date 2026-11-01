package ui

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
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
