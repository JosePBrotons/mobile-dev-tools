// Package runner is the only place in mdt that executes commands.
// Everything else takes a Runner so tests can use Fake.
package runner

import (
	"context"
	"io"
	"os/exec"
)

// Runner runs a bash script and streams its output to out.
type Runner interface {
	Run(ctx context.Context, script string, out io.Writer) error
}

// Exec runs scripts with /bin/bash. Stdin is passed to the script so sudo
// and installer prompts can reach the user. A nil Stdin means no input, which
// keeps a full-screen UI in charge of the terminal.
type Exec struct {
	Stdin io.Reader
}

// SudoValidate returns a command that asks for the admin password and caches
// it. It is a command rather than a Run call because it needs the terminal;
// a TUI hands it to tea.ExecProcess.
func SudoValidate() *exec.Cmd {
	return exec.Command("sudo", "-v")
}

// Run implements Runner.
func (e Exec) Run(ctx context.Context, script string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
	cmd.Stdin = e.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
