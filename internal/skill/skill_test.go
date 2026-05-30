package skill

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadNameFrontmatter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	write(t, p, "---\nname: my-cool-skill\ndescription: x\n---\nbody\n")
	got, err := readName(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "my-cool-skill" {
		t.Fatalf("got %q", got)
	}
}

func TestReadNameNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	write(t, p, "# Just a heading\nname: not-frontmatter\n")
	got, err := readName(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestDiscoverExplicitPath(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "skills", "commit", "SKILL.md"), "---\nname: commit\n---\n")
	found, err := Discover(root, "skills/commit")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "commit" {
		t.Fatalf("got %+v", found)
	}
}

func TestDiscoverExplicitPathMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := Discover(root, "nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverRootSkill(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "SKILL.md"), "---\nname: root-skill\n---\n")
	found, err := Discover(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "root-skill" {
		t.Fatalf("got %+v", found)
	}
}

func TestDiscoverMultiple(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "alpha", "SKILL.md"), "---\nname: alpha\n---\n")
	write(t, filepath.Join(root, "skills", "beta", "SKILL.md"), "") // no frontmatter -> dir name
	write(t, filepath.Join(root, ".hidden", "SKILL.md"), "")        // ignored
	write(t, filepath.Join(root, "docs", "README.md"), "")          // no marker

	found, err := Discover(root, "")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range found {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Fatalf("got %v", names)
	}
}

func TestDiscoverNone(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "docs", "README.md"), "")
	if _, err := Discover(root, ""); err == nil {
		t.Fatal("expected error")
	}
}
