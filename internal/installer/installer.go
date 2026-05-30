// Package installer copies skill directories into the install destination,
// replacing any existing copy atomically.
package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Install copies srcDir into destDir/<name>, replacing an existing skill of the
// same name. The replace is atomic: contents are staged in a sibling temp dir
// and swapped via rename.
func Install(destDir, name, srcDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	final := filepath.Join(destDir, name)
	staging, err := os.MkdirTemp(destDir, ".staging-"+name+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	if err := copyTree(srcDir, staging); err != nil {
		return err
	}

	// Swap: move any existing copy aside, move staging into place, delete old.
	backup := ""
	if _, err := os.Lstat(final); err == nil {
		backup = final + ".old"
		os.RemoveAll(backup)
		if err := os.Rename(final, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(staging, final); err != nil {
		if backup != "" {
			os.Rename(backup, final) // best-effort rollback
		}
		return err
	}
	if backup != "" {
		os.RemoveAll(backup)
	}
	return nil
}

// Remove deletes destDir/<name>. A missing directory is not an error.
func Remove(destDir, name string) error {
	return os.RemoveAll(filepath.Join(destDir, name))
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, 0o755)
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode())
		default:
			return fmt.Errorf("unsupported file type: %s", path)
		}
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
