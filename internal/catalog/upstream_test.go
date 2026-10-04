package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
)

var upstream = flag.Bool("upstream", false, "check catalog packages against Homebrew (needs brew and network)")

type brewInfo struct {
	Formulae []brewPackage `json:"formulae"`
	Casks    []brewPackage `json:"casks"`
}

type brewPackage struct {
	Deprecated        bool   `json:"deprecated"`
	Disabled          bool   `json:"disabled"`
	DeprecationReason string `json:"deprecation_reason"`
	DisableReason     string `json:"disable_reason"`
}

// TestUpstream is part of the catalog refresh checklist (docs/catalog-refresh.md).
// It asks Homebrew about every brew and cask package and fails on packages
// that no longer exist or are deprecated or disabled. It is off by default
// because it needs Homebrew and the network.
func TestUpstream(t *testing.T) {
	if !*upstream {
		t.Skip("run with -upstream to check packages against Homebrew")
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	r := runner.Exec{}
	if err := r.Run(context.Background(), shellenv.Prelude+"command -v brew", &bytes.Buffer{}); err != nil {
		t.Skip("brew is not installed")
	}
	for _, tool := range c.Tools {
		var kind string
		switch tool.Method {
		case MethodBrew:
			kind = "--formula"
		case MethodCask:
			kind = "--cask"
		default:
			continue
		}
		t.Run(tool.ID, func(t *testing.T) {
			var out bytes.Buffer
			script := shellenv.Prelude + "brew info --json=v2 " + kind + " " + tool.Package + " 2>/dev/null"
			if err := r.Run(context.Background(), script, &out); err != nil {
				t.Fatalf("%s %s not found in Homebrew: %v", tool.Method, tool.Package, err)
			}
			var info brewInfo
			if err := json.Unmarshal(out.Bytes(), &info); err != nil {
				t.Fatal(err)
			}
			pkgs := append(info.Formulae, info.Casks...)
			if len(pkgs) != 1 {
				t.Fatalf("brew info returned %d packages for %s", len(pkgs), tool.Package)
			}
			if p := pkgs[0]; p.Disabled {
				t.Errorf("%s is disabled: %s", tool.Package, p.DisableReason)
			} else if p.Deprecated {
				t.Errorf("%s is deprecated: %s", tool.Package, p.DeprecationReason)
			}
		})
	}
}
