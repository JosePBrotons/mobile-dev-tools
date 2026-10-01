// Package detect reports which catalog tools are already installed.
package detect

import (
	"context"
	"io"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// Installed runs each tool's check and returns the ids whose check passed.
// Tools without a check are reported as not installed.
func Installed(ctx context.Context, r runner.Runner, tools []catalog.Tool) map[string]bool {
	out := map[string]bool{}
	for _, t := range tools {
		if t.Check == "" {
			continue
		}
		if err := r.Run(ctx, shellenv.Prelude+t.Check, io.Discard); err == nil {
			out[t.ID] = true
		}
	}
	return out
}
