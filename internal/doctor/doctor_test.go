package doctor

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
	"github.com/josepbrotons/mobile-dev-tools/internal/shellenv"
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

// run checks with a fake home. Checks containing a key of missing fail.
func run(t *testing.T, home string, ids []string, missing ...string) Report {
	t.Helper()
	fake := &runner.Fake{Fail: map[string]error{}}
	for _, m := range missing {
		fake.Fail[m] = errors.New("exit 1")
	}
	rep, err := Run(context.Background(), load(t), fake, Options{Home: home, IDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func write(t *testing.T, home, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func envFinding(rep Report, prefix string) (Finding, bool) {
	for _, f := range rep.Env {
		if strings.HasPrefix(f.Name, prefix) {
			return f, true
		}
	}
	return Finding{}, false
}

func TestDefaultModeHidesMissingTools(t *testing.T) {
	rep := run(t, t.TempDir(), nil, "command -v pod")
	for _, f := range rep.Tools {
		if !f.OK {
			t.Fatalf("default mode reported %q", f.Name)
		}
		if f.Name == "CocoaPods" {
			t.Fatal("missing CocoaPods listed")
		}
	}
	if len(rep.Missing) != 0 {
		t.Fatalf("Missing = %v", rep.Missing)
	}
}

func TestSelectedModeReportsMissing(t *testing.T) {
	rep := run(t, t.TempDir(), []string{"cocoapods"}, "command -v pod")
	if len(rep.Missing) != 1 || rep.Missing[0] != "cocoapods" {
		t.Fatalf("Missing = %v", rep.Missing)
	}
	if rep.Problems() == 0 {
		t.Fatal("a missing selected tool must be a problem")
	}
	var buf bytes.Buffer
	if err := rep.Print(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "mdt install --only cocoapods") {
		t.Fatalf("no install hint:\n%s", buf.String())
	}
}

func TestBrewShellenv(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantOK  bool
		wantFix string
	}{
		{"present", `eval "$(/opt/homebrew/bin/brew shellenv)"` + "\n", true, ""},
		{"missing", "", false, "/opt/homebrew/bin/brew shellenv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			write(t, home, shellenv.ProfileFile, tt.profile)
			rep := run(t, home, []string{"homebrew"})
			f, ok := envFinding(rep, "Homebrew")
			if !ok || f.OK != tt.wantOK || !strings.Contains(f.Fix, tt.wantFix) {
				t.Fatalf("finding %+v, want ok=%v fix containing %q", f, tt.wantOK, tt.wantFix)
			}
		})
	}
}

func TestNVMLoaded(t *testing.T) {
	tests := []struct {
		name, file string
		wantOK     bool
	}{
		{"zshrc", ".zshrc", true},
		{"zprofile", shellenv.ProfileFile, true},
		{"nowhere", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			if tt.file != "" {
				write(t, home, tt.file, `export NVM_DIR="$HOME/.nvm"`+"\n")
			}
			f, ok := envFinding(run(t, home, []string{"nvm"}), "nvm")
			if !ok || f.OK != tt.wantOK {
				t.Fatalf("finding %+v, want ok=%v", f, tt.wantOK)
			}
		})
	}
}

func TestAndroid(t *testing.T) {
	home := t.TempDir()
	write(t, home, shellenv.ProfileFile, shellenv.AndroidHomeLines[0]+"\n")
	rep := run(t, home, []string{"android-studio"})
	lines, _ := envFinding(rep, "ANDROID_HOME")
	sdk, _ := envFinding(rep, "Android SDK")
	if lines.OK || strings.Contains(lines.Fix, shellenv.AndroidHomeLines[0]) || !strings.Contains(lines.Fix, shellenv.AndroidHomeLines[1]) {
		t.Fatalf("partial lines: %+v", lines)
	}
	if sdk.OK {
		t.Fatalf("sdk folder is missing but reported ok: %+v", sdk)
	}

	write(t, home, shellenv.ProfileFile, strings.Join(shellenv.AndroidHomeLines, "\n")+"\n")
	if err := os.MkdirAll(filepath.Join(home, "Library", "Android", "sdk"), 0o755); err != nil {
		t.Fatal(err)
	}
	rep = run(t, home, []string{"android-studio"})
	for _, f := range rep.Env {
		if strings.HasPrefix(f.Name, "ANDROID") || strings.HasPrefix(f.Name, "Android") {
			if !f.OK {
				t.Fatalf("expected ok: %+v", f)
			}
		}
	}
}

func TestEnvChecksSkippedForMissingTool(t *testing.T) {
	rep := run(t, t.TempDir(), []string{"android-studio"}, "Android Studio.app")
	if _, ok := envFinding(rep, "ANDROID_HOME"); ok {
		t.Fatal("android env checked although Android Studio is missing")
	}
}

func TestUnknownTool(t *testing.T) {
	_, err := Run(context.Background(), load(t), &runner.Fake{}, Options{IDs: []string{"nope"}})
	if err == nil {
		t.Fatal("want an error for an unknown id")
	}
}

func TestPrintGolden(t *testing.T) {
	home := t.TempDir()
	write(t, home, shellenv.ProfileFile, shellenv.AndroidHomeLines[0]+"\n")
	rep := run(t, home, []string{"android-studio", "cocoapods", "nvm"}, "command -v pod")
	var buf bytes.Buffer
	if err := rep.Print(&buf); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "doctor.golden")
	if *update {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("output changed (go test ./internal/doctor -update to accept):\n%s", buf.String())
	}
}

func TestMkcertTrusted(t *testing.T) {
	tests := []struct {
		name    string
		missing []string
		wantOK  bool
		wantFix string
	}{
		{"trusted", nil, true, ""},
		{"not trusted", []string{"find-certificate"}, false, "mkcert -install"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := run(t, t.TempDir(), []string{"mkcert"}, tt.missing...)
			f, ok := envFinding(rep, "mkcert")
			if !ok || f.OK != tt.wantOK || f.Fix != tt.wantFix {
				t.Fatalf("finding %+v, want ok=%v fix %q", f, tt.wantOK, tt.wantFix)
			}
		})
	}
}
