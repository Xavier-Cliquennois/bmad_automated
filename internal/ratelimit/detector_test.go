package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDetectRateLimitError_OldFormat(t *testing.T) {
	// Test the original format: "Claude usage limit reached. Your limit will reset at 1pm (Etc/GMT+5)."
	message := "Claude usage limit reached. Your limit will reset at 1pm (Etc/GMT+5)."
	info := DetectRateLimitError(message)

	assert.True(t, info.IsRateLimitError)
	assert.NotZero(t, info.WaitDuration)
	assert.Equal(t, message, info.OriginalMessage)
}

func TestDetectRateLimitError_NewFormat(t *testing.T) {
	// Test the new format: "You've hit your limit · resets 12pm (Europe/Paris)"
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with straight apostrophe",
			message: "You've hit your limit · resets 12pm (Europe/Paris)",
		},
		{
			name:    "with curly apostrophe",
			message: "You've hit your limit · resets 12pm (Europe/Paris)",
		},
		{
			name:    "different timezone",
			message: "You've hit your limit · resets 3pm (America/New_York)",
		},
		{
			name:    "morning reset",
			message: "You've hit your limit · resets 8am (UTC)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectRateLimitError(tt.message)

			assert.True(t, info.IsRateLimitError, "should detect as rate limit error")
			assert.NotZero(t, info.WaitDuration, "should have wait duration")
			assert.Equal(t, tt.message, info.OriginalMessage)
		})
	}
}

func TestDetectRateLimitError_GenericPatterns(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		wantError bool
	}{
		{
			name:      "rate_limit",
			message:   "rate_limit exceeded",
			wantError: true,
		},
		{
			name:      "rate limit space",
			message:   "rate limit reached",
			wantError: true,
		},
		{
			name:      "429 error",
			message:   "Error 429: Too many requests",
			wantError: true,
		},
		{
			name:      "Too Many Requests",
			message:   "Too Many Requests",
			wantError: true,
		},
		{
			name:      "not a rate limit",
			message:   "Connection refused",
			wantError: false,
		},
		{
			name:      "normal text",
			message:   "Processing your request...",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectRateLimitError(tt.message)
			assert.Equal(t, tt.wantError, info.IsRateLimitError)
		})
	}
}

func TestDetectRateLimitError_NoMatch(t *testing.T) {
	info := DetectRateLimitError("Just a normal message")

	assert.False(t, info.IsRateLimitError)
	assert.Zero(t, info.WaitDuration)
}

func TestFormatWaitDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "less than a minute"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 15*time.Minute, "2h15m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatWaitDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetector_ProcessStderrLine(t *testing.T) {
	d := NewDetector()

	// Process a non-rate-limit message
	d.ProcessStderrLine("Some error occurred")
	assert.Nil(t, d.GetLastError())
	assert.Equal(t, []string{"Some error occurred"}, d.GetAllMessages())

	// Process a rate limit message
	d.ProcessStderrLine("Claude usage limit reached. Your limit will reset at 1pm (Etc/GMT+5).")
	assert.NotNil(t, d.GetLastError())
	assert.True(t, d.GetLastError().IsRateLimitError)
}

func TestDetector_ProcessTextMessage(t *testing.T) {
	d := NewDetector()

	// Process a non-rate-limit text message
	d.ProcessTextMessage("Hello, I'm Claude!")
	assert.Nil(t, d.GetLastError())
	assert.Empty(t, d.GetAllMessages()) // Text messages not added unless rate limit

	// Process a rate limit text message (new format from stdout)
	d.ProcessTextMessage("You've hit your limit · resets 12pm (Europe/Paris)")
	assert.NotNil(t, d.GetLastError())
	assert.True(t, d.GetLastError().IsRateLimitError)
	// Rate limit from text should be added with [stdout] prefix
	assert.Contains(t, d.GetAllMessages()[0], "[stdout]")
}

func TestDetector_Reset(t *testing.T) {
	d := NewDetector()

	d.ProcessStderrLine("Error message")
	d.ProcessStderrLine("Claude usage limit reached. Your limit will reset at 1pm (Etc/GMT+5).")

	assert.NotEmpty(t, d.GetAllMessages())
	assert.NotNil(t, d.GetLastError())

	d.Reset()

	assert.Empty(t, d.GetAllMessages())
	assert.Nil(t, d.GetLastError())
}
