// Command skills installs and syncs agent skills from git repositories into one
// or more agent directories (e.g. ~/.claude/skills, ~/.agents/skills). It is
// designed for agent control: a declarative skills.toml is the source of truth
// and `skills sync` converges every target to match.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/owainlewis/skills/internal/agents"
	"github.com/owainlewis/skills/internal/command"
	"github.com/owainlewis/skills/internal/gitx"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

const usage = `skills — install and sync agent skills from git

Usage:
  skills <command> [flags]

Commands:
  init                     Write a starter skills.toml
  add <source>             Add a skill to the manifest and install it
  remove <name>            Remove an installed skill from all targets
  list                     List installed skills
  sync                     Reconcile installed skills with the manifest
  update [name...]         Update installed skills to their latest commit
  agents                   List known agents and their directories

Global flags:
  --config <path>          Manifest path (default ~/.claude/skills.toml, or $SKILLS_CONFIG)
  --dir <path>             Install into one literal directory, bypassing agents (or $SKILLS_DIR)
  --json                   Machine-readable output on stdout
  --version                Print version and exit

Selection flags (add; also override sync/update/list/remove targeting):
  --agent <a,b>            Agent targets (comma-separated). Default: manifest default_targets
  --skill <x,y>            Subset of skills to install (comma-separated). Default: all
  --global                 Install into agents' global (~/...) directories
  --project                Install into agents' project (current repo) directories
  --yes                    Accept defaults without interactive prompts

add flags:
  --ref <ref>              Branch, tag, or commit SHA
  --path <path>            Sub-directory within the repo
  --no-sync                Edit the manifest without installing

sync flags:
  --prune                  Remove installed skills absent from the manifest

Sources: owner/repo[/sub/path], full https URLs, or git@host:owner/repo.git
(SSH and credential-helper auth make private repos work with no extra config).

Exit codes: 0 ok, 1 runtime error, 2 usage error.`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, usage)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(os.Stdout, version)
		return 0
	}

	cmd, rest := args[0], args[1:]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "manifest path")
		dir        = fs.String("dir", "", "install directory override")
		jsonOut    = fs.Bool("json", false, "machine-readable output")
		ref        = fs.String("ref", "", "branch, tag, or commit SHA")
		path       = fs.String("path", "", "sub-directory within the repo")
		noSync     = fs.Bool("no-sync", false, "edit manifest without installing")
		prune      = fs.Bool("prune", false, "remove skills absent from the manifest")
		agentList  = fs.String("agent", "", "agent targets (comma-separated)")
		skillList  = fs.String("skill", "", "subset of skills (comma-separated)")
		global     = fs.Bool("global", false, "use agents' global directories")
		project    = fs.Bool("project", false, "use agents' project directories")
		yes        = fs.Bool("yes", false, "accept defaults without prompting")
	)
	positionals, err := parseInterspersed(fs, rest)
	if err != nil {
		return 2
	}

	scope, err := resolveScope(*global, *project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	cwd, _ := os.Getwd()
	env := &command.Env{
		ConfigPath:  resolveConfigPath(*configPath),
		ProjectRoot: cwd,
		DirOverride: firstNonEmpty(*dir, os.Getenv("SKILLS_DIR")),
		AgentsFlag:  splitList(*agentList),
		Scope:       scope,
		SkillFlag:   splitList(*skillList),
		JSON:        *jsonOut,
		Yes:         *yes,
		Out:         os.Stdout,
		Err:         os.Stderr,
	}

	if needsGit(cmd) && !gitx.Available() {
		fmt.Fprintln(os.Stderr, "error: git not found on PATH (required to fetch skills)")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "init":
		err = command.Init(env)
	case "add":
		err = command.Add(ctx, env, command.AddOpts{
			Source: arg(positionals, 0), Ref: *ref, Path: *path, NoSync: *noSync,
		})
	case "remove", "rm":
		err = command.Remove(ctx, env, arg(positionals, 0))
	case "list", "ls":
		err = command.List(env)
	case "sync":
		err = command.Sync(ctx, env, *prune)
	case "update":
		err = command.Update(ctx, env, positionals)
	case "agents":
		err = command.Agents(env)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s\n", cmd, usage)
		return 2
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// parseInterspersed parses flags that may appear before or after positional
// arguments. Go's flag package stops at the first positional; this re-parses the
// remainder after pulling each positional aside, so `add <src> --path x` works.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positionals, nil
		}
		positionals = append(positionals, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func resolveScope(global, project bool) (string, error) {
	switch {
	case global && project:
		return "", fmt.Errorf("--global and --project are mutually exclusive")
	case global:
		return agents.Global, nil
	case project:
		return agents.Project, nil
	default:
		return "", nil
	}
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func arg(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func needsGit(cmd string) bool {
	switch cmd {
	case "add", "sync", "update":
		return true
	}
	return false
}

func resolveConfigPath(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("SKILLS_CONFIG"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude/skills.toml"
	}
	return filepath.Join(home, ".claude", "skills.toml")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
