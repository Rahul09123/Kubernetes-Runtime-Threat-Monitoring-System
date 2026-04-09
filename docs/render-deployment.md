# Render.com Deployment Guide

Deploy KRTMS to Render.com (free tier) in minutes.

## 1. Create Render Account

1. Go to [Render.com](https://render.com)
2. Click "Get Started"
3. Sign up with GitHub (recommended)
4. Authorize GitHub access

## 2. Deploy from GitHub

### Option A: Blueprint Deploy (Recommended)

1. In your GitHub repo root, Render automatically detects `render.yaml`
2. Go to Render dashboard
3. Click "New +" → "Blueprint"
4. Connect your GitHub repo
5. Select branch: `main`
6. Click "Apply"
7. Render auto-deploys all services

### Option B: Manual Deploy

1. Render dashboard → "New +" → "Web Service"
2. Connect your GitHub repo
3. Configure each service:
   - Image: `ghcr.io/rahul09123/alert-manager` (plus event-collector, analyzer)
   - Port: `8082` (for alert-manager)
   - Plan: Free

## 3. Get Your Public URL

After ~5-10 minutes (first build):

1. Render dashboard → Your blueprint/project
2. Click "alert-manager" service
3. Copy the URL from Service Details
4. Example: `https://krtms-alert-manager.onrender.com`

**Your live monitoring dashboard:**
```
https://<your-render-url>
```

## 4. Service Architecture

All services deployed on Render:

| Service | Port | Status | Public? |
|---------|------|--------|---------|
| NATS | 4222 | Internal | No |
| Event Collector | 8080 | Backend | No |
| Analyzer | 8081 | Backend | No |
| **Alert Manager** | **8082** | **Frontend** | **Yes ✅** |
| Prometheus | 9090 | Metrics | No |

## 5. Auto-Deploy on Push

Render watches your GitHub repo on `main` branch:

1. Edit code on your Mac
2. Commit and push to GitHub
3. Render detects the push
4. Auto-redeploys all services (~5 min)

## 6. View Logs

To troubleshoot:

1. Render dashboard → Your blueprint
2. Click service name (e.g., "alert-manager")
3. Click "Logs" tab
4. Real-time logs appear

## 7. Free Tier Limits

- **Resources:** Shared compute
- **Storage:** Variable (not pinned)
- **Bandwidth:** Unlimited
- **Services:** Up to 3 free web services
- **Sleep:** Free tier services sleep after 15 min of inactivity (can wake with next request)
- **Cost:** Free (optional upgrade to Starter+ tier for $7/month)

## 8. Environment Variables

To customize services, add environment variables:

1. Render dashboard → Service → Environment
2. Add variables:
   - `NATS_URL=nats://nats:4222`
   - `LOG_LEVEL=debug`
   - etc.

## 9. Custom Domain (Optional)

Free tier uses Render's subdomain. To add custom domain:

1. Service details → Custom domains
2. Add your domain
3. Update DNS records (Render provides instructions)

## 10. Troubleshooting

### Services not starting
- **Check Logs:** Service → Logs tab
- **Common issues:** Image build failure, environment vars missing
- **Solution:** Review build logs, ensure images are in GHCR

### Alert Manager shows no data
- **Expected behavior:** There's no Kubernetes cluster on Render, so Event Collector can't watch pods
- **Workaround:** Use smoke test to generate fake alerts
- **Real use:** Deploy on actual Kubernetes (e.g., EKS, GKE) for pod monitoring

### Service keeps restarting
- **Check healthcheck:** Alert Manager expects `/healthz` endpoint to respond
- **Verify:** Each service has a working health check
- **Logs:** Check service output for startup errors

### Out of free tier quota
- **Limit:** 3 free web services (you have 5)
- **Solution:** Keep only essential services or upgrade to paid tier
- **Priority:** NATS, Event Collector, Alert Manager (keep these)

## 11. Render Dashboard Tips

- **Logs:** Real-time streaming of service output
- **Metrics:** CPU, memory usage
- **Events:** Deployment history and restarts
- **Shell:** SSH into running service (for debugging)

## 12. What's Different from Local Minikube

| Aspect | Local Minikube | Render |
|--------|---|---|
| Kubernetes pods | ✅ Monitored | ❌ Not available (no K8s cluster) |
| NATS messaging | ✅ Works | ✅ Works |
| Alert storage | ✅ Works | ✅ Works |
| Prometheus | ✅ Works | ✅ Works (metrics scraping) |
| Falco integration | ✅ Possible | ❌ Requires setup |

## 13. Next Steps

1. ✅ Push `main` branch to GitHub
2. ✅ Render auto-deploys within 5-10 min
3. ✅ Access your public Alert Manager URL
4. ✅ Share URL with your team

## 14. Upgrading Beyond Free Tier

If you need more resources:

1. Render dashboard → Service → Plan
2. Upgrade to Starter ($7/mo) or higher
3. Gets dedicated resources, no sleep

## Support

- Render docs: https://render.com/docs
- GitHub integration: https://render.com/docs/github
- For issues: Check service logs in Render dashboard
