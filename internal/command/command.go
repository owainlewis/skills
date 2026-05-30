// Package command implements the skills subcommands. Conventions across all
// commands: data goes to Out (stdout), logs to Err (stderr), prompts appear only
// in an interactive terminal, and --json yields machine-readable output.
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/config"
	"github.com/owainlewis/skills/internal/gitx"
	"github.com/owainlewis/skills/internal/installer"
	"github.com/owainlewis/skills/internal/skill"
	"github.com/owainlewis/skills/internal/source"
)

// Env carries shared configuration and I/O for a command invocation.
type Env struct {
	ConfigPath  string   // path to skills.toml
	ProjectRoot string   // base dir for project scope (usually the cwd)
	DirOverride string   // --dir / SKILLS_DIR; bypasses the agent model entirely
	AgentsFlag  []string // --agent; overrides target selection
	Scope       string   // --global / --project; "" means unspecified
	SkillFlag   []string // --skill; subset of discovered skills
	JSON        bool
	Yes         bool // --yes: accept defaults without prompting
	Out         io.Writer
	Err         io.Writer
}

// Result describes the outcome for one skill in one target, for logs and JSON.
type Result struct {
	Name   string `json:"name"`
	Agent  string `json:"agent,omitempty"`
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Commit string `json:"commit"`
	Dir    string `json:"dir,omitempty"`
	Status string `json:"status"` // installed | updated | unchanged | removed | missing
}

// target is a resolved agent + concrete directory.
type target struct {
	Agent string
	Dir   string
}

func (e *Env) logf(format string, a ...any) {
	fmt.Fprintf(e.Err, format+"\n", a...)
}

func (e *Env) emitJSON(v any) error {
	enc := json.NewEncoder(e.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (e *Env) loadManifest() (*config.Manifest, error) {
	return config.Load(e.ConfigPath)
}

// interactive reports whether we may prompt: a real terminal, no --json, no --yes.
func (e *Env) interactive() bool {
	return !e.JSON && !e.Yes && isTTY(os.Stdin) && isTTY(os.Stderr)
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// targetNames applies the selection precedence: --agent flag, else the entry's
// targets, else the manifest defaults.
func (e *Env) targetNames(m *config.Manifest, entry config.Entry) []string {
	switch {
	case len(e.AgentsFlag) > 0:
		return e.AgentsFlag
	case len(entry.Targets) > 0:
		return entry.Targets
	default:
		return m.Targets()
	}
}

// scopeFor applies the precedence: --global/--project flag, else the entry's
// scope. Returns "" when unspecified (caller prompts or errors).
func (e *Env) scopeFor(entry config.Entry) string {
	if e.Scope != "" {
		return e.Scope
	}
	return entry.Scope
}

// resolveTargets maps agent names to concrete directories. With --dir set, a
// single literal directory bypasses the agent model.
func (e *Env) resolveTargets(reg *agents.Registry, names []string, scope string) ([]target, error) {
	if e.DirOverride != "" {
		return []target{{Agent: "", Dir: e.DirOverride}}, nil
	}
	var out []target
	for _, n := range names {
		a, ok := reg.Get(n)
		if !ok {
			return nil, fmt.Errorf("unknown agent %q (run `skills agents` to list known agents)", n)
		}
		out = append(out, target{Agent: n, Dir: a.Dir(scope, e.ProjectRoot)})
	}
	return out, nil
}

// lockCache loads one lockfile per directory and saves them together at the end.
type lockCache struct {
	m map[string]*config.Lock
}

func newLockCache() *lockCache { return &lockCache{m: map[string]*config.Lock{}} }

func (c *lockCache) get(dir string) (*config.Lock, error) {
	if l, ok := c.m[dir]; ok {
		return l, nil
	}
	l, err := config.LoadLock(dir)
	if err != nil {
		return nil, err
	}
	c.m[dir] = l
	return l, nil
}

func (c *lockCache) saveAll() error {
	for _, l := range c.m {
		if err := l.Save(); err != nil {
			return err
		}
	}
	return nil
}

// clone clones an entry's source and discovers its skills. The returned cleanup
// removes the temporary checkout.
func clone(ctx context.Context, entry config.Entry) (commit string, found []skill.Found, cleanup func(), err error) {
	src, err := source.Parse(entry.Source, entry.Path)
	if err != nil {
		return "", nil, func() {}, err
	}
	co, err := gitx.Clone(ctx, src.GitURL, entry.Ref)
	if err != nil {
		return "", nil, func() {}, err
	}
	found, err = skill.Discover(co.Dir, src.Path)
	if err != nil {
		co.Cleanup()
		return "", nil, func() {}, fmt.Errorf("%s: %w", entry.Source, err)
	}
	return co.Commit, found, co.Cleanup, nil
}

// installToTargets installs the selected skills into every target directory and
// records each in that directory's lock. A nil filter installs all skills.
func installToTargets(entry config.Entry, found []skill.Found, filter []string, targets []target, commit string, locks *lockCache) ([]Result, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var results []Result
	for _, t := range targets {
		lock, err := locks.get(t.Dir)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			if filter != nil && !contains(filter, f.Name) {
				continue
			}
			status := "installed"
			if prev, had := lock.Get(f.Name); had {
				if prev.Commit == commit && installedOnDisk(t.Dir, f.Name) {
					status = "unchanged"
				} else {
					status = "updated"
				}
			}
			if status != "unchanged" {
				if err := installer.Install(t.Dir, f.Name, f.Dir); err != nil {
					return nil, fmt.Errorf("install %s: %w", f.Name, err)
				}
			}
			lock.Put(config.Installed{
				Name: f.Name, Source: entry.Source, Path: entry.Path,
				Ref: entry.Ref, Agent: t.Agent, Commit: commit, InstalledAt: now,
			})
			results = append(results, Result{
				Name: f.Name, Agent: t.Agent, Source: entry.Source, Path: entry.Path,
				Ref: entry.Ref, Commit: commit, Dir: t.Dir, Status: status,
			})
		}
	}
	return results, nil
}

func skillNames(found []skill.Found) []string {
	names := make([]string, len(found))
	for i, f := range found {
		names[i] = f.Name
	}
	return names
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func installedOnDisk(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.IsDir()
}

func logResults(e *Env, results []Result) {
	for _, r := range results {
		if r.Agent != "" {
			e.logf("%-10s %s -> %s (%s)", r.Status, r.Name, r.Agent, shortCommit(r.Commit))
		} else {
			e.logf("%-10s %s (%s)", r.Status, r.Name, shortCommit(r.Commit))
		}
	}
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}
