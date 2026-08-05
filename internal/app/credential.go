package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

func readToken(fromStdin bool, credentialPath string) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 1024*1024))
		if err != nil {
			return "", fmt.Errorf("从标准输入读取密钥: %w", err)
		}
		return validateToken(string(data))
	}

	inputName, outputName := "/dev/tty", "/dev/tty"
	if runtime.GOOS == "windows" {
		inputName, outputName = "CONIN$", "CONOUT$"
	}
	input, err := os.Open(inputName)
	if err != nil {
		return "", fmt.Errorf("无法打开交互终端；请改用 --token-stdin")
	}
	defer input.Close()
	output, err := os.OpenFile(outputName, os.O_WRONLY, 0)
	if err != nil {
		return "", fmt.Errorf("无法打开交互终端输出")
	}
	defer output.Close()

	hasCurrent := false
	if _, err := os.Stat(credentialPath); err == nil {
		hasCurrent = true
	}
	if hasCurrent {
		fmt.Fprint(output, "请输入新的 AIARE API 密钥（留空保留当前密钥）: ")
	} else {
		fmt.Fprint(output, "请输入 AIARE API 密钥: ")
	}
	data, err := term.ReadPassword(int(input.Fd()))
	fmt.Fprintln(output)
	if err != nil {
		return "", fmt.Errorf("读取密钥: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" && hasCurrent {
		current, err := os.ReadFile(credentialPath)
		if err != nil {
			return "", fmt.Errorf("读取现有密钥: %w", err)
		}
		return validateToken(string(current))
	}
	return validateToken(string(data))
}

func validateToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", fmt.Errorf("API 密钥不能为空")
	}
	if strings.ContainsAny(token, "\r\n\x00") {
		return "", fmt.Errorf("API 密钥格式无效")
	}
	if len(token) > 64*1024 {
		return "", fmt.Errorf("API 密钥长度异常")
	}
	return token, nil
}

func printCredential() error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	file, err := os.Open(paths.Credential)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("凭据不存在，请重新运行 install")
	}
	if err != nil {
		return fmt.Errorf("读取凭据: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(io.LimitReader(file, 64*1024+1))
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("读取凭据: %w", err)
		}
		return fmt.Errorf("凭据为空，请重新运行 install")
	}
	token, err := validateToken(scanner.Text())
	if err != nil {
		return err
	}
	if scanner.Scan() {
		return fmt.Errorf("凭据文件格式无效")
	}
	_, err = fmt.Fprintln(os.Stdout, token)
	return err
}
