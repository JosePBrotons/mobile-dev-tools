// Package runner is the only place in mdt that executes commands.
// Everything else takes a Runner so tests can use Fake.
package runner

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// Runner runs a bash script and streams its output to out.
type Runner interface {
	Run(ctx context.Context, script string, out io.Writer) error
}

// Exec runs scripts with /bin/bash. Stdin is inherited so sudo and
// installer prompts reach the user.
type Exec struct{}

// Run implements Runner.
func (Exec) Run(ctx context.Context, script string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
