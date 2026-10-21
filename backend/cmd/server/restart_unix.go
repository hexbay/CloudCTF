//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func restartCurrentProcess() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	return syscall.Exec(executable, os.Args, os.Environ())
}
