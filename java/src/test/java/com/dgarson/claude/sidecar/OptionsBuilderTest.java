package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;
import com.google.protobuf.Value;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for {@link OptionsBuilder}.
 * Mirrors Go's {@code TestOptionsBuilderSetsFields} in {@code options_builder_test.go}.
 */
class OptionsBuilderTest {

    @Test
    void testBuilderSetsAllFields() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .toolsPreset("claude_code")
                .allowedTools("Read", "Edit")
                .model("claude-sonnet-4-5")
                .maxTurns(2)
                .maxBudgetUsd(1.25)
                .betas("context-1m-2025-08-07")
                .includePartialMessages(true)
                .enablePermissionCallback(true)
                .withClientHook("PreToolUse", "Bash", 10)
                .build();

        assertEquals("claude-sonnet-4-5", options.getModel(),
                "unexpected model");
        assertEquals(2, options.getAllowedToolsCount(),
                "expected allowed_tools size 2");
        assertEquals(2, options.getMaxTurns(),
                "expected max_turns=2");
        assertEquals(1.25, options.getMaxBudgetUsd(), 0.001,
                "expected max_budget_usd=1.25");
        assertTrue(options.getIncludePartialMessages(),
                "expected include_partial_messages=true");

        // Permission callback
        assertTrue(options.getPermissionCallbackEnabled(),
                "expected permission_callback_enabled=true");

        // Client hooks
        assertEquals(1, options.getClientHooksCount(),
                "expected 1 client hook");
        Sidecar.HookSpec hook = options.getClientHooks(0);
        assertEquals("PreToolUse", hook.getHookEvent());
        assertEquals("Bash", hook.getMatcher());
        assertEquals(10, hook.getTimeoutSeconds());
    }

    @Test
    void testBuilderProducesEmptyOptionsWhenNotConfigured() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create().build();
        assertNotNull(options);
        assertEquals("", options.getModel());
        assertEquals(0, options.getAllowedToolsCount());
        assertEquals(0, options.getMaxTurns());
    }

    @Test
    void testBuilderChaining() {
        // Verify fluent API returns same builder for chaining
        OptionsBuilder builder = OptionsBuilder.create();
        OptionsBuilder result = builder
                .model("test")
                .maxTurns(5)
                .cwd("/tmp");

        assertSame(builder, result, "expected fluent chaining to return same builder");
    }

    @Test
    void testBuilderWithToolsList() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .toolsList("Read", "Edit", "Bash")
                .build();

        assertTrue(options.hasToolsList(), "expected tools_list to be set");
        assertEquals(3, options.getToolsList().getToolsCount());
        assertEquals("Read", options.getToolsList().getTools(0));
    }

    @Test
    void testBuilderWithToolsPreset() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .toolsPreset("claude_code")
                .build();

        assertTrue(options.hasToolsPreset(), "expected tools_preset to be set");
        assertEquals("claude_code", options.getToolsPreset().getPreset());
    }

    @Test
    void testBuilderWithSystemPromptText() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .systemPrompt("You are a helpful assistant")
                .build();

        assertEquals("You are a helpful assistant", options.getSystemPromptText());
    }

    @Test
    void testBuilderWithPermissionMode() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .permissionMode("acceptEdits")
                .build();

        assertEquals("acceptEdits", options.getPermissionMode());
    }

    @Test
    void testBuilderWithCwd() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .cwd("/tmp/test")
                .build();

        assertEquals("/tmp/test", options.getCwd());
    }

    @Test
    void testBuilderWithClientTool() {
        Struct schema = ProtoUtil.mapToStruct(java.util.Map.of(
                "type", "object",
                "properties", java.util.Map.of(
                        "input", java.util.Map.of("type", "string")
                )
        ));
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientToolServer("my_server",
                        new ToolSpec("my_tool", "A test tool", schema))
                .build();

        assertEquals(1, options.getClientToolServersCount());
        Sidecar.ClientToolServer server = options.getClientToolServers(0);
        assertEquals("my_server", server.getServerKey());
        assertEquals(1, server.getToolsCount());
        assertEquals("my_tool", server.getTools(0).getName());
        assertEquals("A test tool", server.getTools(0).getDescription());
    }

    @Test
    void testBuilderWithMultipleHooks() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withClientHook("PreToolUse", "Bash", 10)
                .withClientHook("PostToolUse", ".*", 30)
                .build();

        assertEquals(2, options.getClientHooksCount());
        assertEquals("PreToolUse", options.getClientHooks(0).getHookEvent());
        assertEquals("PostToolUse", options.getClientHooks(1).getHookEvent());
    }

    @Test
    void testBuilderWithEnv() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .env("KEY", "VALUE")
                .env("FOO", "BAR")
                .build();

        assertEquals(2, options.getEnvCount());
        assertEquals("VALUE", options.getEnvOrThrow("KEY"));
        assertEquals("BAR", options.getEnvOrThrow("FOO"));
    }

    @Test
    void testBuilderWithMcpStdio() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .withMcpStdio("local", "npx", "-y", "server")
                .build();

        assertTrue(options.containsMcpServers("local"));
        Sidecar.McpServerConfig cfg = options.getMcpServersOrThrow("local");
        assertTrue(cfg.hasStdio());
        assertEquals("npx", cfg.getStdio().getCommand());
        assertEquals(2, cfg.getStdio().getArgsCount());
    }

    @Test
    void testBuilderWithBetas() {
        Sidecar.ClaudeAgentOptions options = OptionsBuilder.create()
                .betas("context-1m-2025-08-07", "extended-thinking")
                .build();

        assertEquals(2, options.getBetasCount());
        assertEquals("context-1m-2025-08-07", options.getBetas(0));
        assertEquals("extended-thinking", options.getBetas(1));
    }
}
