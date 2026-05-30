package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/owainlewis/skills/internal/agents"
)

// agentRow is the JSON/`agents` shape for one known agent.
type agentRow struct {
	Name      string `json:"name"`
	Global    string `json:"global"`
	Project   string `json:"project"`
	Default   bool   `json:"default"`
	GlobalDir string `json:"global_dir"` // ~ expanded
}

// Agents lists the known agent registry (built-ins plus manifest overrides) and
// marks which are default targets.
func Agents(e *Env) error {
	m, err := e.loadManifest()
	if err != nil {
		return err
	}
	reg := agents.New(m.Agents)
	defaults := m.Targets()

	var rows []agentRow
	for _, name := range reg.Names() {
		a, _ := reg.Get(name)
		rows = append(rows, agentRow{
			Name: name, Global: a.Global, Project: a.Project,
			Default: contains(defaults, name), GlobalDir: a.Dir(agents.Global, e.ProjectRoot),
		})
	}

	if e.JSON {
		return e.emitJSON(rows)
	}
	tw := tabwriter.NewWriter(e.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "AGENT\tDEFAULT\tGLOBAL\tPROJECT")
	for _, r := range rows {
		def := ""
		if r.Default {
			def = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Name, def, r.Global, r.Project)
	}
	return tw.Flush()
}
