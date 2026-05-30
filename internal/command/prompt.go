package command

import (
	"github.com/charmbracelet/huh"

	"github.com/owainlewis/skills/internal/agents"
)

// pickSkills asks which discovered skills to install. All are checked by
// default. Returns the chosen subset.
func pickSkills(names []string) ([]string, error) {
	selected := append([]string(nil), names...) // default: all
	opts := make([]huh.Option[string], len(names))
	for i, n := range names {
		opts[i] = huh.NewOption(n, n).Selected(true)
	}
	field := huh.NewMultiSelect[string]().
		Title("Skills to install").
		Options(opts...).
		Value(&selected)
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return nil, err
	}
	return selected, nil
}

// pickAgents asks which agent targets to install into, pre-checking defaults.
func pickAgents(all, defaults []string) ([]string, error) {
	selected := append([]string(nil), defaults...)
	opts := make([]huh.Option[string], len(all))
	for i, n := range all {
		opts[i] = huh.NewOption(n, n).Selected(contains(defaults, n))
	}
	field := huh.NewMultiSelect[string]().
		Title("Install into which agents?").
		Options(opts...).
		Value(&selected)
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return nil, err
	}
	return selected, nil
}

// pickScope asks global vs project.
func pickScope() (string, error) {
	scope := agents.Global
	field := huh.NewSelect[string]().
		Title("Install scope").
		Options(
			huh.NewOption("Global (~/...)", agents.Global),
			huh.NewOption("Project (current repo)", agents.Project),
		).
		Value(&scope)
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return "", err
	}
	return scope, nil
}
