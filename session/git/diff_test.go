package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestDiffNumstatAndFull verifies that Diff() uses numstat (counts only) and DiffFull() returns content
func TestDiffNumstatAndFull(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")

	// Initialize repo and make initial commit
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "a.txt"), []byte("one\n"), 0644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if _, err := wt.Add("a.txt"); err != nil {
		t.Fatalf("add a.txt: %v", err)
	}
	if _, err := wt.Commit("init", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@example.com"}}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Create GitWorktree (detached worktree) and set up
	gw, _, err := NewGitWorktree(repoPath, "sess")
	if err != nil {
		t.Fatalf("NewGitWorktree: %v", err)
	}
	if err := gw.Setup(); err != nil {
		t.Fatalf("setup worktree: %v", err)
	}

	// Modify tracked file and create untracked file in worktree
	if err := os.WriteFile(filepath.Join(gw.worktreePath, "a.txt"), []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatalf("modify a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gw.worktreePath, "new.txt"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatalf("write new.txt: %v", err)
	}

	// Numstat: should count only tracked file change (+1)
	ns := gw.Diff()
	if ns.Error != nil {
		t.Fatalf("numstat diff error: %v", ns.Error)
	}
	if ns.Added < 1 {
		t.Fatalf("expected at least 1 added line from tracked change, got %d", ns.Added)
	}
	if ns.Content != "" {
		t.Fatalf("expected no content in numstat diff, got some")
	}

	// Full diff: should include content and reflect both tracked (+1) and untracked (+2) additions
	fd := gw.DiffFull()
	if fd.Error != nil {
		t.Fatalf("full diff error: %v", fd.Error)
	}
	if fd.Content == "" {
		t.Fatalf("expected full diff content, got empty")
	}
	if fd.Added < 3 { // 1 from tracked change + 2 from new file
		t.Fatalf("expected at least 3 added lines, got %d", fd.Added)
	}
}
