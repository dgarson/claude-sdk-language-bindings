package client

import (
	"fmt"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// OptionsBuilder builds a pb.ClaudeAgentOptions instance with a fluent API.
//
// This is purely a convenience layer. It does not modify sidecar or SDK semantics;
// it just reduces boilerplate when constructing options in compiled clients.
type OptionsBuilder struct {
	options *pb.ClaudeAgentOptions
	err     error
}

func NewOptions() *OptionsBuilder {
	return &OptionsBuilder{options: &pb.ClaudeAgentOptions{}}
}

func (b *OptionsBuilder) Build() *pb.ClaudeAgentOptions {
	if b == nil {
		return &pb.ClaudeAgentOptions{}
	}
	return b.options
}

func (b *OptionsBuilder) Err() error {
	if b == nil {
		return fmt.Errorf("options are nil")
	}
	return b.err
}

func (b *OptionsBuilder) ToolsList(tools ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.Tools = &pb.ClaudeAgentOptions_ToolsList{
		ToolsList: &pb.ToolsList{Tools: append([]string(nil), tools...)},
	}
	return b
}

func (b *OptionsBuilder) ToolsPresetClaudeCode() *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.Tools = &pb.ClaudeAgentOptions_ToolsPreset{
		ToolsPreset: &pb.ToolsPreset{Type: "preset", Preset: "claude_code"},
	}
	return b
}

func (b *OptionsBuilder) AllowedTools(tools ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.AllowedTools = append([]string(nil), tools...)
	return b
}

func (b *OptionsBuilder) DisallowedTools(tools ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.DisallowedTools = append([]string(nil), tools...)
	return b
}

func (b *OptionsBuilder) SystemPromptText(text string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.SystemPrompt = &pb.ClaudeAgentOptions_SystemPromptText{SystemPromptText: text}
	return b
}

func (b *OptionsBuilder) SystemPromptPresetClaudeCode(appendText string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.SystemPrompt = &pb.ClaudeAgentOptions_SystemPromptPreset{
		SystemPromptPreset: &pb.SystemPromptPreset{
			Type:   "preset",
			Preset: "claude_code",
			Append: appendText,
		},
	}
	return b
}

func (b *OptionsBuilder) PermissionMode(mode string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.PermissionMode = mode
	return b
}

func (b *OptionsBuilder) ContinueConversation(enabled bool) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.ContinueConversation = enabled
	return b
}

func (b *OptionsBuilder) EnableFileCheckpointing(enabled bool) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.EnableFileCheckpointing = enabled
	return b
}

func (b *OptionsBuilder) MaxTurns(max uint32) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.MaxTurns = max
	return b
}

func (b *OptionsBuilder) MaxBudgetUSD(max float64) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.MaxBudgetUsd = max
	return b
}

func (b *OptionsBuilder) Model(model string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.Model = model
	return b
}

func (b *OptionsBuilder) FallbackModel(model string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.FallbackModel = model
	return b
}

func (b *OptionsBuilder) Betas(betas ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.Betas = append([]string(nil), betas...)
	return b
}

func (b *OptionsBuilder) MaxThinkingTokens(max uint32) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.MaxThinkingTokens = max
	return b
}

func (b *OptionsBuilder) OutputJSONSchema(schema map[string]any) *OptionsBuilder {
	if b == nil {
		return b
	}
	if schema == nil {
		b.options.OutputFormat = &pb.OutputFormat{}
		return b
	}
	converted, err := structpb.NewStruct(schema)
	if err != nil {
		if b.err == nil {
			b.err = err
		}
		return b
	}
	b.options.OutputFormat = &pb.OutputFormat{
		Type:   "json_schema",
		Schema: converted,
	}
	return b
}

func (b *OptionsBuilder) PermissionPromptToolName(name string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.PermissionPromptToolName = name
	return b
}

func (b *OptionsBuilder) Cwd(path string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.Cwd = path
	return b
}

func (b *OptionsBuilder) SettingsPath(path string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.SettingsPath = path
	return b
}

func (b *OptionsBuilder) AddDirs(paths ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.AddDirs = append([]string(nil), paths...)
	return b
}

func (b *OptionsBuilder) SettingSources(sources ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.SettingSources = append([]string(nil), sources...)
	return b
}

func (b *OptionsBuilder) CliPath(path string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.CliPath = path
	return b
}

func (b *OptionsBuilder) Env(env map[string]string) *OptionsBuilder {
	if b == nil {
		return b
	}
	if env == nil {
		b.options.Env = nil
		return b
	}
	b.options.Env = make(map[string]string, len(env))
	for k, v := range env {
		b.options.Env[k] = v
	}
	return b
}

func (b *OptionsBuilder) ExtraArgValue(flag string, value any) *OptionsBuilder {
	if b == nil {
		return b
	}
	if b.options.ExtraArgs == nil {
		b.options.ExtraArgs = map[string]*structpb.Value{}
	}
	converted, err := structpb.NewValue(value)
	if err != nil {
		if b.err == nil {
			b.err = err
		}
		return b
	}
	b.options.ExtraArgs[flag] = converted
	return b
}

func (b *OptionsBuilder) ExtraArgBool(flag string, value bool) *OptionsBuilder {
	return b.ExtraArgValue(flag, value)
}

func (b *OptionsBuilder) ExtraArgString(flag string, value string) *OptionsBuilder {
	return b.ExtraArgValue(flag, value)
}

func (b *OptionsBuilder) MaxBufferSize(bytes uint32) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.MaxBufferSize = bytes
	return b
}

func (b *OptionsBuilder) User(user string) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.User = user
	return b
}

func (b *OptionsBuilder) IncludePartialMessages(enabled bool) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.IncludePartialMessages = enabled
	return b
}

func (b *OptionsBuilder) PluginsLocal(paths ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	for _, path := range paths {
		b.options.Plugins = append(b.options.Plugins, &pb.SdkPluginConfig{
			Type: "local",
			Path: path,
		})
	}
	return b
}

func (b *OptionsBuilder) SandboxEnabled(enabled bool) *OptionsBuilder {
	if b == nil {
		return b
	}
	if b.options.Sandbox == nil {
		b.options.Sandbox = &pb.SandboxSettings{}
	}
	b.options.Sandbox.Enabled = enabled
	return b
}

func (b *OptionsBuilder) SandboxExcludedCommands(commands ...string) *OptionsBuilder {
	if b == nil {
		return b
	}
	if b.options.Sandbox == nil {
		b.options.Sandbox = &pb.SandboxSettings{}
	}
	b.options.Sandbox.ExcludedCommands = append([]string(nil), commands...)
	return b
}

func (b *OptionsBuilder) WithMcpStdio(serverKey, command string, args []string, env map[string]string) *OptionsBuilder {
	if b == nil {
		return b
	}
	if b.options.McpServers == nil {
		b.options.McpServers = map[string]*pb.McpServerConfig{}
	}
	cfg := &pb.McpServerConfig{
		Cfg: &pb.McpServerConfig_Stdio{
			Stdio: &pb.McpStdioServerConfig{
				Command: command,
				Args:    append([]string(nil), args...),
				Env:     map[string]string{},
			},
		},
	}
	for k, v := range env {
		cfg.GetStdio().Env[k] = v
	}
	b.options.McpServers[serverKey] = cfg
	return b
}

func (b *OptionsBuilder) WithMcpHttp(serverKey, url string, headers map[string]string) *OptionsBuilder {
	if b == nil {
		return b
	}
	if b.options.McpServers == nil {
		b.options.McpServers = map[string]*pb.McpServerConfig{}
	}
	cfg := &pb.McpServerConfig{
		Cfg: &pb.McpServerConfig_Http{
			Http: &pb.McpHttpServerConfig{
				Url:     url,
				Headers: map[string]string{},
			},
		},
	}
	for k, v := range headers {
		cfg.GetHttp().Headers[k] = v
	}
	b.options.McpServers[serverKey] = cfg
	return b
}

func (b *OptionsBuilder) WithMcpSse(serverKey, url string, headers map[string]string) *OptionsBuilder {
	if b == nil {
		return b
	}
	if b.options.McpServers == nil {
		b.options.McpServers = map[string]*pb.McpServerConfig{}
	}
	cfg := &pb.McpServerConfig{
		Cfg: &pb.McpServerConfig_Sse{
			Sse: &pb.McpSseServerConfig{
				Url:     url,
				Headers: map[string]string{},
			},
		},
	}
	for k, v := range headers {
		cfg.GetSse().Headers[k] = v
	}
	b.options.McpServers[serverKey] = cfg
	return b
}

func (b *OptionsBuilder) WithClientHook(event string, matcher string, timeoutSeconds uint32) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.ClientHooks = append(b.options.ClientHooks, &pb.HookSpec{
		HookEvent:      event,
		Matcher:        matcher,
		TimeoutSeconds: timeoutSeconds,
	})
	return b
}

func (b *OptionsBuilder) WithClientTool(serverKey, name, description string, inputSchema map[string]any) *OptionsBuilder {
	if b == nil {
		return b
	}
	schema, err := structpb.NewStruct(inputSchema)
	if err != nil {
		if b.err == nil {
			b.err = err
		}
		return b
	}
	var server *pb.ClientToolServer
	for _, existing := range b.options.ClientToolServers {
		if existing.GetServerKey() == serverKey {
			server = existing
			break
		}
	}
	if server == nil {
		server = &pb.ClientToolServer{ServerKey: serverKey}
		b.options.ClientToolServers = append(b.options.ClientToolServers, server)
	}
	server.Tools = append(server.Tools, &pb.ToolSpec{
		Name:        name,
		Description: description,
		InputSchema: schema,
	})
	return b
}

func (b *OptionsBuilder) EnablePermissionCallback(enabled bool) *OptionsBuilder {
	if b == nil {
		return b
	}
	b.options.PermissionCallbackEnabled = enabled
	return b
}

// Validate performs lightweight validation of the built options (best-effort).
// This does not enforce SDK semantics beyond basic structural expectations.
func (b *OptionsBuilder) Validate() error {
	if b == nil || b.options == nil {
		return fmt.Errorf("options are nil")
	}
	if b.err != nil {
		return b.err
	}
	if _, ok := b.options.Tools.(*pb.ClaudeAgentOptions_ToolsList); ok {
		// Allow empty tools list for parity with CLI "no preset tools".
		return nil
	}
	return nil
}
