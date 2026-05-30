# skills

A minimalist, agent-first installer for **agent skills** — reusable instruction
directories (a folder containing `SKILL.md`) that extend coding agents like Claude Code,
Codex, Hermes, and others.

It is a single static Go binary. A declarative `skills.toml` is the source of truth, and
`skills sync` converges every target to match — so an agent can manage skills with one
deterministic command. **Private repositories work out of the box**: `skills` shells out to
your `git`, inheriting your existing SSH keys, credential helper, and tokens.

It installs the same skills into **multiple agents at once** (e.g. `~/.claude/skills`,
`~/.agents/skills`, `~/.hermes/skills`) and lets you pick which skills and which agents
interactively, or non-interactively via flags.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/owainlewis/skills/main/install.sh | sh
```

or with Homebrew:

```sh
brew install owainlewis/tap/skills
```

Requires `git` on your PATH.

## Quick start

```sh
skills init                        # write ~/.claude/skills.toml
skills add owainlewis/blueprint    # pick skills + agents interactively, then install
skills list                        # one row per skill, with its agents
skills sync                        # make every target match the manifest
skills update                      # pull latest commits everywhere
```

Run in a terminal, `add` prompts you:

```
skills add owainlewis/blueprint
  → Skills to install:        [x] commit  [x] review  [ ] tdd ...   (all pre-checked)
  → Install into which agents?[x] claude  [x] hermes  [x] agents    (your defaults pre-checked)
  → Install scope:            (•) global   ( ) project
```

To run it non-interactively (scripts, CI, agents), pass the choices as flags:

```sh
skills add owainlewis/blueprint --agent claude,hermes --skill commit,review --global --yes
```

## Example: install the blueprint skills

[owainlewis/blueprint](https://github.com/owainlewis/blueprint) keeps a set of skills under
its `skills/` directory (`commit`, `review`, `plan`, `tdd`, …). To install them:

```sh
# Interactive — pick which skills, which agents, and the scope:
skills add owainlewis/blueprint

# Non-interactive — all skills into claude + hermes + agents, global scope:
skills add owainlewis/blueprint --agent claude,hermes,agents --global --yes

# Just a few skills:
skills add owainlewis/blueprint --skill commit,review,plan --global --yes
```

This writes one `[[skill]]` entry to your `skills.toml` and installs into each agent's
directory. Afterwards:

```sh
skills list            # one row per skill, showing which agents have it
skills update          # pull the latest blueprint commit into every agent
```

```
$ skills list
SKILL   AGENTS                SOURCE                COMMIT
commit  agents,claude,hermes  owainlewis/blueprint  3d60e48
plan    agents,claude,hermes  owainlewis/blueprint  3d60e48
review  agents,claude,hermes  owainlewis/blueprint  3d60e48
```

(A missing install shows the agent prefixed with `!`, e.g. `!hermes`. Use `--json` for full
per-agent detail.)

## Agents & scope

A **target** is an agent plus a scope. Each agent has a global dir (under `$HOME`) and a
project dir (relative to the current repo). Built-in agents:

| agent | global | project |
|---|---|---|
| `claude` | `~/.claude/skills` | `./.claude/skills` |
| `codex` | `~/.codex/skills` | `./.codex/skills` |
| `hermes` | `~/.hermes/skills` | `./.hermes/skills` |
| `pi` | `~/.pi/agent/skills` | `./.pi/agent/skills` |
| `agents` | `~/.agents/skills` | `./.agents/skills` |

Run `skills agents` to see them (and which are defaults). Add or override any agent in the
manifest:

```toml
[agents.myagent]
global  = "~/.myagent/skills"
project = ".myagent/skills"
```

`--global` / `--project` choose the scope; in a terminal you're asked when neither is given.

## The manifest

```toml
default_targets = ["agents", "claude", "hermes"]   # pre-checked, and used when an entry omits targets

[[skill]]
source  = "owainlewis/blueprint"  # shorthand, full URL, or git@ SSH URL
path    = "skills/commit"         # optional sub-directory within the repo
ref     = "main"                  # optional branch, tag, or commit SHA
skills  = ["commit", "review"]    # optional subset; omit to install all
targets = ["claude", "hermes"]    # optional; omit to use default_targets
scope   = "global"                # "global" (~/...) or "project" (current repo)
```

`skills add`/`remove` edit this for you. `skills sync` clones each entry, installs the
selected skills into every target, and records resolved commits in a per-directory lockfile
(`<dir>/.skills.lock.toml`). With `--prune` it removes installed skills no longer declared —
including in agents you've dropped from `targets`. It is idempotent.

A **skill** is any directory containing a `SKILL.md`. Its name comes from the `name:` field
of that file's frontmatter, falling back to the directory name. When a source has no `path`,
`skills` installs the repo root (if it is a skill) or every skill found in `*/` and
`skills/*/`.

## Sources

| Form | Resolves to |
|------|-------------|
| `owner/repo` | `https://github.com/owner/repo.git` |
| `owner/repo/sub/dir` | repo above, sub-path `sub/dir` |
| `host.com/owner/repo` | `https://host.com/owner/repo.git` |
| `https://host/owner/repo[/sub]` | passed through, sub-path split off |
| `git@host:owner/repo.git` | passed through (SSH; private repos) |
| `/abs/path` or `./path` or `file://…` | local clone (handy for development) |

## Commands

| Command | Description |
|---------|-------------|
| `skills init` | Write a starter `skills.toml`. |
| `skills add <source> [flags]` | Choose skills/agents/scope and install; record in the manifest. |
| `skills remove <name>` | Uninstall a skill from every target and drop it from the manifest. |
| `skills list [--json]` | List installed skills, one row per skill (`--json` for per-agent detail). |
| `skills sync [--json] [--prune]` | Reconcile all targets with the manifest. |
| `skills update [name...] [--json]` | Update installed skills to their latest commit. |
| `skills agents [--json]` | List known agents and their directories. |

Global flags: `--config <path>` (or `$SKILLS_CONFIG`), `--dir <path>` (install into one
literal directory, bypassing agents; or `$SKILLS_DIR`), `--json`, `--version`.

Selection flags: `--agent <a,b>`, `--skill <x,y>`, `--global`, `--project`, `--yes`.

### Agent use

All commands write data to stdout and logs to stderr, support `--json`, and return stable
exit codes (`0` ok, `1` runtime error, `2` usage error). Interactive prompts appear **only**
in a real terminal with no `--json`/`--yes` and no selection flags — so agents and scripts
get deterministic behavior. "Update our skills to latest" is simply:

```sh
skills sync     # or: skills update
```

## Notes

This tool keeps a small dependency surface — TOML for the manifest and
[`charmbracelet/huh`](https://github.com/charmbracelet/huh) for the interactive pickers. The
pickers are only used in a terminal; non-interactive use never touches them.

## License

MIT
