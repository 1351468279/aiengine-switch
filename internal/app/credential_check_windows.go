//go:build windows

package app

import "os"

func checkCredentialSecurity(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return "Windows ACL 已在安装时收紧", nil
}
