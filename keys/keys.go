package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyName int

const (
    KeyUp KeyName = iota
    KeyDown
    KeyEnter
    KeyNew
    KeyKill
    KeyQuit
    KeyReview
    KeyPush
    KeySubmit

    KeyTab        // Tab is a special keybinding for switching between panes.
    KeySubmitName // SubmitName is a special keybinding for submitting the name of a new instance.

    KeyCheckout
    KeyResume
    KeyPrompt // New key for entering a prompt
    KeyHelp   // Key for showing help screen

    // Diff keybindings
    KeyShiftUp
    KeyShiftDown

    // Scroll/navigation keys
    KeyPgUp
    KeyPgDn
    KeyHome
    KeyEnd
    KeyGoTop
    KeyGoBottom
    KeyHalfUp
    KeyHalfDown

    // Diff navigation
    KeyFilePrev
    KeyFileNext
    KeyHunkPrev
    KeyHunkNext

    // Number selection
    KeyNum1
    KeyNum2
    KeyNum3
    KeyNum4
    KeyNum5
    KeyNum6
    KeyNum7
    KeyNum8
    KeyNum9
    KeyNum0
)

// GlobalKeyStringsMap is a global, immutable map string to keybinding.
var GlobalKeyStringsMap = map[string]KeyName{
    "up":         KeyUp,
    "k":          KeyUp,
    "down":       KeyDown,
    "j":          KeyDown,
    "shift+up":   KeyShiftUp,
    "shift+down": KeyShiftDown,
    "N":          KeyPrompt,
    "enter":      KeyEnter,
    "o":          KeyEnter,
    "n":          KeyNew,
    "D":          KeyKill,
    "q":          KeyQuit,
    "tab":        KeyTab,
    "c":          KeyCheckout,
    "r":          KeyResume,
    "p":          KeySubmit,
    "?":          KeyHelp,

    // Scroll/navigation
    "pgup":       KeyPgUp,
    "pgdown":     KeyPgDn,
    "home":       KeyHome,
    "end":        KeyEnd,
    "g":          KeyGoTop,
    "G":          KeyGoBottom,
    "ctrl+u":     KeyHalfUp,
    "ctrl+d":     KeyHalfDown,

    // Diff nav
    "[":          KeyHunkPrev,
    "]":          KeyHunkNext,
    "{":          KeyFilePrev,
    "}":          KeyFileNext,

    // Number keys
    "1":          KeyNum1,
    "2":          KeyNum2,
    "3":          KeyNum3,
    "4":          KeyNum4,
    "5":          KeyNum5,
    "6":          KeyNum6,
    "7":          KeyNum7,
    "8":          KeyNum8,
    "9":          KeyNum9,
    "0":          KeyNum0,
}

// GlobalkeyBindings is a global, immutable map of KeyName tot keybinding.
var GlobalkeyBindings = map[KeyName]key.Binding{
	KeyUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	KeyDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	KeyShiftUp: key.NewBinding(
		key.WithKeys("shift+up"),
		key.WithHelp("shift+↑", "scroll"),
	),
	KeyShiftDown: key.NewBinding(
		key.WithKeys("shift+down"),
		key.WithHelp("shift+↓", "scroll"),
	),
	KeyEnter: key.NewBinding(
		key.WithKeys("enter", "o"),
		key.WithHelp("↵/o", "open"),
	),
	KeyNew: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	KeyKill: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "kill"),
	),
	KeyHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	KeyQuit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	KeySubmit: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "push branch"),
	),
	KeyPrompt: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "new with prompt"),
	),
	KeyCheckout: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "checkout"),
	),
	KeyTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch tab"),
	),
    KeyResume: key.NewBinding(
        key.WithKeys("r"),
        key.WithHelp("r", "resume"),
    ),

    // -- Special keybindings --

    KeySubmitName: key.NewBinding(
        key.WithKeys("enter"),
        key.WithHelp("enter", "submit name"),
    ),

    // --- Scroll/navigation ---
    KeyPgUp: key.NewBinding(
        key.WithKeys("pgup"),
        key.WithHelp("PgUp", "page up"),
    ),
    KeyPgDn: key.NewBinding(
        key.WithKeys("pgdown"),
        key.WithHelp("PgDn", "page down"),
    ),
    KeyHome: key.NewBinding(
        key.WithKeys("home"),
        key.WithHelp("Home", "go to start"),
    ),
    KeyEnd: key.NewBinding(
        key.WithKeys("end"),
        key.WithHelp("End", "go to end"),
    ),
    KeyGoTop: key.NewBinding(
        key.WithKeys("g"),
        key.WithHelp("g", "top"),
    ),
    KeyGoBottom: key.NewBinding(
        key.WithKeys("G"),
        key.WithHelp("G", "bottom"),
    ),
    KeyHalfUp: key.NewBinding(
        key.WithKeys("ctrl+u"),
        key.WithHelp("C-u", "half up"),
    ),
    KeyHalfDown: key.NewBinding(
        key.WithKeys("ctrl+d"),
        key.WithHelp("C-d", "half down"),
    ),
    // --- Diff navigation ---
    KeyFilePrev: key.NewBinding(
        key.WithKeys("{"),
        key.WithHelp("{", "prev file"),
    ),
    KeyFileNext: key.NewBinding(
        key.WithKeys("}"),
        key.WithHelp("}", "next file"),
    ),
    KeyHunkPrev: key.NewBinding(
        key.WithKeys("["),
        key.WithHelp("[", "prev hunk"),
    ),
    KeyHunkNext: key.NewBinding(
        key.WithKeys("]"),
        key.WithHelp("]", "next hunk"),
    ),
}
