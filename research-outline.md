Parker — here’s the single source of truth the dev will work from. It’s terse, dense, and complete enough to build from without fishing elsewhere.

# Claude Squad: Runtime & UX Overhaul — Research + Dev Outline

## 0) Scope and goals

**Goals**

* Cut idle CPU by ≥70% and subprocess churn by ≥80% while keeping UI snappy.
* Make preview/diff renders scale with pane size and *need*, not time.
* Keep tmux/git work tidy and failure‑safe.
* Ship in small, reversible slices with unit/integration tests.

**Non‑goals**

* No switch away from Bubble Tea, Lipgloss, tmux, or go‑git.
* No feature sprawl (only perf/robustness/UX polish).

**Constraints**

* Code is Go 1.x w/ modules; Bubble Tea v1.3.4 right now.
* TUI must never hang on tmux/git calls.
* No hidden daemons; all long work must be `tea.Cmd`.

---

## 1) Snapshot of the current system (what matters for perf)

**Hot paths now in tree (already partially improved):**

* `app/app.go`

  * `previewTickMsg` loop now at 250ms (from 100ms).
  * `tickUpdateMetadataMessage` loop at 500ms, with simple filter: selected | `AutoYes` | warmup window (5s), `maxScheduled=4`.
  * Diff work moved behind Diff tab and watcher.
* `session/tmux/tmux.go`

  * Unified capture path `CaptureUnified(fullHistory, maxLines)` with 200ms cache TTL.
  * FNV‑1a hashing via `io.WriteString`, not SHA‑256.
  * Trust‑screen auto‑tap for Claude/Aider/Gemini startup; history limit set to 10000.
  * Attach/detach is explicit; Ctrl‑Q detaches; nukes early stdin control noise for \~50ms.
* `session/instance.go`

  * Preview uses viewport‑bounded capture when height set.
  * Diff stats computed on demand; stored in memory; state file truncates diff body to 10KB.
  * Pause/Resume: commits dirty worktree on pause (local), prunes worktrees; resume restores or re‑creates.
  * Direct mode (no worktree) supported; branch hygiene guarded.
* `session/git/worktree_ops.go`

  * Worktree setup fast path (existing branch vs new); guards HEAD‑less repos with clear error.
  * Direct mode checkout guarded; cleanup only restores original branch if known.
* `ui/*`

  * Menu/List/Diff/Preview panes; TabbedWindow manages active tab; scroll hints; confirmation overlay tested.
* Tests: `app/app_test.go`, `ui/preview_test.go` cover overlay states, tab toggle, preview scroll, full‑history capture.

What’s still weak or left on the table:

* Bubble Tea still uses `WithMouseCellMotion()` (motion flood); upgrade needed to use `WithMouse()` once we bump dependency.
* `instanceChanged()` still calls `UpdatePreview` unconditionally; ensure gating at *call site*, not only inside panes.
* Diff debounce lacks an explicit clock; we rely on gating + watcher only.
* No counters/metrics yet; pprof exists but off by env flag only.
* A few tmux/git calls still lack strong backoff and coalescing.

---

## 2) Design tenets

* **Only do work you’ll show.** Gate captures by tab, viewport height, and “changed since last render”.
* **Bound everything.** Pane capture ≤ visible lines; diff counts cheap; full diff on demand + debounce.
* **Never block the Update loop.** All shell outs go through `tea.Cmd` with timeout and clear message types.
* **Prefer event edges over polling rates.** Watcher (dirty edge) drives diff recompute; prompt edge drives AutoYes.
* **Keep state small and strong.** Persist only what you need to restart fast (counts, not bodies).
* **Fail fast, recover fast.** Any subprocess timeout → log once (rate‑limited), surface non‑fatal UI hint, keep UI alive.

---

## 3) Work items (W#), decisions, and acceptance checks

### W1. **Bubble Tea upgrade and mouse event cut**

* **Why**: `WithMouseCellMotion()` spams motion events; we only need wheel/click.
* **Plan**: Bump `github.com/charmbracelet/bubbletea` to a version with `tea.WithMouse()`; keep code paths the same for wheel scroll (`tea.MouseMsg`).
* **Edits**: `app/app.go: Run()` swap `tea.WithMouseCellMotion()` → `tea.WithMouse()`.
* **Tests**: Wheel scroll in Preview and Diff still works; no motion floods under `strace`/log; CPU drop in idle when moving mouse.

### W2. **Hard gate preview updates at call site**

* **Why**: Even if `UpdatePreview` early‑outs, we still bounce through it.
* **Plan**: In `Update` → `case previewTickMsg`, call `instanceChanged()` *only* when `TabbedWindow` active tab == Preview, else schedule next tick.
* **Edits**: `app/app.go: Update()`; add:

  ```go
  if !m.tabbedWindow.IsPreviewTab() {
      return m, nextPreviewTick()
  }
  ```
* **Tests**: Switch to Diff tab; preview capture count \~0/min; switch back; preview resumes with <1 tick delay.

### W3. **Diff debounce clock + cached full diff**

* **Why**: Even w/ gating, typing bursts can trigger too many diffs.
* **Plan**:

  * Add a short debounce (`250–400ms`) after `diffWatchTickedMsg{changed:true}` before running `makeGitDiffCmd`.
  * Cache last full diff in instance; invalidate on next dirty edge. Skip recompute if content equal (hash).
* **Edits**:

  * `app/app.go`: after `diffWatchTickedMsg{changed:true}`, set `m.diffWatchDirtyAt = time.Now()`, schedule a debounce `tea.Cmd` that checks “still dirty and in Diff tab”; then `makeGitDiffCmd`.
  * `session/instance.go`: add `lastFullDiffHash []byte` + `SetDiffStats` computes and stores hash of diff content.
* **Tests**: Rapid file changes → ≤1 diff render per debounce window; no lost update when idle ends.

### W4. **Cheaper diff outside Diff tab**

* **Why**: We don’t need bodies; counts are enough for list badges.
* **Plan**: Ensure that counts are taken via a cheap path (`git diff --numstat --no-ext-diff` or current `Diff()` path) on metadata tick only for selected/AutoYes/warmup instances. No `add -N` outside full diff.
* **Edits**: Verify `GitWorktree.Diff()` doesn’t trigger index writes; ensure `DiffFull()` is the only place that may do `add -N`.
* **Tests**: Confirm no index bit flips (`git diff --staged` clean) after 10 min idle on Preview tab.

### W5. **`instanceChanged()` hygiene + watcher lifecycle**

* **Why**: Avoid stale watchers and needless recompute.
* **Plan**: Keep current logic that starts watcher only on Diff tab for selected instance. On tab switch or selection change, ensure previous watcher shuts down and state resets (`m.diffWatchActive=false`).
* **Edits**: `app/app.go: instanceChanged()`; keep but add debug flags for watcher lifecycle.
* **Tests**: Toggle tabs fast; no leak of goroutines; watcher poll stops when tab hidden.

### W6. **Timeout/backoff on tmux capture and status**

* **Why**: Shell calls can stall on slow boxes.
* **Plan**: Current timeouts (400ms status, 1500ms diff) are fine; add exponential backoff for repeated failures to quiet logs and spare CPU.
* **Edits**:

  * Wrap `makeTmuxStatusCmd` and `makeGitDiffCmd` with small backoff state (per instance) that grows to 2×, capped at 2s; reset on success.
  * Rate‑limit warning logs (`log.WarningLog`) to once per N seconds per key.
* **Tests**: Force tmux to fail (kill server); see one warning per backoff window; UI stays live.

### W7. **Counters + pprof on switch**

* **Why**: Measure, don’t guess.
* **Plan**: Add light counters guarded by build tag or env:

  * `tmuxCaptures`, `gitNumstatCalls`, `gitFullDiffCalls`, `uiRenders`.
* **Edits**: Small package `metrics` with `atomic` ints and `Increment()`; calls at capture sites and `View()`.
* **Tests**: Unit test counters; sample at 1s and dump on exit.

### W8. **Make tick intervals tunable**

* **Why**: Boxes differ; devs need knobs.
* **Plan**: Add fields in `config.Config`:

  * `PreviewTickMs` (default 250), `MetadataTickMs` (500), `DiffDebounceMs` (300), `WatcherPollMs` (300).
  * Read once at startup.
* **Edits**: `config/`, `app/newHome()`, timers use cfg values.
* **Tests**: Set extremes; see no panic and sane behavior.

### W9. **Strengthen Direct Mode guardrails**

* **Why**: Direct mode writes to real branch; less room for error.
* **Plan**:

  * On Direct start: check branch exists for reserved names (`main/master/develop`), refuse auto‑create unless flag set.
  * On Pause: commit message clearly tagged + include head short SHA; copy branch name to clipboard (already).
  * On Cleanup: try to switch back to `OriginalBranch`; warn if fails (already does).
* **Edits**: `session/git/worktree_ops.go` and `session/instance.go` direct path guard.
* **Tests**: Create repo with `main`; start Direct on `main` without flag → error; with flag → OK.

### W10. **Attach/detach polish**

* **Why**: Avoid stray control bytes, preserve buffer.
* **Plan**: Keep current “nuke stdin for \~50ms” on attach; on detach, always `Restore()` a fresh PTY; make errors non‑fatal and guide user.
* **Edits**: Minor log wording; ensure `DetachSafely()` used on Pause.
* **Tests**: Attach, spam input, Ctrl‑Q; re‑attach; pane state survives.

### W11. **State file watchdog**

* **Why**: Prevent regressions that bloat `state.json`.
* **Plan**: After save, if state > 1MB, log warning with per‑instance sizes; offer cleanup path to drop diff bodies (`Content=""`).
* **Edits**: `session/storage` layer (not shown; add here).
* **Tests**: Synthetic big diff; see warning; assert truncated.

### W12. **Error surfacing in UI**

* **Why**: Hard faults should be seen, not buried.
* **Plan**: Keep `errBox` one‑line; add icon + sticky for “repeaters” (same error within window).
* **Edits**: `ui/err_box.go` new style; `app.handleError` tracks last error key and count.
* **Tests**: Inject repeated timeout; verify sticky count grows; clears after 3s silence.

### W13. **Keys + help text truth**

* **Why**: Key help must match logic.
* **Plan**: On Tab toggle, menu should show “shift+↑/↓” only in Diff tab (test exists); ensure Preview has scroll help only when in scroll mode.
* **Edits**: `ui/menu.go` conditional hints based on pane state.
* **Tests**: Already partly covered; add preview scroll hint cases.

### W14. **Fast path on full scrollback preview**

* **Why**: Full history fetch can be costly.
* **Plan**: Use `CaptureUnified(true, 0)` only when entering scroll mode the first time; after that, hold a ring buffer (last N KB) in preview pane state with a cheap hash; refill only on change.
* **Edits**: `ui/preview.go`:

  * `ring []byte`, `ringHash []byte`, `ringAt time.Time`.
  * On `ScrollUp()`: if ring empty or pane hash changed, fetch full, else reuse.
* **Tests**: Toggle scroll many times; no extra tmux calls when content unchanged.

### W15. **Windows/macOS quirks**

* **Why**: PTY and tmux differ across OSs.
* **Plan**:

  * macOS: ship arm64 build; keep history limit at 10000.
  * Windows: `creack/pty` covers ConPTY; keep timeouts modest; don’t assume full ANSI.
* **Edits**: Build doc; CI matrix; no code churn unless needed.

---

## 4) Data flow and concurrency map

**Preview path**

```
previewTick -> (tab==Preview?) -> instanceChanged()
  -> TabbedWindow.UpdatePreview(selected) -> Instance.Preview()
     -> Tmux.CaptureUnified(fullHistory=false, maxLines=height)
        - returns {content, hash, hasPrompt} (200ms cache)
  -> UI rerender only if content hash changed
```

**Diff path**

```
Tab==Diff -> start watcher (poll 300ms):
  git status porcelain -> dirty bool
  edge true -> set dirtyAt, schedule debounce (300ms)
  debounce fires and still dirty -> makeGitDiffCmd
    -> GitWorktree.DiffFull() (no index unless needed)
    -> Instance.SetDiffStats(stats, hash)
  UI renders diff view
```

**Tmux status / AutoYes**

```
metadataTick (500ms) over small set:
  makeTmuxStatusCmd (timeout 400ms)
    -> Tmux.HasUpdated() -> CaptureUnified(false, 0)
    -> if hasPrompt && instance.AutoYes -> Instance.TapEnter()
```

**Attach/detach**

```
Enter -> show help overlay -> List.Attach()
  attach goroutines:
    - copy stdout to user
    - read stdin and watch for Ctrl-Q -> Detach()
    - watch window size -> PTY resize
  on Detach -> Restore() a fresh PTY for managed mode
```

---

## 5) API changes (public in our code)

* `TabbedWindow`

  * add: `IsPreviewTab() bool` (wrapper), `IsInDiffTab() bool` exists.
* `Instance`

  * add: `lastFullDiffHash []byte` (private) and hash check in `SetDiffStats`.
* `app.home`

  * add: debounce fields `diffWatchDirtyAt time.Time`, `diffDebounce *time.Timer` (or schedule via `tea.Cmd`).
  * add: backoff state maps for tmux/diff per instance (`map[*session.Instance]backoff`).
* `config.Config`

  * add: `PreviewTickMs`, `MetadataTickMs`, `DiffDebounceMs`, `WatcherPollMs`.

No breaking changes to external CLI.

---

## 6) File‑by‑file plan (exact edits)

* `app/app.go`

  * `Run`: switch to `tea.WithMouse()` after dep bump (W1).
  * `Init`: preview tick sleep uses cfg.
  * `Update`:

    * In `previewTickMsg`: early return when not in Preview (W2).
    * `tickUpdateMetadataMessage`: honor cfg interval; filter set kept.
    * Handle `diffWatchTickedMsg`: record `dirtyAt`; start debounce `tea.Cmd`; handle cancel on tab switch (W3).
  * `instanceChanged`: ensure watcher teardown when leaving Diff (present, keep).

* `session/tmux/tmux.go`

  * Keep capture TTL; no change. Add backoff wrappers if needed (W6).

* `session/instance.go`

  * In `SetDiffStats`: compute small FNV hash of `stats.Content`; skip UI update if hash same (W3).
  * Guard `SendPrompt` newline timing (already sleeps 100ms; keep).

* `session/git/…`

  * Double‑check `Diff()` vs `DiffFull()` for index writes (W4).

* `ui/preview.go` (not shown here; implement per W14)

  * Add ring and its hash.
  * On `ScrollUp()`: fetch full history once per content hash; else reuse.

* `ui/menu.go`, `ui/err_box.go` (not shown here)

  * Hints per pane; sticky error count (W12, W13).

* `config/` (not shown)

  * Add fields and loader defaults.

* `developer‑docs/`

  * Update both docs to point at this one; mark older proposals as background only.

---

## 7) Tests (must pass before merging each slice)

**Unit**

* `app/app_test.go`

  * Extend `TestTabToggleUpdatesMenu` to assert preview tick gating: when in Diff, `UpdatePreview` mocked count == 0 over N ticks.
  * Add tests for confirmation overlay already exist; keep.
* `ui/preview_test.go`

  * New cases: ring buffer reuse across scroll mode toggles; ensure no extra `capture-pane` when content unchanged.
* `session/tmux` (mock exec)

  * Test capture cache TTL works; repeated calls within 200ms return cached body; HasUpdated toggles on content change.

**Integration (mock shell)**

* Git diff debounce: make files dirty fast; assert ≤1 `DiffFull()` within debounce window.
* AutoYes prompt edge: inject prompt text and see `TapEnter()` is sent once.

**Perf checks (lightweight)**

* Counters: assert `tmuxCaptures` ≪ baseline for same script (document baseline once).

---

## 8) Rollout

**Slices**

1. W2, W3 (gating + debounce) — no deps.
2. W7, W8 (metrics + tunables).
3. W4 (counts vs full).
4. W1 (Bubble Tea upgrade), then swap to `WithMouse()`.
5. W14 (preview ring).
6. W9/W12 polish paths.

**Feature flags**

* Env vars override config ticks: `CSQUAD_PREVIEW_MS`, `CSQUAD_METADATA_MS`, `CSQUAD_DIFF_DEBOUNCE_MS`, `CSQUAD_WATCH_MS`.
* Metrics on by `CSQUAD_DEBUG=1`.
* pprof by `CSQUAD_PPROF=1` (already noted).

**Revert path**

* Each slice touches few files; keep commits small; flip config to old values if needed.

---

## 9) Risks and how we fence them

* **Missed prompt detection** if metadata loop too sparse.
  Fence: keep AutoYes in monitored set; don’t back off beyond 2s; warmup window 5s.

* **Out‑of‑date diff** if debounce too long.
  Fence: cap at 400ms; on tab leave/re‑enter, recompute once.

* **Tmux slow startup on some systems.**
  Fence: current exponential probe for trust screen up to 30–45s; keep.

* **State bloat creeping back.**
  Fence: 1MB watchdog + truncation guard already in place.

---

## 10) Reference behaviors (documented invariants from code)

* **Startup trust screen**:

  * Claude: looks for `Do you trust the files in this folder?` then taps Enter.
  * Aider/Gemini: looks for `Open documentation url for more info`, taps `D` then Enter. Timeouts: 30–45s (backoff).
* **Attach/Detach**:

  * Ctrl‑Q detaches; on EOF without detach, print red warning.
* **Pause**:

  * Commits dirty work locally with `[claudesquad] update from '<title>' on <RFC822> (paused)`, prunes worktrees, copies branch name to clipboard.
* **Kill**:

  * For non‑direct, deny if branch checked out in repo; prevents foot‑gun.
* **Diff stats in list**:

  * Show `+added,-removed` with colors; empty when error/none.

---

## 11) Developer checklists

**Before code**

* Confirm exact Bubble Tea version target (for W1).
* Decide default debounce: 300ms works well; expose knob.

**During code**

* Shell‑out paths: every call behind `tea.Cmd` with timeout.
* Never log the same warning more than once per 5s per key.

**Before merge**

* Run unit + integration tests.
* Sanity manual: 5 sessions open, mouse move across screen — CPU must not spike.

---

## 12) Example snippets to drop in

**Preview gating**

```go
case previewTickMsg:
    if !m.tabbedWindow.IsPreviewTab() {
        return m, func() tea.Msg {
            time.Sleep(time.Duration(m.appConfig.PreviewTickMs) * time.Millisecond)
            return previewTickMsg{}
        }
    }
    cmd := m.instanceChanged()
    return m, tea.Batch(cmd, func() tea.Msg {
        time.Sleep(time.Duration(m.appConfig.PreviewTickMs) * time.Millisecond)
        return previewTickMsg{}
    })
```

**Diff debounce**

```go
type diffDebounceMsg struct{}
func (m *home) scheduleDiffDebounce() tea.Cmd {
    return func() tea.Msg {
        time.Sleep(time.Duration(m.appConfig.DiffDebounceMs) * time.Millisecond)
        return diffDebounceMsg{}
    }
}
// in Update:
case diffWatchTickedMsg:
    if msg.changed && m.tabbedWindow.IsInDiffTab() && m.diffWatchActive {
        m.diffWatchDirtyAt = time.Now()
        return m, m.scheduleDiffDebounce()
    }
case diffDebounceMsg:
    if !m.tabbedWindow.IsInDiffTab() || !m.diffWatchActive { return m, nil }
    // still dirty? always recompute on debounce edge
    return m, makeGitDiffCmd(m.diffWatchInst)
```

**Skip duplicate diff body**

```go
// in Instance.SetDiffStats
sum := fnv.New64a()
_, _ = io.WriteString(sum, stats.Content)
hash := sum.Sum(nil)
if bytes.Equal(hash, i.lastFullDiffHash) { return }
i.lastFullDiffHash = append(i.lastFullDiffHash[:0], hash...)
i.diffStats = stats
```

---

## 13) Build & ship notes

* Build flags:

  * macOS/arm64 native binary: `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w"`.
* tmux hint for users: keep tmux ≥ 3.x; history limit is set per session to 10000 already.
* Terminal hint: leave host term scrollback sane; rely on tmux scrollback in scroll mode.

---

## 14) Appendix: key tables and message types

**Key binds (as implemented)**

* `?` Help overlay
* `n` New session
* `p` Prompt (create+prompt path)
* `tab` Toggle Preview/Diff
* `enter` Attach (w/ help overlay)
* `r` Resume (from Paused)
* `c` Checkout guidance for branch switch (pauses instance)
* `s` Submit (push) — asks confirm, commits/pushes
* `d` Kill — asks confirm, cleans storage+tmux+worktree
* `Shift+↑/↓` Scroll Diff in Diff tab
* Preview scroll: wheel / `Shift+↑/↓` only when in scroll mode

**Tea messages (new/used)**

* `previewTickMsg`, `tickUpdateMetadataMessage`, `tmuxStatusMsg`, `gitDiffMsg`, `diffWatchTickedMsg`, `diffDebounceMsg` (new), `instanceChangedMsg`.

**States**

* `Running`, `Ready`, `Paused`, `Loading` (enum).
* Home states: `stateDefault`, `stateNew`, `statePrompt`, `stateHelp`, `stateConfirm`.

---

## 15) What “done” looks like

* Idle (Preview tab, 3 sessions, 1 selected): tmux captures ≤ 12/min; CPU near 0–1%.
* Idle (Diff tab hidden): no `DiffFull()` calls; only numstat (or none) for selected.
* Rapid typing in worktree: Diff tab shows updates in ≤400ms; Preview stays smooth.
* Mouse motion has near‑zero impact on CPU with `WithMouse()`.

That’s enough to build. If you hit something weird in a corner of the tree, default to the tenets: show only what you must, bound work by the pane, never block the loop, and keep state lean.
