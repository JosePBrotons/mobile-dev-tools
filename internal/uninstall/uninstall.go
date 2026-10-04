// Package uninstall removes installed catalog tools. It refuses what it
// cannot undo safely: tools that others still need, system tools and tools
// without an automatic removal.
package uninstall

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/detect"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// Item is one requested tool.
type Item struct {
	Tool     catalog.Tool
	Commands []string
	// Refused says why the tool is not removed. Empty means it runs.
	Refused string
}

// Plan lists the requested tools, dependents first.
type Plan struct {
	Items []Item
}

// Runnable returns the items that will be removed.
func (p *Plan) Runnable() []Item {
	var out []Item
	for _, it := range p.Items {
		if it.Refused == "" {
			out = append(out, it)
		}
	}
	return out
}

// Check decides what happens to each id. A tool is refused when another
// installed tool still requires it, unless that tool is removed too.
func Check(ctx context.Context, c *catalog.Catalog, r runner.Runner, ids []string) (*Plan, error) {
	for _, id := range ids {
		if _, ok := c.ByID(id); !ok {
			return nil, fmt.Errorf("unknown tool %q (run mdt list)", id)
		}
	}
	// Resolve orders dependencies first and adds the ones that were not
	// asked for; reversed and filtered, it gives the removal order.
	resolved, err := plan.Resolve(c, ids)
	if err != nil {
		return nil, err
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}

	// Tools that need a selected tool, directly or not, decide whether it
	// may go, so their state is needed too.
	relevant := map[string]bool{}
	for id := range selected {
		relevant[id] = true
	}
	for changed := true; changed; {
		changed = false
		for _, t := range c.Tools {
			if relevant[t.ID] {
				continue
			}
			if slices.ContainsFunc(t.Requires, func(r string) bool { return relevant[r] }) {
				relevant[t.ID] = true
				changed = true
			}
		}
	}
	var tools []catalog.Tool
	for _, t := range c.Tools {
		if relevant[t.ID] {
			tools = append(tools, t)
		}
	}
	installed := detect.Installed(ctx, r, tools)

	// staying holds the installed tools that will still be there.
	staying := map[string]catalog.Tool{}
	for _, t := range tools {
		if installed[t.ID] && !selected[t.ID] {
			staying[t.ID] = t
		}
	}

	p := &Plan{}
	for _, it := range slices.Backward(resolved) {
		t := it.Tool
		if !selected[t.ID] {
			continue
		}
		item := Item{Tool: t}
		switch {
		case !installed[t.ID]:
			item.Refused = "not installed"
		case blockedBy(t, staying) != "":
			item.Refused = "needed by " + blockedBy(t, staying)
		case t.Method == catalog.MethodMas:
			item.Refused = "remove it by hand (see " + t.URL + ")"
		case t.Method == catalog.MethodScript && len(t.Uninstall) == 0:
			item.Refused = "no automatic uninstall, remove it by hand (see " + t.URL + ")"
		default:
			item.Commands = commands(t)
		}
		if item.Refused != "" && installed[t.ID] {
			staying[t.ID] = t
		}
		p.Items = append(p.Items, item)
	}
	return p, nil
}

// blockedBy names a staying tool that requires t, or "".
func blockedBy(t catalog.Tool, staying map[string]catalog.Tool) string {
	var names []string
	for _, s := range staying {
		if slices.Contains(s.Requires, t.ID) {
			names = append(names, s.Name)
		}
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

func commands(t catalog.Tool) []string {
	cmds := plan.UndoCommands(t)
	switch t.Method {
	case catalog.MethodBrew:
		cmds = append(cmds, "brew uninstall "+t.Package)
	case catalog.MethodCask:
		cmds = append(cmds, "brew uninstall --cask "+t.Package)
	case catalog.MethodScript:
		cmds = append(cmds, t.Uninstall...)
	}
	return cmds
}

// Print writes the dry-run view of the plan.
func (p *Plan) Print(out io.Writer) error {
	var w bytes.Buffer
	fmt.Fprintf(&w, "Uninstall: %d of %d tools\n\n", len(p.Runnable()), len(p.Items))
	var table bytes.Buffer
	tw := tabwriter.NewWriter(&table, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "  STATUS\tTOOL\tCOMMANDS"); err != nil {
		return err
	}
	for _, it := range p.Items {
		status, detail := "remove", strings.Join(it.Commands, " && ")
		if it.Refused != "" {
			status, detail = "refused", it.Refused
		}
		if _, err := fmt.Fprintf(tw, "  %s\t%s\t%s\n", status, it.Tool.Name, detail); err != nil {
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
	_, err := out.Write(w.Bytes())
	return err
}

// Execute removes the runnable items in order. A failed tool does not stop
// the rest, but the tools it requires are kept, so nothing is pulled out
// from under it.
func Execute(ctx context.Context, p *Plan, r runner.Runner, log io.Writer, emit func(plan.Event)) plan.Report {
	if log == nil {
		log = io.Discard
	}
	if emit == nil {
		emit = func(plan.Event) {}
	}
	var report plan.Report
	kept := map[string]string{} // id -> name of the failed tool that needs it
	for _, it := range p.Items {
		res := plan.Result{ID: it.Tool.ID, Name: it.Tool.Name}
		switch {
		case it.Refused != "":
			res.Status, res.Err = plan.StatusSkipped, fmt.Errorf("%s", it.Refused)
		case kept[it.Tool.ID] != "":
			res.Status, res.Err = plan.StatusSkipped, fmt.Errorf("needed by %s, which failed", kept[it.Tool.ID])
		}
		if res.Status == plan.StatusSkipped {
			emit(plan.Event{Kind: plan.Finished, Result: res})
			report.Results = append(report.Results, res)
			continue
		}
		emit(plan.Event{Kind: plan.Started, Result: res})
		_, _ = fmt.Fprintf(log, "==> Uninstalling %s...\n", it.Tool.Name)
		script := shellenv.Prelude + strings.Join(it.Commands, " &&\n")
		if err := r.Run(ctx, script, log); err != nil {
			res.Status, res.Err = plan.StatusFailed, err
			for _, dep := range it.Tool.Requires {
				kept[dep] = it.Tool.Name
			}
		} else {
			res.Status = plan.StatusRemoved
		}
		emit(plan.Event{Kind: plan.Finished, Result: res})
		report.Results = append(report.Results, res)
	}
	return report
}

// NextSteps lists the manual follow-ups for what the report says was removed.
func NextSteps(p *Plan, r plan.Report) []string {
	removed := map[string]bool{}
	for _, res := range r.Results {
		if res.Status == plan.StatusRemoved {
			removed[res.ID] = true
		}
	}
	if len(removed) == 0 {
		return nil
	}
	var steps []string
	for _, it := range p.Items {
		if !removed[it.Tool.ID] {
			continue
		}
		if hooks := plan.ProfileHooks(it.Tool); len(hooks) > 0 {
			steps = append(steps, fmt.Sprintf("%s: remove the lines added to ~/%s by %s", it.Tool.Name, shellenv.ProfileFile, strings.Join(hooks, ", ")))
		}
	}
	return append(steps, plan.GenericNextStep)
}
