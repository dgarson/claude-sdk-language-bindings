package com.dgarson.claude.sidecar;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Static factory methods for building content block maps.
 *
 * <p>Content blocks are represented as {@code Map<String, Object>} with a {@code "type"} field
 * indicating the block kind. These maps are typically used inside tool results or message content.
 *
 * <p>Mirrors the Go SDK's content block helpers. All methods are thread-safe.
 */
public final class ContentBlocks {

    private ContentBlocks() {}

    /**
     * Creates a text content block.
     *
     * @param text the text content
     * @return map with type="text" and the text value
     */
    public static Map<String, Object> text(String text) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("type", "text");
        map.put("text", text);
        return map;
    }

    /**
     * Creates a JSON content block by wrapping the value.
     *
     * <p>The value is stored as-is; it will be serialized through {@link ProtoUtil#toValue}
     * when the containing map is converted to a protobuf Struct.
     *
     * @param value the JSON-compatible object (String, Number, Boolean, Map, List, or null)
     * @return map with type="json" and the data value
     */
    public static Map<String, Object> json(Object value) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("type", "json");
        map.put("data", value);
        return map;
    }

    /**
     * Creates an image content block.
     *
     * @param base64Data base64-encoded image data
     * @param mimeType   MIME type (e.g. "image/png", "image/jpeg")
     * @return map with type="image" and source fields
     */
    public static Map<String, Object> image(String base64Data, String mimeType) {
        Map<String, Object> source = new LinkedHashMap<>();
        source.put("type", "base64");
        source.put("media_type", mimeType);
        source.put("data", base64Data);

        Map<String, Object> map = new LinkedHashMap<>();
        map.put("type", "image");
        map.put("source", source);
        return map;
    }

    /**
     * Creates a custom content block with an arbitrary kind and fields.
     *
     * @param kind   the content block type identifier
     * @param fields additional fields to include in the block
     * @return map with the specified type and all provided fields
     */
    public static Map<String, Object> custom(String kind, Map<String, Object> fields) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("type", kind);
        if (fields != null) {
            map.putAll(fields);
        }
        return map;
    }
}
