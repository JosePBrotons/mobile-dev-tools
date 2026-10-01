package plan

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

var update = flag.Bool("update", false, "rewrite golden files")

func load(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ids(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Tool.ID)
	}
	return out
}

func profileIDs(c *catalog.Catalog, profile string) []string {
	var out []string
	for _, t := range c.ForProfile(profile) {
		out = append(out, t.ID)
	}
	return out
}

func TestResolve(t *testing.T) {
	c := load(t)
	tests := []struct {
		only    []string
		want    []string
		reasons map[string]string
	}{
		{[]string{"typescript"}, []string{"nvm", "node", "typescript"},
			map[string]string{"nvm": "required by Node.js LTS", "node": "required by TypeScript", "typescript": "selected"}},
		{[]string{"cocoapods"}, []string{"xcode-clt", "homebrew", "cocoapods"},
			map[string]string{"xcode-clt": "required by Homebrew", "homebrew": "required by CocoaPods"}},
		{[]string{"xcode", "homebrew"}, []string{"xcode-clt", "homebrew", "mas", "xcode"},
			map[string]string{"homebrew": "selected", "mas": "required by Xcode"}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.only, ","), func(t *testing.T) {
			items, err := Resolve(c, tt.only)
			if err != nil {
				t.Fatal(err)
			}
			if got := ids(items); !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for _, it := range items {
				if want, ok := tt.reasons[it.Tool.ID]; ok && it.Reason() != want {
					t.Errorf("%s: reason %q, want %q", it.Tool.ID, it.Reason(), want)
				}
			}
		})
	}
}

func TestResolveUnknown(t *testing.T) {
	if _, err := Resolve(load(t), []string{"node", "nope"}); err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("got %v, want unknown tool error", err)
	}
}

// Every dependency must come before the tools that need it, for every profile.
func TestResolveOrdersDependenciesFirst(t *testing.T) {
	c := load(t)
	for _, p := range c.Profiles {
		items, err := Resolve(c, profileIDs(c, p.ID))
		if err != nil {
			t.Fatal(err)
		}
		pos := map[string]int{}
		for i, it := range items {
			pos[it.Tool.ID] = i
		}
		for _, it := range items {
			for _, r := range it.Tool.Requires {
				if pos[r] >= pos[it.Tool.ID] {
					t.Errorf("%s: %s comes after %s", p.ID, r, it.Tool.ID)
				}
			}
		}
	}
}

func TestCommands(t *testing.T) {
	tests := []struct {
		tool catalog.Tool
		want []string
	}{
		{catalog.Tool{Method: catalog.MethodBrew, Package: "git"}, []string{"brew install git"}},
		{catalog.Tool{Method: catalog.MethodCask, Package: "zulu@17"}, []string{"brew install --cask zulu@17"}},
		{catalog.Tool{Method: catalog.MethodMas, Package: "497799835"}, []string{"mas install 497799835"}},
		{catalog.Tool{Method: catalog.MethodScript, Install: []string{"a", "b"}}, []string{"a", "b"}},
		{catalog.Tool{Method: catalog.MethodBrew, Package: "mkcert", PostInstall: []string{"mkcert-install"}},
			[]string{"brew install mkcert", "mkcert -install"}},
		{catalog.Tool{Method: catalog.MethodCask, Package: "android-studio", PostInstall: []string{"android-home-env"}},
			[]string{"brew install --cask android-studio"}},
	}
	for _, tt := range tests {
		if got := Commands(tt.tool); !slices.Equal(got, tt.want) {
			t.Errorf("Commands(%+v) = %v, want %v", tt.tool, got, tt.want)
		}
	}
}

func TestEveryKnownHookIsImplemented(t *testing.T) {
	for _, h := range catalog.KnownPostInstall {
		if hk, ok := hooks[h]; !ok || (hk.command == "" && len(hk.lines) == 0) {
			t.Errorf("post_install hook %q has no implementation", h)
		}
	}
}

func TestPrintGolden(t *testing.T) {
	c := load(t)
	// Pretend the system tools and git are already there.
	fake := &runner.Fake{Fail: map[string]error{}}
	for _, tool := range c.Tools {
		if tool.ID != "xcode-clt" && tool.ID != "homebrew" && tool.ID != "git" {
			fake.Fail[tool.Check] = errors.New("exit 1")
		}
	}
	p, err := New(context.Background(), c, profileIDs(c, "rn"), fake)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := p.Print(&buf); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "rn.golden")
	if *update {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != string(want) {
		t.Fatalf("dry run output changed (go test ./internal/plan -update to accept):\n%s", buf.String())
	}
}

// Legacy entry scripts per profile. Remove with the scripts in Phase 6.
var legacyScripts = map[string]string{
	"rn":      "React Native/react-native-dev-tools.sh",
	"flutter": "Flutter/flutter-dev-tools.sh",
	"java":    "Java/java-dev-tools.sh",
	"web":     "Web/web-dev-tools.sh",
}

var (
	sourceLine = regexp.MustCompile(`(?m)^\s*source\s+(.+)$`)
	pkgLine    = regexp.MustCompile(`(?m)^\s*(brew_formula|brew_cask|mas install)\s+(\S+)\s*$`)
)

// readLegacy returns a script and every script it sources, concatenated.
func readLegacy(t *testing.T, rel string, seen map[string]bool) string {
	t.Helper()
	if seen[rel] {
		return ""
	}
	seen[rel] = true
	data, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, m := range sourceLine.FindAllStringSubmatch(text, -1) {
		path := strings.Trim(strings.ReplaceAll(strings.TrimSpace(m[1]), `\ `, " "), `"`)
		text += readLegacy(t, strings.TrimPrefix(path, "./"), seen)
	}
	return text
}

// Exit criteria for Phase 2: mdt installs what each legacy script installs.
func TestCommandsMatchLegacyScripts(t *testing.T) {
	c := load(t)
	kind := map[string]catalog.Method{
		"brew_formula": catalog.MethodBrew, "brew_cask": catalog.MethodCask, "mas install": catalog.MethodMas,
	}
	for profile, script := range legacyScripts {
		t.Run(profile, func(t *testing.T) {
			text := readLegacy(t, script, map[string]bool{})
			var want []string
			for _, m := range pkgLine.FindAllStringSubmatch(text, -1) {
				want = append(want, string(kind[m[1]])+":"+m[2])
			}
			items, err := Resolve(c, profileIDs(c, profile))
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, it := range items {
				tool := it.Tool
				if tool.Method == catalog.MethodScript {
					continue
				}
				got = append(got, string(tool.Method)+":"+tool.Package)
			}
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("packages differ:\n got %v\nwant %v", got, want)
			}
			// Script installs and hook commands must appear verbatim in the legacy scripts.
			for _, it := range items {
				for _, cmd := range Commands(it.Tool) {
					if it.Tool.Method == catalog.MethodScript || !strings.HasPrefix(cmd, "brew install") && !strings.HasPrefix(cmd, "mas install") {
						if !strings.Contains(text, cmd) {
							t.Errorf("%s: %q is not in the legacy script", it.Tool.ID, cmd)
						}
					}
				}
			}
		})
	}
}
