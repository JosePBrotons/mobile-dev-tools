package upgrade

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

// outdated has packages the catalog knows (gh and git as formulae, iTerm2 as
// a cask), a pinned formula, and two that must be ignored: wget is not in the
// catalog and the cask named gh is not how the catalog installs gh.
const outdated = `{
  "formulae": [
    {"name": "gh", "installed_versions": ["2.80.0"], "current_version": "2.81.0", "pinned": false},
    {"name": "git", "installed_versions": ["2.50.0"], "current_version": "2.51.0", "pinned": true},
    {"name": "wget", "installed_versions": ["1.0"], "current_version": "1.1", "pinned": false}
  ],
  "casks": [
    {"name": "iterm2", "installed_versions": "3.5.0", "current_version": "3.5.1"},
    {"name": "gh", "installed_versions": ["1"], "current_version": "2"}
  ]
}`

func load(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// newFake answers brew outdated and the nvm version query. Scripts that
// contain a key of fail exit 1, which also makes that tool's check fail.
func newFake(brewJSON, nodeOut string, fail ...string) *runner.Fake {
	f := &runner.Fake{
		Fail:   map[string]error{},
		Output: map[string]string{"brew outdated": brewJSON, "nvm version default": nodeOut},
	}
	for _, k := range fail {
		f.Fail[k] = errors.New("exit 1")
	}
	return f
}

func itemIDs(p *Plan) []string {
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Tool.ID)
	}
	return out
}

func find(t *testing.T, p *Plan, id string) Item {
	t.Helper()
	for _, it := range p.Items {
		if it.Tool.ID == id {
			return it
		}
	}
	t.Fatalf("no upgrade for %s in %v", id, itemIDs(p))
	return Item{}
}

func TestCheckBrew(t *testing.T) {
	p, err := Check(context.Background(), load(t), newFake(outdated, "v22.2.0\nv22.2.0\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"git", "gh", "iterm2"}; !slices.Equal(itemIDs(p), want) {
		t.Fatalf("items = %v, want %v", itemIDs(p), want)
	}
	gh := find(t, p, "gh")
	if gh.From != "2.80.0" || gh.To != "2.81.0" || !slices.Equal(gh.Commands, []string{"brew upgrade gh"}) || gh.Skip != "" {
		t.Errorf("gh = %+v", gh)
	}
	iterm := find(t, p, "iterm2")
	if iterm.From != "3.5.0" || !slices.Equal(iterm.Commands, []string{"brew upgrade --cask iterm2"}) {
		t.Errorf("iterm2 = %+v", iterm)
	}
	if git := find(t, p, "git"); git.Skip != "pinned" {
		t.Errorf("git = %+v, want pinned", git)
	}
	if got := len(p.Runnable()); got != 2 {
		t.Errorf("runnable = %d, want 2", got)
	}
}

func TestCheckNode(t *testing.T) {
	tests := []struct {
		name     string
		ids      []string
		nodeOut  string
		fail     []string
		wantFrom string
		wantTo   string
		wantItem bool
	}{
		{"newer LTS", nil, "v20.1.0\nv22.2.0\n", nil, "v20.1.0", "v22.2.0", true},
		{"current", nil, "v22.2.0\nv22.2.0\n", nil, "", "", false},
		{"no default yet", nil, "N/A\nv22.2.0\n", nil, "none", "v22.2.0", true},
		{"only node, nvm not selected", []string{"node"}, "v20.1.0\nv22.2.0\n", nil, "v20.1.0", "v22.2.0", true},
		{"nvm missing", []string{"node"}, "v20.1.0\nv22.2.0\n", []string{"nvm.sh"}, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Check(context.Background(), load(t), newFake(`{}`, tt.nodeOut, tt.fail...), tt.ids)
			if err != nil {
				t.Fatal(err)
			}
			if got := slices.Contains(itemIDs(p), "node"); got != tt.wantItem {
				t.Fatalf("node upgrade = %v, want %v (%v)", got, tt.wantItem, itemIDs(p))
			}
			if !tt.wantItem {
				return
			}
			it := find(t, p, "node")
			if it.From != tt.wantFrom || it.To != tt.wantTo || len(it.Commands) != 2 {
				t.Errorf("node = %+v", it)
			}
			if !strings.Contains(it.Commands[0], "--reinstall-packages-from=default") {
				t.Errorf("commands %v do not keep global packages", it.Commands)
			}
		})
	}
}

func TestCheckNodeBadOutput(t *testing.T) {
	_, err := Check(context.Background(), load(t), newFake(`{}`, "N/A\nN/A\n"), []string{"node"})
	if err == nil || !strings.Contains(err.Error(), "unexpected nvm output") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckSelection(t *testing.T) {
	c := load(t)
	t.Run("only", func(t *testing.T) {
		p, err := Check(context.Background(), c, newFake(outdated, ""), []string{"gh"})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"gh"}; !slices.Equal(itemIDs(p), want) {
			t.Fatalf("items = %v, want %v", itemIDs(p), want)
		}
	})
	t.Run("not installed is ignored", func(t *testing.T) {
		p, err := Check(context.Background(), c, newFake(outdated, ""), []string{"gh"})
		if err != nil || len(p.Items) != 1 {
			t.Fatalf("setup: %v %v", p, err)
		}
		p, err = Check(context.Background(), c, newFake(outdated, "", "command -v gh"), []string{"gh"})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Items) != 0 || p.UpToDate != 0 {
			t.Fatalf("plan = %+v", p)
		}
	})
	t.Run("unknown tool", func(t *testing.T) {
		if _, err := Check(context.Background(), c, newFake(outdated, ""), []string{"nope"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("script tools are not checked", func(t *testing.T) {
		fake := newFake(outdated, "")
		p, err := Check(context.Background(), c, fake, []string{"typescript", "xcode"})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Items) != 0 || p.UpToDate != 0 {
			t.Fatalf("plan = %+v", p)
		}
		for _, call := range fake.Calls() {
			if strings.Contains(call, "brew outdated") || strings.Contains(call, "brew update") {
				t.Errorf("ran brew without a brew tool: %q", call)
			}
		}
	})
}

func TestCheckBrewFailure(t *testing.T) {
	tests := []struct {
		name string
		json string
		fail []string
	}{
		{"outdated fails", `{}`, []string{"brew outdated"}},
		{"bad json", `not json`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Check(context.Background(), load(t), newFake(tt.json, "", tt.fail...), []string{"gh"}); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestCheckBrewUpdateIsBestEffort(t *testing.T) {
	// "brew update" is a substring of nothing else the check runs.
	p, err := Check(context.Background(), load(t), newFake(outdated, "", "brew update"), []string{"gh"})
	if err != nil || len(p.Items) != 1 {
		t.Fatalf("plan %+v, err %v", p, err)
	}
}

func TestExecute(t *testing.T) {
	c := load(t)
	fake := newFake(outdated, "v20.1.0\nv22.2.0\n")
	p, err := Check(context.Background(), c, fake, nil)
	if err != nil {
		t.Fatal(err)
	}
	run := newFake("", "", "brew upgrade --cask iterm2")
	var log bytes.Buffer
	var events []plan.Event
	report := Execute(context.Background(), p, run, &log, func(e plan.Event) { events = append(events, e) })

	got := map[string]plan.Result{}
	for _, r := range report.Results {
		got[r.ID] = r
	}
	if got["gh"].Status != plan.StatusUpgraded || got["node"].Status != plan.StatusUpgraded {
		t.Errorf("results = %+v", report.Results)
	}
	if got["iterm2"].Status != plan.StatusFailed || got["iterm2"].Err == nil {
		t.Errorf("iterm2 = %+v", got["iterm2"])
	}
	if r := got["git"]; r.Status != plan.StatusSkipped || r.Err == nil || r.Err.Error() != "pinned" {
		t.Errorf("git = %+v", r)
	}
	if !report.Failed() {
		t.Error("report should be failed")
	}
	// A failure did not stop the later upgrade.
	if idx := slices.Index(itemIDs(p), "node"); idx < slices.Index(itemIDs(p), "iterm2") {
		t.Fatalf("test needs node after iterm2, items %v", itemIDs(p))
	}
	calls := strings.Join(run.Calls(), "\n")
	if strings.Contains(calls, "brew upgrade git") {
		t.Error("ran the pinned upgrade")
	}
	for _, want := range []string{"brew upgrade gh", "nvm install --lts --reinstall-packages-from=default", "nvm alias default"} {
		if !strings.Contains(calls, want) {
			t.Errorf("calls are missing %q", want)
		}
	}
	if !strings.Contains(log.String(), "==> Upgrading GitHub CLI...") {
		t.Errorf("log = %q", log.String())
	}
	started := 0
	for _, e := range events {
		if e.Kind == plan.Started {
			started++
		}
	}
	if started != 3 {
		t.Errorf("started events = %d, want 3", started)
	}
}

func TestNextSteps(t *testing.T) {
	node := Item{Tool: catalog.Tool{ID: "node", Name: "Node.js LTS"}, From: "v20.1.0", To: "v22.2.0"}
	gh := Item{Tool: catalog.Tool{ID: "gh", Name: "GitHub CLI"}}
	p := &Plan{Items: []Item{gh, node}}
	tests := []struct {
		name   string
		report plan.Report
		want   []string
	}{
		{"nothing upgraded", plan.Report{Results: []plan.Result{{ID: "gh", Status: plan.StatusFailed}}}, nil},
		{"gh only", plan.Report{Results: []plan.Result{{ID: "gh", Status: plan.StatusUpgraded}}}, []string{plan.GenericNextStep}},
		{"node", plan.Report{Results: []plan.Result{{ID: "node", Status: plan.StatusUpgraded}}}, []string{
			"Node.js: v20.1.0 is still installed; free the space with: nvm uninstall v20.1.0", plan.GenericNextStep}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NextSteps(p, tt.report); !slices.Equal(got, tt.want) {
				t.Fatalf("steps = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintGolden(t *testing.T) {
	p, err := Check(context.Background(), load(t), newFake(outdated, "v20.1.0\nv22.2.0\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := p.Print(&buf); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "upgrade.golden")
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
		t.Fatalf("output changed (go test ./internal/upgrade -update to accept):\n%s", buf.String())
	}
}

func TestPrintNothingToUpgrade(t *testing.T) {
	var buf bytes.Buffer
	if err := (&Plan{UpToDate: 4}).Print(&buf); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "Upgrades: 0 available, 4 up to date\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
