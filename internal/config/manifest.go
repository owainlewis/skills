// Package config reads and writes the desired-state skills.toml and the
// machine-managed lockfile.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/pathx"
)

// Entry is a single desired skill in the manifest.
type Entry struct {
	Source  string   `toml:"source"`
	Path    string   `toml:"path,omitempty"`
	Ref     string   `toml:"ref,omitempty"`
	Skills  []string `toml:"skills,omitempty"`  // subset of discovered skills; empty = all
	Targets []string `toml:"targets,omitempty"` // agent names; empty = DefaultTargets
	Scope   string   `toml:"scope,omitempty"`   // "global" | "project"
}

// Manifest is the parsed skills.toml (desired state).
type Manifest struct {
	DefaultTargets []string                `toml:"default_targets,omitempty"`
	Agents         map[string]agents.Agent `toml:"agents,omitempty"`
	Dir            string                  `toml:"dir,omitempty"` // legacy single-dir escape hatch
	Skills         []Entry                 `toml:"skill,omitempty"`

	path string // source file, for Save
}

// Targets returns the manifest's default target agents, falling back to the
// built-in defaults.
func (m *Manifest) Targets() []string {
	if len(m.DefaultTargets) > 0 {
		return m.DefaultTargets
	}
	return agents.DefaultTargets
}

// Load reads the manifest at path. A missing file yields an empty manifest
// (not an error) so commands can operate before `init`.
func Load(path string) (*Manifest, error) {
	m := &Manifest{path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	m.path = path
	return m, nil
}

// Save writes the manifest back to its source path atomically.
func (m *Manifest) Save() error {
	if m.path == "" {
		return fmt.Errorf("manifest has no path")
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	if err := toml.NewEncoder(&b).Encode(m); err != nil {
		return err
	}
	return writeFileAtomic(m.path, []byte(b.String()))
}

// ResolvedDir returns the legacy single install directory with ~ expanded, or
// "" when unset (the agent-target model is used instead).
func (m *Manifest) ResolvedDir() string {
	if m.Dir == "" {
		return ""
	}
	return pathx.ExpandHome(m.Dir)
}

// Upsert adds or replaces an entry keyed by (source, path), returning whether
// an existing entry was replaced.
func (m *Manifest) Upsert(e Entry) (replaced bool) {
	for i, existing := range m.Skills {
		if existing.Source == e.Source && existing.Path == e.Path {
			m.Skills[i] = e
			return true
		}
	}
	m.Skills = append(m.Skills, e)
	return false
}

// Path returns the manifest's source path.
func (m *Manifest) Path() string { return m.path }

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
