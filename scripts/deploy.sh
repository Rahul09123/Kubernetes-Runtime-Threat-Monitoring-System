#!/usr/bin/env bash
set -euo pipefail

kubectl apply -k deployments/k8s/base
kubectl apply -k deployments/k8s/monitoring
kubectl get pods -n krtms
