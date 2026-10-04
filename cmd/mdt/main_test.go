package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

// newApp returns an app with a fake runner. Scripts containing any of
// fail exit 1; everything else succeeds, so checks pass by default.
func newApp(t *testing.T, goos, stdin string, fail ...string) (*app, *runner.Fake, *strings.Builder, *strings.Builder) {
	t.Helper()
	fake := &runner.Fake{Fail: map[string]error{}}
	for _, f := range fail {
		fake.Fail[f] = errors.New("exit 1")
	}
	var stdout, stderr strings.Builder
	a := &app{
		runner: fake,
		home:   t.TempDir(),
		goos:   goos,
		now:    func() time.Time { return time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC) },
		stdin:  strings.NewReader(stdin),
		stdout: &stdout,
		stderr: &stderr,
	}
	return a, fake, &stdout, &stderr
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "Usage:"},
		{[]string{"bogus"}, `unknown command "bogus"`},
		{[]string{"install"}, "exactly one of --profile or --only"},
		{[]string{"install", "--profile", "rn", "--only", "node"}, "exactly one of --profile or --only"},
		{[]string{"install", "--profile", "ios"}, `unknown profile "ios"`},
		{[]string{"install", "--only", "node,nope"}, `unknown tool "nope"`},
		{[]string{"install", "--only", "node", "extra"}, `unexpected argument "extra"`},
		{[]string{"install", "--nope"}, "flag provided but not defined"},
		{[]string{"doctor", "--profile", "rn", "--only", "node"}, "at most one of --profile or --only"},
		{[]string{"doctor", "--profile", "ios"}, `unknown profile "ios"`},
		{[]string{"doctor", "--only", "nope"}, `unknown tool "nope"`},
		{[]string{"doctor", "extra"}, `unexpected argument "extra"`},
		{[]string{"upgrade", "--profile", "rn", "--only", "node"}, "at most one of --profile or --only"},
		{[]string{"upgrade", "--profile", "ios"}, `unknown profile "ios"`},
		{[]string{"upgrade", "--only", "nope"}, `unknown tool "nope"`},
		{[]string{"upgrade", "extra"}, `unexpected argument "extra"`},
		{[]string{"uninstall"}, "name at least one tool"},
		{[]string{"uninstall", "--dry-run"}, "name at least one tool"},
		{[]string{"uninstall", "nope"}, `unknown tool "nope"`},
		{[]string{"uninstall", "chrome", "--nope"}, "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			a, fake, _, stderr := newApp(t, "darwin", "")
			if code := a.run(context.Background(), tt.args); code != 2 {
				t.Fatalf("exit %d, want 2", code)
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr %q does not contain %q", stderr.String(), tt.want)
			}
			if len(fake.Calls()) != 0 {
				t.Fatalf("ran commands on a usage error: %v", fake.Calls())
			}
		})
	}
}

func TestDryRunNeverInstalls(t *testing.T) {
	for _, profile := range []string{"rn", "flutter", "java", "web"} {
		t.Run(profile, func(t *testing.T) {
			a, fake, stdout, _ := newApp(t, "linux", "", "-")
			if code := a.run(context.Background(), []string{"install", "--profile", profile, "--dry-run"}); code != 0 {
				t.Fatalf("exit %d", code)
			}
			if !strings.Contains(stdout.String(), "Plan:") {
				t.Fatalf("no plan in output:\n%s", stdout)
			}
			for _, call := range fake.Calls() {
				if strings.Contains(call, "install") && !strings.Contains(call, "brew list") {
					t.Fatalf("dry run ran an install: %q", call)
				}
			}
			if _, err := os.Stat(filepath.Join(a.home, LogDir)); !os.IsNotExist(err) {
				t.Fatal("dry run created a log folder")
			}
		})
	}
}

func TestInstallRefusedOffMacOS(t *testing.T) {
	a, fake, _, stderr := newApp(t, "linux", "")
	if code := a.run(context.Background(), []string{"install", "--only", "maven", "--yes"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "macOS only") || len(fake.Calls()) != 0 {
		t.Fatalf("stderr %q, calls %v", stderr, fake.Calls())
	}
}

func TestInstall(t *testing.T) {
	missing := []string{"xcode-select -p", "command -v brew", "command -v mvn"}
	tests := []struct {
		name     string
		args     []string
		stdin    string
		fail     []string
		wantCode int
		wantOut  string
		wantRan  bool
	}{
		{"confirmed", []string{"install", "--only", "maven"}, "y\n", nil, 0, "installed (3)", true},
		{"declined", []string{"install", "--only", "maven"}, "n\n", nil, 1, "Proceed?", false},
		{"yes flag", []string{"install", "--only", "maven", "--yes"}, "", nil, 0, "installed (3)", true},
		{"failure", []string{"install", "--only", "maven", "--yes"}, "", []string{"brew install maven"}, 1, "failed (1)", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, fake, stdout, _ := newApp(t, "darwin", tt.stdin, append(missing, tt.fail...)...)
			if code := a.run(context.Background(), tt.args); code != tt.wantCode {
				t.Fatalf("exit %d, want %d\n%s", code, tt.wantCode, stdout)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Fatalf("stdout does not contain %q:\n%s", tt.wantOut, stdout)
			}
			ran := false
			for _, call := range fake.Calls() {
				ran = ran || strings.Contains(call, "brew install maven")
			}
			if ran != tt.wantRan {
				t.Fatalf("installed maven = %v, want %v", ran, tt.wantRan)
			}
			if tt.wantRan {
				log := filepath.Join(a.home, LogDir, "mdt-20261001-093000.log")
				data, err := os.ReadFile(log)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), "==> Installing Maven...") {
					t.Fatalf("log is missing install lines:\n%s", data)
				}
			}
		})
	}
}

func TestList(t *testing.T) {
	a, _, stdout, _ := newApp(t, "linux", "")
	if code := a.run(context.Background(), []string{"list"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"Profiles:", "React Native", "cocoapods", "Build and release:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("list output is missing %q", want)
		}
	}
}

func TestInteractiveNeedsTerminal(t *testing.T) {
	a, _, _, stderr := newApp(t, "darwin", "")
	if code := a.run(context.Background(), []string{"tui"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "needs a terminal") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestNoArgsStartsTUIOnTerminal(t *testing.T) {
	tests := []struct {
		name     string
		failed   bool
		wantCode int
	}{
		{"clean exit", false, 0},
		{"failed installs", true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _, _, _ := newApp(t, "darwin", "")
			a.tty = true
			var started bool
			a.tui = func(context.Context) (bool, error) { started = true; return tt.failed, nil }
			if code := a.run(context.Background(), nil); code != tt.wantCode || !started {
				t.Fatalf("exit %d started %v", code, started)
			}
		})
	}
}

func TestDoctorExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		fail     []string
		wantCode int
		wantOut  string
	}{
		{"missing tool", []string{"doctor", "--only", "cocoapods"}, []string{"command -v pod"}, 1, "mdt install --only cocoapods"},
		{"default mode hides missing", []string{"doctor"}, []string{"command -v pod", "brew", "nvm.sh", "Android Studio.app"}, 0, "No problems found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _, stdout, _ := newApp(t, "linux", "", tt.fail...)
			if code := a.run(context.Background(), tt.args); code != tt.wantCode {
				t.Fatalf("exit %d, want %d\n%s", code, tt.wantCode, stdout)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Fatalf("stdout does not contain %q:\n%s", tt.wantOut, stdout)
			}
		})
	}
}

// outdatedGH makes brew outdated report a newer GitHub CLI.
const outdatedGH = `{"formulae":[{"name":"gh","installed_versions":["2.80.0"],"current_version":"2.81.0"}],"casks":[]}`

func TestUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		args     []string
		stdin    string
		fail     []string
		wantCode int
		wantOut  string
		wantRan  bool
	}{
		{"dry run", "linux", []string{"upgrade", "--only", "gh", "--dry-run"}, "", nil, 0, "brew upgrade gh", false},
		{"refused off macOS", "linux", []string{"upgrade", "--only", "gh"}, "", nil, 1, "", false},
		{"confirmed", "darwin", []string{"upgrade", "--only", "gh"}, "y\n", nil, 0, "upgraded (1): GitHub CLI", true},
		{"declined", "darwin", []string{"upgrade", "--only", "gh"}, "n\n", nil, 1, "Proceed?", false},
		{"yes flag", "darwin", []string{"upgrade", "--only", "gh", "--yes"}, "", nil, 0, "upgraded (1)", true},
		{"failure", "darwin", []string{"upgrade", "--only", "gh", "--yes"}, "", []string{"brew upgrade gh"}, 1, "failed (1)", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, fake, stdout, stderr := newApp(t, tt.goos, tt.stdin, tt.fail...)
			fake.Output = map[string]string{"brew outdated": outdatedGH}
			if code := a.run(context.Background(), tt.args); code != tt.wantCode {
				t.Fatalf("exit %d, want %d\n%s%s", code, tt.wantCode, stdout, stderr)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Fatalf("stdout does not contain %q:\n%s", tt.wantOut, stdout)
			}
			ran := false
			for _, call := range fake.Calls() {
				ran = ran || strings.Contains(call, "brew upgrade gh")
			}
			if ran != tt.wantRan {
				t.Fatalf("ran brew upgrade = %v, want %v", ran, tt.wantRan)
			}
		})
	}
}

func TestUpgradeNothingToDo(t *testing.T) {
	a, fake, stdout, _ := newApp(t, "darwin", "")
	fake.Output = map[string]string{"brew outdated": `{"formulae":[],"casks":[]}`}
	if code := a.run(context.Background(), []string{"upgrade", "--only", "gh"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout.String(), "0 available, 1 up to date") {
		t.Fatalf("stdout:\n%s", stdout)
	}
	if strings.Contains(stdout.String(), "Proceed?") {
		t.Fatal("asked to confirm with nothing to upgrade")
	}
}

func TestUninstall(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		args     []string
		stdin    string
		fail     []string
		wantCode int
		wantOut  string
		wantRan  bool
	}{
		{"dry run, flag last", "linux", []string{"uninstall", "chrome", "--dry-run"}, "", nil, 0, "brew uninstall --cask google-chrome", false},
		{"dry run, flag first", "linux", []string{"uninstall", "--dry-run", "chrome"}, "", nil, 0, "remove", false},
		{"refused off macOS", "linux", []string{"uninstall", "chrome"}, "", nil, 1, "", false},
		{"confirmed", "darwin", []string{"uninstall", "chrome"}, "y\n", nil, 0, "uninstalled (1): Google Chrome", true},
		{"declined", "darwin", []string{"uninstall", "chrome"}, "n\n", nil, 1, "Proceed?", false},
		{"yes flag", "darwin", []string{"uninstall", "chrome", "--yes"}, "", nil, 0, "uninstalled (1)", true},
		{"failure", "darwin", []string{"uninstall", "chrome", "--yes"}, "", []string{"brew uninstall --cask google-chrome"}, 1, "failed (1)", true},
		{"all refused", "darwin", []string{"uninstall", "xcode", "--yes"}, "", nil, 1, "refused", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, fake, stdout, stderr := newApp(t, tt.goos, tt.stdin, tt.fail...)
			if code := a.run(context.Background(), tt.args); code != tt.wantCode {
				t.Fatalf("exit %d, want %d\n%s%s", code, tt.wantCode, stdout, stderr)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Fatalf("stdout does not contain %q:\n%s", tt.wantOut, stdout)
			}
			ran := false
			for _, call := range fake.Calls() {
				ran = ran || strings.Contains(call, "brew uninstall")
			}
			if ran != tt.wantRan {
				t.Fatalf("ran brew uninstall = %v, want %v", ran, tt.wantRan)
			}
		})
	}
}
