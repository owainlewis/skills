// Package agents defines the registry of known agents and where each keeps its
// skills. Every agent has a global directory (under $HOME) and a project
// directory (relative to the current repo). The built-in set can be extended or
// overridden from the manifest's [agents] table.
package agents

import (
	"path/filepath"
	"sort"

	"github.com/owainlewis/skills/internal/pathx"
)

// Scope selects which of an agent's directories to use.
const (
	Global  = "global"
	Project = "project"
)

// Agent is one target: a global dir (with ~) and a project-relative dir.
type Agent struct {
	Name    string `toml:"-"`
	Global  string `toml:"global"`
	Project string `toml:"project"`
}

// Dir resolves the agent's install directory for the given scope. projectRoot is
// the base for project scope (typically the current working directory).
func (a Agent) Dir(scope, projectRoot string) string {
	if scope == Project {
		return filepath.Join(projectRoot, a.Project)
	}
	return pathx.ExpandHome(a.Global)
}

// DefaultTargets are pre-selected in the interactive picker and used when an
// entry names no targets.
var DefaultTargets = []string{"agents", "claude", "hermes"}

// builtin is the known-agent registry, seeded from real-world locations.
func builtin() map[string]Agent {
	return map[string]Agent{
		"claude": {Global: "~/.claude/skills", Project: ".claude/skills"},
		"codex":  {Global: "~/.codex/skills", Project: ".codex/skills"},
		"hermes": {Global: "~/.hermes/skills", Project: ".hermes/skills"},
		"pi":     {Global: "~/.pi/agent/skills", Project: ".pi/agent/skills"},
		"agents": {Global: "~/.agents/skills", Project: ".agents/skills"},
	}
}

// Registry is the resolved set of agents (built-ins merged with overrides).
type Registry struct {
	m map[string]Agent
}

// New builds a registry from the built-ins, applying per-field overrides and
// additions from the manifest's [agents] table.
func New(overrides map[string]Agent) *Registry {
	m := builtin()
	for name, ov := range overrides {
		base := m[name] // zero value if new
		if ov.Global != "" {
			base.Global = ov.Global
		}
		if ov.Project != "" {
			base.Project = ov.Project
		}
		m[name] = base
	}
	return &Registry{m: m}
}

// Get returns the named agent.
func (r *Registry) Get(name string) (Agent, bool) {
	a, ok := r.m[name]
	a.Name = name
	return a, ok
}

// Names returns all known agent names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.m))
	for n := range r.m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
