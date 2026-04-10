package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWriter(t *testing.T) {
	writer := NewWriter("/some/path")

	assert.NotNil(t, writer)
	assert.Equal(t, "/some/path", writer.basePath)
}

func TestWriter_UpdateStatus_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the nested directory structure
	statusDir := filepath.Join(tmpDir, "_bmad-output")
	err := os.MkdirAll(statusDir, 0755)
	require.NoError(t, err)

	// Create initial sprint-status.yaml
	statusContent := `development_status:
  7-1-define-schema: ready-for-dev
  7-2-create-api: in-progress
  7-3-build-ui: backlog
`
	statusPath := filepath.Join(statusDir, "sprint-status.yaml")
	err = os.WriteFile(statusPath, []byte(statusContent), 0644)
	require.NoError(t, err)

	writer := NewWriter(tmpDir)
	err = writer.UpdateStatus("7-1-define-schema", StatusInProgress)

	require.NoError(t, err)

	// Verify the file was updated
	reader := NewReader(tmpDir)
	status, err := reader.GetStoryStatus("7-1-define-schema")
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, status)

	// Verify other stories weren't affected
	status, err = reader.GetStoryStatus("7-2-create-api")
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, status)

	status, err = reader.GetStoryStatus("7-3-build-ui")
	require.NoError(t, err)
	assert.Equal(t, StatusBacklog, status)
}

func TestWriter_UpdateStatus_StoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the nested directory structure
	statusDir := filepath.Join(tmpDir, "_bmad-output")
	err := os.MkdirAll(statusDir, 0755)
	require.NoError(t, err)

	statusContent := `development_status:
  7-1-define-schema: ready-for-dev
`
	statusPath := filepath.Join(statusDir, "sprint-status.yaml")
	err = os.WriteFile(statusPath, []byte(statusContent), 0644)
	require.NoError(t, err)

	writer := NewWriter(tmpDir)
	err = writer.UpdateStatus("nonexistent-story", StatusDone)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "story not found: nonexistent-story")
}

func TestWriter_UpdateStatus_InvalidStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the nested directory structure
	statusDir := filepath.Join(tmpDir, "_bmad-output")
	err := os.MkdirAll(statusDir, 0755)
	require.NoError(t, err)

	statusContent := `development_status:
  7-1-define-schema: ready-for-dev
`
	statusPath := filepath.Join(statusDir, "sprint-status.yaml")
	err = os.WriteFile(statusPath, []byte(statusContent), 0644)
	require.NoError(t, err)

	writer := NewWriter(tmpDir)
	err = writer.UpdateStatus("7-1-define-schema", Status("invalid-status"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestWriter_UpdateStatus_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewWriter(tmpDir)
	err := writer.UpdateStatus("any-story", StatusDone)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read sprint status")
}

func TestWriter_UpdateStatus_PreservesFormatting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the nested directory structure
	statusDir := filepath.Join(tmpDir, "_bmad-output")
	err := os.MkdirAll(statusDir, 0755)
	require.NoError(t, err)

	// Create sprint-status.yaml with comments and specific ordering
	statusContent := `# Sprint Status File
# This tracks story progress

development_status:
  # Epic 7 stories
  7-1-define-schema: ready-for-dev
  7-2-create-api: in-progress  # Currently working on this
  7-3-build-ui: backlog

  # Epic 8 stories
  8-1-setup: done
  8-2-implement: review
`
	statusPath := filepath.Join(statusDir, "sprint-status.yaml")
	err = os.WriteFile(statusPath, []byte(statusContent), 0644)
	require.NoError(t, err)

	writer := NewWriter(tmpDir)
	err = writer.UpdateStatus("7-2-create-api", StatusReview)
	require.NoError(t, err)

	// Read back the file content directly
	updatedContent, err := os.ReadFile(statusPath)
	require.NoError(t, err)

	content := string(updatedContent)

	// Verify comments are preserved
	assert.Contains(t, content, "# Sprint Status File", "top-level comment should be preserved")
	assert.Contains(t, content, "# Epic 7 stories", "section comment should be preserved")
	assert.Contains(t, content, "# Epic 8 stories", "section comment should be preserved")
	assert.Contains(t, content, "# Currently working on this", "inline comment should be preserved")

	// Verify the status was updated
	assert.Contains(t, content, "7-2-create-api: review", "status should be updated to review")

	// Verify key ordering is preserved (7-1 before 7-2 before 7-3, etc.)
	idx71 := indexOf(content, "7-1-define-schema")
	idx72 := indexOf(content, "7-2-create-api")
	idx73 := indexOf(content, "7-3-build-ui")
	idx81 := indexOf(content, "8-1-setup")
	idx82 := indexOf(content, "8-2-implement")

	assert.True(t, idx71 < idx72, "7-1 should come before 7-2")
	assert.True(t, idx72 < idx73, "7-2 should come before 7-3")
	assert.True(t, idx73 < idx81, "7-3 should come before 8-1")
	assert.True(t, idx81 < idx82, "8-1 should come before 8-2")
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestWriter_UpdateStatus_AllStatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus Status
		toStatus   Status
	}{
		{"backlog to ready-for-dev", StatusBacklog, StatusReadyForDev},
		{"ready-for-dev to in-progress", StatusReadyForDev, StatusInProgress},
		{"in-progress to review", StatusInProgress, StatusReview},
		{"review to done", StatusReview, StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			statusDir := filepath.Join(tmpDir, "_bmad-output")
			err := os.MkdirAll(statusDir, 0755)
			require.NoError(t, err)

			statusContent := "development_status:\n  test-story: " + string(tt.fromStatus) + "\n"
			statusPath := filepath.Join(statusDir, "sprint-status.yaml")
			err = os.WriteFile(statusPath, []byte(statusContent), 0644)
			require.NoError(t, err)

			writer := NewWriter(tmpDir)
			err = writer.UpdateStatus("test-story", tt.toStatus)
			require.NoError(t, err)

			reader := NewReader(tmpDir)
			status, err := reader.GetStoryStatus("test-story")
			require.NoError(t, err)
			assert.Equal(t, tt.toStatus, status)
		})
	}
}
