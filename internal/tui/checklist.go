package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
)

type checklist struct {
	cursor    int
	filter    textinput.Model
	filtering bool
}

func newChecklist() checklist {
	in := textinput.New()
	in.Prompt = "/ "
	in.Placeholder = "filter by name"
	return checklist{filter: in}
}

// refreshPlan rebuilds the plan after the selection changed.
func (m *Model) refreshPlan() {
	p, err := m.sel.Plan()
	if err != nil {
		m.notice = err.Error()
		return
	}
	m.plan, m.required = p, Required(p)
}

// visible returns the tools that match the filter, grouped by category in
// catalog order.
func (m Model) visible() []catalog.Tool {
	q := strings.ToLower(strings.TrimSpace(m.list.filter.Value()))
	var out []catalog.Tool
	for _, cat := range m.cfg.Catalog.Categories {
		for _, t := range m.cfg.Catalog.Tools {
			if t.Category != cat.ID {
				continue
			}
			hay := strings.ToLower(t.ID + " " + t.Name + " " + t.Description)
			if q == "" || strings.Contains(hay, q) {
				out = append(out, t)
			}
		}
	}
	return out
}

func (m *Model) clampCursor() {
	m.list.cursor = max(0, min(m.list.cursor, len(m.visible())-1))
}

func (m Model) updateChecklist(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tools := m.visible()
	switch {
	case m.list.filtering && key.Matches(msg, keyBack):
		m.list.filtering = false
		m.list.filter.SetValue("")
		m.list.filter.Blur()
		m.clampCursor()
		return m, nil
	case m.list.filtering && key.Matches(msg, keyEnter):
		m.list.filtering = false
		m.list.filter.Blur()
		return m, nil
	case key.Matches(msg, keyUp):
		m.list.cursor = max(m.list.cursor-1, 0)
		return m, nil
	case key.Matches(msg, keyDown):
		m.list.cursor = min(m.list.cursor+1, max(len(tools)-1, 0))
		return m, nil
	case m.list.filtering:
		var cmd tea.Cmd
		m.list.filter, cmd = m.list.filter.Update(msg)
		m.clampCursor()
		return m, cmd
	}

	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit
	case key.Matches(msg, keyFilter):
		m.list.filtering = true
		return m, m.list.filter.Focus()
	case key.Matches(msg, keyBack):
		if m.list.filter.Value() != "" {
			m.list.filter.SetValue("")
			m.clampCursor()
		} else {
			m.screen = screenProfiles
		}
	case key.Matches(msg, keyToggle):
		if len(tools) == 0 {
			return m, nil
		}
		t := tools[m.list.cursor]
		switch reason, req := m.required[t.ID]; {
		case m.installed[t.ID]:
			m.notice = t.Name + " is already installed."
		case req && !m.sel.Has(t.ID):
			m.notice = t.Name + " is " + reason + " and cannot be removed."
		default:
			m.sel.Toggle(t.ID)
			m.refreshPlan()
		}
	case key.Matches(msg, keyAll):
		ids := make([]string, 0, len(tools))
		all := true
		for _, t := range tools {
			if m.installed[t.ID] {
				continue
			}
			ids = append(ids, t.ID)
			all = all && m.sel.Has(t.ID)
		}
		m.sel.Set(ids, !all)
		m.refreshPlan()
	case key.Matches(msg, keyEnter):
		if m.plan == nil || len(m.plan.Pending()) == 0 {
			m.notice = "Nothing to install. Select at least one tool."
			return m, nil
		}
		m.enterReview()
	}
	return m, nil
}

func (m Model) viewChecklist() string {
	tools := m.visible()
	var lines []string
	focus := 0
	cat := ""
	names := map[string]string{}
	for _, c := range m.cfg.Catalog.Categories {
		names[c.ID] = c.Name
	}
	width := m.w
	for i, t := range tools {
		if t.Category != cat {
			cat = t.Category
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, boldStyle.Render(names[cat]))
		}
		if i == m.list.cursor {
			focus = len(lines)
		}
		lines = append(lines, clip(m.toolLine(t, i == m.list.cursor), width))
	}
	if len(tools) == 0 {
		lines = append(lines, dimStyle.Render("No tools match the filter."))
	}

	pending, present := 0, 0
	if m.plan != nil {
		pending = len(m.plan.Pending())
		present = len(m.plan.Items) - pending
	}
	status := fmt.Sprintf("%d to install, %d already installed", pending, present)
	top := dimStyle.Render(status)
	if m.list.filtering || m.list.filter.Value() != "" {
		top = m.list.filter.View() + "  " + top
	}
	return top + "\n\n" + strings.Join(window(lines, focus, m.bodyHeight()-2), "\n")
}

func (m Model) toolLine(t catalog.Tool, cursor bool) string {
	prefix := "  "
	if cursor {
		prefix = titleStyle.Render("> ")
	}
	reason, req := m.required[t.ID]
	switch {
	case m.installed[t.ID]:
		return prefix + dimStyle.Render("[✓] "+t.Name+"  installed")
	case m.sel.Has(t.ID):
		return prefix + okStyle.Render("[x]") + " " + t.Name + dimStyle.Render("  "+t.Description)
	case req:
		return prefix + warnStyle.Render("[+]") + " " + t.Name + dimStyle.Render("  "+reason)
	}
	return prefix + "[ ] " + t.Name + dimStyle.Render("  "+t.Description)
}
