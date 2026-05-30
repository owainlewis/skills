# skills

A minimalist, agent-first installer for [agent skills](https://github.com/owainlewis/skills) —
reusable instruction directories (a folder containing `SKILL.md`) that extend coding
agents like Claude Code.

It is a single static Go binary. A declarative `skills.toml` is the source of truth, and
`skills sync` converges the installed set to match — so an agent can manage skills with one
deterministic command. **Private repositories work out of the box**: `skills` shells out to
your `git`, inheriting your existing SSH keys, credential helper, and tokens.

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
skills init                                   # write ~/.claude/skills.toml
skills add owainlewis/my-skills/skills/commit # add to manifest + install
skills add git@github.com:me/private.git      # private repo via SSH — no extra config
skills list                                   # show installed skills
skills sync                                   # make installed match the manifest
skills update                                 # pull latest commits for everything
```

## How it works

Skills are declared in `~/.claude/skills.toml`:

```toml
dir = "~/.claude/skills"          # install destination (default)

[[skill]]
source = "owainlewis/my-skills"   # shorthand, full URL, or git@ SSH URL
path   = "skills/commit"          # optional sub-directory within the repo
ref    = "main"                   # optional branch, tag, or commit SHA
```

- `skills add`/`remove` edit this manifest for you.
- `skills sync` clones each entry, installs every `SKILL.md` directory it finds, and records
  resolved commits in a lockfile (`<dir>/.skills.lock.toml`). With `--prune` it also removes
  installed skills no longer in the manifest. It is idempotent.
- `skills update [name...]` re-resolves installed skills to the latest commit for their ref.

A **skill** is any directory containing a `SKILL.md`. Its name comes from the `name:` field
of that file's YAML frontmatter, falling back to the directory name. When a source has no
`path`, `skills` installs the repo root (if it is a skill) or every skill found in `*/` and
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
| `skills add <source> [--ref r] [--path p] [--no-sync]` | Add to the manifest and install. |
| `skills remove <name>` | Uninstall a skill and drop its manifest entry. |
| `skills list [--json]` | List installed skills. |
| `skills sync [--json] [--prune]` | Reconcile installed skills with the manifest. |
| `skills update [name...] [--json]` | Update installed skills to their latest commit. |

Global flags: `--config <path>` (or `$SKILLS_CONFIG`), `--dir <path>` (or `$SKILLS_DIR`),
`--json`, `--version`.

### Agent use

All commands are non-interactive, write data to stdout and logs to stderr, support `--json`,
and return stable exit codes (`0` ok, `1` runtime error, `2` usage error). "Update our skills
to latest" is simply:

```sh
skills sync     # or: skills update
```

## License

MIT
