# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Building and Testing

### Local Development Build
Since this is a fork, build with a custom binary name to avoid conflicts with the original:
```bash
# Build with custom name for testing
go build -o claude-squad-dev
./claude-squad-dev --help

# Or use go run directly
go run . --help

# Build with version info
go build -ldflags "-X main.version=dev-fork" -o claude-squad-dev
```

### Testing the Binary Locally

#### Running on Another Repository
Navigate to your target repository and run:

```bash
# Normal mode (creates worktrees for isolation)
cd /path/to/your/repo
/Users/parkerbeard/claude-squad/claude-squad-dev

# With a specific program
/Users/parkerbeard/claude-squad/claude-squad-dev -p "aider --model claude-3-5-sonnet-20241022"

# Direct mode (edits branches directly, no worktrees)
/Users/parkerbeard/claude-squad/claude-squad-dev -d -b main
/Users/parkerbeard/claude-squad/claude-squad-dev -d -b feature-branch -p "claude"

# Auto-accept mode (experimental)
/Users/parkerbeard/claude-squad/claude-squad-dev -y
```

**Key Differences:**
- **Normal mode**: Creates isolated git worktrees for each session
- **Direct mode (`-d`)**: Works directly on the specified branch without worktrees
- **`-b` flag**: Required in direct mode to specify which branch to edit

### Running Tests
```bash
# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./session/...
go test -v ./app/...

# Run a specific test
go test -v -run TestWorktreeOperations ./session/git
```

### CI/CD
The project uses GitHub Actions for CI. Tests are automatically run on push/PR for:
- Multiple OS: Linux, macOS, Windows
- Multiple architectures: amd64, arm64

## Architecture Overview

### Core Components

**Main Entry Point** (`main.go`):
- CLI built with Cobra framework
- Supports flags: `-p/--program`, `-y/--autoyes`, `-d/--direct`, `-b/--branch`
- Commands: `reset`, `debug`, `version`
- Direct mode allows editing branches without creating worktrees

**Application Layer** (`app/`):
- TUI application using Bubble Tea framework
- `app.go`: Main application state machine and event handling
- State management: default, new, prompt, help, confirm states
- Component orchestration between UI elements

**Session Management** (`session/`):
- `instance.go`: Core session instance management
- `storage.go`: Persistence layer for session state
- Each session runs in isolated tmux session with optional git worktree

**Git Integration** (`session/git/`):
- `worktree.go`, `worktree_ops.go`: Git worktree creation and management
- `worktree_branch.go`: Branch operations
- `diff.go`: Diff generation for preview
- Direct mode support for editing branches without worktrees

**Tmux Integration** (`session/tmux/`):
- `tmux.go`: Tmux session management
- `pty.go`: PTY handling for terminal interaction
- Platform-specific implementations for Unix/Windows

**UI Components** (`ui/`):
- `list.go`: Instance list display
- `menu.go`: Bottom menu navigation
- `tabbed_window.go`: Tab switching between preview/diff
- `preview.go`, `diff.go`: Content display panels
- `overlay/`: Modal overlays for input and confirmation

**Configuration** (`config/`):
- `config.go`: Application configuration (default program, autoyes mode)
- `state.go`: Persistent application state storage
- Config location: `~/.config/claude-squad/` (Unix) or `%APPDATA%\claude-squad\` (Windows)

**Daemon** (`daemon/`):
- Background process for auto-accept mode
- Platform-specific daemon management

### Key Design Patterns

1. **Session Isolation**: Each AI agent runs in its own tmux session with optional git worktree
2. **Direct Mode**: New feature allowing direct branch editing without worktree creation
3. **Event-Driven TUI**: Bubble Tea framework for reactive UI updates
4. **Storage Abstraction**: Clean separation between session logic and persistence
5. **Platform Abstraction**: OS-specific code isolated in `*_unix.go` and `*_windows.go` files

### Session Lifecycle

1. User creates new session → tmux session created
2. In worktree mode: Git worktree created for isolated branch
3. In direct mode: Works directly on specified branch
4. AI agent launched in tmux session
5. Changes tracked via git diff
6. Commit/push operations handle branch integration
7. Cleanup removes worktree (if applicable) and tmux session

## Contributing Back to Upstream

This is a fork intended for contributing improvements back to `smtg-ai/claude-squad`. 
Keep the repository name unchanged to maintain clear fork relationship.
Create feature branches for specific improvements before submitting PRs.