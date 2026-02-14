package client

import (
	"context"
	"testing"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestAskConfirmHandlerFlow(t *testing.T) {
	handler := AskConfirmHandler(
		func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			return PermissionAsk("needs confirmation"), nil
		},
		func(ctx context.Context, prompt string, req *pb.PermissionDecisionRequest) (bool, string, error) {
			if prompt == "" {
				t.Fatalf("expected prompt text")
			}
			return true, "approved", nil
		},
	)

	first, err := handler(context.Background(), &pb.PermissionDecisionRequest{
		ToolName: "mcp__echo__ping",
		Attempt:  1,
	})
	if err != nil {
		t.Fatalf("first attempt: %v", err)
	}
	if first.Behavior != "ask" {
		t.Fatalf("expected ask, got %q", first.Behavior)
	}

	second, err := handler(context.Background(), &pb.PermissionDecisionRequest{
		ToolName: "mcp__echo__ping",
		Attempt:  2,
	})
	if err != nil {
		t.Fatalf("second attempt: %v", err)
	}
	if second.Behavior != "allow" {
		t.Fatalf("expected allow, got %q", second.Behavior)
	}
	if second.Reason != "approved" {
		t.Fatalf("expected reason to be propagated, got %q", second.Reason)
	}
}
