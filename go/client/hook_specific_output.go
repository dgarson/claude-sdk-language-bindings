package client

// Convenience constructors for hookSpecificOutput payloads.
//
// These helpers only build the JSON-ish maps that Claude Code expects for hooks.
// They do not change sidecar or SDK behavior; they just reduce boilerplate for
// compiled clients.

func HookSpecificPreToolUsePermissionDecision(decision string, reason string) map[string]any {
	return PreToolUseHookSpecific{
		PermissionDecision:       decision,
		PermissionDecisionReason: reason,
	}.ToMap()
}

func HookSpecificPreToolUseUpdatedInput(updatedInput map[string]any) map[string]any {
	return PreToolUseHookSpecific{UpdatedInput: updatedInput}.ToMap()
}

func HookSpecificPostToolUseAdditionalContext(additionalContext string) map[string]any {
	return PostToolUseHookSpecific{AdditionalContext: additionalContext}.ToMap()
}

func HookSpecificUserPromptSubmitAdditionalContext(additionalContext string) map[string]any {
	return UserPromptSubmitHookSpecific{AdditionalContext: additionalContext}.ToMap()
}

func HookSpecificSessionStartAdditionalContext(additionalContext string) map[string]any {
	return SessionStartHookSpecific{AdditionalContext: additionalContext}.ToMap()
}
