package client

import (
	"context"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

const (
	messageKindUser        = "user"
	messageKindAssistant   = "assistant"
	messageKindSystem      = "system"
	messageKindResult      = "result"
	messageKindStreamEvent = "stream_event"
)

type Turn struct {
	RequestID    string
	TurnID       string
	TurnIndex    uint32
	Started      bool
	Ended        bool
	Events       []*pb.ServerEvent
	Messages     []*pb.MessageEvent
	Partials     []*pb.MessageEvent
	StreamEvents []*pb.StreamEvent
	Stderr       []string
	Result       *pb.ResultMessage
	Errors       []*pb.SidecarError
	latest       map[string]*pb.MessageEvent
}

func newTurn(turnID string) *Turn {
	return &Turn{
		TurnID: turnID,
		latest: map[string]*pb.MessageEvent{},
	}
}

func (t *Turn) LatestMessage(kind string) *pb.MessageEvent {
	if t == nil {
		return nil
	}
	return t.latest[kind]
}

func (t *Turn) LatestUser() *pb.UserMessage {
	event := t.LatestMessage(messageKindUser)
	if event == nil {
		return nil
	}
	return event.GetUser()
}

func (t *Turn) LatestAssistant() *pb.AssistantMessage {
	event := t.LatestMessage(messageKindAssistant)
	if event == nil {
		return nil
	}
	return event.GetAssistant()
}

func (t *Turn) MergedAssistant() *pb.AssistantMessage {
	if t == nil {
		return nil
	}
	if assistant := t.LatestAssistant(); assistant != nil {
		return assistant
	}
	for i := len(t.Partials) - 1; i >= 0; i-- {
		if partial := t.Partials[i].GetAssistant(); partial != nil {
			return partial
		}
	}
	return nil
}

func (t *Turn) LatestSystem() *pb.SystemMessage {
	event := t.LatestMessage(messageKindSystem)
	if event == nil {
		return nil
	}
	return event.GetSystem()
}

func (t *Turn) LatestResult() *pb.ResultMessage {
	if t == nil {
		return nil
	}
	if t.Result != nil {
		return t.Result
	}
	event := t.LatestMessage(messageKindResult)
	if event == nil {
		return nil
	}
	return event.GetResult()
}

func (t *Turn) LatestStreamEvent() *pb.StreamEvent {
	event := t.LatestMessage(messageKindStreamEvent)
	if event == nil {
		return nil
	}
	return event.GetStreamEvent()
}

func CollectTurns(ctx context.Context, events <-chan *pb.ServerEvent) <-chan *Turn {
	out := make(chan *Turn, 16)
	go func() {
		defer close(out)
		turns := map[string]*Turn{}
		for {
			select {
			case <-ctx.Done():
				for _, turn := range turns {
					out <- turn
				}
				return
			case event, ok := <-events:
				if !ok {
					for _, turn := range turns {
						out <- turn
					}
					return
				}
				turnID := event.GetTurnId()
				if turnID == "" {
					continue
				}
				turn := turns[turnID]
				if turn == nil {
					turn = newTurn(turnID)
					turns[turnID] = turn
				}
				if turn.RequestID == "" && event.GetRequestId() != "" {
					turn.RequestID = event.GetRequestId()
				}
				turn.Events = append(turn.Events, event)
				switch payload := event.Payload.(type) {
				case *pb.ServerEvent_Turn:
					switch payload.Turn.Kind {
					case pb.TurnBoundary_TURN_BEGIN:
						turn.Started = true
						turn.TurnIndex = payload.Turn.TurnIndex
					case pb.TurnBoundary_TURN_END:
						turn.Ended = true
						if turn.TurnIndex == 0 {
							turn.TurnIndex = payload.Turn.TurnIndex
						}
						out <- turn
						delete(turns, turnID)
					}
				case *pb.ServerEvent_Message:
					turn.addMessage(payload.Message)
				case *pb.ServerEvent_StderrLine:
					turn.Stderr = append(turn.Stderr, payload.StderrLine.Line)
				case *pb.ServerEvent_Error:
					turn.Errors = append(turn.Errors, payload.Error)
				}
			}
		}
	}()
	return out
}

func (t *Turn) addMessage(message *pb.MessageEvent) {
	if message == nil {
		return
	}
	if message.IsPartial {
		t.Partials = append(t.Partials, message)
	} else {
		t.Messages = append(t.Messages, message)
	}
	kind := messageKind(message)
	if kind != "" {
		t.latest[kind] = message
	}
	if streamEvent := message.GetStreamEvent(); streamEvent != nil {
		t.StreamEvents = append(t.StreamEvents, streamEvent)
	}
	if result := message.GetResult(); result != nil {
		t.Result = result
	}
}

func messageKind(message *pb.MessageEvent) string {
	switch message.Msg.(type) {
	case *pb.MessageEvent_User:
		return messageKindUser
	case *pb.MessageEvent_Assistant:
		return messageKindAssistant
	case *pb.MessageEvent_System:
		return messageKindSystem
	case *pb.MessageEvent_Result:
		return messageKindResult
	case *pb.MessageEvent_StreamEvent:
		return messageKindStreamEvent
	default:
		return ""
	}
}
