package client

import (
	"testing"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestPermissionWithUpdatedPermissions(t *testing.T) {
	decision := PermissionAllow("ok")
	decision, err := PermissionWithUpdatedPermissions(decision, []map[string]any{
		{
			"type":        "setMode",
			"mode":        "acceptEdits",
			"destination": "session",
		},
	})
	if err != nil {
		t.Fatalf("PermissionWithUpdatedPermissions: %v", err)
	}
	if decision.GetUpdatedPermissions() == nil {
		t.Fatalf("expected UpdatedPermissions to be set")
	}
	raw := decision.GetUpdatedPermissions().AsInterface()
	list, ok := raw.([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("unexpected UpdatedPermissions: %#v", raw)
	}
}

func TestPermissionWithUpdatedPermissionsTyped(t *testing.T) {
	decision := PermissionAllow("ok")
	decision, err := PermissionWithUpdatedPermissionsTyped(decision, []PermissionUpdate{
		PermissionUpdateSetMode("acceptEdits", "session"),
	})
	if err != nil {
		t.Fatalf("PermissionWithUpdatedPermissionsTyped: %v", err)
	}
	if decision.GetUpdatedPermissions() == nil {
		t.Fatalf("expected UpdatedPermissions to be set")
	}
}

func TestPermissionSuggestionList(t *testing.T) {
	value, err := PermissionSuggestionsValue([]map[string]any{
		{"type": "addRules", "behavior": "allow"},
	})
	if err != nil {
		t.Fatalf("PermissionSuggestionsValue: %v", err)
	}
	req := &pb.PermissionDecisionRequest{PermissionSuggestions: value}
	list := PermissionSuggestionList(req)
	if len(list) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(list))
	}
}
