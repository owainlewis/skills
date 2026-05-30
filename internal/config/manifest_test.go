package config

import (
	"path/filepath"
	"testing"
)

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skills.toml")
	m, err := Load(path) // missing -> empty
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Skills) != 0 {
		t.Fatalf("expected empty manifest, got %+v", m.Skills)
	}

	if replaced := m.Upsert(Entry{Source: "owner/repo", Path: "skills/x", Ref: "main"}); replaced {
		t.Fatal("first upsert should not replace")
	}
	if replaced := m.Upsert(Entry{Source: "owner/repo", Path: "skills/x", Ref: "v2"}); !replaced {
		t.Fatal("same source+path should replace")
	}
	m.Upsert(Entry{Source: "other/repo"})
	if len(m.Skills) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.Skills))
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Skills) != 2 {
		t.Fatalf("expected 2 after reload, got %d", len(reloaded.Skills))
	}
	if reloaded.Skills[0].Ref != "v2" {
		t.Fatalf("expected ref v2, got %q", reloaded.Skills[0].Ref)
	}
}

func TestResolvedDir(t *testing.T) {
	if got := (&Manifest{}).ResolvedDir(); got != "" {
		t.Fatalf("unset Dir should resolve to empty, got %q", got)
	}
	if got := (&Manifest{Dir: "~/skills"}).ResolvedDir(); got == "~/skills" || got == "" {
		t.Fatalf("expected ~ expansion, got %q", got)
	}
}

func TestDefaultTargets(t *testing.T) {
	if got := (&Manifest{}).Targets(); len(got) == 0 {
		t.Fatal("expected built-in default targets")
	}
	custom := &Manifest{DefaultTargets: []string{"claude"}}
	if got := custom.Targets(); len(got) != 1 || got[0] != "claude" {
		t.Fatalf("expected [claude], got %v", got)
	}
}

func TestLockRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l, err := LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	l.Put(Installed{Name: "zeta", Source: "o/r", Commit: "abc"})
	l.Put(Installed{Name: "alpha", Source: "o/r", Commit: "def"})
	l.Put(Installed{Name: "zeta", Source: "o/r", Commit: "updated"}) // replace
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Skills) != 2 {
		t.Fatalf("expected 2, got %d", len(reloaded.Skills))
	}
	// Sorted by name on save.
	if reloaded.Skills[0].Name != "alpha" {
		t.Fatalf("expected alpha first, got %q", reloaded.Skills[0].Name)
	}
	z, ok := reloaded.Get("zeta")
	if !ok || z.Commit != "updated" {
		t.Fatalf("zeta = %+v ok=%v", z, ok)
	}
	if !reloaded.Remove("alpha") || reloaded.Remove("nope") {
		t.Fatal("Remove semantics wrong")
	}
}
