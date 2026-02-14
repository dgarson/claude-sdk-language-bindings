package client

import (
	"fmt"

	structpb "google.golang.org/protobuf/types/known/structpb"
)

func ToolResultBlocks(blocks []ContentBlock, isError bool, extra map[string]any) *structpb.Struct {
	content := make([]any, 0, len(blocks))
	for _, block := range blocks {
		content = append(content, map[string]any(block))
	}
	payload := map[string]any{
		"content":  content,
		"is_error": isError,
	}
	for key, value := range extra {
		payload[key] = value
	}
	result, _ := structpb.NewStruct(payload)
	return result
}

func ToolResultRaw(result map[string]any) *structpb.Struct {
	structured, _ := structpb.NewStruct(result)
	return structured
}

func ToolResultJSON(value any) *structpb.Struct {
	return ToolResultBlocks([]ContentBlock{ContentBlockJSON(value)}, false, nil)
}

func ToolResultWithMetadata(blocks []ContentBlock, isError bool, metadata map[string]any) *structpb.Struct {
	if metadata == nil {
		return ToolResultBlocks(blocks, isError, nil)
	}
	extra := map[string]any{"meta": metadata}
	return ToolResultBlocks(blocks, isError, extra)
}

func ToolResultValidated(
	schema map[string]any,
	payload map[string]any,
) (*structpb.Struct, error) {
	if err := ValidateJSONSchema(schema, payload); err != nil {
		return nil, fmt.Errorf("tool result schema validation failed: %w", err)
	}
	return ToolResultRaw(payload), nil
}
