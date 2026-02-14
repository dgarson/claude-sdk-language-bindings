package client

import (
	"context"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	api  pb.ClaudeSidecarClient
}

func Dial(ctx context.Context, addr string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, api: pb.NewClaudeSidecarClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetInfo(ctx context.Context) (*pb.GetInfoResponse, error) {
	return c.api.GetInfo(ctx, &pb.GetInfoRequest{})
}

func (c *Client) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
	return c.api.HealthCheck(ctx, &pb.HealthCheckRequest{})
}

func (c *Client) CreateSession(
	ctx context.Context, request *pb.CreateSessionRequest,
) (*pb.CreateSessionResponse, error) {
	return c.api.CreateSession(ctx, request)
}

func (c *Client) GetSession(ctx context.Context, sidecarSessionID string) (*pb.GetSessionResponse, error) {
	return c.api.GetSession(ctx, &pb.GetSessionRequest{SidecarSessionId: sidecarSessionID})
}

func (c *Client) ListSessions(ctx context.Context) (*pb.ListSessionsResponse, error) {
	return c.api.ListSessions(ctx, &pb.ListSessionsRequest{})
}

func (c *Client) DeleteSession(
	ctx context.Context, sidecarSessionID string, force bool,
) (*pb.DeleteSessionResponse, error) {
	return c.api.DeleteSession(ctx, &pb.DeleteSessionRequest{
		SidecarSessionId: sidecarSessionID,
		Force:            force,
	})
}

func (c *Client) ForkSession(
	ctx context.Context, request *pb.ForkSessionRequest,
) (*pb.ForkSessionResponse, error) {
	return c.api.ForkSession(ctx, request)
}

func (c *Client) RewindFiles(
	ctx context.Context, sidecarSessionID, checkpointUUID string,
) (*pb.RewindFilesResponse, error) {
	return c.api.RewindFiles(ctx, &pb.RewindFilesRequest{
		SidecarSessionId: sidecarSessionID,
		CheckpointUuid:   checkpointUUID,
	})
}

func (c *Client) AttachSession(
	ctx context.Context, sidecarSessionID string, info ClientInfo, handlers Handlers,
) (*Session, error) {
	stream, err := c.api.AttachSession(ctx)
	if err != nil {
		return nil, err
	}
	session := newSession(ctx, sidecarSessionID, stream, handlers)
	if info.Protocol == "" {
		info.Protocol = "v1"
	}
	err = session.send(&pb.ClientEvent{
		Payload: &pb.ClientEvent_Hello{
			Hello: &pb.ClientHello{
				ProtocolVersion: info.Protocol,
				ClientName:      info.Name,
				ClientVersion:   info.Version,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	session.startRecvLoop()
	return session, nil
}
