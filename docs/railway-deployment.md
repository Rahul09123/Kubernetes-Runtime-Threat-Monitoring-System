# Railway.app Deployment Guide

This guide walks you through deploying KRTMS to Railway.app (free tier).

## 1. Create Railway.app Account

1. Go to [Railway.app](https://railway.app)
2. Click "Start New Project"
3. Sign up with GitHub (easiest)
4. Connect your GitHub account

## 2. Deploy from GitHub Repository

1. In Railway dashboard, click "New Project"
2. Select "Deploy from GitHub repo"
3. Choose your repository: `Kubernetes-Runtime-Threat-Monitoring-System`
4. Click "Deploy Now"

Railway automatically detects `docker-compose.yml` and starts building/deploying all services:
- **NATS** (message broker on 4222)
- **Event Collector** (backend on 8080)
- **Analyzer** (backend on 8081)
- **Alert Manager** (frontend on 8082) — **publicly accessible**
- **Prometheus** (metrics on 9090)

## 3. Access Your Deployed App

After ~5-10 minutes (first build takes longer):

1. Go to Railway dashboard
2. Click your project
3. Click "alert-manager" service
4. Copy the public URL from "Deployments" tab
5. Example: `https://krtms-alert-manager-prod.up.railway.app`

**Your live monitoring dashboard is at:**
```
https://<your-railway-url>:8082
```

## 4. Verify Deployment

Once services are running:

1. **Check Alert Manager UI:** Visit the public URL
2. **View logs:** Railway dashboard → service → Logs tab
3. **Check service health:** Each service has `/healthz` endpoint

## 5. Auto-Deploy on GitHub Push

Railway auto-deploys whenever you push to `main` branch:

1. Edit code on your Mac
2. Commit and push to GitHub
3. Railway automatically rebuilds and re-deploys
4. New version live in ~5 minutes

## 6. Free Tier Limits (as of 2026)

- **Storage:** 5 GB
- **RAM:** Shared across all services
- **CPU:** Limited (suitable for demo/testing)
- **Bandwidth:** Included
- **Cost:** Free (no credit card needed for first $5/month)

## 7. Environment Variables

If you need to customize behavior, add environment variables in the Railway dashboard:

1. Project → Service → Variables
2. Examples:
   - `NATS_URL=nats://nats:4222` (default, already set)
   - `LOG_LEVEL=debug`

## 8. View Logs

To debug any service:

1. Railway dashboard → Your project
2. Click service name (e.g., "alert-manager")
3. Click "Logs" tab
4. Real-time logs appear here

## 9. Scale Services (Free Tier)

On free tier, all services run on shared resources. To scale up:

1. Upgrade to Hobby tier ($5/month) or Pro tier
2. Click service → Scale tab
3. Increase replicas or RAM allocation

## 10. Custom Domain (Optional)

Free tier includes a Railway subdomain, but you can add a custom domain:

1. Project settings → Domains
2. Add your own domain (requires DNS setup)

## Troubleshooting

### Services not starting
- Check Logs tab for error messages
- Verify images are building successfully
- Ensure `docker-compose.yml` is in repo root

### No data in Alert Manager UI
- Event Collector might not be receiving Kubernetes pod events (expected on Railway)
- NATS connection might fail if services start too fast
- Check Alert Manager logs: `kubectl logs` → see actual logs in Railway dashboard

### Out of free tier quota
- Services exceed 5GB storage or $5/month billing limit
- Consider upgrading or optimizing services

## Manual Deployment Alternative

If you prefer manual deployment without GitHub:

```bash
npm install -g @railway/cli
railway login
railway init
railway up
```

But GitHub auto-deploy is recommended.

## What's Deployed

| Service | Port | Purpose | Public? |
|---------|------|---------|---------|
| NATS | 4222 | Event bus | No (internal) |
| Event Collector | 8080 | Pod watcher | No (internal) |
| Analyzer | 8081 | Threat detection | No (internal) |
| **Alert Manager** | **8082** | **Monitoring UI** | **Yes** ✅ |
| Prometheus | 9090 | Metrics scraper | No (internal) |

## Next Steps

1. ✅ Push `main` branch to GitHub
2. ✅ Railway auto-deploys within 5 minutes
3. ✅ Access your public Alert Manager URL
4. ✅ Share the URL with your team

## Support

- Railway docs: https://railway.app/docs
- GitHub Actions auto-deploy: Already configured in `.github/workflows/build-deploy.yml`
- For issues: Check Railway dashboard Logs and compare with local Minikube setup
