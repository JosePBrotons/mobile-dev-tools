package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reviewState struct {
	vp viewport.Model
}

// enterReview renders the plan into the viewport and switches screens.
func (m *Model) enterReview() {
	m.review.vp = viewport.New(m.w, m.bodyHeight())
	m.review.vp.SetContent(m.reviewContent())
	m.screen = screenReview
}

// warnings returns the pending requirements the user should know about
// before starting. They come from the tools' notes.
func (m Model) warnings() []string {
	var out []string
	notes := strings.Join(m.plan.Notes(), "\n")
	if strings.Contains(notes, "admin password") {
		out = append(out, "You will be asked for your admin password (sudo).")
	}
	if strings.Contains(notes, "App Store") {
		out = append(out, "Sign in to the App Store with your Apple ID before continuing.")
	}
	return out
}

func (m Model) reviewContent() string {
	var b strings.Builder
	pending := len(m.plan.Pending())
	b.WriteString(boldStyle.Render(itoa(pending)+" to install, "+itoa(len(m.plan.Items)-pending)+" already installed") + "\n\n")
	for _, w := range m.warnings() {
		b.WriteString(warnStyle.Render("! "+w) + "\n")
	}
	if len(m.warnings()) > 0 {
		b.WriteString("\n")
	}
	wrap := lipgloss.NewStyle().Width(max(m.w-6, 20))
	for _, r := range m.plan.Rows() {
		status := dimStyle.Render("installed")
		if r.Status == "install" {
			status = okStyle.Render("install  ")
		}
		line := status + " " + boldStyle.Render(r.Name)
		if r.Reason != "" {
			line += dimStyle.Render("  (" + r.Reason + ")")
		}
		b.WriteString(line + "\n")
		for _, c := range r.Commands {
			for i, l := range strings.Split(wrap.Render(c), "\n") {
				lead := "    $ "
				if i > 0 {
					lead = "      "
				}
				b.WriteString(dimStyle.Render(lead+strings.TrimRight(l, " ")) + "\n")
			}
		}
	}
	if notes := m.plan.Notes(); len(notes) > 0 {
		b.WriteString("\n" + boldStyle.Render("Notes") + "\n")
		for _, n := range notes {
			b.WriteString(wrap.Render("  "+n) + "\n")
		}
	}
	return b.String()
}

func (m Model) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit
	case key.Matches(msg, keyBack):
		m.screen = screenChecklist
	case key.Matches(msg, keyEnter):
		if !m.installsAllowed() {
			m.notice = "Installs run on macOS only. This is a preview."
			return m, nil
		}
		return m.startInstall()
	default:
		var cmd tea.Cmd
		m.review.vp, cmd = m.review.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) viewReview() string {
	return m.review.vp.View()
}
