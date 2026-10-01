// Package sysinfo reports basic facts about the machine for the welcome screen.
package sysinfo

import (
	"context"
	"strconv"
	"strings"

	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

// Info describes the machine. Fields are empty or zero when a check failed.
type Info struct {
	MacOS string // for example "15.1"
	Arch  string // "arm64" or "x86_64"
	// Rosetta is true when the process runs translated on Apple silicon.
	Rosetta bool
	// FreeGB is the free space on / in gigabytes, rounded down.
	FreeGB int
}

// Check runs the probes through r. A failed probe leaves its field empty.
func Check(ctx context.Context, r runner.Runner) Info {
	var info Info
	info.MacOS = run(ctx, r, "sw_vers -productVersion")
	info.Arch = run(ctx, r, "uname -m")
	info.Rosetta = run(ctx, r, "sysctl -n sysctl.proc_translated") == "1"
	info.FreeGB = parseFreeGB(run(ctx, r, "df -Pk /"))
	return info
}

func run(ctx context.Context, r runner.Runner, script string) string {
	var sb strings.Builder
	if err := r.Run(ctx, script, &sb); err != nil {
		return ""
	}
	return strings.TrimSpace(sb.String())
}

// parseFreeGB reads the Available column of POSIX df output (1024 byte blocks).
func parseFreeGB(out string) int {
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return 0
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return 0
	}
	kb, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return 0
	}
	return int(kb / (1024 * 1024))
}
