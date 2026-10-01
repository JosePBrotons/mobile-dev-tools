package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

var (
	keyUp     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	keyDown   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	keyToggle = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle"))
	keyAll    = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all"))
	keyFilter = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	keyEnter  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue"))
	keyBack   = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
	keyQuit   = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	keyHelp   = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help"))
	keyScroll = key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "scroll"))
	keyYes    = key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))
	keyNo     = key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n", "no"))
	keyCancel = key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel install"))
	keyDone   = key.NewBinding(key.WithKeys("enter", "q", "ctrl+c"), key.WithHelp("enter", "exit"))
)

// bindings adapts a pair of key lists to help.KeyMap.
type bindings struct{ short, full []key.Binding }

func (b bindings) ShortHelp() []key.Binding  { return b.short }
func (b bindings) FullHelp() [][]key.Binding { return [][]key.Binding{b.full} }

func helpFor(m Model) help.KeyMap {
	switch m.screen {
	case screenWelcome:
		return bindings{[]key.Binding{keyEnter, keyQuit}, []key.Binding{keyEnter, keyQuit}}
	case screenProfiles:
		return bindings{
			[]key.Binding{keyUp, keyDown, keyEnter, keyBack, keyHelp},
			[]key.Binding{keyUp, keyDown, keyEnter, keyBack, keyQuit},
		}
	case screenChecklist:
		if m.list.filtering {
			return bindings{[]key.Binding{keyEnter, keyBack}, []key.Binding{keyUp, keyDown, keyEnter, keyBack}}
		}
		return bindings{
			[]key.Binding{keyToggle, keyAll, keyFilter, keyEnter, keyBack, keyHelp},
			[]key.Binding{keyUp, keyDown, keyToggle, keyAll, keyFilter, keyEnter, keyBack, keyQuit},
		}
	case screenReview:
		return bindings{
			[]key.Binding{keyScroll, keyEnter, keyBack, keyHelp},
			[]key.Binding{keyUp, keyDown, keyScroll, keyEnter, keyBack, keyQuit},
		}
	case screenProgress:
		if m.prog.confirmQuit {
			return bindings{[]key.Binding{keyYes, keyNo}, []key.Binding{keyYes, keyNo}}
		}
		return bindings{[]key.Binding{keyScroll, keyCancel}, []key.Binding{keyUp, keyDown, keyScroll, keyCancel}}
	default:
		return bindings{[]key.Binding{keyDone}, []key.Binding{keyDone}}
	}
}
