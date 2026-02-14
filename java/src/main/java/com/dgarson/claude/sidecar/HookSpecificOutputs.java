package com.dgarson.claude.sidecar;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Static factory methods for constructing hook-specific output maps.
 *
 * <p>These maps are passed to {@link HookOutputs#withSpecificOutput} and serialized as
 * the {@code hook_specific_output} Struct field on the {@code HookOutput} proto message.
 *
 * <p>Mirrors the Go SDK's hook-specific output helpers. All methods are thread-safe.
 */
public final class HookSpecificOutputs {

    private HookSpecificOutputs() {}

    /**
     * Creates a PreToolUse hook-specific output that carries a permission decision.
     *
     * @param decision "allow", "deny", or "block"
     * @param reason   human-readable reason for the decision
     */
    public static Map<String, Object> preToolUsePermissionDecision(String decision, String reason) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("hookEventName", "PreToolUse");
        map.put("permissionDecision", decision);
        map.put("permissionDecisionReason", reason);
        return map;
    }

    /**
     * Creates a PreToolUse hook-specific output that replaces the tool input.
     *
     * @param updatedInput new tool input map
     */
    public static Map<String, Object> preToolUseUpdatedInput(Map<String, Object> updatedInput) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("hookEventName", "PreToolUse");
        map.put("updatedInput", updatedInput);
        return map;
    }

    /**
     * Creates a PostToolUse hook-specific output that injects additional context.
     *
     * @param context additional context string to inject after tool execution
     */
    public static Map<String, Object> postToolUseAdditionalContext(String context) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("hookEventName", "PostToolUse");
        map.put("additionalContext", context);
        return map;
    }

    /**
     * Creates a UserPromptSubmit hook-specific output that injects additional context.
     *
     * @param context additional context string to prepend/append to user prompt
     */
    public static Map<String, Object> userPromptSubmitAdditionalContext(String context) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("hookEventName", "UserPromptSubmit");
        map.put("additionalContext", context);
        return map;
    }

    /**
     * Creates a SessionStart hook-specific output that injects additional context.
     *
     * @param context additional context string for session initialization
     */
    public static Map<String, Object> sessionStartAdditionalContext(String context) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("hookEventName", "SessionStart");
        map.put("additionalContext", context);
        return map;
    }
}
