package config

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

// LockName is the lockfile's basename, stored inside the install dir.
const LockName = ".skills.lock.toml"

// Installed records one installed skill (machine-managed; do not hand-edit).
type Installed struct {
	Name        string `toml:"name"`
	Source      string `toml:"source"`
	Path        string `toml:"path,omitempty"`
	Ref         string `toml:"ref,omitempty"`
	Commit      string `toml:"commit"`
	InstalledAt string `toml:"installed_at"`
}

// Lock is the installed-state set, keyed by skill name.
type Lock struct {
	Skills []Installed `toml:"skill"`

	path string
}

// LoadLock reads the lockfile inside dir. A missing file yields an empty lock.
func LoadLock(dir string) (*Lock, error) {
	path := filepath.Join(dir, LockName)
	l := &Lock{path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, l); err != nil {
		return nil, err
	}
	l.path = path
	return l, nil
}

// Save writes the lockfile atomically, sorted by name for stable diffs.
func (l *Lock) Save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	sort.Slice(l.Skills, func(i, j int) bool { return l.Skills[i].Name < l.Skills[j].Name })
	f, err := os.CreateTemp(filepath.Dir(l.path), ".tmp-lock-*")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	defer os.Remove(tmpName)
	if err := toml.NewEncoder(f).Encode(l); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, l.path)
}

// Get returns the installed record for name, if present.
func (l *Lock) Get(name string) (Installed, bool) {
	for _, s := range l.Skills {
		if s.Name == name {
			return s, true
		}
	}
	return Installed{}, false
}

// Put inserts or replaces a record by name.
func (l *Lock) Put(in Installed) {
	for i, s := range l.Skills {
		if s.Name == in.Name {
			l.Skills[i] = in
			return
		}
	}
	l.Skills = append(l.Skills, in)
}

// Remove deletes a record by name, reporting whether it existed.
func (l *Lock) Remove(name string) bool {
	for i, s := range l.Skills {
		if s.Name == name {
			l.Skills = append(l.Skills[:i], l.Skills[i+1:]...)
			return true
		}
	}
	return false
}
