package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinAndDir(t *testing.T) {
	reg := New(nil)
	a, ok := reg.Get("codex")
	if !ok {
		t.Fatal("codex should be a built-in")
	}
	if a.Name != "codex" {
		t.Fatalf("Name = %q", a.Name)
	}
	home, _ := os.UserHomeDir()
	if got, want := a.Dir(Global, "/proj"), filepath.Join(home, ".codex/skills"); got != want {
		t.Fatalf("global dir = %q, want %q", got, want)
	}
	if got, want := a.Dir(Project, "/proj"), "/proj/.codex/skills"; got != want {
		t.Fatalf("project dir = %q, want %q", got, want)
	}
}

func TestOverridesAndAdditions(t *testing.T) {
	reg := New(map[string]Agent{
		"claude":  {Global: "~/.custom/claude"},                              // override one field
		"myagent": {Global: "~/.myagent/skills", Project: ".myagent/skills"}, // new
	})
	if a, _ := reg.Get("claude"); a.Global != "~/.custom/claude" {
		t.Fatalf("override not applied: %q", a.Global)
	}
	if a, _ := reg.Get("claude"); a.Project != ".claude/skills" {
		t.Fatalf("untouched field should keep built-in: %q", a.Project)
	}
	if _, ok := reg.Get("myagent"); !ok {
		t.Fatal("addition missing")
	}
	if _, ok := reg.Get("nope"); ok {
		t.Fatal("unknown agent should not resolve")
	}
}
