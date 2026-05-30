package command

import (
	"context"
	"fmt"

	"github.com/owainlewis/skills/internal/manifest"
)

// Update re-resolves installed skills to the latest commit for their recorded
// ref and reinstalls. With no names it updates everything in the lock. Unlike
// Sync it neither adds new manifest entries nor prunes; it refreshes what is
// already installed.
func Update(ctx context.Context, e *Env, names []string) error {
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	dir := e.resolveDir(m)
	lock, err := manifest.LoadLock(dir)
	if err != nil {
		return err
	}

	work := lock.Skills
	if len(names) > 0 {
		want := map[string]bool{}
		for _, n := range names {
			want[n] = true
		}
		var filtered []manifest.Installed
		for _, in := range lock.Skills {
			if want[in.Name] {
				filtered = append(filtered, in)
				delete(want, in.Name)
			}
		}
		for n := range want {
			e.logf("error: not installed: %s", n)
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no matching installed skills")
		}
		work = filtered
	}

	var all []Result
	var failed int
	for _, in := range work {
		entry := manifest.Entry{Source: in.Source, Path: in.Path, Ref: in.Ref}
		results, err := installEntry(ctx, e, dir, entry, lock)
		if err != nil {
			e.logf("error: %s: %v", in.Name, err)
			failed++
			continue
		}
		for _, r := range results {
			all = append(all, r)
			e.logf("%-10s %s (%s)", r.Status, r.Name, shortCommit(r.Commit))
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
		return fmt.Errorf("%d skill(s) failed to update", failed)
	}
	return nil
}
