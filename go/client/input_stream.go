package client

import (
	"errors"

	structpb "google.golang.org/protobuf/types/known/structpb"
)

type InputEvent interface {
	ToMap() map[string]any
}

type RawInputEvent map[string]any

func (e RawInputEvent) ToMap() map[string]any {
	return map[string]any(e)
}

type UserInputMessage struct {
	Role    string
	Content any
}

type UserInputEvent struct {
	Message          UserInputMessage
	ParentToolUseID  string
	SessionID        string
	AdditionalFields map[string]any
}

func UserTextEvent(text string) UserInputEvent {
	return UserInputEvent{
		Message: UserInputMessage{
			Role:    "user",
			Content: text,
		},
	}
}

func UserBlocksEvent(blocks []ContentBlock) UserInputEvent {
	content := make([]any, 0, len(blocks))
	for _, block := range blocks {
		content = append(content, map[string]any(block))
	}
	return UserInputEvent{
		Message: UserInputMessage{
			Role:    "user",
			Content: content,
		},
	}
}

func (e UserInputEvent) ToMap() map[string]any {
	role := e.Message.Role
	if role == "" {
		role = "user"
	}
	payload := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    role,
			"content": e.Message.Content,
		},
	}
	if e.ParentToolUseID != "" {
		payload["parent_tool_use_id"] = e.ParentToolUseID
	}
	if e.SessionID != "" {
		payload["session_id"] = e.SessionID
	}
	for key, value := range e.AdditionalFields {
		payload[key] = value
	}
	return payload
}

type ControlResponseEvent struct {
	Subtype   string
	RequestID string
	Response  map[string]any
	Error     string
}

func ControlResponseSuccess(requestID string, response map[string]any) ControlResponseEvent {
	return ControlResponseEvent{
		Subtype:   "success",
		RequestID: requestID,
		Response:  response,
	}
}

func ControlResponseError(requestID, err string) ControlResponseEvent {
	return ControlResponseEvent{
		Subtype:   "error",
		RequestID: requestID,
		Error:     err,
	}
}

func (e ControlResponseEvent) ToMap() map[string]any {
	subtype := e.Subtype
	if subtype == "" {
		subtype = "success"
	}
	response := map[string]any{
		"subtype":    subtype,
		"request_id": e.RequestID,
	}
	if subtype == "error" {
		response["error"] = e.Error
	} else {
		response["response"] = e.Response
	}
	return map[string]any{
		"type":     "control_response",
		"response": response,
	}
}

func InputEventStruct(event InputEvent) (*structpb.Struct, error) {
	if event == nil {
		return nil, errors.New("input event is nil")
	}
	payload := event.ToMap()
	if payload == nil {
		return nil, errors.New("input event map is nil")
	}
	return structpb.NewStruct(payload)
}
