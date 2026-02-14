package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type e2eHarness struct {
	t      *testing.T
	client *Client
	logs   *bytes.Buffer
}

func newE2EHarness(t *testing.T) *e2eHarness {
	t.Helper()
	if os.Getenv("SIDECAR_E2E") == "" {
		t.Skip("set SIDECAR_E2E=1 to run sidecar E2E test")
	}

	if addr := os.Getenv("SIDECAR_ADDR"); addr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(cancel)
		if err := waitForHealth(ctx, addr); err != nil {
			t.Fatalf("sidecar not healthy: %v", err)
		}
		client, err := Dial(ctx, addr)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		t.Cleanup(func() {
			_ = client.Close()
		})
		return &e2eHarness{
			t:      t,
			client: client,
		}
	}

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	python := resolvePython(root)
	if !pythonHasGrpc(python, root) {
		t.Skipf("grpc not available for %s", python)
	}

	host := "127.0.0.1"
	port := pickPort(t, host)
	addr := fmt.Sprintf("%s:%d", host, port)

	testMode := shouldUseTestMode()
	cmd, buf := startSidecar(t, python, root, host, port, testMode)
	t.Cleanup(func() {
		stopProcess(t, cmd, buf)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	if err := waitForHealth(ctx, addr); err != nil {
		t.Fatalf("sidecar not healthy: %v\nlogs:\n%s", err, buf.String())
	}

	client, err := Dial(ctx, addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	return &e2eHarness{
		t:      t,
		client: client,
		logs:   buf,
	}
}

func (h *e2eHarness) Client() *Client {
	return h.client
}

func (h *e2eHarness) Logs() string {
	if h.logs == nil {
		return ""
	}
	return h.logs.String()
}

func (h *e2eHarness) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func (h *e2eHarness) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func (h *e2eHarness) CreateSession(
	ctx context.Context,
	mode pb.SessionMode,
) (*pb.CreateSessionResponse, error) {
	return h.client.CreateSession(ctx, &pb.CreateSessionRequest{
		Mode:    mode,
		Options: mergeTestOptions(nil),
	})
}

func (h *e2eHarness) CreateSessionWithOptions(
	ctx context.Context,
	mode pb.SessionMode,
	options *pb.ClaudeAgentOptions,
) (*pb.CreateSessionResponse, error) {
	return h.client.CreateSession(ctx, &pb.CreateSessionRequest{
		Mode:    mode,
		Options: mergeTestOptions(options),
	})
}

func (h *e2eHarness) AttachSession(
	ctx context.Context,
	sidecarSessionID string,
	handlers Handlers,
) (*Session, error) {
	return h.client.AttachSession(ctx, sidecarSessionID, ClientInfo{
		Name:    "e2e",
		Version: "test",
	}, handlers)
}

func e2eTestOptions(testMode bool) *pb.ClaudeAgentOptions {
	if !testMode {
		return nil
	}
	return &pb.ClaudeAgentOptions{
		ExtraArgs: map[string]*structpb.Value{
			"test_mode": structpb.NewBoolValue(true),
		},
	}
}

func mergeTestOptions(options *pb.ClaudeAgentOptions) *pb.ClaudeAgentOptions {
	if !shouldUseTestMode() {
		return options
	}
	if options == nil {
		options = &pb.ClaudeAgentOptions{}
	}
	if options.ExtraArgs == nil {
		options.ExtraArgs = map[string]*structpb.Value{}
	}
	options.ExtraArgs["test_mode"] = structpb.NewBoolValue(true)
	return options
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 8; i++ {
		if fileExists(filepath.Join(dir, "python", "sidecar", "claude_sidecar", "serve.py")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found")
}

func resolvePython(root string) string {
	if override := os.Getenv("SIDECAR_PYTHON"); override != "" {
		return override
	}
	venv := filepath.Join(root, ".venv", "bin", "python")
	if fileExists(venv) {
		return venv
	}
	return "python3"
}

func pythonHasGrpc(python, root string) bool {
	cmd := exec.Command(python, "-c", "import grpc")
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(root, "python", "sidecar"))
	return cmd.Run() == nil
}

func pickPort(t *testing.T, host string) int {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func startSidecar(
	t *testing.T,
	python, root, host string,
	port int,
	testMode bool,
) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	cmd := exec.Command(
		python,
		"-m",
		"claude_sidecar.serve",
		"--host",
		host,
		"--port",
		strconv.Itoa(port),
	)
	env := append(os.Environ(), "PYTHONPATH="+filepath.Join(root, "python", "sidecar"))
	if testMode {
		env = append(env, "CLAUDE_SIDECAR_TEST_MODE=1")
	}
	cmd.Env = env
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sidecar: %v", err)
	}
	return cmd, buf
}

func stopProcess(t *testing.T, cmd *exec.Cmd, buf *bytes.Buffer) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
	case <-done:
	}
}

func waitForHealth(ctx context.Context, addr string) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			client := pb.NewClaudeSidecarClient(conn)
			_, err = client.HealthCheck(ctx, &pb.HealthCheckRequest{})
			_ = conn.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func findSessionSummary(
	sessions []*pb.SessionSummary,
	sidecarSessionID string,
) *pb.SessionSummary {
	for _, session := range sessions {
		if session.GetSidecarSessionId() == sidecarSessionID {
			return session
		}
	}
	return nil
}

func shouldUseTestMode() bool {
	if value, ok := envBool("SIDECAR_E2E_TEST_MODE"); ok {
		return value
	}
	if value, ok := envBool("SIDECAR_E2E_LIVE"); ok {
		return !value
	}
	return true
}

func envBool(name string) (bool, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return false, false
	}
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off", "":
		return false, true
	default:
		return false, true
	}
}

func collectUntilSessionClosed(
	ctx context.Context,
	events <-chan *pb.ServerEvent,
) ([]*pb.ServerEvent, []*pb.SidecarError, error) {
	var collected []*pb.ServerEvent
	var errors []*pb.SidecarError
	for {
		select {
		case <-ctx.Done():
			return collected, errors, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return collected, errors, io.EOF
			}
			collected = append(collected, event)
			if errEvent := event.GetError(); errEvent != nil {
				errors = append(errors, errEvent)
			}
			if event.GetSessionClosed() != nil {
				return collected, errors, nil
			}
		}
	}
}

func assertNoSidecarErrors(t *testing.T, errors []*pb.SidecarError) {
	t.Helper()
	if len(errors) == 0 {
		return
	}
	lines := make([]string, 0, len(errors))
	for _, errEvent := range errors {
		if errEvent == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s (fatal=%t): %s", errEvent.GetCode(), errEvent.GetFatal(), errEvent.GetMessage()))
	}
	t.Fatalf("unexpected sidecar errors: %s", strings.Join(lines, "; "))
}

func assertSessionEventIDs(t *testing.T, events []*pb.ServerEvent, sidecarSessionID string) {
	t.Helper()
	for _, event := range events {
		if event.GetSidecarSessionId() != sidecarSessionID {
			t.Fatalf(
				"expected sidecar_session_id %q, got %q",
				sidecarSessionID,
				event.GetSidecarSessionId(),
			)
		}
	}
}

func assertTurnCorrelation(t *testing.T, turn *Turn, requestID string) {
	t.Helper()
	if turn == nil {
		t.Fatalf("expected turn to be set")
	}
	if turn.RequestID == "" {
		t.Fatalf("expected request_id to be set")
	}
	if requestID != "" && turn.RequestID != requestID {
		t.Fatalf("expected request_id %q, got %q", requestID, turn.RequestID)
	}
	if turn.TurnID == "" {
		t.Fatalf("expected turn_id to be set")
	}
	for _, event := range turn.Events {
		if event.GetRequestId() == "" {
			t.Fatalf("missing request_id on server event")
		}
		if event.GetRequestId() != turn.RequestID {
			t.Fatalf("expected request_id %q, got %q", turn.RequestID, event.GetRequestId())
		}
		if event.GetTurnId() == "" {
			t.Fatalf("missing turn_id on server event")
		}
		if event.GetTurnId() != turn.TurnID {
			t.Fatalf("expected turn_id %q, got %q", turn.TurnID, event.GetTurnId())
		}
	}
}

func sawTurnEndBeforeSessionClosed(events []*pb.ServerEvent) bool {
	seenTurnEnd := false
	for _, event := range events {
		if turn := event.GetTurn(); turn != nil && turn.Kind == pb.TurnBoundary_TURN_END {
			seenTurnEnd = true
		}
		if event.GetSessionClosed() != nil {
			return seenTurnEnd
		}
	}
	return false
}

func waitForSessionInit(
	ctx context.Context,
	events <-chan *pb.ServerEvent,
) (*pb.SessionInit, []*pb.SidecarError, error) {
	var errors []*pb.SidecarError
	for {
		select {
		case <-ctx.Done():
			return nil, errors, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil, errors, io.EOF
			}
			if errEvent := event.GetError(); errEvent != nil {
				errors = append(errors, errEvent)
			}
			if init := event.GetSessionInit(); init != nil {
				return init, errors, nil
			}
		}
	}
}

func waitForSidecarError(
	ctx context.Context,
	events <-chan *pb.ServerEvent,
) (*pb.ServerEvent, *pb.SidecarError, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil, nil, io.EOF
			}
			if errEvent := event.GetError(); errEvent != nil {
				return event, errEvent, nil
			}
		}
	}
}

func waitForPartial(
	ctx context.Context,
	partials <-chan *pb.MessageEvent,
) (*pb.MessageEvent, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case partial, ok := <-partials:
			if !ok {
				return nil, io.EOF
			}
			if partial != nil {
				return partial, nil
			}
		}
	}
}

func waitForTurn(
	ctx context.Context,
	session *Session,
	requestID string,
) (*Turn, error) {
	sub := session.mux.SubscribeRequest(requestID, 256)
	defer session.mux.UnsubscribeRequest(requestID, sub)
	turns := CollectTurns(ctx, sub.Chan())
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case turn, ok := <-turns:
			if !ok {
				return nil, io.EOF
			}
			if turn != nil {
				return turn, nil
			}
		}
	}
}

func expectNoPartial(
	ctx context.Context,
	partials <-chan *pb.MessageEvent,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case partial, ok := <-partials:
			if !ok {
				return nil
			}
			if partial != nil {
				return fmt.Errorf("unexpected partial message received")
			}
		}
	}
}

func collectUntilTurnEnd(
	ctx context.Context,
	events <-chan *pb.ServerEvent,
	requestID string,
) ([]*pb.ServerEvent, []*pb.SidecarError, error) {
	var collected []*pb.ServerEvent
	var errors []*pb.SidecarError
	for {
		select {
		case <-ctx.Done():
			return collected, errors, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return collected, errors, io.EOF
			}
			collected = append(collected, event)
			if errEvent := event.GetError(); errEvent != nil {
				errors = append(errors, errEvent)
			}
			if event.GetRequestId() != requestID {
				continue
			}
			if turn := event.GetTurn(); turn != nil && turn.Kind == pb.TurnBoundary_TURN_END {
				return collected, errors, nil
			}
		}
	}
}

func findMessageEventIndex(
	events []*pb.ServerEvent,
	predicate func(*pb.MessageEvent) bool,
) int {
	for i, event := range events {
		message := event.GetMessage()
		if message == nil {
			continue
		}
		if predicate(message) {
			return i
		}
	}
	return -1
}

func turnUserTexts(turn *Turn) []string {
	if turn == nil {
		return nil
	}
	userTexts := []string{}
	for _, message := range turn.Messages {
		user := message.GetUser()
		if user == nil || len(user.Content) == 0 {
			continue
		}
		text := user.Content[0].GetText()
		if text != nil {
			userTexts = append(userTexts, text.Text)
		}
	}
	return userTexts
}
