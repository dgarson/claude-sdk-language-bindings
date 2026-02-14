package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// TestLivePermissionCallback verifies that permission callbacks are invoked in
// live mode. Uses the Write tool which always triggers a permission prompt in
// default mode.
//
// Important: The Claude Code CLI only invokes can_use_tool for tools that
// require permission. Simple Bash commands (echo hello) are auto-approved and
// will NOT trigger the callback. Write always requires permission.
func TestLivePermissionCallback(t *testing.T) {
	harness := newE2EHarness(t)
	if shouldUseTestMode() {
		t.Skip("requires live mode; set SIDECAR_E2E_LIVE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", "Write", 30).
		EnablePermissionCallback(true).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	permCalled := make(chan struct{}, 1)
	hookCalled := make(chan struct{}, 1)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookCalled <- struct{}{}:
			default:
			}
			return &pb.HookOutput{
				Continue_: boolPtr(true),
			}, nil
		},
		Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			select {
			case permCalled <- struct{}{}:
			default:
			}
			return PermissionAllow("allowed by test"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	// Write tool always triggers a permission prompt in default mode.
	result, err := session.Run(ctx, "Write the text 'hello' to /tmp/sidecar_go_perm_test.txt using the Write tool. Do nothing else.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(30 * time.Second)
	defer waitCancel()

	select {
	case <-permCalled:
	case <-waitCtx.Done():
		t.Fatalf("expected permission callback to be invoked (logs: %s)", harness.Logs())
	}
	select {
	case <-hookCalled:
	case <-waitCtx.Done():
		t.Fatalf("expected hook callback to be invoked (logs: %s)", harness.Logs())
	}
}

// TestLiveHookCallback verifies that hook callbacks work in live mode even for
// tools that are auto-approved (like simple Bash commands).
func TestLiveHookCallback(t *testing.T) {
	harness := newE2EHarness(t)
	if shouldUseTestMode() {
		t.Skip("requires live mode; set SIDECAR_E2E_LIVE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", "Bash", 30).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hookCalled := make(chan struct{}, 1)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookCalled <- struct{}{}:
			default:
			}
			return &pb.HookOutput{
				Continue_: boolPtr(true),
				Reason:    "ok",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	result, err := session.Run(ctx, "Use the Bash tool to run: echo hello world. Do nothing else.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(30 * time.Second)
	defer waitCancel()

	select {
	case <-hookCalled:
	case <-waitCtx.Done():
		t.Fatalf("expected hook callback to be invoked (logs: %s)", harness.Logs())
	}
}

// TestLiveHooksAndPermissionsCallbacks verifies both hooks and permissions work
// together in live mode. Uses the Write tool to reliably trigger both.
func TestLiveHooksAndPermissionsCallbacks(t *testing.T) {
	harness := newE2EHarness(t)
	if shouldUseTestMode() {
		t.Skip("requires live mode; set SIDECAR_E2E_LIVE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", "Write", 30).
		EnablePermissionCallback(true).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hookCalled := make(chan struct{}, 1)
	permCalled := make(chan struct{}, 1)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookCalled <- struct{}{}:
			default:
			}
			return &pb.HookOutput{
				Continue_:     boolPtr(true),
				SystemMessage: "hook observed",
				Reason:        "ok",
			}, nil
		},
		Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			select {
			case permCalled <- struct{}{}:
			default:
			}
			decision := PermissionAllow("allowed by test")
			decision, err := PermissionWithUpdatedPermissionsTyped(decision, []PermissionUpdate{
				PermissionUpdateSetMode("acceptEdits", "session"),
			})
			if err != nil {
				return PermissionDeny(err.Error()), nil
			}
			decision.UpdatedInput = &structpb.Struct{}
			return decision, nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	// Write tool triggers both PreToolUse hook and permission callback.
	result, err := session.Run(ctx, "Write the text 'test' to /tmp/sidecar_go_hooks_perm_test.txt using the Write tool. Do nothing else.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(30 * time.Second)
	defer waitCancel()

	select {
	case <-hookCalled:
	case <-waitCtx.Done():
		t.Fatalf("expected hook callback to be invoked (logs: %s)", harness.Logs())
	}
	select {
	case <-permCalled:
	case <-waitCtx.Done():
		t.Fatalf("expected permission callback to be invoked (logs: %s)", harness.Logs())
	}
}
