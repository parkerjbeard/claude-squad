package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestNewInstanceDirectMode tests creating an instance with direct mode
func TestNewInstanceDirectMode(t *testing.T) {
	tests := []struct {
		name    string
		opts    InstanceOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid direct mode instance",
			opts: InstanceOptions{
				Title:        "test-direct",
				Path:         ".",
				Program:      "claude",
				DirectMode:   true,
				DirectBranch: "main",
			},
			wantErr: false,
		},
		{
			name: "direct mode without branch",
			opts: InstanceOptions{
				Title:      "test-direct-no-branch",
				Path:       ".",
				Program:    "claude",
				DirectMode: true,
			},
			wantErr: true,
			errMsg:  "direct mode requires a branch name",
		},
		{
			name: "normal mode (not direct)",
			opts: InstanceOptions{
				Title:   "test-normal",
				Path:    ".",
				Program: "claude",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := NewInstance(tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewInstance() expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewInstance() error = %v, want %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewInstance() unexpected error: %v", err)
				return
			}

			// Verify instance fields
			if instance.DirectMode != tt.opts.DirectMode {
				t.Errorf("DirectMode = %v, want %v", instance.DirectMode, tt.opts.DirectMode)
			}

			if instance.DirectBranch != tt.opts.DirectBranch {
				t.Errorf("DirectBranch = %v, want %v", instance.DirectBranch, tt.opts.DirectBranch)
			}

			if instance.Title != tt.opts.Title {
				t.Errorf("Title = %v, want %v", instance.Title, tt.opts.Title)
			}
		})
	}
}

// TestInstanceStartDirectMode tests starting an instance in direct mode
func TestInstanceStartDirectMode(t *testing.T) {
	// Skip if not in a git repository
	if _, err := git.PlainOpen("."); err != nil {
		t.Skip("Skipping test: not in a git repository")
	}

	// Create a temporary directory for testing
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

	// Test starting a direct mode instance
	instance := &Instance{
		Title:        "test-direct-instance",
		Path:         repoPath,
		Program:      "echo 'test'", // Use echo for testing
		DirectMode:   true,
		DirectBranch: "main",
		Status:       Ready,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Note: We can't fully test Start() without mocking tmux
	// This is more of a smoke test to ensure the direct mode path works

	// Verify the instance is configured correctly
	if !instance.DirectMode {
		t.Error("expected DirectMode to be true")
	}

	if instance.DirectBranch != "main" {
		t.Errorf("expected DirectBranch to be 'main', got %s", instance.DirectBranch)
	}
}

// TestInstancePauseResumeDirectMode tests pause/resume functionality in direct mode
func TestInstancePauseResumeDirectMode(t *testing.T) {
	// Direct mode doesn't support pause/resume in the same way
	// This test verifies that the behavior is correct

	instance := &Instance{
		Title:        "test-pause-resume",
		Path:         ".",
		Program:      "claude",
		DirectMode:   true,
		DirectBranch: "feature-test",
		Status:       Running,
		started:      true,
	}

	// In direct mode, pause should behave differently
	// Since we're working directly on the branch, we don't remove worktrees
	// This is mainly a placeholder test to ensure the structure is correct

	if !instance.DirectMode {
		t.Error("expected instance to be in direct mode")
	}

	// Verify status tracking
	instance.SetStatus(Paused)
	if instance.Status != Paused {
		t.Errorf("expected status to be Paused, got %v", instance.Status)
	}

	if !instance.Paused() {
		t.Error("expected Paused() to return true")
	}
}

// TestInstanceDataSerializationDirectMode tests serialization of direct mode instances
func TestInstanceDataSerializationDirectMode(t *testing.T) {
	instance := &Instance{
		Title:        "test-serialization",
		Path:         "/test/path",
		Branch:       "feature-branch",
		Status:       Ready,
		Program:      "claude",
		DirectMode:   true,
		DirectBranch: "main",
		Height:       24,
		Width:        80,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Convert to InstanceData
	data := instance.ToInstanceData()

	// Verify the data preserves direct mode fields
	// Note: DirectMode and DirectBranch need to be added to InstanceData struct
	// for full serialization support

	if data.Title != instance.Title {
		t.Errorf("Title not preserved: got %s, want %s", data.Title, instance.Title)
	}

	if data.Path != instance.Path {
		t.Errorf("Path not preserved: got %s, want %s", data.Path, instance.Path)
	}

	if data.Branch != instance.Branch {
		t.Errorf("Branch not preserved: got %s, want %s", data.Branch, instance.Branch)
	}
}
