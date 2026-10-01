package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
)

type summaryState struct {
	report plan.Report
	steps  []string
}

func (m *Model) finish(r plan.Report) {
	m.sum = summaryState{report: r, steps: plan.NextSteps(m.plan, r)}
	m.screen = screenSummary
	m.notice = ""
}

func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keyDone) {
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewSummary() string {
	groups := []struct {
		status plan.Status
		style  func(...string) string
	}{
		{plan.StatusInstalled, okStyle.Render},
		{plan.StatusPresent, dimStyle.Render},
		{plan.StatusSkipped, warnStyle.Render},
		{plan.StatusFailed, errStyle.Render},
	}
	var b strings.Builder
	for _, g := range groups {
		var lines []string
		for _, r := range m.sum.report.Results {
			if r.Status != g.status {
				continue
			}
			line := "  " + r.Name
			if r.Err != nil {
				line += dimStyle.Render("  " + r.Err.Error())
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			b.WriteString(g.style(boldStyle.Render(string(g.status))+" ("+itoa(len(lines))+")") + "\n")
			b.WriteString(strings.Join(lines, "\n") + "\n\n")
		}
	}
	if len(m.sum.steps) > 0 {
		b.WriteString(boldStyle.Render("Next steps") + "\n")
		for _, s := range m.sum.steps {
			b.WriteString("  • " + s + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("Log: " + m.prog.logPath))
	return b.String()
}
