package catalog

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// KnownPostInstall lists the hooks internal/plan implements.
var KnownPostInstall = []string{"android-home-env", "mkcert-install"}

// Validate reports every problem found, joined into one error.
func (c *Catalog) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	categories := map[string]bool{}
	for _, cat := range c.Categories {
		if !idPattern.MatchString(cat.ID) || cat.Name == "" {
			add("category %q: needs a kebab-case id and a name", cat.ID)
		}
		if categories[cat.ID] {
			add("category %q: duplicate id", cat.ID)
		}
		categories[cat.ID] = true
	}
	profiles := map[string]bool{}
	for _, p := range c.Profiles {
		if !idPattern.MatchString(p.ID) || p.Name == "" {
			add("profile %q: needs a kebab-case id and a name", p.ID)
		}
		if profiles[p.ID] {
			add("profile %q: duplicate id", p.ID)
		}
		profiles[p.ID] = true
	}

	ids := map[string]bool{}
	for _, t := range c.Tools {
		if ids[t.ID] {
			add("tool %q: duplicate id", t.ID)
		}
		ids[t.ID] = true
	}

	for _, t := range c.Tools {
		if !idPattern.MatchString(t.ID) {
			add("tool %q: id must be kebab-case", t.ID)
		}
		if t.Name == "" || t.Description == "" {
			add("tool %q: name and description are required", t.ID)
		}
		if !categories[t.Category] {
			add("tool %q: unknown category %q", t.ID, t.Category)
		}
		for _, p := range t.Profiles {
			if !profiles[p] {
				add("tool %q: unknown profile %q", t.ID, p)
			}
		}
		switch t.Method {
		case MethodBrew, MethodCask, MethodMas:
			if t.Package == "" {
				add("tool %q: method %s needs a package", t.ID, t.Method)
			}
		case MethodScript:
			if len(t.Install) == 0 {
				add("tool %q: method script needs install commands", t.ID)
			}
		default:
			add("tool %q: unknown method %q", t.ID, t.Method)
		}
		for _, r := range t.Requires {
			if !ids[r] {
				add("tool %q: requires unknown tool %q", t.ID, r)
			}
		}
		for _, h := range t.PostInstall {
			if !slices.Contains(KnownPostInstall, h) {
				add("tool %q: unknown post_install hook %q", t.ID, h)
			}
		}
	}

	errs = append(errs, c.cycles()...)
	return errors.Join(errs...)
}

// cycles finds dependency cycles with a depth-first search.
func (c *Catalog) cycles() []error {
	deps := map[string][]string{}
	for _, t := range c.Tools {
		deps[t.ID] = t.Requires
	}
	const (
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var errs []error
	var visit func(id string, path []string)
	visit = func(id string, path []string) {
		switch state[id] {
		case done:
			return
		case visiting:
			errs = append(errs, fmt.Errorf("dependency cycle: %v -> %s", path, id))
			return
		}
		state[id] = visiting
		for _, d := range deps[id] {
			visit(d, append(path, id))
		}
		state[id] = done
	}
	for _, t := range c.Tools {
		visit(t.ID, nil)
	}
	return errs
}
