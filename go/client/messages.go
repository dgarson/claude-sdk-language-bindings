package client

import (
	"fmt"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

type MessageKind string

const (
	MessageKindUser        MessageKind = "user"
	MessageKindAssistant   MessageKind = "assistant"
	MessageKindSystem      MessageKind = "system"
	MessageKindResult      MessageKind = "result"
	MessageKindStreamEvent MessageKind = "stream_event"
)

type Message interface {
	Kind() MessageKind
}

type ParsedMessage struct {
	Message   Message
	IsPartial bool
}

type UserMessage struct {
	Content         []MessageBlock
	CheckpointUUID  string
	ParentToolUseID string
}

func (m UserMessage) Kind() MessageKind { return MessageKindUser }

type AssistantMessage struct {
	Content         []MessageBlock
	Model           string
	ParentToolUseID string
	Error           string
}

func (m AssistantMessage) Kind() MessageKind { return MessageKindAssistant }

type SystemMessage struct {
	Subtype string
	Data    map[string]any
}

func (m SystemMessage) Kind() MessageKind { return MessageKindSystem }

type ResultMessage struct {
	Subtype          string
	DurationMs       uint64
	DurationApiMs    uint64
	IsError          bool
	NumTurns         uint32
	SessionID        string
	TotalCostUSD     float64
	Usage            map[string]any
	Result           string
	StructuredOutput map[string]any
}

func (m ResultMessage) Kind() MessageKind { return MessageKindResult }

type StreamEventMessage struct {
	UUID            string
	SessionID       string
	Event           map[string]any
	ParentToolUseID string
}

func (m StreamEventMessage) Kind() MessageKind { return MessageKindStreamEvent }

func ParseMessageEvent(event *pb.MessageEvent) (*ParsedMessage, error) {
	if event == nil {
		return nil, fmt.Errorf("message event is nil")
	}
	message, err := MessageFromEvent(event)
	if err != nil {
		return nil, err
	}
	return &ParsedMessage{Message: message, IsPartial: event.IsPartial}, nil
}

func MessageFromEvent(event *pb.MessageEvent) (Message, error) {
	if event == nil {
		return nil, fmt.Errorf("message event is nil")
	}
	switch payload := event.Msg.(type) {
	case *pb.MessageEvent_User:
		content, err := MessageBlocksFromProto(payload.User.Content)
		if err != nil {
			return nil, err
		}
		return UserMessage{
			Content:         content,
			CheckpointUUID:  payload.User.CheckpointUuid,
			ParentToolUseID: payload.User.ParentToolUseId,
		}, nil
	case *pb.MessageEvent_Assistant:
		content, err := MessageBlocksFromProto(payload.Assistant.Content)
		if err != nil {
			return nil, err
		}
		return AssistantMessage{
			Content:         content,
			Model:           payload.Assistant.Model,
			ParentToolUseID: payload.Assistant.ParentToolUseId,
			Error:           payload.Assistant.Error,
		}, nil
	case *pb.MessageEvent_System:
		return SystemMessage{
			Subtype: payload.System.Subtype,
			Data:    StructToMap(payload.System.Data),
		}, nil
	case *pb.MessageEvent_Result:
		return ResultMessage{
			Subtype:          payload.Result.Subtype,
			DurationMs:       payload.Result.DurationMs,
			DurationApiMs:    payload.Result.DurationApiMs,
			IsError:          payload.Result.IsError,
			NumTurns:         payload.Result.NumTurns,
			SessionID:        payload.Result.ClaudeSessionId,
			TotalCostUSD:     payload.Result.TotalCostUsd,
			Usage:            StructToMap(payload.Result.Usage),
			Result:           payload.Result.Result,
			StructuredOutput: StructToMap(payload.Result.StructuredOutput),
		}, nil
	case *pb.MessageEvent_StreamEvent:
		return StreamEventMessage{
			UUID:            payload.StreamEvent.Uuid,
			SessionID:       payload.StreamEvent.SessionId,
			Event:           StructToMap(payload.StreamEvent.Event),
			ParentToolUseID: payload.StreamEvent.ParentToolUseId,
		}, nil
	default:
		return nil, fmt.Errorf("unknown message event")
	}
}
