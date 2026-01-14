# Enhancements to bmad-automate

This document describes the major enhancements added to bmad-automate CLI.

## Table of Contents

- [Automatic Rate Limit Handling](#automatic-rate-limit-handling)
- [Per-Workflow Model Configuration](#per-workflow-model-configuration)
- [Model Display in Dry-Run Mode](#model-display-in-dry-run-mode)
- [All-Epics Command](#all-epics-command)
- [Implementation Details](#implementation-details)

---

## Automatic Rate Limit Handling

### Problem

When executing multiple stories or long-running workflows, Claude API rate limits could cause workflow failures, requiring manual intervention and retry.

### Solution

Added automatic rate limit detection and retry functionality across all commands.

### Features

- **Intelligent Error Detection**: Parses Claude CLI stderr output to detect rate limit errors
- **Time Extraction**: Automatically extracts the reset time from error messages like:
  ```
  Claude usage limit reached. Your limit will reset at 1pm (Etc/GMT+5)
  ```
- **Automatic Retry**: With `--auto-retry` flag, the tool waits until the rate limit resets and automatically retries
- **User-Friendly Feedback**: Shows countdown and retry progress to the user

### Usage

All lifecycle commands now support the `--auto-retry` flag:

```bash
# Single story with auto-retry
bmad-automate run 3-1-welcome-message-display --auto-retry

# Multiple stories with auto-retry
bmad-automate queue 6-5 6-6 6-7 --auto-retry

# All stories in an epic with auto-retry
bmad-automate epic 6 --auto-retry

# All active epics with auto-retry
bmad-automate all-epics --auto-retry
```

### Commands Updated

- ✅ `run` - Single story lifecycle
- ✅ `queue` - Multiple stories
- ✅ `epic` - All stories in an epic
- ✅ `all-epics` - All active epics

### Implementation

**New Package**: `internal/ratelimit`
- `Detector` - Thread-safe rate limit error detection
- `ErrorInfo` - Parsed error information with reset time
- Time parsing with timezone support

**Shared Retry Logic**: `internal/cli/retry.go`
- `executeStoryWithRetry()` - Centralized retry logic used by all commands
- Up to 10 retry attempts with automatic waiting
- Graceful fallback to 5-minute wait if parsing fails

---

## Per-Workflow Model Configuration

### Problem

Different workflows have different complexity requirements. Story creation requires advanced reasoning (Opus), while implementation and review work well with faster, cost-effective models (Sonnet).

### Solution

Added per-workflow model configuration with sensible defaults.

### Configuration

**In `config/workflows.yaml`:**

```yaml
workflows:
  create-story:
    prompt_template: "..."
    model: "opus"  # Use Opus for complex story creation

  dev-story:
    prompt_template: "..."
    model: "sonnet"  # Use Sonnet for implementation

  code-review:
    prompt_template: "..."
    model: "sonnet"  # Use Sonnet for code review

  git-commit:
    prompt_template: "..."
    model: "sonnet"  # Use Sonnet for git operations

claude:
  default_model: "sonnet"  # Fallback if workflow doesn't specify
```

**Built-in Defaults:**

If no config file exists, bmad-automate uses these defaults:
- `create-story`: **opus** (complex reasoning for story planning)
- `dev-story`: **sonnet** (efficient implementation)
- `code-review`: **sonnet** (code analysis)
- `git-commit`: **sonnet** (simple git operations)

### Benefits

- **Cost Optimization**: Use expensive models only where needed
- **Performance**: Faster execution with Sonnet for routine tasks
- **Quality**: Opus provides better story creation and planning
- **Flexibility**: Override per workflow or set project-wide defaults

---

## Model Display in Dry-Run Mode

### Problem

Users couldn't preview which Claude model would be used for each workflow step before execution.

### Solution

Enhanced `--dry-run` mode to display the model for each workflow step.

### Usage

```bash
bmad-automate run 4-1-save-question-functionality --dry-run
```

**Output:**
```
Dry run for story 4-1-save-question-functionality:
  1. dev-story (sonnet) → review
  2. code-review (sonnet) → done
  3. git-commit (sonnet) → done
```

**With create-story workflow:**
```bash
bmad-automate epic 5 --dry-run
```

**Output:**
```
Dry run for epic 5:

Story 5-1-share-dialog-component:
  1. create-story (opus) → ready-for-dev
  2. dev-story (sonnet) → review
  3. code-review (sonnet) → done
  4. git-commit (sonnet) → done

Total: 4 workflows across 1 story
```

### Benefits

- **Transparency**: See exactly which model will be used
- **Cost Estimation**: Preview expensive Opus usage before execution
- **Verification**: Confirm model configuration is correct

---

## All-Epics Command

### Problem

Running multiple epics required manual execution of each epic separately, making it difficult to automate full project development.

### Solution

Added `all-epics` command to automatically process all active epics in sequence.

### Features

- **Smart Epic Discovery**: Automatically finds all epics that are not "done", "deferred", or "optional"
- **Sorted Execution**: Processes epics in numerical order (epic-3, epic-4, epic-5, etc.)
- **Story Completion**: Each story runs to completion before moving to the next
- **Progress Tracking**: Clear visual separators and progress indicators
- **Dry-Run Support**: Preview all workflows across all epics before execution

### Usage

```bash
# Preview all workflows
bmad-automate all-epics --dry-run

# Execute all active epics
bmad-automate all-epics

# With automatic rate limit handling
bmad-automate all-epics --auto-retry
```

**Example Output:**
```
Found 3 active epic(s): [3 4 5]

═══════════════════════════════════════════════════════════════════
  Epic 1 of 3: epic-3
═══════════════════════════════════════════════════════════════════

─── Story 1 of 2: 3-4-similar-saved-question-detection
[... workflow execution ...]
Story 3-4-similar-saved-question-detection completed successfully

Epic 3 completed (2 stories processed)

═══════════════════════════════════════════════════════════════════
  All 3 epic(s) completed successfully!
═══════════════════════════════════════════════════════════════════
```

### Epic Selection Logic

An epic is considered "active" if its status in `sprint-status.yaml` is:
- `backlog` - Not yet started
- `in-progress` - Currently being worked on

Epics are **excluded** if their status is:
- `done` - Completed
- `deferred` - Postponed
- `optional` - Not required

---

## Implementation Details

### Files Created

- `internal/ratelimit/detector.go` - Rate limit error detection and parsing
- `internal/ratelimit/detector_state.go` - Thread-safe state management
- `internal/cli/retry.go` - Shared retry logic for all commands
- `internal/cli/helpers.go` - Helper functions for model resolution
- `internal/cli/all_epics.go` - All-epics command implementation

### Files Modified

- `internal/claude/client.go` - Added model parameter support
- `internal/config/types.go` - Added Model field and default models
- `internal/router/lifecycle.go` - Added Model field to LifecycleStep
- `internal/workflow/workflow.go` - Model resolution logic
- `internal/cli/run.go` - Added --auto-retry flag
- `internal/cli/queue.go` - Added --auto-retry flag
- `internal/cli/epic.go` - Added --auto-retry flag
- `internal/cli/root.go` - Integrated RateLimitDetector
- `internal/status/reader.go` - Added GetAllEpics() method
- `config/workflows.yaml` - Added model configuration

### Architecture Patterns

**Dependency Injection**: All new features use the existing `App` struct pattern for testability

**Interface-Based Design**: Rate limit detector integrated through callback interfaces

**Shared Logic**: Retry logic extracted to prevent code duplication across commands

**Configuration Hierarchy**: Model selection follows: workflow-specific → default_model → empty string (Claude CLI default)

---

## Testing

All enhancements have been tested with:

- ✅ Single story execution (`run`)
- ✅ Multiple story batches (`queue`)
- ✅ Epic processing (`epic`)
- ✅ All epics processing (`all-epics`)
- ✅ Dry-run mode for all commands
- ✅ Rate limit detection and retry
- ✅ Model configuration and display

---

## Benefits Summary

### For Users

- **Unattended Execution**: No manual intervention needed when rate limits are hit
- **Cost Control**: Optimize Claude API costs with per-workflow model selection
- **Better Planning**: Preview models and workflows before execution
- **Automation**: Process entire projects with a single command

### For Developers

- **Clean Architecture**: Shared retry logic, no code duplication
- **Testability**: All new features follow dependency injection patterns
- **Extensibility**: Easy to add new commands or modify retry behavior
- **Documentation**: Clear code comments and structure

---

## Migration Guide

### Existing Users

**No Breaking Changes** - All existing workflows continue to work:

- Default configuration includes model settings
- Commands work without `--auto-retry` (fail-fast behavior preserved)
- Dry-run output enhanced but backward compatible

### Recommended Updates

1. **Add model configuration** to your `config/workflows.yaml`:
   ```yaml
   claude:
     default_model: "sonnet"

   workflows:
     create-story:
       model: "opus"
   ```

2. **Use --auto-retry** for long-running workflows:
   ```bash
   bmad-automate all-epics --auto-retry
   ```

3. **Preview with dry-run** before executing:
   ```bash
   bmad-automate epic 6 --dry-run
   ```

---

## Future Enhancements

Potential improvements for future releases:

- [ ] Custom retry strategies (exponential backoff, max wait time)
- [ ] Parallel epic execution
- [ ] Rate limit quota tracking and warnings
- [ ] Model cost estimation in dry-run mode
- [ ] Resume from failure (checkpoint/resume functionality)

---

## Contributors

This enhancement was developed with contributions from:
- Xavier Cliquennois (@xaviercliquennois)

---

## License

Same as bmad_automated project license.
