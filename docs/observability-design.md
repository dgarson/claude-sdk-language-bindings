# Observability Design for Claude Agent SDK Language Bindings (Full)

> ## Which Version Should I Use?
>
> | Version | When to Use |
> |---------|-------------|
> | **[observability-simple.md](./observability-simple.md)** | Ship quickly with sensible defaults. On/off toggle only. ~80 lines per language. **Start here.** |
> | **This document** | Need fine-grained control: 5 granularity levels, per-tool/hook metrics, cardinality management, elaborate decorator patterns. |
> | **[observable-units-of-work.md](./observable-units-of-work.md)** | Reference catalog of all metrics and spans with label guidance and priority tiers. |
>
> **Recommendation**: Start with the simple version. Migrate to this version only if you need per-tool metrics or cardinality controls that users are actually requesting.
>
> **Status:** This is a design proposal. For the current implemented sidecar metrics + disconnect classification, see `claude-agent-sdk-python/python/sidecar/docs/observability.md`.

---

## Overview

This document outlines the **full-featured** observability strategy for the Claude Agent SDK language bindings, covering metrics, logging, and optional tracing for both the Python sidecar and Go/Java/etc clients.

**Core Principle**: Use **OpenTelemetry (OTEL)** as the unified instrumentation framework across all languages, with **Prometheus** as the metrics export format for cloud-native scraping.

## Why OpenTelemetry?

1. **Vendor-agnostic**: Single API that can export to Prometheus, OTLP, Jaeger, Datadog, etc.
2. **Unified SDK**: Metrics, traces, and logs share the same instrumentation patterns
3. **Language support**: Mature SDKs for Python, Go, Java, Rust, .NET, and more
4. **Industry standard**: CNCF graduated project with broad ecosystem support
5. **Future-proof**: Adding tracing later requires no API changes

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      OpenTelemetry Architecture                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Application Code                                                            │
│       │                                                                      │
│       ▼                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │              OpenTelemetry SDK (Language-specific)                   │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │    │
│  │  │   Metrics    │  │   Traces     │  │    Logs      │               │    │
│  │  │   API        │  │   API        │  │    API       │               │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │    │
│  │           │                │                │                        │    │
│  │           ▼                ▼                ▼                        │    │
│  │  ┌─────────────────────────────────────────────────────────────┐    │    │
│  │  │                    Exporters                                 │    │    │
│  │  │  ┌────────────┐  ┌────────────┐  ┌────────────┐             │    │    │
│  │  │  │ Prometheus │  │   OTLP     │  │  Console   │   ...       │    │    │
│  │  │  │ (pull)     │  │  (push)    │  │  (debug)   │             │    │    │
│  │  │  └────────────┘  └────────────┘  └────────────┘             │    │    │
│  │  └─────────────────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│       │ Prometheus scrape          │ OTLP push                              │
│       ▼                            ▼                                        │
│  ┌─────────────┐            ┌─────────────────┐                             │
│  │ Prometheus  │            │ OTEL Collector  │                             │
│  │ Server      │            │ (optional)      │                             │
│  └─────────────┘            └─────────────────┘                             │
│       │                            │                                        │
│       ▼                            ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         Grafana                                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        System Observability Architecture                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────┐         ┌─────────────────────────────┐   │
│  │     Python Sidecar          │         │     Go/Java Client          │   │
│  │                             │         │                             │   │
│  │  ┌───────────────────────┐  │  gRPC   │  ┌───────────────────────┐  │   │
│  │  │ OTEL gRPC Interceptor │◄─┼─────────┼─►│ OTEL gRPC Interceptor │  │   │
│  │  │ (auto RPC metrics)    │  │         │  │ (auto RPC metrics)    │  │   │
│  │  └───────────────────────┘  │         │  └───────────────────────┘  │   │
│  │                             │         │                             │   │
│  │  ┌───────────────────────┐  │         │  ┌───────────────────────┐  │   │
│  │  │ OTEL Meter Provider   │  │         │  │ OTEL Meter Provider   │  │   │
│  │  │ - session metrics     │  │         │  │ - handler metrics     │  │   │
│  │  │ - callback metrics    │  │         │  │ - turn metrics        │  │   │
│  │  │ - queue depths        │  │         │  │ - stream metrics      │  │   │
│  │  └───────────────────────┘  │         │  └───────────────────────┘  │   │
│  │                             │         │                             │   │
│  │  ┌───────────────────────┐  │         │  ┌───────────────────────┐  │   │
│  │  │ Prometheus Exporter   │  │         │  │ Prometheus Exporter   │  │   │
│  │  │ :9090/metrics         │  │         │  │ :9091/metrics         │  │   │
│  │  └───────────────────────┘  │         │  └───────────────────────┘  │   │
│  │                             │         │                             │   │
│  └─────────────────────────────┘         └─────────────────────────────┘   │
│                                                                              │
│                              ▼                                              │
│                    ┌─────────────────────┐                                  │
│                    │    Prometheus       │                                  │
│                    │    Scraper          │                                  │
│                    └─────────────────────┘                                  │
│                              ▼                                              │
│                    ┌─────────────────────┐                                  │
│                    │    Grafana          │                                  │
│                    │    Dashboards       │                                  │
│                    └─────────────────────┘                                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Observability Levels

The system supports 5 levels of observability granularity, controlled by a single configuration knob:

| Level | Description | Use Case | Cardinality |
|-------|-------------|----------|-------------|
| `disabled` | No metrics collection | Production with no observability needs | None |
| `minimal` | Core operational metrics only | Low-overhead production monitoring | Very Low |
| `standard` | Per-RPC and per-callback-type metrics | **Default** - typical production use | Low |
| `verbose` | Per-tool, queue depths, stream details | Debugging specific issues | Medium |
| `debug` | High-cardinality: session/request IDs | Development/troubleshooting | High |

### Attribute Strategy by Level

OTEL uses "attributes" (equivalent to Prometheus labels):

```
Level      │ Attributes Included
───────────┼─────────────────────────────────────────────────
disabled   │ (none - no metrics emitted)
minimal    │ component, status
standard   │ above + rpc.method, callback.type
verbose    │ above + tool.fqn, hook.event, message.type
debug      │ above + session.id, request.id, turn.id
```

## Configuration

### Environment Variables (Primary)

```bash
# Master controls
OTEL_METRICS_ENABLED=true                    # Master kill switch
OTEL_OBSERVABILITY_LEVEL=standard            # disabled|minimal|standard|verbose|debug

# Prometheus exporter (pull-based)
OTEL_EXPORTER_PROMETHEUS_PORT=9090           # Sidecar metrics port
OTEL_EXPORTER_PROMETHEUS_HOST=0.0.0.0        # Bind address

# OTLP exporter (push-based, optional)
OTEL_EXPORTER_OTLP_ENDPOINT=                 # e.g., http://localhost:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc             # grpc|http/protobuf

# Service identification (standard OTEL)
OTEL_SERVICE_NAME=claude-sidecar             # or claude-client-go
OTEL_SERVICE_VERSION=0.1.0

# Tracing (optional)
OTEL_TRACES_ENABLED=false
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1                  # 10% sampling

# Logging
OTEL_LOG_LEVEL=INFO                          # DEBUG|INFO|WARN|ERROR
```

### Python Configuration

```python
from claude_sidecar.observability import ObservabilityConfig, ObservabilityLevel

# From environment (recommended - uses standard OTEL env vars)
config = ObservabilityConfig.from_env()

# Explicit configuration
config = ObservabilityConfig(
    level=ObservabilityLevel.STANDARD,
    metrics_enabled=True,
    prometheus_port=9090,
    service_name="claude-sidecar",
    service_version="0.1.0",
    traces_enabled=False,
    otlp_endpoint=None,  # Set for push-based export
)
```

### Go Configuration

```go
import "github.com/dgarson/claude-sidecar/client/observability"

// From environment (recommended - uses standard OTEL env vars)
config := observability.ConfigFromEnv()

// Explicit configuration
config := &observability.Config{
    Level:           observability.LevelStandard,
    MetricsEnabled:  true,
    PrometheusPort:  9091,
    ServiceName:     "claude-client-go",
    ServiceVersion:  "0.1.0",
    TracesEnabled:   false,
    OTLPEndpoint:    "", // Set for push-based export
}
```

---

## Metric Taxonomy

### Naming Conventions (OTEL Semantic Conventions)

- **Namespace**: `claude.sidecar.` for Python sidecar, `claude.client.` for clients
- **Units in name**: `_seconds`, `_bytes`, `_total` (counter)
- **Attributes**: dot.separated.names following OTEL conventions

### Python Sidecar Metrics

#### Minimal Level
```
# Server operational metrics
claude.sidecar.sessions{state="active"}                    UpDownCounter
claude.sidecar.sessions.total{status="created|closed|error"}   Counter

# Error tracking
claude.sidecar.errors{error.code}                          Counter
```

#### Standard Level (includes minimal)
```
# RPC metrics (via OTEL gRPC instrumentation)
rpc.server.duration{rpc.method,rpc.grpc.status_code}       Histogram (ms)
rpc.server.request.size{rpc.method}                        Histogram (bytes)
rpc.server.response.size{rpc.method}                       Histogram (bytes)

# Session lifecycle
claude.sidecar.session.duration{status}                    Histogram (s)
claude.sidecar.session.turns{status}                       Counter

# Callback metrics (aggregated by type)
claude.sidecar.callback.requests{callback.type}            Counter
claude.sidecar.callback.duration{callback.type}            Histogram (s)
claude.sidecar.callback.timeouts{callback.type}            Counter
```

#### Verbose Level (includes standard)
```
# Queue depths (observable gauges)
claude.sidecar.queue.depth{queue.name="outgoing|commands"} Gauge

# Per-tool metrics
claude.sidecar.tool.invocations{tool.fqn}                  Counter
claude.sidecar.tool.duration{tool.fqn}                     Histogram (s)

# Per-hook metrics
claude.sidecar.hook.invocations{hook.event}                Counter
claude.sidecar.hook.duration{hook.event}                   Histogram (s)

# Message flow
claude.sidecar.messages{message.type}                      Counter

# SDK interaction
claude.sidecar.sdk.query.duration{}                        Histogram (s)
```

#### Debug Level (includes verbose)
```
# High-cardinality metrics (use with caution)
claude.sidecar.request.duration{session.id,request.id}     Histogram (s)
claude.sidecar.turn.duration{session.id,turn.id}           Histogram (s)
```

### Go/Java Client Metrics

#### Minimal Level
```
# Connection health
claude.client.connections{status="success|error"}          Counter
claude.client.connections.active{}                         UpDownCounter
```

#### Standard Level
```
# RPC metrics (via OTEL gRPC instrumentation)
rpc.client.duration{rpc.method,rpc.grpc.status_code}       Histogram (ms)
rpc.client.request.size{rpc.method}                        Histogram (bytes)
rpc.client.response.size{rpc.method}                       Histogram (bytes)

# Handler metrics (aggregated by type)
claude.client.handler.calls{handler.type,status}           Counter
claude.client.handler.duration{handler.type}               Histogram (s)

# Turn metrics
claude.client.turns{status}                                Counter
claude.client.turn.duration{}                              Histogram (s)
```

#### Verbose Level
```
# Stream metrics
claude.client.stream.events{event.type}                    Counter
claude.client.stream.duration{}                            Histogram (s)

# Event mux metrics
claude.client.event_queue.depth{}                          Gauge
claude.client.subscriptions.active{}                       UpDownCounter

# Per-tool/hook handlers
claude.client.tool_handler.duration{tool.fqn}              Histogram (s)
claude.client.hook_handler.duration{hook.event}            Histogram (s)
```

---

## Implementation Design

### Python Sidecar Implementation

#### File Structure
```
python/sidecar/claude_sidecar/
├── observability/
│   ├── __init__.py              # Public API exports
│   ├── config.py                # ObservabilityConfig, Level enum
│   ├── provider.py              # OTEL MeterProvider setup
│   ├── metrics.py               # Metric instruments (Meters)
│   ├── grpc_instrumentation.py  # OTEL gRPC server instrumentation
│   ├── decorators.py            # @timed, @counted decorators
│   ├── context.py               # Context propagation helpers
│   └── logging.py               # Structured logging with OTEL correlation
```

#### Core Components

**config.py**
```python
from dataclasses import dataclass, field
from enum import IntEnum
import os

class ObservabilityLevel(IntEnum):
    DISABLED = 0
    MINIMAL = 1
    STANDARD = 2
    VERBOSE = 3
    DEBUG = 4

@dataclass
class ObservabilityConfig:
    level: ObservabilityLevel = ObservabilityLevel.STANDARD
    metrics_enabled: bool = True
    traces_enabled: bool = False

    # Prometheus exporter (pull)
    prometheus_port: int = 9090
    prometheus_host: str = "0.0.0.0"

    # OTLP exporter (push, optional)
    otlp_endpoint: str | None = None
    otlp_protocol: str = "grpc"  # grpc | http/protobuf

    # Service identification
    service_name: str = "claude-sidecar"
    service_version: str = "0.1.0"

    # Resource attributes
    resource_attributes: dict[str, str] = field(default_factory=dict)

    @classmethod
    def from_env(cls) -> "ObservabilityConfig":
        level_str = os.getenv("OTEL_OBSERVABILITY_LEVEL", "standard").upper()
        level = getattr(ObservabilityLevel, level_str, ObservabilityLevel.STANDARD)

        return cls(
            level=level,
            metrics_enabled=os.getenv("OTEL_METRICS_ENABLED", "true").lower() == "true",
            traces_enabled=os.getenv("OTEL_TRACES_ENABLED", "false").lower() == "true",
            prometheus_port=int(os.getenv("OTEL_EXPORTER_PROMETHEUS_PORT", "9090")),
            prometheus_host=os.getenv("OTEL_EXPORTER_PROMETHEUS_HOST", "0.0.0.0"),
            otlp_endpoint=os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
            otlp_protocol=os.getenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"),
            service_name=os.getenv("OTEL_SERVICE_NAME", "claude-sidecar"),
            service_version=os.getenv("OTEL_SERVICE_VERSION", "0.1.0"),
        )
```

**provider.py**
```python
"""OpenTelemetry provider setup."""

from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.prometheus import PrometheusMetricReader
from opentelemetry.sdk.resources import Resource, SERVICE_NAME, SERVICE_VERSION
from prometheus_client import start_http_server

from .config import ObservabilityConfig, ObservabilityLevel

_provider: MeterProvider | None = None


def initialize_telemetry(config: ObservabilityConfig) -> MeterProvider | None:
    """Initialize OpenTelemetry with Prometheus exporter."""
    global _provider

    if not config.metrics_enabled or config.level == ObservabilityLevel.DISABLED:
        return None

    if _provider is not None:
        return _provider

    # Create resource with service info
    resource = Resource.create({
        SERVICE_NAME: config.service_name,
        SERVICE_VERSION: config.service_version,
        "deployment.environment": os.getenv("OTEL_DEPLOYMENT_ENVIRONMENT", "development"),
        **config.resource_attributes,
    })

    # Setup Prometheus exporter (pull-based)
    # PrometheusMetricReader starts its own HTTP server
    prometheus_reader = PrometheusMetricReader()

    readers = [prometheus_reader]

    # Optionally add OTLP exporter (push-based)
    if config.otlp_endpoint:
        from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter

        otlp_exporter = OTLPMetricExporter(endpoint=config.otlp_endpoint)
        otlp_reader = PeriodicExportingMetricReader(
            otlp_exporter,
            export_interval_millis=60000,
        )
        readers.append(otlp_reader)

    # Create and set meter provider
    _provider = MeterProvider(resource=resource, metric_readers=readers)
    metrics.set_meter_provider(_provider)

    # Start Prometheus HTTP server
    start_http_server(config.prometheus_port, addr=config.prometheus_host)

    return _provider


def get_meter(name: str, version: str = "") -> metrics.Meter:
    """Get a meter for creating instruments."""
    provider = metrics.get_meter_provider()
    return provider.get_meter(name, version)


def shutdown_telemetry() -> None:
    """Shutdown OpenTelemetry providers."""
    global _provider
    if _provider:
        _provider.shutdown()
        _provider = None
```

**metrics.py**
```python
"""Metric instruments for the sidecar."""

from opentelemetry import metrics
from opentelemetry.metrics import Counter, Histogram, UpDownCounter, ObservableGauge

from .config import ObservabilityConfig, ObservabilityLevel
from .provider import get_meter

# Histogram bucket boundaries
LATENCY_BUCKETS = [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
CALLBACK_BUCKETS = [0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0]


class SidecarMetrics:
    """Container for all sidecar metric instruments."""

    def __init__(self, config: ObservabilityConfig):
        self.config = config
        self._meter = get_meter("claude.sidecar", config.service_version)
        self._instruments: dict[str, Counter | Histogram | UpDownCounter] = {}

        if config.metrics_enabled and config.level >= ObservabilityLevel.MINIMAL:
            self._register_instruments()

    def _register_instruments(self) -> None:
        level = self.config.level

        # === MINIMAL LEVEL ===
        if level >= ObservabilityLevel.MINIMAL:
            self._instruments["sessions_active"] = self._meter.create_up_down_counter(
                name="claude.sidecar.sessions",
                description="Number of active sessions",
                unit="{sessions}",
            )
            self._instruments["sessions_total"] = self._meter.create_counter(
                name="claude.sidecar.sessions.total",
                description="Total sessions created",
                unit="{sessions}",
            )
            self._instruments["errors"] = self._meter.create_counter(
                name="claude.sidecar.errors",
                description="Total errors",
                unit="{errors}",
            )

        # === STANDARD LEVEL ===
        if level >= ObservabilityLevel.STANDARD:
            self._instruments["session_duration"] = self._meter.create_histogram(
                name="claude.sidecar.session.duration",
                description="Session duration",
                unit="s",
            )
            self._instruments["session_turns"] = self._meter.create_counter(
                name="claude.sidecar.session.turns",
                description="Total turns processed",
                unit="{turns}",
            )
            self._instruments["callback_requests"] = self._meter.create_counter(
                name="claude.sidecar.callback.requests",
                description="Total callback requests",
                unit="{requests}",
            )
            self._instruments["callback_duration"] = self._meter.create_histogram(
                name="claude.sidecar.callback.duration",
                description="Callback duration",
                unit="s",
            )
            self._instruments["callback_timeouts"] = self._meter.create_counter(
                name="claude.sidecar.callback.timeouts",
                description="Total callback timeouts",
                unit="{timeouts}",
            )

        # === VERBOSE LEVEL ===
        if level >= ObservabilityLevel.VERBOSE:
            self._instruments["tool_invocations"] = self._meter.create_counter(
                name="claude.sidecar.tool.invocations",
                description="Tool invocations",
                unit="{invocations}",
            )
            self._instruments["tool_duration"] = self._meter.create_histogram(
                name="claude.sidecar.tool.duration",
                description="Tool invocation duration",
                unit="s",
            )
            self._instruments["hook_invocations"] = self._meter.create_counter(
                name="claude.sidecar.hook.invocations",
                description="Hook invocations",
                unit="{invocations}",
            )
            self._instruments["hook_duration"] = self._meter.create_histogram(
                name="claude.sidecar.hook.duration",
                description="Hook invocation duration",
                unit="s",
            )
            self._instruments["messages"] = self._meter.create_counter(
                name="claude.sidecar.messages",
                description="Messages processed",
                unit="{messages}",
            )
            self._instruments["sdk_query_duration"] = self._meter.create_histogram(
                name="claude.sidecar.sdk.query.duration",
                description="SDK query duration",
                unit="s",
            )

    # --- Recording methods with level checks ---

    def inc_sessions_active(self) -> None:
        if inst := self._instruments.get("sessions_active"):
            inst.add(1)

    def dec_sessions_active(self) -> None:
        if inst := self._instruments.get("sessions_active"):
            inst.add(-1)

    def inc_sessions_total(self, status: str) -> None:
        if inst := self._instruments.get("sessions_total"):
            inst.add(1, {"status": status})

    def record_error(self, error_code: str) -> None:
        if inst := self._instruments.get("errors"):
            inst.add(1, {"error.code": error_code})

    def record_session_duration(self, duration_seconds: float, status: str) -> None:
        if inst := self._instruments.get("session_duration"):
            inst.record(duration_seconds, {"status": status})

    def inc_session_turns(self, status: str) -> None:
        if inst := self._instruments.get("session_turns"):
            inst.add(1, {"status": status})

    def record_callback(
        self,
        callback_type: str,
        duration_seconds: float,
        timed_out: bool = False,
        tool_fqn: str | None = None,
        hook_event: str | None = None,
    ) -> None:
        # Standard level: aggregate by type
        if inst := self._instruments.get("callback_requests"):
            inst.add(1, {"callback.type": callback_type})

        if inst := self._instruments.get("callback_duration"):
            inst.record(duration_seconds, {"callback.type": callback_type})

        if timed_out:
            if inst := self._instruments.get("callback_timeouts"):
                inst.add(1, {"callback.type": callback_type})

        # Verbose level: per-tool/hook
        if self.config.level >= ObservabilityLevel.VERBOSE:
            if callback_type == "tool" and tool_fqn:
                if inst := self._instruments.get("tool_invocations"):
                    inst.add(1, {"tool.fqn": tool_fqn})
                if inst := self._instruments.get("tool_duration"):
                    inst.record(duration_seconds, {"tool.fqn": tool_fqn})

            elif callback_type == "hook" and hook_event:
                if inst := self._instruments.get("hook_invocations"):
                    inst.add(1, {"hook.event": hook_event})
                if inst := self._instruments.get("hook_duration"):
                    inst.record(duration_seconds, {"hook.event": hook_event})

    def record_message(self, message_type: str) -> None:
        if inst := self._instruments.get("messages"):
            inst.add(1, {"message.type": message_type})

    def record_sdk_query_duration(self, duration_seconds: float) -> None:
        if inst := self._instruments.get("sdk_query_duration"):
            inst.record(duration_seconds)


# Global metrics instance
_metrics: SidecarMetrics | None = None


def initialize_metrics(config: ObservabilityConfig) -> SidecarMetrics:
    global _metrics
    _metrics = SidecarMetrics(config)
    return _metrics


def get_metrics() -> SidecarMetrics | None:
    return _metrics
```

**grpc_instrumentation.py**
```python
"""OpenTelemetry gRPC server instrumentation."""

from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer

from .config import ObservabilityConfig, ObservabilityLevel

_instrumented = False


def instrument_grpc_server(config: ObservabilityConfig) -> None:
    """Instrument gRPC server with OpenTelemetry.

    This automatically captures:
    - rpc.server.duration
    - rpc.server.request.size
    - rpc.server.response.size
    - Trace propagation (if tracing enabled)
    """
    global _instrumented

    if _instrumented:
        return

    if not config.metrics_enabled or config.level < ObservabilityLevel.STANDARD:
        return

    GrpcInstrumentorServer().instrument()
    _instrumented = True


def uninstrument_grpc_server() -> None:
    """Remove gRPC server instrumentation."""
    global _instrumented
    if _instrumented:
        GrpcInstrumentorServer().uninstrument()
        _instrumented = False
```

**decorators.py**
```python
"""Instrumentation decorators using OTEL."""

import functools
import time
from typing import Callable, TypeVar, ParamSpec
from contextlib import contextmanager

from .config import ObservabilityLevel
from .metrics import get_metrics

P = ParamSpec("P")
T = TypeVar("T")


@contextmanager
def timed_callback(callback_type: str, tool_fqn: str | None = None, hook_event: str | None = None):
    """Context manager for timing callback execution."""
    metrics = get_metrics()
    start = time.perf_counter()
    timed_out = False

    try:
        yield
    except TimeoutError:
        timed_out = True
        raise
    finally:
        if metrics:
            duration = time.perf_counter() - start
            metrics.record_callback(
                callback_type=callback_type,
                duration_seconds=duration,
                timed_out=timed_out,
                tool_fqn=tool_fqn,
                hook_event=hook_event,
            )


def timed_operation(
    metric_method: str,
    min_level: ObservabilityLevel = ObservabilityLevel.STANDARD,
):
    """Decorator to time operations and record to a histogram.

    Args:
        metric_method: Name of the method on SidecarMetrics to call (e.g., "record_sdk_query_duration")
        min_level: Minimum observability level required
    """
    def decorator(func: Callable[P, T]) -> Callable[P, T]:
        @functools.wraps(func)
        async def async_wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
            metrics = get_metrics()
            if not metrics or metrics.config.level < min_level:
                return await func(*args, **kwargs)

            start = time.perf_counter()
            try:
                return await func(*args, **kwargs)
            finally:
                duration = time.perf_counter() - start
                record_fn = getattr(metrics, metric_method, None)
                if record_fn:
                    record_fn(duration)

        @functools.wraps(func)
        def sync_wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
            metrics = get_metrics()
            if not metrics or metrics.config.level < min_level:
                return func(*args, **kwargs)

            start = time.perf_counter()
            try:
                return func(*args, **kwargs)
            finally:
                duration = time.perf_counter() - start
                record_fn = getattr(metrics, metric_method, None)
                if record_fn:
                    record_fn(duration)

        import asyncio
        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        return sync_wrapper

    return decorator
```

#### Integration Points (Minimal Changes)

**serve.py** (add ~15 lines):
```python
from .observability import (
    ObservabilityConfig,
    initialize_telemetry,
    initialize_metrics,
    instrument_grpc_server,
    shutdown_telemetry,
)

async def serve(host: str, port: int) -> None:
    # Initialize observability
    config = ObservabilityConfig.from_env()
    initialize_telemetry(config)
    initialize_metrics(config)
    instrument_grpc_server(config)

    server = grpc.aio.server()
    pb_grpc.add_ClaudeSidecarServicer_to_server(ClaudeSidecarService(), server)
    server.add_insecure_port(f"{host}:{port}")

    await server.start()
    try:
        await server.wait_for_termination()
    finally:
        shutdown_telemetry()
```

**callback_router.py** (add timing):
```python
from .observability.decorators import timed_callback
from .observability.metrics import get_metrics

class CallbackRouter:
    async def request_tool(
        self,
        tool_fqn: str,
        tool_input: dict[str, Any],
        tool_use_id: str | None,
        timeout: float = 60.0,
    ) -> pb.ToolInvocationResponse:
        with timed_callback("tool", tool_fqn=tool_fqn):
            # ... existing implementation ...
            try:
                return await asyncio.wait_for(future, timeout=timeout)
            except asyncio.TimeoutError:
                raise TimeoutError("Tool callback timed out")
```

**sessions.py** (add session metrics):
```python
from .observability.metrics import get_metrics

class Session:
    def __init__(self, request: pb.CreateSessionRequest) -> None:
        # ... existing init ...
        if metrics := get_metrics():
            metrics.inc_sessions_active()
            metrics.inc_sessions_total("created")

    async def close(self, reason: str) -> None:
        if self.closed:
            return
        self.closed = True
        if metrics := get_metrics():
            metrics.dec_sessions_active()
        # ... rest of existing close logic ...
```

---

### Go Client Implementation

#### File Structure
```
go/client/
├── observability/
│   ├── config.go                # Config, FromEnv(), Level enum
│   ├── provider.go              # OTEL MeterProvider + Prometheus exporter
│   ├── metrics.go               # Metric instruments
│   ├── grpc.go                  # OTEL gRPC instrumentation
│   └── logging.go               # slog with OTEL correlation
```

#### Core Components

**config.go**
```go
package observability

import (
    "os"
    "strconv"
    "strings"
)

type Level int

const (
    LevelDisabled Level = iota
    LevelMinimal
    LevelStandard
    LevelVerbose
    LevelDebug
)

type Config struct {
    Level          Level
    MetricsEnabled bool
    TracesEnabled  bool

    // Prometheus exporter
    PrometheusPort int
    PrometheusHost string

    // OTLP exporter (optional)
    OTLPEndpoint string
    OTLPProtocol string // "grpc" | "http/protobuf"

    // Service identification
    ServiceName    string
    ServiceVersion string
}

func DefaultConfig() *Config {
    return &Config{
        Level:          LevelStandard,
        MetricsEnabled: true,
        TracesEnabled:  false,
        PrometheusPort: 9091,
        PrometheusHost: "0.0.0.0",
        OTLPProtocol:   "grpc",
        ServiceName:    "claude-client-go",
        ServiceVersion: "0.1.0",
    }
}

func ConfigFromEnv() *Config {
    config := DefaultConfig()

    if v := os.Getenv("OTEL_OBSERVABILITY_LEVEL"); v != "" {
        config.Level = parseLevel(v)
    }
    if v := os.Getenv("OTEL_METRICS_ENABLED"); v != "" {
        config.MetricsEnabled = strings.ToLower(v) == "true"
    }
    if v := os.Getenv("OTEL_TRACES_ENABLED"); v != "" {
        config.TracesEnabled = strings.ToLower(v) == "true"
    }
    if v := os.Getenv("OTEL_EXPORTER_PROMETHEUS_PORT"); v != "" {
        if port, err := strconv.Atoi(v); err == nil {
            config.PrometheusPort = port
        }
    }
    if v := os.Getenv("OTEL_EXPORTER_PROMETHEUS_HOST"); v != "" {
        config.PrometheusHost = v
    }
    if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
        config.OTLPEndpoint = v
    }
    if v := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); v != "" {
        config.OTLPProtocol = v
    }
    if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
        config.ServiceName = v
    }
    if v := os.Getenv("OTEL_SERVICE_VERSION"); v != "" {
        config.ServiceVersion = v
    }

    return config
}

func parseLevel(s string) Level {
    switch strings.ToLower(s) {
    case "disabled":
        return LevelDisabled
    case "minimal":
        return LevelMinimal
    case "standard":
        return LevelStandard
    case "verbose":
        return LevelVerbose
    case "debug":
        return LevelDebug
    default:
        return LevelStandard
    }
}
```

**provider.go**
```go
package observability

import (
    "context"
    "fmt"
    "net/http"
    "sync"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
    prom "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    meterProvider *metric.MeterProvider
    initOnce      sync.Once
    shutdownOnce  sync.Once
)

// InitializeTelemetry sets up OpenTelemetry with Prometheus exporter.
func InitializeTelemetry(ctx context.Context, config *Config) (*metric.MeterProvider, error) {
    var initErr error

    initOnce.Do(func() {
        if !config.MetricsEnabled || config.Level == LevelDisabled {
            return
        }

        // Create resource
        res, err := resource.Merge(
            resource.Default(),
            resource.NewWithAttributes(
                semconv.SchemaURL,
                semconv.ServiceName(config.ServiceName),
                semconv.ServiceVersion(config.ServiceVersion),
            ),
        )
        if err != nil {
            initErr = fmt.Errorf("failed to create resource: %w", err)
            return
        }

        // Create Prometheus exporter
        registry := prom.NewRegistry()
        promExporter, err := prometheus.New(
            prometheus.WithRegisterer(registry),
        )
        if err != nil {
            initErr = fmt.Errorf("failed to create prometheus exporter: %w", err)
            return
        }

        // Create meter provider
        meterProvider = metric.NewMeterProvider(
            metric.WithResource(res),
            metric.WithReader(promExporter),
        )
        otel.SetMeterProvider(meterProvider)

        // Start Prometheus HTTP server
        go func() {
            mux := http.NewServeMux()
            mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
            addr := fmt.Sprintf("%s:%d", config.PrometheusHost, config.PrometheusPort)
            http.ListenAndServe(addr, mux)
        }()
    })

    return meterProvider, initErr
}

// ShutdownTelemetry gracefully shuts down the meter provider.
func ShutdownTelemetry(ctx context.Context) error {
    var shutdownErr error
    shutdownOnce.Do(func() {
        if meterProvider != nil {
            shutdownErr = meterProvider.Shutdown(ctx)
        }
    })
    return shutdownErr
}

// GetMeterProvider returns the global meter provider.
func GetMeterProvider() *metric.MeterProvider {
    return meterProvider
}
```

**metrics.go**
```go
package observability

import (
    "context"
    "sync"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// ClientMetrics holds all client metric instruments.
type ClientMetrics struct {
    config *Config
    meter  metric.Meter

    // Minimal level
    connectionsTotal  metric.Int64Counter
    connectionsActive metric.Int64UpDownCounter

    // Standard level
    handlerCalls    metric.Int64Counter
    handlerDuration metric.Float64Histogram
    turnsTotal      metric.Int64Counter
    turnDuration    metric.Float64Histogram

    // Verbose level
    streamEvents        metric.Int64Counter
    streamDuration      metric.Float64Histogram
    eventQueueDepth     metric.Int64ObservableGauge
    subscriptionsActive metric.Int64UpDownCounter
    toolHandlerDuration metric.Float64Histogram
    hookHandlerDuration metric.Float64Histogram

    // For observable gauges
    queueDepthValue int64
    queueDepthMu    sync.Mutex
}

var (
    globalMetrics *ClientMetrics
    metricsOnce   sync.Once
)

// InitializeMetrics creates and registers all metric instruments.
func InitializeMetrics(config *Config) (*ClientMetrics, error) {
    var initErr error

    metricsOnce.Do(func() {
        if !config.MetricsEnabled || config.Level == LevelDisabled {
            globalMetrics = &ClientMetrics{config: config}
            return
        }

        m := &ClientMetrics{
            config: config,
            meter:  otel.Meter("claude.client", metric.WithInstrumentationVersion(config.ServiceVersion)),
        }

        var err error

        // === MINIMAL LEVEL ===
        if config.Level >= LevelMinimal {
            m.connectionsTotal, err = m.meter.Int64Counter(
                "claude.client.connections",
                metric.WithDescription("Total connection attempts"),
                metric.WithUnit("{connections}"),
            )
            if err != nil {
                initErr = err
                return
            }

            m.connectionsActive, err = m.meter.Int64UpDownCounter(
                "claude.client.connections.active",
                metric.WithDescription("Active connections"),
                metric.WithUnit("{connections}"),
            )
            if err != nil {
                initErr = err
                return
            }
        }

        // === STANDARD LEVEL ===
        if config.Level >= LevelStandard {
            m.handlerCalls, err = m.meter.Int64Counter(
                "claude.client.handler.calls",
                metric.WithDescription("Handler calls"),
                metric.WithUnit("{calls}"),
            )
            if err != nil {
                initErr = err
                return
            }

            m.handlerDuration, err = m.meter.Float64Histogram(
                "claude.client.handler.duration",
                metric.WithDescription("Handler duration"),
                metric.WithUnit("s"),
                metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30),
            )
            if err != nil {
                initErr = err
                return
            }

            m.turnsTotal, err = m.meter.Int64Counter(
                "claude.client.turns",
                metric.WithDescription("Total turns"),
                metric.WithUnit("{turns}"),
            )
            if err != nil {
                initErr = err
                return
            }

            m.turnDuration, err = m.meter.Float64Histogram(
                "claude.client.turn.duration",
                metric.WithDescription("Turn duration"),
                metric.WithUnit("s"),
                metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120),
            )
            if err != nil {
                initErr = err
                return
            }
        }

        // === VERBOSE LEVEL ===
        if config.Level >= LevelVerbose {
            m.streamEvents, err = m.meter.Int64Counter(
                "claude.client.stream.events",
                metric.WithDescription("Stream events"),
                metric.WithUnit("{events}"),
            )
            if err != nil {
                initErr = err
                return
            }

            m.streamDuration, err = m.meter.Float64Histogram(
                "claude.client.stream.duration",
                metric.WithDescription("Stream duration"),
                metric.WithUnit("s"),
            )
            if err != nil {
                initErr = err
                return
            }

            m.subscriptionsActive, err = m.meter.Int64UpDownCounter(
                "claude.client.subscriptions.active",
                metric.WithDescription("Active subscriptions"),
                metric.WithUnit("{subscriptions}"),
            )
            if err != nil {
                initErr = err
                return
            }

            // Observable gauge for queue depth
            m.eventQueueDepth, err = m.meter.Int64ObservableGauge(
                "claude.client.event_queue.depth",
                metric.WithDescription("Event queue depth"),
                metric.WithUnit("{events}"),
                metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
                    m.queueDepthMu.Lock()
                    defer m.queueDepthMu.Unlock()
                    o.Observe(m.queueDepthValue)
                    return nil
                }),
            )
            if err != nil {
                initErr = err
                return
            }
        }

        globalMetrics = m
    })

    return globalMetrics, initErr
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *ClientMetrics {
    return globalMetrics
}

// --- Recording methods ---

func (m *ClientMetrics) RecordConnection(ctx context.Context, status string) {
    if m.connectionsTotal != nil {
        m.connectionsTotal.Add(ctx, 1, metric.WithAttributes(
            attribute.String("status", status),
        ))
    }
}

func (m *ClientMetrics) IncConnectionsActive(ctx context.Context) {
    if m.connectionsActive != nil {
        m.connectionsActive.Add(ctx, 1)
    }
}

func (m *ClientMetrics) DecConnectionsActive(ctx context.Context) {
    if m.connectionsActive != nil {
        m.connectionsActive.Add(ctx, -1)
    }
}

func (m *ClientMetrics) RecordHandlerCall(ctx context.Context, handlerType, status string, duration time.Duration) {
    if m.handlerCalls != nil {
        m.handlerCalls.Add(ctx, 1, metric.WithAttributes(
            attribute.String("handler.type", handlerType),
            attribute.String("status", status),
        ))
    }
    if m.handlerDuration != nil {
        m.handlerDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
            attribute.String("handler.type", handlerType),
        ))
    }
}

func (m *ClientMetrics) RecordTurn(ctx context.Context, status string, duration time.Duration) {
    if m.turnsTotal != nil {
        m.turnsTotal.Add(ctx, 1, metric.WithAttributes(
            attribute.String("status", status),
        ))
    }
    if m.turnDuration != nil {
        m.turnDuration.Record(ctx, duration.Seconds())
    }
}

func (m *ClientMetrics) RecordStreamEvent(ctx context.Context, eventType string) {
    if m.streamEvents != nil {
        m.streamEvents.Add(ctx, 1, metric.WithAttributes(
            attribute.String("event.type", eventType),
        ))
    }
}

func (m *ClientMetrics) RecordStreamDuration(ctx context.Context, duration time.Duration) {
    if m.streamDuration != nil {
        m.streamDuration.Record(ctx, duration.Seconds())
    }
}

func (m *ClientMetrics) SetEventQueueDepth(depth int) {
    m.queueDepthMu.Lock()
    m.queueDepthValue = int64(depth)
    m.queueDepthMu.Unlock()
}

func (m *ClientMetrics) IncSubscriptionsActive(ctx context.Context) {
    if m.subscriptionsActive != nil {
        m.subscriptionsActive.Add(ctx, 1)
    }
}

func (m *ClientMetrics) DecSubscriptionsActive(ctx context.Context) {
    if m.subscriptionsActive != nil {
        m.subscriptionsActive.Add(ctx, -1)
    }
}
```

**grpc.go**
```go
package observability

import (
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
    "google.golang.org/grpc"
)

// GRPCDialOptions returns gRPC dial options for OTEL instrumentation.
// This automatically captures:
// - rpc.client.duration
// - rpc.client.request.size
// - rpc.client.response.size
// - Trace propagation (if tracing enabled)
func GRPCDialOptions(config *Config) []grpc.DialOption {
    if !config.MetricsEnabled || config.Level < LevelStandard {
        return nil
    }

    return []grpc.DialOption{
        grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
    }
}
```

#### Integration Points

**client.go** (add ~20 lines):
```go
package client

import (
    "context"

    "github.com/dgarson/claude-sidecar/client/observability"
    "google.golang.org/grpc"
)

// ClientOption configures the client.
type ClientOption func(*clientOptions)

type clientOptions struct {
    observabilityConfig *observability.Config
}

// WithObservability configures observability for the client.
func WithObservability(config *observability.Config) ClientOption {
    return func(o *clientOptions) {
        o.observabilityConfig = config
    }
}

func Dial(ctx context.Context, addr string, opts ...interface{}) (*Client, error) {
    // Separate client options from gRPC options
    var clientOpts clientOptions
    var grpcOpts []grpc.DialOption

    for _, opt := range opts {
        switch o := opt.(type) {
        case ClientOption:
            o(&clientOpts)
        case grpc.DialOption:
            grpcOpts = append(grpcOpts, o)
        }
    }

    // Initialize observability
    obsConfig := clientOpts.observabilityConfig
    if obsConfig == nil {
        obsConfig = observability.ConfigFromEnv()
    }

    if _, err := observability.InitializeTelemetry(ctx, obsConfig); err != nil {
        return nil, err
    }
    if _, err := observability.InitializeMetrics(obsConfig); err != nil {
        return nil, err
    }

    // Add OTEL gRPC instrumentation
    grpcOpts = append(grpcOpts, observability.GRPCDialOptions(obsConfig)...)

    // Default credentials if none provided
    if len(grpcOpts) == 0 {
        grpcOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
    }

    conn, err := grpc.DialContext(ctx, addr, grpcOpts...)
    if err != nil {
        if metrics := observability.GetMetrics(); metrics != nil {
            metrics.RecordConnection(ctx, "error")
        }
        return nil, err
    }

    if metrics := observability.GetMetrics(); metrics != nil {
        metrics.RecordConnection(ctx, "success")
        metrics.IncConnectionsActive(ctx)
    }

    return &Client{conn: conn, api: pb.NewClaudeSidecarClient(conn)}, nil
}

func (c *Client) Close() error {
    if metrics := observability.GetMetrics(); metrics != nil {
        metrics.DecConnectionsActive(context.Background())
    }
    return c.conn.Close()
}
```

**Handler wrapper** (instrumented handlers):
```go
package client

import (
    "context"
    "time"

    "github.com/dgarson/claude-sidecar/client/observability"
)

// InstrumentHandlers wraps handlers with metrics instrumentation.
func InstrumentHandlers(handlers Handlers) Handlers {
    return Handlers{
        Tool: func(ctx context.Context, req *pb.ToolInvocationRequest) (*structpb.Struct, error) {
            if handlers.Tool == nil {
                return nil, nil
            }

            start := time.Now()
            result, err := handlers.Tool(ctx, req)

            if metrics := observability.GetMetrics(); metrics != nil {
                status := "ok"
                if err != nil {
                    status = "error"
                }
                metrics.RecordHandlerCall(ctx, "tool", status, time.Since(start))
            }

            return result, err
        },
        Hook: func(ctx context.Context, req *pb.HookInvocationRequest) (*pb.HookOutput, error) {
            if handlers.Hook == nil {
                return nil, nil
            }

            start := time.Now()
            result, err := handlers.Hook(ctx, req)

            if metrics := observability.GetMetrics(); metrics != nil {
                status := "ok"
                if err != nil {
                    status = "error"
                }
                metrics.RecordHandlerCall(ctx, "hook", status, time.Since(start))
            }

            return result, err
        },
        Permission: func(ctx context.Context, req *pb.PermissionDecisionRequest) (*pb.PermissionDecision, error) {
            if handlers.Permission == nil {
                return nil, nil
            }

            start := time.Now()
            result, err := handlers.Permission(ctx, req)

            if metrics := observability.GetMetrics(); metrics != nil {
                status := "ok"
                if err != nil {
                    status = "error"
                }
                metrics.RecordHandlerCall(ctx, "permission", status, time.Since(start))
            }

            return result, err
        },
    }
}
```

---

## Tracing Integration (OpenTelemetry)

Since we're using OTEL, adding tracing is straightforward - the same SDK handles both metrics and traces.

### Python

```python
# observability/tracing.py
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace.sampling import ParentBasedTraceIdRatio

from .config import ObservabilityConfig


def initialize_tracing(config: ObservabilityConfig) -> None:
    """Initialize OpenTelemetry tracing."""
    if not config.traces_enabled:
        return

    # Use same resource as metrics
    from .provider import _create_resource
    resource = _create_resource(config)

    # Configure sampling
    sampler = ParentBasedTraceIdRatio(
        float(os.getenv("OTEL_TRACES_SAMPLER_ARG", "0.1"))
    )

    provider = TracerProvider(resource=resource, sampler=sampler)

    if config.otlp_endpoint:
        exporter = OTLPSpanExporter(endpoint=config.otlp_endpoint)
        provider.add_span_processor(BatchSpanProcessor(exporter))

    trace.set_tracer_provider(provider)

    # gRPC instrumentation automatically propagates traces
    # (already done in grpc_instrumentation.py)
```

### Go

```go
// observability/tracing.go
package observability

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
)

func InitializeTracing(ctx context.Context, config *Config) error {
    if !config.TracesEnabled || config.OTLPEndpoint == "" {
        return nil
    }

    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return err
    }

    // Use same resource as metrics
    res := createResource(config)

    provider := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
        trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(0.1))),
    )

    otel.SetTracerProvider(provider)
    return nil
}
```

---

## Logging with OTEL Correlation

### Python (structlog + OTEL)

```python
# observability/logging.py
import logging
import structlog
from opentelemetry import trace

from .config import ObservabilityConfig


def configure_logging(config: ObservabilityConfig) -> None:
    """Configure structured logging with OTEL trace correlation."""

    def add_otel_context(logger, method_name, event_dict):
        """Add trace/span IDs to log records."""
        span = trace.get_current_span()
        if span.is_recording():
            ctx = span.get_span_context()
            event_dict["trace_id"] = format(ctx.trace_id, "032x")
            event_dict["span_id"] = format(ctx.span_id, "016x")
        return event_dict

    processors = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        add_otel_context,
    ]

    if config.log_format == "json":
        processors.append(structlog.processors.JSONRenderer())
    else:
        processors.append(structlog.dev.ConsoleRenderer())

    structlog.configure(
        processors=processors,
        wrapper_class=structlog.make_filtering_bound_logger(
            getattr(logging, config.log_level.upper(), logging.INFO)
        ),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
    )


def get_logger(name: str) -> structlog.BoundLogger:
    """Get a logger with the given name."""
    return structlog.get_logger(name)
```

### Go (slog + OTEL)

```go
// observability/logging.go
package observability

import (
    "context"
    "log/slog"
    "os"

    "go.opentelemetry.io/otel/trace"
)

// OTELHandler wraps a slog.Handler to add trace context.
type OTELHandler struct {
    slog.Handler
}

func (h *OTELHandler) Handle(ctx context.Context, r slog.Record) error {
    span := trace.SpanFromContext(ctx)
    if span.IsRecording() {
        spanCtx := span.SpanContext()
        r.AddAttrs(
            slog.String("trace_id", spanCtx.TraceID().String()),
            slog.String("span_id", spanCtx.SpanID().String()),
        )
    }
    return h.Handler.Handle(ctx, r)
}

func ConfigureLogging(config *Config) *slog.Logger {
    level := parseLogLevel(config.LogLevel)
    opts := &slog.HandlerOptions{Level: level}

    var handler slog.Handler
    if config.LogFormat == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    // Wrap with OTEL correlation
    handler = &OTELHandler{Handler: handler}

    return slog.New(handler)
}

func parseLogLevel(s string) slog.Level {
    switch strings.ToUpper(s) {
    case "DEBUG":
        return slog.LevelDebug
    case "INFO":
        return slog.LevelInfo
    case "WARN":
        return slog.LevelWarn
    case "ERROR":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

---

## Dependencies

### Python
```toml
# pyproject.toml
[project.optional-dependencies]
observability = [
    # Core OTEL
    "opentelemetry-api>=1.22.0",
    "opentelemetry-sdk>=1.22.0",

    # Exporters
    "opentelemetry-exporter-prometheus>=0.43b0",
    "opentelemetry-exporter-otlp>=1.22.0",

    # Instrumentation
    "opentelemetry-instrumentation-grpc>=0.43b0",

    # Logging
    "structlog>=24.1.0",
]
```

### Go
```go
// go.mod
require (
    // Core OTEL
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/sdk v1.24.0
    go.opentelemetry.io/otel/metric v1.24.0
    go.opentelemetry.io/otel/sdk/metric v1.24.0

    // Prometheus exporter
    go.opentelemetry.io/otel/exporters/prometheus v0.46.0
    github.com/prometheus/client_golang v1.19.0

    // OTLP exporter (optional)
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.24.0

    // gRPC instrumentation
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
)
```

### Java (for future reference)
```xml
<!-- pom.xml -->
<dependencies>
    <!-- OTEL BOM -->
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-bom</artifactId>
        <version>1.35.0</version>
        <type>pom</type>
        <scope>import</scope>
    </dependency>

    <!-- Core -->
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-api</artifactId>
    </dependency>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-sdk</artifactId>
    </dependency>

    <!-- Prometheus exporter -->
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-exporter-prometheus</artifactId>
    </dependency>

    <!-- gRPC instrumentation -->
    <dependency>
        <groupId>io.opentelemetry.instrumentation</groupId>
        <artifactId>opentelemetry-grpc-1.6</artifactId>
    </dependency>
</dependencies>
```

---

## Grafana Dashboard Examples

### Sidecar Overview

```json
{
  "title": "Claude Sidecar Overview",
  "panels": [
    {
      "title": "Active Sessions",
      "type": "stat",
      "targets": [{"expr": "claude_sidecar_sessions{state=\"active\"}"}]
    },
    {
      "title": "RPC Request Rate",
      "type": "graph",
      "targets": [{"expr": "sum(rate(rpc_server_duration_milliseconds_count[5m])) by (rpc_method)"}]
    },
    {
      "title": "RPC Latency (p99)",
      "type": "graph",
      "targets": [{"expr": "histogram_quantile(0.99, sum(rate(rpc_server_duration_milliseconds_bucket[5m])) by (le, rpc_method))"}]
    },
    {
      "title": "Callback Timeout Rate",
      "type": "graph",
      "targets": [{"expr": "sum(rate(claude_sidecar_callback_timeouts_total[5m])) by (callback_type)"}]
    }
  ]
}
```

### Client Performance

```json
{
  "title": "Claude Client Performance",
  "panels": [
    {
      "title": "Active Connections",
      "type": "stat",
      "targets": [{"expr": "claude_client_connections_active"}]
    },
    {
      "title": "Turn Duration (p50/p99)",
      "type": "graph",
      "targets": [
        {"expr": "histogram_quantile(0.50, sum(rate(claude_client_turn_duration_seconds_bucket[5m])) by (le))"},
        {"expr": "histogram_quantile(0.99, sum(rate(claude_client_turn_duration_seconds_bucket[5m])) by (le))"}
      ]
    },
    {
      "title": "Handler Latency by Type",
      "type": "graph",
      "targets": [{"expr": "histogram_quantile(0.99, sum(rate(claude_client_handler_duration_seconds_bucket[5m])) by (le, handler_type))"}]
    }
  ]
}
```

---

## Summary

This design uses **OpenTelemetry as the unified instrumentation framework** with **Prometheus as the export format**:

1. **OTEL API**: Single, vendor-agnostic instrumentation API across all languages
2. **Prometheus exporter**: Pull-based metrics for cloud-native scraping
3. **OTLP exporter**: Optional push-based export to collectors/backends
4. **Automatic gRPC instrumentation**: Zero-code RPC metrics via OTEL instrumentation libraries
5. **Unified tracing**: Same SDK handles metrics and traces with correlation
6. **Structured logging**: Log correlation with trace/span IDs

Key benefits:
- **No direct Prometheus client usage** - OTEL abstracts this
- **Future-proof**: Easy to add new exporters (Datadog, Jaeger, etc.)
- **Consistent patterns** across Python, Go, Java, Rust, etc.
- **Configurable levels** to control cardinality and overhead

---

## Comparison: Full vs Simple

| Aspect | Simple Version | This Version |
|--------|----------------|--------------|
| **Lines of code** | ~80 per language | ~300+ per language |
| **Files** | 1 per language | 5-7 per language |
| **Config options** | 2 env vars | 10+ env vars |
| **Granularity levels** | On/Off | 5 levels (disabled → debug) |
| **Per-tool metrics** | No | Yes (at verbose+) |
| **Per-hook metrics** | No | Yes (at verbose+) |
| **Queue depth gauges** | No | Yes (at verbose+) |
| **Session/request ID labels** | No | Yes (at debug level) |
| **Decorator system** | Minimal | Elaborate |
| **Cardinality control** | None (just disable) | Fine-grained |
| **Time to implement** | Hours | Days |

### When to Use This Version

- You have **cardinality problems** in production and need to dial down without fully disabling
- You need **per-tool or per-hook latency breakdowns** to debug specific integrations
- You want **debug-level metrics** with session/request IDs for development
- You're building a **multi-tenant system** where different deployments need different verbosity

### When to Use the Simple Version

- You're **shipping an MVP** and need observability quickly
- Aggregate metrics (callback latency by type) are **sufficient for your use case**
- You prefer **less code to maintain**
- You can add per-tool metrics later with a 1-line change if needed

**See [observability-simple.md](./observability-simple.md) for the lightweight implementation.**
