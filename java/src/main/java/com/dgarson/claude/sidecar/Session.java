package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar.CancelRequest;
import claude_sidecar.v1.Sidecar.ClientEvent;
import claude_sidecar.v1.Sidecar.EndInputStream;
import claude_sidecar.v1.Sidecar.HookInvocationRequest;
import claude_sidecar.v1.Sidecar.HookInvocationResponse;
import claude_sidecar.v1.Sidecar.HookOutput;
import claude_sidecar.v1.Sidecar.InterruptRequest;
import claude_sidecar.v1.Sidecar.PermissionDecision;
import claude_sidecar.v1.Sidecar.PermissionDecisionRequest;
import claude_sidecar.v1.Sidecar.PermissionDecisionResponse;
import claude_sidecar.v1.Sidecar.QueryRequest;
import claude_sidecar.v1.Sidecar.ServerEvent;
import claude_sidecar.v1.Sidecar.SetModelRequest;
import claude_sidecar.v1.Sidecar.SetPermissionModeRequest;
import claude_sidecar.v1.Sidecar.StreamInputChunk;
import claude_sidecar.v1.Sidecar.ToolInvocationRequest;
import claude_sidecar.v1.Sidecar.ToolInvocationResponse;
import com.google.protobuf.Struct;
import io.grpc.stub.StreamObserver;

import java.security.SecureRandom;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Manages a bidirectional stream to an attached sidecar session.
 * Mirrors Go's session.go.
 *
 * <p>The session dispatches tool, hook, and permission callback requests to the provided
 * {@link Handlers}, and multiplexes all other events through an {@link EventMux}.</p>
 */
public final class Session implements AutoCloseable {

    private static final Logger LOG = Logger.getLogger(Session.class.getName());
    private static final SecureRandom RANDOM = new SecureRandom();

    private final String sessionId;
    private final StreamObserver<ClientEvent> requestObserver;
    private final Handlers handlers;
    private final EventMux mux;
    private final EventMux.Subscription globalSubscription;
    private final AtomicBoolean closed = new AtomicBoolean(false);
    private final Object sendLock = new Object();

    Session(String sessionId,
            StreamObserver<ClientEvent> requestObserver,
            Handlers handlers) {
        this.sessionId = sessionId;
        this.requestObserver = requestObserver;
        this.handlers = handlers;
        this.mux = new EventMux();
        this.globalSubscription = mux.subscribeAll(256);
    }

    // -- Public API --

    /** Returns the sidecar session ID. */
    public String getSessionId() {
        return sessionId;
    }

    /**
     * Returns a blocking queue that receives all server events for this session.
     * This is the Java equivalent of Go's {@code Events() <-chan *pb.ServerEvent}.
     */
    public BlockingQueue<ServerEvent> events() {
        return globalSubscription.queue();
    }

    /**
     * Send a query and return the request ID. The caller should subscribe to events
     * (or use {@link #run} / {@link #stream}) to receive the response.
     */
    public String query(String prompt) {
        String requestId = newId("req");
        send(ClientEvent.newBuilder()
                .setRequestId(requestId)
                .setSidecarSessionId(sessionId)
                .setQuery(QueryRequest.newBuilder()
                        .setPromptText(prompt)
                        .build())
                .build());
        return requestId;
    }

    /**
     * Send a query and block until the turn completes, returning the full result.
     */
    public RunResult run(String prompt) throws Exception {
        StreamHandle handle = stream(prompt);
        return handle.result();
    }

    /**
     * Send a query and return a {@link StreamHandle} for async consumption of events.
     */
    public StreamHandle stream(String prompt) {
        String requestId = query(prompt);
        EventMux.Subscription sub = mux.subscribeRequest(requestId, 256);
        return new StreamHandle(requestId, sub, mux);
    }

    /**
     * Start an input stream for streaming input events. Returns [requestId, streamId].
     */
    public String[] startInputStream() {
        String streamId = newId("input");
        String requestId = newId("req");
        send(ClientEvent.newBuilder()
                .setRequestId(requestId)
                .setSidecarSessionId(sessionId)
                .setQuery(QueryRequest.newBuilder()
                        .setInputStreamId(streamId)
                        .build())
                .build());
        return new String[]{requestId, streamId};
    }

    /** Send a raw input chunk on the given stream. */
    public void sendInputChunk(String streamId, Struct event) {
        send(ClientEvent.newBuilder()
                .setSidecarSessionId(sessionId)
                .setInputChunk(StreamInputChunk.newBuilder()
                        .setInputStreamId(streamId)
                        .setEvent(event)
                        .build())
                .build());
    }

    /** Signal end of input for the given stream. */
    public void endInputStream(String streamId) {
        send(ClientEvent.newBuilder()
                .setSidecarSessionId(sessionId)
                .setEndInput(EndInputStream.newBuilder()
                        .setInputStreamId(streamId)
                        .build())
                .build());
    }

    /** Interrupt the current turn. */
    public void interrupt() {
        send(ClientEvent.newBuilder()
                .setRequestId(newId("req"))
                .setSidecarSessionId(sessionId)
                .setInterrupt(InterruptRequest.getDefaultInstance())
                .build());
    }

    /** Cancel the current turn with a reason. */
    public void cancel(String reason) {
        send(ClientEvent.newBuilder()
                .setRequestId(newId("req"))
                .setSidecarSessionId(sessionId)
                .setCancel(CancelRequest.newBuilder().setReason(reason).build())
                .build());
    }

    /** Set the permission mode for this session. */
    public void setPermissionMode(String mode) {
        send(ClientEvent.newBuilder()
                .setRequestId(newId("req"))
                .setSidecarSessionId(sessionId)
                .setSetPermissionMode(SetPermissionModeRequest.newBuilder()
                        .setMode(mode)
                        .build())
                .build());
    }

    /** Set the model for this session. */
    public void setModel(String model) {
        send(ClientEvent.newBuilder()
                .setRequestId(newId("req"))
                .setSidecarSessionId(sessionId)
                .setSetModel(SetModelRequest.newBuilder()
                        .setModel(model)
                        .build())
                .build());
    }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            try {
                requestObserver.onCompleted();
            } catch (Exception e) {
                LOG.log(Level.FINE, "Error closing send stream", e);
            }
            mux.close();
        }
    }

    // -- Internal: send with mutex --

    void send(ClientEvent event) {
        synchronized (sendLock) {
            requestObserver.onNext(event);
        }
    }

    // -- Internal: receive loop (called from SidecarClient after stream setup) --

    StreamObserver<ServerEvent> createResponseObserver() {
        return new StreamObserver<>() {
            @Override
            public void onNext(ServerEvent event) {
                handleCallback(event);
                mux.enqueue(event);
            }

            @Override
            public void onError(Throwable t) {
                LOG.log(Level.WARNING, "Stream error for session " + sessionId, t);
                mux.close();
            }

            @Override
            public void onCompleted() {
                mux.close();
            }
        };
    }

    // -- Internal: callback dispatch --

    private void handleCallback(ServerEvent event) {
        if (event.hasToolRequest()) {
            handleToolRequest(event.getToolRequest());
        } else if (event.hasHookRequest()) {
            handleHookRequest(event.getHookRequest());
        } else if (event.hasPermissionRequest()) {
            handlePermissionRequest(event.getPermissionRequest());
        }
    }

    private void handleToolRequest(ToolInvocationRequest request) {
        Thread.ofVirtual().name("tool-handler-" + request.getInvocationId()).start(() -> {
            Struct result;
            if (handlers.toolHandler() == null) {
                result = ToolResults.error("missing tool handler");
            } else {
                try {
                    result = handlers.toolHandler().handle(request);
                } catch (Exception e) {
                    result = ToolResults.error(e.getMessage());
                }
            }
            send(ClientEvent.newBuilder()
                    .setSidecarSessionId(sessionId)
                    .setToolResponse(ToolInvocationResponse.newBuilder()
                            .setInvocationId(request.getInvocationId())
                            .setToolResult(result)
                            .build())
                    .build());
        });
    }

    private void handleHookRequest(HookInvocationRequest request) {
        Thread.ofVirtual().name("hook-handler-" + request.getInvocationId()).start(() -> {
            HookOutput output;
            if (handlers.hookHandler() == null) {
                output = HookOutput.newBuilder()
                        .setContinue(false)
                        .setStopReason("no hook handler")
                        .build();
            } else {
                try {
                    output = handlers.hookHandler().handle(request);
                } catch (Exception e) {
                    output = HookOutput.newBuilder()
                            .setContinue(false)
                            .setStopReason(e.getMessage())
                            .build();
                }
            }
            send(ClientEvent.newBuilder()
                    .setSidecarSessionId(sessionId)
                    .setHookResponse(HookInvocationResponse.newBuilder()
                            .setInvocationId(request.getInvocationId())
                            .setOutput(output)
                            .build())
                    .build());
        });
    }

    private void handlePermissionRequest(PermissionDecisionRequest request) {
        Thread.ofVirtual().name("perm-handler-" + request.getInvocationId()).start(() -> {
            PermissionDecision decision;
            if (handlers.permissionHandler() == null) {
                decision = PermissionDecision.newBuilder()
                        .setBehavior("deny")
                        .setReason("no permission handler")
                        .build();
            } else {
                try {
                    decision = handlers.permissionHandler().handle(request);
                } catch (Exception e) {
                    decision = PermissionDecision.newBuilder()
                            .setBehavior("deny")
                            .setReason(e.getMessage())
                            .build();
                }
            }
            send(ClientEvent.newBuilder()
                    .setSidecarSessionId(sessionId)
                    .setPermissionResponse(PermissionDecisionResponse.newBuilder()
                            .setInvocationId(request.getInvocationId())
                            .setDecision(decision)
                            .build())
                    .build());
        });
    }

    // -- ID generation --

    static String newId(String prefix) {
        byte[] buf = new byte[8];
        RANDOM.nextBytes(buf);
        StringBuilder sb = new StringBuilder(prefix.length() + 1 + 16);
        sb.append(prefix).append('_');
        for (byte b : buf) {
            sb.append(String.format("%02x", b & 0xff));
        }
        return sb.toString();
    }
}
