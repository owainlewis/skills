package command

import (
	"context"
	"fmt"

	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/config"
	"github.com/owainlewis/skills/internal/installer"
)

// Remove uninstalls a skill by name from every target directory it is installed
// in, and drops it from the manifest (removing entries whose only skill it was).
func Remove(ctx context.Context, e *Env, name string) error {
	if name == "" {
		return fmt.Errorf("remove: skill name required")
	}
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	reg := agents.New(m.Agents)
	dirs, err := e.allTargetDirs(m, reg)
	if err != nil {
		return err
	}

	locks := newLockCache()
	var results []Result
	for _, t := range dirs {
		lock, err := locks.get(t.Dir)
		if err != nil {
			return err
		}
		in, ok := lock.Get(name)
		if !ok {
			continue
		}
		if err := installer.Remove(t.Dir, name); err != nil {
			return err
		}
		lock.Remove(name)
		results = append(results, Result{Name: name, Agent: in.Agent, Source: in.Source, Commit: in.Commit, Dir: t.Dir, Status: "removed"})
	}
	if len(results) == 0 {
		return fmt.Errorf("not installed: %s", name)
	}
	if err := locks.saveAll(); err != nil {
		return err
	}

	// Manifest cleanup: drop the name from subset lists; remove entries whose
	// only declared skill it was.
	var kept []config.Entry
	for _, entry := range m.Skills {
		if len(entry.Skills) > 0 {
			if contains(entry.Skills, name) {
				entry.Skills = without(entry.Skills, name)
				if len(entry.Skills) == 0 {
					continue // entry no longer installs anything
				}
			}
		}
		kept = append(kept, entry)
	}
	m.Skills = kept
	if err := m.Save(); err != nil {
		return err
	}

	for _, r := range results {
		e.logf("removed %s -> %s", r.Name, r.Agent)
	}
	if e.JSON {
		return e.emitJSON(results)
	}
	return nil
}

// without returns ss with every occurrence of s removed.
func without(ss []string, s string) []string {
	out := ss[:0:0]
	for _, x := range ss {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}
