# v1.4.0 - Per-Run Environment Injection

This minor release adds per-run environment variables to `RunOptions`, so callers can set environment for a single `claude` child process without mutating the parent process environment.

## Highlights

- Adds `RunOptions.Env map[string]string`. Entries are merged over the inherited parent process environment, and a per-run key supersedes an inherited value so the child sees exactly one value per key.
- A nil or empty map leaves the child environment untouched, identical to prior behavior.
- Applies the override at every exec site that launches the CLI: `RunPromptCtx`, `RunFromStdinCtx`, the stream-json path, and the `dangerous` subpackage.
- Unions `DefaultOptions.Env` with a per-run `Env`, with the per-run map winning on conflict.
- Adds `ApplyEnv`, an exported helper the `dangerous` subpackage reuses for identical merge semantics.

## Compatibility Notes

- The public API is additive. Existing callers that never set `Env` keep the current behavior of inheriting the parent environment wholesale.
- The parent process environment is never modified.

## Verification

- `just release check`
- `go test ./...`
- `go test -race ./pkg/claude/...`

## Full Changelog

**[v1.3.0...v1.4.0](https://github.com/lancekrogers/claude-code-go/compare/v1.3.0...v1.4.0)**
