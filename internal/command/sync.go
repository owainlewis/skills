package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/config"
	"github.com/owainlewis/skills/internal/installer"
)

// reconcileOpts tunes the shared sync/update engine.
type reconcileOpts struct {
	prune     bool     // remove installed skills absent from the manifest
	onlyNames []string // restrict to these skill names (update); empty = all
}

// reconcile (re)installs every manifest entry into its targets at the latest
// commit for its ref. It is the engine behind both sync and update.
func reconcile(ctx context.Context, e *Env, opts reconcileOpts) error {
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	reg := agents.New(m.Agents)
	locks := newLockCache()
	managed := map[string]map[string]bool{} // dir -> name -> kept
	touched := map[string]bool{}            // dirs we installed into
	var all []Result
	failed := 0

	for _, entry := range m.Skills {
		commit, found, cleanup, err := clone(ctx, entry)
		if err != nil {
			e.logf("error: %s: %v", entry.Source, err)
			failed++
			continue
		}

		scope := e.scopeFor(entry)
		if scope == "" {
			scope = agents.Global // batch default when never chosen
		}
		targets, err := e.resolveTargets(reg, e.targetNames(m, entry), scope)
		if err != nil {
			e.logf("error: %s: %v", entry.Source, err)
			failed++
			cleanup()
			continue
		}

		filter := effectiveFilter(entry.Skills, opts.onlyNames)
		results, err := installToTargets(entry, found, filter, targets, commit, locks)
		cleanup()
		if err != nil {
			e.logf("error: %s: %v", entry.Source, err)
			failed++
			continue
		}
		for _, r := range results {
			if managed[r.Dir] == nil {
				managed[r.Dir] = map[string]bool{}
			}
			managed[r.Dir][r.Name] = true
			touched[r.Dir] = true
		}
		all = append(all, results...)
		logResults(e, results)
	}

	if opts.prune {
		// Prune every dir we touched plus any dir that still carries our
		// lockfile from a previous sync — so dropping a target cleans it too.
		pruneDirs := map[string]bool{}
		for d := range touched {
			pruneDirs[d] = true
		}
		for _, d := range e.existingLockDirs(reg) {
			pruneDirs[d] = true
		}
		for dir := range pruneDirs {
			lock, err := locks.get(dir)
			if err != nil {
				return err
			}
			for _, in := range append([]config.Installed(nil), lock.Skills...) {
				if managed[dir][in.Name] {
					continue
				}
				if err := installer.Remove(dir, in.Name); err != nil {
					e.logf("error: prune %s: %v", in.Name, err)
					failed++
					continue
				}
				lock.Remove(in.Name)
				all = append(all, Result{Name: in.Name, Agent: in.Agent, Source: in.Source, Commit: in.Commit, Dir: dir, Status: "removed"})
				e.logf("%-10s %s -> %s", "removed", in.Name, in.Agent)
			}
		}
	}

	if err := locks.saveAll(); err != nil {
		return err
	}
	if e.JSON {
		if err := e.emitJSON(all); err != nil {
			return err
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d entr(ies) failed", failed)
	}
	return nil
}

// existingLockDirs returns directories that hold one of our lockfiles across all
// known agents (both scopes), so prune can reconcile targets that were dropped.
func (e *Env) existingLockDirs(reg *agents.Registry) []string {
	if e.DirOverride != "" {
		return []string{e.DirOverride}
	}
	var dirs []string
	for _, name := range reg.Names() {
		a, _ := reg.Get(name)
		for _, scope := range []string{agents.Global, agents.Project} {
			d := a.Dir(scope, e.ProjectRoot)
			if _, err := os.Stat(filepath.Join(d, config.LockName)); err == nil {
				dirs = append(dirs, d)
			}
		}
	}
	return dirs
}

// effectiveFilter combines an entry's skill subset with an update name filter.
// nil means "all skills in the source".
func effectiveFilter(entrySkills, onlyNames []string) []string {
	if len(onlyNames) == 0 {
		return entrySkills
	}
	if entrySkills == nil {
		return onlyNames
	}
	var out []string
	for _, n := range entrySkills {
		if contains(onlyNames, n) {
			out = append(out, n)
		}
	}
	return out
}

// Sync reconciles installed skills with the manifest. With prune it removes
// skills no longer declared. Idempotent.
func Sync(ctx context.Context, e *Env, prune bool) error {
	return reconcile(ctx, e, reconcileOpts{prune: prune})
}
