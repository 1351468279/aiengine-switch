//go:build !windows

package app

import "os"

func secureCredential(path string) error {
	return os.Chmod(path, 0o600)
}
