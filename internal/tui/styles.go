package tui

import "github.com/charmbracelet/lipgloss"

// Palette roles for the TUI. Adaptive colors keep contrast on dark and light
// terminals; every state also carries a text cue, never color alone.
var (
	accent     = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	accentDeep = lipgloss.AdaptiveColor{Light: "#4C1D95", Dark: "#7C3AED"}
	ink        = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#E5E7EB"}
	inkFaint   = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"}
	muted      = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"}
	ok         = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
	warn       = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	danger     = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	info       = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"}
	line       = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#3F3F46"}
	onAccent   = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1E1B4B"}
)

var (
	// Chrome.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ink)

	brandStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(muted)

	ruleStyle = lipgloss.NewStyle().
			Foreground(line)

	crumbStyle = lipgloss.NewStyle().
			Foreground(inkFaint)

	crumbSepStyle = lipgloss.NewStyle().
			Foreground(muted)

	crumbCurrentStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accent)

	// Content.
	screenTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ink)

	sectionLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(muted)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	itemStyle = lipgloss.NewStyle().
			Foreground(ink).
			Padding(0, 2)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(onAccent).
			Background(accent).
			Padding(0, 2)

	// Tables.
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(muted)

	tableCellStyle = lipgloss.NewStyle().
			Foreground(ink)

	tableMetaStyle = lipgloss.NewStyle().
			Foreground(inkFaint)

	// Status and feedback.
	statusOkStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ok)

	statusMutedStyle = lipgloss.NewStyle().
				Foreground(muted)

	errorBannerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(danger).
				Foreground(danger).
				Bold(true).
				Padding(0, 1)

	successBannerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ok).
				Foreground(ok).
				Padding(0, 1)

	warnBannerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warn).
			Foreground(warn).
			Padding(0, 1)

	errorHintStyle = lipgloss.NewStyle().
			Foreground(muted)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(onAccent).
			Background(warn).
			Padding(0, 1)

	// Footer hints and keycaps.
	hintKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ink)

	hintStyle = lipgloss.NewStyle().
			Foreground(muted)

	kbdStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ink).
			Background(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#27272A"}).
			Padding(0, 1)

	// Panels and empty states.
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(line).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	emptyTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(inkFaint)

	emptyHintStyle = lipgloss.NewStyle().
			Foreground(muted)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)
)
