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

// DetectCommandPrefix looks in .claude/commands/ for a *-dev-story.md file
// and returns the detected prefix (e.g. "bmad", "gds", "bmad-bmm").
//
// Detection logic:
//  1. Find any root-level *.md file ending in "-dev-story.md" → extract prefix
//  2. If a file containing "create-story" exists in a subdirectory → "alpha"
//  3. Default → "bmad"
func DetectCommandPrefix(projectDir string) string {
	commandsDir := filepath.Join(projectDir, ".claude", "commands")

	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return defaultPrefix
	}

	// First pass: look for *-dev-story.md at root level (covers v6, beta, gds, any custom prefix)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "-dev-story.md") {
			return strings.TrimSuffix(name, "-dev-story.md")
		}
	}

	// Second pass: check for Alpha format (subdirectory structure)
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
