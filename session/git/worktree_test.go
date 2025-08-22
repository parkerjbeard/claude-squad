package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestNewDirectGitWorktree tests creating a direct mode worktree
func TestNewDirectGitWorktree(t *testing.T) {
	// Create a temporary directory for the test repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	// Create an initial commit
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := wt.Add("test.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Test creating a direct mode worktree
	tests := []struct {
		name       string
		branchName string
		wantErr    bool
	}{
		{
			name:       "create direct worktree on main",
			branchName: "main",
			wantErr:    false,
		},
		{
			name:       "create direct worktree on new branch",
			branchName: "feature-test",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktree, err := NewDirectGitWorktree(repoPath, tt.branchName, "test-session")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDirectGitWorktree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the worktree is in direct mode
				if !worktree.DirectMode {
					t.Error("expected DirectMode to be true")
				}

				// Verify the paths are set correctly
				if worktree.repoPath != repoPath {
					t.Errorf("expected repoPath %s, got %s", repoPath, worktree.repoPath)
				}

				if worktree.worktreePath != repoPath {
					t.Errorf("expected worktreePath %s, got %s", repoPath, worktree.worktreePath)
				}

				// Verify the branch name is set
				if worktree.branchName != tt.branchName {
					t.Errorf("expected branchName %s, got %s", tt.branchName, worktree.branchName)
				}
			}
		})
	}
}

// TestSetupDirect tests setting up a direct mode session
func TestSetupDirect(t *testing.T) {
	// Create a temporary directory for the test repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	// Create an initial commit on master
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := wt.Add("test.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create main branch (git init creates master by default)
	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/main",
		Create: true,
	}); err != nil {
		t.Fatalf("failed to create main branch: %v", err)
	}

	tests := []struct {
		name       string
		branchName string
		expectNew  bool
		wantErr    bool
	}{
		{
			name:       "setup on existing main branch",
			branchName: "main",
			expectNew:  false,
			wantErr:    false,
		},
		{
			name:       "setup on new feature branch",
			branchName: "feature-new",
			expectNew:  true,
			wantErr:    false,
		},
		{
			name:       "setup on master should fail if not exists",
			branchName: "develop",
			expectNew:  false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktree := &GitWorktree{
				repoPath:     repoPath,
				worktreePath: repoPath,
				branchName:   tt.branchName,
				DirectMode:   true,
				sessionName:  "test-session",
			}

			err := worktree.SetupDirect()
			if (err != nil) != tt.wantErr {
				t.Errorf("SetupDirect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify we're on the correct branch
				currentBranch, err := getCurrentBranch(repoPath)
				if err != nil {
					t.Errorf("failed to get current branch: %v", err)
				}
				if currentBranch != tt.branchName {
					t.Errorf("expected to be on branch %s, but on %s", tt.branchName, currentBranch)
				}
			}
		})
	}
}

// TestCleanupDirect tests cleanup for direct mode sessions
func TestCleanupDirect(t *testing.T) {
	// Create a temporary directory for the test repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	// Create an initial commit
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := wt.Add("test.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create main branch
	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/main",
		Create: true,
	}); err != nil {
		t.Fatalf("failed to create main branch: %v", err)
	}

	// Create a direct mode worktree
	worktree := &GitWorktree{
		repoPath:       repoPath,
		worktreePath:   repoPath,
		branchName:     "feature-test",
		DirectMode:     true,
		sessionName:    "test-session",
		OriginalBranch: "main",
	}

	// Setup the direct mode worktree
	if err := worktree.SetupDirect(); err != nil {
		t.Fatalf("failed to setup direct worktree: %v", err)
	}

	// Verify we're on the feature branch
	currentBranch, err := getCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != "feature-test" {
		t.Errorf("expected to be on feature-test, but on %s", currentBranch)
	}

	// Cleanup
	if err := worktree.CleanupDirect(); err != nil {
		t.Errorf("CleanupDirect() error = %v", err)
	}

	// Verify we're back on the original branch
	currentBranch, err = getCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("failed to get current branch after cleanup: %v", err)
	}
	if currentBranch != "main" {
		t.Errorf("expected to be back on main after cleanup, but on %s", currentBranch)
	}
}

// TestIsDirectMode tests the IsDirectMode method
func TestIsDirectMode(t *testing.T) {
	tests := []struct {
		name       string
		directMode bool
		want       bool
	}{
		{
			name:       "direct mode enabled",
			directMode: true,
			want:       true,
		},
		{
			name:       "direct mode disabled",
			directMode: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktree := &GitWorktree{
				DirectMode: tt.directMode,
			}

			if got := worktree.IsDirectMode(); got != tt.want {
				t.Errorf("IsDirectMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
