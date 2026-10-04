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
	// Output maps a substring to text written to out by scripts that
	// contain it.
	Output map[string]string

	mu    sync.Mutex
	calls []string
}

// Run implements Runner.
func (f *Fake) Run(_ context.Context, script string, out io.Writer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, script)
	if out != nil {
		for substr, text := range f.Output {
			if strings.Contains(script, substr) {
				_, _ = io.WriteString(out, text)
			}
		}
	}
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
