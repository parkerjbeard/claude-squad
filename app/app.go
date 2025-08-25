package app

import (
	"claude-squad/config"
	"claude-squad/keys"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/session/git"
	"claude-squad/ui"
	"claude-squad/ui/overlay"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const GlobalInstanceLimit = 10

// Run is the main entrypoint into the application.
func Run(ctx context.Context, program string, autoYes bool, directMode bool, directBranch string) error {
	p := tea.NewProgram(
		newHome(ctx, program, autoYes, directMode, directBranch),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Mouse scroll
	)
	_, err := p.Run()
	return err
}

type state int

const (
	stateDefault state = iota
	// stateNew is the state when the user is creating a new instance.
	stateNew
	// statePrompt is the state when the user is entering a prompt.
	statePrompt
	// stateHelp is the state when a help screen is displayed.
	stateHelp
	// stateConfirm is the state when a confirmation modal is displayed.
	stateConfirm
)

type home struct {
	ctx context.Context

	// -- Storage and Configuration --

	program      string
	autoYes      bool
	directMode   bool
	directBranch string

	// storage is the interface for saving/loading data to/from the app's state
	storage *session.Storage
	// appConfig stores persistent application configuration
	appConfig *config.Config
	// appState stores persistent application state like seen help screens
	appState config.AppState

	// -- State --

	// state is the current discrete state of the application
	state state
	// newInstanceFinalizer is called when the state is stateNew and then you press enter.
	// It registers the new instance in the list after the instance has been started.
	newInstanceFinalizer func()

	// promptAfterName tracks if we should enter prompt mode after naming
	promptAfterName bool

	// keySent is used to manage underlining menu items
	keySent bool

	// -- UI Components --

	// list displays the list of instances
	list *ui.List
	// menu displays the bottom menu
	menu *ui.Menu
	// tabbedWindow displays the tabbed window with preview and diff panes
	tabbedWindow *ui.TabbedWindow
	// errBox displays error messages
	errBox *ui.ErrBox
	// global spinner instance. we plumb this down to where it's needed
	spinner spinner.Model
	// textInputOverlay handles text input with state
	textInputOverlay *overlay.TextInputOverlay
	// textOverlay displays text information
	textOverlay *overlay.TextOverlay
	// confirmationOverlay displays confirmation modals
	confirmationOverlay *overlay.ConfirmationOverlay

	// diff watcher state
    diffWatchInst      *session.Instance
    diffWatchActive    bool
    diffWatchLastDirty bool

    // layout state
    stacked bool
}

func newHome(ctx context.Context, program string, autoYes bool, directMode bool, directBranch string) *home {
	// Load application config
	appConfig := config.LoadConfig()

	// Load application state
	appState := config.LoadState()

	// Initialize storage
	storage, err := session.NewStorage(appState)
	if err != nil {
		fmt.Printf("Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	h := &home{
		ctx:          ctx,
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:         ui.NewMenu(),
		tabbedWindow: ui.NewTabbedWindow(ui.NewPreviewPane(), ui.NewDiffPane()),
		errBox:       ui.NewErrBox(),
		storage:      storage,
		appConfig:    appConfig,
		program:      program,
		autoYes:      autoYes,
		directMode:   directMode,
		directBranch: directBranch,
		state:        stateDefault,
		appState:     appState,
	}
	h.list = ui.NewList(&h.spinner, autoYes)

	// Load saved instances
	instances, err := storage.LoadInstances()
	if err != nil {
		fmt.Printf("Failed to load instances: %v\n", err)
		os.Exit(1)
	}

	// Add loaded instances to the list
	for _, instance := range instances {
		// Call the finalizer immediately.
		h.list.AddInstance(instance)()
		if autoYes {
			instance.AutoYes = true
		}
	}

	return h
}

// updateHandleWindowSizeEvent sets the sizes of the components.
// The components will try to render inside their bounds.
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
    // Breakpoint at 90 columns
    m.stacked = msg.Width < 90

    // Reserve 1 row for error box
    menuHeight := 1
    contentHeight := msg.Height - menuHeight - 1 // account for err box below menu
    m.errBox.SetSize(msg.Width, 1)

    if m.stacked {
        // Stacked: list on top (35% height), tabs below
        listHeight := int(float32(contentHeight) * 0.35)
        if listHeight < 3 {
            listHeight = 3
        }
        tabsHeight := contentHeight - listHeight
        m.list.SetSize(msg.Width, listHeight)
        m.tabbedWindow.SetSize(msg.Width, tabsHeight)
    } else {
        // Wide: side-by-side (30/70)
        listWidth := int(float32(msg.Width) * 0.3)
        tabsWidth := msg.Width - listWidth
        m.list.SetSize(listWidth, contentHeight)
        m.tabbedWindow.SetSize(tabsWidth, contentHeight)
    }

	if m.textInputOverlay != nil {
		m.textInputOverlay.SetSize(int(float32(msg.Width)*0.6), int(float32(msg.Height)*0.4))
	}
	if m.textOverlay != nil {
		m.textOverlay.SetWidth(int(float32(msg.Width) * 0.6))
	}

	previewWidth, previewHeight := m.tabbedWindow.GetPreviewSize()
	if err := m.list.SetSessionPreviewSize(previewWidth, previewHeight); err != nil {
		log.ErrorLog.Print(err)
	}
    m.menu.SetSize(msg.Width, menuHeight)
}

func (m *home) Init() tea.Cmd {
	// Upon starting, we want to start the spinner. Whenever we get a spinner.TickMsg, we
	// update the spinner, which sends a new spinner.TickMsg. I think this lasts forever lol.
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			// Reduce preview polling frequency for lower CPU usage
			time.Sleep(250 * time.Millisecond)
			return previewTickMsg{}
		},
		tickUpdateMetadataCmd,
	)
}

func (m *home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hideErrMsg:
		m.errBox.Clear()
	case previewTickMsg:
		cmd := m.instanceChanged()
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				// Reduce preview polling frequency for lower CPU usage
				time.Sleep(250 * time.Millisecond)
				return previewTickMsg{}
			},
		)
	case keyupMsg:
		m.menu.ClearKeydown()
		return m, nil
	case tickUpdateMetadataMessage:
		// Only update metadata for relevant instances to reduce overhead.
		selected := m.list.GetSelectedInstance()
		now := time.Now()

		var cmds []tea.Cmd
		scheduled := 0
		const maxScheduled = 4 // simple rate limit per tick

		for _, instance := range m.list.GetInstances() {
			if !instance.Started() || instance.Paused() {
				continue
			}

			// Determine whether to process this instance:
			warmup := now.Sub(instance.CreatedAt) < 5*time.Second
			shouldProcess := instance == selected || instance.AutoYes || warmup

			if shouldProcess && scheduled < maxScheduled {
				cmds = append(cmds, makeTmuxStatusCmd(instance))
				scheduled++
			}

			// Diff updates are event-driven; no periodic scheduling here
		}
		// Reschedule next metadata tick and batch async commands
		return m, tea.Batch(append(cmds, tickUpdateMetadataCmd)...)
	case tmuxStatusMsg:
		inst := msg.instance
		if msg.err != nil {
			log.WarningLog.Printf("tmux status error: %v", msg.err)
			return m, nil
		}
		if msg.updated {
			inst.SetStatus(session.Running)
		} else {
			if msg.prompt {
				inst.TapEnter()
			} else {
				inst.SetStatus(session.Ready)
			}
		}
		return m, nil
	case gitDiffMsg:
		inst := msg.instance
		if msg.err != nil {
			log.WarningLog.Printf("git diff error: %v", msg.err)
			return m, nil
		}
		inst.SetDiffStats(msg.stats)
		// Update the diff pane if this is the selected instance
		if inst == m.list.GetSelectedInstance() {
			m.tabbedWindow.UpdateDiff(inst)
		}
		return m, nil
	case diffWatchTickedMsg:
		// Watcher stopped or tab hidden
		if !m.diffWatchActive || m.diffWatchInst == nil || !m.tabbedWindow.IsInDiffTab() {
			m.diffWatchActive = false
			return m, nil
		}
		// On change, compute diff; always continue polling
		var cmds []tea.Cmd
		if msg.changed {
			cmds = append(cmds, makeGitDiffCmd(m.diffWatchInst))
		}
		cmds = append(cmds, m.diffWatchPollCmd())
		return m, tea.Batch(cmds...)
case tea.MouseMsg:
    // Handle mouse wheel events for scrolling the diff/preview pane
    if msg.Action == tea.MouseActionPress {
        if msg.Button == tea.MouseButtonWheelDown || msg.Button == tea.MouseButtonWheelUp {
            selected := m.list.GetSelectedInstance()
            if selected == nil || selected.Status == session.Paused {
                return m, nil
            }

            switch msg.Button {
            case tea.MouseButtonWheelUp:
                m.tabbedWindow.ScrollUp()
            case tea.MouseButtonWheelDown:
                m.tabbedWindow.ScrollDown()
            }
        } else if msg.Button == tea.MouseButtonLeft {
            // Click handling: tabs and list
            // Compute relative coordinates to components (account for top padding of 1 line)
            relY := msg.Y - 1
            // Tabs are on the right after list width
            lw, _ := m.list.GetSize()
            if msg.X >= lw {
                relX := msg.X - lw
                if idx, ok := m.tabbedWindow.HitTestTab(relX, relY); ok {
                    if idx != m.tabbedWindow.GetActiveTab() {
                        _ = m.tabbedWindow.ToggleWithReset(m.list.GetSelectedInstance())
                        m.menu.SetInDiffTab(m.tabbedWindow.IsInDiffTab())
                        return m, m.instanceChanged()
                    }
                }
            } else {
                // Inside list area
                if row := m.list.HitTest(relY); row >= 0 {
                    m.list.SetSelectedInstance(row)
                    return m, m.instanceChanged()
                }
            }
        }
    }
    return m, nil
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.updateHandleWindowSizeEvent(msg)
		return m, nil
	case error:
		// Handle errors from confirmation actions
		return m, m.handleError(msg)
	case instanceChangedMsg:
		// Handle instance changed after confirmation action
		return m, m.instanceChanged()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *home) handleQuit() (tea.Model, tea.Cmd) {
	if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
		return m, m.handleError(err)
	}
	return m, tea.Quit
}

func (m *home) handleMenuHighlighting(msg tea.KeyMsg) (cmd tea.Cmd, returnEarly bool) {
	// Handle menu highlighting when you press a button. We intercept it here and immediately return to
	// update the ui while re-sending the keypress. Then, on the next call to this, we actually handle the keypress.
	if m.keySent {
		m.keySent = false
		return nil, false
	}
	if m.state == statePrompt || m.state == stateHelp || m.state == stateConfirm {
		return nil, false
	}
	// If it's in the global keymap, we should try to highlight it.
	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return nil, false
	}

	if m.list.GetSelectedInstance() != nil && m.list.GetSelectedInstance().Paused() && name == keys.KeyEnter {
		return nil, false
	}
	if name == keys.KeyShiftDown || name == keys.KeyShiftUp {
		return nil, false
	}

	// Skip the menu highlighting if the key is not in the map or we are using the shift up and down keys.
	// TODO: cleanup: when you press enter on stateNew, we use keys.KeySubmitName. We should unify the keymap.
	if name == keys.KeyEnter && m.state == stateNew {
		name = keys.KeySubmitName
	}
	m.keySent = true
	return tea.Batch(
		func() tea.Msg { return msg },
		m.keydownCallback(name)), true
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (mod tea.Model, cmd tea.Cmd) {
    cmd, returnEarly := m.handleMenuHighlighting(msg)
    if returnEarly {
        return m, cmd
    }

	if m.state == stateHelp {
		return m.handleHelpState(msg)
	}

	if m.state == stateNew {
		// Handle quit commands first. Don't handle q because the user might want to type that.
		if msg.String() == "ctrl+c" {
			m.state = stateDefault
			m.promptAfterName = false
			m.list.Kill()
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		}

		instance := m.list.GetInstances()[m.list.NumInstances()-1]
		switch msg.Type {
		// Start the instance (enable previews etc) and go back to the main menu state.
		case tea.KeyEnter:
			if len(instance.Title) == 0 {
				return m, m.handleError(fmt.Errorf("title cannot be empty"))
			}

			if err := instance.Start(true); err != nil {
				m.list.Kill()
				m.state = stateDefault
				return m, m.handleError(err)
			}
			// Save after adding new instance
			if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
				return m, m.handleError(err)
			}
			// Instance added successfully, call the finalizer.
			m.newInstanceFinalizer()
			if m.autoYes {
				instance.AutoYes = true
			}

			m.newInstanceFinalizer()
			m.state = stateDefault
			if m.promptAfterName {
				m.state = statePrompt
				m.menu.SetState(ui.StatePrompt)
				// Initialize the text input overlay
				m.textInputOverlay = overlay.NewTextInputOverlay("Enter prompt", "")
				m.promptAfterName = false
			} else {
				m.menu.SetState(ui.StateDefault)
				m.showHelpScreen(helpStart(instance), nil)
			}

			return m, tea.Batch(tea.WindowSize(), m.instanceChanged())
		case tea.KeyRunes:
			if len(instance.Title) >= 32 {
				return m, m.handleError(fmt.Errorf("title cannot be longer than 32 characters"))
			}
			if err := instance.SetTitle(instance.Title + string(msg.Runes)); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeyBackspace:
			if len(instance.Title) == 0 {
				return m, nil
			}
			if err := instance.SetTitle(instance.Title[:len(instance.Title)-1]); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeySpace:
			if err := instance.SetTitle(instance.Title + " "); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeyEsc:
			m.list.Kill()
			m.state = stateDefault
			m.instanceChanged()

			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		default:
		}
		return m, nil
    } else if m.state == statePrompt {
		// Use the new TextInputOverlay component to handle all key events
		shouldClose := m.textInputOverlay.HandleKeyPress(msg)

		// Check if the form was submitted or canceled
		if shouldClose {
			selected := m.list.GetSelectedInstance()
			// TODO: this should never happen since we set the instance in the previous state.
			if selected == nil {
				return m, nil
			}
			if m.textInputOverlay.IsSubmitted() {
				if err := selected.SendPrompt(m.textInputOverlay.GetValue()); err != nil {
					// TODO: we probably end up in a bad state here.
					return m, m.handleError(err)
				}
			}

			// Close the overlay and reset state
			m.textInputOverlay = nil
			m.state = stateDefault
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					m.showHelpScreen(helpStart(selected), nil)
					return nil
				},
			)
		}

		return m, nil
    }

    // Handle confirmation state
    if m.state == stateConfirm {
        shouldClose := m.confirmationOverlay.HandleKeyPress(msg)
        if shouldClose {
            m.state = stateDefault
            m.confirmationOverlay = nil
            return m, nil
		}
		return m, nil
	}

	// Exit scrolling mode when ESC is pressed and preview pane is in scrolling mode
	// Check if Escape key was pressed and we're not in the diff tab (meaning we're in preview tab)
	// Always check for escape key first to ensure it doesn't get intercepted elsewhere
	if msg.Type == tea.KeyEsc {
		// If in preview tab and in scroll mode, exit scroll mode
		if !m.tabbedWindow.IsInDiffTab() && m.tabbedWindow.IsPreviewInScrollMode() {
			// Use the selected instance from the list
			selected := m.list.GetSelectedInstance()
			err := m.tabbedWindow.ResetPreviewToNormalMode(selected)
			if err != nil {
				return m, m.handleError(err)
			}
			return m, m.instanceChanged()
		}
	}

	// Handle quit commands first
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		return m.handleQuit()
	}

	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}

	switch name {
	case keys.KeyHelp:
		return m.showHelpScreen(helpTypeGeneral{}, nil)
	case keys.KeyPrompt:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m, m.handleError(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:        "",
			Path:         ".",
			Program:      m.program,
			DirectMode:   m.directMode,
			DirectBranch: m.directBranch,
		})
		if err != nil {
			return m, m.handleError(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)
		m.promptAfterName = true

		return m, nil
	case keys.KeyNew:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m, m.handleError(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:        "",
			Path:         ".",
			Program:      m.program,
			DirectMode:   m.directMode,
			DirectBranch: m.directBranch,
		})
		if err != nil {
			return m, m.handleError(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)

		return m, nil
	case keys.KeyUp:
		m.list.Up()
		return m, m.instanceChanged()
	case keys.KeyDown:
		m.list.Down()
		return m, m.instanceChanged()
	case keys.KeyShiftUp:
		m.tabbedWindow.ScrollUp()
		return m, m.instanceChanged()
	case keys.KeyShiftDown:
		m.tabbedWindow.ScrollDown()
		return m, m.instanceChanged()
	case keys.KeyPgUp:
		m.tabbedWindow.PageUp()
		return m, nil
	case keys.KeyPgDn:
		m.tabbedWindow.PageDown()
		return m, nil
	case keys.KeyHome, keys.KeyGoTop:
		m.tabbedWindow.GotoTop()
		return m, nil
	case keys.KeyEnd, keys.KeyGoBottom:
		m.tabbedWindow.GotoBottom()
		return m, nil
	case keys.KeyHalfUp:
		m.tabbedWindow.HalfPageUp()
		return m, nil
	case keys.KeyHalfDown:
		m.tabbedWindow.HalfPageDown()
		return m, nil
	case keys.KeyHunkNext:
		m.tabbedWindow.JumpNextHunk()
		return m, nil
	case keys.KeyHunkPrev:
		m.tabbedWindow.JumpPrevHunk()
		return m, nil
	case keys.KeyFileNext:
		m.tabbedWindow.JumpNextFile()
		return m, nil
	case keys.KeyFilePrev:
		m.tabbedWindow.JumpPrevFile()
		return m, nil
	case keys.KeyTab:
		m.tabbedWindow.Toggle()
		m.menu.SetInDiffTab(m.tabbedWindow.IsInDiffTab())
		return m, m.instanceChanged()
	case keys.KeyKill:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Create the kill action as a tea.Cmd
		killAction := func() tea.Msg {
			// Only check if branch is checked out for non-direct mode
			// In direct mode, we're working on the actual branch so this check doesn't apply
			if !selected.DirectMode {
				// Get worktree and check if branch is checked out
				worktree, err := selected.GetGitWorktree()
				if err != nil {
					return err
				}

				checkedOut, err := worktree.IsBranchCheckedOut()
				if err != nil {
					return err
				}

				if checkedOut {
					return fmt.Errorf("instance %s is currently checked out", selected.Title)
				}
			}

			// Delete from storage first
			if err := m.storage.DeleteInstance(selected.Title); err != nil {
				return err
			}

			// Then kill the instance
			m.list.Kill()
			return instanceChangedMsg{}
		}

		// Show confirmation modal
		message := fmt.Sprintf("[!] Kill session '%s'?", selected.Title)
		return m, m.confirmAction(message, killAction)
	case keys.KeySubmit:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Create the push action as a tea.Cmd
		pushAction := func() tea.Msg {
			// Default commit message with timestamp
			commitMsg := fmt.Sprintf("[claudesquad] update from '%s' on %s", selected.Title, time.Now().Format(time.RFC822))
			worktree, err := selected.GetGitWorktree()
			if err != nil {
				return err
			}
			if err = worktree.PushChanges(commitMsg, true); err != nil {
				return err
			}
			return nil
		}

		// Show confirmation modal
		message := fmt.Sprintf("[!] Push changes from session '%s'?", selected.Title)
		return m, m.confirmAction(message, pushAction)
	case keys.KeyCheckout:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Show help screen before pausing
		m.showHelpScreen(helpTypeInstanceCheckout{}, func() {
			if err := selected.Pause(); err != nil {
				m.handleError(err)
			}
			m.instanceChanged()
		})
		return m, nil
	case keys.KeyResume:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}
		if err := selected.Resume(); err != nil {
			return m, m.handleError(err)
		}
		return m, tea.WindowSize()
	case keys.KeyEnter:
		if m.list.NumInstances() == 0 {
			return m, nil
		}
		selected := m.list.GetSelectedInstance()
		if selected == nil || selected.Paused() || !selected.TmuxAlive() {
			return m, nil
		}
		// Show help screen before attaching
		m.showHelpScreen(helpTypeInstanceAttach{}, func() {
			ch, err := m.list.Attach()
			if err != nil {
				m.handleError(err)
				return
			}
			<-ch
			m.state = stateDefault
		})
		return m, nil
    // Global fall-through handling (default state)
    default:
        // Esc exits preview scroll mode if active
        if msg.String() == "esc" && m.tabbedWindow.IsPreviewInScrollMode() {
            _ = m.tabbedWindow.ResetPreviewToNormalMode(m.list.GetSelectedInstance())
            return m, nil
        }
        // Number key instance selection (1..9, 0 = 10)
        switch msg.String() {
        case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
            idx := 0
            if msg.String() == "0" {
                idx = 9
            } else {
                idx = int(msg.Runes[0]-'1')
            }
            if idx >= 0 && idx < m.list.NumInstances() {
                m.list.SetSelectedInstance(idx)
                return m, m.instanceChanged()
            }
        }
        return m, nil
    }
}

// instanceChanged updates the preview pane, menu, and diff pane based on the selected instance. It returns an error
// Cmd if there was any error.
func (m *home) instanceChanged() tea.Cmd {
	// selected may be nil
	selected := m.list.GetSelectedInstance()

	m.tabbedWindow.UpdateDiff(selected)
	m.tabbedWindow.SetInstance(selected)
	// Update menu with current instance
	m.menu.SetInstance(selected)

	// If there's no selected instance, we don't need to update the preview.
	if err := m.tabbedWindow.UpdatePreview(selected); err != nil {
		return m.handleError(err)
	}
	// Manage diff watcher lifecycle
	if m.tabbedWindow.IsInDiffTab() && selected != nil {
		// Start watcher if not active or instance changed; also prime a diff
		prime := []tea.Cmd{makeGitDiffCmd(selected)}
		if !m.diffWatchActive || m.diffWatchInst != selected {
			m.diffWatchInst = selected
			m.diffWatchActive = true
			m.diffWatchLastDirty = false
			prime = append(prime, m.diffWatchPollCmd())
		}
		return tea.Batch(prime...)
	}
	// Stop watcher when diff tab hidden or no selection
	m.diffWatchActive = false
	m.diffWatchInst = nil
	return nil
}

type keyupMsg struct{}

// keydownCallback clears the menu option highlighting after 500ms.
func (m *home) keydownCallback(name keys.KeyName) tea.Cmd {
	m.menu.Keydown(name)
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(500 * time.Millisecond):
		}

		return keyupMsg{}
	}
}

// hideErrMsg implements tea.Msg and clears the error text from the screen.
type hideErrMsg struct{}

// previewTickMsg implements tea.Msg and triggers a preview update
type previewTickMsg struct{}

type tickUpdateMetadataMessage struct{}

type instanceChangedMsg struct{}

// diffWatchTickedMsg is sent when the diff watcher detects a change
type diffWatchTickedMsg struct{ changed bool }

// Async command results
type tmuxStatusMsg struct {
	instance *session.Instance
	updated  bool
	prompt   bool
	err      error
}

type gitDiffMsg struct {
	instance *session.Instance
	stats    *git.DiffStats
	err      error
}

// tickUpdateMetadataCmd is the callback to update the metadata of the instances every 500ms. Note that we iterate
// overall the instances and capture their output. It's a pretty expensive operation. Let's do it 2x a second only.
var tickUpdateMetadataCmd = func() tea.Msg {
	time.Sleep(500 * time.Millisecond)
	return tickUpdateMetadataMessage{}
}

// --- Async command helpers ---

// makeTmuxStatusCmd captures status in background with a timeout
func makeTmuxStatusCmd(inst *session.Instance) tea.Cmd {
	return func() tea.Msg {
		// Timeout to avoid blocking UI
		timeout := time.After(400 * time.Millisecond)
		done := make(chan struct{})
		var updated bool
		var prompt bool
		var err error
		go func() {
			updated, prompt = inst.HasUpdated()
			close(done)
		}()
		select {
		case <-timeout:
			err = fmt.Errorf("tmux status timeout")
		case <-done:
		}
		return tmuxStatusMsg{instance: inst, updated: updated, prompt: prompt, err: err}
	}
}

// makeGitDiffCmd computes diff stats in background with a timeout
func makeGitDiffCmd(inst *session.Instance) tea.Cmd {
	return func() tea.Msg {
		timeout := time.After(1500 * time.Millisecond)
		done := make(chan struct{})
		var stats *git.DiffStats
		var err error
		go func() {
			// Use full diff path
			if err2 := inst.UpdateDiffStats(); err2 != nil {
				err = err2
			}
			stats = inst.GetDiffStats()
			close(done)
		}()
		select {
		case <-timeout:
			err = fmt.Errorf("git diff timeout")
		case <-done:
		}
		return gitDiffMsg{instance: inst, stats: stats, err: err}
	}
}

// diffWatchPollCmd polls the worktree for changes on a short interval and emits a tick message.
// It leverages git status as a cross-platform fallback for file change events.
func (m *home) diffWatchPollCmd() tea.Cmd {
	return func() tea.Msg {
		// Polling interval with basic debounce effect
		time.Sleep(300 * time.Millisecond)
		// Conditions may have changed
		if !m.diffWatchActive || m.diffWatchInst == nil || !m.tabbedWindow.IsInDiffTab() {
			return nil
		}
		// Query worktree dirty status
		gw, err := m.diffWatchInst.GetGitWorktree()
		if err != nil {
			log.WarningLog.Printf("diff watcher get worktree error: %v", err)
			return diffWatchTickedMsg{changed: false}
		}
		dirty, err := gw.IsDirty()
		if err != nil {
			log.WarningLog.Printf("diff watcher IsDirty error: %v", err)
			return diffWatchTickedMsg{changed: false}
		}
		changed := dirty != m.diffWatchLastDirty
		m.diffWatchLastDirty = dirty
		return diffWatchTickedMsg{changed: changed}
	}
}

// handleError handles all errors which get bubbled up to the app. sets the error message. We return a callback tea.Cmd that returns a hideErrMsg message
// which clears the error message after 3 seconds.
func (m *home) handleError(err error) tea.Cmd {
	log.ErrorLog.Printf("%v", err)
	m.errBox.SetError(err)
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(3 * time.Second):
		}

		return hideErrMsg{}
	}
}

// confirmAction shows a confirmation modal and stores the action to execute on confirm
func (m *home) confirmAction(message string, action tea.Cmd) tea.Cmd {
	m.state = stateConfirm

	// Create and show the confirmation overlay using ConfirmationOverlay
	m.confirmationOverlay = overlay.NewConfirmationOverlay(message)
	// Set a fixed width for consistent appearance
	m.confirmationOverlay.SetWidth(50)

	// Set callbacks for confirmation and cancellation
	m.confirmationOverlay.OnConfirm = func() {
		m.state = stateDefault
		// Execute the action if it exists
		if action != nil {
			_ = action()
		}
	}

	m.confirmationOverlay.OnCancel = func() {
		m.state = stateDefault
	}

	return nil
}

func (m *home) View() string {
    listWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.list.String())
    previewWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.tabbedWindow.String())
    var content string
    if m.stacked {
        content = lipgloss.JoinVertical(lipgloss.Left, listWithPadding, previewWithPadding)
    } else {
        content = lipgloss.JoinHorizontal(lipgloss.Top, listWithPadding, previewWithPadding)
    }

    mainView := lipgloss.JoinVertical(
        lipgloss.Left,
        content,
        m.menu.String(),
        m.errBox.String(),
    )

	if m.state == statePrompt {
		if m.textInputOverlay == nil {
			log.ErrorLog.Printf("text input overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.textInputOverlay.Render(), mainView, true, true)
	} else if m.state == stateHelp {
		if m.textOverlay == nil {
			log.ErrorLog.Printf("text overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.textOverlay.Render(), mainView, true, true)
	} else if m.state == stateConfirm {
		if m.confirmationOverlay == nil {
			log.ErrorLog.Printf("confirmation overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.confirmationOverlay.Render(), mainView, true, true)
	}

	return mainView
}
