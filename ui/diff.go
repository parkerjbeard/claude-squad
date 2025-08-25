package ui

import (
	"claude-squad/session"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
    AdditionStyle = StyleOk()
    DeletionStyle = StyleDanger()
    HunkStyle     = lipgloss.NewStyle().Foreground(Theme.Accent)
)

type DiffPane struct {
    viewport viewport.Model
    diff     string
    stats    string
    width    int
    height   int

    // navigation markers
    fileOffsets []int // line indexes within diff (not including stats header)
    hunkOffsets []int // line indexes within diff (not including stats header)
}

func NewDiffPane() *DiffPane {
	return &DiffPane{
		viewport: viewport.New(0, 0),
	}
}

func (d *DiffPane) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width
	d.viewport.Height = height
	// Update viewport content if diff exists
	if d.diff != "" || d.stats != "" {
		d.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, d.stats, d.diff))
	}
}

func (d *DiffPane) SetDiff(instance *session.Instance) {
	centeredFallbackMessage := lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		"No changes",
	)

	if instance == nil || !instance.Started() {
		d.viewport.SetContent(centeredFallbackMessage)
		return
	}

	stats := instance.GetDiffStats()
	if stats == nil {
		// Show loading message if worktree is not ready
		centeredMessage := lipgloss.Place(
			d.width,
			d.height,
			lipgloss.Center,
			lipgloss.Center,
			"Setting up worktree...",
		)
		d.viewport.SetContent(centeredMessage)
		return
	}

	if stats.Error != nil {
		// Show error message
		centeredMessage := lipgloss.Place(
			d.width,
			d.height,
			lipgloss.Center,
			lipgloss.Center,
			fmt.Sprintf("Error: %v", stats.Error),
		)
		d.viewport.SetContent(centeredMessage)
		return
	}

    if stats.IsEmpty() {
        d.stats = ""
        d.diff = ""
        d.viewport.SetContent(centeredFallbackMessage)
    } else {
        additions := AdditionStyle.Render(fmt.Sprintf("%d additions(+)", stats.Added))
        deletions := DeletionStyle.Render(fmt.Sprintf("%d deletions(-)", stats.Removed))
        d.stats = lipgloss.JoinHorizontal(lipgloss.Center, additions, " ", deletions)
        d.diff = colorizeDiff(stats.Content)
        d.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, d.stats, d.diff))
        d.parseMarkers()
    }
}

func (d *DiffPane) String() string {
    return d.viewport.View()
}

// ScrollUp scrolls the viewport up
func (d *DiffPane) ScrollUp() {
	d.viewport.LineUp(1)
}

// ScrollDown scrolls the viewport down
func (d *DiffPane) ScrollDown() {
    d.viewport.LineDown(1)
}

// PageUp scrolls up one page
func (d *DiffPane) PageUp() { d.viewport.LineUp(max(1, d.viewport.Height-1)) }

// PageDown scrolls down one page
func (d *DiffPane) PageDown() { d.viewport.LineDown(max(1, d.viewport.Height-1)) }

// HalfPageUp scrolls up half a page
func (d *DiffPane) HalfPageUp() { d.viewport.LineUp(max(1, d.viewport.Height/2)) }

// HalfPageDown scrolls down half a page
func (d *DiffPane) HalfPageDown() { d.viewport.LineDown(max(1, d.viewport.Height/2)) }

// GotoTop moves to the top of the diff content area
func (d *DiffPane) GotoTop() { d.viewport.GotoTop() }

// GotoBottom moves to the bottom of the diff content area
func (d *DiffPane) GotoBottom() { d.viewport.GotoBottom() }

// parseMarkers builds file and hunk line indices for navigation
func (d *DiffPane) parseMarkers() {
    d.fileOffsets = d.fileOffsets[:0]
    d.hunkOffsets = d.hunkOffsets[:0]
    if d.diff == "" {
        return
    }
    lines := strings.Split(d.diff, "\n")
    for i, line := range lines {
        if strings.HasPrefix(line, "diff --git ") {
            d.fileOffsets = append(d.fileOffsets, i)
            continue
        }
        if strings.HasPrefix(line, "@@") {
            d.hunkOffsets = append(d.hunkOffsets, i)
        }
    }
}

// JumpNextHunk moves the viewport to the next hunk header if present
func (d *DiffPane) JumpNextHunk() {
    if len(d.hunkOffsets) == 0 {
        return
    }
    headerLines := 0
    if d.stats != "" {
        headerLines = 1
    }
    cur := d.viewport.YOffset - headerLines
    if cur < 0 {
        cur = 0
    }
    for _, off := range d.hunkOffsets {
        if off > cur {
            d.viewport.SetYOffset(off + headerLines)
            return
        }
    }
    // wrap to last
    d.viewport.SetYOffset(d.hunkOffsets[len(d.hunkOffsets)-1] + headerLines)
}

// JumpPrevHunk moves the viewport to the previous hunk header if present
func (d *DiffPane) JumpPrevHunk() {
    if len(d.hunkOffsets) == 0 {
        return
    }
    headerLines := 0
    if d.stats != "" {
        headerLines = 1
    }
    cur := d.viewport.YOffset - headerLines
    if cur < 0 {
        cur = 0
    }
    for i := len(d.hunkOffsets) - 1; i >= 0; i-- {
        if d.hunkOffsets[i] < cur {
            d.viewport.SetYOffset(d.hunkOffsets[i] + headerLines)
            return
        }
    }
    // wrap to first
    d.viewport.SetYOffset(d.hunkOffsets[0] + headerLines)
}

// JumpNextFile moves to next file boundary
func (d *DiffPane) JumpNextFile() {
    if len(d.fileOffsets) == 0 {
        return
    }
    headerLines := 0
    if d.stats != "" {
        headerLines = 1
    }
    cur := d.viewport.YOffset - headerLines
    if cur < 0 {
        cur = 0
    }
    for _, off := range d.fileOffsets {
        if off > cur {
            d.viewport.SetYOffset(off + headerLines)
            return
        }
    }
    d.viewport.SetYOffset(d.fileOffsets[len(d.fileOffsets)-1] + headerLines)
}

// JumpPrevFile moves to previous file boundary
func (d *DiffPane) JumpPrevFile() {
    if len(d.fileOffsets) == 0 {
        return
    }
    headerLines := 0
    if d.stats != "" {
        headerLines = 1
    }
    cur := d.viewport.YOffset - headerLines
    if cur < 0 {
        cur = 0
    }
    for i := len(d.fileOffsets) - 1; i >= 0; i-- {
        if d.fileOffsets[i] < cur {
            d.viewport.SetYOffset(d.fileOffsets[i] + headerLines)
            return
        }
    }
    d.viewport.SetYOffset(d.fileOffsets[0] + headerLines)
}
func colorizeDiff(diff string) string {
	b := getBuilder()
	defer putBuilder(b)

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if len(line) > 0 {
			if strings.HasPrefix(line, "@@") {
				// Color hunk headers cyan
				b.WriteString(HunkStyle.Render(line))
				b.WriteByte('\n')
			} else if line[0] == '+' && (len(line) == 1 || line[1] != '+') {
				// Color added lines green, excluding metadata like '+++'
				b.WriteString(AdditionStyle.Render(line))
				b.WriteByte('\n')
			} else if line[0] == '-' && (len(line) == 1 || line[1] != '-') {
				// Color removed lines red, excluding metadata like '---'
				b.WriteString(DeletionStyle.Render(line))
				b.WriteByte('\n')
			} else {
				// Print metadata and unchanged lines without color
				b.WriteString(line)
				b.WriteByte('\n')
			}
		} else {
			// Preserve empty lines
			b.WriteByte('\n')
		}
	}

	return b.String()
}
