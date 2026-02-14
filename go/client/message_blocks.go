package client

import (
	"fmt"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type MessageBlockKind string

const (
	MessageBlockText       MessageBlockKind = "text"
	MessageBlockThinking   MessageBlockKind = "thinking"
	MessageBlockToolUse    MessageBlockKind = "tool_use"
	MessageBlockToolResult MessageBlockKind = "tool_result"
)

type MessageBlock interface {
	Kind() MessageBlockKind
}

type TextBlock struct {
	Text string
}

func (b TextBlock) Kind() MessageBlockKind { return MessageBlockText }

type ThinkingBlock struct {
	Thinking  string
	Signature string
}

func (b ThinkingBlock) Kind() MessageBlockKind { return MessageBlockThinking }

type ToolUseBlock struct {
	ID    string
	Name  string
	Input map[string]any
}

func (b ToolUseBlock) Kind() MessageBlockKind { return MessageBlockToolUse }

type ToolResultBlock struct {
	ToolUseID string
	Content   any
	IsError   bool
}

func (b ToolResultBlock) Kind() MessageBlockKind { return MessageBlockToolResult }

func MessageBlockFromProto(block *pb.ContentBlock) (MessageBlock, error) {
	if block == nil {
		return nil, fmt.Errorf("content block is nil")
	}
	switch payload := block.Block.(type) {
	case *pb.ContentBlock_Text:
		return TextBlock{Text: payload.Text.Text}, nil
	case *pb.ContentBlock_Thinking:
		return ThinkingBlock{
			Thinking:  payload.Thinking.Thinking,
			Signature: payload.Thinking.Signature,
		}, nil
	case *pb.ContentBlock_ToolUse:
		return ToolUseBlock{
			ID:    payload.ToolUse.Id,
			Name:  payload.ToolUse.Name,
			Input: StructToMap(payload.ToolUse.Input),
		}, nil
	case *pb.ContentBlock_ToolResult:
		var content any
		if payload.ToolResult.Content != nil {
			content = payload.ToolResult.Content.AsInterface()
		}
		return ToolResultBlock{
			ToolUseID: payload.ToolResult.ToolUseId,
			Content:   content,
			IsError:   payload.ToolResult.IsError,
		}, nil
	default:
		return nil, fmt.Errorf("unknown content block")
	}
}

func MessageBlocksFromProto(blocks []*pb.ContentBlock) ([]MessageBlock, error) {
	if len(blocks) == 0 {
		return nil, nil
	}
	out := make([]MessageBlock, 0, len(blocks))
	for _, block := range blocks {
		parsed, err := MessageBlockFromProto(block)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return out, nil
}

func MessageBlockToProto(block MessageBlock) (*pb.ContentBlock, error) {
	switch payload := block.(type) {
	case TextBlock:
		return &pb.ContentBlock{
			Block: &pb.ContentBlock_Text{
				Text: &pb.TextBlock{Text: payload.Text},
			},
		}, nil
	case ThinkingBlock:
		return &pb.ContentBlock{
			Block: &pb.ContentBlock_Thinking{
				Thinking: &pb.ThinkingBlock{
					Thinking:  payload.Thinking,
					Signature: payload.Signature,
				},
			},
		}, nil
	case ToolUseBlock:
		input, err := MapToStruct(payload.Input)
		if err != nil {
			return nil, err
		}
		return &pb.ContentBlock{
			Block: &pb.ContentBlock_ToolUse{
				ToolUse: &pb.ToolUseBlock{
					Id:    payload.ID,
					Name:  payload.Name,
					Input: input,
				},
			},
		}, nil
	case ToolResultBlock:
		var content *structpb.Value
		if payload.Content != nil {
			value, err := structpb.NewValue(payload.Content)
			if err != nil {
				return nil, err
			}
			content = value
		}
		return &pb.ContentBlock{
			Block: &pb.ContentBlock_ToolResult{
				ToolResult: &pb.ToolResultBlock{
					ToolUseId: payload.ToolUseID,
					Content:   content,
					IsError:   payload.IsError,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported content block type")
	}
}

func MessageBlocksToProto(blocks []MessageBlock) ([]*pb.ContentBlock, error) {
	if len(blocks) == 0 {
		return nil, nil
	}
	out := make([]*pb.ContentBlock, 0, len(blocks))
	for _, block := range blocks {
		converted, err := MessageBlockToProto(block)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}
