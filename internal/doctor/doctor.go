// Package doctor checks that installed tools are reachable from a new
// terminal and that the profile edits mdt makes are in place. It only
// reads; fixes are printed for the user to run.
package doctor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/detect"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// Options configures Run.
type Options struct {
	// Home is the user's home folder.
	Home string
	// Arch is runtime.GOARCH; it picks the Homebrew prefix in fixes.
	Arch string
	// IDs limits the check to these tools and their dependencies, and
	// reports missing ones as problems. Nil checks installed tools only.
	IDs []string
}

// Finding is one line of the report.
type Finding struct {
	Name   string
	OK     bool
	Detail string
	// Fix is a command or instruction, set when the finding is a problem.
	Fix string
}

// Report is the outcome of Run.
type Report struct {
	Tools []Finding
	Env   []Finding
	// Missing lists the ids of requested tools that are not installed.
	Missing []string
}

// Problems counts the findings that are not OK.
func (r Report) Problems() int {
	n := 0
	for _, f := range slices.Concat(r.Tools, r.Env) {
		if !f.OK {
			n++
		}
	}
	return n
}

// Run checks the selected tools and their environment setup.
func Run(ctx context.Context, c *catalog.Catalog, r runner.Runner, opts Options) (Report, error) {
	var tools []catalog.Tool
	if opts.IDs == nil {
		tools = c.Tools
	} else {
		items, err := plan.Resolve(c, opts.IDs)
		if err != nil {
			return Report{}, err
		}
		for _, it := range items {
			tools = append(tools, it.Tool)
		}
	}
	installed := detect.Installed(ctx, r, tools)

	var rep Report
	for _, t := range tools {
		switch {
		case installed[t.ID]:
			rep.Tools = append(rep.Tools, Finding{Name: t.Name, OK: true})
		case opts.IDs != nil:
			rep.Tools = append(rep.Tools, Finding{Name: t.Name, Detail: "not installed"})
			rep.Missing = append(rep.Missing, t.ID)
		}
	}

	zprofile := filepath.Join(opts.Home, shellenv.ProfileFile)
	zshrc := filepath.Join(opts.Home, ".zshrc")
	for _, t := range tools {
		if !installed[t.ID] {
			continue
		}
		var f []Finding
		var err error
		switch t.ID {
		case "homebrew":
			f, err = checkBrew(zprofile, opts.Arch)
		case "nvm":
			f, err = checkNVM(zprofile, zshrc)
		}
		if err != nil {
			return rep, err
		}
		rep.Env = append(rep.Env, f...)
		// mkcert-install has no cheap read-only check, so it is not verified.
		if slices.Contains(t.PostInstall, "android-home-env") {
			f, err = checkAndroid(opts.Home, zprofile)
			if err != nil {
				return rep, err
			}
			rep.Env = append(rep.Env, f...)
		}
	}
	return rep, nil
}

func checkBrew(zprofile, arch string) ([]Finding, error) {
	ok, err := shellenv.Contains(zprofile, "brew shellenv")
	if err != nil {
		return nil, err
	}
	f := Finding{Name: "Homebrew is loaded by ~/" + shellenv.ProfileFile, OK: ok}
	if !ok {
		prefix := "/usr/local"
		if arch == "arm64" {
			prefix = "/opt/homebrew"
		}
		f.Fix = fmt.Sprintf(`echo 'eval "$(%s/bin/brew shellenv)"' >> ~/%s`, prefix, shellenv.ProfileFile)
	}
	return []Finding{f}, nil
}

func checkNVM(files ...string) ([]Finding, error) {
	f := Finding{Name: "nvm is loaded by the shell profile"}
	for _, p := range files {
		ok, err := shellenv.Contains(p, "NVM_DIR")
		if err != nil {
			return nil, err
		}
		if ok {
			f.OK = true
		}
	}
	if !f.OK {
		f.Fix = `add 'export NVM_DIR="$HOME/.nvm"' and '[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"' to ~/.zshrc`
	}
	return []Finding{f}, nil
}

func checkAndroid(home, zprofile string) ([]Finding, error) {
	// mdt install only adds these lines when the first one is absent, so a
	// profile with older lines needs the missing ones appended by hand.
	lines := Finding{Name: "ANDROID_HOME is set in ~/" + shellenv.ProfileFile, OK: true}
	var fixes []string
	for _, l := range shellenv.AndroidHomeLines {
		ok, err := shellenv.HasLine(zprofile, l)
		if err != nil {
			return nil, err
		}
		if !ok {
			fixes = append(fixes, fmt.Sprintf("echo '%s' >> ~/%s", l, shellenv.ProfileFile))
		}
	}
	if len(fixes) > 0 {
		lines.OK = false
		lines.Fix = strings.Join(fixes, "\n           ")
	}
	sdk := Finding{Name: "Android SDK folder exists", OK: true}
	if _, err := os.Stat(filepath.Join(home, "Library", "Android", "sdk")); err != nil {
		sdk.OK = false
		sdk.Fix = "open Android Studio once to finish the setup wizard"
	}
	return []Finding{lines, sdk}, nil
}

// Print writes the report as plain text.
func (r Report) Print(out io.Writer) error {
	var b strings.Builder
	section := func(title string, fs []Finding) {
		if len(fs) == 0 {
			return
		}
		fmt.Fprintf(&b, "%s:\n", title)
		for _, f := range fs {
			status := "ok"
			if !f.OK {
				status = "problem"
				if title == "Tools" {
					status = "missing"
				}
			}
			name := f.Name
			if f.Detail != "" {
				name += " (" + f.Detail + ")"
			}
			fmt.Fprintf(&b, "  %-8s %s\n", status, name)
			if f.Fix != "" {
				fmt.Fprintf(&b, "           fix: %s\n", f.Fix)
			}
		}
	}
	section("Tools", r.Tools)
	section("Environment", r.Env)
	if len(r.Missing) > 0 {
		fmt.Fprintf(&b, "\nInstall missing tools: mdt install --only %s\n", strings.Join(r.Missing, ","))
	}
	if n := r.Problems(); n > 0 {
		fmt.Fprintf(&b, "\n%d problems found.\n", n)
	} else {
		b.WriteString("\nNo problems found.\n")
	}
	_, err := io.WriteString(out, b.String())
	return err
}
