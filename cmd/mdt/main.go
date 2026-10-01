// Command mdt installs mobile, web and Java development tools on macOS.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

const usage = `Usage:
  mdt install (--profile <id> | --only <id,id,...>) [--dry-run] [--yes]
  mdt list

Run "mdt list" to see profiles and tool ids.
`

// LogDir is where install logs go, relative to the home folder.
const LogDir = "Library/Logs/mobile-dev-tools"

// errUsage marks errors caused by bad arguments (exit code 2).
var errUsage = errors.New("usage")

type app struct {
	runner runner.Runner
	home   string
	goos   string
	now    func() time.Time
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdt:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	a := &app{
		runner: runner.Exec{Stdin: os.Stdin},
		home:   home,
		goos:   runtime.GOOS,
		now:    time.Now,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	code := a.run(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}

// run executes a subcommand and returns the exit code.
func (a *app) run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		say(a.stderr, "%s", usage)
		return 2
	}
	var err error
	switch args[0] {
	case "install":
		err = a.install(ctx, args[1:])
	case "list":
		err = a.list()
	case "help", "-h", "--help":
		say(a.stdout, "%s", usage)
		return 0
	default:
		err = fmt.Errorf("%w: unknown command %q", errUsage, args[0])
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errUsage):
		say(a.stderr, "mdt: %v\n\n%s", err, usage)
		return 2
	default:
		say(a.stderr, "mdt: %v\n", err)
		return 1
	}
}

type installFlags struct {
	profile string
	only    []string
	dryRun  bool
	yes     bool
}

func parseInstall(args []string, c *catalog.Catalog) (installFlags, error) {
	var f installFlags
	var only string
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&f.profile, "profile", "", "profile to install")
	fs.StringVar(&only, "only", "", "comma separated tool ids")
	fs.BoolVar(&f.dryRun, "dry-run", false, "print the plan without installing")
	fs.BoolVar(&f.yes, "yes", false, "do not ask for confirmation")
	if err := fs.Parse(args); err != nil {
		return f, fmt.Errorf("%w: %v", errUsage, err)
	}
	if fs.NArg() > 0 {
		return f, fmt.Errorf("%w: unexpected argument %q", errUsage, fs.Arg(0))
	}
	for _, id := range strings.Split(only, ",") {
		if id = strings.TrimSpace(id); id != "" {
			f.only = append(f.only, id)
		}
	}
	if (f.profile == "") == (len(f.only) == 0) {
		return f, fmt.Errorf("%w: use exactly one of --profile or --only", errUsage)
	}
	if f.profile != "" && !slices.ContainsFunc(c.Profiles, func(p catalog.Profile) bool { return p.ID == f.profile }) {
		return f, fmt.Errorf("%w: unknown profile %q", errUsage, f.profile)
	}
	for _, id := range f.only {
		if _, ok := c.ByID(id); !ok {
			return f, fmt.Errorf("%w: unknown tool %q", errUsage, id)
		}
	}
	return f, nil
}

func (a *app) install(ctx context.Context, args []string) error {
	c, err := catalog.Load()
	if err != nil {
		return err
	}
	f, err := parseInstall(args, c)
	if err != nil {
		return err
	}
	if !f.dryRun && a.goos != "darwin" {
		return fmt.Errorf("installs run on macOS only; use --dry-run to preview the plan")
	}

	ids := f.only
	if f.profile != "" {
		for _, t := range c.ForProfile(f.profile) {
			ids = append(ids, t.ID)
		}
	}
	say(a.stderr, "Checking installed tools...\n")
	p, err := plan.New(ctx, c, ids, a.runner)
	if err != nil {
		return err
	}
	if err := p.Print(a.stdout); err != nil {
		return err
	}
	if f.dryRun {
		return nil
	}
	if len(p.Pending()) > 0 && !f.yes && !a.confirm() {
		return errors.New("aborted")
	}

	logPath, logFile, err := a.openLog()
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()
	say(a.stdout, "\nLogging to %s\n", logPath)

	out := io.MultiWriter(a.stdout, logFile)
	opts := plan.Options{Home: a.home, Log: out}
	report := plan.Execute(ctx, p, a.runner, opts, func(e plan.Event) { printEvent(out, e) })
	a.printSummary(report, logPath)
	if report.Failed() {
		return errors.New("some tools failed to install")
	}
	return nil
}

func (a *app) confirm() bool {
	say(a.stdout, "\nProceed? [y/N] ")
	line, _ := bufio.NewReader(a.stdin).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func (a *app) openLog() (string, *os.File, error) {
	dir := filepath.Join(a.home, LogDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "mdt-"+a.now().Format("20060102-150405")+".log")
	f, err := os.Create(path)
	return path, f, err
}

// printEvent writes one line per finished tool. Started events need no line:
// Execute already logs "==> Installing X...".
func printEvent(w io.Writer, e plan.Event) {
	if e.Kind != plan.Finished {
		return
	}
	r := e.Result
	if r.Err != nil {
		say(w, "[%s] %s: %v\n", r.Name, r.Status, r.Err)
		return
	}
	say(w, "[%s] %s\n", r.Name, r.Status)
}

func (a *app) printSummary(r plan.Report, logPath string) {
	order := []plan.Status{plan.StatusInstalled, plan.StatusPresent, plan.StatusSkipped, plan.StatusFailed}
	byStatus := map[plan.Status][]string{}
	for _, res := range r.Results {
		name := res.Name
		if res.Err != nil {
			name += " (" + res.Err.Error() + ")"
		}
		byStatus[res.Status] = append(byStatus[res.Status], name)
	}
	say(a.stdout, "\nSummary\n")
	for _, s := range order {
		if names := byStatus[s]; len(names) > 0 {
			say(a.stdout, "  %s (%d): %s\n", s, len(names), strings.Join(names, ", "))
		}
	}
	say(a.stdout, "\nLog: %s\n", logPath)
	if len(byStatus[plan.StatusInstalled]) > 0 {
		say(a.stdout, "Open a new terminal so PATH and profile changes take effect.\n")
	}
}

func (a *app) list() error {
	c, err := catalog.Load()
	if err != nil {
		return err
	}
	var w strings.Builder
	w.WriteString("Profiles:\n")
	for _, p := range c.Profiles {
		fmt.Fprintf(&w, "  %-10s %s\n", p.ID, p.Name)
	}
	tw := tabwriter.NewWriter(&w, 0, 0, 2, ' ', 0)
	for _, cat := range c.Categories {
		var rows []string
		for _, t := range c.Tools {
			if t.Category == cat.ID {
				rows = append(rows, fmt.Sprintf("  %s\t%s\t%s", t.ID, t.Name, strings.Join(t.Profiles, ",")))
			}
		}
		if len(rows) == 0 {
			continue
		}
		rows = append([]string{"\n" + cat.Name + ":"}, rows...)
		for _, row := range rows {
			if _, err := fmt.Fprintln(tw, row); err != nil {
				return err
			}
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err = io.WriteString(a.stdout, w.String())
	return err
}

// say writes to the terminal. A failed terminal write is not worth an error.
func say(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
