// Package upgrade finds installed catalog tools that have a newer version
// and upgrades them through Homebrew and nvm. Like internal/plan it only
// runs commands through a runner.Runner.
package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/detect"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// brewOutdated lists outdated packages as JSON. Stderr is dropped so Homebrew
// warnings cannot corrupt the output.
const brewOutdated = "brew outdated --json=v2 2>/dev/null"

// nodeVersions prints the default Node version and the latest LTS, one per
// line. nvm prints N/A when it has no match.
const nodeVersions = `printf '%s\n%s\n' "$(nvm version default)" "$(nvm version-remote --lts)"`

// nodeCommands move the default Node to the latest LTS and keep the global
// npm packages, such as TypeScript.
var nodeCommands = []string{
	"nvm install --lts --reinstall-packages-from=default",
	"nvm alias default 'lts/*'",
}

// Item is one tool with an upgrade available.
type Item struct {
	Tool     catalog.Tool
	From, To string
	Commands []string
	// Skip says why the upgrade is listed but not run, for example a pinned
	// formula. Empty means it runs.
	Skip string
}

// Plan lists the available upgrades in catalog order.
type Plan struct {
	Items []Item
	// UpToDate counts installed tools that were checked and are current.
	UpToDate int
}

// Runnable returns the items that will be upgraded.
func (p *Plan) Runnable() []Item {
	var out []Item
	for _, it := range p.Items {
		if it.Skip == "" {
			out = append(out, it)
		}
	}
	return out
}

// versions decodes Homebrew's installed_versions, a list for formulae and
// either a list or a string for casks.
type versions []string

func (v *versions) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*v = versions{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*v = many
	return nil
}

type outdatedPackage struct {
	Name              string   `json:"name"`
	InstalledVersions versions `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	Pinned            bool     `json:"pinned"`
}

type outdatedJSON struct {
	Formulae []outdatedPackage `json:"formulae"`
	Casks    []outdatedPackage `json:"casks"`
}

// Check finds the upgrades for ids. Nil ids checks every installed tool.
// Dependencies are not added: upgrading only touches what was asked for.
// Homebrew is updated first so its outdated list is current; that update is
// best effort.
func Check(ctx context.Context, c *catalog.Catalog, r runner.Runner, ids []string) (*Plan, error) {
	tools := c.Tools
	if len(ids) > 0 {
		tools = nil
		for _, id := range ids {
			t, ok := c.ByID(id)
			if !ok {
				return nil, fmt.Errorf("unknown tool %q (run mdt list)", id)
			}
			tools = append(tools, t)
		}
	}
	installed := detect.Installed(ctx, r, tools)

	usesBrew := false
	for _, t := range tools {
		if installed[t.ID] && (t.Method == catalog.MethodBrew || t.Method == catalog.MethodCask) {
			usesBrew = true
		}
	}
	var brew outdatedJSON
	if usesBrew {
		_ = r.Run(ctx, shellenv.Prelude+plan.BrewUpdate, io.Discard)
		var out bytes.Buffer
		if err := r.Run(ctx, shellenv.Prelude+brewOutdated, &out); err != nil {
			return nil, fmt.Errorf("brew outdated: %w", err)
		}
		if err := json.Unmarshal(out.Bytes(), &brew); err != nil {
			return nil, fmt.Errorf("parse brew outdated: %w", err)
		}
	}

	// Node.js is upgraded through nvm, which may not be in the selection.
	nvmReady := installed["nvm"]
	if nvm, ok := c.ByID("nvm"); ok && installed["node"] && !nvmReady {
		nvmReady = detect.Installed(ctx, r, []catalog.Tool{nvm})["nvm"]
	}

	p := &Plan{}
	for _, t := range tools {
		if !installed[t.ID] {
			continue
		}
		var item *Item
		checked := false
		switch {
		case t.Method == catalog.MethodBrew:
			checked = true
			item = fromBrew(t, brew.Formulae, "brew upgrade "+t.Package)
		case t.Method == catalog.MethodCask:
			checked = true
			item = fromBrew(t, brew.Casks, "brew upgrade --cask "+t.Package)
		case t.ID == "node" && nvmReady:
			checked = true
			var err error
			if item, err = nodeUpgrade(ctx, r, t); err != nil {
				return nil, err
			}
		}
		switch {
		case item != nil:
			p.Items = append(p.Items, *item)
		case checked:
			p.UpToDate++
		}
	}
	return p, nil
}

func fromBrew(t catalog.Tool, outdated []outdatedPackage, command string) *Item {
	for _, o := range outdated {
		if o.Name != t.Package {
			continue
		}
		it := &Item{Tool: t, From: strings.Join(o.InstalledVersions, ","), To: o.CurrentVersion, Commands: []string{command}}
		if o.Pinned {
			it.Skip = "pinned"
		}
		return it
	}
	return nil
}

func nodeUpgrade(ctx context.Context, r runner.Runner, t catalog.Tool) (*Item, error) {
	var out bytes.Buffer
	if err := r.Run(ctx, shellenv.Prelude+nodeVersions, &out); err != nil {
		return nil, fmt.Errorf("check Node.js: %w", err)
	}
	lines := strings.Fields(out.String())
	if len(lines) != 2 || lines[1] == "N/A" {
		return nil, fmt.Errorf("check Node.js: unexpected nvm output %q", strings.TrimSpace(out.String()))
	}
	if lines[0] == lines[1] {
		return nil, nil
	}
	from := lines[0]
	if from == "N/A" {
		from = "none"
	}
	return &Item{Tool: t, From: from, To: lines[1], Commands: nodeCommands}, nil
}

// Print writes the dry-run view of the plan.
func (p *Plan) Print(out io.Writer) error {
	var w bytes.Buffer
	fmt.Fprintf(&w, "Upgrades: %d available, %d up to date\n", len(p.Items), p.UpToDate)
	if len(p.Items) > 0 {
		w.WriteString("\n")
		var table bytes.Buffer
		tw := tabwriter.NewWriter(&table, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(tw, "  TOOL\tFROM\tTO\tCOMMANDS"); err != nil {
			return err
		}
		for _, it := range p.Items {
			detail := strings.Join(it.Commands, " && ")
			if it.Skip != "" {
				detail = "(" + it.Skip + ", skipped)"
			}
			if _, err := fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", it.Tool.Name, it.From, it.To, detail); err != nil {
				return err
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		for _, line := range strings.SplitAfter(table.String(), "\n") {
			w.WriteString(strings.TrimRight(line, " \n"))
			if strings.HasSuffix(line, "\n") {
				w.WriteString("\n")
			}
		}
	}
	_, err := out.Write(w.Bytes())
	return err
}

// Execute upgrades the runnable items in order. A failed tool does not stop
// the rest. Items that are listed but skipped are reported as skipped.
func Execute(ctx context.Context, p *Plan, r runner.Runner, log io.Writer, emit func(plan.Event)) plan.Report {
	if log == nil {
		log = io.Discard
	}
	if emit == nil {
		emit = func(plan.Event) {}
	}
	var report plan.Report
	for _, it := range p.Items {
		res := plan.Result{ID: it.Tool.ID, Name: it.Tool.Name}
		if it.Skip != "" {
			res.Status = plan.StatusSkipped
			res.Err = fmt.Errorf("%s", it.Skip)
			emit(plan.Event{Kind: plan.Finished, Result: res})
			report.Results = append(report.Results, res)
			continue
		}
		emit(plan.Event{Kind: plan.Started, Result: res})
		_, _ = fmt.Fprintf(log, "==> Upgrading %s...\n", it.Tool.Name)
		script := shellenv.Prelude + strings.Join(it.Commands, " &&\n")
		if err := r.Run(ctx, script, log); err != nil {
			res.Status, res.Err = plan.StatusFailed, err
		} else {
			res.Status = plan.StatusUpgraded
		}
		emit(plan.Event{Kind: plan.Finished, Result: res})
		report.Results = append(report.Results, res)
	}
	return report
}

// NextSteps lists the follow-ups for what the report says was upgraded.
func NextSteps(p *Plan, r plan.Report) []string {
	upgraded := map[string]bool{}
	for _, res := range r.Results {
		if res.Status == plan.StatusUpgraded {
			upgraded[res.ID] = true
		}
	}
	if len(upgraded) == 0 {
		return nil
	}
	var steps []string
	for _, it := range p.Items {
		if upgraded[it.Tool.ID] && it.Tool.ID == "node" && it.From != "none" {
			steps = append(steps, fmt.Sprintf("Node.js: %s is still installed; free the space with: nvm uninstall %s", it.From, it.From))
		}
	}
	return append(steps, plan.GenericNextStep)
}
