package shellenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLines(t *testing.T) {
	lines := []string{"export A=1", "export B=2"}
	tests := []struct {
		name     string
		initial  *string
		want     string
		wantDiff bool
	}{
		{"missing file", nil, "export A=1\nexport B=2\n", true},
		{"empty file", ptr(""), "export A=1\nexport B=2\n", true},
		{"other content", ptr("alias ll='ls -l'\n"), "alias ll='ls -l'\n\nexport A=1\nexport B=2\n", true},
		{"already present", ptr("export A=1\n"), "export A=1\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".zprofile")
			if tt.initial != nil {
				if err := os.WriteFile(path, []byte(*tt.initial), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			changed, err := EnsureLines(path, lines)
			if err != nil {
				t.Fatal(err)
			}
			if changed != tt.wantDiff {
				t.Fatalf("changed = %v, want %v", changed, tt.wantDiff)
			}
			// A second call never changes the file.
			if again, err := EnsureLines(path, lines); err != nil || again {
				t.Fatalf("second call changed=%v err=%v", again, err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func TestContains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")
	if got, err := Contains(path, "NVM_DIR"); err != nil || got {
		t.Fatalf("missing file: got %v, err %v", got, err)
	}
	if err := os.WriteFile(path, []byte("export NVM_DIR=\"$HOME/.nvm\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		substr string
		want   bool
	}{{"NVM_DIR", true}, {"brew shellenv", false}}
	for _, tt := range tests {
		if got, err := Contains(path, tt.substr); err != nil || got != tt.want {
			t.Fatalf("Contains(%q) = %v, %v; want %v", tt.substr, got, err, tt.want)
		}
	}
}
