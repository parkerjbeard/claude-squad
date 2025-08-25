# Claude Squad UI Overhaul - Feature Implementation Plan

## Current Architecture Analysis

**Tech Stack:**
- Go with Charm/BubbleTea TUI framework
- Lipgloss for styling and layout
- go-git for git operations
- tmux for terminal session management

**Current Structure:**
- `app/` - Application logic and state management
- `ui/` - UI components (List, Menu, TabbedWindow, Preview, Diff, ErrBox)
- `ui/overlay/` - Modal overlays (Confirmation, TextInput, Text)
- `keys/` - Keyboard binding definitions
- `session/` - Instance management and tmux integration

---

## Batch 1: Theme System & Visual Foundation

**Context:** Establish a centralized theming system to replace scattered color definitions and magic numbers throughout the codebase. This provides the foundation for all subsequent visual changes and ensures consistency across light/dark terminals.

### Features:
- **Create Theme System**
  - New file `ui/theme.go` with Palette struct
  - Define adaptive colors for light/dark terminals
  - Export Theme instance with all color tokens
  - Create style helper functions (StyleTitle, StyleMuted, StyleOk, StyleDanger, StyleWarn, StyleBadge)

- **Apply Theme Tokens**
  - Replace hard-coded colors in `ui/list.go`
  - Replace hard-coded colors in `ui/menu.go`
  - Replace hard-coded colors in `ui/preview.go`
  - Replace hard-coded colors in `ui/diff.go`
  - Replace hard-coded colors in `ui/err.go`
  - Replace hard-coded colors in `ui/tabbed_window.go`
  - Replace hard-coded colors in all overlay files (`ui/overlay/*.go`)

- **Remove Magic Numbers**
  - Audit all UI files for hardcoded dimensions
  - Replace with calculated values or theme constants
  - Remove ASCII art banner or gate behind width check

**Dependencies:** None - foundational batch

---

## Batch 2: Enhanced Key Bindings & Input Model

**Context:** Expand and unify keyboard input handling to support new navigation features like scroll modes, diff jumping, and number-based instance selection. This creates the infrastructure needed for enhanced interaction patterns.

### Features:
- **Expand Key Definitions**
  - Add scroll keys in `keys/keys.go` (PgUp, PgDn, Home, End, g/G, Ctrl+u/d)
  - Add diff navigation keys ([, ], {, })
  - Add number keys (1-9, 0) for instance selection
  - Add Alt+number combinations for tens selection (optional)
  - Ensure GlobalKeyBindings covers all new keys

- **Create Unified Key Table**
  - Define key groups (Session, Actions, Nav/System)
  - Create mapping structure for menu and help generation
  - Add Help() method to retrieve key descriptions
  - Implement key state tracking for underline display

- **Input State Management**
  - Add scroll mode state to Preview
  - Add file/hunk cursor state to Diff
  - Track active input context for contextual hints
  - Implement number key accumulator for instance selection

**Dependencies:** Batch 1 (uses theme system for key hint styling)

---

## Batch 3: List Component Overhaul

**Context:** Transform the instance list into a fully interactive component with clear visual hierarchy, status badges, and mouse support. This is the primary navigation element users interact with.

### Features:
- **Visual Structure**
  - Implement numbered prefix (1., 2., ... 10.)
  - Add title with ellipsis truncation
  - Create right-side badge cluster layout
  - Add repo/branch line with git icon (Ꮧ)
  - Implement selection highlighting with theme colors

- **Status Indicators**
  - Ready state: green dot (●)
  - Running state: spinner animation
  - Paused state: grey pause icon (⏸)
  - Direct mode: [Direct] badge
  - Diff counts: +X/−Y colored badges (when non-zero)

- **Mouse Support**
  - Add `HitTest(y int) int` method
  - Calculate row index from Y position
  - Handle click-to-select functionality
  - Integrate with app mouse event handler

- **Edge Cases**
  - Handle long titles with ellipsis
  - Support >99 instances with ".." prefix
  - Hide repo name when single repo
  - Graceful degradation for narrow widths

**Dependencies:** Batch 1 (theme), Batch 2 (number key selection)

---

## Batch 4: Tabbed Window Enhancement

**Context:** Upgrade the tab system to be fully interactive with dynamic tab titles, mouse support, and proper width handling. Remove problematic width adjustments and use proper padding instead.

### Features:
- **Tab Headers**
  - Static "Preview" label
  - Dynamic "Diff (+X/−Y)" with counts
  - Hide counts when zero
  - Active/inactive styling with theme colors
  - Bottom border differentiation

- **Mouse Interaction**
  - Add `HitTestTab(x, y int) (idx int, ok bool)` method
  - Implement click-to-switch functionality
  - Handle tab hit areas properly

- **Layout Improvements**
  - Remove `AdjustPreviewWidth` method
  - Use Lipgloss padding for inner content
  - Calculate proper content width with frame size
  - Ensure consistent spacing

- **Tab State Management**
  - Add `SetTabCounts(added, removed int)` method
  - Track active tab state
  - Handle tab switching with content reset

**Dependencies:** Batch 1 (theme), Batch 3 (for integrated interaction)

---

## Batch 5: Preview Pane Advanced Features

**Context:** Transform the preview pane into a stateful component with clear empty states, scroll mode, and contextual hints. This provides users with better log viewing capabilities and clearer state communication.

### Features:
- **State-Based Display**
  - Empty state: "No sessions yet. Press 'n' to create one."
  - Paused state: Yellow "Paused. Press 'r' to resume." + branch info
  - Normal state: Latest log lines
  - Scroll mode: Content with footer hints

- **Scroll Mode Implementation**
  - Track `scrollMode` boolean state
  - Handle PgUp/PgDn, Home/End, g/G, Ctrl+u/d keys
  - Show footer only in scroll mode
  - Exit scroll with Esc key
  - Maintain viewport position

- **Footer System**
  - Sticky footer at bottom (never scrolls)
  - Show "PgUp/PgDn Home/End Esc to exit" in scroll mode
  - Use muted style from theme
  - Render with `lipgloss.JoinVertical`

- **Content Handling**
  - No text wrapping (maintain formatting)
  - Handle horizontal overflow with truncation
  - Preserve ANSI colors in output
  - Smooth scrolling behavior

**Dependencies:** Batch 1 (theme), Batch 2 (scroll keys), Batch 4 (tab integration)

---

## Batch 6: Diff Pane Navigation Features

**Context:** Add advanced navigation capabilities to the diff pane including file jumping, hunk navigation, and a file drawer for better orientation in large diffs.

### Features:
- **Header Display**
  - Centered "+X additions / −Y deletions" in green/red
  - Use theme colors for consistency
  - Dynamic width adjustment

- **File Drawer**
  - Parse file list from diff content
  - Track current file with cursor
  - Highlight active file
  - Compact display above diff content

- **Navigation Methods**
  - `JumpNextFile()` / `JumpPrevFile()` with {/} keys
  - `JumpNextHunk()` / `JumpPrevHunk()` with [/] keys
  - Parse @@ markers for hunk boundaries
  - Maintain scroll position context

- **Visual Improvements**
  - Keep existing diff coloring
  - Add file boundaries visual markers
  - Centered "No changes" empty state
  - Proper spacing with theme system

**Dependencies:** Batch 1 (theme), Batch 2 (navigation keys), Batch 4 (tab counts)

---

## Batch 7: Menu System Redesign

**Context:** Rebuild the menu as a group-based, context-aware component that clearly shows available actions and provides dynamic hints based on the current UI state.

### Features:
- **Group-Based Architecture**
  - Define menu groups (Session, Actions, Nav/System)
  - Replace index-based option management
  - Dynamic group composition based on state
  - Consistent ordering and spacing

- **State-Aware Display**
  - Dim unavailable actions
  - Show/hide context-specific options (resume vs checkout)
  - Underline on key press
  - Dynamic width management

- **Contextual Hints**
  - Show scroll hints when in Preview scroll mode
  - Show diff navigation hints when in Diff tab
  - Adapt hints based on active component
  - Right-align hint text

- **Visual Consistency**
  - Use theme colors for all text
  - Maintain <80 char width when possible
  - Group separator spacing
  - Consistent key formatting

**Dependencies:** Batch 1 (theme), Batch 2 (key definitions), Batch 5 & 6 (for context hints)

---

## Batch 8: Overlay Improvements

**Context:** Enhance all modal overlays with consistent styling, proper keyboard hints, and improved user feedback. These provide critical interaction points for user input and confirmations.

### Features:
- **Confirmation Overlay**
  - Format: "Kill session 'name'?" with instructions
  - Red border using Theme.Danger
  - Fixed 60 char width
  - Centered positioning
  - Background dimming without over-dimming

- **Text Input Overlay**
  - Bold title in Accent color
  - Clear input focus state
  - Dynamic submit button labels
  - Footer: "Tab to switch • Esc to cancel"
  - Proper tab navigation

- **Text Overlay (Help)**
  - Optional title support
  - Structured content with headings
  - No ASCII art
  - Scrollable for long content
  - Consistent styling

- **General Improvements**
  - Apply theme tokens throughout
  - Consistent escape handling
  - Proper z-index/layering
  - Smooth show/hide transitions

**Dependencies:** Batch 1 (theme), Batch 2 (key table for help)

---

## Batch 9: Error Handling & Display

**Context:** Improve error communication with smart wrapping, expandable details, and better visual treatment. This ensures users understand issues without cluttering the interface.

### Features:
- **Smart Error Display**
  - Single-line with soft wrap
  - Ellipsis for overflow
  - Use Theme.Danger (not bright red)
  - Remove // newline hack

- **Expandable Details**
  - Track full error message
  - Show "?" hint for expansion
  - Open TextOverlay with full error
  - Preserve error history (optional)

- **Error Box Improvements**
  - Implement `wrap(s string, w int)` helper
  - Handle multi-line errors gracefully
  - Clear error on action
  - Timeout for auto-clear (optional)

**Dependencies:** Batch 1 (theme), Batch 8 (TextOverlay for expansion)

---

## Batch 10: Mouse Support Implementation

**Context:** Add comprehensive mouse support throughout the application, enabling click-to-select, wheel scrolling, and click-to-switch for tabs. This provides an alternative interaction model for mouse users.

### Features:
- **Global Mouse Handling**
  - Update `app.go` tea.MouseMsg handler
  - Route clicks to appropriate components
  - Handle wheel events for scrolling
  - Coordinate between components

- **Component Integration**
  - List: Click to select instance
  - Tabs: Click to switch
  - Preview: Wheel to scroll
  - Diff: Wheel to scroll
  - Menu: Hover highlights (optional)

- **Event Routing**
  - Implement hit testing for all clickable areas
  - Priority-based event handling
  - Prevent event bubbling
  - Handle drag events (optional)

**Dependencies:** Batch 3 (list HitTest), Batch 4 (tab HitTest), Batch 5 & 6 (scroll support)

---

## Batch 11: Responsive Layout System

**Context:** Implement adaptive layouts that gracefully handle different terminal sizes, with special handling for narrow terminals (<90 columns) by stacking components vertically.

### Features:
- **Layout Detection**
  - Check terminal width in `updateHandleWindowSizeEvent`
  - Define breakpoint at 90 columns
  - Calculate component dimensions dynamically

- **Wide Layout (≥90 cols)**
  - Side-by-side: List 30%, Tabs 70%
  - Full-height components
  - Horizontal spacing between panels

- **Narrow Layout (<90 cols)**
  - Stacked: List on top (35% height)
  - Tabs below (remaining height)
  - Full width for both components
  - Adjusted menu sizing

- **Component Adjustments**
  - Remove 10% content shrinking
  - Use padding styles instead
  - Maintain readable minimums
  - Handle extreme sizes gracefully

**Dependencies:** All previous batches for proper component behavior

---

## Batch 12: Help System & Onboarding

**Context:** Create a streamlined help system and first-run experience that teaches the interface in 30 seconds, with compact, actionable information pulled from the unified key table.

### Features:
- **First-Run Card**
  - Detect first launch
  - Show 5-line quick start
  - Key commands: n, enter, tab, ctrl+q, ?
  - No banner art
  - Dismissible with any key

- **Help Overlay**
  - Pull from unified key table
  - Group by category (Session/Actions/Nav/Scroll)
  - Show both Preview and Diff scroll keys
  - Compact, scannable format
  - Version/about info at bottom

- **Integration**
  - Store "seen help" in AppState
  - Access with ? key globally
  - Context-sensitive help (optional)
  - Keep in sync with menu

**Dependencies:** Batch 2 (key table), Batch 8 (TextOverlay)

---

## Testing & Polish Batch

**Context:** Final validation and polish pass to ensure all features work correctly together and edge cases are handled properly.

### Features:
- **Acceptance Testing**
  - Empty repo state validation
  - Two-session interaction test
  - Scroll mode verification
  - Diff navigation test
  - Menu state validation
  - Error handling test
  - Narrow width validation

- **Edge Case Handling**
  - >99 instances display
  - Very long titles/paths
  - High diff counts (9,999+)
  - Minimal terminal sizes
  - Rapid state changes
  - Concurrent operations

- **Performance Optimization**
  - Profile rendering performance
  - Optimize builder pool usage
  - Reduce allocations in hot paths
  - Cache computed layouts

- **Documentation Updates**
  - Update README with new key table
  - Fix shell install URL in web docs
  - Update screenshots
  - Create user guide

---

## Implementation Order & Dependencies

1. **Batch 1** - Theme System (Foundation - no dependencies)
2. **Batch 2** - Key Bindings (Depends on Batch 1)
3. **Batch 3** - List Component (Depends on Batches 1, 2)
4. **Batch 4** - Tabbed Window (Depends on Batches 1, 3)
5. **Batch 5** - Preview Pane (Depends on Batches 1, 2, 4)
6. **Batch 6** - Diff Pane (Depends on Batches 1, 2, 4)
7. **Batch 7** - Menu System (Depends on Batches 1, 2, 5, 6)
8. **Batch 8** - Overlays (Depends on Batches 1, 2)
9. **Batch 9** - Error Handling (Depends on Batches 1, 8)
10. **Batch 10** - Mouse Support (Depends on Batches 3-6)
11. **Batch 11** - Responsive Layout (Depends on all component batches)
12. **Batch 12** - Help System (Depends on Batches 2, 8)
13. **Testing & Polish** - Final batch after all features complete

## Key Research Details from Spec

- **Charm/Lipgloss specifics**: Use `lipgloss.JoinVertical` and `lipgloss.JoinHorizontal` for layouts
- **Adaptive colors**: Must work in both light and dark terminals
- **Frame size calculation**: Use `GetHorizontalFrameSize()` for proper width calculations
- **Mouse events**: Already configured with `tea.WithMouseCellMotion()`
- **Spinner models**: Existing spinner integration to preserve
- **Git integration**: Maintain existing git/tmux logic, only change UI layer