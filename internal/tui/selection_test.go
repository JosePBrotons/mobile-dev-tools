package tui

import (
	"slices"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
)

func loadCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewSelection(t *testing.T) {
	c := loadCatalog(t)
	tests := []struct {
		name      string
		profile   string
		installed map[string]bool
		wantOn    []string
		wantOff   []string
	}{
		{"rn preselects its tools", "rn", nil, []string{"node", "cocoapods"}, []string{"bun"}},
		{"installed tools stay off", "rn", map[string]bool{"git": true}, []string{"node"}, []string{"git"}},
		{"custom selects nothing", customID, nil, nil, []string{"node", "git"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSelection(c, tt.installed, tt.profile)
			for _, id := range tt.wantOn {
				if !s.Has(id) {
					t.Errorf("%s should be selected", id)
				}
			}
			for _, id := range tt.wantOff {
				if s.Has(id) {
					t.Errorf("%s should not be selected", id)
				}
			}
		})
	}
}

func TestSelectionToggleAndRequired(t *testing.T) {
	c := loadCatalog(t)
	s := NewSelection(c, map[string]bool{"git": true}, customID)

	if s.Toggle("git") {
		t.Fatal("installed tools cannot be toggled")
	}
	if !s.Toggle("cocoapods") || !s.Has("cocoapods") {
		t.Fatal("cocoapods should be selected")
	}
	p, err := s.Plan()
	if err != nil {
		t.Fatal(err)
	}
	req := Required(p)
	if req["homebrew"] != "required by CocoaPods" {
		t.Fatalf("required = %v", req)
	}
	if _, ok := req["cocoapods"]; ok {
		t.Fatal("a selected tool is not a dependency")
	}
	if got := s.IDs(); !slices.Equal(got, []string{"cocoapods"}) {
		t.Fatalf("ids = %v", got)
	}
	s.Toggle("cocoapods")
	if len(s.IDs()) != 0 {
		t.Fatalf("ids = %v", s.IDs())
	}
}
