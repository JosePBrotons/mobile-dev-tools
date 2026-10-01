package catalog

import (
	"slices"
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

// ids the legacy scripts install, per profile. Update together with the scripts.
var legacy = map[string][]string{
	"rn": {"xcode-clt", "homebrew", "git", "gh", "iterm2", "postman", "vscode", "nvm", "node", "pnpm",
		"watchman", "cocoapods", "jdk", "android-studio", "mas", "xcode", "fastlane", "typescript", "ngrok"},
	"flutter": {"xcode-clt", "homebrew", "git", "gh", "iterm2", "postman", "vscode", "cocoapods", "jdk",
		"android-studio", "flutter", "mas", "xcode", "fastlane"},
	"java": {"xcode-clt", "homebrew", "git", "gh", "jdk", "intellij-idea", "maven"},
	"web": {"xcode-clt", "homebrew", "git", "gh", "iterm2", "postman", "vscode", "nvm", "node", "pnpm", "bun",
		"mkcert", "orbstack", "firefox-dev", "brave", "ungoogled-chromium", "typescript", "ngrok"},
}

func TestProfilesMatchLegacyScripts(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for profile, want := range legacy {
		t.Run(profile, func(t *testing.T) {
			var got []string
			for _, tool := range c.ForProfile(profile) {
				got = append(got, tool.ID)
			}
			slices.Sort(got)
			want := slices.Sorted(slices.Values(want))
			if !slices.Equal(got, want) {
				t.Fatalf("profile %s:\n got %v\nwant %v", profile, got, want)
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
