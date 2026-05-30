package command

import (
	"context"
	"fmt"

	"github.com/owainlewis/skills/internal/installer"
	"github.com/owainlewis/skills/internal/manifest"
)

// Sync reconciles the installed set with the manifest: it (re)installs every
// manifest entry to its ref's latest commit and, when prune is set, removes
// installed skills no longer present in the manifest. Idempotent.
func Sync(ctx context.Context, e *Env, prune bool) error {
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	dir := e.resolveDir(m)
	lock, err := manifest.LoadLock(dir)
	if err != nil {
		return err
	}

	managed := map[string]bool{}
	var all []Result
	var failed int
	for _, entry := range m.Skills {
		results, err := installEntry(ctx, e, dir, entry, lock)
		if err != nil {
			e.logf("error: %s: %v", entry.Source, err)
			failed++
			continue
		}
		for _, r := range results {
			managed[r.Name] = true
			all = append(all, r)
			e.logf("%-10s %s (%s)", r.Status, r.Name, shortCommit(r.Commit))
		}
	}

	if prune {
		for _, in := range append([]manifest.Installed(nil), lock.Skills...) {
			if managed[in.Name] {
				continue
			}
			if err := installer.Remove(dir, in.Name); err != nil {
				e.logf("error: prune %s: %v", in.Name, err)
				failed++
				continue
			}
			lock.Remove(in.Name)
			all = append(all, Result{Name: in.Name, Source: in.Source, Commit: in.Commit, Status: "removed"})
			e.logf("%-10s %s", "removed", in.Name)
		}
	}

	if err := lock.Save(); err != nil {
		return err
	}
	if e.JSON {
		if err := e.emitJSON(all); err != nil {
			return err
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d skill(s) failed to sync", failed)
	}
	return nil
}
