package client

import "testing"

func TestValidateJSONSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	}

	if err := ValidateJSONSchema(schema, map[string]any{"name": "ok"}); err != nil {
		t.Fatalf("expected valid schema, got %v", err)
	}
	if err := ValidateJSONSchema(schema, map[string]any{"name": 3}); err == nil {
		t.Fatalf("expected schema validation error")
	}
}
