package com.dgarson.claude.sidecar;

import com.google.protobuf.Struct;

/**
 * Convenience record for specifying a client tool's name, description, and input schema.
 *
 * @param name        tool name (will be namespaced as {@code mcp__<serverKey>__<name>})
 * @param description human-readable description of what the tool does
 * @param inputSchema JSON Schema for the tool input, as a protobuf {@link Struct}
 */
public record ToolSpec(String name, String description, Struct inputSchema) {

    /**
     * Creates a ToolSpec with no input schema (schema will be an empty Struct).
     */
    public ToolSpec(String name, String description) {
        this(name, description, Struct.getDefaultInstance());
    }
}
