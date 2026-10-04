// Package catalog loads and validates the declarative tool catalog.
package catalog

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v3"

	embedded "github.com/josepbrotons/mobile-dev-tools/catalog"
)

// Method says how a tool is installed.
type Method string

const (
	MethodBrew   Method = "brew"
	MethodCask   Method = "cask"
	MethodMas    Method = "mas"
	MethodScript Method = "script"
)

// Category groups tools in the UI.
type Category struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Profile is a preselection of tools.
type Profile struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Tool is one installable item.
type Tool struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category"`
	Profiles    []string `yaml:"profiles"`
	Method      Method   `yaml:"method"`
	Package     string   `yaml:"package"`
	Install     []string `yaml:"install"`
	Check       string   `yaml:"check"`
	Requires    []string `yaml:"requires"`
	PostInstall []string `yaml:"post_install"`
	Notes       []string `yaml:"notes"`
	// NextSteps are manual follow-ups shown on the summary after the tool
	// was installed.
	NextSteps []string `yaml:"next_steps"`
	URL       string   `yaml:"url"`
}

// Catalog is the whole parsed file.
type Catalog struct {
	Categories []Category `yaml:"categories"`
	Profiles   []Profile  `yaml:"profiles"`
	Tools      []Tool     `yaml:"tools"`
}

// Load parses and validates the catalog embedded in the binary.
func Load() (*Catalog, error) {
	return Parse(embedded.ToolsYAML)
}

// Parse decodes data strictly (unknown fields are errors) and validates it.
func Parse(data []byte) (*Catalog, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var c Catalog
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// ByID returns the tool with the given id.
func (c *Catalog) ByID(id string) (Tool, bool) {
	for _, t := range c.Tools {
		if t.ID == id {
			return t, true
		}
	}
	return Tool{}, false
}

// ForProfile returns the tools preselected by a profile, in file order.
func (c *Catalog) ForProfile(profile string) []Tool {
	var out []Tool
	for _, t := range c.Tools {
		for _, p := range t.Profiles {
			if p == profile {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
