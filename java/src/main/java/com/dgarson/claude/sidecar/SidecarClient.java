package com.dgarson.claude.sidecar;

import claude_sidecar.v1.ClaudeSidecarGrpc;
import claude_sidecar.v1.Sidecar.ClientEvent;
import claude_sidecar.v1.Sidecar.ClientHello;
import claude_sidecar.v1.Sidecar.CreateSessionRequest;
import claude_sidecar.v1.Sidecar.CreateSessionResponse;
import claude_sidecar.v1.Sidecar.DeleteSessionRequest;
import claude_sidecar.v1.Sidecar.DeleteSessionResponse;
import claude_sidecar.v1.Sidecar.ForkSessionRequest;
import claude_sidecar.v1.Sidecar.ForkSessionResponse;
import claude_sidecar.v1.Sidecar.GetInfoRequest;
import claude_sidecar.v1.Sidecar.GetInfoResponse;
import claude_sidecar.v1.Sidecar.GetSessionRequest;
import claude_sidecar.v1.Sidecar.GetSessionResponse;
import claude_sidecar.v1.Sidecar.HealthCheckRequest;
import claude_sidecar.v1.Sidecar.HealthCheckResponse;
import claude_sidecar.v1.Sidecar.ListSessionsRequest;
import claude_sidecar.v1.Sidecar.ListSessionsResponse;
import claude_sidecar.v1.Sidecar.RewindFilesRequest;
import claude_sidecar.v1.Sidecar.RewindFilesResponse;
import claude_sidecar.v1.Sidecar.ServerEvent;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.stub.StreamObserver;

import java.util.concurrent.TimeUnit;

/**
 * High-level client for the Claude Sidecar gRPC service.
 * Mirrors Go's client.go.
 *
 * <p>Provides control-plane operations (create/get/list/delete/fork sessions, health checks)
 * and data-plane attachment via {@link #attachSession}.</p>
 *
 * <p>Usage:</p>
 * <pre>{@code
 * try (SidecarClient client = SidecarClient.connect("localhost:50051")) {
 *     var resp = client.createSession(CreateSessionRequest.newBuilder()...build());
 *     var session = client.attachSession(resp.getSidecarSessionId(),
 *         ClientInfo.builder().name("my-app").version("1.0").build(),
 *         Handlers.builder().tool(req -> ToolResults.text("done")).build());
 *     RunResult result = session.run("Hello!");
 *     session.close();
 * }
 * }</pre>
 */
public final class SidecarClient implements AutoCloseable {

    private final ManagedChannel channel;
    private final ClaudeSidecarGrpc.ClaudeSidecarBlockingStub blockingStub;
    private final ClaudeSidecarGrpc.ClaudeSidecarStub asyncStub;

    private SidecarClient(ManagedChannel channel) {
        this.channel = channel;
        this.blockingStub = ClaudeSidecarGrpc.newBlockingStub(channel);
        this.asyncStub = ClaudeSidecarGrpc.newStub(channel);
    }

    /**
     * Connect to a sidecar instance at the given address (e.g. "localhost:50051").
     * Uses plaintext (insecure) transport by default.
     */
    public static SidecarClient connect(String address) {
        ManagedChannel channel = ManagedChannelBuilder.forTarget(address)
                .usePlaintext()
                .build();
        return new SidecarClient(channel);
    }

    /**
     * Connect using a pre-built {@link ManagedChannel} (e.g. for TLS or custom configuration).
     */
    public static SidecarClient fromChannel(ManagedChannel channel) {
        return new SidecarClient(channel);
    }

    // -- Control plane --

    public GetInfoResponse getInfo() {
        return blockingStub.getInfo(GetInfoRequest.getDefaultInstance());
    }

    public HealthCheckResponse healthCheck() {
        return blockingStub.healthCheck(HealthCheckRequest.getDefaultInstance());
    }

    public CreateSessionResponse createSession(CreateSessionRequest request) {
        return blockingStub.createSession(request);
    }

    public GetSessionResponse getSession(String sidecarSessionId) {
        return blockingStub.getSession(GetSessionRequest.newBuilder()
                .setSidecarSessionId(sidecarSessionId)
                .build());
    }

    public ListSessionsResponse listSessions() {
        return blockingStub.listSessions(ListSessionsRequest.getDefaultInstance());
    }

    public DeleteSessionResponse deleteSession(String sidecarSessionId, boolean force) {
        return blockingStub.deleteSession(DeleteSessionRequest.newBuilder()
                .setSidecarSessionId(sidecarSessionId)
                .setForce(force)
                .build());
    }

    public ForkSessionResponse forkSession(ForkSessionRequest request) {
        return blockingStub.forkSession(request);
    }

    public RewindFilesResponse rewindFiles(RewindFilesRequest request) {
        return blockingStub.rewindFiles(request);
    }

    // -- Data plane --

    /**
     * Attach to an existing session, establishing a bidirectional stream for queries and events.
     *
     * <p>Sends a {@link ClientHello} immediately upon attachment and starts the background
     * receive loop for dispatching events and callbacks.</p>
     *
     * @param sidecarSessionId the session ID returned by {@link #createSession}
     * @param clientInfo       identification for this client
     * @param handlers         callback handlers for tool/hook/permission requests
     * @return a {@link Session} for sending queries and receiving events
     */
    public Session attachSession(String sidecarSessionId, ClientInfo clientInfo, Handlers handlers) {
        // Use a bridge to resolve the circular dependency: the async stub needs a
        // StreamObserver<ServerEvent> to call attachSession(), but the Session that will
        // handle those events needs the StreamObserver<ClientEvent> returned by the stub.
        var bridge = new BidiStreamBridge();
        StreamObserver<ClientEvent> requestObserver = asyncStub.attachSession(bridge);

        // Now create the session with the real requestObserver and wire the bridge to it
        Session session = new Session(sidecarSessionId, requestObserver, handlers);
        bridge.setSession(session);

        // Send ClientHello
        String protocol = clientInfo.protocol();
        if (protocol == null || protocol.isEmpty()) {
            protocol = "v1";
        }
        session.send(ClientEvent.newBuilder()
                .setSidecarSessionId(sidecarSessionId)
                .setHello(ClientHello.newBuilder()
                        .setProtocolVersion(protocol)
                        .setClientName(clientInfo.name() != null ? clientInfo.name() : "")
                        .setClientVersion(clientInfo.version() != null ? clientInfo.version() : "")
                        .build())
                .build());

        return session;
    }

    @Override
    public void close() {
        try {
            channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
        } catch (InterruptedException e) {
            channel.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Returns the underlying channel for advanced use cases.
     */
    public ManagedChannel getChannel() {
        return channel;
    }

    // -------------------------------------------------------------------------
    // Bridge to resolve the circular dependency between Session and StreamObserver
    // -------------------------------------------------------------------------

    /**
     * Bridges the gRPC StreamObserver callback to the Session's internal response observer.
     * This resolves the circular dependency: the stub needs a response observer before we
     * can create the Session (which needs the request observer the stub returns).
     */
    private static final class BidiStreamBridge implements StreamObserver<ServerEvent> {
        private volatile StreamObserver<ServerEvent> delegate;

        BidiStreamBridge() {}

        void setSession(Session session) {
            this.delegate = session.createResponseObserver();
        }

        @Override
        public void onNext(ServerEvent event) {
            StreamObserver<ServerEvent> d = delegate;
            if (d != null) {
                d.onNext(event);
            }
        }

        @Override
        public void onError(Throwable t) {
            StreamObserver<ServerEvent> d = delegate;
            if (d != null) {
                d.onError(t);
            }
        }

        @Override
        public void onCompleted() {
            StreamObserver<ServerEvent> d = delegate;
            if (d != null) {
                d.onCompleted();
            }
        }
    }
}
