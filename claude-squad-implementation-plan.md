# Claude Squad Implementation Plan

## Architecture Overview

**Tech Stack:**
- Go 1.23+ with modules
- Bubble Tea v1.3.4 (TUI framework) 
- Lipgloss (styling)
- go-git v5 (git operations)
- tmux (session management)
- PTY management via creack/pty

**Current Patterns:**
- Message-driven architecture using tea.Msg/tea.Cmd
- MVC-like separation: app layer → session layer → UI components
- Async command execution with timeouts
- Cached tmux captures (200ms TTL)
- Tab-based UI gating

**Key Components:**
- `app/app.go`: Main application loop and state management
- `session/`: Instance, tmux, and git worktree management
- `ui/`: TabbedWindow, Preview, Diff, List, Menu, overlays
- `config/`: Configuration and state persistence

---

## Batch 1: Core Performance Infrastructure

**Context:** Establish foundational performance improvements with minimal risk. These changes reduce idle CPU by >50% immediately through better event handling and selective updates.

### Feature: Bubble Tea Mouse Event Optimization
- Upgrade Bubble Tea dependency to latest stable version with `tea.WithMouse()` support
- Replace `tea.WithMouseCellMotion()` with `tea.WithMouse()` in app/app.go:28
- Verify mouse wheel scroll still works in Preview and Diff panes
- Test: Monitor CPU with mouse movement - should drop from ~5-10% to near 0%

### Feature: Preview Update Gating at Call Site
- Add tab-aware gating in app/app.go Update() case previewTickMsg
- Check `!m.tabbedWindow.IsPreviewTab()` before calling instanceChanged()
- Return early with next preview tick scheduled when not in Preview tab
- Ensure preview resumes within one tick when switching back to Preview
- Test: Verify preview capture count drops to ~0/min when in Diff tab

### Feature: Metrics and Counters System
- Create new `metrics/` package with atomic counters
- Implement counters: tmuxCaptures, gitNumstatCalls, gitFullDiffCalls, uiRenders
- Add Increment() methods guarded by build tag or CSQUAD_DEBUG env var
- Instrument capture sites in tmux.go, git operations, and View() calls
- Add periodic dump on exit and optional 1s sampling
- Test: Unit test counter increments; verify overhead <0.1% when disabled

### Feature: Configurable Tick Intervals
- Add fields to config.Config: PreviewTickMs (250), MetadataTickMs (500), DiffDebounceMs (300), WatcherPollMs (300)
- Load config values once at startup in newHome()
- Update all tick timers to use config values instead of hardcoded durations
- Support env var overrides: CSQUAD_PREVIEW_MS, CSQUAD_METADATA_MS, etc.
- Test: Set extreme values (50ms, 5000ms) and verify no panics

---

## Batch 2: Diff System Optimization

**Context:** The diff system is a major CPU consumer. This batch implements smart debouncing, caching, and tab-aware computation to cut diff-related CPU by >70%.

### Feature: Diff Debounce Clock Implementation
- Add debounce state to app.home: diffWatchDirtyAt time.Time
- Create diffDebounceMsg type and scheduleDiffDebounce() tea.Cmd
- On diffWatchTickedMsg with changed=true, record dirtyAt and schedule debounce
- Debounce fires after 300ms (configurable) if still dirty and in Diff tab
- Cancel pending debounce on tab switch away from Diff
- Test: Rapid file changes trigger ≤1 diff render per debounce window

### Feature: Cached Full Diff with Hash Validation
- Add lastFullDiffHash []byte field to session.Instance (private)
- In SetDiffStats(), compute FNV-1a hash of stats.Content
- Skip UI update if hash matches previous (content unchanged)
- Invalidate cache on next dirty edge detection
- Store hash alongside diff stats for persistence
- Test: Identical diff content doesn't trigger re-render

### Feature: Cheaper Diff Stats Outside Diff Tab
- Ensure GitWorktree.Diff() uses `git diff --numstat --no-ext-diff` for counts only
- Reserve `git add -N` operations exclusively for DiffFull() 
- Metadata tick only computes counts for selected/AutoYes/warmup instances
- Verify no index bit modifications outside full diff operations
- Test: Run 10 min idle on Preview tab, verify `git diff --staged` stays clean

### Feature: Diff Watcher Lifecycle Management
- Strengthen watcher start/stop logic in instanceChanged()
- Start watcher only when: Diff tab active AND instance selected
- On tab switch or selection change, ensure previous watcher shutdown
- Reset m.diffWatchActive=false on watcher stop
- Add debug logging for watcher lifecycle events (when CSQUAD_DEBUG=1)
- Test: Fast tab toggles show no goroutine leaks; watcher polling stops when hidden

---

## Batch 3: Subprocess Resilience

**Context:** Shell operations can stall or fail. This batch adds timeouts, exponential backoff, and graceful error handling to keep the UI responsive even when subprocesses misbehave.

### Feature: Timeout and Backoff for Tmux Operations
- Wrap makeTmuxStatusCmd with backoff state map[*session.Instance]backoffState
- Implement exponential backoff: 1s → 2s → 4s (cap at 2s for UX)
- Reset backoff to base on successful operation
- Apply same pattern to tmux capture operations
- Current timeouts: 400ms status, 1500ms diff (keep these)
- Test: Kill tmux server, verify UI stays responsive with one warning per backoff window

### Feature: Git Operations Backoff
- Apply backoff wrapper to makeGitDiffCmd 
- Track failures per instance with exponential growth
- Cap backoff at 2s to maintain responsiveness
- Reset on successful git operation
- Add timeout enforcement for all git commands (1500ms default)
- Test: Corrupt git repo, verify graceful degradation without UI freeze

### Feature: Rate-Limited Warning Logs
- Create log.WarningLog wrapper with rate limiting
- Limit to one warning per key per 5 seconds
- Use key format: "operation:instance:error"
- Apply to all subprocess failure paths
- Include occurrence count in rate-limited messages
- Test: Inject repeated failures, verify log doesn't flood

### Feature: Error Surfacing in UI
- Enhance ui.ErrBox with sticky error display for repeaters
- Track last error key and count in app.handleError
- Show error icon + count for same error within 3s window
- Clear sticky count after 3s of no errors
- Keep single-line constraint for error display
- Test: Repeated timeout shows sticky count; clears after silence

---

## Batch 4: Preview Optimization

**Context:** Preview operations, especially full scrollback, can be expensive. This batch adds smart caching and ring buffers to minimize redundant captures.

### Feature: Preview Ring Buffer for Scrollback
- Add to ui.PreviewPane: ring []byte, ringHash []byte, ringAt time.Time
- On first ScrollUp(), fetch full history via CaptureUnified(true, 0)
- Store in ring buffer (last N KB) with FNV hash
- On subsequent scrolls, check if pane hash changed
- Reuse ring buffer if content unchanged, refetch if different
- Test: Multiple scroll toggles show no extra tmux calls when content static

### Feature: Viewport-Bounded Preview Capture
- Use instance.Height to limit preview capture to visible lines
- Pass maxLines parameter to CaptureUnified when not in scroll mode
- Maintain full capture only when explicitly scrolling
- Update preview height on window resize events
- Test: Large tmux history doesn't impact preview performance when not scrolling

### Feature: Smart Preview Cache Invalidation
- Track content hash in preview pane state
- Only trigger re-render when hash changes
- Preserve scroll position across updates when possible
- Handle edge case of content shrinking below scroll position
- Test: Static content shows 0 captures after initial load

---

## Batch 5: Direct Mode Safeguards

**Context:** Direct mode operates on real branches without worktrees, requiring extra safety measures. This batch strengthens guardrails to prevent accidental data loss.

### Feature: Direct Mode Branch Protection
- Check for reserved branch names (main/master/develop) on start
- Refuse auto-create of protected branches without explicit flag
- Add --force-direct flag for overriding protection
- Validate branch exists before attempting operations
- Clear warning when operating on protected branches
- Test: Starting direct mode on 'main' without flag shows error

### Feature: Enhanced Pause/Resume for Direct Mode
- Include head SHA in commit message for traceability
- Tag commits clearly: "[claudesquad-direct] pause from '<title>' at <sha>"
- Copy branch name + SHA to clipboard on pause
- Verify working directory clean before pause operations
- Add rollback instructions in pause message
- Test: Pause/resume cycle preserves all changes with clear audit trail

### Feature: Direct Mode Cleanup Safety
- Always attempt to restore OriginalBranch on cleanup
- Log warning if branch switch fails (non-fatal)
- Preserve reflog entry for cleanup operations
- Never force-delete branches in direct mode
- Show cleanup status in UI during operation
- Test: Cleanup from detached HEAD shows warning but completes

---

## Batch 6: State Management

**Context:** State persistence must be efficient and bounded. This batch adds size monitoring and smart truncation to prevent state file bloat.

### Feature: State File Watchdog
- Check state.json size after each save
- Log warning if size exceeds 1MB threshold
- Break down size by instance for diagnostics
- Implement cleanup helper to drop diff bodies
- Truncate diff content to 10KB max in persistence layer
- Test: Large diff triggers warning and auto-truncation

### Feature: State Migration System
- Add version field to state.json
- Implement forward migration for schema changes
- Keep backward compatibility for one major version
- Auto-backup state before migration
- Add state validation on load
- Test: Old state files migrate cleanly to new format

### Feature: Selective State Persistence
- Only persist diff stats, not full content
- Store only necessary git worktree metadata
- Implement lazy loading for heavy fields
- Add compression for large text fields
- Keep state writes atomic with temp file + rename
- Test: State file stays under 100KB for 10 instances

---

## Batch 7: UI/UX Polish

**Context:** Small UX improvements that enhance usability and provide better feedback. These changes improve the user experience without adding complexity.

### Feature: Dynamic Key Help Text
- Show "shift+↑/↓" only in Diff tab (already partial)
- Display scroll hints only when preview in scroll mode
- Update help text based on instance state
- Add context-sensitive hints for current operation
- Highlight available actions based on state
- Test: Help text changes appropriately with context

### Feature: Attach/Detach Polish
- Keep stdin drain for ~50ms on attach (prevent noise)
- Always Restore() fresh PTY on detach
- Make attach/detach errors non-fatal with guidance
- Preserve tmux pane buffer across detach cycles
- Add visual feedback during attach process
- Test: Spam input → Ctrl-Q → re-attach preserves state

### Feature: Loading State Improvements
- Show spinner during long operations
- Add progress indicators for multi-step operations
- Display estimated time for known operations
- Cancel button for interruptible operations
- Queue visual feedback for pending operations
- Test: All operations >500ms show visual feedback

---

## Batch 8: Windows/macOS Platform Support

**Context:** Platform-specific quirks need handling. This batch ensures consistent behavior across operating systems.

### Feature: macOS Optimizations
- Build native arm64 binary for M1/M2 Macs
- Keep tmux history limit at 10000 (optimal for macOS)
- Handle macOS-specific PTY behaviors
- Test tmux 3.x compatibility
- Add Homebrew formula support
- Test: Native arm64 build shows 2x performance vs Rosetta

### Feature: Windows ConPTY Support
- Ensure creack/pty ConPTY support is active
- Handle Windows line ending conversions
- Adjust timeouts for slower Windows PTY operations
- Support Windows Terminal and legacy console
- Handle path separators correctly
- Test: Full functionality in Windows Terminal

### Feature: Linux Distribution Compatibility
- Test on Ubuntu, Fedora, Arch
- Handle different tmux versions (2.x vs 3.x)
- Support both systemd and non-systemd systems
- Package as AppImage for distribution
- Handle different terminal emulators
- Test: Works on minimal Linux installations

---

## Implementation Order & Dependencies

### Phase 1: Foundation (Batches 1 & 3)
- No dependencies, safe to implement immediately
- Provides immediate performance wins
- Establishes metrics for measuring further improvements

### Phase 2: Core Optimizations (Batches 2 & 4)  
- Depends on Phase 1 metrics
- Major CPU reduction from diff and preview optimization
- Most complex changes, need careful testing

### Phase 3: Safety & Polish (Batches 5, 6, 7)
- Can be done in parallel
- Lower risk, primarily defensive improvements
- Enhances user experience

### Phase 4: Platform Support (Batch 8)
- Depends on all previous batches
- Final testing and platform-specific adjustments
- Release preparation

---

## Success Metrics

### Performance Targets
- Idle CPU: ≤1% with 3 sessions (currently ~5-10%)
- Subprocess calls: ≤12/min when idle (currently ~60/min)
- Mouse motion CPU impact: ~0% (currently 5-10%)
- Diff computation: ≤1 per 400ms during rapid changes
- Preview updates: 0/min when not in Preview tab

### Reliability Targets
- UI responsiveness: <100ms for all user actions
- Subprocess timeout recovery: UI stays responsive
- State file size: <100KB for typical usage
- Memory usage: <50MB for 10 instances
- Zero data loss in Direct mode operations

### Code Quality Targets
- Test coverage: >80% for new code
- All batches independently revertible
- No breaking changes to CLI interface
- Clean migration path for existing users
- Documentation for all new features

---

## Risk Mitigation

### Testing Strategy
- Unit tests for all new functions
- Integration tests for subprocess handling
- Performance benchmarks before/after
- Manual testing on all platforms
- Beta testing with power users

### Rollback Plan
- Each batch in separate commits
- Feature flags for risky changes
- Config options to restore old behavior
- State file backups before migration
- Clear revert instructions in docs

### Monitoring
- Metrics dashboard for production
- Error reporting with aggregation
- Performance regression alerts
- User feedback channels
- Automated benchmark CI jobs