//go:build !windows

package app

import (
	"errors"
	"os"
)

func removeInstalledBinary(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
