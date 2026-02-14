package client

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	"google.golang.org/grpc/metadata"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type fakeStream struct {
	mu     sync.Mutex
	sent   []*pb.ClientEvent
	recvCh chan *pb.ServerEvent
	ctx    context.Context
}

func newFakeStream(ctx context.Context) *fakeStream {
	return &fakeStream{
		recvCh: make(chan *pb.ServerEvent, 8),
		ctx:    ctx,
	}
}

func (f *fakeStream) Send(event *pb.ClientEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, event)
	return nil
}

func (f *fakeStream) Recv() (*pb.ServerEvent, error) {
	event, ok := <-f.recvCh
	if !ok {
		return nil, io.EOF
	}
	return event, nil
}

func (f *fakeStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeStream) Trailer() metadata.MD         { return nil }
func (f *fakeStream) CloseSend() error             { return nil }
func (f *fakeStream) Context() context.Context     { return f.ctx }
func (f *fakeStream) SendMsg(m any) error          { return nil }
func (f *fakeStream) RecvMsg(m any) error          { return nil }

func (f *fakeStream) push(event *pb.ServerEvent) {
	f.recvCh <- event
}

func (f *fakeStream) closeRecv() {
	close(f.recvCh)
}

func (f *fakeStream) Sent() []*pb.ClientEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	sent := make([]*pb.ClientEvent, len(f.sent))
	copy(sent, f.sent)
	return sent
}

func TestSessionQuerySetsSessionID(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-1", stream, Handlers{})

	_, err := session.Query(ctx, "hello")
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	sent := stream.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 send, got %d", len(sent))
	}
	if sent[0].SidecarSessionId != "sess-1" {
		t.Fatalf("expected session id on query, got %q", sent[0].SidecarSessionId)
	}
	if _, ok := sent[0].Payload.(*pb.ClientEvent_Query); !ok {
		t.Fatalf("expected query payload")
	}
}

func TestToolHandlerSendsResponse(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-2", stream, Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			return ToolResultText("ok"), nil
		},
	})
	session.startRecvLoop()
	defer stream.closeRecv()

	stream.push(&pb.ServerEvent{Payload: &pb.ServerEvent_ToolRequest{
		ToolRequest: &pb.ToolInvocationRequest{
			InvocationId: "inv-1",
			ToolFqn:      "mcp__echo__ping",
		},
	}})

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for tool response")
		default:
		}
		for _, event := range stream.Sent() {
			payload, ok := event.Payload.(*pb.ClientEvent_ToolResponse)
			if !ok {
				continue
			}
			if payload.ToolResponse.InvocationId != "inv-1" {
				continue
			}
			if event.SidecarSessionId != "sess-2" {
				t.Fatalf("expected session id on tool response, got %q", event.SidecarSessionId)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStartInputStream(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-3", stream, Handlers{})

	requestID, streamID, err := session.StartInputStream(ctx)
	if err != nil {
		t.Fatalf("start input stream: %v", err)
	}
	if requestID == "" || streamID == "" {
		t.Fatalf("expected ids to be set")
	}

	sent := stream.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 send, got %d", len(sent))
	}
	payload, ok := sent[0].Payload.(*pb.ClientEvent_Query)
	if !ok {
		t.Fatalf("expected query payload")
	}
	if payload.Query.GetInputStreamId() != streamID {
		t.Fatalf("expected input stream id %q, got %q", streamID, payload.Query.GetInputStreamId())
	}
	if sent[0].SidecarSessionId != "sess-3" {
		t.Fatalf("expected session id on stream start, got %q", sent[0].SidecarSessionId)
	}
}

func TestPermissionHandlerSendsResponse(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-4", stream, Handlers{
		Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
			return &pb.PermissionDecision{Behavior: "allow"}, nil
		},
	})
	session.startRecvLoop()
	defer stream.closeRecv()

	stream.push(&pb.ServerEvent{Payload: &pb.ServerEvent_PermissionRequest{
		PermissionRequest: &pb.PermissionDecisionRequest{
			InvocationId: "perm-1",
			ToolName:     "mcp__echo__ping",
		},
	}})

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for permission response")
		default:
		}
		for _, event := range stream.Sent() {
			payload, ok := event.Payload.(*pb.ClientEvent_PermissionResponse)
			if !ok {
				continue
			}
			if payload.PermissionResponse.InvocationId != "perm-1" {
				continue
			}
			if event.SidecarSessionId != "sess-4" {
				t.Fatalf("expected session id on permission response, got %q", event.SidecarSessionId)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEventMuxRoutesByRequest(t *testing.T) {
	ctx := context.Background()
	stream := newFakeStream(ctx)
	session := newSession(ctx, "sess-mux", stream, Handlers{})
	session.startRecvLoop()
	defer stream.closeRecv()

	reqA := "req-a"
	reqB := "req-b"
	subA := session.mux.SubscribeRequest(reqA, 4)
	subB := session.mux.SubscribeRequest(reqB, 4)

	stream.push(&pb.ServerEvent{
		RequestId: reqA,
		TurnId:    "turn-a",
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_BEGIN},
		},
	})
	stream.push(&pb.ServerEvent{
		RequestId: reqB,
		TurnId:    "turn-b",
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_BEGIN},
		},
	})

	select {
	case event := <-subA.Chan():
		if event.GetRequestId() != reqA {
			t.Fatalf("expected request %s, got %s", reqA, event.GetRequestId())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reqA event")
	}

	select {
	case event := <-subB.Chan():
		if event.GetRequestId() != reqB {
			t.Fatalf("expected request %s, got %s", reqB, event.GetRequestId())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reqB event")
	}

	stream.push(&pb.ServerEvent{
		RequestId: reqA,
		TurnId:    "turn-a",
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_END},
		},
	})

	select {
	case event, ok := <-subA.Chan():
		if !ok {
			t.Fatal("expected turn end event before close")
		}
		payload, ok := event.Payload.(*pb.ServerEvent_Turn)
		if !ok || payload.Turn.Kind != pb.TurnBoundary_TURN_END {
			t.Fatal("expected turn end event")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subA close")
	}
	select {
	case _, ok := <-subA.Chan():
		if ok {
			t.Fatal("expected subA to close after turn end")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subA close")
	}

	stream.push(&pb.ServerEvent{
		RequestId: reqB,
		TurnId:    "turn-b",
		Payload: &pb.ServerEvent_Turn{
			Turn: &pb.TurnBoundary{Kind: pb.TurnBoundary_TURN_END},
		},
	})
	select {
	case event, ok := <-subB.Chan():
		if !ok {
			t.Fatal("expected turn end event before close")
		}
		payload, ok := event.Payload.(*pb.ServerEvent_Turn)
		if !ok || payload.Turn.Kind != pb.TurnBoundary_TURN_END {
			t.Fatal("expected turn end event")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subB close")
	}
	select {
	case _, ok := <-subB.Chan():
		if ok {
			t.Fatal("expected subB to close after turn end")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subB close")
	}
}
