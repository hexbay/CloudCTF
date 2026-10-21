//go:build !windows

package updater

import (
	"fmt"
	"os"
)

func installUpdate(executable, updatePath, backupPath string) error {
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove previous update backup: %w", err)
	}
	if err := os.Link(executable, backupPath); err != nil {
		return fmt.Errorf("back up current executable: %w", err)
	}
	if err := os.Rename(updatePath, executable); err != nil {
		_ = os.Remove(backupPath)
		return fmt.Errorf("install update: %w", err)
	}
	return nil
}
