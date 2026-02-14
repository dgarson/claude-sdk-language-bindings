package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.*;

/**
 * E2E tests for hooks and permissions callbacks.
 * Mirrors Go's {@code e2e_hooks_permissions_test.go}.
 */
@Tag("e2e")
class HooksPermissionsE2ETest {

    private static final String ECHO_TOOL_FQN = "mcp__echo__ping";

    private E2EHarness harness;

    @BeforeEach
    void setUp() {
        harness = E2EHarness.create();
        Assumptions.assumeTrue(harness.isTestMode(),
                "requires test mode; set SIDECAR_E2E_TEST_MODE=1");
    }

    @AfterEach
    void tearDown() {
        if (harness != null) {
            harness.close();
        }
    }

    @Test
    void testHookAsyncDefaultContinue() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", ECHO_TOOL_FQN, 10)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Sidecar.HookInvocationRequest> hookCalled = new CompletableFuture<>();
        CompletableFuture<Sidecar.ToolInvocationRequest> toolCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> {
                    hookCalled.complete(req);
                    // Intentionally omit continue to validate default semantics (continue=true).
                    return Sidecar.HookOutput.newBuilder()
                            .setAsync(true)
                            .setAsyncTimeoutMs(25)
                            .build();
                })
                .tool((req) -> {
                    toolCalled.complete(req);
                    return ToolResults.text("ok");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            RunResult result = session.run("hello");
            assertNotNull(result, "expected turn result");
            assertNotNull(result.getTurn(), "expected turn");

            // Verify hook was called
            Sidecar.HookInvocationRequest hookReq =
                    hookCalled.get(2, TimeUnit.SECONDS);
            assertEquals("PreToolUse", hookReq.getHookEvent(),
                    "expected hook_event PreToolUse");
            assertFalse(hookReq.getToolUseId().isEmpty(),
                    "expected tool_use_id to be set");

            // Verify tool was called (async hook did not block)
            Sidecar.ToolInvocationRequest toolReq =
                    toolCalled.get(2, TimeUnit.SECONDS);
            assertEquals(ECHO_TOOL_FQN, toolReq.getToolFqn(),
                    "expected tool_fqn " + ECHO_TOOL_FQN);
        } finally {
            session.close();
        }
    }

    @Test
    void testHookDecisionBlockPreventsTool() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", ECHO_TOOL_FQN, 10)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Void> hookCalled = new CompletableFuture<>();
        CompletableFuture<Void> toolCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> {
                    hookCalled.complete(null);
                    // Block the tool via decision="block"
                    return Sidecar.HookOutput.newBuilder()
                            .setDecision("block")
                            .setReason("blocked by test")
                            .setSystemMessage("blocked by test")
                            .build();
                })
                .tool((req) -> {
                    toolCalled.complete(null);
                    return ToolResults.text("unexpected");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            RunResult result = session.run("hello");
            assertNotNull(result, "expected turn result");

            // Hook should have been called
            hookCalled.get(2, TimeUnit.SECONDS);

            // Tool should NOT have been called
            assertThrows(java.util.concurrent.TimeoutException.class,
                    () -> toolCalled.get(250, TimeUnit.MILLISECONDS),
                    "tool callback should not be invoked when hook blocks");
        } finally {
            session.close();
        }
    }

    @Test
    void testPermissionUpdatedPermissionsAffectSubsequentTurns() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .enablePermissionCallback(true)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        AtomicInteger permCalls = new AtomicInteger(0);
        AtomicInteger toolCalls = new AtomicInteger(0);

        Handlers handlers = Handlers.builder()
                .permission((req) -> {
                    permCalls.incrementAndGet();
                    return PermissionDecisions.withUpdatedPermissions(
                            PermissionDecisions.allow("allowed by test"),
                            List.of(PermissionUpdate.addRules(
                                    "allow", "session",
                                    List.of(new PermissionRule(ECHO_TOOL_FQN, null))
                            ))
                    );
                })
                .tool((req) -> {
                    toolCalls.incrementAndGet();
                    return ToolResults.text("ok");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            session.run("first");
            session.run("second");

            assertEquals(1, permCalls.get(),
                    "expected 1 permission callback after updated_permissions apply");
            assertEquals(2, toolCalls.get(),
                    "expected 2 tool callbacks");
        } finally {
            session.close();
        }
    }

    @Test
    void testPermissionAskConfirmTwoAttempts() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .enablePermissionCallback(true)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        record PermCall(int attempt, String invocationId) {}
        CopyOnWriteArrayList<PermCall> calls = new CopyOnWriteArrayList<>();
        CompletableFuture<Void> twoCallsDone = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .permission((req) -> {
                    calls.add(new PermCall((int) req.getAttempt(), req.getInvocationId()));
                    if (calls.size() >= 2) {
                        twoCallsDone.complete(null);
                    }
                    if (req.getAttempt() <= 1) {
                        return PermissionDecisions.ask("confirm");
                    }
                    return PermissionDecisions.allow("confirmed");
                })
                .tool((req) -> ToolResults.text("ok"))
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            session.run("hello");

            twoCallsDone.get(2, TimeUnit.SECONDS);

            assertTrue(calls.size() >= 2,
                    "expected at least 2 permission attempts, got " + calls.size());
            assertEquals(1, calls.get(0).attempt(),
                    "expected first attempt = 1");
            assertEquals(2, calls.get(1).attempt(),
                    "expected second attempt = 2");
            assertFalse(calls.get(0).invocationId().isEmpty(),
                    "expected non-empty invocation_id");
            assertEquals(calls.get(0).invocationId(), calls.get(1).invocationId(),
                    "expected same invocation_id across attempts");
        } finally {
            session.close();
        }
    }

    @Test
    void testPermissionInterruptClosesSession() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .enablePermissionCallback(true)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Void> toolCalled = new CompletableFuture<>();
        CompletableFuture<Sidecar.SessionClosed> closed = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .permission((req) -> {
                    return PermissionDecisions.withInterrupt(
                            PermissionDecisions.deny("deny+interrupt"), true);
                })
                .tool((req) -> {
                    toolCalled.complete(null);
                    return ToolResults.text("unexpected");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            // Start a background listener for SessionClosed
            Thread listener = new Thread(() -> {
                try {
                    while (!Thread.currentThread().isInterrupted()) {
                        Sidecar.ServerEvent event = session.events().poll(
                                3, TimeUnit.SECONDS);
                        if (event == null) continue;
                        if (event.hasSessionClosed()) {
                            closed.complete(event.getSessionClosed());
                            return;
                        }
                    }
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                }
            });
            listener.setDaemon(true);
            listener.start();

            // The run should complete (possibly with an error), then session should close
            try {
                session.run("hello");
            } catch (Exception e) {
                // Expected - session may close during run
            }

            // Tool should NOT have been called
            assertThrows(java.util.concurrent.TimeoutException.class,
                    () -> toolCalled.get(250, TimeUnit.MILLISECONDS),
                    "tool callback should not be invoked when permission denies");

            // Session should close due to interrupt
            Sidecar.SessionClosed sc = closed.get(3, TimeUnit.SECONDS);
            assertEquals("permission_interrupted", sc.getReason(),
                    "expected session_closed reason permission_interrupted");
        } finally {
            session.close();
        }
    }

    @Test
    void testPreToolUseUpdatedInputRewrite() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", ECHO_TOOL_FQN, 10)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Map<String, Object>> toolInput = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> HookOutputs.withSpecificOutput(
                        HookOutputs.hookContinue(),
                        HookSpecificOutputs.preToolUseUpdatedInput(
                                Map.of("text", "rewritten"))
                ))
                .tool((req) -> {
                    toolInput.complete(ProtoUtil.structToMap(req.getToolInput()));
                    return ToolResults.text("ok");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            session.run("hello");

            Map<String, Object> input = toolInput.get(2, TimeUnit.SECONDS);
            assertEquals("rewritten", input.get("text"),
                    "expected rewritten tool input");
        } finally {
            session.close();
        }
    }

    @Test
    void testPreToolUseHookInputContext() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", ECHO_TOOL_FQN, 10)
                .permissionMode("acceptEdits")
                .cwd("/tmp/echo-hook-test")
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Map<String, Object>> hookInputData = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> {
                    hookInputData.complete(ProtoUtil.structToMap(req.getInputData()));
                    return HookOutputs.hookContinue();
                })
                .tool((req) -> ToolResults.text("ok"))
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            session.run("hello");

            Map<String, Object> input = hookInputData.get(2, TimeUnit.SECONDS);

            // Validate context fields
            assertNotNull(input.get("session_id"), "expected session_id to be set");
            assertFalse(((String) input.get("session_id")).isEmpty(),
                    "expected session_id to be non-empty");
            assertEquals("PreToolUse", input.get("hook_event_name"),
                    "expected hook_event_name PreToolUse");
            assertEquals(ECHO_TOOL_FQN, input.get("tool_name"),
                    "expected tool_name " + ECHO_TOOL_FQN);
            assertEquals("acceptEdits", input.get("permission_mode"),
                    "expected permission_mode acceptEdits");
            assertEquals("/tmp/echo-hook-test", input.get("cwd"),
                    "expected cwd /tmp/echo-hook-test");
            assertTrue(input.containsKey("transcript_path"),
                    "expected transcript_path to be present");

            // Validate tool_input
            @SuppressWarnings("unchecked")
            Map<String, Object> toolInput = (Map<String, Object>) input.get("tool_input");
            assertNotNull(toolInput, "expected tool_input map");
            assertEquals("ping", toolInput.get("text"),
                    "expected tool_input text ping");
        } finally {
            session.close();
        }
    }

    @Test
    void testPreToolUseMatcherFilters() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", "does_not_match", 10)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Void> hookCalled = new CompletableFuture<>();
        CompletableFuture<Void> toolCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> {
                    hookCalled.complete(null);
                    return HookOutputs.hookContinue();
                })
                .tool((req) -> {
                    toolCalled.complete(null);
                    return ToolResults.text("ok");
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            session.run("hello");

            // Tool should be called (no hook blocking it)
            toolCalled.get(2, TimeUnit.SECONDS);

            // Hook should NOT be called (matcher doesn't match echo tool)
            assertThrows(java.util.concurrent.TimeoutException.class,
                    () -> hookCalled.get(250, TimeUnit.MILLISECONDS),
                    "expected hook to be filtered by matcher");
        } finally {
            session.close();
        }
    }
}
