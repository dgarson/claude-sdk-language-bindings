package com.dgarson.claude.sidecar;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Represents a permission update to be applied via a {@link claude_sidecar.v1.Sidecar.PermissionDecision}.
 *
 * <p>Mirrors the Go SDK's permission update types. Use the static factory methods to create
 * instances for each update type.
 *
 * @param type        update type (e.g. "setMode", "addRules", "replaceRules", "removeRules",
 *                    "addDirectories", "removeDirectories")
 * @param behavior    permission behavior (e.g. "allow", "deny")
 * @param mode        permission mode value (for "setMode" type)
 * @param destination destination for the update (e.g. "session", "project")
 * @param rules       list of permission rules (for rule-based updates)
 * @param directories list of directory paths (for directory-based updates)
 */
public record PermissionUpdate(
        String type,
        String behavior,
        String mode,
        String destination,
        List<PermissionRule> rules,
        List<String> directories
) {

    public record Rule(String toolName, String ruleContent) {
        public Map<String, Object> toMap() {
            Map<String, Object> map = new LinkedHashMap<>();
            map.put("toolName", toolName);
            if (ruleContent != null && !ruleContent.isEmpty()) {
                map.put("ruleContent", ruleContent);
            }
            return map;
        }
    }

    /**
     * Creates a "setMode" permission update.
     */
    public static PermissionUpdate setMode(String mode, String destination) {
        return new PermissionUpdate("setMode", null, mode, destination, List.of(), List.of());
    }

    /**
     * Creates a "setMode" permission update (without destination).
     */
    public static PermissionUpdate setMode(String mode) {
        return new PermissionUpdate("setMode", null, mode, null, List.of(), List.of());
    }

    /**
     * Creates an "addRules" permission update.
     */
    public static PermissionUpdate addRules(String behavior, String destination,
                                            List<PermissionRule> rules) {
        return new PermissionUpdate("addRules", behavior, null, destination, rules, List.of());
    }

    /**
     * Creates a "replaceRules" permission update.
     */
    public static PermissionUpdate replaceRules(String behavior, String destination,
                                                List<PermissionRule> rules) {
        return new PermissionUpdate("replaceRules", behavior, null, destination, rules, List.of());
    }

    /**
     * Creates a "removeRules" permission update.
     */
    public static PermissionUpdate removeRules(String behavior, String destination,
                                               List<PermissionRule> rules) {
        return new PermissionUpdate("removeRules", behavior, null, destination, rules, List.of());
    }

    /**
     * Creates an "addDirectories" permission update.
     */
    public static PermissionUpdate addDirectories(String destination, List<String> directories) {
        return new PermissionUpdate("addDirectories", null, null, destination, List.of(), directories);
    }

    /**
     * Creates an "addDirectories" permission update (without destination).
     */
    public static PermissionUpdate addDirectories(List<String> directories) {
        return new PermissionUpdate("addDirectories", null, null, null, List.of(), directories);
    }

    /**
     * Creates a "removeDirectories" permission update.
     */
    public static PermissionUpdate removeDirectories(String destination, List<String> directories) {
        return new PermissionUpdate("removeDirectories", null, null, destination, List.of(), directories);
    }

    /**
     * Creates a "removeDirectories" permission update (without destination).
     */
    public static PermissionUpdate removeDirectories(List<String> directories) {
        return new PermissionUpdate("removeDirectories", null, null, null, List.of(), directories);
    }

    /**
     * Converts this update to a map suitable for protobuf serialization.
     * Matches the Go SDK's ToMap() format with camelCase keys.
     */
    public Map<String, Object> toMap() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("type", type);

        if (destination != null && !destination.isEmpty()) {
            map.put("destination", destination);
        }
        if (behavior != null && !behavior.isEmpty()) {
            map.put("behavior", behavior);
        }
        if (mode != null && !mode.isEmpty()) {
            map.put("mode", mode);
        }
        if (directories != null && !directories.isEmpty()) {
            map.put("directories", directories);
        }
        if (rules != null && !rules.isEmpty()) {
            List<Map<String, Object>> ruleList = new ArrayList<>();
            for (PermissionRule rule : rules) {
                Map<String, Object> ruleMap = new LinkedHashMap<>();
                ruleMap.put("toolName", rule.toolName());
                if (rule.ruleContent() != null && !rule.ruleContent().isEmpty()) {
                    ruleMap.put("ruleContent", rule.ruleContent());
                }
                ruleList.add(ruleMap);
            }
            map.put("rules", ruleList);
        }

        return map;
    }
}
