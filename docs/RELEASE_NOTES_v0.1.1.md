# v0.1.1 - Prompt Surface Refresh

This release updates `claude-code-go` to match the current Claude Code non-interactive `-p/--print` CLI surface more closely and removes drift from older flags that no longer exist in the upstream binary.

## Highlights

- Added current prompt-surface flags for agents, effort, settings, tools, plugin directories, budget limits, input format, hook events, partial messages, replayed user messages, debug files, bare mode, brief mode, beta headers, file resources, display names, and dynamic prompt exclusion.
- Stopped emitting removed flags such as `--permission-prompt-tool`, `--max-turns`, `--config`, `--disable-autoupdate`, and `--theme`.
- Wired plugin lifecycle hooks, tool-use callbacks, and shared budget tracking into both JSON and stream-json execution paths.
- Updated `SubagentManager` to execute through the real `--agent` and `--agents` prompt surface instead of a separate SDK-only shim.
- Rewrote the README and release notes to scope the SDK honestly to `claude -p`.

## Added Prompt Flags

`RunOptions` now covers the current wrapper-safe prompt flags below:

- `Agent`, `Agents`, `AgentsJSON`
- `Effort`
- `InputFormat`
- `IncludeHookEvents`
- `IncludePartialMessages`
- `ReplayUserMessages`
- `DebugFile`
- `Bare`
- `Brief`
- `Betas`
- `Files`
- `ExcludeDynamicSystemPromptSections`
- `AllowDangerouslySkipPermissions`
- `MaxBudgetUSD`
- `SettingSources`
- `Settings`
- `Tools`
- `Name`
- `PluginDirs`

## Compatibility Notes

- `PermissionTool`, `MaxTurns`, `ConfigFile`, `DisableAutoUpdate`, `Theme`, and `PermissionCallback` remain in `RunOptions` for source compatibility but are deprecated and ignored by argument construction.
- `PermissionModeDelegate` is now rejected during validation because the current Claude CLI no longer supports delegate permission mode.
- The SDK still wraps the prompt-oriented `claude -p` workflow. Interactive sessions and management commands such as `auth`, `mcp`, `plugins`, `install`, and `update` are intentionally not wrapped here.

## Verification

- `go test ./pkg/claude/...`
