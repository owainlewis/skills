package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/config"
)

// allTargetDirs returns the distinct (agent, dir) targets referenced by the
// manifest. With --dir set, it returns that single directory.
func (e *Env) allTargetDirs(m *config.Manifest, reg *agents.Registry) ([]target, error) {
	if e.DirOverride != "" {
		return []target{{Dir: e.DirOverride}}, nil
	}
	seen := map[string]bool{}
	var out []target
	for _, entry := range m.Skills {
		scope := e.scopeFor(entry)
		if scope == "" {
			scope = agents.Global
		}
		targets, err := e.resolveTargets(reg, e.targetNames(m, entry), scope)
		if err != nil {
			return nil, err
		}
		for _, t := range targets {
			if !seen[t.Dir] {
				seen[t.Dir] = true
				out = append(out, t)
			}
		}
	}
	return out, nil
}

// List prints installed skills across every target directory the manifest
// references. Status is "ok" when present on disk, "missing" when deleted.
func List(e *Env) error {
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	reg := agents.New(m.Agents)
	dirs, err := e.allTargetDirs(m, reg)
	if err != nil {
		return err
	}

	var results []Result
	for _, t := range dirs {
		lock, err := config.LoadLock(t.Dir)
		if err != nil {
			return err
		}
		for _, in := range lock.Skills {
			status := "ok"
			if !installedOnDisk(t.Dir, in.Name) {
				status = "missing"
			}
			agent := in.Agent
			if agent == "" {
				agent = t.Agent
			}
			results = append(results, Result{
				Name: in.Name, Agent: agent, Source: in.Source, Path: in.Path,
				Ref: in.Ref, Commit: in.Commit, Dir: t.Dir, Status: status,
			})
		}
	}

	if e.JSON {
		return e.emitJSON(results)
	}
	if len(results) == 0 {
		e.logf("no skills installed")
		return nil
	}
	tw := tabwriter.NewWriter(e.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tAGENT\tSOURCE\tREF\tCOMMIT\tSTATUS")
	for _, r := range results {
		ref := r.Ref
		if ref == "" {
			ref = "-"
		}
		agent := r.Agent
		if agent == "" {
			agent = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", r.Name, agent, r.Source, ref, shortCommit(r.Commit), r.Status)
	}
	return tw.Flush()
}
