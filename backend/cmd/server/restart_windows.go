//go:build windows

package main

import "errors"

func activatePersistentUpdate(string, string) error {
	return nil
}

func restartCurrentProcess(string, string) error {
	return errors.New("automatic restart is not supported on Windows")
}
