//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"strings"
)

func removeInstalledBinary(path string) error {
	escaped := strings.ReplaceAll(path, `"`, `""`)
	script := `ping 127.0.0.1 -n 2 >nul & del /f /q "` + escaped + `"`
	command := exec.Command("cmd.exe", "/d", "/c", "start", "", "/b", "cmd.exe", "/d", "/c", script)
	if err := command.Start(); err != nil {
		return fmt.Errorf("安排删除 %s: %w", path, err)
	}
	return nil
}
