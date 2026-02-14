package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;
import com.google.protobuf.Value;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for {@link Messages} (message parsing from proto events).
 * Mirrors Go's {@code messages_test.go}.
 */
class MessagesTest {

    @Test
    void testFromEventUser() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setUser(Sidecar.UserMessage.newBuilder()
                        .setCheckpointUuid("chk-1")
                        .addContent(Sidecar.ContentBlock.newBuilder()
                                .setText(Sidecar.TextBlock.newBuilder()
                                        .setText("hello")
                                        .build())
                                .build())
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertNotNull(parsed);
        assertInstanceOf(Messages.UserMessage.class, parsed);

        Messages.UserMessage user = (Messages.UserMessage) parsed;
        assertEquals("chk-1", user.checkpointUUID());
        assertEquals(1, user.content().size());
    }

    @Test
    void testFromEventAssistant() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setAssistant(Sidecar.AssistantMessage.newBuilder()
                        .setModel("claude-sonnet-4-5")
                        .addContent(Sidecar.ContentBlock.newBuilder()
                                .setText(Sidecar.TextBlock.newBuilder()
                                        .setText("I can help with that.")
                                        .build())
                                .build())
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertNotNull(parsed);
        assertInstanceOf(Messages.AssistantMessage.class, parsed);

        Messages.AssistantMessage assistant = (Messages.AssistantMessage) parsed;
        assertEquals("claude-sonnet-4-5", assistant.model());
        assertEquals(1, assistant.content().size());
    }

    @Test
    void testFromEventAssistantWithToolUse() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setAssistant(Sidecar.AssistantMessage.newBuilder()
                        .addContent(Sidecar.ContentBlock.newBuilder()
                                .setToolUse(Sidecar.ToolUseBlock.newBuilder()
                                        .setId("tool-123")
                                        .setName("mcp__echo__ping")
                                        .setInput(Struct.newBuilder()
                                                .putFields("text", Value.newBuilder()
                                                        .setStringValue("hello")
                                                        .build())
                                                .build())
                                        .build())
                                .build())
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertNotNull(parsed);
        assertInstanceOf(Messages.AssistantMessage.class, parsed);
    }

    @Test
    void testFromEventResult() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setResult(Sidecar.ResultMessage.newBuilder()
                        .setSubtype("success")
                        .setDurationMs(1234)
                        .setIsError(false)
                        .setNumTurns(3)
                        .setClaudeSessionId("session-1")
                        .setTotalCostUsd(0.05)
                        .setResult("ok")
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertNotNull(parsed);
        assertInstanceOf(Messages.ResultMessage.class, parsed);

        Messages.ResultMessage result = (Messages.ResultMessage) parsed;
        assertEquals("success", result.subtype());
        assertEquals(1234, result.durationMs());
        assertFalse(result.isError());
        assertEquals(3, result.numTurns());
        assertEquals("session-1", result.sessionId());
        assertEquals("ok", result.result());
    }

    @Test
    void testFromEventStreamEvent() {
        Struct payload = Struct.newBuilder()
                .putFields("type", Value.newBuilder()
                        .setStringValue("delta")
                        .build())
                .build();

        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setStreamEvent(Sidecar.StreamEvent.newBuilder()
                        .setUuid("stream-1")
                        .setSessionId("session-1")
                        .setEvent(payload)
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertNotNull(parsed);
        assertInstanceOf(Messages.StreamEventMessage.class, parsed);

        Messages.StreamEventMessage stream = (Messages.StreamEventMessage) parsed;
        assertEquals("stream-1", stream.uuid());
        assertEquals("session-1", stream.sessionId());
        assertEquals("delta", stream.event().get("type"));
    }

    @Test
    void testFromEventSystem() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setSystem(Sidecar.SystemMessage.newBuilder()
                        .setSubtype("init")
                        .setData(Struct.newBuilder()
                                .putFields("version", Value.newBuilder()
                                        .setStringValue("1.0")
                                        .build())
                                .build())
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertNotNull(parsed);
        assertInstanceOf(Messages.SystemMessage.class, parsed);

        Messages.SystemMessage system = (Messages.SystemMessage) parsed;
        assertEquals("init", system.subtype());
        assertEquals("1.0", system.data().get("version"));
    }

    @Test
    void testFromEventPartialFlag() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setIsPartial(true)
                .setAssistant(Sidecar.AssistantMessage.newBuilder()
                        .addContent(Sidecar.ContentBlock.newBuilder()
                                .setText(Sidecar.TextBlock.newBuilder()
                                        .setText("partial content")
                                        .build())
                                .build())
                        .build())
                .build();

        assertTrue(event.getIsPartial(), "expected is_partial=true");
    }

    @Test
    void testFromEventUserWithParentToolUseId() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setUser(Sidecar.UserMessage.newBuilder()
                        .setParentToolUseId("tool-456")
                        .addContent(Sidecar.ContentBlock.newBuilder()
                                .setText(Sidecar.TextBlock.newBuilder()
                                        .setText("nested response")
                                        .build())
                                .build())
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertInstanceOf(Messages.UserMessage.class, parsed);
        Messages.UserMessage user = (Messages.UserMessage) parsed;
        assertEquals("tool-456", user.parentToolUseId());
    }

    @Test
    void testFromEventResultWithStructuredOutput() {
        Struct structured = Struct.newBuilder()
                .putFields("key", Value.newBuilder()
                        .setStringValue("value")
                        .build())
                .build();

        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setResult(Sidecar.ResultMessage.newBuilder()
                        .setResult("done")
                        .setStructuredOutput(structured)
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertInstanceOf(Messages.ResultMessage.class, parsed);
        Messages.ResultMessage result = (Messages.ResultMessage) parsed;
        assertEquals("done", result.result());
        assertNotNull(result.structuredOutput());
        assertEquals("value", result.structuredOutput().get("key"));
    }

    @Test
    void testFromEventStreamEventWithParentToolUseId() {
        Sidecar.MessageEvent event = Sidecar.MessageEvent.newBuilder()
                .setStreamEvent(Sidecar.StreamEvent.newBuilder()
                        .setUuid("stream-2")
                        .setParentToolUseId("tool-789")
                        .setEvent(Struct.getDefaultInstance())
                        .build())
                .build();

        Messages.ParsedMessage parsed = Messages.fromEvent(event);
        assertInstanceOf(Messages.StreamEventMessage.class, parsed);
        Messages.StreamEventMessage stream = (Messages.StreamEventMessage) parsed;
        assertEquals("tool-789", stream.parentToolUseId());
    }
}
