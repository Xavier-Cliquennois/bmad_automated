# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
just build              # Build binary to ./bmad-automate
just test               # Run all tests
just test-verbose       # Run tests with verbose output
just test-pkg ./internal/claude  # Test specific package
just test-coverage      # Generate coverage.html
just lint               # Run golangci-lint
just check              # Run fmt, vet, and test
just run --help         # Build and run with arguments
```

## Architecture

This is a CLI tool that orchestrates Claude CLI to run automated development workflows. It spawns Claude as a subprocess, parses its streaming JSON output, and displays formatted results.

### Package Dependencies

```text
cmd/bmad-automate/main.go
         │
         ▼
    internal/cli (Cobra commands)
         │
         ├──► internal/workflow (orchestration)
         │         │
         │         ├──► internal/claude (Claude execution + JSON parsing)
         │         └──► internal/output (terminal formatting)
         │
         └──► internal/config (Viper configuration)
```

### Key Interfaces for Testing

- **`claude.Executor`** - Interface for running Claude CLI. Use `MockExecutor` in tests to avoid spawning real processes.
- **`output.Printer`** - Interface for terminal output. Use `NewPrinterWithWriter(buf)` to capture output in tests.

### Data Flow

1. CLI command receives story key
2. `config.Config.GetPrompt()` expands Go template with `{{.StoryKey}}`
3. `workflow.Runner` calls `claude.Executor.ExecuteWithResult()`
4. `claude.Parser` reads streaming JSON, emits `Event` structs
5. `output.Printer` formats and displays events

### Configuration

Workflow prompts are in `config/workflows.yaml` using Go templates. Config loads via Viper with env var overrides (`BMAD_` prefix).

### Claude CLI Integration

The executor always passes `--dangerously-skip-permissions` and `--output-format stream-json`. Each JSON line from stdout is parsed into `StreamEvent` structs, then converted to the higher-level `Event` type with convenience methods (`IsText()`, `IsToolUse()`, `IsToolResult()`).

## Versioning

**IMPORTANT**: Always update the version when committing changes. Follow [Semantic Versioning](https://semver.org/) (SemVer):

- **MAJOR** (X.0.0): Breaking changes / incompatible API modifications
- **MINOR** (x.Y.0): New features (`feat` commits) - backward compatible
- **PATCH** (x.y.Z): Bug fixes (`fix` commits) - backward compatible

Version file location: `internal/cli/version.go`

```go
const (
    Version     = "X.Y.Z"
    ReleaseDate = "YYYY-MM-DD"
    Features    = "Brief description of changes"
)
```

Update all three constants with each release.
