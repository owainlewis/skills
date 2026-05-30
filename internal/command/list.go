package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/owainlewis/skills/internal/manifest"
)

// List prints installed skills from the lockfile. Status is "ok" when the
// skill's directory is present, "missing" when it has been deleted on disk.
func List(e *Env) error {
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	dir := e.resolveDir(m)
	lock, err := manifest.LoadLock(dir)
	if err != nil {
		return err
	}

	results := make([]Result, 0, len(lock.Skills))
	for _, in := range lock.Skills {
		status := "ok"
		if !installedOnDisk(dir, in.Name) {
			status = "missing"
		}
		results = append(results, Result{
			Name: in.Name, Source: in.Source, Path: in.Path,
			Ref: in.Ref, Commit: in.Commit, Status: status,
		})
	}

	if e.JSON {
		return e.emitJSON(results)
	}
	if len(results) == 0 {
		e.logf("no skills installed (dir: %s)", dir)
		return nil
	}
	tw := tabwriter.NewWriter(e.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSOURCE\tREF\tCOMMIT\tSTATUS")
	for _, r := range results {
		ref := r.Ref
		if ref == "" {
			ref = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Name, r.Source, ref, shortCommit(r.Commit), r.Status)
	}
	return tw.Flush()
}
