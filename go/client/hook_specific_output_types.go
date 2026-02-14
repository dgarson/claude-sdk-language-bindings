package client

// Typed helpers for hookSpecificOutput.
//
// These types serialize into the same JSON-ish shapes Claude Code expects for
// hookSpecificOutput. They do not alter runtime behavior.

type HookSpecific interface {
	ToMap() map[string]any
}

type PreToolUseHookSpecific struct {
	PermissionDecision       string
	PermissionDecisionReason string
	UpdatedInput             map[string]any
}

func (s PreToolUseHookSpecific) ToMap() map[string]any {
	out := map[string]any{
		"hookEventName": "PreToolUse",
	}
	if s.PermissionDecision != "" {
		out["permissionDecision"] = s.PermissionDecision
	}
	if s.PermissionDecisionReason != "" {
		out["permissionDecisionReason"] = s.PermissionDecisionReason
	}
	if s.UpdatedInput != nil {
		out["updatedInput"] = s.UpdatedInput
	}
	return out
}

type PostToolUseHookSpecific struct {
	AdditionalContext string
}

func (s PostToolUseHookSpecific) ToMap() map[string]any {
	out := map[string]any{"hookEventName": "PostToolUse"}
	if s.AdditionalContext != "" {
		out["additionalContext"] = s.AdditionalContext
	}
	return out
}

type UserPromptSubmitHookSpecific struct {
	AdditionalContext string
}

func (s UserPromptSubmitHookSpecific) ToMap() map[string]any {
	out := map[string]any{"hookEventName": "UserPromptSubmit"}
	if s.AdditionalContext != "" {
		out["additionalContext"] = s.AdditionalContext
	}
	return out
}

type SessionStartHookSpecific struct {
	AdditionalContext string
}

func (s SessionStartHookSpecific) ToMap() map[string]any {
	out := map[string]any{"hookEventName": "SessionStart"}
	if s.AdditionalContext != "" {
		out["additionalContext"] = s.AdditionalContext
	}
	return out
}
