# Kubernetes Runtime Threat Monitoring System (KRTMS)

KRTMS is a Go-based microservices system for runtime threat monitoring in Kubernetes. It watches pod behavior, detects suspicious activity, consumes Falco runtime alerts, and notifies users through automated channels.

## What This Project Does

- Monitors Kubernetes pod lifecycle events in near real time.
- Detects anomalies such as abnormal restarts and privileged workload indicators.
- Ingests Falco runtime threat events for syscall-level security detections.
- Sends alerts to Slack and/or Email.
- Exposes operational metrics for Prometheus and Grafana.
- Supports DevSecOps delivery with Jenkins + Trivy and Kubernetes deployment automation.

## Core Services and Responsibilities

### 1) Event Collector Service

Path: `cmd/event-collector/main.go`

Responsibilities:
- Connects to Kubernetes using `client-go` informers.
- Watches pod add/update/delete events.
- Normalizes pod events into a shared `PodEvent` model.
- Publishes events to NATS subject `pods.events`.
- Exposes health and Prometheus metrics endpoints.

Why it exists:
- Decouples raw Kubernetes activity from threat logic.
- Provides a clean event stream for downstream processing.

### 2) Analyzer Service

Path: `cmd/analyzer/main.go`

Responsibilities:
- Subscribes to `pods.events` from NATS.
- Applies threat rules (restart anomalies, privileged labels).
- Accepts Falco webhook events via `POST /falco`.
- Converts detections into `ThreatAlert` messages.
- Publishes alerts to NATS subject `threats.alerts`.
- Exposes health and Prometheus metrics endpoints.

Why it exists:
- Centralizes threat decision logic.
- Combines Kubernetes behavior-based detection with Falco runtime findings.

### 3) Alert Manager Service

Path: `cmd/alert-manager/main.go`

Responsibilities:
- Subscribes to `threats.alerts` from NATS.
- Dispatches notifications to Slack webhook and/or SMTP email.
- Logs alerts if no notification channel is configured.
- Exposes health and Prometheus metrics endpoints.

Why it exists:
- Separates detection from notification delivery.
- Makes delivery channels replaceable without changing analyzer logic.

### 4) NATS Message Bus

Responsibilities:
- Provides asynchronous communication between microservices.
- Reduces direct coupling between producer and consumer services.

Subjects used:
- `pods.events` (Event Collector -> Analyzer)
- `threats.alerts` (Analyzer -> Alert Manager)

### 5) Monitoring Stack

Files:
- `deployments/k8s/monitoring/prometheus.yaml`
- `deployments/k8s/monitoring/grafana.yaml`

Responsibilities:
- Prometheus scrapes service metrics.
- Grafana visualizes event throughput and alert rates.

## End-to-End Workflow

1. Kubernetes emits pod lifecycle changes.
2. Event Collector captures and publishes normalized events.
3. Analyzer consumes events and runs detection rules.
4. Analyzer also ingests Falco runtime alerts through webhook.
5. Analyzer publishes high-confidence threat alerts.
6. Alert Manager sends Slack/Email notifications.
7. Prometheus scrapes all service metrics.
8. Grafana dashboards show system behavior and security signal trends.

## Threat Detection Logic (Current MVP)

- Pod restart increase -> medium severity anomaly alert.
- Label `security.privileged=true` -> high severity privilege escalation alert.
- Falco webhook event -> runtime threat alert (severity inferred from Falco priority).

## API and Runtime Endpoints

Event Collector:
- `GET /healthz`
- `GET /metrics`

Analyzer:
- `GET /healthz`
- `GET /metrics`
- `POST /falco`

Alert Manager:
- `GET /healthz`
- `GET /metrics`

## Repository Structure

- `cmd/` service entry points.
- `internal/common/` shared event models, queue client, metrics helpers.
- `deployments/docker/` Dockerfiles for each microservice.
- `deployments/k8s/base/` runtime components and services.
- `deployments/k8s/monitoring/` Prometheus and Grafana resources.
- `ansible/` deployment automation playbooks.
- `docs/` architecture notes.

## Local Build and Test

Requirements:
- Go 1.22+
- Docker
- Kubernetes cluster (kind/minikube/k3d/cloud)
- kubectl

Commands:

```bash
make tidy
make test
make build
```

## Container Build

```bash
REGISTRY=ghcr.io/<your-org> TAG=latest make docker
```

## Kubernetes Deployment

1. Update image names in:
- `deployments/k8s/base/event-collector.yaml`
- `deployments/k8s/base/analyzer.yaml`
- `deployments/k8s/base/alert-manager.yaml`

2. Deploy base + monitoring:

```bash
make deploy
```

3. Validate:

```bash
kubectl get pods -n krtms
kubectl get svc -n krtms
```

## Dashboard Access

```bash
kubectl port-forward svc/prometheus 9090:9090 -n krtms
kubectl port-forward svc/grafana 3000:3000 -n krtms
```

Grafana default credentials:
- User: `admin`
- Password: `admin`

## Falco Integration

Analyzer accepts Falco webhook payloads at:
- `POST http://analyzer:8081/falco`

Use Falco Sidekick or webhook forwarding to route Falco events to this endpoint.

## Alert Channel Configuration

Set environment variables in Alert Manager deployment:

Slack:
- `SLACK_WEBHOOK_URL`

Email:
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USER`
- `SMTP_PASS`
- `SMTP_FROM`
- `SMTP_TO`

## DevSecOps CI/CD (Jenkins + Trivy)

`Jenkinsfile` pipeline stages:
- Checkout source.
- Run Go tests.
- Build service images.
- Scan images with Trivy (fail on HIGH/CRITICAL).
- Push images on `main` branch.

## Automated Setup (Ansible)

```bash
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/setup_and_deploy.yml
```

## Future Improvements

- Add richer rule engine and baseline profiling.
- Add persistence for alert history.
- Add RBAC hardening for service accounts.
- Add integration tests and synthetic attack simulation.
