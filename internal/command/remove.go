package command

import (
	"context"
	"fmt"

	"github.com/owainlewis/skills/internal/config"
	"github.com/owainlewis/skills/internal/installer"
)

// Remove deletes an installed skill by name: it removes the installed directory,
// the lock entry, and the manifest entry that produced it (matched by source +
// path). A missing skill is an error.
func Remove(ctx context.Context, e *Env, name string) error {
	if name == "" {
		return fmt.Errorf("remove: skill name required")
	}
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	dir := e.resolveDir(m)
	lock, err := config.LoadLock(dir)
	if err != nil {
		return err
	}

	in, ok := lock.Get(name)
	if !ok {
		return fmt.Errorf("not installed: %s", name)
	}
	if err := installer.Remove(dir, name); err != nil {
		return err
	}
	lock.Remove(name)
	if err := lock.Save(); err != nil {
		return err
	}

	// Drop the matching manifest entry, if present.
	for i, entry := range m.Skills {
		if entry.Source == in.Source && entry.Path == in.Path {
			m.Skills = append(m.Skills[:i], m.Skills[i+1:]...)
			if err := m.Save(); err != nil {
				return err
			}
			break
		}
	}

	e.logf("removed %s", name)
	if e.JSON {
		return e.emitJSON([]Result{{Name: name, Source: in.Source, Commit: in.Commit, Status: "removed"}})
	}
	return nil
}
