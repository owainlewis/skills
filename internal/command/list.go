package command

import (
	"fmt"
	"io"
	"sort"
	"strings"
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

	// --json keeps the full per-(skill, agent) detail; the human view collapses
	// to one row per skill, listing its agents compactly.
	if e.JSON {
		return e.emitJSON(results)
	}
	if len(results) == 0 {
		e.logf("no skills installed")
		return nil
	}
	return printGrouped(e.Out, results)
}

// printGrouped renders one row per skill: SKILL, AGENTS (missing ones prefixed
// with !), SOURCE (or "(multiple)"), COMMIT (or "(mixed)").
func printGrouped(w io.Writer, results []Result) error {
	type group struct {
		agents  []string
		sources map[string]bool
		commits map[string]bool
	}
	groups := map[string]*group{}
	var order []string
	for _, r := range results {
		g := groups[r.Name]
		if g == nil {
			g = &group{sources: map[string]bool{}, commits: map[string]bool{}}
			groups[r.Name] = g
			order = append(order, r.Name)
		}
		label := r.Agent
		if label == "" {
			label = "-"
		}
		if r.Status == "missing" {
			label = "!" + label
		}
		g.agents = append(g.agents, label)
		g.sources[r.Source] = true
		g.commits[shortCommit(r.Commit)] = true
	}
	sort.Strings(order)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SKILL\tAGENTS\tSOURCE\tCOMMIT")
	for _, name := range order {
		g := groups[name]
		sort.Strings(g.agents)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", name, strings.Join(g.agents, ","), one(g.sources, "(multiple)"), one(g.commits, "(mixed)"))
	}
	return tw.Flush()
}

// one returns the single key of m, or alt when there are several.
func one(m map[string]bool, alt string) string {
	if len(m) == 1 {
		for k := range m {
			return k
		}
	}
	return alt
}
