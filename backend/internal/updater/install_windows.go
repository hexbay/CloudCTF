//go:build windows

package updater

import (
	"fmt"
	"os"
)

// Windows is not a supported update target. This implementation keeps unit
// tests and cross-compilation working without relying on Unix hard links.
func installUpdate(executable, updatePath, backupPath string) error {
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove previous update backup: %w", err)
	}
	if err := os.Rename(executable, backupPath); err != nil {
		return fmt.Errorf("back up current executable: %w", err)
	}
	if err := os.Rename(updatePath, executable); err != nil {
		if rollbackErr := os.Rename(backupPath, executable); rollbackErr != nil {
			return fmt.Errorf("install update: %v (rollback also failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("install update: %w", err)
	}
	return nil
}
