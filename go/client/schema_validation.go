package client

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func ValidateJSONSchema(schema map[string]any, value any) error {
	if schema == nil {
		return nil
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", bytes.NewReader(encoded)); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	if err := compiled.Validate(value); err != nil {
		return err
	}
	return nil
}

func ValidateStructSchema(schema *structpb.Struct, value *structpb.Struct) error {
	if schema == nil {
		return nil
	}
	if value == nil {
		return fmt.Errorf("value is nil")
	}
	return ValidateJSONSchema(schema.AsMap(), value.AsMap())
}
