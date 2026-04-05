# Kubernetes Runtime Threat Monitoring System Architecture

## Data Flow
1. Kubernetes emits pod lifecycle activity.
2. Event Collector watches pod resources using `client-go` informers.
3. Pod events are published into NATS (`pods.events`).
4. Analyzer consumes events, runs threat rules, and ingests Falco webhook alerts (`/falco`).
5. Analyzer publishes threat alerts into NATS (`threats.alerts`).
6. Alert Manager consumes threat alerts and dispatches notifications to Slack/Email.
7. All services expose Prometheus metrics.
8. Grafana visualizes metrics from Prometheus.

## Threat Detection Logic (MVP)
- High restart count spikes -> anomaly alert.
- Privileged pod indicator label (`security.privileged=true`) -> privilege escalation alert.
- Falco runtime event webhook payload -> runtime threat alert.

## CI/CD Security Gates
- Unit tests with `go test ./...`
- Trivy image scan with HIGH/CRITICAL fail threshold
- Image push on `main` branch only
