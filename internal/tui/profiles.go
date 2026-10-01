package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
)

// customID is the profile entry that preselects nothing.
const customID = ""

func (m Model) profileEntries() []catalog.Profile {
	return append(append([]catalog.Profile(nil), m.cfg.Catalog.Profiles...), catalog.Profile{ID: customID, Name: "Custom"})
}

func (m Model) updateProfiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.profileEntries()
	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit
	case key.Matches(msg, keyBack):
		m.screen = screenWelcome
	case key.Matches(msg, keyUp):
		m.profileCursor = max(m.profileCursor-1, 0)
	case key.Matches(msg, keyDown):
		m.profileCursor = min(m.profileCursor+1, len(entries)-1)
	case key.Matches(msg, keyEnter):
		m.profile = entries[m.profileCursor].ID
		m.sel = NewSelection(m.cfg.Catalog, m.installed, m.profile)
		m.list = newChecklist()
		m.refreshPlan()
		m.screen = screenChecklist
	}
	return m, nil
}

func (m Model) viewProfiles() string {
	var b strings.Builder
	b.WriteString("Start from a profile. You can add or remove tools on the next screen.\n\n")
	for i, p := range m.profileEntries() {
		cursor, name := "  ", p.Name
		if i == m.profileCursor {
			cursor, name = titleStyle.Render("> "), boldStyle.Render(p.Name)
		}
		detail := "start with nothing selected"
		if p.ID != customID {
			detail = itoa(len(m.cfg.Catalog.ForProfile(p.ID))) + " tools"
		}
		b.WriteString(cursor + name + dimStyle.Render("  "+detail) + "\n")
	}
	return b.String()
}
