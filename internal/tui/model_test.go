package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func newModel(t *testing.T, fake *runner.Fake, goos string) Model {
	t.Helper()
	dir := t.TempDir()
	cfg := Config{
		Catalog: loadCatalog(t),
		Runner:  fake,
		Home:    dir,
		GOOS:    goos,
		OpenLog: func() (string, io.WriteCloser, error) {
			return filepath.Join(dir, "log"), nopCloser{io.Discard}, nil
		},
	}
	m := New(context.Background(), cfg)
	m.w, m.h = 100, 30
	return m
}

// drain runs cmd and feeds its messages back into the model until the
// commands run dry. Spinner ticks are dropped so it terminates.
func drain(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case nil, spinner.TickMsg:
		case tea.BatchMsg:
			queue = append(queue, msg...)
		default:
			next, nc := m.Update(msg)
			m = next.(Model)
			queue = append(queue, nc)
		}
	}
	return m
}

func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, cmd := m.Update(msg)
		m = drain(t, next.(Model), cmd)
	}
	return m
}

func ready(t *testing.T, fake *runner.Fake, goos string) Model {
	t.Helper()
	m := newModel(t, fake, goos)
	m = drain(t, m, func() tea.Msg { return m.Init()().(tea.BatchMsg)[1]() })
	if m.checking {
		t.Fatal("system check did not finish")
	}
	return m
}

func TestFlowToReview(t *testing.T) {
	fake := &runner.Fake{
		Fail:   map[string]error{"command -v pod": errors.New("exit 1")},
		Output: map[string]string{"sw_vers": "15.1\n", "uname": "arm64\n"},
	}
	m := ready(t, fake, "linux")
	if m.info.MacOS != "15.1" {
		t.Fatalf("info = %+v", m.info)
	}
	if !strings.Contains(m.View(), "Not macOS") {
		t.Fatal("welcome should say installs are disabled")
	}

	m = press(t, m, "enter") // profiles
	if m.screen != screenProfiles {
		t.Fatalf("screen = %v", m.screen)
	}
	m = press(t, m, "enter") // first profile: rn
	if m.screen != screenChecklist || !m.sel.Has("cocoapods") {
		t.Fatalf("screen %v, cocoapods selected %v", m.screen, m.sel.Has("cocoapods"))
	}

	// Deselect everything, then nothing is left to review.
	m = press(t, m, "a")
	if len(m.sel.IDs()) != 0 {
		t.Fatalf("select all should toggle off: %v", m.sel.IDs())
	}
	m = press(t, m, "enter")
	if m.screen != screenChecklist || !strings.Contains(m.View(), "Nothing to install") {
		t.Fatalf("expected a notice, screen %v", m.screen)
	}

	m = press(t, m, "a", "enter")
	if m.screen != screenReview {
		t.Fatalf("screen = %v", m.screen)
	}
	m = press(t, m, "enter")
	if m.screen != screenReview || !strings.Contains(m.View(), "macOS only") {
		t.Fatal("installs must be refused off macOS")
	}
	m = press(t, m, "esc", "esc")
	if m.screen != screenProfiles {
		t.Fatalf("esc should walk back, screen = %v", m.screen)
	}
}

func TestChecklistFilterAndRequired(t *testing.T) {
	m := ready(t, &runner.Fake{Fail: map[string]error{"": errors.New("exit 1")}}, "linux")
	m = press(t, m, "enter", "down", "down", "down", "down", "enter") // custom profile
	if m.screen != screenChecklist || len(m.sel.IDs()) != 0 {
		t.Fatalf("screen %v ids %v", m.screen, m.sel.IDs())
	}

	m = press(t, m, "/", "c", "o", "c", "o", "a", "enter")
	tools := m.visible()
	if len(tools) != 1 || tools[0].ID != "cocoapods" {
		t.Fatalf("visible = %v", tools)
	}
	m = press(t, m, " ")
	if !m.sel.Has("cocoapods") || m.required["homebrew"] == "" {
		t.Fatalf("cocoapods should pull in homebrew: %v", m.required)
	}

	// Dependencies cannot be removed while something needs them.
	m = press(t, m, "esc", "/", "h", "o", "m", "e", "b", "r", "e", "w", "enter", " ")
	if m.sel.Has("homebrew") || !strings.Contains(m.View(), "cannot be removed") {
		t.Fatalf("homebrew toggled or no notice: %q", m.notice)
	}
}

func TestInstallFlowOnMac(t *testing.T) {
	fake := &runner.Fake{Fail: map[string]error{"command -v pod": errors.New("exit 1")}}
	m := ready(t, fake, "darwin")
	m = press(t, m, "enter", "enter") // rn profile
	m = press(t, m, "enter")          // review
	if m.screen != screenReview {
		t.Fatalf("screen = %v", m.screen)
	}
	m = press(t, m, "enter") // start, no sudo configured
	if m.screen != screenSummary {
		t.Fatalf("screen = %v, notice %q", m.screen, m.notice)
	}
	if m.Failed() {
		t.Fatal("nothing should fail with the fake runner")
	}
	if !strings.Contains(m.View(), "Next steps") {
		t.Fatalf("summary view: %s", m.View())
	}
	var ran bool
	for _, c := range fake.Calls() {
		ran = ran || strings.Contains(c, "brew install cocoapods")
	}
	if !ran {
		t.Fatal("cocoapods was not installed")
	}
	if _, err := os.Stat(m.cfg.Home); err != nil {
		t.Fatal(err)
	}
}

func TestInstallFailureShownInSummary(t *testing.T) {
	fake := &runner.Fake{Fail: map[string]error{
		"command -v pod":         errors.New("exit 1"),
		"brew install cocoapods": errors.New("exit 1"),
	}}
	m := ready(t, fake, "darwin")
	m = press(t, m, "enter", "enter", "enter", "enter")
	if m.screen != screenSummary || !m.Failed() {
		t.Fatalf("screen %v failed %v", m.screen, m.Failed())
	}
	if !strings.Contains(m.View(), "failed") {
		t.Fatalf("summary: %s", m.View())
	}
}

func TestAppendLog(t *testing.T) {
	var p progressState
	p.appendLog("\x1b[32mhello\x1b[0m wor")
	p.appendLog("ld\r50%\r100%\nlast")
	want := []string{"hello world", "50%", "100%"}
	if strings.Join(p.lines, "|") != strings.Join(want, "|") || p.partial != "last" {
		t.Fatalf("lines %q partial %q", p.lines, p.partial)
	}
}
