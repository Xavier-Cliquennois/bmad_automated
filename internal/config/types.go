// Package config provides configuration loading and management for bmad-automate.
//
// Configuration is loaded using Viper, supporting YAML config files and environment
// variable overrides. The package provides sensible defaults that work out of the
// box, with the ability to customize workflows, output formatting, and Claude CLI
// settings.
//
// Key types:
//   - [Config] is the root configuration container with all settings
//   - [Loader] handles Viper-based configuration loading
//   - [WorkflowConfig] defines a single workflow's prompt template
//   - [ClaudeConfig] contains Claude CLI binary settings
//
// Configuration priority (highest to lowest):
//  1. Environment variables (BMAD_ prefix)
//  2. Config file specified by BMAD_CONFIG_PATH
//  3. ./config/workflows.yaml
//  4. [DefaultConfig] defaults
package config

import "strings"

// Config represents the root configuration structure.
//
// This is the main configuration container loaded by [Loader] and used throughout
// the application. Use [DefaultConfig] to get sensible defaults.
type Config struct {
	// Workflows maps workflow names to their configurations.
	// Keys are workflow names (e.g., "create-story", "dev-story").
	Workflows map[string]WorkflowConfig `mapstructure:"workflows"`

	// FullCycle defines the steps for full lifecycle execution.
	// Used by run, queue, and epic commands.
	FullCycle FullCycleConfig `mapstructure:"full_cycle"`

	// Claude contains Claude CLI binary configuration.
	Claude ClaudeConfig `mapstructure:"claude"`

	// Output contains terminal output formatting configuration.
	Output OutputConfig `mapstructure:"output"`
}

// WorkflowConfig represents a single workflow configuration.
//
// Each workflow has a prompt template that is expanded with story data
// using Go's text/template package.
type WorkflowConfig struct {
	// PromptTemplate is the Go template string for the workflow prompt.
	// Use {{.StoryKey}} to reference the story key.
	// Example: "Work on story: {{.StoryKey}}"
	PromptTemplate string `mapstructure:"prompt_template"`

	// Model is the Claude model to use for this workflow.
	// Examples: "sonnet", "opus", "claude-opus-4-6", "claude-sonnet-4-6"
	// If empty, uses ClaudeConfig.DefaultModel.
	Model string `mapstructure:"model"`

	// Effort is the effort level for this workflow.
	// Valid values: "low", "medium", "high", "max"
	// If empty, defaults based on model: opus -> "medium", sonnet -> "high".
	Effort string `mapstructure:"effort"`
}

// FullCycleConfig defines the steps for a full development cycle.
//
// This configuration is used by the run, queue, and epic commands
// to determine the sequence of workflows to execute.
type FullCycleConfig struct {
	// Steps is the ordered list of workflow names to execute.
	// Default: ["create-story", "dev-story", "code-review", "git-commit"]
	Steps []string `mapstructure:"steps"`
}

// ClaudeConfig contains Claude CLI configuration.
//
// These settings control how the Claude CLI binary is invoked.
type ClaudeConfig struct {
	// OutputFormat is the output format passed to Claude CLI.
	// Should be "stream-json" for structured event parsing.
	OutputFormat string `mapstructure:"output_format"`

	// BinaryPath is the path to the Claude CLI binary.
	// Default: "claude" (assumes Claude is in PATH).
	// Can be overridden with BMAD_CLAUDE_PATH environment variable.
	BinaryPath string `mapstructure:"binary_path"`

	// DefaultModel is the default Claude model to use for all workflows.
	// Examples: "sonnet", "opus", "claude-opus-4-6", "claude-sonnet-4-6"
	// If empty, uses Claude CLI's default model.
	// Can be overridden per-workflow via WorkflowConfig.Model.
	DefaultModel string `mapstructure:"default_model"`

	// DefaultEffort is the default effort level for all workflows.
	// Valid values: "low", "medium", "high", "max"
	// If empty, defaults based on model: opus -> "medium", sonnet -> "high".
	// Can be overridden per-workflow via WorkflowConfig.Effort.
	DefaultEffort string `mapstructure:"default_effort"`
}

// OutputConfig contains terminal output formatting configuration.
//
// These settings control how Claude's output is formatted in the terminal.
type OutputConfig struct {
	// TruncateLines is the maximum number of lines to display per event.
	// Additional lines are hidden with a "... (N more lines)" indicator.
	// Default: 20
	TruncateLines int `mapstructure:"truncate_lines"`

	// TruncateLength is the maximum length of each output line.
	// Longer lines are truncated with "..." suffix.
	// Default: 60
	TruncateLength int `mapstructure:"truncate_length"`
}

// DefaultConfig returns a new [Config] with sensible defaults.
//
// The defaults include standard workflow prompts for create-story, dev-story,
// code-review, and git-commit workflows, as well as Claude CLI and output
// formatting settings. These defaults work out of the box without any
// configuration file.
// defaultEffortForModel returns the default effort level for a given model.
// opus -> "medium", all others (sonnet, haiku, etc.) -> "high".
func defaultEffortForModel(model string) string {
	if model == "opus" || strings.HasPrefix(model, "claude-opus") {
		return "medium"
	}
	return "high"
}

func DefaultConfig() *Config {
	return &Config{
		Workflows: map[string]WorkflowConfig{
			"create-story": {
				PromptTemplate: "/bmad-create-story - Create story: {{.StoryKey}}. Do not ask questions.",
				Model:          "opus",
				Effort:         defaultEffortForModel("opus"), // medium
			},
			"dev-story": {
				PromptTemplate: "/bmad-dev-story - Work on story: {{.StoryKey}}. Complete all tasks. Run tests after each implementation. Do not ask clarifying questions - use best judgment based on existing patterns.",
				Model:          "sonnet",
				Effort:         defaultEffortForModel("sonnet"), // high
			},
			"code-review": {
				PromptTemplate: "/bmad-code-review - Review story: {{.StoryKey}}. When presenting fix options, always choose to auto-fix all issues immediately. Do not wait for user input.",
				Model:          "opus",
				Effort:         defaultEffortForModel("opus"), // medium
			},
			"git-commit": {
				PromptTemplate: "Commit all changes for story {{.StoryKey}} with a descriptive commit message following conventional commits format. Then push to the current branch. Do not ask questions.",
				Model:          "sonnet",
				Effort:         "medium", // git ops don't need high reasoning
			},
		},
		FullCycle: FullCycleConfig{
			Steps: []string{"create-story", "dev-story", "code-review", "git-commit"},
		},
		Claude: ClaudeConfig{
			OutputFormat:  "stream-json",
			BinaryPath:    "claude",
			DefaultModel:  "sonnet",
			DefaultEffort: defaultEffortForModel("sonnet"), // high
		},
		Output: OutputConfig{
			TruncateLines:  20,
			TruncateLength: 60,
		},
	}
}

// PromptData contains data for workflow template expansion.
//
// This struct is passed to Go's text/template when expanding workflow prompts.
// Fields are accessible in templates using {{.FieldName}} syntax.
type PromptData struct {
	// StoryKey is the identifier of the story being processed.
	// Access in templates with {{.StoryKey}}.
	StoryKey string
}

// AdaptSlashCommands replaces slash command names in all workflow prompt
// templates to match the detected command prefix. This ensures templates
// work for any BMAD module: standard agile (bmad-*), game dev (gds-*),
// or any custom module prefix.
//
// For the Alpha format, use prefix "alpha" to get colon-separated commands.
// For any other prefix, commands are built as /{prefix}-{command-base-name}.
func (c *Config) AdaptSlashCommands(prefix string) {
	for workflowName, baseName := range workflowCommandNames {
		wf, exists := c.Workflows[workflowName]
		if !exists {
			continue
		}

		var target string
		if prefix == alphaPrefix {
			target = "/bmad:bmm:workflows:" + baseName
		} else {
			target = "/" + prefix + "-" + baseName
		}

		// Replace any known command pattern with the target
		wf.PromptTemplate = replaceAnyKnownCommand(wf.PromptTemplate, baseName, target)
		c.Workflows[workflowName] = wf
	}
}

// replaceAnyKnownCommand replaces any slash command ending in "-{baseName}"
// or matching the alpha colon-format for baseName with the target command.
func replaceAnyKnownCommand(template, baseName, target string) string {
	// Replace alpha format: /bmad:bmm:workflows:{baseName}
	alpha := "/bmad:bmm:workflows:" + baseName
	if strings.Contains(template, alpha) {
		return strings.Replace(template, alpha, target, 1)
	}

	// Replace any /{prefix}-{baseName} pattern by scanning for "-{baseName}"
	suffix := "-" + baseName
	for {
		idx := strings.Index(template, suffix)
		if idx < 0 {
			break
		}
		// Walk back to find the slash that starts the command
		start := idx
		for start > 0 && template[start-1] != '/' && template[start-1] != ' ' && template[start-1] != '\n' {
			start--
		}
		if start > 0 && template[start-1] == '/' {
			start-- // include the slash
		}
		// Verify this looks like a slash command (starts with /)
		if start >= len(template) || template[start] != '/' {
			break
		}
		end := idx + len(suffix)
		template = template[:start] + target + template[end:]
		break
	}

	return template
}

// ApplyCostOptimizedMode modifies workflow models for cost optimization.
// It sets dev-story and code-review to use "sonnet" instead of "opus",
// and updates their effort levels accordingly (sonnet -> "high").
// create-story always uses "opus" and git-commit always uses "sonnet".
func (c *Config) ApplyCostOptimizedMode() {
	costOptimizedModels := map[string]string{
		"dev-story":   "sonnet",
		"code-review": "sonnet",
	}

	for workflowName, model := range costOptimizedModels {
		if wf, ok := c.Workflows[workflowName]; ok {
			wf.Model = model
			wf.Effort = defaultEffortForModel(model)
			c.Workflows[workflowName] = wf
		}
	}
}
