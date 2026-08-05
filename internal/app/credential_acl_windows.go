//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"os/user"
)

func secureCredential(path string) error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("确定当前 Windows 用户失败: %w", err)
	}
	command := exec.Command("icacls", path, "/inheritance:r", "/grant:r", "*S-1-5-18:F", "/grant:r", "*S-1-5-32-544:F", "/grant:r", current.Username+":F")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("收紧凭据 ACL 失败: %w (%s)", err, output)
	}
	return nil
}
