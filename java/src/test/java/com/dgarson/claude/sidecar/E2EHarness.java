package com.dgarson.claude.sidecar;

import claude_sidecar.v1.ClaudeSidecarGrpc;
import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Value;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import org.junit.jupiter.api.Assumptions;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.TimeUnit;

/**
 * Reusable E2E test harness for the Claude Sidecar Java client.
 *
 * <p>Connects to a sidecar at the address specified by the {@code sidecar.addr} system property
 * (defaults to {@code 127.0.0.1:50051}). Tests are skipped unless {@code sidecar.e2e} is set.
 *
 * <p>Mirrors the Go {@code e2e_harness_test.go} harness.
 */
public final class E2EHarness implements AutoCloseable {

    private static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(10);

    private final SidecarClient client;
    private final ManagedChannel channel;
    private final boolean testMode;

    private E2EHarness(SidecarClient client, ManagedChannel channel, boolean testMode) {
        this.client = client;
        this.channel = channel;
        this.testMode = testMode;
    }

    // ------------------------------------------------------------------
    // Factory
    // ------------------------------------------------------------------

    /**
     * Creates a new harness. Assumes the sidecar is already running.
     * Skips the calling test if {@code sidecar.e2e} is not set.
     */
    public static E2EHarness create() {
        String e2e = System.getProperty("sidecar.e2e", "");
        Assumptions.assumeTrue(!e2e.isEmpty(), "set SIDECAR_E2E=1 to run sidecar E2E tests");

        String addr = System.getProperty("sidecar.addr", "127.0.0.1:50051");
        boolean testMode = shouldUseTestMode();

        ManagedChannel channel = ManagedChannelBuilder.forTarget(addr)
                .usePlaintext()
                .build();

        // Health check
        ClaudeSidecarGrpc.ClaudeSidecarBlockingStub stub =
                ClaudeSidecarGrpc.newBlockingStub(channel);
        Sidecar.HealthCheckResponse health = stub.healthCheck(
                Sidecar.HealthCheckRequest.getDefaultInstance());
        if (health == null || health.getStatus().isEmpty()) {
            throw new RuntimeException("Sidecar health check failed");
        }

        SidecarClient client = SidecarClient.connect(addr);
        return new E2EHarness(client, channel, testMode);
    }

    // ------------------------------------------------------------------
    // Accessors
    // ------------------------------------------------------------------

    public SidecarClient client() {
        return client;
    }

    public boolean isTestMode() {
        return testMode;
    }

    public Duration defaultTimeout() {
        return DEFAULT_TIMEOUT;
    }

    // ------------------------------------------------------------------
    // Session helpers
    // ------------------------------------------------------------------

    public Sidecar.CreateSessionResponse createSession(Sidecar.SessionMode mode) {
        return client.createSession(Sidecar.CreateSessionRequest.newBuilder()
                .setMode(mode)
                .setOptions(mergeTestOptions(null))
                .build());
    }

    public Sidecar.CreateSessionResponse createSessionWithOptions(
            Sidecar.SessionMode mode,
            Sidecar.ClaudeAgentOptions options) {
        return client.createSession(Sidecar.CreateSessionRequest.newBuilder()
                .setMode(mode)
                .setOptions(mergeTestOptions(options))
                .build());
    }

    public Session attachSession(String sidecarSessionId, Handlers handlers) {
        return client.attachSession(sidecarSessionId,
                ClientInfo.builder().name("e2e").version("test").build(), handlers);
    }

    // ------------------------------------------------------------------
    // Test mode / option merging
    // ------------------------------------------------------------------

    public static boolean shouldUseTestMode() {
        String testModeVal = System.getProperty("sidecar.e2e.test_mode", "");
        if (!testModeVal.isEmpty()) {
            return parseBool(testModeVal);
        }
        String liveVal = System.getProperty("sidecar.e2e.live", "");
        if (!liveVal.isEmpty()) {
            return !parseBool(liveVal);
        }
        return true;
    }

    public static Sidecar.ClaudeAgentOptions mergeTestOptions(Sidecar.ClaudeAgentOptions options) {
        if (!shouldUseTestMode()) {
            return options != null ? options : Sidecar.ClaudeAgentOptions.getDefaultInstance();
        }
        Sidecar.ClaudeAgentOptions.Builder builder;
        if (options != null) {
            builder = options.toBuilder();
        } else {
            builder = Sidecar.ClaudeAgentOptions.newBuilder();
        }
        builder.putExtraArgs("test_mode", Value.newBuilder().setBoolValue(true).build());
        return builder.build();
    }

    // ------------------------------------------------------------------
    // Event collection helpers
    // ------------------------------------------------------------------

    /**
     * Collects events from the session event channel until a SessionClosed event is received
     * or the timeout expires.
     */
    public static CollectedEvents collectUntilSessionClosed(
            Session session, Duration timeout) throws InterruptedException {
        long deadline = System.currentTimeMillis() + timeout.toMillis();
        List<Sidecar.ServerEvent> events = new ArrayList<>();
        List<Sidecar.SidecarError> errors = new ArrayList<>();

        while (System.currentTimeMillis() < deadline) {
            Sidecar.ServerEvent event = session.events().poll(
                    Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            if (event == null) continue;
            events.add(event);
            if (event.hasError()) {
                errors.add(event.getError());
            }
            if (event.hasSessionClosed()) {
                return new CollectedEvents(events, errors);
            }
        }
        throw new InterruptedException("Timed out waiting for SessionClosed");
    }

    /**
     * Collects events until a TurnBoundary TURN_END with the given requestId is received.
     */
    public static CollectedEvents collectUntilTurnEnd(
            Session session, String requestId, Duration timeout) throws InterruptedException {
        long deadline = System.currentTimeMillis() + timeout.toMillis();
        List<Sidecar.ServerEvent> events = new ArrayList<>();
        List<Sidecar.SidecarError> errors = new ArrayList<>();

        while (System.currentTimeMillis() < deadline) {
            Sidecar.ServerEvent event = session.events().poll(
                    Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            if (event == null) continue;
            events.add(event);
            if (event.hasError()) {
                errors.add(event.getError());
            }
            if (!event.getRequestId().equals(requestId)) continue;
            if (event.hasTurn() && event.getTurn().getKind() == Sidecar.TurnBoundary.Kind.TURN_END) {
                return new CollectedEvents(events, errors);
            }
        }
        throw new InterruptedException("Timed out waiting for TURN_END");
    }

    /**
     * Waits for the SessionInit event on the session event channel.
     */
    public static Sidecar.SessionInit waitForSessionInit(
            Session session, Duration timeout) throws InterruptedException {
        long deadline = System.currentTimeMillis() + timeout.toMillis();
        while (System.currentTimeMillis() < deadline) {
            Sidecar.ServerEvent event = session.events().poll(
                    Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            if (event == null) continue;
            if (event.hasSessionInit()) {
                return event.getSessionInit();
            }
        }
        throw new InterruptedException("Timed out waiting for SessionInit");
    }

    /**
     * Waits for a SidecarError event.
     */
    public static Sidecar.SidecarError waitForSidecarError(
            Session session, Duration timeout) throws InterruptedException {
        long deadline = System.currentTimeMillis() + timeout.toMillis();
        while (System.currentTimeMillis() < deadline) {
            Sidecar.ServerEvent event = session.events().poll(
                    Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            if (event == null) continue;
            if (event.hasError()) {
                return event.getError();
            }
        }
        throw new InterruptedException("Timed out waiting for SidecarError");
    }

    /**
     * Finds a SessionSummary with the given sidecar session ID.
     */
    public static Sidecar.SessionSummary findSessionSummary(
            List<Sidecar.SessionSummary> sessions, String sidecarSessionId) {
        for (Sidecar.SessionSummary s : sessions) {
            if (s.getSidecarSessionId().equals(sidecarSessionId)) {
                return s;
            }
        }
        return null;
    }

    /**
     * Checks that no sidecar errors are in the list. Throws AssertionError if any found.
     */
    public static void assertNoSidecarErrors(List<Sidecar.SidecarError> errors) {
        if (errors == null || errors.isEmpty()) return;
        StringBuilder sb = new StringBuilder("unexpected sidecar errors: ");
        for (Sidecar.SidecarError err : errors) {
            sb.append(String.format("%s (fatal=%b): %s; ",
                    err.getCode(), err.getFatal(), err.getMessage()));
        }
        throw new AssertionError(sb.toString());
    }

    /**
     * Checks whether a TURN_END appeared before SessionClosed in the event list.
     */
    public static boolean sawTurnEndBeforeSessionClosed(List<Sidecar.ServerEvent> events) {
        boolean seenTurnEnd = false;
        for (Sidecar.ServerEvent event : events) {
            if (event.hasTurn() && event.getTurn().getKind() == Sidecar.TurnBoundary.Kind.TURN_END) {
                seenTurnEnd = true;
            }
            if (event.hasSessionClosed()) {
                return seenTurnEnd;
            }
        }
        return false;
    }

    /**
     * Finds the index of a message event matching a predicate.
     */
    public static int findMessageEventIndex(
            List<Sidecar.ServerEvent> events,
            java.util.function.Predicate<Sidecar.MessageEvent> predicate) {
        for (int i = 0; i < events.size(); i++) {
            Sidecar.ServerEvent event = events.get(i);
            if (!event.hasMessage()) continue;
            if (predicate.test(event.getMessage())) {
                return i;
            }
        }
        return -1;
    }

    /**
     * Extracts user text messages from a Turn's messages list.
     */
    public static List<String> turnUserTexts(Turn turn) {
        List<String> texts = new ArrayList<>();
        if (turn == null) return texts;
        for (Sidecar.MessageEvent msg : turn.getMessages()) {
            if (!msg.hasUser()) continue;
            Sidecar.UserMessage user = msg.getUser();
            if (user.getContentCount() == 0) continue;
            Sidecar.ContentBlock first = user.getContent(0);
            if (first.hasText()) {
                texts.add(first.getText().getText());
            }
        }
        return texts;
    }

    // ------------------------------------------------------------------
    // Lifecycle
    // ------------------------------------------------------------------

    @Override
    public void close() {
        if (client != null) {
            client.close();
        }
        if (channel != null) {
            channel.shutdown();
            try {
                channel.awaitTermination(3, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                channel.shutdownNow();
            }
        }
    }

    // ------------------------------------------------------------------
    // Internal
    // ------------------------------------------------------------------

    private static boolean parseBool(String s) {
        if (s == null) return false;
        String trimmed = s.trim().toLowerCase();
        return switch (trimmed) {
            case "1", "true", "yes", "y", "on" -> true;
            default -> false;
        };
    }

    // ------------------------------------------------------------------
    // Collected events holder
    // ------------------------------------------------------------------

    public record CollectedEvents(
            List<Sidecar.ServerEvent> events,
            List<Sidecar.SidecarError> errors
    ) {}
}
