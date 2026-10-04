package uninstall

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

var update = flag.Bool("update", false, "rewrite golden files")

func load(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// parse builds a small catalog: b requires a, and both are brew tools.
func parse(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Parse([]byte(`
categories: [{id: system, name: System}]
profiles: [{id: rn, name: RN}]
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: pa, check: check-a}
  - {id: b, name: B, description: d, category: system, method: cask, package: pb, check: check-b, requires: [a]}
`))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// fake makes every check pass except those containing a key of missing.
func fake(missing ...string) *runner.Fake {
	f := &runner.Fake{Fail: map[string]error{}}
	for _, m := range missing {
		f.Fail[m] = errors.New("exit 1")
	}
	return f
}

func check(t *testing.T, c *catalog.Catalog, f *runner.Fake, ids ...string) *Plan {
	t.Helper()
	p, err := Check(context.Background(), c, f, ids)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func item(t *testing.T, p *Plan, id string) Item {
	t.Helper()
	for _, it := range p.Items {
		if it.Tool.ID == id {
			return it
		}
	}
	t.Fatalf("no item %s", id)
	return Item{}
}

func TestCheckCommands(t *testing.T) {
	c := load(t)
	tests := []struct {
		id   string
		want []string
	}{
		{"chrome", []string{"brew uninstall --cask google-chrome"}},
		{"gh", []string{"brew uninstall gh"}},
		{"typescript", []string{"npm uninstall -g typescript"}},
		{"mkcert", []string{"mkcert -uninstall", "brew uninstall mkcert"}},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			it := item(t, check(t, c, fake(), tt.id), tt.id)
			if it.Refused != "" || !slices.Equal(it.Commands, tt.want) {
				t.Fatalf("item = %+v, want commands %v", it, tt.want)
			}
		})
	}
}

func TestCheckRefusals(t *testing.T) {
	c := load(t)
	tests := []struct {
		name    string
		id      string
		missing []string
		want    string
	}{
		{"not installed", "gh", []string{"command -v gh"}, "not installed"},
		{"mas", "xcode", nil, "remove it by hand"},
		{"script without uninstall", "nvm", []string{"command -v node"}, "no automatic uninstall"},
		{"installed dependent", "node", nil, "needed by TypeScript"},
		{"system tool", "homebrew", nil, "needed by"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := item(t, check(t, c, fake(tt.missing...), tt.id), tt.id)
			if !strings.Contains(it.Refused, tt.want) || len(it.Commands) != 0 {
				t.Fatalf("item = %+v, want refusal containing %q", it, tt.want)
			}
		})
	}
}

func TestCheckRefusalCascades(t *testing.T) {
	// Xcode cannot be removed here, so mas stays for it.
	p := check(t, load(t), fake(), "mas", "xcode")
	if got := item(t, p, "mas").Refused; got != "needed by Xcode" {
		t.Fatalf("mas refused = %q", got)
	}
	if len(p.Runnable()) != 0 {
		t.Fatalf("runnable = %v", p.Runnable())
	}
}

func TestCheckDependentsRemovedTogether(t *testing.T) {
	c := parse(t)
	t.Run("alone", func(t *testing.T) {
		if got := item(t, check(t, c, fake(), "a"), "a").Refused; got != "needed by B" {
			t.Fatalf("a refused = %q", got)
		}
	})
	t.Run("together", func(t *testing.T) {
		p := check(t, c, fake(), "a", "b")
		var order []string
		for _, it := range p.Items {
			order = append(order, it.Tool.ID)
		}
		if !slices.Equal(order, []string{"b", "a"}) || len(p.Runnable()) != 2 {
			t.Fatalf("order %v, runnable %d", order, len(p.Runnable()))
		}
	})
	t.Run("dependent not installed", func(t *testing.T) {
		if it := item(t, check(t, c, fake("check-b"), "a"), "a"); it.Refused != "" {
			t.Fatalf("a refused = %q", it.Refused)
		}
	})
}

func TestCheckUnknownTool(t *testing.T) {
	if _, err := Check(context.Background(), load(t), fake(), []string{"nope"}); err == nil {
		t.Fatal("want an error")
	}
}

func TestExecute(t *testing.T) {
	c := parse(t)
	p := check(t, c, fake(), "a", "b")
	t.Run("order and statuses", func(t *testing.T) {
		run := fake()
		var log bytes.Buffer
		report := Execute(context.Background(), p, run, &log, nil)
		if report.Failed() {
			t.Fatalf("report = %+v", report)
		}
		calls := run.Calls()
		if len(calls) != 2 || !strings.HasSuffix(calls[0], "brew uninstall --cask pb") || !strings.HasSuffix(calls[1], "brew uninstall pa") {
			t.Fatalf("calls = %q", calls)
		}
		for _, r := range report.Results {
			if r.Status != plan.StatusRemoved {
				t.Errorf("result = %+v", r)
			}
		}
		if !strings.Contains(log.String(), "==> Uninstalling B...") {
			t.Errorf("log = %q", log.String())
		}
	})
	t.Run("failure keeps what it needs", func(t *testing.T) {
		run := fake("brew uninstall --cask pb")
		report := Execute(context.Background(), p, run, nil, nil)
		if report.Results[0].Status != plan.StatusFailed {
			t.Fatalf("b = %+v", report.Results[0])
		}
		a := report.Results[1]
		if a.Status != plan.StatusSkipped || a.Err == nil || !strings.Contains(a.Err.Error(), "needed by B") {
			t.Fatalf("a = %+v", a)
		}
		if len(run.Calls()) != 1 {
			t.Fatalf("calls = %q", run.Calls())
		}
	})
	t.Run("refused items are skipped", func(t *testing.T) {
		run := fake()
		report := Execute(context.Background(), check(t, c, fake(), "a"), run, nil, nil)
		if len(run.Calls()) != 0 || report.Results[0].Status != plan.StatusSkipped {
			t.Fatalf("calls %q, report %+v", run.Calls(), report)
		}
	})
}

func TestNextSteps(t *testing.T) {
	c := load(t)
	p := check(t, c, fake(), "android-studio", "chrome")
	removed := plan.Report{Results: []plan.Result{
		{ID: "android-studio", Status: plan.StatusRemoved},
		{ID: "chrome", Status: plan.StatusRemoved},
	}}
	steps := NextSteps(p, removed)
	if len(steps) != 2 || !strings.Contains(steps[0], "Android Studio: remove the lines added to ~/.zprofile by android-home-env") ||
		steps[1] != plan.GenericNextStep {
		t.Fatalf("steps = %q", steps)
	}
	if got := NextSteps(p, plan.Report{Results: []plan.Result{{ID: "chrome", Status: plan.StatusFailed}}}); got != nil {
		t.Fatalf("steps after failure = %q", got)
	}
}

func TestPrintGolden(t *testing.T) {
	p := check(t, load(t), fake("command -v gh"), "mkcert", "chrome", "typescript", "xcode", "nvm", "gh")
	var buf bytes.Buffer
	if err := p.Print(&buf); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "uninstall.golden")
	if *update {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != string(want) {
		t.Fatalf("output changed (go test ./internal/uninstall -update to accept):\n%s", buf.String())
	}
}
