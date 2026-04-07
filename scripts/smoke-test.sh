#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-krtms}"
POD_NAME="smoke-test-$(date +%s)"

cleanup() {
  kubectl delete pod "$POD_NAME" -n "$NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[1/4] Creating test pod: $POD_NAME"
kubectl run "$POD_NAME" \
  -n "$NAMESPACE" \
  --image=nginx:1.27 \
  --restart=Never \
  --labels=security.privileged=true >/dev/null

echo "[2/4] Waiting for pod readiness"
kubectl wait --for=condition=Ready "pod/$POD_NAME" -n "$NAMESPACE" --timeout=120s >/dev/null

echo "[3/4] Checking alert manager API for generated alert"
if ! kubectl run smoke-curl \
  -n "$NAMESPACE" \
  --rm -i \
  --restart=Never \
  --image=curlimages/curl:8.8.0 \
  --command -- \
  sh -c "for i in 1 2 3 4 5 6 7 8 9 10; do curl -fsS http://alert-manager:8082/api/alerts | grep -q '$POD_NAME' && exit 0; done; exit 1"; then
  echo "Smoke test failed: alert for pod '$POD_NAME' not found in alert-manager API"
  exit 1
fi

echo "[4/4] Smoke test passed: end-to-end alert pipeline is healthy"
