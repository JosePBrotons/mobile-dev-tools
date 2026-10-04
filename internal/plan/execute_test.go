package plan

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

func tool(id string, m catalog.Method, requires ...string) catalog.Tool {
	return catalog.Tool{ID: id, Name: id, Method: m, Package: id, Install: []string{"install-" + id}, Requires: requires}
}

func statuses(r Report) map[string]Status {
	out := map[string]Status{}
	for _, res := range r.Results {
		out[res.ID] = res.Status
	}
	return out
}

func TestExecute(t *testing.T) {
	brew := tool("brew", catalog.MethodScript)
	a := tool("a", catalog.MethodBrew, "brew")
	b := tool("b", catalog.MethodCask, "brew")
	c := tool("c", catalog.MethodScript, "a")
	d := tool("d", catalog.MethodScript, "c")

	tests := []struct {
		name       string
		items      []Item
		fail       string
		want       map[string]Status
		wantUpdate int
	}{
		{
			name:       "all succeed, one brew update",
			items:      []Item{{Tool: brew}, {Tool: a}, {Tool: b}},
			want:       map[string]Status{"brew": StatusInstalled, "a": StatusInstalled, "b": StatusInstalled},
			wantUpdate: 1,
		},
		{
			name:       "no brew update when brew tools are present",
			items:      []Item{{Tool: brew}, {Tool: a, Installed: true}, {Tool: c}},
			want:       map[string]Status{"brew": StatusInstalled, "a": StatusPresent, "c": StatusInstalled},
			wantUpdate: 0,
		},
		{
			name:       "failure skips dependents only",
			items:      []Item{{Tool: brew}, {Tool: a}, {Tool: b}, {Tool: c}, {Tool: d}},
			fail:       "brew install a",
			want:       map[string]Status{"brew": StatusInstalled, "a": StatusFailed, "b": StatusInstalled, "c": StatusSkipped, "d": StatusSkipped},
			wantUpdate: 1,
		},
		{
			name:  "failed brew update does not stop installs",
			items: []Item{{Tool: a}},
			fail:  BrewUpdate,
			want:  map[string]Status{"a": StatusInstalled},
			// The failing update is still attempted once.
			wantUpdate: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &runner.Fake{Fail: map[string]error{}}
			if tt.fail != "" {
				fake.Fail[tt.fail] = errors.New("exit 1")
			}
			var events []Event
			report := Execute(context.Background(), &Plan{Items: tt.items}, fake, Options{Home: t.TempDir()},
				func(e Event) { events = append(events, e) })

			got := statuses(report)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for id, want := range tt.want {
				if got[id] != want {
					t.Errorf("%s: %s, want %s", id, got[id], want)
				}
			}
			updates := 0
			for _, call := range fake.Calls() {
				if !strings.HasPrefix(call, shellenv.Prelude) {
					t.Errorf("script ran without the prelude: %q", call)
				}
				if strings.HasSuffix(call, BrewUpdate) {
					updates++
				}
			}
			if updates != tt.wantUpdate {
				t.Errorf("brew update ran %d times, want %d", updates, tt.wantUpdate)
			}
			if report.Failed() != (tt.fail != "" && tt.fail != BrewUpdate) {
				t.Errorf("Failed() = %v", report.Failed())
			}
			finished := 0
			for _, e := range events {
				if e.Kind == Finished && e.Result.ID != "brew-update" {
					finished++
				}
			}
			if finished != len(tt.items) {
				t.Errorf("got %d Finished events, want %d", finished, len(tt.items))
			}
		})
	}
}

func TestExecuteAndroidHomeHook(t *testing.T) {
	studio := catalog.Tool{ID: "android-studio", Name: "Android Studio", Method: catalog.MethodCask,
		Package: "android-studio", PostInstall: []string{"android-home-env"}}
	tests := []struct {
		name      string
		installed bool
	}{
		{"after install", false},
		{"already installed", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			p := &Plan{Items: []Item{{Tool: studio, Installed: tt.installed}}}
			report := Execute(context.Background(), p, &runner.Fake{}, Options{Home: home}, nil)
			if report.Failed() {
				t.Fatalf("report: %+v", report)
			}
			data, err := os.ReadFile(filepath.Join(home, ".zprofile"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), shellenv.AndroidHomeLines[0]) {
				t.Fatalf(".zprofile = %q", data)
			}
		})
	}
}

func TestNextSteps(t *testing.T) {
	a := tool("a", catalog.MethodScript)
	a.NextSteps = []string{"sign in"}
	b := tool("b", catalog.MethodScript)
	b.NextSteps = []string{"never shown"}
	p := &Plan{Items: []Item{{Tool: a}, {Tool: b, Installed: true}}}

	got := NextSteps(p, Report{Results: []Result{{ID: "a", Status: StatusInstalled}, {ID: "b", Status: StatusPresent}}})
	want := []string{"a: sign in", GenericNextStep}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := NextSteps(p, Report{Results: []Result{{ID: "b", Status: StatusPresent}}}); got != nil {
		t.Fatalf("got %v, want none", got)
	}
}

func TestExecuteBrewShellenvHook(t *testing.T) {
	brew := catalog.Tool{ID: "homebrew", Name: "Homebrew", Method: catalog.MethodScript,
		Install: []string{"true"}, PostInstall: []string{"brew-shellenv"}}
	tests := []struct {
		name    string
		profile string
		want    string
	}{
		{"missing profile", "", shellenv.BrewShellenvLine + "\n"},
		{"added once", shellenv.BrewShellenvLine + "\n", shellenv.BrewShellenvLine + "\n"},
		{"hand written line", "eval \"$(/opt/homebrew/bin/brew shellenv)\"\n", "eval \"$(/opt/homebrew/bin/brew shellenv)\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, ".zprofile")
			if tt.profile != "" {
				if err := os.WriteFile(path, []byte(tt.profile), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			p := &Plan{Items: []Item{{Tool: brew, Installed: true}}}
			if report := Execute(context.Background(), p, &runner.Fake{}, Options{Home: home}, nil); report.Failed() {
				t.Fatalf("report: %+v", report)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want {
				t.Fatalf(".zprofile = %q, want %q", data, tt.want)
			}
		})
	}
}

func TestExecuteInteractive(t *testing.T) {
	sdk := tool("sdk", catalog.MethodScript)
	sdk.Interactive = true
	tests := []struct {
		name       string
		hook       bool
		wantRunner int
		wantHook   int
		hookErr    error
		wantStatus Status
	}{
		{"hook runs the step", true, 0, 1, nil, StatusInstalled},
		{"hook failure fails the tool", true, 0, 1, errors.New("declined"), StatusFailed},
		{"no hook uses the runner", false, 1, 0, nil, StatusInstalled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &runner.Fake{}
			opts := Options{Home: t.TempDir()}
			hooked := 0
			if tt.hook {
				opts.Interactive = func(_ context.Context, script string, _ io.Writer) error {
					hooked++
					if !strings.HasPrefix(script, shellenv.Prelude) || !strings.Contains(script, "install-sdk") {
						t.Errorf("unexpected script %q", script)
					}
					return tt.hookErr
				}
			}
			report := Execute(context.Background(), &Plan{Items: []Item{{Tool: sdk}}}, fake, opts, nil)
			if got := report.Results[0].Status; got != tt.wantStatus {
				t.Errorf("status %s, want %s", got, tt.wantStatus)
			}
			if hooked != tt.wantHook || len(fake.Calls()) != tt.wantRunner {
				t.Errorf("hook ran %d times, runner %d", hooked, len(fake.Calls()))
			}
		})
	}
}

func TestResolveInteractiveLast(t *testing.T) {
	c := load(t)
	for _, profile := range []string{"rn", "flutter"} {
		items, err := Resolve(c, profileIDs(c, profile))
		if err != nil {
			t.Fatal(err)
		}
		last := items[len(items)-1].Tool
		if last.ID != "android-sdk" || !last.Interactive {
			t.Errorf("%s: last tool is %s, want android-sdk", profile, last.ID)
		}
		seen := map[string]bool{}
		for _, it := range items {
			for _, r := range it.Tool.Requires {
				if !seen[r] {
					t.Errorf("%s: %s comes before its dependency %s", profile, it.Tool.ID, r)
				}
			}
			seen[it.Tool.ID] = true
		}
	}
}
