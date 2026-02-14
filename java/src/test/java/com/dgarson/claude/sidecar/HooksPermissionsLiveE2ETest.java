package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Live-mode E2E tests for hooks and permissions callbacks.
 * Only runs when {@code SIDECAR_E2E_LIVE=1} is set.
 * Mirrors Go's {@code e2e_hooks_permissions_live_test.go}.
 *
 * <p><b>Important design note</b>: The Claude Code CLI only invokes the
 * {@code can_use_tool} permission callback for tools that actually require
 * permission in the default permission mode. Simple Bash commands like
 * {@code echo hello} are auto-approved and will NOT trigger the callback.
 * Use the Write tool (which always requires permission) to test permission
 * callbacks reliably.
 */
@Tag("e2e")
@Tag("live")
class HooksPermissionsLiveE2ETest {

    private E2EHarness harness;

    @BeforeEach
    void setUp() {
        harness = E2EHarness.create();
        Assumptions.assumeTrue(!harness.isTestMode(),
                "requires live mode; set SIDECAR_E2E_LIVE=1");
    }

    @AfterEach
    void tearDown() {
        if (harness != null) {
            harness.close();
        }
    }

    /**
     * Tests that permission callbacks are invoked in live mode.
     * Uses the Write tool which always triggers a permission prompt.
     */
    @Test
    void testLivePermissionCallback() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .enablePermissionCallback(true)
                .withClientHook("PreToolUse", "Write", 30)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Void> permCalled = new CompletableFuture<>();
        CompletableFuture<Void> hookCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .permission((req) -> {
                    permCalled.complete(null);
                    return PermissionDecisions.allow("allowed by test");
                })
                .hook((req) -> {
                    hookCalled.complete(null);
                    return Sidecar.HookOutput.newBuilder()
                            .setContinue(true)
                            .build();
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            // Write tool always triggers a permission prompt in default mode.
            RunResult result = session.run(
                    "Write the text 'hello' to /tmp/sidecar_perm_test.txt using the Write tool. Do nothing else.");
            assertNotNull(result, "expected turn result");
            assertNotNull(result.getTurn(), "expected turn");

            // Permission callback must be invoked for Write tool.
            permCalled.get(30, TimeUnit.SECONDS);
            // Hook callback must be invoked for PreToolUse on Write.
            hookCalled.get(30, TimeUnit.SECONDS);
        } finally {
            session.close();
        }
    }

    /**
     * Tests that hook callbacks work in live mode independently of permissions.
     * Uses the Bash tool with a PreToolUse hook (Bash is auto-approved but
     * hooks still fire).
     */
    @Test
    void testLiveHookCallback() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", "Bash", 30)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Void> hookCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> {
                    hookCalled.complete(null);
                    return Sidecar.HookOutput.newBuilder()
                            .setContinue(true)
                            .setReason("ok")
                            .build();
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            RunResult result = session.run(
                    "Use the Bash tool to run: echo hello world. Do nothing else.");
            assertNotNull(result, "expected turn result");
            assertNotNull(result.getTurn(), "expected turn");

            // Hook should fire even though Bash is auto-approved.
            hookCalled.get(30, TimeUnit.SECONDS);
        } finally {
            session.close();
        }
    }

    /**
     * Tests both hooks and permissions together in live mode.
     * Uses the Write tool to reliably trigger both mechanisms.
     */
    @Test
    void testLiveHooksAndPermissionsCallbacks() throws Exception {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", "Write", 30)
                .enablePermissionCallback(true)
                .build();

        Sidecar.CreateSessionResponse createResp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        CompletableFuture<Void> hookCalled = new CompletableFuture<>();
        CompletableFuture<Void> permCalled = new CompletableFuture<>();

        Handlers handlers = Handlers.builder()
                .hook((req) -> {
                    hookCalled.complete(null);
                    return Sidecar.HookOutput.newBuilder()
                            .setContinue(true)
                            .setSystemMessage("hook observed")
                            .setReason("ok")
                            .build();
                })
                .permission((req) -> {
                    permCalled.complete(null);
                    Sidecar.PermissionDecision decision =
                            PermissionDecisions.allow("allowed by test");
                    decision = PermissionDecisions.withUpdatedPermissions(decision,
                            java.util.List.of(
                                    PermissionUpdate.setMode("acceptEdits")
                            ));
                    return decision.toBuilder()
                            .setUpdatedInput(Struct.getDefaultInstance())
                            .build();
                })
                .build();

        Session session = harness.attachSession(
                createResp.getSidecarSessionId(), handlers);

        try {
            // Write tool triggers both PreToolUse hook and permission callback.
            RunResult result = session.run(
                    "Write the text 'test' to /tmp/sidecar_hooks_perm_test.txt using the Write tool. Do nothing else.");
            assertNotNull(result, "expected turn result");
            assertNotNull(result.getTurn(), "expected turn");

            hookCalled.get(30, TimeUnit.SECONDS);
            permCalled.get(30, TimeUnit.SECONDS);
        } finally {
            session.close();
        }
    }
}
