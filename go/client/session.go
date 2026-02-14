package client

import (
	"context"
	"io"
	"sync"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type Session struct {
	stream    pb.ClaudeSidecar_AttachSessionClient
	events    <-chan *pb.ServerEvent
	sendMu    sync.Mutex
	handlers  Handlers
	ctx       context.Context
	cancel    context.CancelFunc
	sessionID string
	mux       *eventMux
}

func newSession(
	ctx context.Context,
	sessionID string,
	stream pb.ClaudeSidecar_AttachSessionClient,
	handlers Handlers,
) *Session {
	sessionCtx, cancel := context.WithCancel(ctx)
	mux := newEventMux()
	events := mux.SubscribeAll(256).Chan()
	return &Session{
		stream:    stream,
		events:    events,
		handlers:  handlers,
		ctx:       sessionCtx,
		cancel:    cancel,
		sessionID: sessionID,
		mux:       mux,
	}
}

func (s *Session) Events() <-chan *pb.ServerEvent {
	return s.events
}

// Turns consumes the Events channel; use one or the other for a session.
func (s *Session) Turns(ctx context.Context) <-chan *Turn {
	return CollectTurns(ctx, s.events)
}

func (s *Session) Close() {
	s.cancel()
	_ = s.stream.CloseSend()
	s.mux.Close()
}

func (s *Session) Query(ctx context.Context, prompt string) (string, error) {
	requestID := newID("req")
	err := s.send(&pb.ClientEvent{
		RequestId:        requestID,
		SidecarSessionId: s.sessionID,
		Payload: &pb.ClientEvent_Query{
			Query: &pb.QueryRequest{Prompt: &pb.QueryRequest_PromptText{PromptText: prompt}},
		},
	})
	return requestID, err
}

func (s *Session) QueryTurn(ctx context.Context, prompt string) (*Turn, error) {
	result, err := s.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}
	if result.Turn == nil {
		return nil, io.EOF
	}
	return result.Turn, nil
}

func (s *Session) StartInputStream(ctx context.Context) (string, string, error) {
	streamID := newID("input")
	requestID := newID("req")
	err := s.send(&pb.ClientEvent{
		RequestId:        requestID,
		SidecarSessionId: s.sessionID,
		Payload: &pb.ClientEvent_Query{
			Query: &pb.QueryRequest{Prompt: &pb.QueryRequest_InputStreamId{InputStreamId: streamID}},
		},
	})
	return requestID, streamID, err
}

func (s *Session) SendInputChunk(ctx context.Context, streamID string, event *structpb.Struct) error {
	return s.send(&pb.ClientEvent{
		SidecarSessionId: s.sessionID,
		Payload: &pb.ClientEvent_InputChunk{
			InputChunk: &pb.StreamInputChunk{
				InputStreamId: streamID,
				Event:         event,
			},
		},
	})
}

func (s *Session) SendInputEvent(ctx context.Context, streamID string, event InputEvent) error {
	payload, err := InputEventStruct(event)
	if err != nil {
		return err
	}
	return s.SendInputChunk(ctx, streamID, payload)
}

func (s *Session) SendInputMap(ctx context.Context, streamID string, event map[string]any) error {
	return s.SendInputEvent(ctx, streamID, RawInputEvent(event))
}

func (s *Session) EndInputStream(ctx context.Context, streamID string) error {
	return s.send(&pb.ClientEvent{
		SidecarSessionId: s.sessionID,
		Payload: &pb.ClientEvent_EndInput{
			EndInput: &pb.EndInputStream{InputStreamId: streamID},
		},
	})
}

func (s *Session) Interrupt(ctx context.Context) error {
	return s.send(&pb.ClientEvent{
		RequestId:        newID("req"),
		SidecarSessionId: s.sessionID,
		Payload:          &pb.ClientEvent_Interrupt{Interrupt: &pb.InterruptRequest{}},
	})
}

func (s *Session) Cancel(ctx context.Context, reason string) error {
	return s.send(&pb.ClientEvent{
		RequestId:        newID("req"),
		SidecarSessionId: s.sessionID,
		Payload:          &pb.ClientEvent_Cancel{Cancel: &pb.CancelRequest{Reason: reason}},
	})
}

func (s *Session) SetPermissionMode(ctx context.Context, mode string) error {
	return s.send(&pb.ClientEvent{
		RequestId:        newID("req"),
		SidecarSessionId: s.sessionID,
		Payload: &pb.ClientEvent_SetPermissionMode{
			SetPermissionMode: &pb.SetPermissionModeRequest{Mode: mode},
		},
	})
}

func (s *Session) SetModel(ctx context.Context, model string) error {
	return s.send(&pb.ClientEvent{
		RequestId:        newID("req"),
		SidecarSessionId: s.sessionID,
		Payload: &pb.ClientEvent_SetModel{
			SetModel: &pb.SetModelRequest{Model: model},
		},
	})
}

func (s *Session) send(event *pb.ClientEvent) error {
	if event.SidecarSessionId == "" {
		event.SidecarSessionId = s.sessionID
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.stream.Send(event)
}

func (s *Session) startRecvLoop() {
	go func() {
		for {
			event, err := s.stream.Recv()
			if err != nil {
				s.mux.Close()
				return
			}
			s.handleCallback(event)
			s.mux.Enqueue(event)
		}
	}()
}

func (s *Session) handleCallback(event *pb.ServerEvent) {
	switch payload := event.Payload.(type) {
	case *pb.ServerEvent_ToolRequest:
		if s.handlers.Tool == nil {
			go s.sendToolResponse(payload.ToolRequest, ToolResultError("missing tool handler"))
			return
		}
		go func(req *pb.ToolInvocationRequest) {
			result, err := s.handlers.Tool(s.ctx, req)
			if err != nil {
				result = ToolResultError(err.Error())
			}
			s.sendToolResponse(req, result)
		}(payload.ToolRequest)
	case *pb.ServerEvent_HookRequest:
		if s.handlers.Hook == nil {
			go s.sendHookResponse(payload.HookRequest, &pb.HookOutput{Continue_: boolPtr(false), StopReason: "no hook handler"})
			return
		}
		go func(req *pb.HookInvocationRequest) {
			output, err := s.handlers.Hook(s.ctx, req)
			if err != nil {
				output = &pb.HookOutput{Continue_: boolPtr(false), StopReason: err.Error()}
			}
			s.sendHookResponse(req, output)
		}(payload.HookRequest)
	case *pb.ServerEvent_PermissionRequest:
		if s.handlers.Permission == nil {
			go s.sendPermissionResponse(payload.PermissionRequest, &pb.PermissionDecision{Behavior: "deny", Reason: "no permission handler"})
			return
		}
		go func(req *pb.PermissionDecisionRequest) {
			decision, err := s.handlers.Permission(s.ctx, req)
			if err != nil {
				decision = &pb.PermissionDecision{Behavior: "deny", Reason: err.Error()}
			}
			s.sendPermissionResponse(req, decision)
		}(payload.PermissionRequest)
	}
}

func (s *Session) sendToolResponse(req *pb.ToolInvocationRequest, result *structpb.Struct) {
	_ = s.send(&pb.ClientEvent{
		Payload: &pb.ClientEvent_ToolResponse{
			ToolResponse: &pb.ToolInvocationResponse{
				InvocationId: req.InvocationId,
				ToolResult:   result,
			},
		},
	})
}

func (s *Session) sendHookResponse(req *pb.HookInvocationRequest, output *pb.HookOutput) {
	_ = s.send(&pb.ClientEvent{
		Payload: &pb.ClientEvent_HookResponse{
			HookResponse: &pb.HookInvocationResponse{
				InvocationId: req.InvocationId,
				Output:       output,
			},
		},
	})
}

func (s *Session) sendPermissionResponse(req *pb.PermissionDecisionRequest, decision *pb.PermissionDecision) {
	_ = s.send(&pb.ClientEvent{
		Payload: &pb.ClientEvent_PermissionResponse{
			PermissionResponse: &pb.PermissionDecisionResponse{
				InvocationId: req.InvocationId,
				Decision:     decision,
			},
		},
	})
}

func DecodeStruct(value *structpb.Struct) map[string]any {
	return StructToMap(value)
}
