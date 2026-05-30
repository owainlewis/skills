// Package command implements the skills subcommands. Conventions across all
// commands: data goes to Out (stdout), logs to Err (stderr), commands are
// non-interactive, and --json yields machine-readable output.
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/owainlewis/skills/internal/gitx"
	"github.com/owainlewis/skills/internal/installer"
	"github.com/owainlewis/skills/internal/manifest"
	"github.com/owainlewis/skills/internal/skill"
	"github.com/owainlewis/skills/internal/source"
)

// Env carries shared configuration and I/O for a command invocation.
type Env struct {
	ConfigPath  string // path to skills.toml
	DirOverride string // --dir / SKILLS_DIR; takes precedence over manifest.Dir
	JSON        bool
	Out         io.Writer
	Err         io.Writer
}

// Result describes the outcome for a single skill, used in logs and JSON.
type Result struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Commit string `json:"commit"`
	Status string `json:"status"` // installed | updated | unchanged | removed | missing
}

func (e *Env) logf(format string, a ...any) {
	fmt.Fprintf(e.Err, format+"\n", a...)
}

func (e *Env) emitJSON(v any) error {
	enc := json.NewEncoder(e.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (e *Env) loadManifest() (*manifest.Manifest, error) {
	return manifest.Load(e.ConfigPath)
}

// resolveDir picks the install directory: --dir override, else manifest, else default.
func (e *Env) resolveDir(m *manifest.Manifest) string {
	if e.DirOverride != "" {
		return expandHome(e.DirOverride)
	}
	return m.ResolvedDir()
}

// installEntry clones one manifest entry, discovers its skill(s), installs each,
// and records them in the lock. It returns one Result per skill. A prior lock
// entry with the same commit yields status "unchanged".
func installEntry(ctx context.Context, e *Env, dir string, entry manifest.Entry, lock *manifest.Lock) ([]Result, error) {
	src, err := source.Parse(entry.Source, entry.Path)
	if err != nil {
		return nil, err
	}
	co, err := gitx.Clone(ctx, src.GitURL, entry.Ref)
	if err != nil {
		return nil, err
	}
	defer co.Cleanup()

	found, err := skill.Discover(co.Dir, src.Path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", entry.Source, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var results []Result
	for _, f := range found {
		prev, had := lock.Get(f.Name)
		status := "installed"
		if had && prev.Commit == co.Commit && installedOnDisk(dir, f.Name) {
			status = "unchanged"
		} else if had {
			status = "updated"
		}

		if status != "unchanged" {
			if err := installer.Install(dir, f.Name, f.Dir); err != nil {
				return nil, fmt.Errorf("install %s: %w", f.Name, err)
			}
		}
		lock.Put(manifest.Installed{
			Name:        f.Name,
			Source:      entry.Source,
			Path:        entry.Path,
			Ref:         entry.Ref,
			Commit:      co.Commit,
			InstalledAt: now,
		})
		results = append(results, Result{
			Name: f.Name, Source: entry.Source, Path: entry.Path,
			Ref: entry.Ref, Commit: co.Commit, Status: status,
		})
	}
	return results, nil
}

func installedOnDisk(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.IsDir()
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
