package client

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const echoToolFQN = "mcp__echo__ping"

func TestE2EHookAsyncDefaultContinue(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", echoToolFQN, 10).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hookCalled := make(chan *pb.HookInvocationRequest, 1)
	toolCalled := make(chan *pb.ToolInvocationRequest, 1)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookCalled <- req:
			default:
			}
			// Intentionally omit Continue_ to validate default semantics (continue=true).
			return &pb.HookOutput{
				Async_:         true,
				AsyncTimeoutMs: 25,
			}, nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			select {
			case toolCalled <- req:
			default:
			}
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	result, err := session.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run: %v (logs: %s)", err, harness.Logs())
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(2 * time.Second)
	defer waitCancel()

	select {
	case req := <-hookCalled:
		if req.GetHookEvent() != "PreToolUse" {
			t.Fatalf("expected hook_event PreToolUse, got %q", req.GetHookEvent())
		}
		if req.GetToolUseId() == "" {
			t.Fatalf("expected tool_use_id to be set")
		}
	case <-waitCtx.Done():
		t.Fatalf("expected hook callback (logs: %s)", harness.Logs())
	}

	select {
	case req := <-toolCalled:
		if req.GetToolFqn() != echoToolFQN {
			t.Fatalf("expected tool_fqn %q, got %q", echoToolFQN, req.GetToolFqn())
		}
	case <-waitCtx.Done():
		t.Fatalf("expected tool callback (logs: %s)", harness.Logs())
	}
}

func TestE2EHookDecisionBlockPreventsTool(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", echoToolFQN, 10).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hookCalled := make(chan struct{}, 1)
	toolCalled := make(chan struct{}, 1)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookCalled <- struct{}{}:
			default:
			}
			// Intentionally omit Continue_ and rely on decision="block" to block.
			return &pb.HookOutput{
				Decision:      "block",
				Reason:        "blocked by test",
				SystemMessage: "blocked by test",
			}, nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			select {
			case toolCalled <- struct{}{}:
			default:
			}
			return ToolResultText("unexpected"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	result, err := session.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run: %v (logs: %s)", err, harness.Logs())
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}

	select {
	case <-hookCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected hook callback (logs: %s)", harness.Logs())
	}

	select {
	case <-toolCalled:
		t.Fatalf("tool callback should not be invoked when hook blocks")
	case <-time.After(250 * time.Millisecond):
	}
}

func TestE2EPermissionUpdatedPermissionsAffectSubsequentTurns(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		EnablePermissionCallback(true).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var permCalls atomic.Int32
	var toolCalls atomic.Int32

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			permCalls.Add(1)
			decision, err := PermissionWithUpdatedPermissionsTyped(
				PermissionAllow("allowed by test"),
				[]PermissionUpdate{
					PermissionUpdateAddRules(
						"allow",
						"session",
						PermissionRule{ToolName: echoToolFQN},
					),
				},
			)
			if err != nil {
				return PermissionDeny(err.Error()), nil
			}
			return decision, nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			toolCalls.Add(1)
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	if _, err := session.Run(ctx, "first"); err != nil {
		t.Fatalf("first run: %v (logs: %s)", err, harness.Logs())
	}
	if _, err := session.Run(ctx, "second"); err != nil {
		t.Fatalf("second run: %v (logs: %s)", err, harness.Logs())
	}

	if got := permCalls.Load(); got != 1 {
		t.Fatalf("expected 1 permission callback after updated_permissions apply, got %d", got)
	}
	if got := toolCalls.Load(); got != 2 {
		t.Fatalf("expected 2 tool callbacks, got %d", got)
	}
}

func TestE2EPermissionAskConfirmTwoAttempts(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		EnablePermissionCallback(true).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	type permCall struct {
		attempt      uint32
		invocationID string
	}
	calls := make(chan permCall, 4)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			calls <- permCall{attempt: req.GetAttempt(), invocationID: req.GetInvocationId()}
			if req.GetAttempt() <= 1 {
				return PermissionAsk("confirm"), nil
			}
			return PermissionAllow("confirmed"), nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	if _, err := session.Run(ctx, "hello"); err != nil {
		t.Fatalf("run: %v (logs: %s)", err, harness.Logs())
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(2 * time.Second)
	defer waitCancel()

	var got []permCall
	for len(got) < 2 {
		select {
		case call := <-calls:
			got = append(got, call)
		case <-waitCtx.Done():
			t.Fatalf("expected 2 permission attempts, got %d (logs: %s)", len(got), harness.Logs())
		}
	}
	if got[0].attempt != 1 || got[1].attempt != 2 {
		t.Fatalf("expected attempts 1 then 2, got %#v", got)
	}
	if got[0].invocationID == "" || got[1].invocationID == "" || got[0].invocationID != got[1].invocationID {
		t.Fatalf("expected same non-empty invocation_id across attempts, got %#v", got)
	}
}

func TestE2EPermissionInterruptClosesSession(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		EnablePermissionCallback(true).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	toolCalled := make(chan struct{}, 1)
	closed := make(chan *pb.SessionClosed, 1)

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			decision := PermissionDeny("deny+interrupt")
			decision.Interrupt = true
			return decision, nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			select {
			case toolCalled <- struct{}{}:
			default:
			}
			return ToolResultText("unexpected"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	go func() {
		for event := range session.Events() {
			if sc := event.GetSessionClosed(); sc != nil {
				select {
				case closed <- sc:
				default:
				}
				return
			}
		}
	}()

	// The run should complete a turn, then the session should close due to interrupt.
	_, _ = session.Run(ctx, "hello")

	waitCtx, waitCancel := harness.ContextWithTimeout(3 * time.Second)
	defer waitCancel()

	select {
	case <-toolCalled:
		t.Fatalf("tool callback should not be invoked when permission denies")
	case <-time.After(250 * time.Millisecond):
	}

	select {
	case sc := <-closed:
		if sc.GetReason() != "permission_interrupted" {
			t.Fatalf("expected session_closed reason permission_interrupted, got %q", sc.GetReason())
		}
	case <-waitCtx.Done():
		t.Fatalf("expected session to close (logs: %s)", harness.Logs())
	}
}

func TestE2EPreToolUseUpdatedInputRewrite(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", echoToolFQN, 10).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			return HookWithSpecificOutput(
				HookContinue(),
				HookSpecificPreToolUseUpdatedInput(map[string]any{"text": "rewritten"}),
			)
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			got := req.GetToolInput().AsMap()
			if got["text"] != "rewritten" {
				t.Fatalf("expected rewritten tool input, got %#v", got)
			}
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	if _, err := session.Run(ctx, "hello"); err != nil {
		t.Fatalf("run: %v (logs: %s)", err, harness.Logs())
	}
}

func TestE2EPreToolUseHookInputContext(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", echoToolFQN, 10).
		PermissionMode("acceptEdits").
		Cwd("/tmp/echo-hook-test").
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hookInputs := make(chan map[string]any, 1)
	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookInputs <- StructToMap(req.GetInputData()):
			default:
			}
			return HookContinue(), nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	if _, err := session.Run(ctx, "hello"); err != nil {
		t.Fatalf("run: %v (logs: %s)", err, harness.Logs())
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(2 * time.Second)
	defer waitCancel()

	var input map[string]any
	select {
	case input = <-hookInputs:
	case <-waitCtx.Done():
		t.Fatalf("expected hook input data (logs: %s)", harness.Logs())
	}

	sessionID, ok := input["session_id"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("expected session_id to be set, got %#v", input["session_id"])
	}
	if input["hook_event_name"] != "PreToolUse" {
		t.Fatalf("expected hook_event_name PreToolUse, got %#v", input["hook_event_name"])
	}
	if input["tool_name"] != echoToolFQN {
		t.Fatalf("expected tool_name %q, got %#v", echoToolFQN, input["tool_name"])
	}
	if input["permission_mode"] != "acceptEdits" {
		t.Fatalf("expected permission_mode acceptEdits, got %#v", input["permission_mode"])
	}
	if input["cwd"] != "/tmp/echo-hook-test" {
		t.Fatalf("expected cwd /tmp/echo-hook-test, got %#v", input["cwd"])
	}
	if _, ok := input["transcript_path"]; !ok {
		t.Fatalf("expected transcript_path to be present")
	}
	toolInput, ok := input["tool_input"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool_input map, got %#v", input["tool_input"])
	}
	if toolInput["text"] != "ping" {
		t.Fatalf("expected tool_input text ping, got %#v", toolInput["text"])
	}
}

func TestE2EPreToolUseMatcherFilters(t *testing.T) {
	harness := newE2EHarness(t)
	if !shouldUseTestMode() {
		t.Skip("requires test mode; set SIDECAR_E2E_TEST_MODE=1")
	}

	ctx, cancel := harness.Context()
	defer cancel()

	options := NewOptions().
		WithClientHook("PreToolUse", "does_not_match", 10).
		Build()

	createResp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hookCalled := make(chan struct{}, 1)
	toolCalled := make(chan struct{}, 1)
	session, err := harness.AttachSession(ctx, createResp.SidecarSessionId, Handlers{
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			select {
			case hookCalled <- struct{}{}:
			default:
			}
			return HookContinue(), nil
		},
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			select {
			case toolCalled <- struct{}{}:
			default:
			}
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	if _, err := session.Run(ctx, "hello"); err != nil {
		t.Fatalf("run: %v (logs: %s)", err, harness.Logs())
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(2 * time.Second)
	defer waitCancel()

	select {
	case <-toolCalled:
	case <-waitCtx.Done():
		t.Fatalf("expected tool callback (logs: %s)", harness.Logs())
	}

	select {
	case <-hookCalled:
		t.Fatalf("expected hook to be filtered by matcher")
	case <-time.After(250 * time.Millisecond):
	}
}
