# Claude Agent SDK Monitoring

Local monitoring stack for developing and testing Claude Agent SDK observability.

## Quick Start

```bash
cd monitoring
docker compose up -d
```

**Access:**
- **Grafana:** http://localhost:3000 (login: `admin` / `admin`)
- **Prometheus:** http://localhost:9095

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Host Machine                                 │
│                                                                  │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐    │
│  │   Sidecar    │     │  Go Client   │     │ Java Client  │    │
│  │  (Python)    │     │              │     │              │    │
│  │  :9090       │     │  :9091       │     │  :9092       │    │
│  └──────────────┘     └──────────────┘     └──────────────┘    │
│         │                   │                    │              │
└─────────┼───────────────────┼────────────────────┼──────────────┘
          │                   │                    │
          └───────────────────┼────────────────────┘
                              │
                    host.docker.internal
                              │
┌─────────────────────────────┼───────────────────────────────────┐
│                     Docker Network                               │
│                              │                                   │
│  ┌──────────────┐           │           ┌──────────────┐        │
│  │  Prometheus  │◄──────────┘           │   Grafana    │        │
│  │  :9095       │                       │   :3000      │        │
│  └──────────────┘                       └──────────────┘        │
│         │                                      │                 │
│         └──────────────────────────────────────┘                 │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Expected Metric Endpoints

This stack assumes Prometheus can scrape a Prometheus-format `/metrics` endpoint.

If you export metrics via OTLP to an OpenTelemetry Collector (recommended), configure Prometheus to scrape the Collector’s Prometheus exporter endpoint instead (commonly `:9464`), and use `make sidecar-metrics` for quick local dumps.

If you expose Prometheus metrics directly from a process, configure your services to use these ports:

| Service | Default Port | Environment Variable |
|---------|--------------|---------------------|
| Python Sidecar | 9090 | (if using an in-process Prometheus exporter) |
| Go Client | 9091 | (if using an in-process Prometheus exporter) |
| Java Client | 9092 | (customize in prometheus.yml) |

## Dashboards

Two pre-configured dashboards are automatically loaded:

### Claude Sidecar Overview
- Active sessions, request/error rates
- Callback latency (p50/p95/p99) by type
- gRPC latency by method
- Session lifecycle metrics
- Token usage by model

### Claude Client Overview
- Active connections
- Turn rates and error rates
- Handler latency by type
- gRPC client metrics (latency, message sizes)
- Error breakdown by gRPC status code

## Configuration

### Adding More Scrape Targets

Edit `prometheus/prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'my-new-client'
    static_configs:
      - targets: ['host.docker.internal:9093']
        labels:
          component: 'client'
          language: 'rust'
```

Then reload Prometheus:
```bash
curl -X POST http://localhost:9095/-/reload
```

### Linux Host Networking

On Linux, `host.docker.internal` may not work by default. The docker-compose.yml includes `extra_hosts` configuration, but if you still have issues:

**Option 1:** Use host's IP address directly in prometheus.yml
```yaml
- targets: ['192.168.1.100:9090']
```

**Option 2:** Use host network mode (loses network isolation)
```yaml
services:
  prometheus:
    network_mode: host
```

## Dashboard Variables

Both dashboards include template variables:

| Variable | Description |
|----------|-------------|
| `$datasource` | Prometheus datasource selector |
| `$job` | Filter by Prometheus job name |

## Metrics Reference

See [docs/observable-units-of-work.md](../docs/observable-units-of-work.md) for:
- Complete metrics catalog
- Label naming conventions
- Priority tiers (P0-P3)
- PromQL query examples

## Troubleshooting

### No data in Grafana

1. Check Prometheus targets: http://localhost:9095/targets
2. Verify your services are exposing metrics:
   ```bash
   curl http://localhost:9464/metrics  # OTEL Collector prometheus exporter (recommended)
   curl http://localhost:9090/metrics  # sidecar (if exporting Prometheus directly)
   curl http://localhost:9091/metrics  # go client (if exporting Prometheus directly)
   ```
3. Ensure `OTEL_METRICS_ENABLED=true` is set

### Connection refused to host.docker.internal

1. Verify service is running and bound to all interfaces (0.0.0.0), not just localhost
2. On Linux, check firewall rules for the metrics ports

### Dashboards not loading

1. Check Grafana logs: `docker compose logs grafana`
2. Verify dashboard JSON files in `dashboards/` directory
3. Check provisioning config in `grafana/provisioning/dashboards/dashboards.yml`

## Cleanup

```bash
# Stop services
docker compose down

# Remove all data (metrics history)
docker compose down -v
```
