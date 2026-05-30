package command

import (
	"context"
	"fmt"

	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/config"
)

// AddOpts configures Add.
type AddOpts struct {
	Source string
	Ref    string
	Path   string
	NoSync bool
}

// Add resolves a source, lets the user choose which skills, which agents, and
// the scope (interactively or via flags), installs into every chosen target,
// and records the selection in the manifest.
func Add(ctx context.Context, e *Env, opts AddOpts) error {
	if opts.Source == "" {
		return fmt.Errorf("add: source required")
	}
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	reg := agents.New(m.Agents)
	entry := config.Entry{Source: opts.Source, Path: opts.Path, Ref: opts.Ref}

	if opts.NoSync {
		entry.Targets = e.AgentsFlag
		entry.Scope = e.scopeFor(entry)
		entry.Skills = e.SkillFlag
		recordAndReport(e, m, entry, nil)
		return m.Save()
	}

	commit, found, cleanup, err := clone(ctx, entry)
	if err != nil {
		return err
	}
	defer cleanup()
	names := skillNames(found)

	// 1. Which skills.
	filter := e.SkillFlag
	if filter == nil && e.interactive() && len(names) > 1 {
		filter, err = pickSkills(names)
		if err != nil {
			return err
		}
	}
	if err := validateSkillNames(filter, names); err != nil {
		return err
	}

	// 2. Which agents.
	targetNames := e.AgentsFlag
	if len(targetNames) == 0 && e.interactive() {
		targetNames, err = pickAgents(reg.Names(), m.Targets())
		if err != nil {
			return err
		}
	}
	if len(targetNames) == 0 {
		targetNames = m.Targets()
	}

	// 3. Scope (not needed when --dir targets a literal directory).
	scope := e.scopeFor(entry)
	if e.DirOverride == "" {
		if scope == "" {
			if e.interactive() {
				if scope, err = pickScope(); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("scope required: pass --global or --project")
			}
		}
		if scope != agents.Global && scope != agents.Project {
			return fmt.Errorf("invalid scope %q (want global or project)", scope)
		}
	}

	targets, err := e.resolveTargets(reg, targetNames, scope)
	if err != nil {
		return err
	}

	locks := newLockCache()
	results, err := installToTargets(entry, found, filter, targets, commit, locks)
	if err != nil {
		return err
	}
	if err := locks.saveAll(); err != nil {
		return err
	}

	// Persist the entry. A subset filter is stored only when it omits something.
	if filter != nil && len(filter) < len(names) {
		entry.Skills = filter
	}
	entry.Targets = targetNames
	entry.Scope = scope
	recordAndReport(e, m, entry, results)
	if err := m.Save(); err != nil {
		return err
	}
	if e.JSON {
		return e.emitJSON(results)
	}
	return nil
}

func recordAndReport(e *Env, m *config.Manifest, entry config.Entry, results []Result) {
	if replaced := m.Upsert(entry); replaced {
		e.logf("updated manifest entry: %s", entry.Source)
	} else {
		e.logf("added manifest entry: %s", entry.Source)
	}
	logResults(e, results)
}

func validateSkillNames(filter, available []string) error {
	for _, n := range filter {
		if !contains(available, n) {
			return fmt.Errorf("no such skill %q in source (available: %v)", n, available)
		}
	}
	return nil
}
