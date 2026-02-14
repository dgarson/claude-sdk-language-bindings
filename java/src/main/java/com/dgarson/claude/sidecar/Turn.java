package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar.AssistantMessage;
import claude_sidecar.v1.Sidecar.MessageEvent;
import claude_sidecar.v1.Sidecar.ResultMessage;
import claude_sidecar.v1.Sidecar.ServerEvent;
import claude_sidecar.v1.Sidecar.SidecarError;
import claude_sidecar.v1.Sidecar.StreamEvent;
import claude_sidecar.v1.Sidecar.SystemMessage;
import claude_sidecar.v1.Sidecar.UserMessage;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Collects events for a single conversational turn.
 * Mirrors Go's turns.go Turn struct.
 */
public final class Turn {

    private static final String KIND_USER = "user";
    private static final String KIND_ASSISTANT = "assistant";
    private static final String KIND_SYSTEM = "system";
    private static final String KIND_RESULT = "result";
    private static final String KIND_STREAM_EVENT = "stream_event";

    private String requestId = "";
    private final String turnId;
    private int turnIndex;
    private boolean started;
    private boolean ended;

    private final List<ServerEvent> events = new ArrayList<>();
    private final List<MessageEvent> messages = new ArrayList<>();
    private final List<MessageEvent> partials = new ArrayList<>();
    private final List<StreamEvent> streamEvents = new ArrayList<>();
    private final List<String> stderr = new ArrayList<>();
    private final List<SidecarError> errors = new ArrayList<>();
    private ResultMessage result;
    private final Map<String, MessageEvent> latest = new HashMap<>();

    Turn(String turnId) {
        this.turnId = turnId;
    }

    // -- Accessors --

    public String getRequestId() {
        return requestId;
    }

    void setRequestId(String requestId) {
        this.requestId = requestId;
    }

    public String getTurnId() {
        return turnId;
    }

    public int getTurnIndex() {
        return turnIndex;
    }

    void setTurnIndex(int turnIndex) {
        this.turnIndex = turnIndex;
    }

    public boolean isStarted() {
        return started;
    }

    void setStarted(boolean started) {
        this.started = started;
    }

    public boolean isEnded() {
        return ended;
    }

    void setEnded(boolean ended) {
        this.ended = ended;
    }

    public List<ServerEvent> getEvents() {
        return Collections.unmodifiableList(events);
    }

    void addEvent(ServerEvent event) {
        events.add(event);
    }

    public List<MessageEvent> getMessages() {
        return Collections.unmodifiableList(messages);
    }

    public List<MessageEvent> getPartials() {
        return Collections.unmodifiableList(partials);
    }

    public List<StreamEvent> getStreamEvents() {
        return Collections.unmodifiableList(streamEvents);
    }

    public List<String> getStderr() {
        return Collections.unmodifiableList(stderr);
    }

    void addStderr(String line) {
        stderr.add(line);
    }

    public List<SidecarError> getErrors() {
        return Collections.unmodifiableList(errors);
    }

    void addError(SidecarError error) {
        errors.add(error);
    }

    public ResultMessage getResult() {
        if (result != null) {
            return result;
        }
        MessageEvent event = latest.get(KIND_RESULT);
        if (event != null && event.hasResult()) {
            return event.getResult();
        }
        return null;
    }

    // -- Message accessors --

    public MessageEvent latestMessage(String kind) {
        return latest.get(kind);
    }

    public UserMessage latestUser() {
        MessageEvent event = latestMessage(KIND_USER);
        return event != null && event.hasUser() ? event.getUser() : null;
    }

    public AssistantMessage latestAssistant() {
        MessageEvent event = latestMessage(KIND_ASSISTANT);
        return event != null && event.hasAssistant() ? event.getAssistant() : null;
    }

    /**
     * Returns the latest complete assistant message, falling back to the most recent partial.
     */
    public AssistantMessage mergedAssistant() {
        AssistantMessage assistant = latestAssistant();
        if (assistant != null) {
            return assistant;
        }
        for (int i = partials.size() - 1; i >= 0; i--) {
            MessageEvent partial = partials.get(i);
            if (partial.hasAssistant()) {
                return partial.getAssistant();
            }
        }
        return null;
    }

    public SystemMessage latestSystem() {
        MessageEvent event = latestMessage(KIND_SYSTEM);
        return event != null && event.hasSystem() ? event.getSystem() : null;
    }

    public ResultMessage latestResult() {
        return getResult();
    }

    public StreamEvent latestStreamEvent() {
        MessageEvent event = latestMessage(KIND_STREAM_EVENT);
        return event != null && event.hasStreamEvent() ? event.getStreamEvent() : null;
    }

    // -- Internal mutation --

    void addMessage(MessageEvent message) {
        if (message == null) {
            return;
        }
        if (message.getIsPartial()) {
            partials.add(message);
        } else {
            messages.add(message);
        }
        String kind = messageKind(message);
        if (!kind.isEmpty()) {
            latest.put(kind, message);
        }
        if (message.hasStreamEvent()) {
            streamEvents.add(message.getStreamEvent());
        }
        if (message.hasResult()) {
            result = message.getResult();
        }
    }

    private static String messageKind(MessageEvent message) {
        if (message.hasUser()) return KIND_USER;
        if (message.hasAssistant()) return KIND_ASSISTANT;
        if (message.hasSystem()) return KIND_SYSTEM;
        if (message.hasResult()) return KIND_RESULT;
        if (message.hasStreamEvent()) return KIND_STREAM_EVENT;
        return "";
    }
}
