package ui

import "github.com/charmbracelet/lipgloss"

// Palette centralizes adaptive colors for the TUI
type Palette struct {
    Fg        lipgloss.AdaptiveColor
    FgMuted   lipgloss.AdaptiveColor
    BgAlt     lipgloss.AdaptiveColor
    Accent    lipgloss.AdaptiveColor
    AccentAlt lipgloss.AdaptiveColor
    Ok        lipgloss.AdaptiveColor
    Warn      lipgloss.AdaptiveColor
    Danger    lipgloss.AdaptiveColor
    Hint      lipgloss.AdaptiveColor
}

// Theme holds the shared color palette
var Theme = Palette{
    Fg:        lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#e5e5e5"},
    FgMuted:   lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"},
    BgAlt:     lipgloss.AdaptiveColor{Light: "#f3f4f6", Dark: "#1f2937"},
    Accent:    lipgloss.AdaptiveColor{Light: "#6e79d8", Dark: "#8ea2ff"},
    AccentAlt: lipgloss.AdaptiveColor{Light: "#a78bfa", Dark: "#b79bff"},
    Ok:        lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#22c55e"},
    Warn:      lipgloss.AdaptiveColor{Light: "#f59e0b", Dark: "#fbbf24"},
    Danger:    lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#ef4444"},
    Hint:      lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"},
}

// Style helpers
func StyleTitle() lipgloss.Style {
    return lipgloss.NewStyle().Bold(true).Foreground(Theme.Accent)
}

func StyleMuted() lipgloss.Style { return lipgloss.NewStyle().Foreground(Theme.FgMuted) }

func StyleOk() lipgloss.Style { return lipgloss.NewStyle().Foreground(Theme.Ok) }

func StyleDanger() lipgloss.Style { return lipgloss.NewStyle().Foreground(Theme.Danger) }

func StyleWarn() lipgloss.Style { return lipgloss.NewStyle().Foreground(Theme.Warn) }

func StyleBadge() lipgloss.Style {
    return lipgloss.NewStyle().Foreground(Theme.Fg).Background(Theme.BgAlt).Padding(0, 1).Bold(true)
}

