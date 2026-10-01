package detect

import (
	"context"
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

func TestInstalled(t *testing.T) {
	tools := []catalog.Tool{
		{ID: "git", Check: "brew list --formula git"},
		{ID: "pod", Check: "command -v pod"},
		{ID: "nocheck"},
	}
	fake := &runner.Fake{Fail: map[string]error{"command -v pod": errors.New("exit 1")}}
	got := Installed(context.Background(), fake, tools)
	want := map[string]bool{"git": true}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	calls := fake.Calls()
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2: %v", len(calls), calls)
	}
	for _, c := range calls {
		if !strings.HasPrefix(c, shellenv.Prelude) {
			t.Fatalf("check ran without the prelude: %q", c)
		}
	}
}
