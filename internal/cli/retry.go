package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bmad-automate/internal/lifecycle"
	"bmad-automate/internal/ratelimit"
	"bmad-automate/internal/router"
)

// executeStoryWithRetry executes a story's lifecycle with optional automatic retry on rate limit errors.
// This is used across all commands (run, queue, epic, all-epics) to provide consistent retry behavior.
func executeStoryWithRetry(ctx context.Context, executor *lifecycle.Executor, detector *ratelimit.Detector, storyKey string, autoRetry bool) error {
	maxRetries := 1
	if autoRetry {
		maxRetries = 10 // Allow up to 10 retries for rate limits
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := executor.Execute(ctx, storyKey)
		if err == nil {
			return nil // Success
		}

		// Check if it's a rate limit error
		rateLimitErr := detector.GetLastError()
		if rateLimitErr == nil || !rateLimitErr.IsRateLimitError {
			// Not a rate limit error, return the error immediately
			return err
		}

		// It's a rate limit error
		if !autoRetry {
			// Auto-retry not enabled, return the error
			fmt.Printf("\n⚠️  Rate limit reached. Use --auto-retry to automatically wait and retry.\n")
			return err
		}

		// Auto-retry enabled - wait and try again
		if attempt < maxRetries {
			waitTime := rateLimitErr.WaitDuration
			fmt.Printf("\n⏳ Rate limit reached. Waiting %s before retry (attempt %d/%d)...\n",
				ratelimit.FormatWaitDuration(waitTime), attempt, maxRetries)

			if !rateLimitErr.ResetTime.IsZero() {
				fmt.Printf("   Reset time: %s\n\n", rateLimitErr.ResetTime.Format("15:04:05 MST"))
			} else {
				fmt.Println()
			}

			// Wait for the specified duration
			select {
			case <-time.After(waitTime):
				// Reset detector for next attempt
				detector.Reset()
				fmt.Printf("Retrying story %s...\n\n", storyKey)
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			fmt.Printf("\n❌ Max retries reached (%d attempts)\n", maxRetries)
			return err
		}
	}

	return errors.New("max retries exceeded")
}

// shouldSkipCompletedStory checks if an error indicates a completed story.
// Returns true if the story should be skipped, false otherwise.
func shouldSkipCompletedStory(err error) bool {
	return errors.Is(err, router.ErrStoryComplete)
}
