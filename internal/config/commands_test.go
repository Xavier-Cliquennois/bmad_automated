package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCommandPrefix_SkillsDir_GameDev(t *testing.T) {
	dir := t.TempDir()

	// BMAD v6 native skills: directories in .claude/skills/
	skillsDir := filepath.Join(dir, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(skillsDir, "gds-dev-story"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Also simulate unrelated bmad-* skills present alongside gds-*
	if err := os.MkdirAll(filepath.Join(skillsDir, "bmad-help"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandPrefix(dir)
	if got != "gds" {
		t.Errorf("expected 'gds', got %q", got)
	}
}

func TestDetectCommandPrefix_SkillsDir_TakesPriorityOverCommands(t *testing.T) {
	dir := t.TempDir()

	// .claude/skills/ has gds-dev-story
	skillsDir := filepath.Join(dir, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(skillsDir, "gds-dev-story"), 0o755); err != nil {
		t.Fatal(err)
	}
	// .claude/commands/ has bmad-bmm-dev-story.md (old format)
	cmdDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "bmad-bmm-dev-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// skills/ should win
	got := DetectCommandPrefix(dir)
	if got != "gds" {
		t.Errorf("expected 'gds' (from skills/), got %q", got)
	}
}

func TestDetectCommandPrefix_V6(t *testing.T) {
	dir := t.TempDir()

	cmdDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "bmad-dev-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandPrefix(dir)
	if got != "bmad" {
		t.Errorf("expected 'bmad', got %q", got)
	}
}

func TestDetectCommandPrefix_GameDev(t *testing.T) {
	dir := t.TempDir()

	cmdDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "gds-dev-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandPrefix(dir)
	if got != "gds" {
		t.Errorf("expected 'gds', got %q", got)
	}
}

func TestDetectCommandPrefix_Beta(t *testing.T) {
	dir := t.TempDir()

	cmdDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "bmad-bmm-dev-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandPrefix(dir)
	if got != "bmad-bmm" {
		t.Errorf("expected 'bmad-bmm', got %q", got)
	}
}

func TestDetectCommandPrefix_Alpha(t *testing.T) {
	dir := t.TempDir()

	subDir := filepath.Join(dir, ".claude", "commands", "bmad", "bmm", "workflows")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "create-story.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCommandPrefix(dir)
	if got != alphaPrefix {
		t.Errorf("expected alpha prefix, got %q", got)
	}
}

func TestDetectCommandPrefix_Default(t *testing.T) {
	dir := t.TempDir()

	got := DetectCommandPrefix(dir)
	if got != defaultPrefix {
		t.Errorf("expected default prefix %q, got %q", defaultPrefix, got)
	}
}

func TestAdaptSlashCommands_V6(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands("bmad")

	tests := map[string]string{
		"create-story": "/bmad-create-story",
		"dev-story":    "/bmad-dev-story",
		"code-review":  "/bmad-code-review",
	}
	checkPrompts(t, cfg, tests)
}

func TestAdaptSlashCommands_GameDev(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands("gds")

	tests := map[string]string{
		"create-story": "/gds-create-story",
		"dev-story":    "/gds-dev-story",
		"code-review":  "/gds-code-review",
	}
	checkPrompts(t, cfg, tests)
}

func TestAdaptSlashCommands_Beta(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands("bmad-bmm")

	tests := map[string]string{
		"create-story": "/bmad-bmm-create-story",
		"dev-story":    "/bmad-bmm-dev-story",
		"code-review":  "/bmad-bmm-code-review",
	}
	checkPrompts(t, cfg, tests)
}

func TestAdaptSlashCommands_Alpha(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands(alphaPrefix)

	tests := map[string]string{
		"create-story": "/bmad:bmm:workflows:create-story",
		"dev-story":    "/bmad:bmm:workflows:dev-story",
		"code-review":  "/bmad:bmm:workflows:code-review",
	}
	checkPrompts(t, cfg, tests)
}

func TestAdaptSlashCommands_IdempotentV6(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands("bmad")
	cfg.AdaptSlashCommands("bmad")

	wf := cfg.Workflows["dev-story"]
	if !containsSubstring(wf.PromptTemplate, "/bmad-dev-story") {
		t.Errorf("expected /bmad-dev-story, got %q", wf.PromptTemplate)
	}
}

func TestAdaptSlashCommands_AlphaThenGameDev(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AdaptSlashCommands(alphaPrefix)
	cfg.AdaptSlashCommands("gds")

	tests := map[string]string{
		"dev-story":   "/gds-dev-story",
		"code-review": "/gds-code-review",
	}
	checkPrompts(t, cfg, tests)
}

func TestDetectCommandFormat_BackwardCompat(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected CommandFormat
	}{
		{"v6 bmad", "bmad-dev-story.md", FormatV6},
		{"beta bmad-bmm", "bmad-bmm-dev-story.md", FormatBeta},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cmdDir := filepath.Join(dir, ".claude", "commands")
			if err := os.MkdirAll(cmdDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(cmdDir, tt.file), []byte("# test"), 0o644); err != nil {
				t.Fatal(err)
			}

			got := DetectCommandFormat(dir)
			if got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func checkPrompts(t *testing.T, cfg *Config, expected map[string]string) {
	t.Helper()
	for workflow, expectedCmd := range expected {
		wf, ok := cfg.Workflows[workflow]
		if !ok {
			t.Errorf("workflow %s not found", workflow)
			continue
		}
		if !containsSubstring(wf.PromptTemplate, expectedCmd) {
			t.Errorf("workflow %s: expected template to contain %q, got %q", workflow, expectedCmd, wf.PromptTemplate)
		}
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
