package client

import "testing"

func TestToolResultBlocksAndStream(t *testing.T) {
	result := ToolResultBlocks([]ContentBlock{
		ContentBlockText("hello"),
		ContentBlockJSON(map[string]any{"ok": true}),
	}, false, map[string]any{"meta": map[string]any{"k": "v"}})

	if result == nil {
		t.Fatalf("expected result struct")
	}
	content := result.AsMap()["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}
	if content[0].(map[string]any)["type"] != "text" {
		t.Fatalf("expected text block")
	}
	if content[1].(map[string]any)["type"] != "json" {
		t.Fatalf("expected json block")
	}
	meta := result.AsMap()["meta"].(map[string]any)
	if meta["k"] != "v" {
		t.Fatalf("expected meta to be preserved")
	}
}
