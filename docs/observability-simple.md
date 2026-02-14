# Observability Design (Simple)

> **See also:**
> - [observable-units-of-work.md](./observable-units-of-work.md) - Full catalog of metrics and spans with label guidance
> - [observability-design.md](./observability-design.md) - Full-featured implementation with 5 granularity levels
>
> **Status:** This document is a design sketch. The currently implemented sidecar metrics are documented in `claude-agent-sdk-python/python/sidecar/docs/observability.md` and are emitted via the OpenTelemetry metrics API (exported via an OTEL Collector Prometheus exporter).
>
> **Use this version** if you want to ship quickly with sensible defaults. **Use the full version** if you need fine-grained control over metric cardinality or per-tool/hook breakdowns.

## Overview

Minimal observability using OpenTelemetry with Prometheus export. Two modes: enabled or disabled.

## Configuration

```bash
OTEL_METRICS_ENABLED=true           # true | false
OTEL_EXPORTER_PROMETHEUS_PORT=9090  # metrics endpoint port
OTEL_SERVICE_NAME=claude-sidecar    # service identifier
```

## What You Get

### Automatic (from OTEL gRPC instrumentation)
- `rpc.server.duration{rpc.method, rpc.grpc.status_code}` - RPC latency
- `rpc.client.duration{rpc.method, rpc.grpc.status_code}` - Client RPC latency

### Custom Metrics

**Sidecar:**
```
claude.sidecar.sessions              UpDownCounter  - active session count
claude.sidecar.sessions.total        Counter        - total sessions created
claude.sidecar.callback.duration     Histogram      - callback latency by type
claude.sidecar.callback.timeouts     Counter        - callback timeout count
claude.sidecar.errors                Counter        - error count by code
```

**Client:**
```
claude.client.connections.active     UpDownCounter  - active connections
claude.client.handler.duration       Histogram      - handler latency by type
claude.client.turns                  Counter        - turn count
```

---

## Python Implementation

### File: `observability.py` (~80 lines)

```python
"""Simple OpenTelemetry observability with Prometheus export."""

import os
from contextlib import contextmanager
from time import perf_counter

from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.exporter.prometheus import PrometheusMetricReader
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer

_enabled = os.getenv("OTEL_METRICS_ENABLED", "true").lower() == "true"
_initialized = False

# Metric instruments (None if disabled)
sessions_active = None
sessions_total = None
callback_duration = None
callback_timeouts = None
errors_total = None


def init():
    """Initialize OpenTelemetry. Call once at startup."""
    global _initialized, sessions_active, sessions_total
    global callback_duration, callback_timeouts, errors_total

    if _initialized or not _enabled:
        return

    resource = Resource.create({
        SERVICE_NAME: os.getenv("OTEL_SERVICE_NAME", "claude-sidecar"),
    })

    provider = MeterProvider(
        resource=resource,
        metric_readers=[PrometheusMetricReader()],
    )
    metrics.set_meter_provider(provider)

    # Auto-instrument gRPC
    GrpcInstrumentorServer().instrument()

    # Create custom metrics
    meter = provider.get_meter("claude.sidecar")

    sessions_active = meter.create_up_down_counter(
        "claude.sidecar.sessions",
        description="Active sessions",
    )
    sessions_total = meter.create_counter(
        "claude.sidecar.sessions.total",
        description="Total sessions created",
    )
    callback_duration = meter.create_histogram(
        "claude.sidecar.callback.duration",
        unit="s",
        description="Callback duration",
    )
    callback_timeouts = meter.create_counter(
        "claude.sidecar.callback.timeouts",
        description="Callback timeouts",
    )
    errors_total = meter.create_counter(
        "claude.sidecar.errors",
        description="Errors",
    )

    _initialized = True


@contextmanager
def timed_callback(callback_type: str):
    """Context manager to time and record callback duration."""
    start = perf_counter()
    timed_out = False
    try:
        yield
    except TimeoutError:
        timed_out = True
        raise
    finally:
        if callback_duration:
            callback_duration.record(perf_counter() - start, {"type": callback_type})
        if timed_out and callback_timeouts:
            callback_timeouts.add(1, {"type": callback_type})
```

### Integration

**serve.py** (+2 lines):
```python
from . import observability

async def serve(host: str, port: int) -> None:
    observability.init()  # Add this line
    # ... rest unchanged
```

**sessions.py** (+4 lines):
```python
from . import observability

class Session:
    def __init__(self, ...):
        # ... existing code ...
        if observability.sessions_active:
            observability.sessions_active.add(1)
            observability.sessions_total.add(1, {"status": "created"})

    async def close(self, reason: str):
        if self.closed:
            return
        self.closed = True
        if observability.sessions_active:
            observability.sessions_active.add(-1)
        # ... rest unchanged
```

**callback_router.py** (+3 lines):
```python
from .observability import timed_callback

class CallbackRouter:
    async def request_tool(self, tool_fqn: str, ...):
        with timed_callback("tool"):
            # ... existing implementation unchanged
```

---

## Go Implementation

### File: `observability/observability.go` (~100 lines)

```go
package observability

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
)

var (
	enabled     = os.Getenv("OTEL_METRICS_ENABLED") != "false"
	initialized = false
	initOnce    sync.Once
	meter       metric.Meter

	// Metrics
	ConnectionsActive metric.Int64UpDownCounter
	HandlerDuration   metric.Float64Histogram
	TurnsTotal        metric.Int64Counter
)

func Init(ctx context.Context) error {
	if !enabled {
		return nil
	}

	var initErr error
	initOnce.Do(func() {
		res, err := resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(getEnv("OTEL_SERVICE_NAME", "claude-client-go")),
			),
		)
		if err != nil {
			initErr = err
			return
		}

		registry := prom.NewRegistry()
		exporter, err := prometheus.New(prometheus.WithRegisterer(registry))
		if err != nil {
			initErr = err
			return
		}

		provider := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(exporter),
		)
		otel.SetMeterProvider(provider)
		meter = provider.Meter("claude.client")

		// Create metrics
		ConnectionsActive, _ = meter.Int64UpDownCounter("claude.client.connections.active")
		HandlerDuration, _ = meter.Float64Histogram("claude.client.handler.duration", metric.WithUnit("s"))
		TurnsTotal, _ = meter.Int64Counter("claude.client.turns")

		// Start Prometheus HTTP server
		port := getEnv("OTEL_EXPORTER_PROMETHEUS_PORT", "9091")
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
			http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
		}()

		initialized = true
	})
	return initErr
}

// GRPCDialOptions returns dial options for auto gRPC instrumentation.
func GRPCDialOptions() []grpc.DialOption {
	if !enabled {
		return nil
	}
	return []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}

// RecordHandlerDuration records handler execution time.
func RecordHandlerDuration(ctx context.Context, handlerType string, d time.Duration) {
	if HandlerDuration != nil {
		HandlerDuration.Record(ctx, d.Seconds(), metric.WithAttributes(
			attribute.String("type", handlerType),
		))
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

### Integration

**client.go** (+5 lines):
```go
func Dial(ctx context.Context, addr string, opts ...grpc.DialOption) (*Client, error) {
    observability.Init(ctx)  // Add this

    opts = append(opts, observability.GRPCDialOptions()...)  // Add this

    // ... rest unchanged
}
```

---

## Dependencies

### Python
```toml
[project.optional-dependencies]
observability = [
    "opentelemetry-api>=1.22.0",
    "opentelemetry-sdk>=1.22.0",
    "opentelemetry-exporter-prometheus>=0.43b0",
    "opentelemetry-instrumentation-grpc>=0.43b0",
]
```

### Go
```go
require (
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/sdk/metric v1.24.0
    go.opentelemetry.io/otel/exporters/prometheus v0.46.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
    github.com/prometheus/client_golang v1.19.0
)
```

---

## Grafana Queries

```promql
# Active sessions
claude_sidecar_sessions

# RPC latency p99 by method
histogram_quantile(0.99, sum(rate(rpc_server_duration_milliseconds_bucket[5m])) by (le, rpc_method))

# Callback latency p99 by type
histogram_quantile(0.99, sum(rate(claude_sidecar_callback_duration_bucket[5m])) by (le, type))

# Error rate
sum(rate(claude_sidecar_errors[5m])) by (code)
```

---

## Adding More Later

Need per-tool metrics? Add one label:
```python
callback_duration.record(duration, {"type": "tool", "tool_fqn": tool_fqn})
```

Need to reduce cardinality? See [observability-design.md](./observability-design.md) for the leveled approach.

Need tracing? Add OTEL trace exporter - same SDK, just enable it:
```bash
OTEL_TRACES_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```
