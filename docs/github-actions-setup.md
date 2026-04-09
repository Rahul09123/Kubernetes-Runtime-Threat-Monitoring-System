# GitHub Actions Setup Guide

This project now uses GitHub Actions for CI/CD instead of Jenkins. The workflow automatically builds, tests, scans, and deploys the application on every push to the `main` branch.

## Workflow Overview

The `.github/workflows/build-deploy.yml` workflow includes:

1. **Test** (`checkout-and-test`): Runs Go tests
2. **Build Images** (`build-images`): Builds three Docker images:
   - `ghcr.io/rahul09123/event-collector:${RUN_NUMBER}`
   - `ghcr.io/rahul09123/analyzer:${RUN_NUMBER}`
   - `ghcr.io/rahul09123/alert-manager:${RUN_NUMBER}`
3. **Security Scan** (`scan-images`): Scans images with Trivy for HIGH/CRITICAL vulnerabilities
4. **Push Images** (`push-images`): Pushes images to GitHub Container Registry (main branch only)
5. **Deploy** (`deploy-kubernetes`): Applies Kubernetes manifests and updates deployments (main branch only)
6. **Smoke Test** (`smoke-test`): Verifies deployment health (main branch only, optional)

## Required GitHub Secrets

Before the workflow will run successfully, configure these secrets in your GitHub repository:

### 1. `GITHUB_TOKEN` (Automatic)
- **Already available** in GitHub Actions by default
- Used to push Docker images to GitHub Container Registry
- No additional setup required

### 2. `K8S_KUBECONFIG` (Required)
This is the kubeconfig file for your Kubernetes cluster, base64-encoded:

**Steps to set up:**

1. **Encode your kubeconfig:**
   ```bash
   cat ~/.kube/config | base64
   ```

2. **Add to GitHub repository secrets:**
   - Go to your GitHub repository
   - Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `K8S_KUBECONFIG`
   - Value: Paste the base64-encoded kubeconfig output
   - Click "Add secret"

**Alternative (Using GitHub CLI):**
```bash
cat ~/.kube/config | base64 | gh secret set K8S_KUBECONFIG
```

## Registry Authentication

The workflow uses `GITHUB_TOKEN` for GHCR authentication, which works out of the box. However, if you need to use different registry credentials:

1. Update the registry URL in `.github/workflows/build-deploy.yml` (currently `ghcr.io/rahul09123`)
2. Add appropriate secrets for registry username/password if needed

## Triggering Builds

Builds run automatically on:
- **Every push to `main` branch** → Full pipeline (build, scan, push, deploy, smoke test)
- **Pull requests to `main`** → Run tests and scans only (no push/deploy)

To manually trigger a workflow:
1. Go to Actions tab in GitHub
2. Select "Build, Test, Scan & Deploy"
3. Click "Run workflow" → Select branch and click "Run"

## Monitoring Workflow Runs

1. Go to your repository's **Actions** tab
2. Click on a workflow run to see detailed logs for each job
3. Failed jobs show error output; use this to diagnose issues

## Comparing with Jenkins

| Aspect | Jenkins | GitHub Actions |
|--------|---------|-----------------|
| Configuration | `Jenkinsfile` at repo root | `.github/workflows/*.yml` files |
| Secrets | Jenkins Credentials Store | GitHub Secrets UI + encrypted env vars |
| Runners | Self-hosted agents | GitHub-hosted runners (Ubuntu) |
| Logs | Jenkins UI | GitHub Actions UI |
| Cost | Server infrastructure | Free for public repos (includes free minutes) |

## Troubleshooting

### Images not pushing
- Check `GITHUB_TOKEN` is available (it's automatic)
- Verify GitHub Actions has permissions: Settings → Actions → General → Workflow permissions set to "Read and write permissions"

### Kubernetes deployment fails
- Verify `K8S_KUBECONFIG` secret is set and valid
- Check kubeconfig was base64-encoded correctly
- Ensure cluster is accessible from GitHub Actions runners (may need firewall/VPN rules)

### Smoke test fails
- Check `make smoke` target exists in Makefile
- Verify cluster connectivity with the encoded kubeconfig
- Review smoke test logs in GitHub Actions UI

## Disabling Jenkins

If you no longer need Jenkins, you can:
1. Remove the `Jenkinsfile` from the repository
2. Disable the Jenkins job/pipeline
3. Archive Jenkins configuration if needed

The `Jenkinsfile` is still available in git history if you need to reference it.
