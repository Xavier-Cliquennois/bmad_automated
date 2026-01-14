package ratelimit

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrorInfo contains information about a rate limit error.
type ErrorInfo struct {
	// IsRateLimitError indicates if this is a rate limit error
	IsRateLimitError bool

	// ResetTime is when the rate limit will reset (if available)
	ResetTime time.Time

	// WaitDuration is how long to wait before retrying
	WaitDuration time.Duration

	// OriginalMessage is the original error message
	OriginalMessage string
}

var (
	// Pattern for "Claude usage limit reached. Your limit will reset at 1pm (Etc/GMT+5)."
	usageLimitPattern = regexp.MustCompile(`Claude usage limit reached\. Your limit will reset at (\d{1,2})(am|pm) \(([^)]+)\)`)

	// Pattern for general rate limit errors
	rateLimitPattern = regexp.MustCompile(`rate[_ ]limit|429|Too Many Requests`)
)

// DetectRateLimitError analyzes an error message to determine if it's a rate limit error
// and extracts timing information if available.
func DetectRateLimitError(message string) ErrorInfo {
	info := ErrorInfo{
		OriginalMessage: message,
	}

	// Check if it's a rate limit error
	if !rateLimitPattern.MatchString(message) && !strings.Contains(message, "Claude usage limit reached") {
		return info
	}

	info.IsRateLimitError = true

	// Try to extract reset time
	matches := usageLimitPattern.FindStringSubmatch(message)
	if len(matches) >= 4 {
		// matches[1] = hour (e.g., "1")
		// matches[2] = am/pm
		// matches[3] = timezone (e.g., "Etc/GMT+5")

		hour := matches[1] + matches[2]
		timezone := matches[3]

		// Parse the reset time
		resetTime, duration := parseResetTime(hour, timezone)
		if !resetTime.IsZero() {
			info.ResetTime = resetTime
			info.WaitDuration = duration
		}
	}

	// If we couldn't extract timing info, use a default wait time
	if info.WaitDuration == 0 {
		// Default to 5 minutes if we can't determine the exact reset time
		info.WaitDuration = 5 * time.Minute
	}

	return info
}

// parseResetTime parses the reset time string and returns the next reset time and wait duration.
// hour format: "1pm", "12am", etc.
// timezone format: "Etc/GMT+5", "UTC", etc.
func parseResetTime(hour string, timezone string) (time.Time, time.Duration) {
	now := time.Now()

	// Parse timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// If timezone parsing fails, use local time
		loc = now.Location()
	}

	// Parse hour
	today := now.In(loc).Format("2006-01-02")
	resetTimeStr := today + " " + hour
	resetTime, err := time.ParseInLocation("2006-01-02 3pm", resetTimeStr, loc)
	if err != nil {
		return time.Time{}, 0
	}

	// If the reset time is in the past, it's tomorrow
	if resetTime.Before(now) {
		resetTime = resetTime.Add(24 * time.Hour)
	}

	duration := time.Until(resetTime)
	if duration < 0 {
		duration = 0
	}

	return resetTime, duration
}

// FormatWaitDuration formats a duration into a human-readable string.
func FormatWaitDuration(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%02dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dm", minutes)
}
