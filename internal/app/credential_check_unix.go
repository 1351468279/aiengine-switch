//go:build !windows

package app

import (
	"fmt"
	"os"
)

func checkCredentialSecurity(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("权限为 %04o，应为 0600", info.Mode().Perm())
	}
	return fmt.Sprintf("权限 %04o", info.Mode().Perm()), nil
}
