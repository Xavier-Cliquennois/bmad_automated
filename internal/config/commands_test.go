package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCommandFormat_BetaFiles(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/commands/bmad-bmm-create-story.md
	cmdDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "bmad-bmm-create-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandFormat(dir)
	if got != FormatBeta {
		t.Errorf("expected FormatBeta, got %d", got)
	}
}

func TestDetectCommandFormat_AlphaFiles(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/commands/bmad/bmm/workflows/create-story.md (subdirectory structure)
	subDir := filepath.Join(dir, ".claude", "commands", "bmad", "bmm", "workflows")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "create-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandFormat(dir)
	if got != FormatAlpha {
		t.Errorf("expected FormatAlpha, got %d", got)
	}
}

func TestDetectCommandFormat_NoFiles(t *testing.T) {
	dir := t.TempDir()

	got := DetectCommandFormat(dir)
	if got != FormatBeta {
		t.Errorf("expected FormatBeta (default), got %d", got)
	}
}

func TestAdaptSlashCommands_BetaToAlpha(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands(FormatAlpha)

	tests := map[string]string{
		"create-story": "/bmad:bmm:workflows:create-story",
		"dev-story":    "/bmad:bmm:workflows:dev-story",
		"code-review":  "/bmad:bmm:workflows:code-review",
	}

	for workflow, expectedCmd := range tests {
		wf, ok := cfg.Workflows[workflow]
		if !ok {
			t.Errorf("workflow %s not found", workflow)
			continue
		}
		if !contains(wf.PromptTemplate, expectedCmd) {
			t.Errorf("workflow %s: expected template to contain %q, got %q", workflow, expectedCmd, wf.PromptTemplate)
		}
	}
}

func TestAdaptSlashCommands_AlphaStaysAlpha(t *testing.T) {
	cfg := DefaultConfig()

	// First adapt to Alpha
	cfg.AdaptSlashCommands(FormatAlpha)
	// Adapt to Alpha again - should not change
	cfg.AdaptSlashCommands(FormatAlpha)

	wf := cfg.Workflows["create-story"]
	expected := "/bmad:bmm:workflows:create-story"
	if !contains(wf.PromptTemplate, expected) {
		t.Errorf("expected template to contain %q, got %q", expected, wf.PromptTemplate)
	}
}

func TestAdaptSlashCommands_BetaStaysBeta(t *testing.T) {
	cfg := DefaultConfig()

	// Templates are already in Beta format, adapting to Beta should not change them
	cfg.AdaptSlashCommands(FormatBeta)

	wf := cfg.Workflows["create-story"]
	expected := "/bmad-bmm-create-story"
	if !contains(wf.PromptTemplate, expected) {
		t.Errorf("expected template to contain %q, got %q", expected, wf.PromptTemplate)
	}
}

func TestAdaptSlashCommands_AlphaToBeta(t *testing.T) {
	cfg := DefaultConfig()

	// First set to Alpha
	cfg.AdaptSlashCommands(FormatAlpha)
	// Then adapt back to Beta
	cfg.AdaptSlashCommands(FormatBeta)

	tests := map[string]string{
		"create-story": "/bmad-bmm-create-story",
		"dev-story":    "/bmad-bmm-dev-story",
		"code-review":  "/bmad-bmm-code-review",
	}

	for workflow, expectedCmd := range tests {
		wf, ok := cfg.Workflows[workflow]
		if !ok {
			t.Errorf("workflow %s not found", workflow)
			continue
		}
		if !contains(wf.PromptTemplate, expectedCmd) {
			t.Errorf("workflow %s: expected template to contain %q, got %q", workflow, expectedCmd, wf.PromptTemplate)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
