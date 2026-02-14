package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Value;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * E2E tests for session lifecycle operations.
 * Mirrors Go's {@code e2e_sessions_test.go}.
 */
@Tag("e2e")
class SessionE2ETest {

    private E2EHarness harness;

    @BeforeEach
    void setUp() {
        harness = E2EHarness.create();
    }

    @AfterEach
    void tearDown() {
        if (harness != null) {
            harness.close();
        }
    }

    @Test
    void testCreateSessionReturnsIDs() {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        assertFalse(resp.getSidecarSessionId().isEmpty(),
                "expected sidecar_session_id to be set");
        // claude_session_id is best-effort, may be empty
        if (resp.getClaudeSessionId().isEmpty()) {
            System.out.println("claude_session_id not set (best-effort)");
        }
    }

    @Test
    void testListSessionsIncludesActiveSession() {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Sidecar.ListSessionsResponse listResp = harness.client().listSessions();
        assertNotNull(listResp);

        Sidecar.SessionSummary found = E2EHarness.findSessionSummary(
                listResp.getSessionsList(), resp.getSidecarSessionId());
        assertNotNull(found,
                "expected session " + resp.getSidecarSessionId() + " in list");
    }

    @Test
    void testGetSessionReturnsSummary() {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Sidecar.GetSessionResponse getResp = harness.client().getSession(
                resp.getSidecarSessionId());
        assertNotNull(getResp);

        Sidecar.SessionSummary summary = getResp.getSession();
        assertNotNull(summary, "expected session summary");

        assertEquals(resp.getSidecarSessionId(), summary.getSidecarSessionId(),
                "sidecar_session_id mismatch");
        assertEquals(Sidecar.SessionMode.INTERACTIVE, summary.getMode(),
                "mode mismatch");
        assertTrue(summary.hasCreatedAt(), "expected created_at to be set");
    }

    @Test
    void testDeleteSessionClosesSession() throws Exception {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), Handlers.builder().build());
        try {
            Sidecar.DeleteSessionResponse deleteResp = harness.client().deleteSession(
                    resp.getSidecarSessionId(), true);
            assertTrue(deleteResp.getSuccess(), "expected delete success");

            E2EHarness.CollectedEvents collected = E2EHarness.collectUntilSessionClosed(
                    session, Duration.ofSeconds(5));
            E2EHarness.assertNoSidecarErrors(collected.errors());

            // Verify all events have the correct session ID
            for (Sidecar.ServerEvent event : collected.events()) {
                assertEquals(resp.getSidecarSessionId(), event.getSidecarSessionId(),
                        "sidecar_session_id mismatch on event");
            }
        } finally {
            session.close();
        }
    }

    @Test
    void testOneShotAutoCloses() throws Exception {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.ONE_SHOT);

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), Handlers.builder().build());

        try {
            StreamHandle stream = session.stream("hello");
            String requestId = stream.getRequestId();

            RunResult result = stream.result();
            assertNotNull(result, "expected stream result");
            assertNotNull(result.getTurn(), "expected turn in result");

            // Turn correlation checks
            assertFalse(result.getTurn().getRequestId().isEmpty(),
                    "expected request_id to be set");
            assertEquals(requestId, result.getTurn().getRequestId(),
                    "request_id mismatch");
            assertFalse(result.getTurn().getTurnId().isEmpty(),
                    "expected turn_id to be set");
            E2EHarness.assertNoSidecarErrors(result.getTurn().getErrors());

            // Should receive SessionClosed after one-shot completes
            E2EHarness.CollectedEvents collected = E2EHarness.collectUntilSessionClosed(
                    session, Duration.ofSeconds(5));
            E2EHarness.assertNoSidecarErrors(collected.errors());

            // Verify all events have correct session ID
            for (Sidecar.ServerEvent event : collected.events()) {
                assertEquals(resp.getSidecarSessionId(), event.getSidecarSessionId(),
                        "sidecar_session_id mismatch on event");
            }

            assertTrue(E2EHarness.sawTurnEndBeforeSessionClosed(collected.events()),
                    "expected TURN_END before SessionClosed");
        } finally {
            session.close();
        }
    }

    @Test
    void testForkSessionWithNewOptions() throws Exception {
        Sidecar.CreateSessionResponse baseResp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Session baseSession = harness.attachSession(
                baseResp.getSidecarSessionId(), Handlers.builder().build());

        try {
            // Wait for base session init
            Sidecar.SessionInit baseInit = E2EHarness.waitForSessionInit(
                    baseSession, Duration.ofSeconds(5));
            assertFalse(baseInit.getClaudeSessionId().isEmpty(),
                    "expected claude_session_id to be set");

            // Build fork options with partials enabled
            Sidecar.ClaudeAgentOptions.Builder optionsBuilder =
                    Sidecar.ClaudeAgentOptions.newBuilder()
                            .setIncludePartialMessages(true);
            if (harness.isTestMode()) {
                optionsBuilder.putExtraArgs("test_mode",
                        Value.newBuilder().setBoolValue(true).build());
            }

            Sidecar.ForkSessionResponse forkResp = harness.client().forkSession(
                    Sidecar.ForkSessionRequest.newBuilder()
                            .setSidecarSessionId(baseResp.getSidecarSessionId())
                            .setOptions(optionsBuilder.build())
                            .build());

            assertFalse(forkResp.getSidecarSessionId().isEmpty(),
                    "expected forked sidecar_session_id to be set");
            assertNotEquals(baseResp.getSidecarSessionId(),
                    forkResp.getSidecarSessionId(),
                    "expected forked session to have a new sidecar_session_id");
            assertEquals(baseInit.getClaudeSessionId(),
                    forkResp.getClaudeSessionId(),
                    "expected forked claude_session_id to match base");

            // Attach to forked session
            Session forkSession = harness.attachSession(
                    forkResp.getSidecarSessionId(), Handlers.builder().build());

            try {
                Sidecar.SessionInit forkInit = E2EHarness.waitForSessionInit(
                        forkSession, Duration.ofSeconds(5));
                assertEquals(baseInit.getClaudeSessionId(),
                        forkInit.getClaudeSessionId(),
                        "forked session init claude_session_id mismatch");

                // Stream with partials to verify forked options took effect
                StreamHandle stream = forkSession.stream("hello");
                RunResult result = stream.result();
                assertNotNull(result, "expected fork stream result");
                assertNotNull(result.getTurn(), "expected turn in fork result");
                E2EHarness.assertNoSidecarErrors(result.getTurn().getErrors());
            } finally {
                forkSession.close();
            }
        } finally {
            baseSession.close();
        }
    }

    @Test
    void testResumeSessionWithClaudeSessionID() throws Exception {
        Sidecar.CreateSessionResponse baseResp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Session baseSession = harness.attachSession(
                baseResp.getSidecarSessionId(), Handlers.builder().build());

        try {
            Sidecar.SessionInit baseInit = E2EHarness.waitForSessionInit(
                    baseSession, Duration.ofSeconds(5));
            assertFalse(baseInit.getClaudeSessionId().isEmpty(),
                    "expected claude_session_id to be set");

            // Create resumed session
            Sidecar.ClaudeAgentOptions.Builder optionsBuilder =
                    Sidecar.ClaudeAgentOptions.newBuilder();
            if (harness.isTestMode()) {
                optionsBuilder.putExtraArgs("test_mode",
                        Value.newBuilder().setBoolValue(true).build());
            }

            Sidecar.CreateSessionResponse resumeResp = harness.client().createSession(
                    Sidecar.CreateSessionRequest.newBuilder()
                            .setMode(Sidecar.SessionMode.INTERACTIVE)
                            .setResumeClaudeSessionId(baseInit.getClaudeSessionId())
                            .setOptions(optionsBuilder.build())
                            .build());

            assertEquals(baseInit.getClaudeSessionId(),
                    resumeResp.getClaudeSessionId(),
                    "expected resumed claude_session_id to match base");

            Session resumeSession = harness.attachSession(
                    resumeResp.getSidecarSessionId(), Handlers.builder().build());

            try {
                Sidecar.SessionInit resumeInit = E2EHarness.waitForSessionInit(
                        resumeSession, Duration.ofSeconds(5));
                assertEquals(baseInit.getClaudeSessionId(),
                        resumeInit.getClaudeSessionId(),
                        "resume session init claude_session_id mismatch");

                RunResult result = resumeSession.run("hello");
                assertNotNull(result, "expected run result");
                assertNotNull(result.getTurn(), "expected turn in result");
                E2EHarness.assertNoSidecarErrors(result.getTurn().getErrors());

                assertNotNull(result.getResult(), "expected result message");
                assertEquals(baseInit.getClaudeSessionId(),
                        result.getResult().getClaudeSessionId(),
                        "result claude_session_id mismatch");
            } finally {
                resumeSession.close();
            }
        } finally {
            baseSession.close();
        }
    }

    @Test
    void testAttachSessionAlreadyAttachedError() throws Exception {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Session first = harness.attachSession(
                resp.getSidecarSessionId(), Handlers.builder().build());

        try {
            Session second = harness.attachSession(
                    resp.getSidecarSessionId(), Handlers.builder().build());

            try {
                Sidecar.SidecarError err = E2EHarness.waitForSidecarError(
                        second, Duration.ofSeconds(5));

                assertEquals("ALREADY_ATTACHED", err.getCode(),
                        "expected ALREADY_ATTACHED error");
                assertTrue(err.getFatal(),
                        "expected ALREADY_ATTACHED error to be fatal");
            } finally {
                second.close();
            }
        } finally {
            first.close();
        }
    }
}
