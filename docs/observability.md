# Observability

GateKey provides comprehensive observability features including Prometheus metrics and OpenTelemetry distributed tracing for monitoring, alerting, and debugging.

## Prometheus Metrics

GateKey exposes Prometheus-compatible metrics at the `/metrics` endpoint for integration with monitoring systems.

### Configuration

Enable metrics in your server configuration:

```yaml
metrics:
  enabled: true
  path: "/metrics"
```

### Accessing Metrics

```bash
curl http://localhost:8080/metrics
```

### Available Metrics

#### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests processed |
| `gatekey_http_request_duration_seconds` | Histogram | `method`, `path` | Request latency distribution |
| `gatekey_http_requests_in_flight` | Gauge | - | Current requests being processed |

#### Authentication Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_auth_sessions_active` | Gauge | - | Number of active user sessions |
| `gatekey_auth_logins_total` | Counter | `provider`, `result` | Login attempts by provider and result |

#### PKI & Certificate Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_certificates_issued_total` | Counter | `type` | Total certificates issued |
| `gatekey_certificates_revoked_total` | Counter | - | Total certificates revoked |
| `gatekey_ca_expiry_seconds` | Gauge | - | Seconds until CA certificate expiration |

#### Gateway Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_gateway_connections_active` | Gauge | `gateway` | Active connections per gateway |
| `gatekey_gateway_heartbeats_total` | Counter | `gateway`, `status` | Gateway heartbeat events |

#### Database Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_db_connections_open` | Gauge | - | Total open database connections |
| `gatekey_db_connections_in_use` | Gauge | - | Database connections currently in use |
| `gatekey_db_connections_idle` | Gauge | - | Idle database connections |

#### Policy Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_policy_evaluations_total` | Counter | `result` | Policy evaluations (allowed/denied) |
| `gatekey_policy_evaluation_duration_seconds` | Histogram | - | Policy evaluation latency |

#### Server Info

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gatekey_server_info` | Gauge | `version`, `go_version` | Server version information (always 1) |

### Prometheus Scrape Configuration

Add GateKey to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'gatekey'
    static_configs:
      - targets: ['gatekey.example.com:8080']
    # Optional: for HTTPS with self-signed certificates
    scheme: https
    tls_config:
      insecure_skip_verify: true
```

### Kubernetes ServiceMonitor

For Prometheus Operator deployments, create a ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: gatekey-server
  labels:
    release: prometheus  # Must match your Prometheus selector
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: gatekey
      app.kubernetes.io/component: server
  namespaceSelector:
    matchNames:
      - gatekey
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      scrapeTimeout: 10s
```

### Example Alert Rules

```yaml
groups:
  - name: gatekey
    rules:
      - alert: GatekeyHighAuthFailureRate
        expr: rate(gatekey_auth_logins_total{result="failure"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High authentication failure rate
          description: "{{ $value | printf \"%.2f\" }} auth failures per second"

      - alert: GatekeyNoActiveGateways
        expr: gatekey_gateway_connections_active == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: No active gateway connections
          description: All gateways appear to be offline

      - alert: GatekeyHighLatency
        expr: histogram_quantile(0.95, rate(gatekey_http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High API latency (p95 > 1s)

      - alert: GatekeyDBPoolExhausted
        expr: gatekey_db_connections_in_use / gatekey_db_connections_open > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: Database connection pool nearly exhausted
```

## OpenTelemetry Tracing

GateKey supports distributed tracing via OpenTelemetry for end-to-end request visibility across your infrastructure.

### Configuration

Enable tracing in your server configuration:

```yaml
telemetry:
  enabled: true
  service_name: "gatekey"
  service_version: "1.7.0"  # Optional, auto-detected if not set
  environment: "production"
  otlp_endpoint: "otel-collector:4317"
  otlp_protocol: "grpc"     # or "http"
  otlp_insecure: false      # Set true for non-TLS connections
  sample_rate: 1.0          # 1.0 = 100%, 0.1 = 10%
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable OpenTelemetry tracing |
| `service_name` | string | `"gatekey"` | Service name in traces |
| `service_version` | string | auto | Service version (uses build version if not set) |
| `environment` | string | `"production"` | Deployment environment name |
| `otlp_endpoint` | string | required | OTLP collector endpoint (host:port) |
| `otlp_protocol` | string | `"grpc"` | Export protocol: `grpc` or `http` |
| `otlp_insecure` | bool | `false` | Skip TLS verification |
| `sample_rate` | float | `1.0` | Trace sampling rate (0.0 to 1.0) |

### Span Attributes

Each HTTP request span includes the following attributes:

| Attribute | Description |
|-----------|-------------|
| `http.method` | HTTP method (GET, POST, etc.) |
| `http.url` | Full request URL |
| `http.route` | Matched route pattern |
| `http.status_code` | Response status code |
| `http.response_content_length` | Response body size |
| `user_agent.original` | Client user agent string |
| `client.address` | Client IP address |
| `error` | Set to `true` for 4xx/5xx responses |

### Trace Context Propagation

GateKey automatically propagates trace context using W3C Trace Context headers:
- `traceparent` - Contains trace ID, span ID, and trace flags
- `tracestate` - Vendor-specific trace information

This enables distributed tracing across microservices that call GateKey APIs.

### Backend Integration

#### Grafana Tempo

Configure the OTEL collector to export to Tempo:

```yaml
exporters:
  otlp:
    endpoint: tempo.monitoring:4317
    tls:
      insecure: true
```

#### Jaeger

```yaml
exporters:
  jaeger:
    endpoint: jaeger-collector:14250
    tls:
      insecure: true
```

#### Cloud Providers

- **AWS X-Ray**: Use the AWS Distro for OpenTelemetry
- **Google Cloud Trace**: Use the Google Cloud exporter
- **Azure Monitor**: Use the Azure Monitor exporter
- **Datadog**: Use the Datadog exporter or OTLP ingestion

### Kubernetes Deployment Example

Deploy the OpenTelemetry Collector:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
    spec:
      containers:
        - name: collector
          image: otel/opentelemetry-collector-contrib:latest
          ports:
            - containerPort: 4317  # OTLP gRPC
            - containerPort: 4318  # OTLP HTTP
          volumeMounts:
            - name: config
              mountPath: /etc/otelcol
      volumes:
        - name: config
          configMap:
            name: otel-collector-config
---
apiVersion: v1
kind: Service
metadata:
  name: otel-collector
  namespace: monitoring
spec:
  ports:
    - name: otlp-grpc
      port: 4317
    - name: otlp-http
      port: 4318
  selector:
    app: otel-collector
```

Configure GateKey to send traces:

```yaml
telemetry:
  enabled: true
  service_name: "gatekey"
  environment: "production"
  otlp_endpoint: "otel-collector.monitoring:4317"
  otlp_protocol: "grpc"
  otlp_insecure: true  # Within cluster, TLS not required
  sample_rate: 1.0
```

## Health Endpoints

GateKey provides health check endpoints for orchestration systems:

### Liveness Probe

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Readiness Probe

```bash
curl http://localhost:8080/ready
```

Response:
```json
{
  "status": "ready",
  "database": "connected"
}
```

### Kubernetes Configuration

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

## Grafana Dashboard

### Recommended Panels

1. **Request Rate**: `rate(gatekey_http_requests_total[5m])`
2. **Error Rate**: `rate(gatekey_http_requests_total{status=~"5.."}[5m])`
3. **Latency P50**: `histogram_quantile(0.50, rate(gatekey_http_request_duration_seconds_bucket[5m]))`
4. **Latency P95**: `histogram_quantile(0.95, rate(gatekey_http_request_duration_seconds_bucket[5m]))`
5. **Latency P99**: `histogram_quantile(0.99, rate(gatekey_http_request_duration_seconds_bucket[5m]))`
6. **Active Sessions**: `gatekey_auth_sessions_active`
7. **Gateway Connections**: `gatekey_gateway_connections_active`
8. **DB Connection Pool**: `gatekey_db_connections_open` and `gatekey_db_connections_in_use`

### Sample Dashboard JSON

```json
{
  "title": "GateKey Overview",
  "panels": [
    {
      "title": "Request Rate",
      "type": "timeseries",
      "targets": [
        {
          "expr": "sum(rate(gatekey_http_requests_total[5m]))",
          "legendFormat": "Requests/sec"
        }
      ]
    },
    {
      "title": "Error Rate",
      "type": "timeseries",
      "targets": [
        {
          "expr": "sum(rate(gatekey_http_requests_total{status=~\"5..\"}[5m])) / sum(rate(gatekey_http_requests_total[5m])) * 100",
          "legendFormat": "Error %"
        }
      ]
    },
    {
      "title": "Latency Distribution",
      "type": "timeseries",
      "targets": [
        {
          "expr": "histogram_quantile(0.50, sum(rate(gatekey_http_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p50"
        },
        {
          "expr": "histogram_quantile(0.95, sum(rate(gatekey_http_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p95"
        },
        {
          "expr": "histogram_quantile(0.99, sum(rate(gatekey_http_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p99"
        }
      ]
    },
    {
      "title": "Active Sessions",
      "type": "stat",
      "targets": [
        {
          "expr": "gatekey_auth_sessions_active"
        }
      ]
    }
  ]
}
```

## Best Practices

1. **Set appropriate sample rates** - In high-traffic production environments, use sampling (e.g., `sample_rate: 0.1` for 10%) to reduce overhead
2. **Monitor key metrics** - Set up alerts for error rates, latency, and gateway health
3. **Use structured logging** - Enable JSON logging format for log aggregation
4. **Correlate traces and logs** - Include trace IDs in log messages for debugging
5. **Secure metrics endpoint** - Consider restricting `/metrics` access in production
6. **Regular dashboard review** - Periodically review dashboards for anomalies
