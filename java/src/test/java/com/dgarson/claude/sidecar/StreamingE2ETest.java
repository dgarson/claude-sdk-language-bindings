package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

/**
 * E2E tests for streaming, input streams, and concurrent turns.
 * Mirrors Go's {@code e2e_streaming_test.go}.
 */
@Tag("e2e")
class StreamingE2ETest {

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
    void testIncludePartialsTrueEmitsStreamEvent() throws Exception {
        Sidecar.ClaudeAgentOptions options = Sidecar.ClaudeAgentOptions.newBuilder()
                .setIncludePartialMessages(true)
                .build();
        Sidecar.CreateSessionResponse resp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), Handlers.builder().build());

        try {
            StreamHandle stream = session.stream("hello");

            // Wait for at least one partial
            Sidecar.MessageEvent partial = stream.partials().poll(5, TimeUnit.SECONDS);
            assertNotNull(partial, "expected partial message");
            assertTrue(partial.getIsPartial(), "expected is_partial to be true");
            assertTrue(partial.hasStreamEvent(), "expected stream_event in partial");

            RunResult result = stream.result();
            assertNotNull(result, "expected stream result");
            assertNotNull(result.getTurn(), "expected turn in result");

            // Turn correlation checks
            assertEquals(stream.getRequestId(), result.getTurn().getRequestId(),
                    "request_id mismatch");
            assertFalse(result.getTurn().getTurnId().isEmpty(),
                    "expected turn_id to be set");
            E2EHarness.assertNoSidecarErrors(result.getTurn().getErrors());
        } finally {
            session.close();
        }
    }

    @Test
    void testIncludePartialsFalseSuppressesStreamEvent() throws Exception {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), Handlers.builder().build());

        try {
            StreamHandle stream = session.stream("hello");

            RunResult result = stream.result();
            assertNotNull(result, "expected stream result");
            assertNotNull(result.getTurn(), "expected turn in result");
            assertEquals(stream.getRequestId(), result.getTurn().getRequestId(),
                    "request_id mismatch");
            E2EHarness.assertNoSidecarErrors(result.getTurn().getErrors());

            // Should not receive any partials when include_partial_messages is false
            Sidecar.MessageEvent partial = stream.partials().poll(200, TimeUnit.MILLISECONDS);
            assertNull(partial, "expected no partials when include_partial_messages is false");
        } finally {
            session.close();
        }
    }

    @Test
    void testInputStreamWithTwoUserMessages() throws Exception {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), Handlers.builder().build());

        try {
            String[] streamInfo = session.startInputStream();
            String requestId = streamInfo[0];
            String streamId = streamInfo[1];

            session.sendInputChunk(streamId, ProtoUtil.mapToStruct(InputEvents.userText("first")));
            session.sendInputChunk(streamId, ProtoUtil.mapToStruct(InputEvents.userText("second")));
            session.endInputStream(streamId);

            // Collect events until turn ends
            E2EHarness.CollectedEvents collected = E2EHarness.collectUntilTurnEnd(
                    session, requestId, Duration.ofSeconds(5));
            E2EHarness.assertNoSidecarErrors(collected.errors());

            // Extract user text messages from the collected events
            List<String> userTexts = new java.util.ArrayList<>();
            for (Sidecar.ServerEvent event : collected.events()) {
                if (!event.hasMessage()) continue;
                Sidecar.MessageEvent msg = event.getMessage();
                if (!msg.hasUser()) continue;
                Sidecar.UserMessage user = msg.getUser();
                if (user.getContentCount() == 0) continue;
                Sidecar.ContentBlock first = user.getContent(0);
                if (first.hasText()) {
                    userTexts.add(first.getText().getText());
                }
            }

            assertEquals(2, userTexts.size(), "expected 2 user messages");
            assertEquals("first", userTexts.get(0), "expected first user message");
            assertEquals("second", userTexts.get(1), "expected second user message");
        } finally {
            session.close();
        }
    }

    @Test
    void testInputStreamEndWaitsForFirstResult() throws Exception {
        Sidecar.CreateSessionResponse resp = harness.createSession(
                Sidecar.SessionMode.INTERACTIVE);

        Handlers handlers = Handlers.builder()
                .tool((req) -> ToolResults.text("ok"))
                .build();

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), handlers);

        try {
            String[] streamInfo = session.startInputStream();
            String requestId = streamInfo[0];
            String streamId = streamInfo[1];

            session.sendInputChunk(streamId, ProtoUtil.mapToStruct(InputEvents.userText("first")));
            session.endInputStream(streamId);

            E2EHarness.CollectedEvents collected = E2EHarness.collectUntilTurnEnd(
                    session, requestId, Duration.ofSeconds(5));
            E2EHarness.assertNoSidecarErrors(collected.errors());

            // Verify no SessionClosed before result
            for (Sidecar.ServerEvent event : collected.events()) {
                assertFalse(event.hasSessionClosed(),
                        "unexpected SessionClosed before first result");
            }

            // Verify ResultMessage exists
            boolean foundResult = false;
            for (Sidecar.ServerEvent event : collected.events()) {
                if (event.hasMessage() && event.getMessage().hasResult()) {
                    foundResult = true;
                    break;
                }
            }
            assertTrue(foundResult, "expected ResultMessage before turn end");
        } finally {
            session.close();
        }
    }

    @Test
    void testStreamEventParentToolUseIDPropagation() throws Exception {
        Sidecar.ClaudeAgentOptions options = Sidecar.ClaudeAgentOptions.newBuilder()
                .setIncludePartialMessages(true)
                .build();
        Sidecar.CreateSessionResponse resp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        Handlers handlers = Handlers.builder()
                .tool((req) -> ToolResults.text("ok"))
                .build();

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), handlers);

        try {
            RunResult result = session.run("hello");
            assertNotNull(result, "expected turn result");
            assertNotNull(result.getTurn(), "expected turn");
            E2EHarness.assertNoSidecarErrors(result.getTurn().getErrors());

            // Find tool_use_id from assistant messages
            String toolUseId = "";
            for (Sidecar.MessageEvent msg : result.getTurn().getMessages()) {
                if (!msg.hasAssistant()) continue;
                for (Sidecar.ContentBlock block : msg.getAssistant().getContentList()) {
                    if (block.hasToolUse()) {
                        toolUseId = block.getToolUse().getId();
                        break;
                    }
                }
                if (!toolUseId.isEmpty()) break;
            }
            assertFalse(toolUseId.isEmpty(), "expected tool_use_id to be set");

            // Find parent_tool_use_id from stream event partials
            String streamParent = "";
            for (Sidecar.MessageEvent partial : result.getTurn().getPartials()) {
                if (!partial.hasStreamEvent()) continue;
                String parentId = partial.getStreamEvent().getParentToolUseId();
                if (!parentId.isEmpty()) {
                    streamParent = parentId;
                    break;
                }
            }
            assertFalse(streamParent.isEmpty(),
                    "expected stream_event parent_tool_use_id to be set");
            assertEquals(toolUseId, streamParent,
                    "expected parent_tool_use_id to match tool_use_id");
        } finally {
            session.close();
        }
    }

    @Test
    void testMultipleConcurrentTurnsWithStreaming() throws Exception {
        Sidecar.ClaudeAgentOptions options = Sidecar.ClaudeAgentOptions.newBuilder()
                .setIncludePartialMessages(true)
                .build();
        Sidecar.CreateSessionResponse resp = harness.createSessionWithOptions(
                Sidecar.SessionMode.INTERACTIVE, options);

        Handlers handlers = Handlers.builder()
                .tool((req) -> ToolResults.text("ok"))
                .build();

        Session session = harness.attachSession(
                resp.getSidecarSessionId(), handlers);

        try {
            StreamHandle streamA = session.stream("alpha");
            StreamHandle streamB = session.stream("bravo");

            CompletableFuture<RunResult> resultA = CompletableFuture.supplyAsync(() -> {
                try {
                    return streamA.result();
                } catch (Exception e) {
                    throw new RuntimeException(e);
                }
            });
            CompletableFuture<RunResult> resultB = CompletableFuture.supplyAsync(() -> {
                try {
                    return streamB.result();
                } catch (Exception e) {
                    throw new RuntimeException(e);
                }
            });

            RunResult resA = resultA.get(15, TimeUnit.SECONDS);
            RunResult resB = resultB.get(15, TimeUnit.SECONDS);

            assertNotNull(resA, "expected alpha stream result");
            assertNotNull(resA.getTurn(), "expected alpha turn");
            assertNotNull(resB, "expected bravo stream result");
            assertNotNull(resB.getTurn(), "expected bravo turn");

            // Turn correlation
            assertEquals(streamA.getRequestId(), resA.getTurn().getRequestId(),
                    "alpha request_id mismatch");
            assertEquals(streamB.getRequestId(), resB.getTurn().getRequestId(),
                    "bravo request_id mismatch");
            E2EHarness.assertNoSidecarErrors(resA.getTurn().getErrors());
            E2EHarness.assertNoSidecarErrors(resB.getTurn().getErrors());

            // Verify user texts match
            List<String> alphaTexts = E2EHarness.turnUserTexts(resA.getTurn());
            List<String> bravoTexts = E2EHarness.turnUserTexts(resB.getTurn());
            assertFalse(alphaTexts.isEmpty(), "expected alpha user message");
            assertEquals("alpha", alphaTexts.get(0), "alpha user message mismatch");
            assertFalse(bravoTexts.isEmpty(), "expected bravo user message");
            assertEquals("bravo", bravoTexts.get(0), "bravo user message mismatch");
        } finally {
            session.close();
        }
    }
}
