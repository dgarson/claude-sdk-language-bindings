package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;

import java.util.Collection;

/**
 * Constants and helpers for sidecar capability strings.
 *
 * <p>Capabilities are advertised by the sidecar in the {@link Sidecar.GetInfoResponse}
 * and indicate which features are supported by the running sidecar version.
 *
 * <p>Mirrors the Go SDK's capabilities.go. All methods are thread-safe.
 */
public final class Capabilities {

    private Capabilities() {}

    // Capability constants
    public static final String HOOKS = "hooks";
    public static final String SDK_MCP = "sdk_mcp";
    public static final String CHECKPOINTING = "checkpointing";
    public static final String STRUCTURED_OUTPUTS = "structured_outputs";
    public static final String SANDBOX = "sandbox";
    public static final String PERMISSION_CALLBACK = "permission_callback";
    public static final String CLIENT_TOOLS = "client_tools";
    public static final String INPUT_STREAM = "input_stream";
    public static final String FORK = "fork";
    public static final String AGENTS = "agents";
    public static final String PLUGINS = "plugins";
    public static final String BETAS = "betas";

    /**
     * Checks whether a specific capability is present in the given list.
     *
     * @param capabilities the list of capability strings (from GetInfoResponse)
     * @param capability   the capability to check for
     * @return true if the capability is present
     */
    public static boolean hasCapability(Collection<String> capabilities, String capability) {
        if (capabilities == null || capability == null) {
            return false;
        }
        return capabilities.contains(capability);
    }

    /**
     * Checks whether all of the specified capabilities are present.
     *
     * @param capabilities the list of capability strings
     * @param required     the capabilities that must all be present
     * @return true if all required capabilities are present
     */
    public static boolean hasAllCapabilities(Collection<String> capabilities, String... required) {
        if (capabilities == null || required == null) {
            return false;
        }
        for (String cap : required) {
            if (!capabilities.contains(cap)) {
                return false;
            }
        }
        return true;
    }

    /**
     * Checks whether any of the specified capabilities are present.
     *
     * @param capabilities the list of capability strings
     * @param candidates   the capabilities to check
     * @return true if at least one capability is present
     */
    public static boolean hasAnyCapability(Collection<String> capabilities, String... candidates) {
        if (capabilities == null || candidates == null) {
            return false;
        }
        for (String cap : candidates) {
            if (capabilities.contains(cap)) {
                return true;
            }
        }
        return false;
    }
}
