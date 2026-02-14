package client

import (
	"testing"
)

func TestHookAsyncHelper(t *testing.T) {
	out := HookAsync(1234)
	if out.GetContinue_() != true {
		t.Fatalf("expected Continue_=true")
	}
	if out.GetAsync_() != true {
		t.Fatalf("expected Async_=true")
	}
	if out.GetAsyncTimeoutMs() != 1234 {
		t.Fatalf("expected AsyncTimeoutMs=1234, got %d", out.GetAsyncTimeoutMs())
	}
}

func TestHookBlockHelper(t *testing.T) {
	out := HookBlock("blocked", "warn")
	if out.GetContinue_() != false {
		t.Fatalf("expected Continue_=false")
	}
	if out.GetDecision() != "block" {
		t.Fatalf("expected Decision=block, got %q", out.GetDecision())
	}
	if out.GetReason() != "blocked" {
		t.Fatalf("expected Reason=blocked, got %q", out.GetReason())
	}
	if out.GetSystemMessage() != "warn" {
		t.Fatalf("expected SystemMessage=warn, got %q", out.GetSystemMessage())
	}
}

func TestHookWithSpecificOutput(t *testing.T) {
	out, err := HookWithSpecificOutput(HookContinue(), map[string]any{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": "nope",
	})
	if err != nil {
		t.Fatalf("HookWithSpecificOutput: %v", err)
	}
	if out.GetHookSpecificOutput() == nil {
		t.Fatalf("expected HookSpecificOutput to be set")
	}
	fields := out.GetHookSpecificOutput().AsMap()
	if fields["hookEventName"] != "PreToolUse" {
		t.Fatalf("unexpected hookEventName: %#v", fields["hookEventName"])
	}
}

func TestHookSpecificPreToolUseDecisionHelper(t *testing.T) {
	out, err := HookWithSpecificOutput(
		HookContinue(),
		HookSpecificPreToolUsePermissionDecision("deny", "nope"),
	)
	if err != nil {
		t.Fatalf("HookWithSpecificOutput: %v", err)
	}
	fields := out.GetHookSpecificOutput().AsMap()
	if fields["permissionDecision"] != "deny" {
		t.Fatalf("unexpected permissionDecision: %#v", fields["permissionDecision"])
	}
}

func TestHookWithSpecificTyped(t *testing.T) {
	out, err := HookWithSpecific(
		HookContinue(),
		PreToolUseHookSpecific{PermissionDecision: "deny", PermissionDecisionReason: "nope"},
	)
	if err != nil {
		t.Fatalf("HookWithSpecific: %v", err)
	}
	fields := out.GetHookSpecificOutput().AsMap()
	if fields["hookEventName"] != "PreToolUse" {
		t.Fatalf("unexpected hookEventName: %#v", fields["hookEventName"])
	}
}
