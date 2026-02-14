package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for {@link PermissionDecisions}.
 * Mirrors Go's {@code permissions_test.go}.
 */
class PermissionDecisionsTest {

    @Test
    void testAllow() {
        Sidecar.PermissionDecision decision = PermissionDecisions.allow("ok");
        assertEquals("allow", decision.getBehavior());
        assertEquals("ok", decision.getReason());
    }

    @Test
    void testDeny() {
        Sidecar.PermissionDecision decision = PermissionDecisions.deny("not allowed");
        assertEquals("deny", decision.getBehavior());
        assertEquals("not allowed", decision.getReason());
    }

    @Test
    void testAsk() {
        Sidecar.PermissionDecision decision = PermissionDecisions.ask("confirm");
        assertEquals("ask", decision.getBehavior());
        assertEquals("confirm", decision.getReason());
    }

    @Test
    void testWithUpdatedPermissions() {
        Sidecar.PermissionDecision decision = PermissionDecisions.allow("ok");
        decision = PermissionDecisions.withUpdatedPermissions(decision, List.of(
                PermissionUpdate.setMode("acceptEdits")
        ));

        assertTrue(decision.hasUpdatedPermissions(),
                "expected UpdatedPermissions to be set");

        // Verify the value is a list
        Object raw = decision.getUpdatedPermissions().getListValue();
        assertNotNull(raw, "expected list value");
    }

    @Test
    void testWithUpdatedPermissionsAddRules() {
        Sidecar.PermissionDecision decision = PermissionDecisions.allow("ok");
        decision = PermissionDecisions.withUpdatedPermissions(decision, List.of(
                PermissionUpdate.addRules("allow", "session",
                        List.of(new PermissionRule("mcp__echo__ping", null)))
        ));

        assertTrue(decision.hasUpdatedPermissions(),
                "expected UpdatedPermissions to be set");
    }

    @Test
    void testWithInterrupt() {
        Sidecar.PermissionDecision decision = PermissionDecisions.deny("stop");
        decision = PermissionDecisions.withInterrupt(decision, true);

        assertTrue(decision.getInterrupt(),
                "expected interrupt=true");
        assertEquals("deny", decision.getBehavior());
    }

    @Test
    void testWithInterruptFalse() {
        Sidecar.PermissionDecision decision = PermissionDecisions.deny("stop");
        decision = PermissionDecisions.withInterrupt(decision, false);

        assertFalse(decision.getInterrupt(),
                "expected interrupt=false");
    }

    @Test
    void testWithUpdatedInput() {
        Sidecar.PermissionDecision decision = PermissionDecisions.allow("ok");
        decision = PermissionDecisions.withUpdatedInput(decision,
                java.util.Map.of("key", "value"));

        assertTrue(decision.hasUpdatedInput(),
                "expected UpdatedInput to be set");
        java.util.Map<String, Object> inputMap = ProtoUtil.structToMap(
                decision.getUpdatedInput());
        assertEquals("value", inputMap.get("key"));
    }

    @Test
    void testAllowWithAllExtensions() {
        Sidecar.PermissionDecision decision = PermissionDecisions.allow("allowed");
        decision = PermissionDecisions.withUpdatedPermissions(decision, List.of(
                PermissionUpdate.setMode("acceptEdits"),
                PermissionUpdate.addRules("allow", "session",
                        List.of(new PermissionRule("mcp__echo__ping", null)))
        ));
        decision = PermissionDecisions.withUpdatedInput(decision,
                java.util.Map.of("text", "modified"));
        decision = PermissionDecisions.withInterrupt(decision, false);

        assertEquals("allow", decision.getBehavior());
        assertEquals("allowed", decision.getReason());
        assertTrue(decision.hasUpdatedPermissions());
        assertTrue(decision.hasUpdatedInput());
        assertFalse(decision.getInterrupt());
    }
}
