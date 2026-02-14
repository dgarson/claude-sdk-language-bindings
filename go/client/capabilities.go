package client

import pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"

const (
	CapabilityHooks               = "hooks"
	CapabilityPermissions         = "permissions"
	CapabilityPermissionSuggest   = "permission_suggestions"
	CapabilityPermissionUpdates   = "permission_updates"
	CapabilityPermissionInterrupt = "permission_interrupt"
	CapabilitySdkMcp              = "sdk_mcp"
	CapabilityMcpExternal         = "mcp_external"
	CapabilityClientTools         = "client_tools"
	CapabilityCheckpointing       = "checkpointing"
	CapabilityRewindFiles         = "rewind_files"
	CapabilityStructuredOutputs   = "structured_outputs"
	CapabilitySandbox             = "sandbox"
	CapabilityAgents              = "agents"
	CapabilityPlugins             = "plugins"
	CapabilitySessions            = "sessions"
	CapabilityResume              = "resume"
	CapabilityFork                = "fork"
	CapabilityInputStream         = "input_stream"
	CapabilityStderr              = "stderr"
	CapabilityPartialMessages     = "partial_messages"
	CapabilityDynamicControl      = "dynamic_control"
)

func HasCapability(info *pb.GetInfoResponse, capability string) bool {
	if info == nil || capability == "" {
		return false
	}
	for _, item := range info.Capabilities {
		if item == capability {
			return true
		}
	}
	return false
}
