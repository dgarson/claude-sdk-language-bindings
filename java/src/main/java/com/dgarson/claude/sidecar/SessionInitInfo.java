package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;

/**
 * Parsed representation of a {@link Sidecar.SessionInit} event, providing typed accessors
 * for commonly used session initialization fields.
 *
 * <p>Mirrors the Go SDK's SessionInitInfo. Instances are immutable and thread-safe.
 */
public class SessionInitInfo {

    private final String claudeSessionId;
    private final List<String> tools;
    private final Map<String, Object> raw;

    /**
     * Constructs a SessionInitInfo from a proto {@link Sidecar.SessionInit} message.
     */
    public SessionInitInfo(Sidecar.SessionInit init) {
        this.claudeSessionId = init.getClaudeSessionId();
        this.tools = Collections.unmodifiableList(new ArrayList<>(init.getToolsList()));
        this.raw = init.hasRawInit()
                ? Collections.unmodifiableMap(ProtoUtil.structToMap(init.getRawInit()))
                : Collections.emptyMap();
    }

    /**
     * Returns the Claude session ID assigned to this session.
     */
    public String claudeSessionId() {
        return claudeSessionId;
    }

    /**
     * Returns the list of tool names available in this session.
     */
    public List<String> tools() {
        return tools;
    }

    /**
     * Returns the full raw init payload as a map.
     */
    public Map<String, Object> raw() {
        return raw;
    }

    /**
     * Gets a string value from the raw init payload by key.
     *
     * @param key the field key
     * @return the string value, or null if not present or not a string
     */
    public String getString(String key) {
        Object val = raw.get(key);
        return val instanceof String s ? s : null;
    }

    /**
     * Gets a string list from the raw init payload by key.
     *
     * @param key the field key
     * @return the list of strings, or an empty list if not present
     */
    @SuppressWarnings("unchecked")
    public List<String> getStringList(String key) {
        Object val = raw.get(key);
        if (val instanceof List<?> list) {
            List<String> result = new ArrayList<>();
            for (Object item : list) {
                if (item instanceof String s) {
                    result.add(s);
                }
            }
            return Collections.unmodifiableList(result);
        }
        return List.of();
    }

    /**
     * Gets a map from the raw init payload by key.
     *
     * @param key the field key
     * @return the map value, or an empty map if not present
     */
    @SuppressWarnings("unchecked")
    public Map<String, Object> getMap(String key) {
        Object val = raw.get(key);
        if (val instanceof Map<?, ?> map) {
            return Collections.unmodifiableMap((Map<String, Object>) map);
        }
        return Map.of();
    }

    /**
     * Returns the available commands from the raw init payload, if present.
     */
    public List<String> commands() {
        return getStringList("commands");
    }

    /**
     * Returns the output style from the raw init payload, if present.
     */
    public String outputStyle() {
        return getString("output_style");
    }

    @Override
    public String toString() {
        return "SessionInitInfo{sessionId=" + claudeSessionId
                + ", tools=" + tools.size()
                + "}";
    }
}
