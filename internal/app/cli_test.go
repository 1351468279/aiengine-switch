package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestPromptInstallToolRequiresOneClient(t *testing.T) {
	var output bytes.Buffer
	tool, err := promptInstallToolFrom(strings.NewReader("\ninvalid\n2\n"), &output, []string{"codex", "claude", desktopTool})
	if err != nil {
		t.Fatal(err)
	}
	if tool != "claude" {
		t.Fatalf("selected %q, want claude", tool)
	}
	if !strings.Contains(output.String(), "选择无效") {
		t.Fatalf("invalid selection was not reported: %s", output.String())
	}
}

func TestInstallRejectsAllTools(t *testing.T) {
	_, err := detectInstallTool("all")
	if err == nil || !strings.Contains(err.Error(), "一次只能配置一个客户端") {
		t.Fatalf("unexpected error: %v", err)
	}
}
