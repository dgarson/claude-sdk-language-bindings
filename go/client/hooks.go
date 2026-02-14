package client

import (
	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func boolPtr(value bool) *bool {
	return &value
}

// HookDefault returns an output with all fields unset.
//
// This is useful when you want Claude Code's default semantics: when "continue"
// is omitted, it defaults to true. (Our internal proto models this via an
// optional bool pointer.)
func HookDefault() *pb.HookOutput {
	return &pb.HookOutput{}
}

func HookContinue() *pb.HookOutput {
	return &pb.HookOutput{Continue_: boolPtr(true)}
}

func HookStop(reason string) *pb.HookOutput {
	return &pb.HookOutput{Continue_: boolPtr(false), StopReason: reason}
}

func HookSuppressOutput(systemMessage string) *pb.HookOutput {
	return &pb.HookOutput{
		Continue_:      boolPtr(true),
		SuppressOutput: true,
		SystemMessage:  systemMessage,
	}
}

func HookAsync(timeoutMs uint32) *pb.HookOutput {
	return &pb.HookOutput{
		Continue_:      boolPtr(true),
		Async_:         true,
		AsyncTimeoutMs: timeoutMs,
	}
}

func HookBlock(reason string, systemMessage string) *pb.HookOutput {
	return &pb.HookOutput{
		Continue_:     boolPtr(false),
		Decision:      "block",
		Reason:        reason,
		SystemMessage: systemMessage,
	}
}

func HookWithSpecificOutput(output *pb.HookOutput, specific map[string]any) (*pb.HookOutput, error) {
	if output == nil {
		output = HookContinue()
	}
	if specific == nil {
		return output, nil
	}
	converted, err := structpb.NewStruct(specific)
	if err != nil {
		return nil, err
	}
	output.HookSpecificOutput = converted
	return output, nil
}

func HookWithSpecific(output *pb.HookOutput, specific HookSpecific) (*pb.HookOutput, error) {
	if specific == nil {
		return HookWithSpecificOutput(output, nil)
	}
	return HookWithSpecificOutput(output, specific.ToMap())
}

// HookShouldContinue applies Claude Code's default semantics: continue=true when
// continue is omitted, and decision="block" blocks even if continue is omitted.
//
// This is a convenience helper for compiled clients; it does not change sidecar
// or Agent SDK behavior.
func HookShouldContinue(output *pb.HookOutput) bool {
	if output == nil {
		return true
	}
	if output.GetDecision() == "block" {
		return false
	}
	if output.Continue_ == nil {
		return true
	}
	return output.GetContinue_()
}
