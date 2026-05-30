package command

import (
	"context"
	"fmt"

	"github.com/owainlewis/skills/internal/manifest"
)

// AddOpts configures Add.
type AddOpts struct {
	Source string
	Ref    string
	Path   string
	NoSync bool
}

// Add appends or updates a manifest entry and, unless NoSync, installs it.
func Add(ctx context.Context, e *Env, opts AddOpts) error {
	if opts.Source == "" {
		return fmt.Errorf("add: source required")
	}
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	entry := manifest.Entry{Source: opts.Source, Path: opts.Path, Ref: opts.Ref}
	if replaced := m.Upsert(entry); replaced {
		e.logf("updated manifest entry: %s", opts.Source)
	} else {
		e.logf("added manifest entry: %s", opts.Source)
	}
	if err := m.Save(); err != nil {
		return err
	}
	if opts.NoSync {
		return nil
	}

	dir := e.resolveDir(m)
	lock, err := manifest.LoadLock(dir)
	if err != nil {
		return err
	}
	results, err := installEntry(ctx, e, dir, entry, lock)
	if err != nil {
		return err
	}
	if err := lock.Save(); err != nil {
		return err
	}
	for _, r := range results {
		e.logf("%-10s %s (%s)", r.Status, r.Name, shortCommit(r.Commit))
	}
	if e.JSON {
		return e.emitJSON(results)
	}
	return nil
}
