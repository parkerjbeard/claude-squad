package ui

import (
    "github.com/charmbracelet/lipgloss"
    "strings"
)

type ErrBox struct {
    height, width int
    err           error
}

var errStyle = lipgloss.NewStyle().Foreground(Theme.Danger)

func NewErrBox() *ErrBox {
	return &ErrBox{}
}

func (e *ErrBox) SetError(err error) {
	e.err = err
}

func (e *ErrBox) Clear() {
	e.err = nil
}

func (e *ErrBox) SetSize(width, height int) {
	e.width = width
	e.height = height
}

func (e *ErrBox) String() string {
    var err string
    if e.err != nil {
        err = wrapErrorSingleLine(e.err.Error(), e.width)
    }
    return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, errStyle.Render(err))
}

// wrapErrorSingleLine wraps to a single line with ellipsis.
func wrapErrorSingleLine(s string, width int) string {
    if width <= 0 {
        return ""
    }
    // Collapse newlines and excessive spaces
    s = strings.ReplaceAll(s, "\n", " ")
    s = strings.Join(strings.Fields(s), " ")
    // Reserve 1 char for ellipsis when truncating
    if len(s) <= width {
        return s
    }
    limit := maxInt(1, width-1)
    cut := limit
    // Try break at last space before limit
    if idx := strings.LastIndex(s[:limit], " "); idx > 0 {
        cut = idx
    }
    return s[:cut] + "…"
}

func maxInt(a, b int) int { if a > b { return a }; return b }
