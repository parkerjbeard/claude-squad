Here’s the single, UI-only build spec. No perf talk. Just what to change, how it should look, and the exact code touch points.

# Claude Squad — Frontend/UI Overhaul Spec

## Goals

* Make the TUI legible, predictable, and teachable in 30 seconds.
* Remove hidden states and unclear affordances.
* Unify keys, clicks, and hints. One path per action; no duplicates.
* Keep all styles in one theme file; remove magic numbers from views.
* Zero surprises when width is tight.

## Non-Goals

* No back-end or perf shifts. No tmux/git logic changes. Only UI, input, and rendering.

## Information layout

Primary screen is two zones:

* Left: **Instances List** (selectable, status/diff badges, repo/branch line).
* Right: **Tabbed Window** with two tabs:

  * **Preview**: live log with clear states; optional scroll mode with footer hints.
  * **Diff**: header with `+ / −` counts, file list, and colored hunks.
* Bottom: **Menu**; grouped actions; dimmed when not usable.
* One-line **Error** bar under the menu.

On narrow terms (< 90 cols), stack as rows: List on top, Tabs below, then Menu and Error.

## Input model

* Keys

  * Global: `n` new, `N` new+prompt, `D` kill, `tab` switch tab, `?` help, `q` quit.
  * Attach: `enter`/`o`; detach: `ctrl+q` (unchanged, but teach it).
  * Preview scroll: `PgUp/PgDn`, `Home/End`, `g/G`, `Ctrl+u/Ctrl+d`. `Esc` exits scroll.
  * Diff scroll: `PgUp/PgDn`, `[`/`]` to jump hunks, `{`/`}` to jump files.
  * Select by number: `1..9` jump to nth instance; `0` = 10th; repeat by tens with `Alt+1..9` (optional).
* Mouse

  * Wheel scrolls Preview/Diff.
  * Click list item to select.
  * Click tab label to switch.

Menu always mirrors what works **now** and dims what doesn’t.

## Visual design system

Add a theme token file; stop scattering colors. Use adaptive colors with safe contrast in both light and dark.

**File**: `ui/theme.go` (new)

```go
package ui

import "github.com/charmbracelet/lipgloss"

type Palette struct {
    Fg lipgloss.AdaptiveColor
    FgMuted lipgloss.AdaptiveColor
    BgAlt lipgloss.AdaptiveColor
    Accent lipgloss.AdaptiveColor
    AccentAlt lipgloss.AdaptiveColor
    Ok lipgloss.AdaptiveColor
    Warn lipgloss.AdaptiveColor
    Danger lipgloss.AdaptiveColor
    Hint lipgloss.AdaptiveColor
}

var Theme = Palette{
    Fg:       lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#e5e5e5"},
    FgMuted:  lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"},
    BgAlt:    lipgloss.AdaptiveColor{Light: "#f3f4f6", Dark: "#1f2937"},
    Accent:   lipgloss.AdaptiveColor{Light: "#6e79d8", Dark: "#8ea2ff"},
    AccentAlt:lipgloss.AdaptiveColor{Light: "#a78bfa", Dark: "#b79bff"},
    Ok:       lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#22c55e"},
    Warn:     lipgloss.AdaptiveColor{Light: "#f59e0b", Dark: "#fbbf24"},
    Danger:   lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#ef4444"},
    Hint:     lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"},
}

func StyleTitle() lipgloss.Style {
    return lipgloss.NewStyle().Bold(true).Foreground(Theme.Accent)
}
func StyleMuted() lipgloss.Style  { return lipgloss.NewStyle().Foreground(Theme.FgMuted) }
func StyleOk() lipgloss.Style     { return lipgloss.NewStyle().Foreground(Theme.Ok) }
func StyleDanger() lipgloss.Style { return lipgloss.NewStyle().Foreground(Theme.Danger) }
func StyleWarn() lipgloss.Style   { return lipgloss.NewStyle().Foreground(Theme.Warn) }
func StyleBadge() lipgloss.Style  { return lipgloss.NewStyle().Foreground(Theme.Fg).Background(Theme.BgAlt).Padding(0,1).Bold(true) }
```

Replace all hard-coded colors in `ui/*.go` with these tokens.

## Components

### 1) Instances List

**File**: `ui/list.go`

* Row structure:

  * Prefix index `1.` `2.` … up to `9.` then `10.` etc.
  * Title, cut with ellipsis if too long.
  * Right side badges:

    * Status: `●` Ready (green), spinner Running, `⏸` Paused (grey).
    * Diff: `+X/−Y` colored (only when non-zero).
    * Mode: `[Direct]` badge when in direct mode.
    * Repo/branch line under title: `Ꮧ branch (repo)`; hide repo if only one repo in play.
* Selection highlights: invert fg/bg using theme; no double borders.
* Click support: map mouse X/Y to row index and call `SetSelectedInstance(idx)`.

**Changes**:

* Replace per-file styles with tokens.
* Add `Direct` badge: when `instance.DirectMode`, append `StyleBadge().Render("Direct")` to the right cluster.
* Add `OnClick(x,y)` method on `List` to find index by row height; call from `app.Update` on `tea.MouseMsg`.

### 2) Tabs

**File**: `ui/tabbed_window.go`

* Tab labels:

  * `Preview`
  * `Diff (+12/−3)` when counts exist; blank counts when zero.
* Tab hit area clickable.
* Active tab border uses Accent; inactive uses AccentAlt with faded bottom.

**Changes**:

* Add `SetTabCounts(added, removed int)` to update Diff title.
* Add `HitTestTab(x,y) (idx int, ok bool)` returning 0/1; call from mouse handler.
* Drop `AdjustPreviewWidth`; let the container give full width; handle inner padding with Lipgloss.

### 3) Preview Pane

**File**: `ui/preview.go`

* States:

  * Empty: simple centered hint: “No sessions yet. Press ‘n’ to create one.”
  * Paused: yellow hint: “Paused. Press ‘r’ to resume.” plus branch line copied note.
  * Normal: last lines; bottom one-line footer hidden by default.
  * Scroll mode: footer shows `PgUp/PgDn Home/End Esc to exit`.
* Footer style: muted; sticky at bottom; never scrolls up with text.
* Content wrap: never wrap; show horizontal scroll? In TUI, keep no wrap; cut off.

**Changes**:

* Keep a `showFooter bool` flag: set true only in scroll mode.
* Render with `lipgloss.JoinVertical`: viewport content then footer line built with `StyleMuted`.

### 4) Diff Pane

**File**: `ui/diff.go`

* Top header: `+X additions / −Y deletions` in green/red, centered.
* Below: file list (optional, small) with current file highlighted; keys `{`/`}` move file; `[`/`]` move hunk.
* Body: color rules already good. Keep empty-state centered “No changes”.

**Changes**:

* Add a tiny file drawer above the diff. Add a field `files []string` and `cursor int`.
* Parse files from existing `stats.Content`:

  * lines starting with `diff --git a/... b/...` → filename.
* New methods:

  * `JumpNextFile()` `JumpPrevFile()`
  * `JumpNextHunk()` `JumpPrevHunk()` search for `@@`.
* Key plumbing in `app.Update` to call into these when Diff tab active.

### 5) Menu

**File**: `ui/menu.go`

* Rebuild menu from groups instead of index math.

Groups:

* Session: `n` New, `N` New+Prompt, `D` Kill
* Actions: `enter` Attach, `p` Push, `c` Checkout / `r` Resume (choose one)
* Nav/System: `tab` Switch, `?` Help, `q` Quit

Rules:

* Dim items not usable now (e.g., `r` when not paused).
* When in Preview scroll mode, show `PgUp/PgDn` hint tail on the right; when in Diff tab, show `[` `]` `{` `}` hint tail.

**Changes**:

* Replace `updateOptions` with group-driven build:

  ```go
  type Group struct { Items []keys.KeyName }
  var groups = []Group{ {...}, {...}, {...} }
  ```
* Add `func (m *Menu) renderItem(k keys.KeyName, enabled bool, underline bool) string`.
* Get underline from `m.keyDown`.

### 6) Overlays

**Files**: `ui/overlay/*.go`

* **ConfirmationOverlay**

  * Text: `Kill session 'name'?` with `(y = yes, n/esc = cancel)`.
  * Red border (Danger).
  * Fixed width 60 chars; height sized to content.
  * Centered both ways; dim the background; do not over-dim colored text.

* **TextInputOverlay**

  * Title line bold in Accent.
  * Single input with clear button focus state.
  * Submit button label matches action: `Create`, `Send`, etc.
  * Footer: `Tab` to switch, `Esc` to cancel.

* **TextOverlay**

  * Title optional; for Help, use bold headings, bullets, no ascii art.

**Changes**:

* Replace ad-hoc styles with Theme tokens.
* Add footer hints to TextInputOverlay.

### 7) Error Box

**File**: `ui/err.go`

* Keep one line, but wrap soft within width; if >1 line, append `…` and provide `?` to open a TextOverlay with the full error.
* Color: Danger, not pure bright red when in dark theme; use `Theme.Danger`.

**Changes**:

* Remove the `//` newline hack; use a small string wrap:

  ```go
  func wrap(s string, w int) string { /* break on spaces; append … */ }
  ```

### 8) Help and First-run

**File**: `app/help.go`

* First launch only: one small card:

  * Top keys: `n` new, `enter` attach, `tab` switch, `ctrl+q` detach, `?` help.
  * 5 lines max. No banner art.
* `?` help: group keys by Session / Actions / Nav / Scroll. Show both Preview and Diff scroll keys.

**Changes**:

* Replace long help text with compact lists pulled from the same key table used by Menu so they never diverge.

## Mouse handling

**File**: `app/app.go`

* In `case tea.MouseMsg`:

  * If click within tab row → `m.tabbedWindow.ToggleWithReset(selected)`.
  * If click within list pane → compute row and `m.list.SetSelectedInstance(row)`.
  * Wheel: if active tab == Preview, call `preview.ScrollUp/Down`; else `diff.ScrollUp/Down`.

Add helper in list:

```go
// Returns index for a y within the list area or -1.
func (l *List) HitTest(y int) int { /* compute row based on title+desc block height */ }
```

Add helper in tabbed window:

```go
func (w *TabbedWindow) HitTestTab(x, y int) (idx int, ok bool) { /* within top row bounds */ }
```

## Layout rules

**File**: `app/app.go: updateHandleWindowSizeEvent`

* If width < 90 cols:

  * Set list to full width; fixed height = 35% of screen; tabs below get rest.
* Else:

  * Keep side-by-side; list width 30%, tabs 70%.
* Menu row always 1 line (plus 1 for error bar).
* Do not shrink content by 10%; remove `AdjustPreviewWidth`. Instead, give inner panes a padding style.

**TabbedWindow**:

* Replace width math with:

  ```go
  window := windowStyle.
      Padding(0,1).
      Width(w.width - windowStyle.GetHorizontalFrameSize()).
      Render(content)
  ```

## Key tables, single source

**File**: `keys/keys.go`

* Add bindings for Preview scroll and Diff jumps. Add `Numbers` slice for `1..9` and a helper to map runes.

```go
// New
KeyPgUp, KeyPgDn, KeyHome, KeyEnd, KeyGoTop, KeyGoBottom, KeyHalfUp, KeyHalfDown
KeyFilePrev, KeyFileNext, KeyHunkPrev, KeyHunkNext
KeyNum1..KeyNum9, KeyNum0
```

The Menu and Help must pull label text from `key.Binding.Help()` so UI strings do not drift from actual bindings.

## Edge cases

* Long titles: cut with ellipsis in the title line; branch line may be blank if no room.
* Many instances: when index > 99, show `..` prefix, but the row still selectable with arrows; number-jump limited to first 10.
* High diff counts: clamp display to 4 digits; on overflow show `9,999+`.

## Acceptance checks

* Start with empty repo:

  * See compact first-run card; `n` brings New overlay; `N` shows prompt overlay; `Esc` cancels.
* With two sessions running:

  * List shows spinner/ready/paused icons and a `[Direct]` badge for direct mode.
  * Click on tab label switches tabs.
  * Diff tab title shows counts; counts hide when zero.
* Preview scroll:

  * Tap `PgUp` → footer appears; `Esc` exits; `Home` jumps to start; `End` to end.
* Diff nav:

  * `[`, `]` jump hunks; `{`, `}` jump files; current file highlighted.
* Menu:

  * When paused, `c` hidden, `r` shown; push dim when no changes.
* Error box:

  * Inject a long error; see wrapped one-liner; `?` opens full error overlay; `Esc` closes.
* Narrow width:

  * List stacks above tabs; titles cut cleanly; menu is still on one line.

## File-by-file change list

* `ui/theme.go` new token file; import in `ui/*.go`.
* `ui/list.go` use tokens; add Direct badge; add `HitTest(y)`; add click support.
* `ui/tabbed_window.go` clickable tabs; `SetTabCounts`; drop `AdjustPreviewWidth`; use padding.
* `ui/preview.go` footer hints; scroll footer on; clear empty/paused hints; use tokens.
* `ui/diff.go` header counts; small file drawer with jump funcs; color from tokens.
* `ui/menu.go` group-driven build; dim disabled; context hints for scroll.
* `ui/err.go` soft wrap; full error overlay hook.
* `app/app.go` mouse click routing to list/tabs; preview/diff wheel; minor layout logic for stacked mode.
* `keys/keys.go` new bindings for scroll/jumps and number keys; ensure `GlobalkeyBindings` covers them.
* `app/help.go` compact help; pull strings from `keys`.
* `ui/overlay/*` unify styles to Theme, add footer hints to text input.
* `ui/consts.go` remove banner ASCII or gate it behind very wide width.

## Developer notes

* Keep **no** ascii art in fallback; it breaks small terms.
* Always show what the next step is. Empty state must tell the key to press.
* Keep menu under 80 chars when possible; if it spills, shorten labels, not keys.
* Test with dark and light terminals; contrast must be legible in both.

## Tiny code sketches

Clickable tabs:

```go
// app/app.go in tea.MouseMsg
if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
    if idx, ok := m.tabbedWindow.HitTestTab(msg.X, msg.Y); ok {
        if idx != m.tabbedWindow.GetActiveTab() {
            _ = m.tabbedWindow.ToggleWithReset(m.list.GetSelectedInstance())
            m.menu.SetInDiffTab(m.tabbedWindow.IsInDiffTab())
            return m, m.instanceChanged()
        }
    }
    if row := m.list.HitTest(msg.Y); row >= 0 {
        m.list.SetSelectedInstance(row)
        return m, m.instanceChanged()
    }
}
```

Menu group render:

```go
type Group struct{ Items []keys.KeyName }
var groups = []Group{
  {[]keys.KeyName{keys.KeyNew, keys.KeyPrompt, keys.KeyKill}},
  {[]keys.KeyName{keys.KeyEnter, keys.KeySubmit /* or KeyCheckout/KeyResume */}},
  {[]keys.KeyName{keys.KeyTab, keys.KeyHelp, keys.KeyQuit}},
}
```

Diff header:

```go
header := lipgloss.JoinHorizontal(lipgloss.Center,
    StyleOk().Render(fmt.Sprintf("+%d additions", added)),
    lipgloss.NewStyle().Width(2).Render(" "),
    StyleDanger().Render(fmt.Sprintf("-%d deletions", removed)),
)
d.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, header, fileBar, coloredBody))
```

Text input overlay footer:

```go
footer := StyleMuted().Render("Tab to switch • Enter to submit • Esc to cancel")
return style.Render(title + "\n" + t.textarea.View() + "\n\n" + buttons + "\n" + footer)
```

## Web docs nits (keep in sync)

* Fix shell install URL: `smtg-ai` not `stmg-ai` (`web/src/app/page.tsx`).
* Add a tiny key table in README that matches Menu groups.
* Keep the screenshot current after the TUI paint.

That’s all. Build to this blueprint and the tool will feel crisp: clear state, clear keys, sober look, and no hidden moves.
