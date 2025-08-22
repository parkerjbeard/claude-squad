package tmux

import (
	cmd2 "claude-squad/cmd"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"claude-squad/cmd/cmd_test"

	"github.com/stretchr/testify/require"
)

type MockPtyFactory struct {
	t *testing.T

	// Array of commands and the corresponding file handles representing PTYs.
	cmds  []*exec.Cmd
	files []*os.File
}

func (pt *MockPtyFactory) Start(cmd *exec.Cmd) (*os.File, error) {
	filePath := filepath.Join(pt.t.TempDir(), fmt.Sprintf("pty-%s-%d", pt.t.Name(), rand.Int31()))
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
	if err == nil {
		pt.cmds = append(pt.cmds, cmd)
		pt.files = append(pt.files, f)
	}
	return f, err
}

func (pt *MockPtyFactory) Close() {}

func NewMockPtyFactory(t *testing.T) *MockPtyFactory {
	return &MockPtyFactory{
		t: t,
	}
}

func TestSanitizeName(t *testing.T) {
	session := NewTmuxSession("asdf", "program")
	require.Equal(t, TmuxPrefix+"asdf", session.sanitizedName)

	session = NewTmuxSession("a sd f . . asdf", "program")
	require.Equal(t, TmuxPrefix+"asdf__asdf", session.sanitizedName)
}

func TestStartTmuxSession(t *testing.T) {
	ptyFactory := NewMockPtyFactory(t)

	created := false
	cmdExec := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error {
			if strings.Contains(cmd.String(), "has-session") && !created {
				created = true
				return fmt.Errorf("session already exists")
			}
			return nil
		},
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("output"), nil
		},
	}

	workdir := t.TempDir()
	session := newTmuxSession("test-session", "claude", ptyFactory, cmdExec)

	err := session.Start(workdir)
	require.NoError(t, err)
	require.Equal(t, 2, len(ptyFactory.cmds))
	require.Equal(t, fmt.Sprintf("tmux new-session -d -s claudesquad_test-session -c %s claude", workdir),
		cmd2.ToString(ptyFactory.cmds[0]))
	require.Equal(t, "tmux attach-session -t claudesquad_test-session",
		cmd2.ToString(ptyFactory.cmds[1]))

	require.Equal(t, 2, len(ptyFactory.files))

	// File should be closed.
	_, err = ptyFactory.files[0].Stat()
	require.Error(t, err)
	// File should be open
	_, err = ptyFactory.files[1].Stat()
	require.NoError(t, err)
}

func TestHasUpdated_ChangeDetection(t *testing.T) {
	ptyFactory := NewMockPtyFactory(t)

	content := "hello world"
	cmdExec := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error { return nil },
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			// Only care about capture-pane outputs in this test
			if strings.Contains(cmd.String(), "capture-pane") {
				return []byte(content), nil
			}
			return []byte(""), nil
		},
	}

	session := newTmuxSession("test-hash", "other", ptyFactory, cmdExec)
	session.monitor = newStatusMonitor()

	// First call: no previous hash, treated as updated
	updated, prompt := session.HasUpdated()
	require.True(t, updated)
	require.False(t, prompt)

	// Second call: same content, not updated
	updated, prompt = session.HasUpdated()
	require.False(t, updated)
	require.False(t, prompt)

	// Third call with different content; wait for cache TTL to expire
	content = "different content"
	time.Sleep(captureCacheTTL + 10*time.Millisecond)
	updated, prompt = session.HasUpdated()
	require.True(t, updated)
	require.False(t, prompt)
}

func TestHasUpdated_PromptDetection_Claude(t *testing.T) {
	ptyFactory := NewMockPtyFactory(t)
	cmdExec := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error { return nil },
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("No, and tell Claude what to do differently"), nil
		},
	}
	session := newTmuxSession("p1", ProgramClaude, ptyFactory, cmdExec)
	session.monitor = newStatusMonitor()
	_, prompt := session.HasUpdated()
	require.True(t, prompt)
}

func TestHasUpdated_PromptDetection_Aider(t *testing.T) {
	ptyFactory := NewMockPtyFactory(t)
	cmdExec := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error { return nil },
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("(Y)es/(N)o/(D)on't ask again"), nil
		},
	}
	session := newTmuxSession("p2", "aider --model something", ptyFactory, cmdExec)
	session.monitor = newStatusMonitor()
	_, prompt := session.HasUpdated()
	require.True(t, prompt)
}

func TestHasUpdated_PromptDetection_Gemini(t *testing.T) {
	ptyFactory := NewMockPtyFactory(t)
	cmdExec := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error { return nil },
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("Yes, allow once"), nil
		},
	}
	session := newTmuxSession("p3", ProgramGemini, ptyFactory, cmdExec)
	session.monitor = newStatusMonitor()
	_, prompt := session.HasUpdated()
	require.True(t, prompt)
}

func TestCaptureUnifiedCaching(t *testing.T) {
	ptyFactory := NewMockPtyFactory(t)
	callCount := 0
	cmdExec := cmd_test.MockCmdExec{
		RunFunc: func(cmd *exec.Cmd) error { return nil },
		OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
			if strings.Contains(cmd.String(), "capture-pane") {
				callCount++
				return []byte("a"), nil
			}
			return []byte(""), nil
		},
	}
	session := newTmuxSession("cache", ProgramClaude, ptyFactory, cmdExec)
	session.monitor = newStatusMonitor()

	// First call populates cache
	_, _, _, err := session.CaptureUnified(false, 0)
	require.NoError(t, err)
	// Second call within TTL should use cache
	_, _, _, err = session.CaptureUnified(false, 0)
	require.NoError(t, err)

	require.Equal(t, 1, callCount, "expected capture-pane to be called once due to caching")
}
