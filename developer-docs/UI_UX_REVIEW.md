# UI/UX Review — Claude Squad

This report evaluates the terminal UI (TUI) and the web docs site, and proposes concrete, prioritized improvements to usability, clarity, and maintainability. Recommendations are based on a code-level pass of the repository and current behavior expressed in the UI code.

## Scope
- TUI: `app/`, `ui/`, `keys/`, `session/` integration (git/tmux), help + overlays.
- Web: `web/` (Next.js site), install instructions, theming/accessibility.

---

## Highlights (What Works)
- Clear layout: list of sessions + tabbed window (Preview/Diff) + bottom menu + error line.
- Good status feedback: running/ready/paused indicators, spinner; per-item diff stats in list.
- Familiar navigation: `j/k`, arrows, `tab` to switch tabs, mouse wheel scrolling.
- Helpful overlays: on-first-run help, confirmation dialogs, prompt overlay; consistent visual language.
- Persistence: sessions/state survive restarts; progressive help via seen-bitmask.

---

## TUI Findings & Recommendations

<!-- ### 1) Information Architecture & Layout
- Issue: Fixed splits (list ~30%, preview ~70%) and an extra 90% width reduction in the tab window can cause truncation and unused whitespace on small terminals.
- Recommendation:
  - Make layout adaptive: on narrow widths switch to stacked (list above tabs). Reduce padding/borders.
  - Remove or conditionally disable the extra `AdjustPreviewWidth(0.9*width)` at small widths.

Code pointers:
- `app/app.go`: `updateHandleWindowSizeEvent`
- `ui/tabbed_window.go`: `AdjustPreviewWidth`, `SetSize`, and framing math -->

### 2) Visual Design & Theming
- Issue: Inconsistent color usage (mix of hex and 256-color codes), inconsistent contrast; repeated style definitions.
- Recommendation:
  - Centralize a theme palette in a new `ui/theme.go` with named tokens (foreground, muted, accent, success, danger, warning, selection, bgAlt). Use `AdaptiveColor` consistently.
  - Audit contrast for light/dark terminals; target WCAG-ish contrast where reasonable for TUIs.

Code pointers:
- `ui/list.go` (title/desc/selected styles, ready/paused/added/removed colors)
- `ui/menu.go` (key/desc/sep colors, actionGroupStyle)
- `ui/preview.go`, `ui/diff.go` (pane and diff colors)

### 3) State & Feedback (Empty/Paused/Error)
- Empty state:
  - Issue: Large ASCII fallback can overflow small terminals.
  - Recommendation: Gate the ASCII banner behind a width threshold; otherwise show a compact message with CTA “Press n to create a session”.
- Paused state:
  - Issue: The message mentions the branch is copied to clipboard, but there’s no explicit menu action.
  - Recommendation: Add a visible “Yank branch (y)” action that re-copies and flashes a confirmation in the info/error line.
- Error line:
  - Issue: Newlines flattened to `//`, hard truncation.
  - Recommendation: Soft-wrap within width, keep a max of N lines, then append “…”. Provide a “Press ? for details” overlay when longer.

Code pointers:
- `ui/consts.go` (fallback), `ui/preview.go` (fallback assembly & centering)
- `ui/err.go` (rendering and truncation behavior)

### 4) Navigation & Interaction
- Labels & clarity:
  - Rename “Submit” → “Push”, “Open” → “Attach” in menu/help/readme.
  - Show a small “Direct” badge on the branch line when in direct mode. Disable/rename “Checkout” accordingly.
- Scroll affordance:
  - Add `pgup/pgdn/home/end` for preview/diff; show footer hint when in scroll mode: “PgUp/PgDn, ESC exits”.
- Quick select:
  - Allow number keys `1..9` to jump to sessions.
- Mouse:
  - Make tabs clickable; allow click-to-select list items.

Code pointers:
- `ui/menu.go` (labels, grouping)
- `keys/keys.go` (keymap additions)
- `ui/preview.go`, `ui/diff.go` (scroll handlers and footers)
- `app/app.go` (mouse handling & key dispatch)

### 5) Menu Structure & Resilience
- Issue: Option groups are hard-coded by indices, so separators/highlighting break when options change.
- Recommendation: Replace index math with structured groups, e.g., `[]Group{ {name: "Instance", options: [...]}, {name: "Actions", ...}, {name: "System", ...} }`. Drive separators and highlighting by group boundaries. Dim disabled actions rather than hiding them.

Code pointers:
- `ui/menu.go` (`groups := []struct{start,end int}{...}` and `addInstanceOptions`)

### 6) Diff Experience
- Current: A single scrollable diff view with colored additions/deletions and hunk headers.
- Improvements:
  - Add per-file summary above diff with counts; allow jumping by file/hunk.
  - Update the Diff tab label to include counts, e.g., `Diff (+4/−1)`.

Code pointers:
- `ui/diff.go` (expand Model to maintain file list; compute from `stats.Content`)
- `app/app.go` (update tab label based on `DiffStats`)

### 7) Performance & Responsiveness
- Preview/Diff polling already throttled; good use of async and timeouts.
- Opportunity: Cache the dimmed background in `overlay.PlaceOverlay` to avoid regex replacements each frame.

Code pointers:
- `ui/overlay/overlay.go` (ANSI color regex and replacement in `PlaceOverlay`)

---

## Web Docs Findings & Recommendations

### Strengths
- Clean, minimal landing with clear call-to-actions; copy-to-clipboard and theme toggle are handy.
- Responsive layout is solid; install snippets easy to copy.

### Issues
- Install URL typo:
  - In `web/src/app/page.tsx`, fix `stmg-ai` → `smtg-ai` for the curl command.
- Theme flashing (FOUC/FART):
  - The component defaults to `light` and sets `data-theme` post-mount, overriding system preference on first load.
- Video UX:
  - Add a `poster` image; consider disabling autoplay on mobile.
- Accessibility:
  - Ensure link/button contrast meets thresholds in both themes.
  - Provide alt text/titles, maintain focus outlines.
- Content clarity:
  - Replace “10x your productivity” with outcome-focused phrasing.
  - Add a concise “First steps”: run `cs`, press `n`, switch tabs, push/checkout.
  - Add a Troubleshooting section (tmux, gh auth, repo requirement), and a keymap table mirroring the TUI help.
- Consistency:
  - Use one product name formatting consistently (“Claude Squad”). Mirror README flags, including direct-mode caveats.

Code pointers:
- `web/src/app/page.tsx` (install curl link)
- `web/src/app/components/ThemeToggle.tsx` (initialize from `prefers-color-scheme` if no saved value)
- `web/src/app/page.module.css` (contrast checks)

---

## Notable Code Observations (Bugs/Risks)
- Double finalizer call:
  - `app/app.go` (stateNew, after `Start`): `m.newInstanceFinalizer()` is invoked twice; can double-register repo counts.
- Menu grouping brittleness:
  - `ui/menu.go`: group boundaries via static indices; error-prone when options vary with state.
- Style reuse inconsistency:
  - `ui/preview.go`: footer style is rebuilt in `ScrollDown` instead of using `previewFooterStyle`.
- Double-shrinking of widths:
  - `AdjustPreviewWidth` + `List.SetSize` interplay leads to compounded width reduction; logic should be consolidated.

---

## Prioritized Roadmap

### Phase 1 — Quick Wins (1–2 days)
- Rename labels: “Submit”→“Push”, “Open”→“Attach” in menu/help/README.
- Diff tab count: Show `(+added/−removed)` in the Diff tab title.
- Scrolling UX: Add `PgUp/PgDn/Home/End`; show footer hint in scroll mode.
- Error wrapping: Preserve line breaks; soft-wrap within width; cap visible lines.
- Web: Fix curl URL; initialize theme from system preference if no saved theme.

### Phase 2 — Structural Polish (2–4 days)
- Menu grouping: Refactor to explicit groups with disabled/dim states.
- Layout adaptivity: Stack list above tabs on narrow widths; reduce padding.
- Direct-mode affordance: Badge on branch line; adjust action labels/availability.
- Centralize theming: `ui/theme.go` with named tokens.

### Phase 3 — Experience Upgrades (4–8 days)
- Clickable tabs and list selection via mouse.
- Diff improvements: Per-file list, hunk navigation, and tab label counts.
- Details overlay: Session info (path/branch/program/mode/timestamps), rename, per-session Auto-Yes toggle.
- Overlay performance: Cache dimmed background.

---

## Suggested Implementation Notes
- Keys
  - Add bindings for `pgup/pgdn/home/end` and `1..9` mapping to select session n.
  - Keep ESC-to-exit-scroll; add footer hint on first entry to scroll mode.
- Menu
  - Represent as `[]Group{}`; separators derived from group boundaries rather than indices. Maintain underline-on-key-down across groups.
- Diff tab label
  - Store counts in `DiffPane` or derive from `DiffStats`; render into tab titles via `TabbedWindow`.
- Errors
  - In `ui/err.go`, replace `//` substitution with newline-preserving wrap; optionally show an overlay for longer messages when `?` is pressed.
- Web theming
  - On mount: if `localStorage.theme` is unset, detect `window.matchMedia('(prefers-color-scheme: dark)')` and set `data-theme` accordingly.

---

## Appendix

### A. Potential Theme Tokens
- `fg`, `fgMuted`, `bg`, `bgAlt`, `accent`, `accentAlt`, `success`, `danger`, `warning`, `selection`, `separator`.

### B. Copy Recommendations
- Change “10x your productivity” → “Run multiple AI agents in parallel, review diffs, and ship confidently.”
- Empty state minimal CTA: “No sessions yet. Press n to create one.”
- Paused state: “Session paused. Branch name copied. Press r to resume, y to copy again.”

### C. Keymap (additions)
- Scroll: `PgUp/PgDn/Home/End`
- Select: `1..9` to jump to nth session
- Yank branch (paused): `y`

---

Prepared by: UI/UX Review
Date: 2025-08-22

