package command

import (
	"fmt"
	"os"
	"path/filepath"
)

const starterManifest = `# skills.toml — desired skills (the agent-control surface).
# Edit this file, then run "skills sync" to converge the installed set.

# Agents to install into by default. Run "skills agents" to list known agents.
default_targets = ["agents", "claude", "hermes"]

# Add or override agent directories (built-ins: agents, claude, codex, hermes, pi):
# [agents.myagent]
# global  = "~/.myagent/skills"
# project = ".myagent/skills"

# Add skills as [[skill]] blocks:
#
# [[skill]]
# source  = "owner/repo"          # shorthand, full URL, or git@ SSH (private repos)
# path    = "skills/example"      # optional sub-directory within the repo
# ref     = "main"                # optional branch, tag, or commit SHA
# skills  = ["commit", "review"]  # optional subset; omit to install all
# targets = ["claude", "hermes"]  # optional; omit to use default_targets
# scope   = "global"              # "global" (~/...) or "project" (current repo)
`

// Init writes a starter manifest if none exists.
func Init(e *Env) error {
	if _, err := os.Stat(e.ConfigPath); err == nil {
		e.logf("manifest already exists: %s", e.ConfigPath)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(e.ConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(e.ConfigPath, []byte(starterManifest), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(e.Out, e.ConfigPath)
	e.logf("wrote starter manifest")
	return nil
}
