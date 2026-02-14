package client

import (
	"testing"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestHasCapability(t *testing.T) {
	info := &pb.GetInfoResponse{Capabilities: []string{CapabilityHooks, CapabilitySandbox}}
	if !HasCapability(info, CapabilityHooks) {
		t.Fatalf("expected hooks capability to be present")
	}
	if HasCapability(info, "does_not_exist") {
		t.Fatalf("expected does_not_exist capability to be absent")
	}
}
