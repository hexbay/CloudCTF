//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"

	"cloudctf/backend/internal/updater"
)

func activatePersistentUpdate(currentVersion, updateDir string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	selected, ok, err := updater.SelectPersistentExecutable(currentVersion, updateDir, executable)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return syscall.Exec(selected, os.Args, os.Environ())
}

func restartCurrentProcess(currentVersion, updateDir string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	if updater.RunningInContainer() {
		if selected, ok, selectErr := updater.SelectPersistentExecutable(currentVersion, updateDir, executable); selectErr == nil && ok {
			return syscall.Exec(selected, os.Args, os.Environ())
		}
	}
	return syscall.Exec(executable, os.Args, os.Environ())
}
