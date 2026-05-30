// Package source resolves user-supplied skill sources into a git URL plus an
// optional sub-path within the repository.
//
// Accepted forms:
//
//	owner/repo                    -> https://github.com/owner/repo.git
//	owner/repo/sub/dir            -> https://github.com/owner/repo.git, path "sub/dir"
//	host.com/owner/repo[/sub]     -> https://host.com/owner/repo.git
//	https://host/owner/repo[/sub] -> passed through (sub-path split off)
//	git@host:owner/repo.git       -> passed through unchanged (SSH; private repos)
//
// SSH and full URLs are handed to git as-is, which is how authentication for
// private repositories works: git uses the user's existing credentials.
package source

import (
	"fmt"
	"strings"
)

// Source is a resolved skill location.
type Source struct {
	// GitURL is the cloneable URL handed directly to git.
	GitURL string
	// Path is an optional sub-directory within the repo ("" means repo root).
	Path string
	// Raw is the original user input, stored verbatim in the manifest.
	Raw string
}

// Parse resolves a raw source string. An explicit path argument (from --path)
// overrides any sub-path encoded in the source itself.
func Parse(raw, explicitPath string) (Source, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Source{}, fmt.Errorf("empty source")
	}

	s := Source{Raw: raw}

	switch {
	case strings.HasPrefix(raw, "git@"):
		// SSH form: git@host:owner/repo.git — never carries a sub-path.
		s.GitURL = raw
	case isLocalPath(raw):
		// Local filesystem path or file:// URL — git clones these directly.
		// Useful for development and offline use; no sub-path is inferred.
		s.GitURL = raw
	case strings.Contains(raw, "://"):
		var err error
		s.GitURL, s.Path, err = splitURL(raw)
		if err != nil {
			return Source{}, err
		}
	default:
		var err error
		s.GitURL, s.Path, err = splitShorthand(raw)
		if err != nil {
			return Source{}, err
		}
	}

	if explicitPath != "" {
		s.Path = strings.Trim(explicitPath, "/")
	}
	return s, nil
}

// splitURL handles scheme://host/owner/repo[/sub/path]. Everything after the
// repo segment becomes the sub-path. A trailing ".git" on the repo is honored.
func splitURL(raw string) (gitURL, path string, err error) {
	schemeIdx := strings.Index(raw, "://")
	scheme := raw[:schemeIdx+3]
	rest := raw[schemeIdx+3:]

	host, tail, ok := strings.Cut(rest, "/")
	if !ok {
		return "", "", fmt.Errorf("invalid url %q: missing path", raw)
	}
	owner, tail2, ok := strings.Cut(tail, "/")
	if !ok {
		return "", "", fmt.Errorf("invalid url %q: expected host/owner/repo", raw)
	}
	repo, sub, _ := strings.Cut(tail2, "/")
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid url %q: expected host/owner/repo", raw)
	}
	repo = ensureGitSuffix(repo)
	gitURL = scheme + host + "/" + owner + "/" + repo
	return gitURL, strings.Trim(sub, "/"), nil
}

// splitShorthand handles [host/]owner/repo[/sub/path]. A bare two-segment
// value assumes github.com; a leading segment containing a dot is treated as a
// host.
func splitShorthand(raw string) (gitURL, path string, err error) {
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	host := "github.com"
	if len(parts) > 0 && strings.Contains(parts[0], ".") {
		host = parts[0]
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid source %q: expected owner/repo[/path]", raw)
	}
	owner, repo := parts[0], ensureGitSuffix(parts[1])
	sub := strings.Join(parts[2:], "/")
	gitURL = "https://" + host + "/" + owner + "/" + repo
	return gitURL, sub, nil
}

// isLocalPath reports whether raw refers to a local clone source rather than a
// remote shorthand: an absolute/relative filesystem path or a file:// URL.
func isLocalPath(raw string) bool {
	return strings.HasPrefix(raw, "/") ||
		strings.HasPrefix(raw, "./") ||
		strings.HasPrefix(raw, "../") ||
		strings.HasPrefix(raw, "file://")
}

func ensureGitSuffix(repo string) string {
	if strings.HasSuffix(repo, ".git") {
		return repo
	}
	return repo + ".git"
}
