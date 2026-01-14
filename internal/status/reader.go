package status

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultStatusPath is the canonical location of the sprint-status.yaml file
// relative to the project root. This path is used by both [Reader] and [Writer].
const DefaultStatusPath = "_bmad-output/implementation-artifacts/sprint-status.yaml"

// Reader reads sprint status from YAML files at [DefaultStatusPath].
//
// The basePath field specifies the project root directory. When empty,
// the current working directory is used. The full path to the status file
// is constructed as: basePath + DefaultStatusPath.
type Reader struct {
	basePath string
}

// NewReader creates a new [Reader] with the specified base path.
//
// The basePath is the project root directory. Pass an empty string to use
// the current working directory.
func NewReader(basePath string) *Reader {
	return &Reader{
		basePath: basePath,
	}
}

// Read reads and parses the complete sprint status file.
//
// It returns the full [SprintStatus] structure containing all story statuses.
// Returns an error if the file cannot be read or parsed.
func (r *Reader) Read() (*SprintStatus, error) {
	fullPath := filepath.Join(r.basePath, DefaultStatusPath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sprint status: %w", err)
	}

	var status SprintStatus
	if err := yaml.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to read sprint status: %w", err)
	}

	return &status, nil
}

// GetStoryStatus returns the [Status] for a specific story key.
//
// It reads the status file and looks up the given story. Returns an error
// if the file cannot be read or if the story key is not found in the file.
func (r *Reader) GetStoryStatus(storyKey string) (Status, error) {
	sprintStatus, err := r.Read()
	if err != nil {
		return "", err
	}

	status, ok := sprintStatus.DevelopmentStatus[storyKey]
	if !ok {
		return "", fmt.Errorf("story not found: %s", storyKey)
	}

	return status, nil
}

// GetEpicStories returns all story keys belonging to an epic, sorted by story number.
//
// Story keys are matched using the pattern {epicID}-{N}-*, where N is a numeric
// story number. Results are sorted numerically by story number (1, 2, 10 not 1, 10, 2).
//
// Returns an error if the file cannot be read or if no stories are found for the epic.
func (r *Reader) GetEpicStories(epicID string) ([]string, error) {
	sprintStatus, err := r.Read()
	if err != nil {
		return nil, err
	}

	// Collect all keys matching the epic ID pattern
	type storyWithNum struct {
		key string
		num int
	}
	var stories []storyWithNum

	prefix := epicID + "-"
	for key := range sprintStatus.DevelopmentStatus {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Extract the story number (second segment)
		// Format: {epicID}-{storyNum}-{rest}
		remainder := strings.TrimPrefix(key, prefix)
		parts := strings.SplitN(remainder, "-", 2)
		if len(parts) < 1 {
			continue
		}

		num, err := strconv.Atoi(parts[0])
		if err != nil {
			// Not a numeric story number, skip
			continue
		}

		stories = append(stories, storyWithNum{key: key, num: num})
	}

	if len(stories) == 0 {
		return nil, fmt.Errorf("no stories found for epic: %s", epicID)
	}

	// Sort by story number
	sort.Slice(stories, func(i, j int) bool {
		return stories[i].num < stories[j].num
	})

	// Extract just the keys
	result := make([]string, len(stories))
	for i, s := range stories {
		result[i] = s.key
	}

	return result, nil
}

// GetAllEpics returns all epic IDs that are not done or deferred, sorted numerically.
//
// Epic IDs are matched using the pattern "epic-{N}" where N is numeric.
// Epics with status "done", "deferred", or "optional" are excluded.
// Results are sorted numerically by epic number.
//
// Returns an error if the file cannot be read.
func (r *Reader) GetAllEpics() ([]string, error) {
	sprintStatus, err := r.Read()
	if err != nil {
		return nil, err
	}

	// Collect all epic IDs that are not complete
	type epicWithNum struct {
		id  string
		num int
	}
	var epics []epicWithNum

	for key, status := range sprintStatus.DevelopmentStatus {
		// Match epic-N pattern
		if !strings.HasPrefix(key, "epic-") {
			continue
		}

		// Skip completed, deferred, or optional epics
		if status == StatusDone || status == "deferred" || status == "optional" {
			continue
		}

		// Extract epic number
		numStr := strings.TrimPrefix(key, "epic-")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			// Not a numeric epic ID, skip
			continue
		}

		epics = append(epics, epicWithNum{id: numStr, num: num})
	}

	if len(epics) == 0 {
		return nil, fmt.Errorf("no active epics found")
	}

	// Sort by epic number
	sort.Slice(epics, func(i, j int) bool {
		return epics[i].num < epics[j].num
	})

	// Extract just the IDs
	result := make([]string, len(epics))
	for i, e := range epics {
		result[i] = e.id
	}

	return result, nil
}
