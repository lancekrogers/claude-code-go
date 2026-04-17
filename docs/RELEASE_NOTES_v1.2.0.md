# v1.2.0 - Prompt Surface Refresh

This release aligns the SDK with the current wrapper-safe `claude -p` surface.

## Highlights

- Added current prompt-surface flags for agents, effort, settings, tools, plugin directories, budget limits, input format, hook events, partial messages, replayed user messages, debug files, bare mode, brief mode, beta headers, file resources, display names, and dynamic prompt exclusion.
- Stopped emitting removed flags such as `--permission-prompt-tool`, `--max-turns`, `--config`, `--disable-autoupdate`, and `--theme`.
- Wired plugin lifecycle hooks, tool-use callbacks, and shared budget tracking into both JSON and stream-json execution paths.
- Updated `SubagentManager` to execute through the real `--agent` and `--agents` prompt surface instead of a separate SDK-only shim.
- Rewrote the README and examples to scope the SDK honestly to `claude -p`.

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

## Behavior Changes

- `SubagentConfig.ToRunOptions` now targets Claude's native `--agent` and `--agents` prompt surface. The returned `RunOptions` keeps the subagent definition in `Agents` and selects it with `Agent`; it no longer flattens the subagent prompt and tool list into top-level `SystemPrompt` and `AllowedTools`.
- Deprecated top-level fields retained for source compatibility now emit one-time warnings when set, instead of being silently ignored at argv construction time.

## Compatibility Notes

- `PermissionTool`, `MaxTurns`, `ConfigFile`, `DisableAutoUpdate`, `Theme`, and `PermissionCallback` remain in `RunOptions` for source compatibility. `PreprocessOptions` emits one-time warnings when they are set, and argv construction still ignores them because the current Claude CLI no longer supports them.
- `PermissionModeDelegate` is now rejected during validation because the current Claude CLI no longer supports delegate permission mode.
- The SDK still wraps the prompt-oriented `claude -p` workflow. Interactive sessions and management commands such as `auth`, `mcp`, `plugins`, `install`, and `update` are intentionally not wrapped here.

## Verification

- `just release check`

## Full Changelog

**[v1.1.0...v1.2.0](https://github.com/lancekrogers/claude-code-go/compare/v1.1.0...v1.2.0)**
