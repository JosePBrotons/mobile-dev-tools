// Package tui is the Bubble Tea interface of mdt. It holds no install logic:
// it calls internal/plan, internal/detect and internal/runner.
package tui

import (
	"context"
	"io"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/detect"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/sysinfo"
)

type screen int

const (
	screenWelcome screen = iota
	screenProfiles
	screenChecklist
	screenReview
	screenProgress
	screenSummary
)

// Config wires the TUI to the rest of mdt.
type Config struct {
	Catalog *catalog.Catalog
	Runner  runner.Runner
	Home    string
	GOOS    string
	// OpenLog creates the install log and returns its path.
	OpenLog func() (path string, w io.WriteCloser, err error)
	// SudoCmd returns the command that caches the admin password before an
	// install. Nil skips the step.
	SudoCmd func() *exec.Cmd
}

// Model is the root Bubble Tea model.
type Model struct {
	ctx context.Context
	cfg Config

	screen   screen
	w, h     int
	spin     spinner.Model
	help     help.Model
	showHelp bool
	notice   string // one line message shown above the footer

	checking  bool
	info      sysinfo.Info
	installed map[string]bool

	profile       string // profile id, empty for Custom
	profileCursor int

	sel      *Selection
	plan     *plan.Plan
	required map[string]string
	list     checklist

	review reviewState
	prog   progressState
	sum    summaryState
}

// New returns the model for the first screen. ctx cancels background work.
func New(ctx context.Context, cfg Config) Model {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = titleStyle
	return Model{
		ctx:      ctx,
		cfg:      cfg,
		spin:     sp,
		help:     help.New(),
		checking: true,
		list:     newChecklist(),
		w:        80,
		h:        24,
	}
}

// Failed reports whether the install ended with failed tools.
func (m Model) Failed() bool { return m.sum.report.Failed() }

type checkedMsg struct {
	info      sysinfo.Info
	installed map[string]bool
}

// Init starts the system check.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		return checkedMsg{
			info:      sysinfo.Check(m.ctx, m.cfg.Runner),
			installed: detect.Installed(m.ctx, m.cfg.Runner, m.cfg.Catalog.Tools),
		}
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.resize()
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case checkedMsg:
		m.checking = false
		m.info, m.installed = msg.info, msg.installed
		return m, nil
	case eventMsg, logMsg, doneMsg, sudoMsg:
		return m.updateProgress(msg)
	case tea.KeyMsg:
		m.notice = ""
		if msg.String() == "ctrl+c" && m.screen != screenProgress {
			return m, tea.Quit
		}
		if key.Matches(msg, keyHelp) && !m.list.filtering && m.screen != screenWelcome {
			m.showHelp = !m.showHelp
			return m, nil
		}
		switch m.screen {
		case screenWelcome:
			return m.updateWelcome(msg)
		case screenProfiles:
			return m.updateProfiles(msg)
		case screenChecklist:
			return m.updateChecklist(msg)
		case screenReview:
			return m.updateReview(msg)
		case screenProgress:
			return m.updateProgress(msg)
		case screenSummary:
			return m.updateSummary(msg)
		}
	}
	return m, nil
}

// bodyHeight is the space left for a screen between the header and footer.
func (m Model) bodyHeight() int {
	return max(m.h-5, 3)
}

func (m *Model) resize() {
	m.review.vp.Width, m.review.vp.Height = m.w, m.bodyHeight()
	m.prog.resize(m.w, m.bodyHeight())
}

var stepNames = map[screen]string{
	screenWelcome:   "System check",
	screenProfiles:  "Profile",
	screenChecklist: "Tools",
	screenReview:    "Review",
	screenProgress:  "Installing",
	screenSummary:   "Summary",
}

// View implements tea.Model.
func (m Model) View() string {
	var body string
	switch m.screen {
	case screenWelcome:
		body = m.viewWelcome()
	case screenProfiles:
		body = m.viewProfiles()
	case screenChecklist:
		body = m.viewChecklist()
	case screenReview:
		body = m.viewReview()
	case screenProgress:
		body = m.viewProgress()
	case screenSummary:
		body = m.viewSummary()
	}
	m.help.Width = m.w
	m.help.ShowAll = m.showHelp
	footer := m.help.View(helpFor(m))
	if m.notice != "" {
		footer = warnStyle.Render(m.notice) + "\n" + footer
	}
	header := titleStyle.Render("mdt") + dimStyle.Render("  ·  "+stepNames[m.screen])
	return header + "\n\n" + body + "\n\n" + footer
}

func (m Model) installsAllowed() bool { return m.cfg.GOOS == "darwin" }

func (m Model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit
	case key.Matches(msg, keyEnter):
		if m.checking {
			m.notice = "Still checking this Mac..."
			return m, nil
		}
		m.screen = screenProfiles
	}
	return m, nil
}

func (m Model) viewWelcome() string {
	if m.checking {
		return m.spin.View() + " Checking this Mac..."
	}
	var b strings.Builder
	row := func(label, value string) { b.WriteString(boldStyle.Render(pad(label, 16)) + value + "\n") }
	row("macOS", orUnknown(m.info.MacOS))
	arch := orUnknown(m.info.Arch)
	if m.info.Rosetta {
		arch += warnStyle.Render("  (running under Rosetta)")
	}
	row("Architecture", arch)
	row("Xcode CLT", yesNo(m.installed["xcode-clt"]))
	row("Homebrew", yesNo(m.installed["homebrew"]))
	disk := "unknown"
	if m.info.FreeGB > 0 {
		disk = itoa(m.info.FreeGB) + " GB free"
		if m.info.FreeGB < 30 {
			disk = warnStyle.Render(disk + "  (Xcode and Android Studio need about 30 GB)")
		}
	}
	row("Disk", disk)
	if !m.installsAllowed() {
		b.WriteString("\n" + warnStyle.Render("Not macOS: you can preview plans, but installs are disabled.") + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("Press enter to choose what to install."))
	return b.String()
}

func yesNo(ok bool) string {
	if ok {
		return okStyle.Render("installed")
	}
	return dimStyle.Render("not installed")
}

func orUnknown(s string) string {
	if s == "" {
		return dimStyle.Render("unknown")
	}
	return s
}

func pad(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

// Run shows the TUI until the user quits and reports whether any tool
// failed to install.
func Run(ctx context.Context, cfg Config) (failed bool, err error) {
	final, err := tea.NewProgram(New(ctx, cfg), tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	if err != nil {
		return false, err
	}
	return final.(Model).Failed(), nil
}
