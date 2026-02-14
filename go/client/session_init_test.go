package client

import (
	"testing"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func TestParseSessionInit(t *testing.T) {
	raw, _ := structpb.NewStruct(map[string]any{
		"commands":     []any{"cmd-a", "cmd-b"},
		"output_style": "default",
	})
	init := &pb.SessionInit{
		ClaudeSessionId: "claude-1",
		Tools:           []string{"tool-a", "tool-b"},
		RawInit:         raw,
	}
	info := ParseSessionInit(init)
	if info == nil {
		t.Fatal("expected session init info")
	}
	if info.ClaudeSessionID != "claude-1" {
		t.Fatalf("expected claude session id")
	}
	if len(info.Tools) != 2 {
		t.Fatalf("expected tools to be copied")
	}
	commands := info.Commands()
	if len(commands) != 2 || commands[0] != "cmd-a" {
		t.Fatalf("expected commands in raw init")
	}
	if info.OutputStyle() != "default" {
		t.Fatalf("expected output_style")
	}
}
