package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar;
import com.google.protobuf.Struct;

import java.util.List;
import java.util.Map;

/**
 * Static factory methods for constructing {@link Sidecar.PermissionDecision} proto messages.
 *
 * <p>Mirrors the Go SDK's permission helper functions. All methods are thread-safe.
 */
public final class PermissionDecisions {

    private PermissionDecisions() {}

    /**
     * Creates an "allow" permission decision with the given reason.
     */
    public static Sidecar.PermissionDecision allow(String reason) {
        return Sidecar.PermissionDecision.newBuilder()
                .setBehavior("allow")
                .setReason(reason)
                .build();
    }

    /**
     * Creates a "deny" permission decision with the given reason.
     */
    public static Sidecar.PermissionDecision deny(String reason) {
        return Sidecar.PermissionDecision.newBuilder()
                .setBehavior("deny")
                .setReason(reason)
                .build();
    }

    /**
     * Creates an "ask" permission decision, deferring to the user/UI with the given reason.
     */
    public static Sidecar.PermissionDecision ask(String reason) {
        return Sidecar.PermissionDecision.newBuilder()
                .setBehavior("ask")
                .setReason(reason)
                .build();
    }

    /**
     * Returns a new decision with the tool input replaced by the given updated input map.
     *
     * @param decision base decision to copy from
     * @param updated  new tool input values
     */
    public static Sidecar.PermissionDecision withUpdatedInput(
            Sidecar.PermissionDecision decision,
            Map<String, Object> updated) {
        Struct updatedStruct = ProtoUtil.mapToStruct(updated);
        return decision.toBuilder()
                .setUpdatedInput(updatedStruct)
                .build();
    }

    /**
     * Returns a new decision with permission updates attached.
     *
     * <p>Each {@link PermissionUpdate} is converted to its map form and the list
     * is serialized as a protobuf Value (list of structs).
     *
     * @param decision base decision to copy from
     * @param updates  list of permission updates to apply
     */
    public static Sidecar.PermissionDecision withUpdatedPermissions(
            Sidecar.PermissionDecision decision,
            List<PermissionUpdate> updates) {
        List<Object> updateMaps = updates.stream()
                .map(PermissionUpdate::toMap)
                .map(m -> (Object) m)
                .toList();
        return decision.toBuilder()
                .setUpdatedPermissions(ProtoUtil.toValue(updateMaps))
                .build();
    }

    /**
     * Returns a new decision with the interrupt flag set.
     *
     * @param decision  base decision to copy from
     * @param interrupt whether to interrupt the current run/session
     */
    public static Sidecar.PermissionDecision withInterrupt(
            Sidecar.PermissionDecision decision,
            boolean interrupt) {
        return decision.toBuilder()
                .setInterrupt(interrupt)
                .build();
    }
}
