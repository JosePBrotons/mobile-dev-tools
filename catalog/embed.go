// Package catalog embeds the tool catalog so the binary needs no files at runtime.
package catalog

import _ "embed"

// ToolsYAML is the raw content of tools.yaml.
//
//go:embed tools.yaml
var ToolsYAML []byte
