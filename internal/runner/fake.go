package runner

import (
	"context"
	"io"
	"strings"
	"sync"
)

// Fake records every script and fails those that contain a key of Fail.
// Scripts that match nothing succeed.
type Fake struct {
	Fail map[string]error

	mu    sync.Mutex
	calls []string
}

// Run implements Runner.
func (f *Fake) Run(_ context.Context, script string, _ io.Writer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, script)
	for substr, err := range f.Fail {
		if strings.Contains(script, substr) {
			return err
		}
	}
	return nil
}

// Calls returns the scripts run so far, in order.
func (f *Fake) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}
