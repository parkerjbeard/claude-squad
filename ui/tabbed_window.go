package ui

import (
    "claude-squad/log"
    "claude-squad/session"
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
    inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
    activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
    highlightColor    = Theme.Accent
    inactiveTabStyle  = lipgloss.NewStyle().
                        Border(inactiveTabBorder, true).
                        BorderForeground(highlightColor).
                        AlignHorizontal(lipgloss.Center)
    activeTabStyle = inactiveTabStyle.
                        Border(activeTabBorder, true).
                        AlignHorizontal(lipgloss.Center)
    windowStyle = lipgloss.NewStyle().
                        BorderForeground(highlightColor).
                        Border(lipgloss.NormalBorder(), false, true, true, true)
)

const (
	PreviewTab int = iota
	DiffTab
)

type Tab struct {
	Name   string
	Render func(width int, height int) string
}

// TabbedWindow has tabs at the top of a pane which can be selected. The tabs
// take up one rune of height.
type TabbedWindow struct {
	tabs []string

	activeTab int
	height    int
	width     int

	preview  *PreviewPane
	diff     *DiffPane
	instance *session.Instance
}

func NewTabbedWindow(preview *PreviewPane, diff *DiffPane) *TabbedWindow {
	return &TabbedWindow{
		tabs: []string{
			"Preview",
			"Diff",
		},
		preview: preview,
		diff:    diff,
	}
}

func (w *TabbedWindow) SetInstance(instance *session.Instance) {
    w.instance = instance
}

func (w *TabbedWindow) SetSize(width, height int) {
    w.width = width
    w.height = height

    // Calculate the content height by subtracting:
    // 1. Tab height (including border and padding)
    // 2. Window style vertical frame size
	// 3. Additional padding/spacing (2 for the newline and spacing)
	tabHeight := activeTabStyle.GetVerticalFrameSize() + 1
	contentHeight := height - tabHeight - windowStyle.GetVerticalFrameSize() - 2
	contentWidth := w.width - windowStyle.GetHorizontalFrameSize()

	w.preview.SetSize(contentWidth, contentHeight)
    w.diff.SetSize(contentWidth, contentHeight)
}

func (w *TabbedWindow) GetPreviewSize() (width, height int) {
	return w.preview.width, w.preview.height
}

func (w *TabbedWindow) Toggle() {
	w.activeTab = (w.activeTab + 1) % len(w.tabs)
}

// ToggleWithReset toggles the tab and resets preview pane to normal mode
func (w *TabbedWindow) ToggleWithReset(instance *session.Instance) error {
	// Reset preview pane to normal mode before switching
	if err := w.preview.ResetToNormalMode(instance); err != nil {
		return err
	}
	w.activeTab = (w.activeTab + 1) % len(w.tabs)
	return nil
}

// UpdatePreview updates the content of the preview pane. instance may be nil.
func (w *TabbedWindow) UpdatePreview(instance *session.Instance) error {
	if w.activeTab != PreviewTab {
		return nil
	}
	return w.preview.UpdateContent(instance)
}

func (w *TabbedWindow) UpdateDiff(instance *session.Instance) {
    if w.activeTab != DiffTab {
        return
    }
    w.diff.SetDiff(instance)
    if instance != nil && instance.GetDiffStats() != nil && instance.GetDiffStats().Error == nil {
        stats := instance.GetDiffStats()
        w.SetTabCounts(stats.Added, stats.Removed)
    } else {
        w.SetTabCounts(0, 0)
    }
}

// ResetPreviewToNormalMode resets the preview pane to normal mode
func (w *TabbedWindow) ResetPreviewToNormalMode(instance *session.Instance) error {
	return w.preview.ResetToNormalMode(instance)
}

// Add these new methods for handling scroll events
func (w *TabbedWindow) ScrollUp() {
    if w.activeTab == PreviewTab {
        err := w.preview.ScrollUp(w.instance)
        if err != nil {
            log.InfoLog.Printf("tabbed window failed to scroll up: %v", err)
        }
    } else {
        w.diff.ScrollUp()
    }
}

func (w *TabbedWindow) ScrollDown() {
    if w.activeTab == PreviewTab {
        err := w.preview.ScrollDown(w.instance)
        if err != nil {
            log.InfoLog.Printf("tabbed window failed to scroll down: %v", err)
        }
    } else {
        w.diff.ScrollDown()
    }
}

// PageUp scrolls a page in the active pane
func (w *TabbedWindow) PageUp() {
    if w.activeTab == PreviewTab {
        if err := w.preview.PageUp(w.instance); err != nil {
            log.InfoLog.Printf("tabbed window page up failed: %v", err)
        }
    } else {
        w.diff.PageUp()
    }
}

// PageDown scrolls a page in the active pane
func (w *TabbedWindow) PageDown() {
    if w.activeTab == PreviewTab {
        if err := w.preview.PageDown(w.instance); err != nil {
            log.InfoLog.Printf("tabbed window page down failed: %v", err)
        }
    } else {
        w.diff.PageDown()
    }
}

// HalfPageUp scrolls half a page in the active pane
func (w *TabbedWindow) HalfPageUp() {
    if w.activeTab == PreviewTab {
        if err := w.preview.HalfPageUp(w.instance); err != nil {
            log.InfoLog.Printf("tabbed window half page up failed: %v", err)
        }
    } else {
        w.diff.HalfPageUp()
    }
}

// HalfPageDown scrolls half a page in the active pane
func (w *TabbedWindow) HalfPageDown() {
    if w.activeTab == PreviewTab {
        if err := w.preview.HalfPageDown(w.instance); err != nil {
            log.InfoLog.Printf("tabbed window half page down failed: %v", err)
        }
    } else {
        w.diff.HalfPageDown()
    }
}

// GotoTop moves to top in the active pane
func (w *TabbedWindow) GotoTop() {
    if w.activeTab == PreviewTab {
        if err := w.preview.GotoTop(w.instance); err != nil {
            log.InfoLog.Printf("tabbed window goto top failed: %v", err)
        }
    } else {
        w.diff.GotoTop()
    }
}

// GotoBottom moves to bottom in the active pane
func (w *TabbedWindow) GotoBottom() {
    if w.activeTab == PreviewTab {
        if err := w.preview.GotoBottom(w.instance); err != nil {
            log.InfoLog.Printf("tabbed window goto bottom failed: %v", err)
        }
    } else {
        w.diff.GotoBottom()
    }
}

// JumpNextHunk moves to next hunk in diff pane
func (w *TabbedWindow) JumpNextHunk() { if w.activeTab == DiffTab { w.diff.JumpNextHunk() } }

// JumpPrevHunk moves to previous hunk in diff pane
func (w *TabbedWindow) JumpPrevHunk() { if w.activeTab == DiffTab { w.diff.JumpPrevHunk() } }

// JumpNextFile moves to next file in diff pane
func (w *TabbedWindow) JumpNextFile() { if w.activeTab == DiffTab { w.diff.JumpNextFile() } }

// JumpPrevFile moves to previous file in diff pane
func (w *TabbedWindow) JumpPrevFile() { if w.activeTab == DiffTab { w.diff.JumpPrevFile() } }

// IsInDiffTab returns true if the diff tab is currently active
func (w *TabbedWindow) IsInDiffTab() bool {
	return w.activeTab == 1
}

// GetActiveTab returns the currently active tab index.
func (w *TabbedWindow) GetActiveTab() int {
	return w.activeTab
}

// IsPreviewInScrollMode returns true if the preview pane is in scroll mode
func (w *TabbedWindow) IsPreviewInScrollMode() bool {
	return w.preview.isScrolling
}

func (w *TabbedWindow) String() string {
	if w.width == 0 || w.height == 0 {
		return ""
	}

	var renderedTabs []string

	tabWidth := w.width / len(w.tabs)
	lastTabWidth := w.width - tabWidth*(len(w.tabs)-1)
	tabHeight := activeTabStyle.GetVerticalFrameSize() + 1 // get padding border margin size + 1 for character height

	for i, t := range w.tabs {
		width := tabWidth
		if i == len(w.tabs)-1 {
			width = lastTabWidth
		}

		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(w.tabs)-1, i == w.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		style = style.Width(width - 1)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	var content string
	if w.activeTab == 0 {
		content = w.preview.String()
	} else {
		content = w.diff.String()
	}
	window := windowStyle.Render(
		lipgloss.Place(
			w.width, w.height-2-windowStyle.GetVerticalFrameSize()-tabHeight,
			lipgloss.Left, lipgloss.Top, content))

    return lipgloss.JoinVertical(lipgloss.Left, "\n", row, window)
}

// SetTabCounts updates the Diff tab label to include counts when non-zero
func (w *TabbedWindow) SetTabCounts(added, removed int) {
    if len(w.tabs) < 2 {
        return
    }
    if added <= 0 && removed <= 0 {
        w.tabs[1] = "Diff"
        return
    }
    w.tabs[1] = lipgloss.JoinHorizontal(lipgloss.Left,
        "Diff ",
        StyleOk().Render(fmt.Sprintf("+%d", added)),
        " / ",
        StyleDanger().Render(fmt.Sprintf("-%d", removed)),
    )
}

// HitTestTab returns the tab index for a coordinate inside the tab row area.
// x,y are relative to the TabbedWindow's top-left corner.
func (w *TabbedWindow) HitTestTab(x, y int) (idx int, ok bool) {
    if w.width == 0 || len(w.tabs) == 0 {
        return 0, false
    }
    tabHeight := activeTabStyle.GetVerticalFrameSize() + 1
    // Row is rendered after a leading newline in String(), so tabs start at y==1
    if y < 1 || y > 1+tabHeight {
        return 0, false
    }
    per := w.width / len(w.tabs)
    if per <= 0 {
        return 0, false
    }
    idx = min(x/per, len(w.tabs)-1)
    return idx, true
}
