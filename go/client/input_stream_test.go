package client

import (
	"context"
	"testing"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestUserInputEventToMap(t *testing.T) {
	event := UserTextEvent("hello")
	payload := event.ToMap()

	if payload["type"] != "user" {
		t.Fatalf("expected type user, got %v", payload["type"])
	}
	message := payload["message"].(map[string]any)
	if message["role"] != "user" {
		t.Fatalf("expected role user, got %v", message["role"])
	}
	if message["content"] != "hello" {
		t.Fatalf("expected content hello, got %v", message["content"])
	}
}

func TestInputEventStructConversion(t *testing.T) {
	event := UserTextEvent("hi")
	payload, err := InputEventStruct(event)
	if err != nil {
		t.Fatalf("input event struct: %v", err)
	}
	asMap := payload.AsMap()
	if asMap["type"] != "user" {
		t.Fatalf("expected type user, got %v", asMap["type"])
	}
}

func TestSendInputEventUsesChunk(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-input", stream, Handlers{})

	err := session.SendInputEvent(ctx, "stream-1", UserTextEvent("hello"))
	if err != nil {
		t.Fatalf("send input event: %v", err)
	}

	sent := stream.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 send, got %d", len(sent))
	}
	payload, ok := sent[0].Payload.(*pb.ClientEvent_InputChunk)
	if !ok {
		t.Fatalf("expected input chunk payload")
	}
	if payload.InputChunk.InputStreamId != "stream-1" {
		t.Fatalf("expected input stream id stream-1, got %q", payload.InputChunk.InputStreamId)
	}
	asMap := payload.InputChunk.Event.AsMap()
	if asMap["type"] != "user" {
		t.Fatalf("expected user input map, got %v", asMap["type"])
	}
}
