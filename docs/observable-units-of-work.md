# Observable Units of Work

Reference catalog of metrics and trace spans for the Claude Agent SDK language bindings.

**Related docs:**
- [observability-simple.md](./observability-simple.md) - Minimal implementation (~80 lines)
- [observability-design.md](./observability-design.md) - Full implementation with levels

**Status:** This is a catalog/proposal. The currently implemented sidecar metrics are documented in `claude-agent-sdk-python/python/sidecar/docs/observability.md`.

---

## Label Hygiene Principles

### Avoid High-Cardinality Labels
These cause metric explosion and should **never** be metric labels:
- `session_id`, `conversation_id`, `request_id`
- `user_id`, `prompt_hash`, `prompt_text`
- `tool_args`, `error_message` (free text)

Use these in **traces/logs only**, not metrics.

### Prefer Bounded Enums
- `status`: `ok`, `error`
- `error_class`: `timeout`, `rate_limit`, `auth`, `invalid_request`, `tool_error`, `hook_error`, `io`, `unknown`
- `direction`: `input`, `output`
- `decision`: `allow`, `deny`, `prompt`

### gRPC Status Codes (for `rpc.grpc.status_code` label)
OTEL gRPC instrumentation automatically uses these as label values:
- `OK`, `CANCELLED`, `UNKNOWN`, `INVALID_ARGUMENT`, `DEADLINE_EXCEEDED`
- `NOT_FOUND`, `ALREADY_EXISTS`, `PERMISSION_DENIED`, `RESOURCE_EXHAUSTED`
- `FAILED_PRECONDITION`, `ABORTED`, `OUT_OF_RANGE`, `UNIMPLEMENTED`
- `INTERNAL`, `UNAVAILABLE`, `DATA_LOSS`, `UNAUTHENTICATED`

Use `grpc_code` (not `http_status_class`) for any custom RPC-level metrics.

### Rely on Scrape-Time Labels for Infrastructure
Let Prometheus/collector add these; don't bake into app:
- `job`, `instance`, `namespace`, `pod`, `cluster`, `region`

Keep in-app labels focused on **product behavior**.

---

## Metrics Catalog

### Priority Legend
- **P0**: Must have for MVP (implement in simple version)
- **P1**: High value, implement early
- **P2**: Useful for debugging, implement when needed
- **P3**: Nice to have

---

### Sidecar Metrics

#### Request/Response Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.requests` | Counter | P0 | `operation`, `status` | Total requests by endpoint |
| `claude.sidecar.request.duration` | Histogram | P0 | `operation` | Request latency (seconds) |
| `claude.sidecar.inflight_requests` | Gauge | P1 | `operation` | Current concurrent requests |
| `claude.sidecar.errors` | Counter | P0 | `error_class`, `component` | Errors by class and component |
| `claude.sidecar.retries` | Counter | P2 | `operation`, `retry_reason` | Retry attempts |

**Label values:**
- `operation`: `create_session`, `attach_session`, `query`, `delete_session`, `fork_session`, `health_check`
- `status`: `ok`, `error`
- `error_class`: `timeout`, `rate_limit`, `auth`, `invalid_request`, `tool_error`, `hook_error`, `io`, `unknown`
- `component`: `client`, `sidecar`, `hook`, `tool`, `mcp`, `sdk`
- `retry_reason`: `timeout`, `rate_limit`, `unavailable`, `resource_exhausted`, `network`, `reset`

#### Session Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.sessions` | UpDownCounter | P0 | — | Active session count |
| `claude.sidecar.sessions.total` | Counter | P0 | `status` | Total sessions created |
| `claude.sidecar.session.duration` | Histogram | P1 | `status` | Session lifetime (seconds) |

**Label values:**
- `status`: `created`, `closed`, `error`, `timeout`

#### Turn Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.turns` | Counter | P1 | `status` | Total turns processed |
| `claude.sidecar.turn.duration` | Histogram | P1 | `status`, `has_tool_calls` | Turn E2E latency (seconds) |

**Label values:**
- `status`: `ok`, `error`, `interrupted`
- `has_tool_calls`: `true`, `false`

#### Token Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.tokens` | Counter | P1 | `direction`, `model` | Token usage |

**Label values:**
- `direction`: `input`, `output`
- `model`: `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku`, etc. (bounded by supported models)

#### Callback Metrics (Tool/Hook/Permission)

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.callback.requests` | Counter | P0 | `callback_type` | Callback invocations |
| `claude.sidecar.callback.duration` | Histogram | P0 | `callback_type` | Callback latency (seconds) |
| `claude.sidecar.callback.timeouts` | Counter | P0 | `callback_type` | Callback timeouts |

**Label values:**
- `callback_type`: `tool`, `hook`, `permission`

#### Tool Metrics (Detailed)

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.tool.calls` | Counter | P2 | `tool_name`, `status`, `tool_source` | Tool invocations |
| `claude.sidecar.tool.duration` | Histogram | P2 | `tool_name`, `status` | Tool execution time (seconds) |

**Label values:**
- `tool_name`: bounded by configured tools (e.g., `read_file`, `write_file`, `bash`)
- `status`: `ok`, `error`
- `tool_source`: `local`, `mcp`, `client`

#### Hook Metrics (Detailed)

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.hook.executions` | Counter | P2 | `hook_name`, `hook_phase`, `status` | Hook invocations |
| `claude.sidecar.hook.duration` | Histogram | P2 | `hook_name`, `hook_phase` | Hook execution time (seconds) |

**Label values:**
- `hook_name`: bounded by configured hooks
- `hook_phase`: `pre`, `post`
- `status`: `ok`, `error`, `skipped`

#### Permission Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.permission.decisions` | Counter | P2 | `decision`, `scope` | Permission outcomes |

**Label values:**
- `decision`: `allow`, `deny`, `prompt`
- `scope`: `tool`, `fs`, `network`, `mcp`

#### MCP Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.mcp.requests` | Counter | P2 | `mcp_server`, `mcp_method`, `status` | MCP server requests |
| `claude.sidecar.mcp.duration` | Histogram | P2 | `mcp_server`, `mcp_method` | MCP request latency (seconds) |

**Label values:**
- `mcp_server`: bounded by configured servers
- `mcp_method`: `list_tools`, `call_tool`, `list_resources`, `read_resource`
- `status`: `ok`, `error`

#### Stream Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.stream.disconnects` | Counter | P2 | `reason` | Stream disconnections |
| `claude.sidecar.stream.events` | Counter | P2 | `event_type` | Stream events processed |

**Label values:**
- `reason`: `client_cancel`, `timeout`, `server_close`, `network_error`, `complete`
- `event_type`: `message`, `tool_use`, `tool_result`, `partial`, `turn_boundary`

#### Queue Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.sidecar.queue.depth` | Gauge | P3 | `queue_name` | Queue depth |

**Label values:**
- `queue_name`: `outgoing`, `commands`

---

### Client Metrics

#### Connection Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.client.connections` | Counter | P0 | `status` | Connection attempts |
| `claude.client.connections.active` | UpDownCounter | P0 | — | Active connections |

**Label values:**
- `status`: `ok`, `error`

#### RPC Metrics (Auto from OTEL gRPC)

These are provided automatically by OTEL gRPC instrumentation:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `rpc.client.duration` | Histogram | `rpc.method`, `rpc.grpc.status_code` | RPC latency |
| `rpc.client.request.size` | Histogram | `rpc.method` | Request size (bytes) |
| `rpc.client.response.size` | Histogram | `rpc.method` | Response size (bytes) |

#### Handler Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.client.handler.calls` | Counter | P1 | `handler_type`, `status` | Handler invocations |
| `claude.client.handler.duration` | Histogram | P1 | `handler_type` | Handler execution time (seconds) |

**Label values:**
- `handler_type`: `tool`, `hook`, `permission`
- `status`: `ok`, `error`

#### Turn Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.client.turns` | Counter | P1 | `status` | Turns processed |
| `claude.client.turn.duration` | Histogram | P1 | — | Turn duration (seconds) |

**Label values:**
- `status`: `ok`, `error`, `interrupted`

#### Event Mux Metrics

| Metric | Type | Priority | Labels | Description |
|--------|------|----------|--------|-------------|
| `claude.client.subscriptions.active` | UpDownCounter | P3 | — | Active subscriptions |
| `claude.client.event_queue.depth` | Gauge | P3 | — | Event queue depth |

---

## Trace Spans Catalog

Spans provide request-scoped context with high-cardinality attributes (session_id, request_id) that are unsafe for metrics.

### Sidecar Spans

| Span Name | Priority | Attributes | Description |
|-----------|----------|------------|-------------|
| `sidecar.session.create` | P1 | `session_id`, `mode` | Session creation |
| `sidecar.session.attach` | P1 | `session_id`, `client_name` | Client attachment |
| `sidecar.session.close` | P1 | `session_id`, `reason` | Session closure |
| `sidecar.query` | P0 | `session_id`, `request_id`, `prompt_length` | Query processing |
| `sidecar.turn` | P0 | `session_id`, `turn_id`, `turn_index` | Turn processing |
| `sidecar.sdk.query` | P1 | `session_id`, `model` | SDK query to Claude |
| `sidecar.sdk.receive` | P1 | `session_id`, `message_type` | SDK message reception |
| `sidecar.callback.tool` | P1 | `session_id`, `tool_fqn`, `invocation_id` | Tool callback roundtrip |
| `sidecar.callback.hook` | P1 | `session_id`, `hook_event`, `invocation_id` | Hook callback roundtrip |
| `sidecar.callback.permission` | P1 | `session_id`, `tool_name`, `invocation_id` | Permission callback roundtrip |
| `sidecar.mcp.request` | P2 | `mcp_server`, `mcp_method`, `mcp_tool` | MCP server request |

### Client Spans

| Span Name | Priority | Attributes | Description |
|-----------|----------|------------|-------------|
| `client.dial` | P1 | `address` | Connection establishment |
| `client.session.attach` | P1 | `session_id` | Session attachment |
| `client.query` | P0 | `session_id`, `request_id` | Query submission |
| `client.run` | P0 | `session_id`, `request_id` | Run (query + wait for result) |
| `client.stream` | P1 | `session_id`, `request_id` | Stream consumption |
| `client.handler.tool` | P1 | `invocation_id`, `tool_fqn` | Tool handler execution |
| `client.handler.hook` | P1 | `invocation_id`, `hook_event` | Hook handler execution |
| `client.handler.permission` | P1 | `invocation_id`, `tool_name` | Permission handler execution |

---

## Implementation Tiers

### Tier 1: MVP (Simple Version)
Implement these in `observability-simple.md`:

**Sidecar:**
- `claude.sidecar.sessions` (UpDownCounter)
- `claude.sidecar.sessions.total` (Counter)
- `claude.sidecar.callback.duration` (Histogram)
- `claude.sidecar.callback.timeouts` (Counter)
- `claude.sidecar.errors` (Counter)
- Auto gRPC metrics via OTEL

**Client:**
- `claude.client.connections.active` (UpDownCounter)
- `claude.client.handler.duration` (Histogram)
- `claude.client.turns` (Counter)
- Auto gRPC metrics via OTEL

### Tier 2: Enhanced Observability
Add when needed:

- Token counting (`claude.sidecar.tokens`)
- Turn duration (`claude.sidecar.turn.duration`)
- Inflight requests gauge
- Retry metrics
- Session duration

### Tier 3: Detailed Debugging
Add for specific debugging needs:

- Per-tool metrics (`claude.sidecar.tool.*`)
- Per-hook metrics (`claude.sidecar.hook.*`)
- MCP metrics (`claude.sidecar.mcp.*`)
- Permission decision metrics
- Stream disconnect tracking
- Queue depth gauges

### Tier 4: Tracing
Add when distributed tracing is needed:

- All spans listed above
- Trace context propagation across gRPC
- Correlation with metrics via exemplars

---

## Histogram Bucket Recommendations

| Metric Type | Buckets (seconds) |
|-------------|-------------------|
| RPC/Request latency | 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 |
| Callback/Handler | 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60 |
| Turn/Query E2E | 0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300 |
| Session lifetime | 1, 5, 10, 30, 60, 300, 600, 1800, 3600 |

---

## Prometheus Naming (After OTEL Export)

OTEL metrics get converted to Prometheus format:
- Dots → underscores: `claude.sidecar.sessions` → `claude_sidecar_sessions`
- Histogram suffixes added: `_bucket`, `_sum`, `_count`
- Counter suffix: `_total`

Example queries:
```promql
# Request rate by operation
sum(rate(claude_sidecar_requests_total[5m])) by (operation)

# p99 callback latency by type
histogram_quantile(0.99, sum(rate(claude_sidecar_callback_duration_seconds_bucket[5m])) by (le, callback_type))

# Error rate by class
sum(rate(claude_sidecar_errors_total[5m])) by (error_class)

# Token burn rate by model
sum(rate(claude_sidecar_tokens_total[5m])) by (model, direction)

# gRPC error rate by status code (from auto-instrumented metrics)
sum(rate(rpc_server_duration_milliseconds_count{rpc_grpc_status_code!="OK"}[5m])) by (rpc_grpc_status_code, rpc_method)

# gRPC p99 latency by method
histogram_quantile(0.99, sum(rate(rpc_server_duration_milliseconds_bucket[5m])) by (le, rpc_method))
```
