package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for {@link HookOutputs}.
 * Mirrors Go's hook tests in {@code hooks_test.go}.
 */
class HookOutputsTest {

    @Test
    void testHookContinue() {
        Sidecar.HookOutput output = HookOutputs.hookContinue();
        assertNotNull(output);
        assertTrue(output.hasContinue(), "expected continue to be set");
        assertTrue(output.getContinue(), "expected continue=true");
    }

    @Test
    void testHookStop() {
        Sidecar.HookOutput output = HookOutputs.hookStop("stopped for test");
        assertNotNull(output);
        assertTrue(output.hasContinue(), "expected continue to be set");
        assertFalse(output.getContinue(), "expected continue=false");
        assertEquals("stopped for test", output.getStopReason());
    }

    @Test
    void testHookBlock() {
        Sidecar.HookOutput output = HookOutputs.hookBlock("blocked", "warn");
        assertNotNull(output);
        assertTrue(output.hasContinue(), "expected continue to be set");
        assertFalse(output.getContinue(), "expected continue=false");
        assertEquals("block", output.getDecision(), "expected decision=block");
        assertEquals("blocked", output.getReason(), "expected reason=blocked");
        assertEquals("warn", output.getSystemMessage(), "expected system_message=warn");
    }

    @Test
    void testHookAsync() {
        Sidecar.HookOutput output = HookOutputs.hookAsync(1234);
        assertNotNull(output);
        assertTrue(output.hasContinue(), "expected continue to be set");
        assertTrue(output.getContinue(), "expected continue=true");
        assertTrue(output.getAsync(), "expected async=true");
        assertEquals(1234, output.getAsyncTimeoutMs(), "expected async_timeout_ms=1234");
    }

    @Test
    void testHookSuppressOutput() {
        Sidecar.HookOutput output = HookOutputs.hookSuppressOutput("suppressed");
        assertNotNull(output);
        assertTrue(output.getContinue(), "expected continue=true");
        assertTrue(output.getSuppressOutput(), "expected suppress_output=true");
        assertEquals("suppressed", output.getSystemMessage());
    }

    @Test
    void testHookWithSpecificOutput() {
        Sidecar.HookOutput output = HookOutputs.withSpecificOutput(
                HookOutputs.hookContinue(),
                Map.of(
                        "hookEventName", "PreToolUse",
                        "permissionDecision", "deny",
                        "permissionDecisionReason", "nope"
                )
        );
        assertNotNull(output);
        assertTrue(output.hasHookSpecificOutput(),
                "expected HookSpecificOutput to be set");

        Map<String, Object> fields = ProtoUtil.structToMap(
                output.getHookSpecificOutput());
        assertEquals("PreToolUse", fields.get("hookEventName"));
        assertEquals("deny", fields.get("permissionDecision"));
        assertEquals("nope", fields.get("permissionDecisionReason"));
    }

    @Test
    void testHookSpecificPreToolUsePermissionDecision() {
        Sidecar.HookOutput output = HookOutputs.withSpecificOutput(
                HookOutputs.hookContinue(),
                HookSpecificOutputs.preToolUsePermissionDecision("deny", "nope")
        );
        assertNotNull(output);

        Map<String, Object> fields = ProtoUtil.structToMap(
                output.getHookSpecificOutput());
        assertEquals("deny", fields.get("permissionDecision"));
        assertEquals("nope", fields.get("permissionDecisionReason"));
    }

    @Test
    void testHookSpecificPreToolUseUpdatedInput() {
        Sidecar.HookOutput output = HookOutputs.withSpecificOutput(
                HookOutputs.hookContinue(),
                HookSpecificOutputs.preToolUseUpdatedInput(
                        Map.of("text", "rewritten"))
        );
        assertNotNull(output);

        Map<String, Object> fields = ProtoUtil.structToMap(
                output.getHookSpecificOutput());
        @SuppressWarnings("unchecked")
        Map<String, Object> updatedInput = (Map<String, Object>) fields.get("updatedInput");
        assertNotNull(updatedInput, "expected updatedInput");
        assertEquals("rewritten", updatedInput.get("text"));
    }

    @Test
    void testHookSpecificPostToolUseAdditionalContext() {
        Sidecar.HookOutput output = HookOutputs.withSpecificOutput(
                HookOutputs.hookContinue(),
                HookSpecificOutputs.postToolUseAdditionalContext("extra context")
        );
        assertNotNull(output);

        Map<String, Object> fields = ProtoUtil.structToMap(
                output.getHookSpecificOutput());
        assertEquals("extra context", fields.get("additionalContext"));
    }

    @Test
    void testHookDefault() {
        Sidecar.HookOutput output = HookOutputs.hookDefault();
        assertNotNull(output);
        // hookDefault sets continue=true (same as hookContinue)
        assertTrue(output.getContinue(), "expected continue=true");
    }

    @Test
    void testHookShouldContinueLogic() {
        // continue=true -> true
        assertTrue(HookOutputs.hookShouldContinue(HookOutputs.hookContinue()));

        // continue=false -> false
        assertFalse(HookOutputs.hookShouldContinue(HookOutputs.hookStop("stop")));

        // No continue set -> true (default)
        Sidecar.HookOutput emptyOutput = Sidecar.HookOutput.getDefaultInstance();
        assertTrue(HookOutputs.hookShouldContinue(emptyOutput));

        // hookDefault -> true
        assertTrue(HookOutputs.hookShouldContinue(HookOutputs.hookDefault()));
    }
}
