package session

import (
	"claude-squad/cmd/cmd_test"
	"claude-squad/session/tmux"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that Preview uses viewport-bounded capture (-S -<height> -E -1)
func TestInstancePreviewUsesBoundedCapture(t *testing.T) {
	// Create instance
	inst, err := NewInstance(InstanceOptions{Title: "t", Path: t.TempDir(), Program: "claude"})
	require.NoError(t, err)
	// Mark as started and set height directly to avoid SetPreviewSize side effects
	inst.started = true
	inst.Height = 42

	// Mock tmux session
	calls := 0
	execMock := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error { return nil },
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			if strings.Contains(cmd.String(), "capture-pane") {
				calls++
				// Expect -S -42 -E -1 in the command
				if !strings.Contains(cmd.String(), fmt.Sprintf("-S -%d -E -1", inst.Height)) {
					t.Fatalf("expected bounded capture flags in command, got: %s", cmd.String())
				}
				return []byte("ok"), nil
			}
			return []byte(""), nil
		},
	}

	// Inject tmux session with mock executor
	ts := tmux.NewTmuxSessionWithDeps("t", "claude", nil, execMock)
	inst.SetTmuxSession(ts)

	// Call Preview which should perform bounded capture
	_, err = inst.Preview()
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}
