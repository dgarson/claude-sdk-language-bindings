package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSidecarE2E(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	createResp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	toolCalled := make(chan struct{}, 1)
	handlers := Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			select {
			case toolCalled <- struct{}{}:
			default:
			}
			return ToolResultText("ok"), nil
		},
	}

	session, err := harness.Client().AttachSession(ctx, createResp.SidecarSessionId, ClientInfo{
		Name:    "e2e",
		Version: "test",
	}, handlers)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer session.Close()

	result, err := session.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	if result.Assistant() == nil {
		t.Fatalf("expected assistant message, got %+v", result.Turn)
	}
	if shouldUseTestMode() {
		if result.Result() == nil || result.Result().Result != "ok" {
			t.Fatalf("expected result ok, got %+v", result)
		}
	}

	if shouldUseTestMode() {
		select {
		case <-toolCalled:
		case <-time.After(2 * time.Second):
			t.Fatalf("tool handler was not invoked")
		}
	} else {
		select {
		case <-toolCalled:
		case <-time.After(2 * time.Second):
			t.Log("tool handler was not invoked in live mode")
		}
	}
}
