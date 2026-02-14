package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Basic E2E test for the Claude Sidecar Java client.
 * Mirrors Go's {@code TestSidecarE2E} in {@code e2e_test.go}.
 */
@Tag("e2e")
class SidecarE2ETest {

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
    void testBasicSessionCreationAndQuery() throws Exception {
        Sidecar.CreateSessionResponse createResp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);
        assertNotNull(createResp);
        assertFalse(createResp.getSidecarSessionId().isEmpty(),
                "sidecar_session_id should be set");

        CompletableFuture<Sidecar.ToolInvocationRequest> toolCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .tool((req) -> {
                    toolCalled.complete(req);
                    return ToolResults.text("ok");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);
        assertNotNull(session);

        try {
            RunResult result = session.run("hello");
            assertNotNull(result, "expected RunResult");
            assertNotNull(result.getTurn(), "expected turn in result");
            assertNotNull(result.getAssistant(), "expected assistant message");

            if (harness.isTestMode()) {
                assertNotNull(result.getResult(), "expected result message in test mode");
                assertEquals("ok", result.getResult().getResult(),
                        "expected result to be 'ok'");
            }

            // Verify tool handler was invoked
            if (harness.isTestMode()) {
                Sidecar.ToolInvocationRequest toolReq =
                        toolCalled.get(2, TimeUnit.SECONDS);
                assertNotNull(toolReq, "tool handler was not invoked");
            } else {
                try {
                    toolCalled.get(2, TimeUnit.SECONDS);
                } catch (Exception e) {
                    // Tool may not be invoked in live mode
                }
            }
        } finally {
            session.close();
        }
    }

    @Test
    void testToolHandlerInvocationInTestMode() throws Exception {
        org.junit.jupiter.api.Assumptions.assumeTrue(
                harness.isTestMode(),
                "requires test mode; set SIDECAR_E2E_TEST_MODE=1");

        Sidecar.CreateSessionResponse createResp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        CompletableFuture<String> toolFqn = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .tool((req) -> {
                    toolFqn.complete(req.getToolFqn());
                    return ToolResults.text("ok");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            RunResult result = session.run("hello");
            assertNotNull(result);
            assertNotNull(result.getAssistant());

            String fqn = toolFqn.get(2, TimeUnit.SECONDS);
            assertNotNull(fqn, "tool_fqn should be set");
            assertFalse(fqn.isEmpty(), "tool_fqn should not be empty");
        } finally {
            session.close();
        }
    }
}
