//go:build windows

package main

import "errors"

func restartCurrentProcess() error {
	return errors.New("automatic restart is not supported on Windows")
}
