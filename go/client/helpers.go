package client

import (
	"crypto/rand"
	"encoding/hex"

	structpb "google.golang.org/protobuf/types/known/structpb"
)

func newID(prefix string) string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}

func StructToMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value.AsMap()
}

func ToolResultText(text string) *structpb.Struct {
	return ToolResultBlocks([]ContentBlock{ContentBlockText(text)}, false, nil)
}

func ToolResultError(text string) *structpb.Struct {
	return ToolResultBlocks([]ContentBlock{ContentBlockText(text)}, true, nil)
}

func MapToStruct(value map[string]any) (*structpb.Struct, error) {
	return structpb.NewStruct(value)
}
