// Package plan turns a tool selection into ordered install steps, prints
// them for a dry run and executes them through a runner.Runner.
package plan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/detect"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// BrewUpdate runs once before the first Homebrew install, like Misc/brew-helpers.sh.
const BrewUpdate = "brew update"

// Item is one tool in a plan.
type Item struct {
	Tool catalog.Tool
	// RequiredBy names the tool that pulled this one in. Empty when the
	// user selected it.
	RequiredBy string
	Installed  bool
}

// Reason says why the tool is in the plan.
func (i Item) Reason() string {
	if i.RequiredBy == "" {
		return "selected"
	}
	return "required by " + i.RequiredBy
}

// Plan is an ordered list of tools, dependencies first.
type Plan struct {
	Items []Item
}

// hook is a post_install step. Commands run after a fresh install; profile
// edits are idempotent and also run when the tool was already installed.
type hook struct {
	command string
	lines   []string
}

var hooks = map[string]hook{
	"mkcert-install":   {command: "mkcert -install"},
	"android-home-env": {lines: shellenv.AndroidHomeLines},
}

// New resolves ids against the catalog and detects what is installed.
func New(ctx context.Context, c *catalog.Catalog, ids []string, r runner.Runner) (*Plan, error) {
	items, err := Resolve(c, ids)
	if err != nil {
		return nil, err
	}
	tools := make([]catalog.Tool, len(items))
	for i, it := range items {
		tools[i] = it.Tool
	}
	installed := detect.Installed(ctx, r, tools)
	for i := range items {
		items[i].Installed = installed[items[i].Tool.ID]
	}
	return &Plan{Items: prune(items)}, nil
}

// prune drops dependencies that no pending tool needs, for example nvm
// when Node.js is already installed some other way. Items are ordered
// dependencies first, so a reverse walk sees every dependent first.
func prune(items []Item) []Item {
	needed := map[string]bool{}
	keep := make([]bool, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		it := items[i]
		keep[i] = it.RequiredBy == "" || needed[it.Tool.ID]
		if keep[i] && !it.Installed {
			for _, r := range it.Tool.Requires {
				needed[r] = true
			}
		}
	}
	var out []Item
	for i, it := range items {
		if keep[i] {
			out = append(out, it)
		}
	}
	return out
}

// Resolve adds every required tool and orders the result so dependencies
// come first. Ties follow catalog file order, so output is deterministic.
func Resolve(c *catalog.Catalog, ids []string) ([]Item, error) {
	selected := map[string]bool{}
	for _, id := range ids {
		if _, ok := c.ByID(id); !ok {
			return nil, fmt.Errorf("unknown tool %q (run mdt list)", id)
		}
		selected[id] = true
	}

	// Walk requires breadth first from the selection, in catalog order,
	// remembering the first tool that pulled each dependency in.
	requiredBy := map[string]string{}
	needed := map[string]bool{}
	var queue []catalog.Tool
	for _, t := range c.Tools {
		if selected[t.ID] {
			needed[t.ID] = true
			queue = append(queue, t)
		}
	}
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		for _, r := range t.Requires {
			if needed[r] {
				continue
			}
			needed[r] = true
			requiredBy[r] = t.Name
			dep, _ := c.ByID(r)
			queue = append(queue, dep)
		}
	}

	var items []Item
	done := map[string]bool{}
	var visit func(t catalog.Tool)
	visit = func(t catalog.Tool) {
		if done[t.ID] {
			return
		}
		done[t.ID] = true
		for _, r := range t.Requires {
			dep, _ := c.ByID(r)
			visit(dep)
		}
		items = append(items, Item{Tool: t, RequiredBy: requiredBy[t.ID]})
	}
	for _, t := range c.Tools {
		if needed[t.ID] {
			visit(t)
		}
	}
	return items, nil
}

// Commands returns the shell lines that install t, including post_install
// commands.
func Commands(t catalog.Tool) []string {
	var cmds []string
	switch t.Method {
	case catalog.MethodBrew:
		cmds = append(cmds, "brew install "+t.Package)
	case catalog.MethodCask:
		cmds = append(cmds, "brew install --cask "+t.Package)
	case catalog.MethodMas:
		cmds = append(cmds, "mas install "+t.Package)
	case catalog.MethodScript:
		cmds = append(cmds, t.Install...)
	}
	for _, h := range t.PostInstall {
		if c := hooks[h].command; c != "" {
			cmds = append(cmds, c)
		}
	}
	return cmds
}

// profileEdits describes the profile changes a tool's hooks make.
func profileEdits(t catalog.Tool) []string {
	var out []string
	for _, h := range t.PostInstall {
		if lines := hooks[h].lines; len(lines) > 0 {
			out = append(out, fmt.Sprintf("add %q to ~/%s", lines[0], shellenv.ProfileFile))
		}
	}
	return out
}

func usesBrew(t catalog.Tool) bool {
	return t.Method == catalog.MethodBrew || t.Method == catalog.MethodCask
}

// Pending returns the items that will be installed.
func (p *Plan) Pending() []Item {
	var out []Item
	for _, it := range p.Items {
		if !it.Installed {
			out = append(out, it)
		}
	}
	return out
}

// Print writes the dry-run view of the plan.
func (p *Plan) Print(out io.Writer) error {
	var w bytes.Buffer
	pending := len(p.Pending())
	fmt.Fprintf(&w, "Plan: %d tools, %d to install, %d already installed\n\n",
		len(p.Items), pending, len(p.Items)-pending)

	var table bytes.Buffer
	tw := tabwriter.NewWriter(&table, 0, 0, 2, ' ', 0)
	rows := []string{"  STATUS\tTOOL\tCOMMANDS"}
	updated := false
	for _, it := range p.Items {
		status, cmds := "installed", profileEdits(it.Tool)
		if !it.Installed {
			status = "install"
			if usesBrew(it.Tool) && !updated {
				updated = true
				rows = append(rows, "  install\t(Homebrew update)\t"+BrewUpdate)
			}
			cmds = append(Commands(it.Tool), cmds...)
		}
		detail := strings.Join(cmds, " && ")
		if it.RequiredBy != "" {
			detail = strings.TrimSpace(detail + "  (" + it.Reason() + ")")
		}
		rows = append(rows, fmt.Sprintf("  %s\t%s\t%s", status, it.Tool.Name, detail))
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, row); err != nil {
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

	var notes []string
	for _, it := range p.Pending() {
		for _, n := range it.Tool.Notes {
			notes = append(notes, fmt.Sprintf("  %s: %s", it.Tool.Name, n))
		}
	}
	if len(notes) > 0 {
		fmt.Fprintf(&w, "\nNotes:\n%s\n", strings.Join(notes, "\n"))
	}
	_, err := out.Write(w.Bytes())
	return err
}
