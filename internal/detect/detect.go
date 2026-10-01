// Package detect reports which catalog tools are already installed.
package detect

import (
	"context"
	"io"
	"sync"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

// workers bounds how many checks run at once. Checks are read-only.
const workers = 8

// Installed runs each tool's check and returns the ids whose check passed.
// Tools without a check are reported as not installed. The runner must be
// safe for concurrent use.
func Installed(ctx context.Context, r runner.Runner, tools []catalog.Tool) map[string]bool {
	out := map[string]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for _, t := range tools {
		if t.Check == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := r.Run(ctx, shellenv.Prelude+t.Check, io.Discard); err == nil {
				mu.Lock()
				out[t.ID] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return out
}
