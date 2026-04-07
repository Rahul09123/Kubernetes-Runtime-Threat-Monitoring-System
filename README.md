# Kubernetes Runtime Threat Monitoring System (KRTMS)

KRTMS is a microservices-based runtime monitoring system for Kubernetes workloads.
It captures pod lifecycle behavior, applies threat detection rules, ingests Falco runtime events, and routes normalized alerts to notification channels and dashboards.

This repository includes:
- 3 Go services (`event-collector`, `analyzer`, `alert-manager`)
- NATS as the event bus
- Kubernetes manifests (base + monitoring)
- Prometheus and Grafana integration
- Jenkins CI pipeline with optional deploy + smoke validation
- An end-to-end smoke test for production-like verification

Quick onboarding path:
- See `docs/quickstart.md` for a fast Minikube setup and verification guide.

## 1. Architecture Overview

### Components

1. Event Collector
- Watches Kubernetes Pod events using `client-go` informers.
- Publishes normalized events to NATS subject `pods.events`.
- Exposes `/healthz` and `/metrics`.

2. Analyzer
- Subscribes to `pods.events`.
- Applies built-in detection rules.
- Ingests Falco webhook events at `POST /falco`.
- Publishes threat alerts to NATS subject `threats.alerts`.
- Exposes `/healthz` and `/metrics`.

3. Alert Manager
- Subscribes to `threats.alerts`.
- Stores recent alerts in memory (latest 200).
- Sends notifications via Slack and/or SMTP.
- Serves web dashboard and `GET /api/alerts`.
- Exposes `/healthz` and `/metrics`.

4. NATS
- Decouples producers and consumers.
- Provides asynchronous event flow between services.

5. Prometheus + Grafana
- Prometheus scrapes metrics from all services.
- Grafana visualizes throughput and alert trends.

### Data Flow

1. Kubernetes emits pod add/update/delete state changes.
2. Event Collector converts changes into `PodEvent` objects.
3. Event Collector publishes events to `pods.events`.
4. Analyzer consumes pod events and applies rule checks.
5. Analyzer ingests Falco events via webhook and normalizes to alerts.
6. Analyzer publishes alerts to `threats.alerts`.
7. Alert Manager consumes alerts, stores latest records, dispatches notifications.
8. Prometheus scrapes service metrics.
9. Grafana renders operational and security panels.

## 2. Threat Detection Logic (Current Rules)

### Pod behavior rules

1. Restart anomaly
- Trigger: event type `POD_RESTART`
- Severity: `medium`
- Category: `anomaly`

2. Privileged label indicator
- Trigger: pod label `security.privileged=true`
- Severity: `high`
- Category: `privilege-escalation`

### Falco ingestion rule

- Trigger: request to `POST /falco`
- Category: `runtime-threat`
- Severity: derived from Falco `priority` (lower-cased), defaults to `high` when absent

## 3. Service Endpoints

### Event Collector
- `GET /healthz`
- `GET /metrics`

### Analyzer
- `GET /healthz`
- `GET /metrics`
- `POST /falco`

### Alert Manager
- `GET /healthz`
- `GET /metrics`
- `GET /api/alerts`
- `GET /` (embedded static web UI)

## 4. Repository Layout

- `cmd/`
  - `event-collector/`
  - `analyzer/`
  - `alert-manager/`
- `internal/common/`
  - Shared metrics, queue, and event/alert models
- `deployments/docker/`
  - Service Dockerfiles
- `deployments/k8s/base/`
  - Namespace, NATS, services, deployments, RBAC
- `deployments/k8s/monitoring/`
  - Prometheus + Grafana resources
- `scripts/`
  - Deployment helper and smoke test
- `ansible/`
  - Automation playbooks
- `docs/`
  - Architecture notes
- `Jenkinsfile`
  - CI pipeline

## 5. Prerequisites

Minimum recommended toolchain:

1. Go 1.22+
2. Docker
3. Kubernetes cluster (Minikube, Kind, K3d, or cloud)
4. kubectl
5. make

Optional:
- Falco / Falco Sidekick for runtime syscall alerts
- Jenkins for CI automation

## 6. Local Build and Test

```bash
make tidy
make test
make build
```

## 7. Build Container Images

### Generic registry flow

```bash
REGISTRY=ghcr.io/<your-org> TAG=latest make docker
```

### Minikube local-image flow

If you are deploying to Minikube and do not want to push images to a remote registry:

```bash
eval $(minikube docker-env)
make docker
```

This builds images directly into the Minikube Docker daemon so Kubernetes can pull them locally.

## 8. Kubernetes Deployment

### Deploy all components

```bash
make deploy
```

`make deploy` applies Kustomize overlays for:
- `deployments/k8s/base`
- `deployments/k8s/monitoring`

### Validate rollout

```bash
kubectl get pods -n krtms
kubectl get deploy -n krtms
kubectl get svc -n krtms
```

### Important RBAC detail

Event Collector requires pod watch permissions cluster-wide.
This repository includes:
- ServiceAccount `event-collector-sa`
- ClusterRole for `pods` with `get/list/watch`
- ClusterRoleBinding from service account to role

## 9. Observability and Dashboards

### Prometheus

Port-forward:

```bash
kubectl port-forward svc/prometheus 9090:9090 -n krtms
```

Open: `http://localhost:9090/targets`

Expected targets: `event-collector:8080`, `analyzer:8081`, `alert-manager:8082`

### Grafana

Port-forward:

```bash
kubectl port-forward svc/grafana 3000:3000 -n krtms
```

Open: `http://localhost:3000`

Default credentials:
- Username: `admin`
- Password: `admin`

Provisioning included:
- Prometheus datasource
- Dashboard provider (`KRTMS` folder)
- KRTMS dashboard panels for events and alerts

If dashboard changes do not appear:

```bash
kubectl apply -k deployments/k8s/monitoring
kubectl rollout restart deploy/grafana -n krtms
```

### Alert Manager UI/API

Port-forward:

```bash
kubectl port-forward svc/alert-manager 8082:8082 -n krtms
```

- Web UI: `http://localhost:8082/`
- Alert API: `http://localhost:8082/api/alerts`

## 10. Alert Delivery Configuration

Alert Manager sends notifications to Slack and/or Email when configured.

### Slack

Environment variable:
- `SLACK_WEBHOOK_URL`

### Email (SMTP)

Environment variables:
- `SMTP_HOST`
- `SMTP_PORT` (default 587)
- `SMTP_USER`
- `SMTP_PASS`
- `SMTP_FROM` (optional, defaults to `SMTP_USER`)
- `SMTP_TO`

If neither Slack nor SMTP is configured, alerts are still processed and logged.

## 11. Falco Integration

Analyzer accepts Falco webhook payloads at:
- `POST http://analyzer:8081/falco`

Typical integration path:
1. Falco detects runtime syscall events.
2. Falco Sidekick forwards webhook payloads to Analyzer.
3. Analyzer converts event to `ThreatAlert` and publishes to `threats.alerts`.
4. Alert Manager stores and dispatches.

## 12. Smoke Test (End-to-End Validation)

A complete pipeline check is included:

```bash
make smoke
```

What it does:
1. Creates a temporary pod in the target namespace with label `security.privileged=true`.
2. Waits for pod readiness.
3. Queries `alert-manager` API from inside cluster.
4. Verifies an alert exists for the generated pod name.
5. Cleans up test pod automatically.

Environment override:
- `NAMESPACE=<namespace> make smoke`

## 13. CI/CD (Jenkins)

Pipeline stages:
1. Checkout
2. Go Test
3. Build Images
4. Trivy Scan (HIGH/CRITICAL fail the build)
5. Push Images (main branch)
6. Deploy to Kubernetes (optional)
7. Smoke Test (optional, after deploy)

### Jenkins control flags

- `RUN_DEPLOY=true|false`
- `RUN_SMOKE_TEST=true|false`
- `K8S_NAMESPACE=<namespace>`

Behavior:
- Deploy stage runs only on `main` and only when `RUN_DEPLOY=true`.
- Smoke stage runs only on `main` and only when both `RUN_DEPLOY=true` and `RUN_SMOKE_TEST=true`.

## 14. Troubleshooting Guide

### A. Pods are running but no alerts appear

Checklist:
1. Verify NATS is healthy:
```bash
kubectl get pods -n krtms
```
2. Check service logs:
```bash
kubectl logs deploy/event-collector -n krtms --tail=100
kubectl logs deploy/analyzer -n krtms --tail=100
kubectl logs deploy/alert-manager -n krtms --tail=100
```
3. Run smoke test:
```bash
make smoke
```

### B. Grafana opens but dashboard is missing

1. Reapply monitoring manifests:
```bash
kubectl apply -k deployments/k8s/monitoring
```
2. Restart Grafana:
```bash
kubectl rollout restart deploy/grafana -n krtms
```

### C. Metrics targets are down in Prometheus

1. Confirm services exist:
```bash
kubectl get svc -n krtms
```
2. Confirm corresponding pods are ready:
```bash
kubectl get pods -n krtms
```
3. Open `http://localhost:9090/targets` and inspect error messages for each target.

### D. Event Collector cannot watch pods

- Ensure RBAC resources are applied from `deployments/k8s/base`.
- Verify deployment uses service account `event-collector-sa`.

### E. Minikube cannot find images

Use Minikube Docker daemon before building:

```bash
eval $(minikube docker-env)
make docker
make deploy
```

## 15. Security Notes

1. Default Grafana admin password is `admin`; change in production.
2. SMTP and Slack secrets should be managed with Kubernetes Secrets, not plain manifests.
3. Add network policies and stricter RBAC for production hardening.
4. Enable image signing and policy enforcement if required by your platform.

## 16. Operational Recommendations

1. Add persistent storage for alert history if long-term retention is required.
2. Add readiness/liveness probes for stronger self-healing behavior.
3. Add namespace/cluster filters if event volume is high.
4. Add load and chaos tests for resiliency validation.

## 17. Useful Commands Cheat Sheet

### Build and deploy (Minikube)

```bash
eval $(minikube docker-env)
make docker
make deploy
```

### Health checks

```bash
kubectl get pods -n krtms
kubectl get deploy -n krtms
kubectl get svc -n krtms
```

### Logs

```bash
kubectl logs deploy/event-collector -n krtms --tail=100
kubectl logs deploy/analyzer -n krtms --tail=100
kubectl logs deploy/alert-manager -n krtms --tail=100
```

### Smoke test

```bash
make smoke
```

---

If you are onboarding a new contributor, the fastest validation path is:
1. Build images in Minikube.
2. Run `make deploy`.
3. Run `make smoke`.
4. Open Grafana and Alert Manager API.
