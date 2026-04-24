package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check workflows exist
	assert.Contains(t, cfg.Workflows, "create-story")
	assert.Contains(t, cfg.Workflows, "dev-story")
	assert.Contains(t, cfg.Workflows, "code-review")
	assert.Contains(t, cfg.Workflows, "git-commit")

	// Check full cycle steps
	assert.Equal(t, []string{"create-story", "dev-story", "code-review", "git-commit"}, cfg.FullCycle.Steps)

	// Check defaults
	assert.Equal(t, "stream-json", cfg.Claude.OutputFormat)
	assert.Equal(t, "claude", cfg.Claude.BinaryPath)
	assert.Equal(t, 20, cfg.Output.TruncateLines)
	assert.Equal(t, 60, cfg.Output.TruncateLength)
}

func TestConfig_GetPrompt(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name         string
		workflowName string
		storyKey     string
		wantContains string
		wantErr      bool
	}{
		{
			name:         "create-story",
			workflowName: "create-story",
			storyKey:     "test-123",
			wantContains: "test-123",
			wantErr:      false,
		},
		{
			name:         "dev-story",
			workflowName: "dev-story",
			storyKey:     "feature-456",
			wantContains: "feature-456",
			wantErr:      false,
		},
		{
			name:         "unknown workflow",
			workflowName: "unknown",
			storyKey:     "test",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := cfg.GetPrompt(tt.workflowName, tt.storyKey)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, prompt, tt.wantContains)
			}
		})
	}
}

func TestConfig_GetFullCycleSteps(t *testing.T) {
	cfg := DefaultConfig()
	steps := cfg.GetFullCycleSteps()

	assert.Equal(t, []string{"create-story", "dev-story", "code-review", "git-commit"}, steps)
}

func TestLoader_LoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
workflows:
  custom-workflow:
    prompt_template: "Custom: {{.StoryKey}}"
full_cycle:
  steps:
    - custom-workflow
claude:
  binary_path: /custom/path/claude
output:
  truncate_lines: 50
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	cfg, err := loader.LoadFromFile(configPath)

	require.NoError(t, err)
	assert.Contains(t, cfg.Workflows, "custom-workflow")
	assert.Equal(t, []string{"custom-workflow"}, cfg.FullCycle.Steps)
	assert.Equal(t, "/custom/path/claude", cfg.Claude.BinaryPath)
	assert.Equal(t, 50, cfg.Output.TruncateLines)
}

func TestLoader_Load_WithEnvOverride(t *testing.T) {
	// Set environment variable
	os.Setenv("BMAD_CLAUDE_PATH", "/env/claude")
	defer os.Unsetenv("BMAD_CLAUDE_PATH")

	loader := NewLoader()
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "/env/claude", cfg.Claude.BinaryPath)
}

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     PromptData
		want     string
		wantErr  bool
	}{
		{
			name:     "simple substitution",
			template: "Story: {{.StoryKey}}",
			data:     PromptData{StoryKey: "test-123"},
			want:     "Story: test-123",
			wantErr:  false,
		},
		{
			name:     "multiple substitutions",
			template: "{{.StoryKey}} - {{.StoryKey}}",
			data:     PromptData{StoryKey: "abc"},
			want:     "abc - abc",
			wantErr:  false,
		},
		{
			name:     "no substitution",
			template: "Static text",
			data:     PromptData{StoryKey: "ignored"},
			want:     "Static text",
			wantErr:  false,
		},
		{
			name:     "invalid template",
			template: "{{.Invalid",
			data:     PromptData{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandTemplate(tt.template, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	assert.NotNil(t, loader)
	assert.NotNil(t, loader.v)
}

func TestLoader_LoadFromFile_NonExistent(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadFromFile("/nonexistent/path/config.yaml")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestLoader_LoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	invalidContent := `
workflows:
  - this is not valid yaml for this structure
    missing: colon here
`
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	_, err = loader.LoadFromFile(configPath)

	// Should error on unmarshal due to wrong structure
	assert.Error(t, err)
}

func TestLoader_Load_DefaultsWithNoConfigFile(t *testing.T) {
	// Ensure no config file exists in current dir
	// Load() should fall back to defaults
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	// Clear any env vars that might interfere
	os.Unsetenv("BMAD_CONFIG_PATH")
	os.Unsetenv("BMAD_CLAUDE_PATH")

	loader := NewLoader()
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
	// Should have default values
	assert.Equal(t, "claude", cfg.Claude.BinaryPath)
	assert.Equal(t, "stream-json", cfg.Claude.OutputFormat)
}

func TestLoader_Load_WithConfigPathEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.yaml")

	configContent := `
claude:
  binary_path: /from/env/path/claude
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	os.Setenv("BMAD_CONFIG_PATH", configPath)
	defer os.Unsetenv("BMAD_CONFIG_PATH")

	loader := NewLoader()
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "/from/env/path/claude", cfg.Claude.BinaryPath)
}

func TestLoader_Load_EnvOverridesTakePrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config file sets one path
	configContent := `
claude:
  binary_path: /from/file/claude
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	os.Setenv("BMAD_CONFIG_PATH", configPath)
	os.Setenv("BMAD_CLAUDE_PATH", "/from/env/override/claude")
	defer os.Unsetenv("BMAD_CONFIG_PATH")
	defer os.Unsetenv("BMAD_CLAUDE_PATH")

	loader := NewLoader()
	cfg, err := loader.Load()

	require.NoError(t, err)
	// Env var should take precedence
	assert.Equal(t, "/from/env/override/claude", cfg.Claude.BinaryPath)
}

func TestMustLoad_Success(t *testing.T) {
	// MustLoad should not panic when loading defaults
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	os.Unsetenv("BMAD_CONFIG_PATH")
	os.Unsetenv("BMAD_CLAUDE_PATH")

	// Should not panic
	cfg := MustLoad()
	assert.NotNil(t, cfg)
}

func TestConfig_GetPrompt_AllWorkflows(t *testing.T) {
	cfg := DefaultConfig()

	workflows := []string{"create-story", "dev-story", "code-review", "git-commit"}

	for _, wf := range workflows {
		t.Run(wf, func(t *testing.T) {
			prompt, err := cfg.GetPrompt(wf, "test-key")
			assert.NoError(t, err)
			assert.NotEmpty(t, prompt)
			assert.Contains(t, prompt, "test-key")
		})
	}
}

// TestLoader_Load_GDSProjectEndToEnd simulates the full "kingdom-style" scenario:
// a project using the GDS module (.claude/skills/gds-dev-story, gds-create-story,
// gds-code-review) with a config/workflows.yaml still written for the bmm module
// (/bmad-bmm-* commands). After Load(), every workflow MUST target /gds-* so that
// bmad-automate invokes the correct slash commands for the detected module.
func TestLoader_Load_GDSProjectEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate kingdom's .claude/skills/ layout (BMAD v6 native skills)
	skillsDir := filepath.Join(tmpDir, ".claude", "skills")
	for _, skill := range []string{"gds-create-story", "gds-dev-story", "gds-code-review"} {
		require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, skill), 0o755))
	}

	// Simulate the project's config/workflows.yaml with the old bmm-prefixed commands
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	workflowsYAML := `
workflows:
  create-story:
    prompt_template: "/bmad-bmm-create-story - Create story: {{.StoryKey}}."
    model: "opus"
  dev-story:
    prompt_template: "/bmad-bmm-dev-story - Work on story: {{.StoryKey}}."
    model: "opus"
    effort: "xhigh"
  code-review:
    prompt_template: "/bmad-bmm-code-review - Review story: {{.StoryKey}}."
    model: "opus"
    effort: "xhigh"
  git-commit:
    prompt_template: "Commit all changes for story {{.StoryKey}}."
    model: "sonnet"
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "workflows.yaml"), []byte(workflowsYAML), 0o644))

	// Load from the project root so DetectCommandPrefix(".") picks up gds
	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(originalWd)

	os.Unsetenv("BMAD_CONFIG_PATH")
	os.Unsetenv("BMAD_CLAUDE_PATH")

	cfg, err := NewLoader().Load()
	require.NoError(t, err)

	// Every BMAD workflow must be rewritten to /gds-*
	expected := map[string]string{
		"create-story": "/gds-create-story",
		"dev-story":    "/gds-dev-story",
		"code-review":  "/gds-code-review",
	}
	for workflow, cmd := range expected {
		wf, ok := cfg.Workflows[workflow]
		require.True(t, ok, "workflow %s missing", workflow)
		assert.Contains(t, wf.PromptTemplate, cmd,
			"workflow %s should invoke %s, got %q", workflow, cmd, wf.PromptTemplate)
		assert.NotContains(t, wf.PromptTemplate, "/bmad-bmm-",
			"workflow %s still references bmm prefix: %q", workflow, wf.PromptTemplate)
		assert.NotContains(t, wf.PromptTemplate, "/bmad-create-story",
			"workflow %s still references plain bmad prefix: %q", workflow, wf.PromptTemplate)
	}

	// git-commit has no BMAD slash command — it must stay untouched
	gitCommit := cfg.Workflows["git-commit"].PromptTemplate
	assert.NotContains(t, gitCommit, "/gds-", "git-commit should not be rewritten")
	assert.Contains(t, gitCommit, "Commit all changes")

	// Rendered prompts (the actual string sent to Claude) must carry /gds-*
	for workflow, cmd := range expected {
		prompt, err := cfg.GetPrompt(workflow, "STORY-42")
		require.NoError(t, err)
		assert.Contains(t, prompt, cmd,
			"rendered prompt for %s should start with %s", workflow, cmd)
		assert.Contains(t, prompt, "STORY-42")
	}
}

func TestLoader_LoadFromFile_DifferentExtension(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write valid JSON config
	jsonContent := `{
		"claude": {
			"binary_path": "/json/path/claude"
		}
	}`
	err := os.WriteFile(configPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	cfg, err := loader.LoadFromFile(configPath)

	require.NoError(t, err)
	assert.Equal(t, "/json/path/claude", cfg.Claude.BinaryPath)
}

func TestDefaultConfig_WorkflowTemplates(t *testing.T) {
	cfg := DefaultConfig()

	// Verify each workflow has a non-empty template
	for name, workflow := range cfg.Workflows {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, workflow.PromptTemplate, "workflow %s should have a template", name)
		})
	}
}

func TestPromptData_StoryKey(t *testing.T) {
	data := PromptData{StoryKey: "ABC-123"}
	assert.Equal(t, "ABC-123", data.StoryKey)
}

func TestConfig_ApplyCostOptimizedMode(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default models
	assert.Equal(t, "opus", cfg.Workflows["create-story"].Model, "create-story should default to opus")
	assert.Equal(t, "opus", cfg.Workflows["dev-story"].Model, "dev-story should default to opus")
	assert.Equal(t, "xhigh", cfg.Workflows["dev-story"].Effort, "dev-story should default to xhigh effort")
	assert.Equal(t, "opus", cfg.Workflows["code-review"].Model, "code-review should default to opus")
	assert.Equal(t, "xhigh", cfg.Workflows["code-review"].Effort, "code-review should default to xhigh effort")
	assert.Equal(t, "sonnet", cfg.Workflows["git-commit"].Model, "git-commit should default to sonnet")
	assert.Equal(t, "medium", cfg.Workflows["git-commit"].Effort, "git-commit should default to medium effort")

	// Apply cost optimization
	cfg.ApplyCostOptimizedMode()

	// Verify models after optimization
	assert.Equal(t, "opus", cfg.Workflows["create-story"].Model, "create-story should remain opus after cost optimization")
	assert.Equal(t, "sonnet", cfg.Workflows["dev-story"].Model, "dev-story should switch to sonnet after cost optimization")
	assert.Equal(t, "sonnet", cfg.Workflows["code-review"].Model, "code-review should switch to sonnet after cost optimization")
	assert.Equal(t, "sonnet", cfg.Workflows["git-commit"].Model, "git-commit should remain sonnet after cost optimization")
	assert.Equal(t, "medium", cfg.Workflows["git-commit"].Effort, "git-commit effort should stay medium (invariant)")
}

func TestConfig_ApplyEconomyMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ApplyEconomyMode()

	assert.Equal(t, "opus", cfg.Workflows["create-story"].Model, "create-story untouched by economy mode")
	assert.Equal(t, "opus", cfg.Workflows["dev-story"].Model, "dev-story stays opus in economy mode")
	assert.Equal(t, "xhigh", cfg.Workflows["dev-story"].Effort, "dev-story effort is xhigh in economy mode")
	assert.Equal(t, "sonnet", cfg.Workflows["code-review"].Model, "code-review drops to sonnet in economy mode")
	assert.Equal(t, "high", cfg.Workflows["code-review"].Effort, "code-review effort is high in economy mode")
	assert.Equal(t, "sonnet", cfg.Workflows["git-commit"].Model, "git-commit invariant: sonnet")
	assert.Equal(t, "medium", cfg.Workflows["git-commit"].Effort, "git-commit invariant: medium")
}

func TestConfig_ApplyCostOptimizedMode_Idempotent(t *testing.T) {
	cfg := DefaultConfig()

	// Apply twice
	cfg.ApplyCostOptimizedMode()
	cfg.ApplyCostOptimizedMode()

	// Should still be correct
	assert.Equal(t, "sonnet", cfg.Workflows["dev-story"].Model)
	assert.Equal(t, "sonnet", cfg.Workflows["code-review"].Model)
}
