package ratelimit

import (
	"sync"
)

// Detector captures and analyzes stderr messages to detect rate limit errors.
// It's designed to be shared between the stderr handler and command execution logic.
type Detector struct {
	mu sync.Mutex

	// lastError holds the most recent rate limit error info
	lastError *ErrorInfo

	// errorMessages accumulates all stderr messages
	errorMessages []string
}

// NewDetector creates a new rate limit detector.
func NewDetector() *Detector {
	return &Detector{
		errorMessages: make([]string, 0),
	}
}

// ProcessStderrLine analyzes a single stderr line for rate limit errors.
// This should be called from the stderr handler for each line received.
func (d *Detector) ProcessStderrLine(line string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.errorMessages = append(d.errorMessages, line)

	// Detect rate limit error
	info := DetectRateLimitError(line)
	if info.IsRateLimitError {
		d.lastError = &info
	}
}

// GetLastError returns the most recent rate limit error, if any.
// Returns nil if no rate limit error has been detected.
func (d *Detector) GetLastError() *ErrorInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.lastError
}

// Reset clears all captured errors and messages.
// This should be called before starting a new command execution.
func (d *Detector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastError = nil
	d.errorMessages = d.errorMessages[:0]
}

// GetAllMessages returns all stderr messages captured so far.
func (d *Detector) GetAllMessages() []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	result := make([]string, len(d.errorMessages))
	copy(result, d.errorMessages)
	return result
}
