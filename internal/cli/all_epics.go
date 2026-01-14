package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"bmad-automate/internal/lifecycle"
)

func newAllEpicsCommand(app *App) *cobra.Command {
	var dryRun bool
	var autoRetry bool

	cmd := &cobra.Command{
		Use:   "all-epics",
		Short: "Run full lifecycle for all active epics sequentially",
		Long: `Run the complete lifecycle for all active epics in order.

Finds all epics that are not "done", "deferred", or "optional", sorts them numerically,
and processes each epic's stories to completion before moving to the next epic.

For each story in each epic, executes all remaining workflows based on its current status:
  - backlog       → create-story → dev-story → code-review → git-commit → done
  - ready-for-dev → dev-story → code-review → git-commit → done
  - in-progress   → dev-story → code-review → git-commit → done
  - review        → code-review → git-commit → done
  - done          → skipped (story already complete)

The command stops on the first failure. Done stories are skipped and do not cause failure.
Status is updated in sprint-status.yaml after each successful workflow.

Rate Limiting:
  If a rate limit is encountered, the command will fail unless --auto-retry is enabled.
  With --auto-retry, the tool will automatically wait for the rate limit to reset and retry.

Use --dry-run to preview workflows without executing them.

Example:
  bmad-automate all-epics
  # Processes all active epics (e.g., epic 3, 4, 5) in order

  bmad-automate all-epics --auto-retry
  # Automatically waits and retries when rate limits are hit`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Get all active epics
			epicIDs, err := app.StatusReader.GetAllEpics()
			if err != nil {
				cmd.SilenceUsage = true
				fmt.Printf("Error getting epics: %v\n", err)
				return NewExitError(1)
			}

			fmt.Printf("Found %d active epic(s): %v\n\n", len(epicIDs), epicIDs)

			// Create lifecycle executor with app dependencies
			executor := lifecycle.NewExecutor(app.Runner, app.StatusReader, app.StatusWriter)

			// Handle dry-run mode
			if dryRun {
				return runAllEpicsDryRun(cmd, app, executor, epicIDs)
			}

			// Execute each epic in order
			for epicIdx, epicID := range epicIDs {
				fmt.Printf("═══════════════════════════════════════════════════════════════════\n")
				fmt.Printf("  Epic %d of %d: epic-%s\n", epicIdx+1, len(epicIDs), epicID)
				fmt.Printf("═══════════════════════════════════════════════════════════════════\n\n")

				// Get all stories for this epic
				storyKeys, err := app.StatusReader.GetEpicStories(epicID)
				if err != nil {
					cmd.SilenceUsage = true
					fmt.Printf("Error getting stories for epic %s: %v\n", epicID, err)
					return NewExitError(1)
				}

				// Execute full lifecycle for each story in order
				for storyIdx, storyKey := range storyKeys {
					fmt.Printf("─── Story %d of %d: %s\n", storyIdx+1, len(storyKeys), storyKey)

					// Reset rate limit detector before each story
					app.RateLimitDetector.Reset()

					err := executeStoryWithRetry(ctx, executor, app.RateLimitDetector, storyKey, autoRetry)
					if err != nil {
						if shouldSkipCompletedStory(err) {
							fmt.Printf("Story %s is already complete, skipping\n\n", storyKey)
							continue
						}
						cmd.SilenceUsage = true
						fmt.Printf("Error running lifecycle for story %s: %v\n", storyKey, err)
						return NewExitError(1)
					}
					fmt.Printf("Story %s completed successfully\n\n", storyKey)
				}

				fmt.Printf("Epic %s completed (%d stories processed)\n\n", epicID, len(storyKeys))
			}

			fmt.Printf("═══════════════════════════════════════════════════════════════════\n")
			fmt.Printf("  All %d epic(s) completed successfully!\n", len(epicIDs))
			fmt.Printf("═══════════════════════════════════════════════════════════════════\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workflows without executing them")
	cmd.Flags().BoolVar(&autoRetry, "auto-retry", false, "Automatically retry when rate limits are hit")

	return cmd
}

func runAllEpicsDryRun(cmd *cobra.Command, app *App, executor *lifecycle.Executor, epicIDs []string) error {
	fmt.Printf("Dry run for all epics:\n\n")

	totalEpics := len(epicIDs)
	totalStories := 0
	totalWorkflows := 0
	totalComplete := 0

	for epicIdx, epicID := range epicIDs {
		fmt.Printf("Epic %d of %d: epic-%s\n", epicIdx+1, totalEpics, epicID)

		// Get all stories for this epic
		storyKeys, err := app.StatusReader.GetEpicStories(epicID)
		if err != nil {
			cmd.SilenceUsage = true
			fmt.Printf("  Error: %v\n", err)
			return NewExitError(1)
		}

		epicStories := 0
		epicWorkflows := 0
		epicComplete := 0

		for _, storyKey := range storyKeys {
			steps, err := executor.GetSteps(storyKey)
			if err != nil {
				if shouldSkipCompletedStory(err) {
					epicComplete++
					continue
				}
				cmd.SilenceUsage = true
				fmt.Printf("  Error for story %s: %v\n", storyKey, err)
				return NewExitError(1)
			}

			epicStories++
			epicWorkflows += len(steps)
		}

		totalStories += epicStories
		totalWorkflows += epicWorkflows
		totalComplete += epicComplete

		if epicComplete > 0 {
			fmt.Printf("  %d workflows across %d stories (%d already complete)\n", epicWorkflows, epicStories, epicComplete)
		} else {
			fmt.Printf("  %d workflows across %d stories\n", epicWorkflows, epicStories)
		}
		fmt.Println()
	}

	fmt.Printf("═══════════════════════════════════════════════════════════════════\n")
	if totalComplete > 0 {
		fmt.Printf("Total: %d workflows across %d stories in %d epics (%d already complete)\n", totalWorkflows, totalStories, totalEpics, totalComplete)
	} else {
		fmt.Printf("Total: %d workflows across %d stories in %d epics\n", totalWorkflows, totalStories, totalEpics)
	}
	fmt.Printf("═══════════════════════════════════════════════════════════════════\n")

	return nil
}
