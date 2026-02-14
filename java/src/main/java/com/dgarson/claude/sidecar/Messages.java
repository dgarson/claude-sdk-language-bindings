package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;

/**
 * Message parsing utilities for converting proto {@link Sidecar.MessageEvent} instances
 * into typed Java records.
 *
 * <p>Mirrors the Go SDK's message parsing (messages.go + message_blocks.go).
 * All methods are thread-safe.
 */
public final class Messages {

    private Messages() {}

    // =====================
    // Parsed message types (sealed hierarchy)
    // =====================

    /**
     * Sealed interface for all parsed message types from a {@link Sidecar.MessageEvent}.
     */
    public sealed interface ParsedMessage
            permits UserMessage, AssistantMessage, SystemMessage, ResultMessage, StreamEventMessage {}

    public record UserMessage(
            List<MessageBlock> content,
            String checkpointUUID,
            String parentToolUseId
    ) implements ParsedMessage {}

    public record AssistantMessage(
            List<MessageBlock> content,
            String model,
            String parentToolUseId,
            String error
    ) implements ParsedMessage {}

    public record SystemMessage(
            String subtype,
            Map<String, Object> data
    ) implements ParsedMessage {}

    public record ResultMessage(
            String subtype,
            long durationMs,
            long durationApiMs,
            boolean isError,
            int numTurns,
            String sessionId,
            double totalCostUsd,
            Map<String, Object> usage,
            String result,
            Map<String, Object> structuredOutput
    ) implements ParsedMessage {}

    public record StreamEventMessage(
            String uuid,
            String sessionId,
            Map<String, Object> event,
            String parentToolUseId
    ) implements ParsedMessage {}

    // =====================
    // Message block types (sealed hierarchy)
    // =====================

    /**
     * Sealed interface for content blocks within messages.
     */
    public sealed interface MessageBlock
            permits TextBlock, ThinkingBlock, ToolUseBlock, ToolResultBlock {}

    public record TextBlock(String text) implements MessageBlock {}

    public record ThinkingBlock(String thinking, String signature) implements MessageBlock {}

    public record ToolUseBlock(
            String id,
            String name,
            Map<String, Object> input
    ) implements MessageBlock {}

    public record ToolResultBlock(
            String toolUseId,
            Object content,
            boolean isError
    ) implements MessageBlock {}

    // =====================
    // Parsing methods
    // =====================

    /**
     * Parses a proto {@link Sidecar.MessageEvent} into a typed {@link ParsedMessage}.
     *
     * @param event the message event from the server
     * @return parsed message, or null if the event has no recognized message type
     */
    public static ParsedMessage fromEvent(Sidecar.MessageEvent event) {
        if (event == null) {
            return null;
        }
        return switch (event.getMsgCase()) {
            case USER -> parseUserMessage(event.getUser());
            case ASSISTANT -> parseAssistantMessage(event.getAssistant());
            case SYSTEM -> parseSystemMessage(event.getSystem());
            case RESULT -> parseResultMessage(event.getResult());
            case STREAM_EVENT -> parseStreamEvent(event.getStreamEvent());
            case MSG_NOT_SET -> null;
        };
    }

    /**
     * Converts a proto {@link Sidecar.ContentBlock} to a typed {@link MessageBlock}.
     *
     * @param block the proto content block
     * @return parsed message block, or null if the block type is not recognized
     */
    public static MessageBlock blockFromProto(Sidecar.ContentBlock block) {
        if (block == null) {
            return null;
        }
        return switch (block.getBlockCase()) {
            case TEXT -> new TextBlock(block.getText().getText());
            case THINKING -> new ThinkingBlock(
                    block.getThinking().getThinking(),
                    block.getThinking().getSignature());
            case TOOL_USE -> new ToolUseBlock(
                    block.getToolUse().getId(),
                    block.getToolUse().getName(),
                    ProtoUtil.structToMap(block.getToolUse().getInput()));
            case TOOL_RESULT -> new ToolResultBlock(
                    block.getToolResult().getToolUseId(),
                    ProtoUtil.fromValue(block.getToolResult().getContent()),
                    block.getToolResult().getIsError());
            case BLOCK_NOT_SET -> null;
        };
    }

    // =====================
    // Internal helpers
    // =====================

    private static UserMessage parseUserMessage(Sidecar.UserMessage msg) {
        return new UserMessage(
                parseBlocks(msg.getContentList()),
                msg.getCheckpointUuid(),
                msg.getParentToolUseId());
    }

    private static AssistantMessage parseAssistantMessage(Sidecar.AssistantMessage msg) {
        return new AssistantMessage(
                parseBlocks(msg.getContentList()),
                msg.getModel(),
                msg.getParentToolUseId(),
                msg.getError());
    }

    private static SystemMessage parseSystemMessage(Sidecar.SystemMessage msg) {
        Map<String, Object> data = msg.hasData()
                ? ProtoUtil.structToMap(msg.getData())
                : Collections.emptyMap();
        return new SystemMessage(msg.getSubtype(), data);
    }

    private static ResultMessage parseResultMessage(Sidecar.ResultMessage msg) {
        Map<String, Object> usage = msg.hasUsage()
                ? ProtoUtil.structToMap(msg.getUsage())
                : Collections.emptyMap();
        Map<String, Object> structuredOutput = msg.hasStructuredOutput()
                ? ProtoUtil.structToMap(msg.getStructuredOutput())
                : Collections.emptyMap();
        return new ResultMessage(
                msg.getSubtype(),
                msg.getDurationMs(),
                msg.getDurationApiMs(),
                msg.getIsError(),
                msg.getNumTurns(),
                msg.getClaudeSessionId(),
                msg.getTotalCostUsd(),
                usage,
                msg.getResult(),
                structuredOutput);
    }

    private static StreamEventMessage parseStreamEvent(Sidecar.StreamEvent msg) {
        Map<String, Object> event = msg.hasEvent()
                ? ProtoUtil.structToMap(msg.getEvent())
                : Collections.emptyMap();
        return new StreamEventMessage(
                msg.getUuid(),
                msg.getSessionId(),
                event,
                msg.getParentToolUseId());
    }

    private static List<MessageBlock> parseBlocks(List<Sidecar.ContentBlock> protoBlocks) {
        if (protoBlocks == null || protoBlocks.isEmpty()) {
            return List.of();
        }
        List<MessageBlock> blocks = new ArrayList<>(protoBlocks.size());
        for (Sidecar.ContentBlock pb : protoBlocks) {
            MessageBlock block = blockFromProto(pb);
            if (block != null) {
                blocks.add(block);
            }
        }
        return Collections.unmodifiableList(blocks);
    }
}
