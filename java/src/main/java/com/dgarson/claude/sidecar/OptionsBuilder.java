package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

/**
 * Fluent builder for constructing {@link Sidecar.ClaudeAgentOptions} proto messages.
 *
 * <p>Mirrors the Go SDK's {@code OptionsBuilder}. All setters return {@code this} for chaining.
 *
 * <p>Thread-safety: instances are NOT thread-safe. Build from a single thread or synchronize
 * externally.
 */
public class OptionsBuilder {

    private final Sidecar.ClaudeAgentOptions.Builder builder;
    private final List<Sidecar.HookSpec> hookSpecs = new ArrayList<>();
    private final List<Sidecar.ClientToolServer> clientToolServers = new ArrayList<>();

    private OptionsBuilder() {
        this.builder = Sidecar.ClaudeAgentOptions.newBuilder();
    }

    /**
     * Creates a new empty OptionsBuilder.
     */
    public static OptionsBuilder create() {
        return new OptionsBuilder();
    }

    // --- Tool allow/disallow ---

    public OptionsBuilder allowedTools(String... tools) {
        builder.addAllAllowedTools(Arrays.asList(tools));
        return this;
    }

    public OptionsBuilder disallowedTools(String... tools) {
        builder.addAllDisallowedTools(Arrays.asList(tools));
        return this;
    }

    // --- Model ---

    public OptionsBuilder model(String model) {
        builder.setModel(model);
        return this;
    }

    public OptionsBuilder fallbackModel(String model) {
        builder.setFallbackModel(model);
        return this;
    }

    // --- Token/turn/budget limits ---

    public OptionsBuilder maxThinkingTokens(int tokens) {
        builder.setMaxThinkingTokens(tokens);
        return this;
    }

    public OptionsBuilder maxTurns(int turns) {
        builder.setMaxTurns(turns);
        return this;
    }

    public OptionsBuilder maxBudgetUsd(double budget) {
        builder.setMaxBudgetUsd(budget);
        return this;
    }

    // --- Permission mode ---

    public OptionsBuilder permissionMode(String mode) {
        builder.setPermissionMode(mode);
        return this;
    }

    // --- Flags ---

    public OptionsBuilder includePartialMessages(boolean include) {
        builder.setIncludePartialMessages(include);
        return this;
    }

    public OptionsBuilder enableFileCheckpointing(boolean enable) {
        builder.setEnableFileCheckpointing(enable);
        return this;
    }

    public OptionsBuilder enablePermissionCallback(boolean enable) {
        builder.setPermissionCallbackEnabled(enable);
        return this;
    }

    public OptionsBuilder continueConversation(boolean cont) {
        builder.setContinueConversation(cont);
        return this;
    }

    // --- Working directory ---

    public OptionsBuilder cwd(String cwd) {
        builder.setCwd(cwd);
        return this;
    }

    // --- System prompt ---

    public OptionsBuilder systemPrompt(String text) {
        builder.setSystemPromptText(text);
        return this;
    }

    public OptionsBuilder systemPromptPreset(String preset) {
        builder.setSystemPromptPreset(
                Sidecar.SystemPromptPreset.newBuilder()
                        .setType("preset")
                        .setPreset(preset)
                        .build());
        return this;
    }

    public OptionsBuilder systemPromptPreset(String preset, String append) {
        builder.setSystemPromptPreset(
                Sidecar.SystemPromptPreset.newBuilder()
                        .setType("preset")
                        .setPreset(preset)
                        .setAppend(append)
                        .build());
        return this;
    }

    // --- Tools list/preset ---

    public OptionsBuilder toolsList(String... tools) {
        builder.setToolsList(
                Sidecar.ToolsList.newBuilder()
                        .addAllTools(Arrays.asList(tools))
                        .build());
        return this;
    }

    public OptionsBuilder toolsPreset(String preset) {
        builder.setToolsPreset(
                Sidecar.ToolsPreset.newBuilder()
                        .setType("preset")
                        .setPreset(preset)
                        .build());
        return this;
    }

    // --- Output format ---

    public OptionsBuilder outputFormat(String type, Struct schema) {
        builder.setOutputFormat(
                Sidecar.OutputFormat.newBuilder()
                        .setType(type)
                        .setSchema(schema)
                        .build());
        return this;
    }

    // --- Client hooks ---

    /**
     * Adds a client-side hook specification.
     *
     * @param event          hook event type (e.g. "PreToolUse", "PostToolUse", "Stop")
     * @param matcher        regex matcher (empty string matches all)
     * @param timeoutSeconds timeout in seconds (0 uses server default of 60)
     */
    public OptionsBuilder withClientHook(String event, String matcher, int timeoutSeconds) {
        hookSpecs.add(Sidecar.HookSpec.newBuilder()
                .setHookEvent(event)
                .setMatcher(matcher)
                .setTimeoutSeconds(timeoutSeconds)
                .build());
        return this;
    }

    // --- Client tool servers ---

    /**
     * Registers client-side tools under a server key.
     *
     * @param serverKey unique key for this tool server
     * @param tools     one or more tool specifications
     */
    public OptionsBuilder withClientToolServer(String serverKey, ToolSpec... tools) {
        Sidecar.ClientToolServer.Builder cts = Sidecar.ClientToolServer.newBuilder()
                .setServerKey(serverKey);
        for (ToolSpec ts : tools) {
            cts.addTools(Sidecar.ToolSpec.newBuilder()
                    .setName(ts.name())
                    .setDescription(ts.description())
                    .setInputSchema(ts.inputSchema())
                    .build());
        }
        clientToolServers.add(cts.build());
        return this;
    }

    // --- MCP servers ---

    /**
     * Adds an MCP stdio server configuration.
     */
    public OptionsBuilder withMcpStdio(String key, String command, String... args) {
        builder.putMcpServers(key, Sidecar.McpServerConfig.newBuilder()
                .setStdio(Sidecar.McpStdioServerConfig.newBuilder()
                        .setCommand(command)
                        .addAllArgs(Arrays.asList(args))
                        .build())
                .build());
        return this;
    }

    /**
     * Adds an MCP HTTP server configuration.
     */
    public OptionsBuilder withMcpHttp(String key, String url) {
        builder.putMcpServers(key, Sidecar.McpServerConfig.newBuilder()
                .setHttp(Sidecar.McpHttpServerConfig.newBuilder()
                        .setUrl(url)
                        .build())
                .build());
        return this;
    }

    /**
     * Adds an MCP SSE server configuration.
     */
    public OptionsBuilder withMcpSse(String key, String url) {
        builder.putMcpServers(key, Sidecar.McpServerConfig.newBuilder()
                .setSse(Sidecar.McpSseServerConfig.newBuilder()
                        .setUrl(url)
                        .build())
                .build());
        return this;
    }

    // --- Environment variables ---

    public OptionsBuilder env(String key, String value) {
        builder.putEnv(key, value);
        return this;
    }

    // --- Betas ---

    public OptionsBuilder betas(String... betas) {
        builder.addAllBetas(Arrays.asList(betas));
        return this;
    }

    // --- Additional directories ---

    public OptionsBuilder addDirs(String... dirs) {
        builder.addAllAddDirs(Arrays.asList(dirs));
        return this;
    }

    // --- User ---

    public OptionsBuilder user(String user) {
        builder.setUser(user);
        return this;
    }

    // --- CLI path ---

    public OptionsBuilder cliPath(String path) {
        builder.setCliPath(path);
        return this;
    }

    // --- Max buffer size ---

    public OptionsBuilder maxBufferSize(int size) {
        builder.setMaxBufferSize(size);
        return this;
    }

    // --- Build ---

    /**
     * Builds the immutable {@link Sidecar.ClaudeAgentOptions} proto message.
     */
    public Sidecar.ClaudeAgentOptions build() {
        builder.addAllClientHooks(hookSpecs);
        builder.addAllClientToolServers(clientToolServers);
        return builder.build();
    }
}
