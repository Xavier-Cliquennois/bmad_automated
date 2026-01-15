package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bmad-automate/internal/config"
)

func TestVersionCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newVersionCommand(app)

	assert.Equal(t, "version", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestVersionCommand_Output(t *testing.T) {
	app := &App{
		Config: config.DefaultConfig(),
	}

	rootCmd := NewRootCommand(app)
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "bmad-automate version")
	assert.Contains(t, output, Version)
	assert.Contains(t, output, "Release date:")
	assert.Contains(t, output, "Features:")
}

func TestVersionConstants(t *testing.T) {
	// Verify version constants are set
	assert.NotEmpty(t, Version)
	assert.NotEmpty(t, ReleaseDate)
	assert.NotEmpty(t, Features)

	// Version should follow semver format (basic check)
	assert.Contains(t, Version, ".")
}
