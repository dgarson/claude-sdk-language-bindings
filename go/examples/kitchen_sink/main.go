package main

import (
	context "context"
	fmt "fmt"
	log "log"
	os "os"
	time "time"

	sidecar "github.com/dgarson/claude-sidecar/client"
	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func main() {
	addr := os.Getenv("SIDECAR_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}
	useInputStream := os.Getenv("SIDECAR_INPUT_STREAM") != ""
	testMode := os.Getenv("SIDECAR_LIVE") == ""

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := sidecar.Dial(ctx, addr)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer client.Close()

	calcSchema, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "number"},
			"b": map[string]any{"type": "number"},
		},
		"required": []any{"a", "b"},
	})

	options := &pb.ClaudeAgentOptions{
		IncludePartialMessages:    true,
		EnableFileCheckpointing:   true,
		PermissionCallbackEnabled: true,
		AllowedTools:              []string{"mcp__calc__add"},
		ClientToolServers: []*pb.ClientToolServer{
			{
				ServerKey: "calc",
				Tools: []*pb.ToolSpec{
					{
						Name:        "add",
						Description: "Add two numbers",
						InputSchema: calcSchema,
					},
				},
			},
		},
		ClientHooks: []*pb.HookSpec{
			{
				HookEvent:      "PreToolUse",
				Matcher:        "mcp__calc__add",
				TimeoutSeconds: 10,
			},
		},
		OutputFormat: &pb.OutputFormat{
			Type:   "json_schema",
			Schema: calcSchema,
		},
		ExtraArgs: map[string]*structpb.Value{
			"test_mode": structpb.NewBoolValue(testMode),
		},
	}

	createResp, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		Mode:    pb.SessionMode_INTERACTIVE,
		Options: options,
	})
	if err != nil {
		log.Fatalf("create session: %v", err)
	}

	handlers := sidecar.Handlers{
		Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
			log.Printf("tool request: %s", req.ToolFqn)
			if req.ToolFqn != "mcp__calc__add" {
				return sidecar.ToolResultError("unsupported tool"), nil
			}
			input := sidecar.StructToMap(req.ToolInput)
			if err := sidecar.ValidateJSONSchema(calcSchema.AsMap(), input); err != nil {
				return sidecar.ToolResultError(fmt.Sprintf("invalid input: %v", err)), nil
			}
			a, _ := input["a"].(float64)
			b, _ := input["b"].(float64)
			sum := a + b
			return sidecar.ToolResultWithMetadata([]sidecar.ContentBlock{
				sidecar.ContentBlockText(fmt.Sprintf("%v", sum)),
				sidecar.ContentBlockJSON(map[string]any{"sum": sum}),
			}, false, map[string]any{"unit": "number"}), nil
		},
		Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
			log.Printf("hook %s", req.HookEvent)
			if req.HookEvent == "PreToolUse" {
				return sidecar.HookContinue(), nil
			}
			return sidecar.HookContinue(), nil
		},
		Permission: sidecar.AskConfirmHandler(
			func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
				log.Printf("permission request %s attempt %d", req.ToolName, req.Attempt)
				if req.Attempt <= 1 {
					return sidecar.PermissionAsk("confirm tool usage"), nil
				}
				return sidecar.PermissionAllow("auto-approved"), nil
			},
			func(ctx context.Context, prompt string, req *pb.PermissionDecisionRequest) (bool, string, error) {
				if os.Getenv("SIDECAR_CONFIRM") != "" {
					return sidecar.ConsoleConfirm(os.Stdin, os.Stdout)(ctx, prompt, req)
				}
				return true, "auto-approved", nil
			},
		),
	}

	session, err := client.AttachSession(ctx, createResp.SidecarSessionId, sidecar.ClientInfo{
		Name:    "kitchen-sink",
		Version: "0.1.0",
	}, handlers)
	if err != nil {
		log.Fatalf("attach: %v", err)
	}

	if useInputStream {
		_, streamID, err := session.StartInputStream(ctx)
		if err != nil {
			log.Fatalf("start input stream: %v", err)
		}
		if err := session.SendInputEvent(ctx, streamID, sidecar.UserBlocksEvent([]sidecar.ContentBlock{
			sidecar.ContentBlockText("add 1 + 2 using the calculator"),
		})); err != nil {
			log.Fatalf("send input event: %v", err)
		}
		if err := session.EndInputStream(ctx, streamID); err != nil {
			log.Fatalf("end input stream: %v", err)
		}
	} else {
		_, err = session.Query(ctx, "add 1 + 2 using the calculator")
		if err != nil {
			log.Fatalf("query: %v", err)
		}
	}

	for event := range session.Events() {
		switch payload := event.Payload.(type) {
		case *pb.ServerEvent_SessionInit:
			initInfo := sidecar.ParseSessionInit(payload.SessionInit)
			fmt.Printf("init session_id=%s\n", payload.SessionInit.ClaudeSessionId)
			if initInfo != nil {
				if len(initInfo.Tools) > 0 {
					fmt.Printf("init tools=%v\n", initInfo.Tools)
				}
				if outputStyle := initInfo.OutputStyle(); outputStyle != "" {
					fmt.Printf("init output_style=%s\n", outputStyle)
				}
			}
		case *pb.ServerEvent_Message:
			parsed, err := sidecar.ParseMessageEvent(payload.Message)
			if err != nil {
				log.Printf("message parse error: %v", err)
				continue
			}
			switch message := parsed.Message.(type) {
			case sidecar.UserMessage:
				fmt.Printf("user checkpoint=%s\n", message.CheckpointUUID)
				for _, block := range message.Content {
					if text, ok := block.(sidecar.TextBlock); ok {
						fmt.Printf("user text=%s\n", text.Text)
					}
				}
			case sidecar.AssistantMessage:
				fmt.Printf("assistant model=%s\n", message.Model)
				for _, block := range message.Content {
					switch typed := block.(type) {
					case sidecar.TextBlock:
						fmt.Printf("assistant text=%s\n", typed.Text)
					case sidecar.ToolUseBlock:
						fmt.Printf("assistant tool=%s id=%s\n", typed.Name, typed.ID)
					case sidecar.ToolResultBlock:
						fmt.Printf("assistant tool_result=%s error=%v\n", typed.ToolUseID, typed.IsError)
					}
				}
			case sidecar.ResultMessage:
				fmt.Printf("result=%s\n", message.Result)
				if len(message.StructuredOutput) > 0 {
					fmt.Printf("structured=%v\n", message.StructuredOutput)
				}
			case sidecar.StreamEventMessage:
				fmt.Printf("stream event=%v\n", message.Event)
			case sidecar.SystemMessage:
				fmt.Printf("system subtype=%s\n", message.Subtype)
			}
		case *pb.ServerEvent_Error:
			fmt.Printf("error=%s\n", payload.Error.Message)
		case *pb.ServerEvent_SessionClosed:
			fmt.Printf("closed=%s\n", payload.SessionClosed.Reason)
		}
	}
}
