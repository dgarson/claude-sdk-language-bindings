package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar.HookInvocationRequest;
import claude_sidecar.v1.Sidecar.HookOutput;
import claude_sidecar.v1.Sidecar.PermissionDecision;
import claude_sidecar.v1.Sidecar.PermissionDecisionRequest;
import claude_sidecar.v1.Sidecar.ToolInvocationRequest;
import com.google.protobuf.Struct;

/**
 * Callback handlers for tool invocations, hook invocations, and permission decisions.
 * Mirrors Go's types.go Handlers struct.
 */
public final class Handlers {

    @FunctionalInterface
    public interface ToolHandler {
        Struct handle(ToolInvocationRequest request) throws Exception;
    }

    @FunctionalInterface
    public interface HookHandler {
        HookOutput handle(HookInvocationRequest request) throws Exception;
    }

    @FunctionalInterface
    public interface PermissionHandler {
        PermissionDecision handle(PermissionDecisionRequest request) throws Exception;
    }

    private final ToolHandler toolHandler;
    private final HookHandler hookHandler;
    private final PermissionHandler permissionHandler;

    private Handlers(ToolHandler toolHandler, HookHandler hookHandler, PermissionHandler permissionHandler) {
        this.toolHandler = toolHandler;
        this.hookHandler = hookHandler;
        this.permissionHandler = permissionHandler;
    }

    public ToolHandler toolHandler() {
        return toolHandler;
    }

    public HookHandler hookHandler() {
        return hookHandler;
    }

    public PermissionHandler permissionHandler() {
        return permissionHandler;
    }

    public static Builder builder() {
        return new Builder();
    }

    /** Creates empty handlers (all null). Callback requests will get default error responses. */
    public static Handlers empty() {
        return new Handlers(null, null, null);
    }

    public static final class Builder {
        private ToolHandler toolHandler;
        private HookHandler hookHandler;
        private PermissionHandler permissionHandler;

        private Builder() {}

        public Builder tool(ToolHandler handler) {
            this.toolHandler = handler;
            return this;
        }

        public Builder hook(HookHandler handler) {
            this.hookHandler = handler;
            return this;
        }

        public Builder permission(PermissionHandler handler) {
            this.permissionHandler = handler;
            return this;
        }

        public Handlers build() {
            return new Handlers(toolHandler, hookHandler, permissionHandler);
        }
    }
}
