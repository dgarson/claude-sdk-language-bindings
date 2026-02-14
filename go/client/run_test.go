package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestStreamAggregatesTurn(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-run", stream, Handlers{})
	session.startRecvLoop()
	defer stream.closeRecv()

	run, err := session.Stream(ctx, "hello")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	requestID := run.RequestID()

	stream.push(&pb.ServerEvent{
		RequestId: requestID,
		TurnId:    "turn-1",
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_BEGIN, TurnIndex: 1},
		},
	})
	stream.push(&pb.ServerEvent{
		RequestId: requestID,
		TurnId:    "turn-1",
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				IsPartial: true,
				Msg: &pb.MessageEvent_Assistant{
					Assistant: &pb.AssistantMessage{Model: "partial"},
				},
			},
		},
	})
	stream.push(&pb.ServerEvent{
		RequestId: requestID,
		TurnId:    "turn-1",
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				Msg: &pb.MessageEvent_Assistant{
					Assistant: &pb.AssistantMessage{Model: "final"},
				},
			},
		},
	})
	stream.push(&pb.ServerEvent{
		RequestId: requestID,
		TurnId:    "turn-1",
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				Msg: &pb.MessageEvent_Result{
					Result: &pb.ResultMessage{Result: "ok"},
				},
			},
		},
	})
	stream.push(&pb.ServerEvent{
		RequestId: requestID,
		TurnId:    "turn-1",
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_END, TurnIndex: 1},
		},
	})

	select {
	case partial := <-run.Partials():
		if partial.GetAssistant() == nil || partial.GetAssistant().Model != "partial" {
			t.Fatalf("expected partial assistant message")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for partial")
	}

	result, err := run.Result(context.Background())
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	if result.Turn == nil || result.Turn.TurnID != "turn-1" {
		t.Fatalf("expected turn data to be populated")
	}
	if result.Assistant() == nil || result.Assistant().Model != "final" {
		t.Fatalf("expected merged assistant message to be final")
	}
	if result.Result() == nil || result.Result().Result != "ok" {
		t.Fatalf("expected result message")
	}
}
