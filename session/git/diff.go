package git

import (
	"strconv"
	"strings"
)

// DiffStats holds statistics about the changes in a diff
type DiffStats struct {
	// Content is the full diff content
	Content string
	// Added is the number of added lines
	Added int
	// Removed is the number of removed lines
	Removed int
	// Error holds any error that occurred during diff computation
	// This allows propagating setup errors (like missing base commit) without breaking the flow
	Error error
}

func (d *DiffStats) IsEmpty() bool {
	return d.Added == 0 && d.Removed == 0 && d.Content == ""
}

// Diff returns the git diff between the worktree and the base branch along with statistics
func (g *GitWorktree) Diff() *DiffStats {
	// Lightweight counts-only diff using numstat. Does not include untracked files.
	stats := &DiffStats{}
	output, err := g.runGitCommand(g.worktreePath, "--no-pager", "diff", "--numstat", "--no-ext-diff", g.GetBaseCommitSHA())
	if err != nil {
		stats.Error = err
		return stats
	}
	if strings.TrimSpace(output) == "" {
		return stats
	}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		// Format: added\tremoved\tpath
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		// Binary files show '-' instead of numbers
		if parts[0] != "-" {
			if v, e := strconv.Atoi(parts[0]); e == nil {
				stats.Added += v
			}
		}
		if parts[1] != "-" {
			if v, e := strconv.Atoi(parts[1]); e == nil {
				stats.Removed += v
			}
		}
	}
	return stats
}

// DiffFull returns the full diff content and statistics. This operation is more expensive
// and will stage untracked files with intent-to-add to include them in the diff.
func (g *GitWorktree) DiffFull() *DiffStats {
	stats := &DiffStats{}

	// Stage untracked files with intent-to-add so they appear in the diff output
	if _, err := g.runGitCommand(g.worktreePath, "add", "-N", "."); err != nil {
		stats.Error = err
		return stats
	}

	content, err := g.runGitCommand(g.worktreePath, "--no-pager", "diff", g.GetBaseCommitSHA())
	if err != nil {
		stats.Error = err
		return stats
	}
	// Count additions/removals from content body
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.Added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.Removed++
		}
	}
	stats.Content = content
	return stats
}
