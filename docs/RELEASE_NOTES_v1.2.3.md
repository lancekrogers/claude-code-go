# v1.2.3 - Large Prompt Streaming Fix

This patch release makes `StreamPrompt` reliable for prompts that exceed the operating system's per-argument limit.

## Highlights

- Routes prompts at or above 100 KiB through stdin instead of passing them as a single argv element.
- Encodes `StreamJSONInput` prompts as valid user events on stdin.
- Adds regression coverage for threshold routing, stdin preservation, and stream-json input.

## Compatibility Notes

- Small text-input prompts retain the existing inline argv behavior.
- The public API is unchanged.

## Verification

- `just release check`
- `go test ./...`
- `go test -race ./pkg/claude`

## Full Changelog

**[v1.2.2...v1.2.3](https://github.com/lancekrogers/claude-code-go/compare/v1.2.2...v1.2.3)**
