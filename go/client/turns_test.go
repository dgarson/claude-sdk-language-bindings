package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

func TestCollectTurnsAggregates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events := make(chan *pb.ServerEvent, 16)
	turns := CollectTurns(ctx, events)

	turnID := "turn-1"
	requestID := "req-1"

	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_BEGIN, TurnIndex: 1},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				IsPartial: true,
				Msg: &pb.MessageEvent_Assistant{
					Assistant: &pb.AssistantMessage{Model: "partial"},
				},
			},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				Msg: &pb.MessageEvent_Assistant{
					Assistant: &pb.AssistantMessage{Model: "final"},
				},
			},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				Msg: &pb.MessageEvent_StreamEvent{
					StreamEvent: &pb.StreamEvent{Uuid: "stream-1"},
				},
			},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_StderrLine{
			StderrLine: &pb.StderrLine{Line: "warning"},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Error{
			Error: &pb.SidecarError{Code: "INTERNAL", Message: "boom"},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Message{
			Message: &pb.MessageEvent{
				Msg: &pb.MessageEvent_Result{
					Result: &pb.ResultMessage{Result: "ok"},
				},
			},
		},
	}
	events <- &pb.ServerEvent{
		RequestId: requestID,
		TurnId:    turnID,
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_END, TurnIndex: 1},
		},
	}
	close(events)

	turn, ok := <-turns
	if !ok || turn == nil {
		t.Fatal("expected turn output")
	}
	if turn.TurnID != turnID {
		t.Fatalf("expected turn id %q, got %q", turnID, turn.TurnID)
	}
	if turn.RequestID != requestID {
		t.Fatalf("expected request id %q, got %q", requestID, turn.RequestID)
	}
	if !turn.Started || !turn.Ended {
		t.Fatalf("expected turn boundaries, got started=%v ended=%v", turn.Started, turn.Ended)
	}
	if turn.TurnIndex != 1 {
		t.Fatalf("expected turn index 1, got %d", turn.TurnIndex)
	}
	if len(turn.Partials) != 1 {
		t.Fatalf("expected 1 partial message, got %d", len(turn.Partials))
	}
	if len(turn.Messages) != 3 {
		t.Fatalf("expected 3 final messages, got %d", len(turn.Messages))
	}
	if len(turn.StreamEvents) != 1 || turn.StreamEvents[0].Uuid != "stream-1" {
		t.Fatalf("expected stream event to be recorded")
	}
	if len(turn.Stderr) != 1 || turn.Stderr[0] != "warning" {
		t.Fatalf("expected stderr line to be recorded")
	}
	if len(turn.Errors) != 1 || turn.Errors[0].Message != "boom" {
		t.Fatalf("expected error to be recorded")
	}
	if turn.Result == nil || turn.Result.Result != "ok" {
		t.Fatalf("expected result to be recorded")
	}
	if turn.LatestAssistant() == nil || turn.LatestAssistant().Model != "final" {
		t.Fatalf("expected latest assistant message to be final")
	}
	if turn.LatestStreamEvent() == nil || turn.LatestStreamEvent().Uuid != "stream-1" {
		t.Fatalf("expected latest stream event")
	}
}
