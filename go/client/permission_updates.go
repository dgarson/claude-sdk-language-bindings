package client

import (
	"fmt"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// PermissionSuggestionList converts a PermissionDecisionRequest's permission_suggestions
// into a generic list (best-effort).
func PermissionSuggestionList(req *pb.PermissionDecisionRequest) []any {
	if req == nil || req.PermissionSuggestions == nil {
		return nil
	}
	raw, ok := req.PermissionSuggestions.AsInterface().([]any)
	if !ok {
		return nil
	}
	return raw
}

// PermissionSuggestionsValue encodes a JSON list of permission update objects into
// a google.protobuf.Value suitable for PermissionDecisionRequest.permission_suggestions.
func PermissionSuggestionsValue(updates []map[string]any) (*structpb.Value, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	items := make([]any, 0, len(updates))
	for _, update := range updates {
		items = append(items, update)
	}
	return structpb.NewValue(items)
}

// PermissionUpdatesValue encodes a JSON list of permission update objects into
// a google.protobuf.Value suitable for PermissionDecision.updated_permissions.
func PermissionUpdatesValue(updates []map[string]any) (*structpb.Value, error) {
	return PermissionSuggestionsValue(updates)
}

func PermissionUpdatesValueTyped(updates []PermissionUpdate) (*structpb.Value, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	items := make([]map[string]any, 0, len(updates))
	for _, update := range updates {
		items = append(items, update.ToMap())
	}
	return PermissionUpdatesValue(items)
}

func PermissionSuggestionsValueTyped(updates []PermissionUpdate) (*structpb.Value, error) {
	return PermissionUpdatesValueTyped(updates)
}

func PermissionWithUpdatedPermissions(
	decision *pb.PermissionDecision, updates []map[string]any,
) (*pb.PermissionDecision, error) {
	if decision == nil {
		return nil, fmt.Errorf("decision is nil")
	}
	value, err := PermissionUpdatesValue(updates)
	if err != nil {
		return nil, err
	}
	decision.UpdatedPermissions = value
	return decision, nil
}

func PermissionWithUpdatedPermissionsTyped(
	decision *pb.PermissionDecision, updates []PermissionUpdate,
) (*pb.PermissionDecision, error) {
	if decision == nil {
		return nil, fmt.Errorf("decision is nil")
	}
	value, err := PermissionUpdatesValueTyped(updates)
	if err != nil {
		return nil, err
	}
	decision.UpdatedPermissions = value
	return decision, nil
}

func PermissionWithInterrupt(decision *pb.PermissionDecision, interrupt bool) *pb.PermissionDecision {
	if decision == nil {
		return nil
	}
	decision.Interrupt = interrupt
	return decision
}
