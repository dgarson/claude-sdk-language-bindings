package client

import (
	"fmt"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

type SessionInitInfo struct {
	ClaudeSessionID string
	Tools           []string
	Raw             map[string]any
}

func ParseSessionInit(init *pb.SessionInit) *SessionInitInfo {
	if init == nil {
		return nil
	}
	return &SessionInitInfo{
		ClaudeSessionID: init.ClaudeSessionId,
		Tools:           append([]string{}, init.Tools...),
		Raw:             StructToMap(init.RawInit),
	}
}

func (s *SessionInitInfo) GetString(key string) string {
	if s == nil {
		return ""
	}
	value, ok := s.Raw[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func (s *SessionInitInfo) GetStringSlice(key string) []string {
	if s == nil {
		return nil
	}
	value, ok := s.Raw[key]
	if !ok {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if str, ok := item.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func (s *SessionInitInfo) GetMap(key string) map[string]any {
	if s == nil {
		return nil
	}
	value, ok := s.Raw[key]
	if !ok {
		return nil
	}
	if asMap, ok := value.(map[string]any); ok {
		return asMap
	}
	return nil
}

func (s *SessionInitInfo) Commands() []string {
	return s.GetStringSlice("commands")
}

func (s *SessionInitInfo) OutputStyle() string {
	return s.GetString("output_style")
}
