package com.dgarson.claude.sidecar;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Static factory methods for building input event maps used with the sidecar's input stream.
 *
 * <p>Input events are sent as {@code StreamInputChunk} messages via the {@code AttachSession}
 * bidirectional stream. These methods build the event dict that becomes the chunk's
 * {@code event} Struct.
 *
 * <p>Mirrors the Go SDK's input stream helpers. All methods are thread-safe.
 */
public final class InputEvents {

    private InputEvents() {}

    /**
     * Creates a user text input event matching the Go SDK's UserTextEvent format.
     * Format: {"type": "user", "message": {"role": "user", "content": text}}
     *
     * @param text the user's text prompt
     * @return event map
     */
    public static Map<String, Object> userText(String text) {
        Map<String, Object> message = new LinkedHashMap<>();
        message.put("role", "user");
        message.put("content", text);

        Map<String, Object> event = new LinkedHashMap<>();
        event.put("type", "user");
        event.put("message", message);
        return event;
    }

    /**
     * Creates a user input event with content blocks matching the Go SDK's UserBlocksEvent format.
     * Format: {"type": "user", "message": {"role": "user", "content": [blocks...]}}
     *
     * @param blocks list of content block maps (created via {@link ContentBlocks})
     * @return event map
     */
    public static Map<String, Object> userBlocks(List<Map<String, Object>> blocks) {
        Map<String, Object> message = new LinkedHashMap<>();
        message.put("role", "user");
        message.put("content", blocks);

        Map<String, Object> event = new LinkedHashMap<>();
        event.put("type", "user");
        event.put("message", message);
        return event;
    }

    /**
     * Creates a control response event indicating success.
     * Matches the Go SDK's ControlResponseSuccess format.
     *
     * @param requestId the request ID to correlate with the original control request
     * @param response  the response payload
     * @return event map with type="control_response"
     */
    public static Map<String, Object> controlResponseSuccess(String requestId,
                                                              Map<String, Object> response) {
        Map<String, Object> inner = new LinkedHashMap<>();
        inner.put("subtype", "success");
        inner.put("request_id", requestId);
        inner.put("response", response);

        Map<String, Object> event = new LinkedHashMap<>();
        event.put("type", "control_response");
        event.put("response", inner);
        return event;
    }

    /**
     * Creates a control response event indicating an error.
     * Matches the Go SDK's ControlResponseError format.
     *
     * @param requestId the request ID to correlate with the original control request
     * @param error     error message describing the failure
     * @return event map with type="control_response"
     */
    public static Map<String, Object> controlResponseError(String requestId, String error) {
        Map<String, Object> inner = new LinkedHashMap<>();
        inner.put("subtype", "error");
        inner.put("request_id", requestId);
        inner.put("error", error);

        Map<String, Object> event = new LinkedHashMap<>();
        event.put("type", "control_response");
        event.put("response", inner);
        return event;
    }
}
