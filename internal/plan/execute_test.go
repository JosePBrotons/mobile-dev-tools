package plan

import (
	"context"
	"errors"
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
