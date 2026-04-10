package config

import (
	"os"
	"path/filepath"
	"strings"
)

// CommandFormat represents the naming convention used for BMAD slash commands.
// Kept for backward compatibility; prefer DetectCommandPrefix for new code.
type CommandFormat int

const (
	// FormatV6 is the v6.2+ format: /bmad-create-story
	FormatV6 CommandFormat = iota
	// FormatBeta is the Beta 4+ format: /bmad-bmm-create-story
	FormatBeta
	// FormatAlpha is the Alpha v6 format: /bmad:bmm:workflows:create-story
	FormatAlpha
)

// alphaPrefix is the sentinel prefix value for the colon-separated Alpha format.
const alphaPrefix = "alpha"

// defaultPrefix is the v6 standard prefix used when no commands are found.
const defaultPrefix = "bmad"

// workflowCommandNames maps workflow names to their base command segment.
var workflowCommandNames = map[string]string{
	"create-story": "create-story",
	"dev-story":    "dev-story",
	"code-review":  "code-review",
}

// DetectCommandPrefix detects the slash command prefix for this project by
// scanning for the *-dev-story skill or command file.
//
// Detection order:
//  1. .claude/skills/ directory: look for an entry named *-dev-story (BMAD v6 native skills)
//  2. .claude/commands/ directory: look for a *-dev-story.md file (flat file format)
//  3. .claude/commands/ subdirectories: Alpha colon-separated format → returns "alpha"
//  4. Default → "bmad"
func DetectCommandPrefix(projectDir string) string {
	claudeDir := filepath.Join(projectDir, ".claude")

	// Priority 1: .claude/skills/ — BMAD v6 native skill directories (e.g. gds-dev-story/)
	if prefix := findPrefixInDir(filepath.Join(claudeDir, "skills"), "-dev-story", true); prefix != "" {
		return prefix
	}

	commandsDir := filepath.Join(claudeDir, "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return defaultPrefix
	}

	// Priority 2: .claude/commands/ — flat .md files (e.g. bmad-dev-story.md)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), "-dev-story.md") {
			return strings.TrimSuffix(entry.Name(), "-dev-story.md")
		}
	}

	// Priority 3: .claude/commands/ — subdirectory Alpha format
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(commandsDir, entry.Name())
		if containsFile(subDir, "create-story") {
			return alphaPrefix
		}
	}

	return defaultPrefix
}

// findPrefixInDir scans dir for an entry whose name ends with suffix and returns
// the prefix (everything before the suffix). If allowDirs is false, only files
// are considered; if true, both files and directories are considered.
func findPrefixInDir(dir, suffix string, allowDirs bool) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !allowDirs && entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Strip .md extension for file-based skills
		base := strings.TrimSuffix(name, ".md")
		if strings.HasSuffix(base, suffix) {
			return strings.TrimSuffix(base, suffix)
		}
	}
	return ""
}

// DetectCommandFormat checks the project directory to determine which
// BMAD slash command naming convention is in use.
//
// Deprecated: Use DetectCommandPrefix for more general prefix detection.
func DetectCommandFormat(projectDir string) CommandFormat {
	prefix := DetectCommandPrefix(projectDir)
	switch prefix {
	case alphaPrefix:
		return FormatAlpha
	case "bmad-bmm":
		return FormatBeta
	case "bmad":
		return FormatV6
	default:
		return FormatV6
	}
}

// containsFile recursively searches dir for a file whose name contains pattern.
func containsFile(dir, pattern string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.Contains(d.Name(), pattern) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
