package cli

import (
	"bmad-automate/internal/config"
	"bmad-automate/internal/router"
)

// getModelForStep returns the Claude model configured for a workflow step.
// Returns the workflow-specific model if configured, otherwise the default model.
func getModelForStep(step router.LifecycleStep, cfg *config.Config) string {
	// Get model from workflow config, fallback to default model
	if wf, ok := cfg.Workflows[step.Workflow]; ok {
		if wf.Model != "" {
			return wf.Model
		}
	}
	return cfg.Claude.DefaultModel
}
