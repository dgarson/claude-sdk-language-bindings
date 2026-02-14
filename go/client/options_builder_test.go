package client

import (
	"testing"
)

func TestOptionsBuilderSetsFields(t *testing.T) {
	builder := NewOptions().
		ToolsPresetClaudeCode().
		AllowedTools("Read", "Edit").
		Model("claude-sonnet-4-5").
		MaxTurns(2).
		MaxBudgetUSD(1.25).
		Betas("context-1m-2025-08-07").
		IncludePartialMessages(true).
		ExtraArgBool("debug-to-stderr", true).
		WithMcpHttp("ext", "http://example", map[string]string{"x": "1"}).
		EnablePermissionCallback(true).
		WithClientHook("PreToolUse", "Bash", 10)

	if err := builder.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	options := builder.Build()
	if options.GetModel() != "claude-sonnet-4-5" {
		t.Fatalf("unexpected model: %q", options.GetModel())
	}
	if len(options.GetAllowedTools()) != 2 {
		t.Fatalf("expected allowed_tools size 2, got %d", len(options.GetAllowedTools()))
	}
	if options.GetMaxTurns() != 2 {
		t.Fatalf("expected max_turns=2, got %d", options.GetMaxTurns())
	}
	if options.GetMaxBudgetUsd() != 1.25 {
		t.Fatalf("expected max_budget_usd=1.25, got %v", options.GetMaxBudgetUsd())
	}
	if options.GetIncludePartialMessages() != true {
		t.Fatalf("expected include_partial_messages=true")
	}
	if options.GetExtraArgs()["debug-to-stderr"] == nil {
		t.Fatalf("expected extra_args debug-to-stderr to be set")
	}
	if options.GetMcpServers()["ext"] == nil || options.GetMcpServers()["ext"].GetHttp().GetUrl() != "http://example" {
		t.Fatalf("expected http mcp server ext to be set")
	}
	if !options.GetPermissionCallbackEnabled() {
		t.Fatalf("expected permission_callback_enabled=true")
	}
	if len(options.GetClientHooks()) != 1 {
		t.Fatalf("expected 1 client hook, got %d", len(options.GetClientHooks()))
	}
}
