package client

import (
	"context"
	"io"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

type RunResult struct {
	Turn *Turn
}

func (r *RunResult) Assistant() *pb.AssistantMessage {
	if r == nil || r.Turn == nil {
		return nil
	}
	return r.Turn.MergedAssistant()
}

func (r *RunResult) Result() *pb.ResultMessage {
	if r == nil || r.Turn == nil {
		return nil
	}
	return r.Turn.LatestResult()
}

type Stream struct {
	requestID string
	events    <-chan *pb.ServerEvent
	partials  chan *pb.MessageEvent
	done      chan *RunResult
	err       chan error
	cancel    context.CancelFunc
}

func (s *Stream) RequestID() string {
	return s.requestID
}

func (s *Stream) Events() <-chan *pb.ServerEvent {
	return s.events
}

func (s *Stream) Partials() <-chan *pb.MessageEvent {
	return s.partials
}

func (s *Stream) Done() <-chan *RunResult {
	return s.done
}

func (s *Stream) Err() <-chan error {
	return s.err
}

func (s *Stream) Result(ctx context.Context) (*RunResult, error) {
	for {
		select {
		case result, ok := <-s.done:
			if ok && result != nil {
				return result, nil
			}
			if !ok {
				return nil, io.EOF
			}
		case err, ok := <-s.err:
			if ok && err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (s *Stream) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Session) Run(ctx context.Context, prompt string) (*RunResult, error) {
	stream, err := s.Stream(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return stream.Result(ctx)
}

func (s *Session) Stream(ctx context.Context, prompt string) (*Stream, error) {
	requestID, err := s.Query(ctx, prompt)
	if err != nil {
		return nil, err
	}
	sub := s.mux.SubscribeRequest(requestID, 256)
	streamCtx, cancel := context.WithCancel(ctx)
	stream := &Stream{
		requestID: requestID,
		events:    sub.Chan(),
		partials:  make(chan *pb.MessageEvent, 64),
		done:      make(chan *RunResult, 1),
		err:       make(chan error, 1),
		cancel:    cancel,
	}
	go stream.run(streamCtx, s.mux, sub)
	return stream, nil
}

func (s *Stream) run(ctx context.Context, mux *eventMux, sub *subscription) {
	defer mux.UnsubscribeRequest(s.requestID, sub)
	defer close(s.partials)
	defer close(s.done)

	var turn *Turn

	for {
		select {
		case <-ctx.Done():
			s.err <- ctx.Err()
			return
		case event, ok := <-sub.Chan():
			if !ok {
				if turn != nil {
					s.done <- &RunResult{Turn: turn}
				} else {
					s.err <- io.EOF
				}
				return
			}
			if turn == nil {
				turnID := event.GetTurnId()
				if turnID != "" {
					turn = newTurn(turnID)
				} else {
					continue
				}
			}
			if turn.RequestID == "" && event.GetRequestId() != "" {
				turn.RequestID = event.GetRequestId()
			}
			turn.Events = append(turn.Events, event)
			switch payload := event.Payload.(type) {
			case *pb.ServerEvent_Turn:
				if payload.Turn != nil {
					if payload.Turn.Kind == pb.TurnBoundary_TURN_BEGIN {
						turn.Started = true
						turn.TurnIndex = payload.Turn.TurnIndex
					} else if payload.Turn.Kind == pb.TurnBoundary_TURN_END {
						turn.Ended = true
						if turn.TurnIndex == 0 {
							turn.TurnIndex = payload.Turn.TurnIndex
						}
						s.done <- &RunResult{Turn: turn}
						return
					}
				}
			case *pb.ServerEvent_Message:
				if payload.Message != nil {
					if payload.Message.IsPartial {
						s.partials <- payload.Message
					}
					turn.addMessage(payload.Message)
				}
			case *pb.ServerEvent_StderrLine:
				if payload.StderrLine != nil {
					turn.Stderr = append(turn.Stderr, payload.StderrLine.Line)
				}
			case *pb.ServerEvent_Error:
				if payload.Error != nil {
					turn.Errors = append(turn.Errors, payload.Error)
				}
			}
		}
	}
}
