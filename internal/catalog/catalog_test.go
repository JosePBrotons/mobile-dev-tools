package catalog

import (
	"strings"
	"testing"
)

const header = `
categories: [{id: system, name: System}]
profiles: [{id: rn, name: RN}]
`

func TestLoadEmbedded(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Tools) == 0 {
		t.Fatal("catalog is empty")
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		tools string
		want  string
	}{
		{"duplicate id", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a}
  - {id: a, name: A, description: d, category: system, method: brew, package: a}`, "duplicate id"},
		{"unknown category", `
tools:
  - {id: a, name: A, description: d, category: nope, method: brew, package: a}`, "unknown category"},
		{"unknown profile", `
tools:
  - {id: a, name: A, description: d, category: system, profiles: [x], method: brew, package: a}`, "unknown profile"},
		{"missing package", `
tools:
  - {id: a, name: A, description: d, category: system, method: cask}`, "needs a package"},
		{"missing install", `
tools:
  - {id: a, name: A, description: d, category: system, method: script}`, "needs install commands"},
		{"unknown method", `
tools:
  - {id: a, name: A, description: d, category: system, method: pip, package: a}`, "unknown method"},
		{"unknown require", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a, requires: [b]}`, "requires unknown tool"},
		{"unknown hook", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a, post_install: [x]}`, "unknown post_install hook"},
		{"uninstall on brew", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a, uninstall: [x]}`, "uninstall is only for method script"},
		{"interactive on brew", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a, interactive: true}`, "interactive is only for method script"},
		{"requires interactive", `
tools:
  - {id: a, name: A, description: d, category: system, method: script, install: [x], interactive: true}
  - {id: b, name: B, description: d, category: system, method: brew, package: b, requires: [a]}`, "cannot require interactive tool"},
		{"bad id", `
tools:
  - {id: Bad_ID, name: A, description: d, category: system, method: brew, package: a}`, "kebab-case"},
		{"cycle", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a, requires: [b]}
  - {id: b, name: B, description: d, category: system, method: brew, package: b, requires: [a]}`, "dependency cycle"},
		{"unknown field", `
tools:
  - {id: a, name: A, description: d, category: system, method: brew, package: a, typo: 1}`, "field typo not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(header + tt.tools))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestByID(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if tool, ok := c.ByID("cocoapods"); !ok || tool.Package != "cocoapods" {
		t.Fatalf("ByID(cocoapods) = %+v, %v", tool, ok)
	}
	if _, ok := c.ByID("missing"); ok {
		t.Fatal("ByID found a tool that does not exist")
	}
}
