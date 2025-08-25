package ui

import "github.com/charmbracelet/lipgloss"

// FallBackText is a compact, ASCII-free fallback title for narrow terminals
var FallBackText = lipgloss.JoinVertical(lipgloss.Center,
    StyleTitle().Render("Claude Squad"),
)
