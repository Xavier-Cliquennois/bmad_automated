package config

import (
	"os"
	"path/filepath"
	"strings"
)

// CommandFormat represents the naming convention used for BMAD slash commands.
type CommandFormat int

const (
	// FormatBeta is the Beta 4+ format: /bmad-bmm-create-story
	FormatBeta CommandFormat = iota
	// FormatAlpha is the Alpha v6 format: /bmad:bmm:workflows:create-story
	FormatAlpha
)

// slashCommands maps workflow names to their slash command per format.
var slashCommands = map[string]map[CommandFormat]string{
	"create-story": {
		FormatBeta:  "/bmad-bmm-create-story",
		FormatAlpha: "/bmad:bmm:workflows:create-story",
	},
	"dev-story": {
		FormatBeta:  "/bmad-bmm-dev-story",
		FormatAlpha: "/bmad:bmm:workflows:dev-story",
	},
	"code-review": {
		FormatBeta:  "/bmad-bmm-code-review",
		FormatAlpha: "/bmad:bmm:workflows:code-review",
	},
}

// DetectCommandFormat checks the project directory to determine which
// BMAD slash command naming convention is in use.
//
// Detection logic:
//  1. If .claude/commands/bmad-bmm-create-story.md exists → FormatBeta
//  2. If a file containing "create-story" exists in a subdirectory → FormatAlpha
//  3. Default → FormatBeta
func DetectCommandFormat(projectDir string) CommandFormat {
	// Check for Beta format file at root of .claude/commands/
	betaFile := filepath.Join(projectDir, ".claude", "commands", "bmad-bmm-create-story.md")
	if _, err := os.Stat(betaFile); err == nil {
		return FormatBeta
	}

	// Check for Alpha format: look for create-story in subdirectories
	commandsDir := filepath.Join(projectDir, ".claude", "commands")
	if entries, err := os.ReadDir(commandsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			// Look recursively in subdirectories for a file matching create-story
			subDir := filepath.Join(commandsDir, entry.Name())
			if containsFile(subDir, "create-story") {
				return FormatAlpha
			}
		}
	}

	// Default to Beta format
	return FormatBeta
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
