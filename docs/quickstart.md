# KRTMS Quickstart (Minikube)

This is a fast, first-time setup guide to run KRTMS locally on Minikube and verify the full alert pipeline.

## Goal

In about 5-10 minutes you will:
1. Build local images into Minikube.
2. Deploy KRTMS services and monitoring stack.
3. Verify all workloads are healthy.
4. Run an end-to-end smoke test.
5. Open Alert Manager UI and Prometheus.

## Prerequisites

1. Minikube installed and running.
2. kubectl installed and pointing to Minikube context.
3. Docker installed.
4. make installed.

## Step 1: Start Minikube

```bash
minikube start
kubectl config current-context
```

Expected context: `minikube`.

## Step 2: Build Images Into Minikube

From the project root:

```bash
eval $(minikube docker-env)
make docker
```

Why this matters:
- It builds service images directly inside Minikube's Docker daemon so Kubernetes can pull them without pushing to a remote registry.

## Step 3: Deploy to Kubernetes

```bash
make deploy
```

This applies:
- Base runtime resources (namespace, NATS, app services, RBAC)
- Monitoring resources (Prometheus)

## Step 4: Confirm Everything Is Running

```bash
kubectl get pods -n krtms
kubectl get deploy -n krtms
kubectl get svc -n krtms
```

All deployments should show available replicas.

## Step 5: Run End-to-End Smoke Test

```bash
make smoke
```

What smoke test validates:
1. A temporary labeled pod is created.
2. Event Collector captures it.
3. Analyzer generates an alert.
4. Alert Manager stores it and serves it through API.
5. Temporary pod is cleaned up automatically.

## Step 6: Open Dashboards and API

Use separate terminals:

```bash
kubectl port-forward svc/prometheus 9090:9090 -n krtms
kubectl port-forward svc/alert-manager 8082:8082 -n krtms
```

Access:
- Prometheus: http://localhost:9090/targets
- Alert Manager UI: http://localhost:8082/
- Alert API: http://localhost:8082/api/alerts

## Troubleshooting

### Images not found

```bash
eval $(minikube docker-env)
make docker
make deploy
```

### Alert Manager UI is empty

```bash
make smoke
```

### Verify service logs

```bash
kubectl logs deploy/event-collector -n krtms --tail=100
kubectl logs deploy/analyzer -n krtms --tail=100
kubectl logs deploy/alert-manager -n krtms --tail=100
```

## Optional Cleanup

```bash
kubectl delete namespace krtms
```

Or stop Minikube:

```bash
minikube stop
```
