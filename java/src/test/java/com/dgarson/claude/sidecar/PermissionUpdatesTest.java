package com.dgarson.claude.sidecar;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for {@link PermissionUpdate} serialization.
 * Mirrors Go's {@code permission_updates_test.go}.
 */
class PermissionUpdatesTest {

    @Test
    void testSetModeToMap() {
        PermissionUpdate update = PermissionUpdate.setMode("acceptEdits", "session");
        Map<String, Object> map = update.toMap();

        assertEquals("setMode", map.get("type"));
        assertEquals("acceptEdits", map.get("mode"));
        assertEquals("session", map.get("destination"));
        assertFalse(map.containsKey("rules"), "should not have rules");
    }

    @Test
    void testAddRulesToMap() {
        PermissionUpdate update = PermissionUpdate.addRules("allow", "session",
                List.of(
                        new PermissionRule("mcp__echo__ping", null),
                        new PermissionRule("Bash", "echo *")
                )
        );
        Map<String, Object> map = update.toMap();

        assertEquals("addRules", map.get("type"));
        assertEquals("allow", map.get("behavior"));
        assertEquals("session", map.get("destination"));

        @SuppressWarnings("unchecked")
        List<Map<String, Object>> rules = (List<Map<String, Object>>) map.get("rules");
        assertNotNull(rules, "expected rules list");
        assertEquals(2, rules.size());
        assertEquals("mcp__echo__ping", rules.get(0).get("toolName"));
        assertFalse(rules.get(0).containsKey("ruleContent"),
                "should not include ruleContent when null");
        assertEquals("Bash", rules.get(1).get("toolName"));
        assertEquals("echo *", rules.get(1).get("ruleContent"));
    }

    @Test
    void testReplaceRulesToMap() {
        PermissionUpdate update = PermissionUpdate.replaceRules("deny", "project",
                List.of(new PermissionRule("Edit", "*.py"))
        );
        Map<String, Object> map = update.toMap();

        assertEquals("replaceRules", map.get("type"));
        assertEquals("deny", map.get("behavior"));
        assertEquals("project", map.get("destination"));

        @SuppressWarnings("unchecked")
        List<Map<String, Object>> rules = (List<Map<String, Object>>) map.get("rules");
        assertEquals(1, rules.size());
    }

    @Test
    void testRemoveRulesToMap() {
        PermissionUpdate update = PermissionUpdate.removeRules("allow", "session",
                List.of(new PermissionRule("mcp__echo__ping", null))
        );
        Map<String, Object> map = update.toMap();

        assertEquals("removeRules", map.get("type"));
        assertEquals("allow", map.get("behavior"));
    }

    @Test
    void testAddDirectoriesToMap() {
        PermissionUpdate update = PermissionUpdate.addDirectories("session",
                List.of("/tmp/dir1", "/tmp/dir2"));
        Map<String, Object> map = update.toMap();

        assertEquals("addDirectories", map.get("type"));
        assertEquals("session", map.get("destination"));

        @SuppressWarnings("unchecked")
        List<String> dirs = (List<String>) map.get("directories");
        assertNotNull(dirs, "expected directories list");
        assertEquals(2, dirs.size());
        assertEquals("/tmp/dir1", dirs.get(0));
        assertEquals("/tmp/dir2", dirs.get(1));
        assertFalse(map.containsKey("rules"), "should not have rules");
    }

    @Test
    void testRemoveDirectoriesToMap() {
        PermissionUpdate update = PermissionUpdate.removeDirectories("project",
                List.of("/old/dir"));
        Map<String, Object> map = update.toMap();

        assertEquals("removeDirectories", map.get("type"));
        assertEquals("project", map.get("destination"));

        @SuppressWarnings("unchecked")
        List<String> dirs = (List<String>) map.get("directories");
        assertEquals(1, dirs.size());
        assertEquals("/old/dir", dirs.get(0));
    }

    @Test
    void testPermissionRuleFields() {
        PermissionRule rule = new PermissionRule("MyTool", null);
        assertEquals("MyTool", rule.toolName());
        assertNull(rule.ruleContent());
    }

    @Test
    void testPermissionRuleWithContent() {
        PermissionRule rule = new PermissionRule("Bash", "echo *");
        assertEquals("Bash", rule.toolName());
        assertEquals("echo *", rule.ruleContent());
    }

    @Test
    void testSetModeNoExtraFields() {
        PermissionUpdate update = PermissionUpdate.setMode("bypassPermissions", "session");
        Map<String, Object> map = update.toMap();

        assertTrue(map.containsKey("type"));
        assertTrue(map.containsKey("mode"));
        assertTrue(map.containsKey("destination"));
    }

    @Test
    void testMultipleUpdatesSerialize() {
        List<PermissionUpdate> updates = List.of(
                PermissionUpdate.setMode("acceptEdits", "session"),
                PermissionUpdate.addRules("allow", "session",
                        List.of(new PermissionRule("mcp__echo__ping", null))),
                PermissionUpdate.addDirectories("session", List.of("/tmp/work"))
        );

        List<Map<String, Object>> maps = updates.stream()
                .map(PermissionUpdate::toMap)
                .toList();

        assertEquals(3, maps.size());
        assertEquals("setMode", maps.get(0).get("type"));
        assertEquals("addRules", maps.get(1).get("type"));
        assertEquals("addDirectories", maps.get(2).get("type"));
    }
}
