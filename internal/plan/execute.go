package plan

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// Status is the outcome of one tool.
type Status string

const (
	StatusInstalled Status = "installed"
	StatusPresent   Status = "already installed"
	StatusSkipped   Status = "skipped"
	StatusFailed    Status = "failed"
	// StatusDone is used for the Homebrew update, which installs nothing.
	StatusDone Status = "done"
)

// Result is the outcome of one tool.
type Result struct {
	ID, Name string
	Status   Status
	// Err is set for failed tools; for skipped tools it says which
	// dependency failed.
	Err error
}

// EventKind tells a Started event from a Finished one.
type EventKind int

const (
	Started EventKind = iota
	Finished
)

// Event reports progress. The CLI prints it; the Phase 3 TUI will render it.
// The Homebrew update uses ID "brew-update" and is not part of the Report.
type Event struct {
	Kind   EventKind
	Result Result
}

// Report lists every tool's outcome in plan order.
type Report struct {
	Results []Result
}

// Failed reports whether any tool failed.
func (r Report) Failed() bool {
	for _, res := range r.Results {
		if res.Status == StatusFailed {
			return true
		}
	}
	return false
}

// Options configures Execute.
type Options struct {
	// Home is the user's home folder; profile edits go to Home/.zprofile.
	Home string
	// Log receives every command's output. Nil discards it.
	Log io.Writer
}

// Execute installs the pending tools in order. A failed tool does not stop
// the rest; tools that depend on it are skipped.
func Execute(ctx context.Context, p *Plan, r runner.Runner, opts Options, emit func(Event)) Report {
	if opts.Log == nil {
		opts.Log = io.Discard
	}
	if emit == nil {
		emit = func(Event) {}
	}
	var report Report
	broken := map[string]string{} // id -> name of the failed tool behind it
	updated := false

	for _, it := range p.Items {
		t := it.Tool
		res := Result{ID: t.ID, Name: t.Name}

		for _, dep := range t.Requires {
			if name, ok := broken[dep]; ok {
				res.Status = StatusSkipped
				res.Err = fmt.Errorf("dependency %s failed", name)
				broken[t.ID] = name
				break
			}
		}
		if res.Status == StatusSkipped {
			emit(Event{Kind: Finished, Result: res})
			report.Results = append(report.Results, res)
			continue
		}

		if !it.Installed && usesBrew(t) && !updated {
			updated = true
			brewUpdate(ctx, r, opts.Log, emit)
		}
		emit(Event{Kind: Started, Result: res})
		var err error
		if it.Installed {
			res.Status = StatusPresent
		} else {
			logf(opts.Log, "==> Installing %s...\n", t.Name)
			script := shellenv.Prelude + strings.Join(Commands(t), " &&\n")
			err = r.Run(ctx, script, opts.Log)
			res.Status = StatusInstalled
		}
		if err == nil {
			err = applyEdits(it, opts)
		}
		if err != nil {
			res.Status = StatusFailed
			res.Err = err
			broken[t.ID] = t.Name
		}
		emit(Event{Kind: Finished, Result: res})
		report.Results = append(report.Results, res)
	}
	return report
}

// brewUpdate is best effort: a failure is logged and installs go on.
func brewUpdate(ctx context.Context, r runner.Runner, log io.Writer, emit func(Event)) {
	res := Result{ID: "brew-update", Name: "Homebrew update"}
	emit(Event{Kind: Started, Result: res})
	logf(log, "==> Updating Homebrew...\n")
	res.Status = StatusDone
	if err := r.Run(ctx, shellenv.Prelude+BrewUpdate, log); err != nil {
		res.Status, res.Err = StatusFailed, fmt.Errorf("%w (continuing)", err)
	}
	emit(Event{Kind: Finished, Result: res})
}

func applyEdits(it Item, opts Options) error {
	for _, h := range it.Tool.PostInstall {
		lines := hooks[h].lines
		if len(lines) == 0 {
			continue
		}
		path := filepath.Join(opts.Home, shellenv.ProfileFile)
		if skip := hooks[h].skipIf; skip != "" {
			done, err := shellenv.Contains(path, skip)
			if err != nil {
				return fmt.Errorf("%s: %w", h, err)
			}
			if done {
				continue
			}
		}
		changed, err := shellenv.EnsureLines(path, lines)
		if err != nil {
			return fmt.Errorf("%s: %w", h, err)
		}
		if changed {
			logf(opts.Log, "==> Added %s lines to %s\n", h, path)
		}
	}
	return nil
}

// logf writes to the log. Log write errors must not abort an install.
func logf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// GenericNextStep is shown whenever something was installed.
const GenericNextStep = "Open a new terminal so PATH and profile changes take effect."

// NextSteps lists the follow-ups for what the report says was installed:
// the catalog's next_steps in plan order, then the generic one.
func NextSteps(p *Plan, r Report) []string {
	installed := map[string]bool{}
	for _, res := range r.Results {
		if res.Status == StatusInstalled {
			installed[res.ID] = true
		}
	}
	if len(installed) == 0 {
		return nil
	}
	var steps []string
	for _, it := range p.Items {
		if installed[it.Tool.ID] {
			for _, s := range it.Tool.NextSteps {
				steps = append(steps, it.Tool.Name+": "+s)
			}
		}
	}
	return append(steps, GenericNextStep)
}
