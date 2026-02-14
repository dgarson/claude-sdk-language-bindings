package client

import (
	"context"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type ClientInfo struct {
	Name     string
	Version  string
	Protocol string
}

type ToolHandler func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error)

type HookHandler func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error)

type PermissionHandler func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error)

type Handlers struct {
	Tool       ToolHandler
	Hook       HookHandler
	Permission PermissionHandler
}
