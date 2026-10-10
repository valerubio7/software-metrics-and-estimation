package tui

import "github.com/charmbracelet/lipgloss"

// Palette roles for the TUI base. Adaptive colors keep contrast on dark and
// light terminals; every state also carries a text cue, never color alone.
var (
	accent = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#A78BFA"}
	ink    = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	muted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	ok     = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
	danger = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	amber  = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	line   = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(muted)

	ruleStyle = lipgloss.NewStyle().
			Foreground(accent)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1E1B4B"}).
			Background(accent).
			Padding(0, 2)

	itemStyle = lipgloss.NewStyle().
			Foreground(ink).
			Padding(0, 2)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	hintKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ink)

	hintStyle = lipgloss.NewStyle().
			Foreground(muted)

	statusOkStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ok)

	errorBannerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(danger).
				Foreground(danger).
				Bold(true).
				Padding(0, 1)

	errorHintStyle = lipgloss.NewStyle().
			Foreground(muted)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#451A03"}).
			Background(amber).
			Padding(0, 1)

	screenTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ink)
)
