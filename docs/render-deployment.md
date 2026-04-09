# Render.com + GitHub Actions Deployment Guide

This repo now uses GitHub Actions to build images and trigger Render deploy hooks.

## 1. Create the Render services once

Create your Render services manually in the dashboard and set them to use these image URLs:

- `nats: docker.io/library/nats:2.10-alpine`
- `event-collector: ghcr.io/rahul09123/event-collector:latest`
- `analyzer: ghcr.io/rahul09123/analyzer:latest`
- `alert-manager: ghcr.io/rahul09123/alert-manager:latest`
- `prometheus: docker.io/prom/prometheus:v2.54.1`

For the app services, keep `/healthz` as the health check path and the existing `NATS_URL` / `HTTP_ADDR` environment variables.

## 2. Add GitHub repository secrets

In GitHub, add these Actions secrets:

- `RENDER_NATS_DEPLOY_HOOK_URL`
- `RENDER_EVENT_COLLECTOR_DEPLOY_HOOK_URL`
- `RENDER_ANALYZER_DEPLOY_HOOK_URL`
- `RENDER_ALERT_MANAGER_DEPLOY_HOOK_URL`
- `RENDER_PROMETHEUS_DEPLOY_HOOK_URL`

Each secret should contain the deploy hook URL from the Render service settings page.

## 3. Workflow behavior

The workflow at [.github/workflows/render-deploy.yml](../.github/workflows/render-deploy.yml) does this:

1. Runs Go tests.
2. Builds and pushes `latest` and commit-tagged Docker images to GHCR.
3. Triggers the Render deploy hooks for each service.

That means a push to `main` becomes the deploy signal.

## 4. Public URL

After Render finishes the deploy, open the `alert-manager` service in the Render dashboard and copy its public URL.

## 5. Notes on free tier

Render free tier limits may prevent all five services from running long term on free instances. If you hit that limit, keep the frontend and required backend services on Render, or move the full stack to another platform.

## 6. Troubleshooting

- If deploys do not start, verify the deploy hook secrets are present and correct.
- If images are missing, confirm the GHCR package names and that the workflow pushed `latest`.
- If a service fails health checks, inspect the Render logs for that service.
