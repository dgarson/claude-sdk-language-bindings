package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

type ConfirmHandler func(ctx context.Context, prompt string, req *pb.PermissionDecisionRequest) (bool, string, error)

func PermissionAllow(reason string) *pb.PermissionDecision {
	return &pb.PermissionDecision{Behavior: "allow", Reason: reason}
}

func PermissionDeny(reason string) *pb.PermissionDecision {
	return &pb.PermissionDecision{Behavior: "deny", Reason: reason}
}

func PermissionAsk(reason string) *pb.PermissionDecision {
	return &pb.PermissionDecision{Behavior: "ask", Reason: reason}
}

func PermissionWithUpdatedInput(
	decision *pb.PermissionDecision, updated map[string]any,
) (*pb.PermissionDecision, error) {
	if decision == nil {
		return nil, fmt.Errorf("decision is nil")
	}
	if updated == nil {
		return decision, nil
	}
	updatedStruct, err := MapToStruct(updated)
	if err != nil {
		return nil, err
	}
	decision.UpdatedInput = updatedStruct
	return decision, nil
}

func DefaultPermissionPrompt(req *pb.PermissionDecisionRequest) string {
	if req == nil {
		return "Permission required to proceed."
	}
	if req.Prompt != "" {
		return req.Prompt
	}
	if req.ToolName != "" {
		return fmt.Sprintf("Allow tool %s to run? (y/n): ", req.ToolName)
	}
	return "Allow the requested tool to run? (y/n): "
}

func AskConfirmHandler(
	decide PermissionHandler,
	confirm ConfirmHandler,
) PermissionHandler {
	return AskConfirmHandlerWithPrompt(decide, confirm, DefaultPermissionPrompt)
}

func AskConfirmHandlerWithPrompt(
	decide PermissionHandler,
	confirm ConfirmHandler,
	prompt func(*pb.PermissionDecisionRequest) string,
) PermissionHandler {
	return func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
		if req.Attempt <= 1 {
			if decide != nil {
				return decide(ctx, req)
			}
			if confirm != nil {
				return PermissionAsk("confirmation required"), nil
			}
			return PermissionDeny("no permission handler"), nil
		}
		if confirm != nil {
			promptText := DefaultPermissionPrompt(req)
			if prompt != nil {
				promptText = prompt(req)
			}
			ok, reason, err := confirm(ctx, promptText, req)
			if err != nil {
				return PermissionDeny(err.Error()), nil
			}
			if ok {
				return PermissionAllow(reason), nil
			}
			return PermissionDeny(reason), nil
		}
		if decide != nil {
			return decide(ctx, req)
		}
		return PermissionDeny("no confirmation handler"), nil
	}
}

func ConsoleConfirm(in io.Reader, out io.Writer) ConfirmHandler {
	return func(ctx context.Context, prompt string, req *pb.PermissionDecisionRequest) (bool, string, error) {
		reader := bufio.NewReader(in)
		for {
			_, _ = fmt.Fprint(out, prompt)
			line, err := reader.ReadString('\n')
			if err != nil {
				return false, "confirmation read failed", err
			}
			switch strings.ToLower(strings.TrimSpace(line)) {
			case "y", "yes":
				return true, "confirmed", nil
			case "n", "no":
				return false, "denied", nil
			default:
				if ctx.Err() != nil {
					return false, "confirmation canceled", ctx.Err()
				}
				prompt = "Please reply y/n: "
			}
		}
	}
}
