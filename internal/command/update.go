package command

import "context"

// Update re-installs manifest skills at the latest commit for their ref. With no
// names it updates everything; with names it restricts to those skills. It never
// prunes or adds manifest entries — it refreshes what is declared.
func Update(ctx context.Context, e *Env, names []string) error {
	return reconcile(ctx, e, reconcileOpts{onlyNames: names})
}
