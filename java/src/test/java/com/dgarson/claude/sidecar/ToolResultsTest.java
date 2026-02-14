package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;
import com.google.protobuf.Value;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for {@link ToolResults}.
 * Mirrors Go's {@code TestToolResultBlocksAndStream} in {@code tool_results_test.go}.
 */
class ToolResultsTest {

    @Test
    void testTextResult() {
        Struct result = ToolResults.text("hello world");
        assertNotNull(result, "expected result struct");

        Map<String, Object> map = ProtoUtil.structToMap(result);
        @SuppressWarnings("unchecked")
        List<Object> content = (List<Object>) map.get("content");
        assertNotNull(content, "expected content list");
        assertEquals(1, content.size(), "expected 1 content block");

        @SuppressWarnings("unchecked")
        Map<String, Object> block = (Map<String, Object>) content.get(0);
        assertEquals("text", block.get("type"), "expected text block type");
        assertEquals("hello world", block.get("text"), "expected text content");

        assertEquals(false, map.get("is_error"), "expected is_error=false");
    }

    @Test
    void testErrorResult() {
        Struct result = ToolResults.error("something went wrong");
        assertNotNull(result, "expected result struct");

        Map<String, Object> map = ProtoUtil.structToMap(result);
        assertEquals(true, map.get("is_error"), "expected is_error=true");

        @SuppressWarnings("unchecked")
        List<Object> content = (List<Object>) map.get("content");
        assertNotNull(content, "expected content list");
        assertEquals(1, content.size());

        @SuppressWarnings("unchecked")
        Map<String, Object> block = (Map<String, Object>) content.get(0);
        assertEquals("text", block.get("type"));
        assertEquals("something went wrong", block.get("text"));
    }

    @Test
    void testJsonResult() {
        Struct result = ToolResults.json(Map.of("ok", true, "count", 42));
        assertNotNull(result, "expected result struct");

        Map<String, Object> map = ProtoUtil.structToMap(result);
        @SuppressWarnings("unchecked")
        List<Object> content = (List<Object>) map.get("content");
        assertNotNull(content, "expected content list");
        assertEquals(1, content.size());

        @SuppressWarnings("unchecked")
        Map<String, Object> block = (Map<String, Object>) content.get(0);
        assertEquals("json", block.get("type"), "expected json block type");
    }

    @Test
    void testRawResult() {
        Struct result = ToolResults.raw(Map.of(
                "custom_key", "custom_value",
                "count", 10
        ));
        assertNotNull(result, "expected result struct");

        Map<String, Object> map = ProtoUtil.structToMap(result);
        assertEquals("custom_value", map.get("custom_key"));
    }

    @Test
    void testBlocksWithMultipleContentTypes() {
        List<Sidecar.ContentBlock> protoBlocks = List.of(
                Sidecar.ContentBlock.newBuilder()
                        .setText(Sidecar.TextBlock.newBuilder().setText("hello").build())
                        .build(),
                Sidecar.ContentBlock.newBuilder()
                        .setToolUse(Sidecar.ToolUseBlock.newBuilder()
                                .setId("t1")
                                .setName("test")
                                .setInput(Struct.newBuilder()
                                        .putFields("ok", Value.newBuilder()
                                                .setBoolValue(true).build())
                                        .build())
                                .build())
                        .build()
        );
        Struct result = ToolResults.blocks(protoBlocks, false,
                Map.of("meta", Map.of("k", "v")));
        assertNotNull(result, "expected result struct");

        Map<String, Object> map = ProtoUtil.structToMap(result);
        @SuppressWarnings("unchecked")
        List<Object> content = (List<Object>) map.get("content");
        assertNotNull(content, "expected content list");
        assertEquals(2, content.size(), "expected 2 content blocks");

        @SuppressWarnings("unchecked")
        Map<String, Object> textBlock = (Map<String, Object>) content.get(0);
        assertEquals("text", textBlock.get("type"), "expected text block");

        // Verify extra metadata was preserved
        @SuppressWarnings("unchecked")
        Map<String, Object> meta = (Map<String, Object>) map.get("meta");
        assertNotNull(meta, "expected meta to be preserved");
        assertEquals("v", meta.get("k"), "expected meta value");
    }

    @Test
    void testBlocksWithIsError() {
        List<Sidecar.ContentBlock> protoBlocks = List.of(
                Sidecar.ContentBlock.newBuilder()
                        .setText(Sidecar.TextBlock.newBuilder().setText("failure").build())
                        .build()
        );
        Struct result = ToolResults.blocks(protoBlocks, true, null);
        assertNotNull(result);

        Map<String, Object> map = ProtoUtil.structToMap(result);
        assertEquals(true, map.get("is_error"), "expected is_error=true");
    }

    @Test
    void testWithMetadata() {
        List<Sidecar.ContentBlock> protoBlocks = List.of(
                Sidecar.ContentBlock.newBuilder()
                        .setText(Sidecar.TextBlock.newBuilder().setText("data").build())
                        .build()
        );
        Struct result = ToolResults.withMetadata(protoBlocks, false,
                Map.of("version", "1.0", "source", "test"));
        assertNotNull(result);

        Map<String, Object> map = ProtoUtil.structToMap(result);
        @SuppressWarnings("unchecked")
        Map<String, Object> meta = (Map<String, Object>) map.get("metadata");
        assertNotNull(meta, "expected metadata");
        assertEquals("1.0", meta.get("version"));
        assertEquals("test", meta.get("source"));
    }
}
