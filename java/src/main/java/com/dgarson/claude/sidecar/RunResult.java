package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar.AssistantMessage;
import claude_sidecar.v1.Sidecar.ResultMessage;

/**
 * Wraps a completed {@link Turn}, providing convenience accessors for the assistant and result
 * messages. Mirrors Go's run.go RunResult.
 */
public final class RunResult {

    private final Turn turn;

    public RunResult(Turn turn) {
        this.turn = turn;
    }

    public Turn getTurn() {
        return turn;
    }

    /** Returns the merged assistant message for this turn, or null. */
    public AssistantMessage getAssistant() {
        return turn != null ? turn.mergedAssistant() : null;
    }

    /** Returns the latest result message for this turn, or null. */
    public ResultMessage getResult() {
        return turn != null ? turn.latestResult() : null;
    }
}
