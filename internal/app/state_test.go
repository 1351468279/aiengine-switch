package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadStateRejectsNullToolState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	data := `{"schema_version":2,"tools":{"codex":null}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := loadState(path)
	if err == nil || !strings.Contains(err.Error(), "工具 codex 缺少状态") {
		t.Fatalf("loadState error = %v", err)
	}
}
