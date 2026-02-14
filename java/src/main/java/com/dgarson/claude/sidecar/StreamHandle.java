package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar.MessageEvent;
import claude_sidecar.v1.Sidecar.ServerEvent;
import claude_sidecar.v1.Sidecar.TurnBoundary;

import java.io.EOFException;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.LinkedBlockingQueue;

/**
 * Provides async access to a streaming query result. Mirrors Go's run.go Stream struct.
 *
 * <p>Consumers can read partial events from {@link #partials()}, raw server events from
 * {@link #events()}, or block on the final {@link RunResult} via {@link #result()}.</p>
 */
public final class StreamHandle {

    private final String requestId;
    private final EventMux.Subscription subscription;
    private final EventMux mux;

    private final BlockingQueue<ServerEvent> events;
    private final BlockingQueue<MessageEvent> partials = new LinkedBlockingQueue<>();
    private final CompletableFuture<RunResult> doneFuture = new CompletableFuture<>();

    StreamHandle(String requestId, EventMux.Subscription subscription, EventMux mux) {
        this.requestId = requestId;
        this.subscription = subscription;
        this.mux = mux;
        this.events = subscription.queue();

        Thread.ofVirtual().name("stream-" + requestId).start(this::run);
    }

    public String getRequestId() {
        return requestId;
    }

    /**
     * Returns the queue of raw server events for this request.
     * Events are available as they arrive from the sidecar.
     */
    public BlockingQueue<ServerEvent> events() {
        return events;
    }

    /**
     * Returns the queue of partial (streaming) assistant message events.
     */
    public BlockingQueue<MessageEvent> partials() {
        return partials;
    }

    /**
     * Returns a future that completes with the final {@link RunResult} when the turn ends.
     */
    public CompletableFuture<RunResult> resultFuture() {
        return doneFuture;
    }

    /**
     * Blocks until the turn completes and returns the result.
     *
     * @throws Exception if the stream encounters an error or is interrupted
     */
    public RunResult result() throws Exception {
        return doneFuture.get();
    }

    /** Cancel this stream. */
    public void close() {
        mux.unsubscribeRequest(requestId, subscription);
    }

    // -- Internal dispatch loop --

    private void run() {
        Turn turn = null;
        try {
            while (true) {
                ServerEvent event = events.poll(30, java.util.concurrent.TimeUnit.SECONDS);
                if (event == null) {
                    // Check if subscription is closed (no more events coming)
                    if (subscription.isClosed() && events.isEmpty()) {
                        break;
                    }
                    continue;
                }

                // Initialize the turn on first event with a turn ID
                if (turn == null) {
                    String turnId = event.getTurnId();
                    if (turnId != null && !turnId.isEmpty()) {
                        turn = new Turn(turnId);
                    } else {
                        continue;
                    }
                }

                if (turn.getRequestId().isEmpty() && event.getRequestId() != null
                        && !event.getRequestId().isEmpty()) {
                    turn.setRequestId(event.getRequestId());
                }
                turn.addEvent(event);

                if (event.hasTurn()) {
                    var boundary = event.getTurn();
                    if (boundary.getKind() == TurnBoundary.Kind.TURN_BEGIN) {
                        turn.setStarted(true);
                        turn.setTurnIndex(boundary.getTurnIndex());
                    } else if (boundary.getKind() == TurnBoundary.Kind.TURN_END) {
                        turn.setEnded(true);
                        if (turn.getTurnIndex() == 0) {
                            turn.setTurnIndex(boundary.getTurnIndex());
                        }
                        doneFuture.complete(new RunResult(turn));
                        return;
                    }
                }
                if (event.hasMessage()) {
                    MessageEvent msg = event.getMessage();
                    if (msg.getIsPartial()) {
                        partials.offer(msg);
                    }
                    turn.addMessage(msg);
                }
                if (event.hasStderrLine()) {
                    turn.addStderr(event.getStderrLine().getLine());
                }
                if (event.hasError()) {
                    turn.addError(event.getError());
                }
            }

            // Stream ended without TURN_END
            if (turn != null) {
                doneFuture.complete(new RunResult(turn));
            } else {
                doneFuture.completeExceptionally(new EOFException("stream ended with no turn"));
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            doneFuture.completeExceptionally(e);
        } catch (Exception e) {
            doneFuture.completeExceptionally(e);
        } finally {
            mux.unsubscribeRequest(requestId, subscription);
        }
    }
}
