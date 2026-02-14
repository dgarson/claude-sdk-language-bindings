package client

import (
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestCreateSessionReturnsIDs(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if resp.GetSidecarSessionId() == "" {
		t.Fatalf("expected sidecar_session_id to be set")
	}
	if resp.GetClaudeSessionId() == "" {
		t.Log("claude_session_id not set (best-effort)")
	}
}

func TestListSessionsIncludesActiveSession(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	listResp, err := harness.Client().ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if findSessionSummary(listResp.GetSessions(), resp.GetSidecarSessionId()) == nil {
		t.Fatalf("expected session %q in list", resp.GetSidecarSessionId())
	}
}

func TestGetSessionReturnsSummary(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	getResp, err := harness.Client().GetSession(ctx, resp.GetSidecarSessionId())
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	summary := getResp.GetSession()
	if summary == nil {
		t.Fatalf("expected session summary")
	}
	if summary.GetSidecarSessionId() != resp.GetSidecarSessionId() {
		t.Fatalf(
			"expected sidecar_session_id %q, got %q",
			resp.GetSidecarSessionId(),
			summary.GetSidecarSessionId(),
		)
	}
	if summary.GetMode() != pb.SessionMode_INTERACTIVE {
		t.Fatalf("expected mode %v, got %v", pb.SessionMode_INTERACTIVE, summary.GetMode())
	}
	if summary.GetCreatedAt() == nil {
		t.Fatalf("expected created_at to be set")
	}
}

func TestDeleteSessionClosesSession(t *testing.T) {
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

	deleteResp, err := harness.Client().DeleteSession(ctx, resp.GetSidecarSessionId(), true)
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if !deleteResp.GetSuccess() {
		t.Fatalf("expected delete success")
	}

	waitCtx, waitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer waitCancel()

	events, errors, err := collectUntilSessionClosed(waitCtx, session.Events())
	if err != nil {
		t.Fatalf("wait for session closed: %v", err)
	}
	assertNoSidecarErrors(t, errors)
	assertSessionEventIDs(t, events, resp.GetSidecarSessionId())
}

func TestOneShotAutoCloses(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	resp, err := harness.CreateSession(ctx, pb.SessionMode_ONE_SHOT)
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
	requestID := stream.RequestID()

	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("stream result: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	assertTurnCorrelation(t, result.Turn, requestID)
	assertNoSidecarErrors(t, result.Turn.Errors)

	waitCtx, waitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer waitCancel()

	events, errors, err := collectUntilSessionClosed(waitCtx, session.Events())
	if err != nil {
		t.Fatalf("wait for session closed: %v", err)
	}
	assertNoSidecarErrors(t, errors)
	assertSessionEventIDs(t, events, resp.GetSidecarSessionId())
	if !sawTurnEndBeforeSessionClosed(events) {
		t.Fatalf("expected TURN_END before SessionClosed")
	}
}

func TestForkSessionWithNewOptions(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	baseResp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	baseSession, err := harness.AttachSession(ctx, baseResp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer baseSession.Close()

	initCtx, initCancel := harness.ContextWithTimeout(5 * time.Second)
	defer initCancel()
	baseInit, baseErrors, err := waitForSessionInit(initCtx, baseSession.Events())
	if err != nil {
		t.Fatalf("wait for session init: %v", err)
	}
	assertNoSidecarErrors(t, baseErrors)
	if baseInit.GetClaudeSessionId() == "" {
		t.Fatalf("expected claude_session_id to be set")
	}

	options := e2eTestOptions(shouldUseTestMode())
	if options == nil {
		options = &pb.ClaudeAgentOptions{}
	}
	options.IncludePartialMessages = true

	forkResp, err := harness.Client().ForkSession(ctx, &pb.ForkSessionRequest{
		SidecarSessionId: baseResp.GetSidecarSessionId(),
		Options:          options,
	})
	if err != nil {
		t.Fatalf("fork session: %v", err)
	}
	if forkResp.GetSidecarSessionId() == "" {
		t.Fatalf("expected forked sidecar_session_id to be set")
	}
	if forkResp.GetSidecarSessionId() == baseResp.GetSidecarSessionId() {
		t.Fatalf("expected forked session to have a new sidecar_session_id")
	}
	if forkResp.GetClaudeSessionId() != baseInit.GetClaudeSessionId() {
		t.Fatalf(
			"expected forked claude_session_id %q, got %q",
			baseInit.GetClaudeSessionId(),
			forkResp.GetClaudeSessionId(),
		)
	}

	forkSession, err := harness.AttachSession(ctx, forkResp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach forked session: %v", err)
	}
	defer forkSession.Close()

	forkInitCtx, forkInitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer forkInitCancel()
	forkInit, forkErrors, err := waitForSessionInit(forkInitCtx, forkSession.Events())
	if err != nil {
		t.Fatalf("wait for fork session init: %v", err)
	}
	assertNoSidecarErrors(t, forkErrors)
	if forkInit.GetClaudeSessionId() != baseInit.GetClaudeSessionId() {
		t.Fatalf(
			"expected forked session init claude_session_id %q, got %q",
			baseInit.GetClaudeSessionId(),
			forkInit.GetClaudeSessionId(),
		)
	}

	stream, err := forkSession.Stream(ctx, "hello")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	partialCtx, partialCancel := harness.ContextWithTimeout(5 * time.Second)
	defer partialCancel()
	if _, err := waitForPartial(partialCtx, stream.Partials()); err != nil {
		t.Fatalf("expected partial message, got %v", err)
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

func TestResumeSessionWithClaudeSessionID(t *testing.T) {
	harness := newE2EHarness(t)
	ctx, cancel := harness.Context()
	defer cancel()

	baseResp, err := harness.CreateSession(ctx, pb.SessionMode_INTERACTIVE)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	baseSession, err := harness.AttachSession(ctx, baseResp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach session: %v", err)
	}
	defer baseSession.Close()

	initCtx, initCancel := harness.ContextWithTimeout(5 * time.Second)
	defer initCancel()
	baseInit, baseErrors, err := waitForSessionInit(initCtx, baseSession.Events())
	if err != nil {
		t.Fatalf("wait for session init: %v", err)
	}
	assertNoSidecarErrors(t, baseErrors)
	if baseInit.GetClaudeSessionId() == "" {
		t.Fatalf("expected claude_session_id to be set")
	}

	resumeResp, err := harness.Client().CreateSession(ctx, &pb.CreateSessionRequest{
		Mode:                  pb.SessionMode_INTERACTIVE,
		ResumeClaudeSessionId: baseInit.GetClaudeSessionId(),
		Options:               e2eTestOptions(shouldUseTestMode()),
	})
	if err != nil {
		t.Fatalf("create resume session: %v", err)
	}
	if resumeResp.GetClaudeSessionId() != baseInit.GetClaudeSessionId() {
		t.Fatalf(
			"expected resumed claude_session_id %q, got %q",
			baseInit.GetClaudeSessionId(),
			resumeResp.GetClaudeSessionId(),
		)
	}

	resumeSession, err := harness.AttachSession(ctx, resumeResp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach resume session: %v", err)
	}
	defer resumeSession.Close()

	resumeInitCtx, resumeInitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer resumeInitCancel()
	resumeInit, resumeErrors, err := waitForSessionInit(resumeInitCtx, resumeSession.Events())
	if err != nil {
		t.Fatalf("wait for resume session init: %v", err)
	}
	assertNoSidecarErrors(t, resumeErrors)
	if resumeInit.GetClaudeSessionId() != baseInit.GetClaudeSessionId() {
		t.Fatalf(
			"expected resume session init claude_session_id %q, got %q",
			baseInit.GetClaudeSessionId(),
			resumeInit.GetClaudeSessionId(),
		)
	}

	result, err := resumeSession.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result == nil || result.Turn == nil {
		t.Fatalf("expected turn result, got %+v", result)
	}
	assertTurnCorrelation(t, result.Turn, result.Turn.RequestID)
	assertNoSidecarErrors(t, result.Turn.Errors)
	if result.Result() == nil || result.Result().GetClaudeSessionId() != baseInit.GetClaudeSessionId() {
		t.Fatalf(
			"expected result claude_session_id %q, got %+v",
			baseInit.GetClaudeSessionId(),
			result.Result(),
		)
	}
}

func TestAttachSessionAlreadyAttachedError(t *testing.T) {
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

	second, err := harness.AttachSession(ctx, resp.SidecarSessionId, Handlers{})
	if err != nil {
		t.Fatalf("attach second session: %v", err)
	}
	defer second.Close()

	waitCtx, waitCancel := harness.ContextWithTimeout(5 * time.Second)
	defer waitCancel()

	event, sidecarErr, err := waitForSidecarError(waitCtx, second.Events())
	if err != nil {
		t.Fatalf("wait for already attached error: %v", err)
	}
	if event.GetSidecarSessionId() != resp.GetSidecarSessionId() {
		t.Fatalf(
			"expected sidecar_session_id %q, got %q",
			resp.GetSidecarSessionId(),
			event.GetSidecarSessionId(),
		)
	}
	if sidecarErr.GetCode() != "ALREADY_ATTACHED" {
		t.Fatalf("expected ALREADY_ATTACHED error, got %q", sidecarErr.GetCode())
	}
	if !sidecarErr.GetFatal() {
		t.Fatalf("expected ALREADY_ATTACHED error to be fatal")
	}
}
