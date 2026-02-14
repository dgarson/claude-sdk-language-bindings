package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func TestIncludePartialsTrueEmitsStreamEvent(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	options := &pb.ClaudeAgentOptions{IncludePartialMessages: true}
	resp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	stream, err := session.Stream(ctx, "hello")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	partialCtx, partialCancel := harness.ContextWithTimeout(5 * time.Second)
	defer partialCancel()
	partial, err := waitForPartial(partialCtx, stream.Partials())
	if err != nil {
		t.Fatalf("wait for partial: %v", err)
	}
	if partial == nil || !partial.GetIsPartial() || partial.GetStreamEvent() == nil {
		t.Fatalf("expected partial stream_event message, got %+v", partial)
	}

	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("stream result: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	assertTurnCorrelation(t, result.Turn, stream.RequestID())
	assertNoSidecarErrors(t, result.Turn.Errors)
}

func TestIncludePartialsFalseSuppressesStreamEvent(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	stream, err := session.Stream(ctx, "hello")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("stream result: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	assertTurnCorrelation(t, result.Turn, stream.RequestID())
	assertNoSidecarErrors(t, result.Turn.Errors)

	partialCtx, partialCancel := harness.ContextWithTimeout(200 * time.Millisecond)
	defer partialCancel()
	if err := expectNoPartial(partialCtx, stream.Partials()); err != nil {
		t.Fatalf("expected no partials, got %v", err)
	}
}

func TestInputStreamWithTwoUserMessages(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	requestID, streamID, err := session.StartInputStream(ctx)
	if err != nil {
		t.Fatalf("start input stream: %v", err)
	}

	turnCtx, turnCancel := harness.ContextWithTimeout(5 * time.Second)
	defer turnCancel()
	turns := make(chan *Turn, 1)
	turnErrs := make(chan error, 1)
	go func() {
		turn, err := waitForTurn(turnCtx, session, requestID)
		if err != nil {
			turnErrs <- err
			return
		}
		turns <- turn
	}()

	if err := session.SendInputEvent(ctx, streamID, UserTextEvent("first")); err != nil {
		t.Fatalf("send input event 1: %v", err)
	}
	if err := session.SendInputEvent(ctx, streamID, UserTextEvent("second")); err != nil {
		t.Fatalf("send input event 2: %v", err)
	}
	if err := session.EndInputStream(ctx, streamID); err != nil {
		t.Fatalf("end input stream: %v", err)
	}

	select {
	case err := <-turnErrs:
		t.Fatalf("wait for turn: %v", err)
	case turn := <-turns:
		if turn == nil {
			t.Fatalf("expected turn result")
		}
		assertTurnCorrelation(t, turn, requestID)
		assertNoSidecarErrors(t, turn.Errors)
		userTexts := []string{}
		for _, message := range turn.Messages {
			user := message.GetUser()
			if user == nil || len(user.Content) == 0 {
				continue
			}
			text := user.Content[0].GetText()
			if text != nil {
				userTexts = append(userTexts, text.Text)
			}
		}
		if len(userTexts) != 2 {
			t.Fatalf("expected 2 user messages, got %d", len(userTexts))
		}
		if userTexts[0] != "first" || userTexts[1] != "second" {
			t.Fatalf("expected user messages in order, got %v", userTexts)
		}
	}
}

func TestInputStreamEndWaitsForFirstResult(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	requestID, streamID, err := session.StartInputStream(ctx)
	if err != nil {
		t.Fatalf("start input stream: %v", err)
	}
	if err := session.SendInputEvent(ctx, streamID, UserTextEvent("first")); err != nil {
		t.Fatalf("send input event: %v", err)
	}
	if err := session.EndInputStream(ctx, streamID); err != nil {
		t.Fatalf("end input stream: %v", err)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer waitCancel()

	events, errors, err := collectUntilTurnEnd(waitCtx, session.Events(), requestID)
	if err != nil {
		t.Fatalf("wait for turn end: %v", err)
	}
	assertNoSidecarErrors(t, errors)
	for _, event := range events {
		if event.GetSessionClosed() != nil {
			t.Fatalf("unexpected SessionClosed before first result")
		}
	}
	foundResult := false
	for _, event := range events {
		message := event.GetMessage()
		if message != nil && message.GetResult() != nil {
			foundResult = true
			break
		}
	}
	if !foundResult {
		t.Fatalf("expected ResultMessage before turn end")
	}
}

func TestStreamEventParentToolUseIDPropagation(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	options := &pb.ClaudeAgentOptions{IncludePartialMessages: true}
	resp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	result, err := session.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	assertNoSidecarErrors(t, result.Turn.Errors)

	toolUseID := ""
	for _, message := range result.Turn.Messages {
		assistant := message.GetAssistant()
		if assistant == nil {
			continue
		}
		for _, block := range assistant.Content {
			if toolUse := block.GetToolUse(); toolUse != nil {
				toolUseID = toolUse.Id
				break
			}
		}
		if toolUseID != "" {
			break
		}
	}
	if toolUseID == "" {
		t.Fatalf("expected tool_use_id to be set")
	}

	streamParent := ""
	for _, partial := range result.Turn.Partials {
		streamEvent := partial.GetStreamEvent()
		if streamEvent == nil {
			continue
		}
		streamParent = streamEvent.GetParentToolUseId()
		if streamParent != "" {
			break
		}
	}
	if streamParent == "" {
		t.Fatalf("expected stream_event parent_tool_use_id to be set")
	}
	if streamParent != toolUseID {
		t.Fatalf("expected parent_tool_use_id %q, got %q", toolUseID, streamParent)
	}
}

func TestPartialAssistantContentOrdering(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	options := &pb.ClaudeAgentOptions{IncludePartialMessages: true}
	resp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	result, err := session.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	assertNoSidecarErrors(t, result.Turn.Errors)

	partialIndex := findMessageEventIndex(result.Turn.Events, func(message *pb.MessageEvent) bool {
		if !message.GetIsPartial() {
			return false
		}
		assistant := message.GetAssistant()
		if assistant == nil {
			return false
		}
		for _, block := range assistant.Content {
			if text := block.GetText(); text != nil && text.Text == "Echo tool result received" {
				return true
			}
		}
		return false
	})
	if partialIndex == -1 {
		t.Fatalf("expected partial assistant content block")
	}

	finalIndex := findMessageEventIndex(result.Turn.Events, func(message *pb.MessageEvent) bool {
		if message.GetIsPartial() {
			return false
		}
		assistant := message.GetAssistant()
		if assistant == nil {
			return false
		}
		for _, block := range assistant.Content {
			if text := block.GetText(); text != nil && text.Text == "Echo tool result received" {
				return true
			}
		}
		return false
	})
	if finalIndex == -1 {
		t.Fatalf("expected final assistant content block")
	}
	if partialIndex >= finalIndex {
		t.Fatalf("expected partial content before final assistant message")
	}
}

func TestStreamInputControlResponseInterleaving(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	toolCalled := make(chan struct{}, 1)
	releaseTool := make(chan struct{})
	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			select {
			case toolCalled <- struct{}{}:
			default:
			}
			<-releaseTool
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	requestID, streamID, err := session.StartInputStream(ctx)
	if err != nil {
		t.Fatalf("start input stream: %v", err)
	}
	if err := session.SendInputEvent(ctx, streamID, UserTextEvent("first")); err != nil {
		t.Fatalf("send input event 1: %v", err)
	}

	select {
	case <-toolCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("tool callback not triggered")
	}

	if err := session.SendInputEvent(ctx, streamID, UserTextEvent("second")); err != nil {
		t.Fatalf("send input event 2: %v", err)
	}
	close(releaseTool)
	if err := session.EndInputStream(ctx, streamID); err != nil {
		t.Fatalf("end input stream: %v", err)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer waitCancel()
	events, errors, err := collectUntilTurnEnd(waitCtx, session.Events(), requestID)
	if err != nil {
		t.Fatalf("wait for turn end: %v", err)
	}
	assertNoSidecarErrors(t, errors)

	toolUseIndex := findMessageEventIndex(events, func(message *pb.MessageEvent) bool {
		assistant := message.GetAssistant()
		if assistant == nil {
			return false
		}
		for _, block := range assistant.Content {
			if block.GetToolUse() != nil {
				return true
			}
		}
		return false
	})
	if toolUseIndex == -1 {
		t.Fatalf("expected tool_use message")
	}

	secondUserIndex := findMessageEventIndex(events, func(message *pb.MessageEvent) bool {
		user := message.GetUser()
		if user == nil || len(user.Content) == 0 {
			return false
		}
		text := user.Content[0].GetText()
		return text != nil && text.Text == "second"
	})
	if secondUserIndex == -1 {
		t.Fatalf("expected second user message")
	}

	toolResultIndex := findMessageEventIndex(events, func(message *pb.MessageEvent) bool {
		assistant := message.GetAssistant()
		if assistant == nil {
			return false
		}
		for _, block := range assistant.Content {
			if block.GetToolResult() != nil {
				return true
			}
		}
		return false
	})
	if toolResultIndex == -1 {
		t.Fatalf("expected tool_result message")
	}

	if !(toolUseIndex < secondUserIndex && secondUserIndex < toolResultIndex) {
		t.Fatalf(
			"expected user message between tool_use and tool_result, got tool_use=%d user=%d tool_result=%d",
			toolUseIndex,
			secondUserIndex,
			toolResultIndex,
		)
	}
}

func TestInputStreamErrorHandling(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	_, streamID, err := session.StartInputStream(ctx)
	if err != nil {
		t.Fatalf("start input stream: %v", err)
	}
	if err := session.SendInputMap(ctx, streamID, map[string]any{"message": "bad"}); err != nil {
		t.Fatalf("send input event: %v", err)
	}
	if err := session.EndInputStream(ctx, streamID); err != nil {
		t.Fatalf("end input stream: %v", err)
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer waitCancel()
	_, sidecarErr, err := waitForSidecarError(waitCtx, session.Events())
	if err != nil {
		t.Fatalf("wait for sidecar error: %v", err)
	}
	if sidecarErr.GetCode() != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST error, got %q", sidecarErr.GetCode())
	}
}

func TestMultipleConcurrentTurnsWithStreaming(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	options := &pb.ClaudeAgentOptions{IncludePartialMessages: true}
	resp, err := harness.CreateSessionWithOptions(ctx, pb.SessionMode_INTERACTIVE, options)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer session.Close()

	streamA, err := session.Stream(ctx, "alpha")
	if err != nil {
		t.Fatalf("stream alpha: %v", err)
	}
	streamB, err := session.Stream(ctx, "bravo")
	if err != nil {
		t.Fatalf("stream bravo: %v", err)
	}

	type streamResult struct {
		id     string
		result *RunResult
		err    error
	}
	results := make(chan streamResult, 2)

	go func() {
		res, err := streamA.Result(ctx)
		results <- streamResult{id: "alpha", result: res, err: err}
	}()
	go func() {
		res, err := streamB.Result(ctx)
		results <- streamResult{id: "bravo", result: res, err: err}
	}()

	expectedRequestIDs := map[string]string{
		"alpha": streamA.RequestID(),
		"bravo": streamB.RequestID(),
	}

	found := map[string]*RunResult{}
	for i := 0; i < 2; i++ {
		out := <-results
		if out.err != nil {
			t.Fatalf("stream %s result: %v", out.id, out.err)
		}
		if out.result == nil || out.result.Turn == nil {
			t.Fatalf("stream %s missing turn result", out.id)
		}
		assertTurnCorrelation(t, out.result.Turn, expectedRequestIDs[out.id])
		assertNoSidecarErrors(t, out.result.Turn.Errors)
		found[out.id] = out.result
	}

	alphaTexts := turnUserTexts(found["alpha"].Turn)
	bravoTexts := turnUserTexts(found["bravo"].Turn)
	if len(alphaTexts) == 0 || alphaTexts[0] != "alpha" {
		t.Fatalf("expected alpha stream user message, got %v", alphaTexts)
	}
	if len(bravoTexts) == 0 || bravoTexts[0] != "bravo" {
		t.Fatalf("expected bravo stream user message, got %v", bravoTexts)
	}
}
