// Package routing exposes the bundled routing instruction markdown injected
// by hook-sessionstart at the start of each Claude Code session.
package routing

import _ "embed"

//go:embed claude-code.md
var claudeCodeMD string

// RoutingInstructions returns the full bundled routing markdown for Claude Code.
func RoutingInstructions() string { return claudeCodeMD }
