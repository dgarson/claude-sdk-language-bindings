package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;

import java.util.Map;

/**
 * Static factory methods for constructing {@link Sidecar.HookOutput} proto messages.
 *
 * <p>Mirrors the Go SDK's hook helper functions. All methods are thread-safe.
 */
public final class HookOutputs {

    private HookOutputs() {}

    /**
     * Returns a default hook output that continues execution (the sidecar default).
     */
    public static Sidecar.HookOutput hookDefault() {
        return Sidecar.HookOutput.newBuilder()
                .setContinue(true)
                .build();
    }

    /**
     * Returns a hook output that explicitly continues execution.
     */
    public static Sidecar.HookOutput hookContinue() {
        return Sidecar.HookOutput.newBuilder()
                .setContinue(true)
                .build();
    }

    /**
     * Returns a hook output that stops execution with a reason.
     */
    public static Sidecar.HookOutput hookStop(String reason) {
        return Sidecar.HookOutput.newBuilder()
                .setContinue(false)
                .setStopReason(reason)
                .build();
    }

    /**
     * Returns a hook output that blocks execution with a reason and optional system message.
     */
    public static Sidecar.HookOutput hookBlock(String reason, String systemMessage) {
        Sidecar.HookOutput.Builder b = Sidecar.HookOutput.newBuilder()
                .setContinue(false)
                .setDecision("block")
                .setReason(reason);
        if (systemMessage != null && !systemMessage.isEmpty()) {
            b.setSystemMessage(systemMessage);
        }
        return b.build();
    }

    /**
     * Returns a hook output that defers execution asynchronously with a timeout.
     */
    public static Sidecar.HookOutput hookAsync(int timeoutMs) {
        return Sidecar.HookOutput.newBuilder()
                .setContinue(true)
                .setAsync(true)
                .setAsyncTimeoutMs(timeoutMs)
                .build();
    }

    /**
     * Returns a hook output that suppresses the tool/event output and optionally injects a system message.
     */
    public static Sidecar.HookOutput hookSuppressOutput(String systemMessage) {
        Sidecar.HookOutput.Builder b = Sidecar.HookOutput.newBuilder()
                .setContinue(true)
                .setSuppressOutput(true);
        if (systemMessage != null && !systemMessage.isEmpty()) {
            b.setSystemMessage(systemMessage);
        }
        return b.build();
    }

    /**
     * Returns a new hook output with the given hook-specific output fields merged in.
     *
     * @param output   base hook output to copy from
     * @param specific map of hook-specific output fields
     * @return new HookOutput with hook_specific_output set
     */
    public static Sidecar.HookOutput withSpecificOutput(Sidecar.HookOutput output,
                                                         Map<String, Object> specific) {
        Struct specificStruct = ProtoUtil.mapToStruct(specific);
        return output.toBuilder()
                .setHookSpecificOutput(specificStruct)
                .build();
    }

    /**
     * Returns whether the given hook output indicates execution should continue.
     *
     * <p>If the {@code continue} field is not explicitly set, defaults to {@code true}
     * (matching Claude Code behavior).
     */
    public static boolean hookShouldContinue(Sidecar.HookOutput output) {
        if (!output.hasContinue()) {
            return true; // Claude Code defaults continue=true when omitted
        }
        return output.getContinue();
    }
}
