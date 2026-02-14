package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Static factory methods for building tool result {@link Struct} instances.
 *
 * <p>Tool results are returned as protobuf Struct values that follow the MCP tool result
 * convention: {@code {"content": [...], "is_error": false, ...}}.
 *
 * <p>Mirrors the Go SDK's tool result helpers. All methods are thread-safe.
 */
public final class ToolResults {

    private ToolResults() {}

    /**
     * Creates a simple text tool result.
     */
    public static Struct text(String text) {
        List<Map<String, Object>> content = List.of(ContentBlocks.text(text));
        return buildResult(content, false, Map.of());
    }

    /**
     * Creates an error tool result.
     */
    public static Struct error(String text) {
        List<Map<String, Object>> content = List.of(ContentBlocks.text(text));
        return buildResult(content, true, Map.of());
    }

    /**
     * Creates a JSON tool result by serializing the given value.
     */
    public static Struct json(Object value) {
        List<Map<String, Object>> content = List.of(ContentBlocks.json(value));
        return buildResult(content, false, Map.of());
    }

    /**
     * Creates a tool result from a raw map. The map is used directly as the Struct fields.
     */
    public static Struct raw(Map<String, Object> result) {
        return ProtoUtil.mapToStruct(result);
    }

    /**
     * Creates a tool result from a list of {@link Sidecar.ContentBlock} proto messages.
     *
     * @param blocks  content blocks from the proto definition
     * @param isError whether this result represents an error
     * @param extra   additional top-level fields to merge into the result
     */
    public static Struct blocks(List<Sidecar.ContentBlock> blocks, boolean isError,
                                Map<String, Object> extra) {
        List<Map<String, Object>> contentList = new ArrayList<>();
        for (Sidecar.ContentBlock block : blocks) {
            contentList.add(contentBlockToMap(block));
        }
        return buildResult(contentList, isError, extra);
    }

    /**
     * Creates a tool result with metadata attached alongside content blocks.
     *
     * @param blocks   content blocks from the proto definition
     * @param isError  whether this result represents an error
     * @param metadata metadata map to include at the top level
     */
    public static Struct withMetadata(List<Sidecar.ContentBlock> blocks, boolean isError,
                                      Map<String, Object> metadata) {
        List<Map<String, Object>> contentList = new ArrayList<>();
        for (Sidecar.ContentBlock block : blocks) {
            contentList.add(contentBlockToMap(block));
        }
        Map<String, Object> extra = new LinkedHashMap<>();
        if (metadata != null) {
            extra.put("metadata", metadata);
        }
        return buildResult(contentList, isError, extra);
    }

    // --- Internal helpers ---

    private static Struct buildResult(List<Map<String, Object>> content, boolean isError,
                                      Map<String, Object> extra) {
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("content", content);
        result.put("is_error", isError);
        if (extra != null) {
            result.putAll(extra);
        }
        return ProtoUtil.mapToStruct(result);
    }

    private static Map<String, Object> contentBlockToMap(Sidecar.ContentBlock block) {
        return switch (block.getBlockCase()) {
            case TEXT -> ContentBlocks.text(block.getText().getText());
            case THINKING -> {
                Map<String, Object> map = new LinkedHashMap<>();
                map.put("type", "thinking");
                map.put("thinking", block.getThinking().getThinking());
                map.put("signature", block.getThinking().getSignature());
                yield map;
            }
            case TOOL_USE -> {
                Map<String, Object> map = new LinkedHashMap<>();
                map.put("type", "tool_use");
                map.put("id", block.getToolUse().getId());
                map.put("name", block.getToolUse().getName());
                map.put("input", ProtoUtil.structToMap(block.getToolUse().getInput()));
                yield map;
            }
            case TOOL_RESULT -> {
                Map<String, Object> map = new LinkedHashMap<>();
                map.put("type", "tool_result");
                map.put("tool_use_id", block.getToolResult().getToolUseId());
                map.put("content", ProtoUtil.fromValue(block.getToolResult().getContent()));
                map.put("is_error", block.getToolResult().getIsError());
                yield map;
            }
            case BLOCK_NOT_SET -> Map.of();
        };
    }
}
