# Kubernetes Runtime Threat Monitoring System (KRTMS)

KRTMS is a Go-based microservices platform for real-time Kubernetes runtime threat monitoring using a DevSecOps workflow.

## Components
- `event-collector`: Watches pod lifecycle events with `client-go` and publishes normalized events.
- `analyzer`: Detects suspicious behavior and ingests Falco alerts.
- `alert-manager`: Sends threat notifications to Slack/Email.
- `prometheus` + `grafana`: Metrics and dashboards.
- `nats`: Lightweight message bus connecting microservices.

## Tech Stack
- Go, Docker, Kubernetes
- Jenkins + Trivy (CI/CD security gate)
- Ansible automation
- Falco, Prometheus, Grafana

## Repository Layout
- `cmd/` service entrypoints
- `internal/common/` shared contracts, queue, metrics
- `deployments/docker/` Dockerfiles per service
- `deployments/k8s/` Kubernetes manifests (base + monitoring)
- `ansible/` infra and deploy automation
- `docs/` architecture and design notes

## Quick Start
### 1) Install dependencies
- Go 1.22+
- Docker
- Kubernetes cluster (kind, minikube, k3d, or cloud cluster)
- kubectl

### 2) Build and test
```bash
make tidy
make test
make build
```

### 3) Build container images
```bash
REGISTRY=ghcr.io/<your-org> TAG=latest make docker
```

### 4) Deploy to Kubernetes
Update image names in:
- `deployments/k8s/base/event-collector.yaml`
- `deployments/k8s/base/analyzer.yaml`
- `deployments/k8s/base/alert-manager.yaml`

Then run:
```bash
make deploy
```

### 5) Access dashboards
```bash
kubectl port-forward svc/prometheus 9090:9090 -n krtms
kubectl port-forward svc/grafana 3000:3000 -n krtms
```
Grafana default credentials:
- user: `admin`
- password: `admin`

## Falco Integration
This repository exposes analyzer endpoint:
- `POST /falco` on analyzer service (`analyzer:8081/falco`)

Configure Falco output to send matching events to this endpoint using Falco Sidekick or webhook forwarding.

## Alerting Configuration
Set environment variables on `alert-manager` deployment:
- Slack: `SLACK_WEBHOOK_URL`
- Email: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`, `SMTP_TO`

## Jenkins + Trivy
The provided `Jenkinsfile` includes:
- Checkout
- Go tests
- Docker image build
- Trivy vulnerability scan (fail on HIGH/CRITICAL)
- Push images on `main`

## Ansible Deployment
```bash
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/setup_and_deploy.yml
```

## Suggested Incremental Git Commits
1. `chore: initialize project structure`
2. `feat: add event collector service and pod watcher`
3. `feat: add nats message queue integration`
4. `feat: add analyzer service with threat rules and falco webhook`
5. `feat: add alert manager with slack and email notifiers`
6. `feat: add prometheus metrics and grafana dashboard`
7. `chore: add dockerfiles and kubernetes manifests`
8. `ci: add jenkins pipeline with trivy scan`
9. `chore: add ansible deployment automation`
10. `docs: add architecture and runbook`
