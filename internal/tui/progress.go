package tui

import (
	"context"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

const (
	maxLogLines   = 5000
	sudoKeepAlive = time.Minute
)

type (
	eventMsg plan.Event
	logMsg   string
	doneMsg  struct{ report plan.Report }
	// sudoMsg reports the result of caching the admin password.
	sudoMsg struct{ err error }
)

type progressState struct {
	status      map[string]plan.Status
	errs        map[string]error
	running     string // name of the step in progress
	lines       []string
	partial     string
	vp          viewport.Model
	ch          chan tea.Msg
	cancel      context.CancelFunc
	confirmQuit bool
	cancelled   bool
	logPath     string
}

func (p *progressState) resize(w, h int) {
	p.vp.Width = max(w-listWidth(w)-2, 10)
	p.vp.Height = max(h-2, 3)
	p.refresh()
}

func listWidth(w int) int { return min(34, w/3) }

// startInstall caches the admin password, then runs the plan.
func (m Model) startInstall() (tea.Model, tea.Cmd) {
	if m.cfg.SudoCmd == nil {
		return m.runInstall()
	}
	return m, tea.ExecProcess(m.cfg.SudoCmd(), func(err error) tea.Msg { return sudoMsg{err} })
}

func (m Model) runInstall() (tea.Model, tea.Cmd) {
	logPath, logFile, err := m.cfg.OpenLog()
	if err != nil {
		m.notice = "Cannot open the log: " + err.Error()
		return m, nil
	}
	ctx, cancel := context.WithCancel(m.ctx)
	ch := make(chan tea.Msg, 256)
	send := func(msg tea.Msg) {
		select {
		case ch <- msg:
		case <-m.ctx.Done():
		}
	}
	m.prog = progressState{
		status:  map[string]plan.Status{},
		errs:    map[string]error{},
		ch:      ch,
		cancel:  cancel,
		logPath: logPath,
	}
	m.prog.vp = viewport.New(1, 1)
	m.resize()
	m.screen = screenProgress
	m.showHelp = false

	p, r, opts := m.plan, m.cfg.Runner, plan.Options{Home: m.cfg.Home, Log: io.MultiWriter(chanWriter(send), logFile)}
	keepSudo := m.cfg.SudoCmd != nil
	go func() {
		stop := make(chan struct{})
		if keepSudo {
			go keepAlive(ctx, r, stop)
		}
		report := plan.Execute(ctx, p, r, opts, func(e plan.Event) { send(eventMsg(e)) })
		close(stop)
		_ = logFile.Close()
		send(doneMsg{report})
	}()
	return m, tea.Batch(m.spin.Tick, waitMsg(ch))
}

// keepAlive refreshes the cached sudo timestamp so a long install does not
// ask for the password again halfway through.
func keepAlive(ctx context.Context, r runner.Runner, stop <-chan struct{}) {
	t := time.NewTicker(sudoKeepAlive)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			_ = r.Run(ctx, "sudo -n -v", io.Discard)
		}
	}
}

type chanWriter func(tea.Msg)

func (w chanWriter) Write(p []byte) (int, error) {
	w(logMsg(string(p)))
	return len(p), nil
}

// waitMsg blocks until the install goroutine sends its next message.
func waitMsg(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sudoMsg:
		if msg.err != nil {
			m.notice = "Could not get admin access: " + msg.err.Error()
			return m, nil
		}
		return m.runInstall()
	case eventMsg:
		res := msg.Result
		if msg.Kind == plan.Started {
			m.prog.running = res.Name
		} else if res.ID != "brew-update" {
			m.prog.status[res.ID] = res.Status
			m.prog.errs[res.ID] = res.Err
		}
		return m, waitMsg(m.prog.ch)
	case logMsg:
		m.prog.appendLog(string(msg))
		return m, waitMsg(m.prog.ch)
	case doneMsg:
		m.prog.cancel()
		m.finish(msg.report)
		return m, nil
	case tea.KeyMsg:
		return m.progressKey(msg)
	}
	return m, nil
}

func (m Model) progressKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.prog.confirmQuit {
		switch {
		case key.Matches(msg, keyYes):
			m.prog.confirmQuit, m.prog.cancelled = false, true
			m.prog.cancel()
			m.notice = "Cancelling..."
		case key.Matches(msg, keyNo):
			m.prog.confirmQuit = false
		}
		return m, nil
	}
	if key.Matches(msg, keyCancel) {
		m.prog.confirmQuit = true
		return m, nil
	}
	var cmd tea.Cmd
	m.prog.vp, cmd = m.prog.vp.Update(msg)
	return m, cmd
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// appendLog adds command output, dropping color codes and treating a
// carriage return as a line break so progress bars stay readable.
func (p *progressState) appendLog(s string) {
	s = ansiRE.ReplaceAllString(s, "")
	s = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(s)
	parts := strings.Split(p.partial+s, "\n")
	p.partial = parts[len(parts)-1]
	p.lines = append(p.lines, parts[:len(parts)-1]...)
	if extra := len(p.lines) - maxLogLines; extra > 0 {
		p.lines = p.lines[extra:]
	}
	p.refresh()
}

func (p *progressState) refresh() {
	follow := p.vp.AtBottom()
	lines := p.lines
	if p.partial != "" {
		lines = append(append([]string(nil), lines...), p.partial)
	}
	for i, l := range lines {
		lines[i] = clip(l, p.vp.Width)
	}
	p.vp.SetContent(strings.Join(lines, "\n"))
	if follow {
		p.vp.GotoBottom()
	}
}

func (m Model) viewProgress() string {
	var steps []string
	current := 0
	for i, it := range m.plan.Items {
		name := it.Tool.Name
		var mark string
		switch st, done := m.prog.status[it.Tool.ID]; {
		case done && st == plan.StatusFailed:
			mark = errStyle.Render("✗")
		case done && st == plan.StatusSkipped:
			mark = dimStyle.Render("–")
		case done:
			mark = okStyle.Render("✓")
		case name == m.prog.running:
			mark, current = m.spin.View(), i
		default:
			mark = dimStyle.Render("·")
		}
		steps = append(steps, mark+" "+name)
	}
	lw := listWidth(m.w)
	left := make([]string, 0, len(steps))
	for _, l := range window(steps, current, m.bodyHeight()-2) {
		left = append(left, clip(l, lw))
	}
	leftCol := lipglossWidth(lw).Render(strings.Join(left, "\n"))
	head := m.spin.View() + " " + orDefault(m.prog.running, "Starting...")
	if m.prog.confirmQuit {
		head = warnStyle.Render("Cancel the installation? y/n")
	}
	right := head + "\n" + dimStyle.Render("log: "+m.prog.logPath) + "\n" + m.prog.vp.View()
	return joinColumns(leftCol, right)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
