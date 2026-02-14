package com.dgarson.claude.sidecar;

/**
 * A single permission rule targeting a specific tool.
 *
 * @param toolName    the tool this rule applies to
 * @param ruleContent the rule content/pattern
 */
public record PermissionRule(String toolName, String ruleContent) {}
