package tui

import (
	"github.com/josepbrotons/mobile-dev-tools/internal/catalog"
	"github.com/josepbrotons/mobile-dev-tools/internal/plan"
)

// Selection is the set of tools the user picked. It holds no install logic:
// dependencies and ordering come from internal/plan.
type Selection struct {
	c         *catalog.Catalog
	installed map[string]bool
	on        map[string]bool
}

// NewSelection preselects the tools of a profile that are not installed yet.
// An empty or unknown profile (Custom) preselects nothing.
func NewSelection(c *catalog.Catalog, installed map[string]bool, profile string) *Selection {
	s := &Selection{c: c, installed: installed, on: map[string]bool{}}
	if profile != "" {
		for _, t := range c.ForProfile(profile) {
			if !installed[t.ID] {
				s.on[t.ID] = true
			}
		}
	}
	return s
}

// Has reports whether the user selected id.
func (s *Selection) Has(id string) bool { return s.on[id] }

// Toggle flips id. Installed tools cannot be selected; it reports whether
// anything changed.
func (s *Selection) Toggle(id string) bool {
	if s.installed[id] {
		return false
	}
	s.on[id] = !s.on[id]
	if !s.on[id] {
		delete(s.on, id)
	}
	return true
}

// Set selects or deselects every id that is not installed.
func (s *Selection) Set(ids []string, on bool) {
	for _, id := range ids {
		if s.installed[id] {
			continue
		}
		if on {
			s.on[id] = true
		} else {
			delete(s.on, id)
		}
	}
}

// IDs returns the selected ids in catalog order.
func (s *Selection) IDs() []string {
	var out []string
	for _, t := range s.c.Tools {
		if s.on[t.ID] {
			out = append(out, t.ID)
		}
	}
	return out
}

// Plan resolves the selection, dependencies included.
func (s *Selection) Plan() (*plan.Plan, error) {
	return plan.Build(s.c, s.IDs(), s.installed)
}

// Required maps the id of every pending dependency to the reason it was
// added, for example "required by CocoaPods".
func Required(p *plan.Plan) map[string]string {
	out := map[string]string{}
	for _, it := range p.Items {
		if it.RequiredBy != "" && !it.Installed {
			out[it.Tool.ID] = it.Reason()
		}
	}
	return out
}
