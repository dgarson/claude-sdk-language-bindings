## Flow Helpers (Go client)

These helpers are optional convenience layers for compiled clients. They only
construct protobuf messages and JSON-ish hook/permission payloads in the shapes
that Claude Code already expects. They do not change sidecar or SDK behavior.

### Options builder

Use `NewOptions()` for fluent construction of `ClaudeAgentOptions`:

- `ToolsList(...)` or `ToolsPresetClaudeCode()`
- `AllowedTools(...)`, `DisallowedTools(...)`
- `Model(...)`, `FallbackModel(...)`, `Betas(...)`, `MaxThinkingTokens(...)`
- `MaxTurns(...)`, `MaxBudgetUSD(...)`
- `WithMcpStdio(...)`, `WithMcpHttp(...)`, `WithMcpSse(...)`
- `WithClientTool(...)`, `WithClientHook(...)`
- `ExtraArgBool(...)`, `ExtraArgString(...)` (for forward-compatible CLI flags)

Call `Validate()` (or `Err()`) if you want to fail fast on conversion errors.

### Typed payload helpers

Hooks:
- `HookWithSpecific(..., PreToolUseHookSpecific{...})`
- Or map helpers in `hook_specific_output.go`

Permissions:
- `PermissionWithUpdatedPermissionsTyped(..., []PermissionUpdate{...})`
- `PermissionSuggestionsValueTyped(...)` / `PermissionUpdatesValueTyped(...)`

### Live E2E validation (hooks + permissions)

There is a live-only sidecar E2E test that attempts to trigger `PreToolUse` and
permission callbacks using a `Bash` tool call:

- `go test ./client -run TestLiveHooksAndPermissionsCallbacks -v`

It is skipped unless:

- `SIDECAR_E2E=1`
- `SIDECAR_E2E_LIVE=1`

You must also have Claude Code authenticated locally (e.g. via macOS Keychain or
your normal Claude Code login flow).
