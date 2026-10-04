package runner

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestExec(t *testing.T) {
	tests := []struct {
		script   string
		wantOut  string
		wantCode int
	}{
		{"echo hi", "hi\n", 0},
		{"echo oops >&2; exit 3", "oops\n", 3},
	}
	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			var out strings.Builder
			err := Exec{}.Run(context.Background(), tt.script, &out)
			code := 0
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				code = exitErr.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			if code != tt.wantCode || out.String() != tt.wantOut {
				t.Fatalf("got code %d out %q, want %d %q", code, out.String(), tt.wantCode, tt.wantOut)
			}
		})
	}
}

func TestFake(t *testing.T) {
	boom := errors.New("boom")
	f := &Fake{Fail: map[string]error{"bad": boom}}
	if err := f.Run(context.Background(), "good", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.Run(context.Background(), "a bad one", nil); !errors.Is(err, boom) {
		t.Fatalf("got %v, want boom", err)
	}
	if got := f.Calls(); len(got) != 2 || got[1] != "a bad one" {
		t.Fatalf("calls = %v", got)
	}
}

func TestFakeOutput(t *testing.T) {
	f := &Fake{Output: map[string]string{"sw_vers": "15.1\n"}}
	var out strings.Builder
	if err := f.Run(context.Background(), "sw_vers -productVersion", &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "15.1\n" {
		t.Fatalf("out = %q", out.String())
	}
}

func TestExecStdin(t *testing.T) {
	var out strings.Builder
	if err := (Exec{Stdin: strings.NewReader("typed\n")}).Run(context.Background(), "read x; echo got $x", &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "got typed\n" {
		t.Fatalf("out = %q", out.String())
	}
}
