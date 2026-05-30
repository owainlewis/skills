// Package skill discovers skill directories within a fetched repository.
//
// A skill is any directory that contains a SKILL.md file. The skill's name is
// taken from the `name:` field of the SKILL.md YAML frontmatter, falling back
// to the directory's base name.
package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const marker = "SKILL.md"

// Found is a discovered skill on disk.
type Found struct {
	// Name is the install name (frontmatter `name:` or directory base name).
	Name string
	// Dir is the absolute path to the skill directory.
	Dir string
}

// Discover finds skills under root, optionally scoped to subPath.
//
//   - subPath set:           that single directory is the skill (must contain SKILL.md).
//   - subPath empty, root has SKILL.md: root is the single skill.
//   - subPath empty, otherwise: scan for SKILL.md in immediate children and in
//     skills/* (depth <= 2). Every match is returned.
func Discover(root, subPath string) ([]Found, error) {
	if subPath != "" {
		dir := filepath.Join(root, filepath.FromSlash(subPath))
		if !hasMarker(dir) {
			return nil, fmt.Errorf("no %s in %s", marker, subPath)
		}
		f, err := newFound(dir)
		return []Found{f}, err
	}

	if hasMarker(root) {
		f, err := newFound(root)
		return []Found{f}, err
	}

	var found []Found
	seen := map[string]bool{}
	for _, base := range []string{root, filepath.Join(root, "skills")} {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue // base may not exist; that's fine
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			dir := filepath.Join(base, e.Name())
			if !hasMarker(dir) || seen[dir] {
				continue
			}
			seen[dir] = true
			f, err := newFound(dir)
			if err != nil {
				return nil, err
			}
			found = append(found, f)
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no %s found at repo root, in */, or skills/*/", marker)
	}
	return found, nil
}

func hasMarker(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, marker))
	return err == nil && !info.IsDir()
}

func newFound(dir string) (Found, error) {
	name, err := readName(filepath.Join(dir, marker))
	if err != nil {
		return Found{}, err
	}
	if name == "" {
		name = filepath.Base(dir)
	}
	return Found{Name: name, Dir: dir}, nil
}

// readName extracts the `name:` field from YAML frontmatter delimited by `---`
// fences. Returns "" (no error) when absent. No YAML dependency: a line scan is
// sufficient for a single top-level scalar field.
func readName(skillMD string) (string, error) {
	f, err := os.Open(skillMD)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	inFrontmatter := false
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break // end of frontmatter
		}
		if !inFrontmatter {
			// Frontmatter must start on the first non-empty line.
			if trimmed == "" {
				continue
			}
			return "", nil
		}
		if v, ok := strings.CutPrefix(line, "name:"); ok {
			return cleanScalar(v), nil
		}
	}
	return "", sc.Err()
}

func cleanScalar(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"'`)
	return strings.TrimSpace(v)
}
