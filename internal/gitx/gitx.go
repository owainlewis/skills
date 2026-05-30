// Package gitx wraps the git command line. Shelling out to the user's git lets
// private repositories work with no auth code of our own: git uses the caller's
// SSH keys, credential helper, GH_TOKEN, and .netrc.
package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Checkout is a fetched working tree on disk.
type Checkout struct {
	// Dir is the temporary clone directory. Call Cleanup when done.
	Dir string
	// Commit is the resolved HEAD SHA.
	Commit string
}

// Cleanup removes the temporary clone.
func (c *Checkout) Cleanup() {
	if c.Dir != "" {
		os.RemoveAll(c.Dir)
	}
}

// Clone shallow-clones gitURL at ref (branch, tag, or commit SHA) into a fresh
// temp directory and resolves HEAD. An empty ref uses the remote's default
// branch.
func Clone(ctx context.Context, gitURL, ref string) (*Checkout, error) {
	dir, err := os.MkdirTemp("", "skills-clone-")
	if err != nil {
		return nil, err
	}
	c := &Checkout{Dir: dir}

	args := []string{"clone", "--depth", "1", "--quiet"}
	if ref != "" && !looksLikeSHA(ref) {
		args = append(args, "--branch", ref)
	}
	args = append(args, gitURL, dir)
	if err := run(ctx, "", args...); err != nil {
		c.Cleanup()
		return nil, fmt.Errorf("clone %s: %w", gitURL, err)
	}

	// A full-length SHA can't be reached with --branch; fetch it explicitly.
	if looksLikeSHA(ref) {
		if err := run(ctx, dir, "fetch", "--depth", "1", "--quiet", "origin", ref); err != nil {
			c.Cleanup()
			return nil, fmt.Errorf("fetch %s: %w", ref, err)
		}
		if err := run(ctx, dir, "checkout", "--quiet", ref); err != nil {
			c.Cleanup()
			return nil, fmt.Errorf("checkout %s: %w", ref, err)
		}
	}

	commit, err := output(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		c.Cleanup()
		return nil, err
	}
	c.Commit = strings.TrimSpace(commit)
	return c, nil
}

// Available reports whether git is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func looksLikeSHA(ref string) bool {
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}
	for _, r := range ref {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

func output(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
