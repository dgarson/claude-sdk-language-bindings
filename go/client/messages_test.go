package client

import (
	"testing"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func TestMessageBlockRoundTrip(t *testing.T) {
	block := &pb.ContentBlock{
		Block: &pb.ContentBlock_Text{
			Text: &pb.TextBlock{Text: "hi"},
		},
	}
	parsed, err := MessageBlockFromProto(block)
	if err != nil {
		t.Fatalf("parse block: %v", err)
	}
	text, ok := parsed.(TextBlock)
	if !ok || text.Text != "hi" {
		t.Fatalf("expected text block hi")
	}
	converted, err := MessageBlockToProto(parsed)
	if err != nil {
		t.Fatalf("convert block: %v", err)
	}
	if converted.GetText() == nil || converted.GetText().Text != "hi" {
		t.Fatalf("expected converted text block")
	}
}

func TestMessageFromEventUser(t *testing.T) {
	event := &pb.MessageEvent{
		Msg: &pb.MessageEvent_User{
			User: &pb.UserMessage{
				CheckpointUuid: "chk-1",
				Content: []*pb.ContentBlock{
					{
						Block: &pb.ContentBlock_Text{
							Text: &pb.TextBlock{Text: "hello"},
						},
					},
				},
			},
		},
	}
	message, err := MessageFromEvent(event)
	if err != nil {
		t.Fatalf("message from event: %v", err)
	}
	user, ok := message.(UserMessage)
	if !ok {
		t.Fatalf("expected user message")
	}
	if user.CheckpointUUID != "chk-1" {
		t.Fatalf("expected checkpoint uuid")
	}
	if len(user.Content) != 1 {
		t.Fatalf("expected content blocks")
	}
}

func TestMessageFromEventStreamEvent(t *testing.T) {
	payload, _ := structpb.NewStruct(map[string]any{"type": "delta"})
	event := &pb.MessageEvent{
		Msg: &pb.MessageEvent_StreamEvent{
			StreamEvent: &pb.StreamEvent{
				Uuid:      "stream-1",
				SessionId: "session-1",
				Event:     payload,
			},
		},
	}
	message, err := MessageFromEvent(event)
	if err != nil {
		t.Fatalf("message from event: %v", err)
	}
	stream, ok := message.(StreamEventMessage)
	if !ok || stream.UUID != "stream-1" {
		t.Fatalf("expected stream event message")
	}
	if stream.Event["type"] != "delta" {
		t.Fatalf("expected stream event payload")
	}
}
