# MimirInsights Production Deployment Guide

This guide covers deploying MimirInsights to production with monitoring and LLM features disabled for stability.

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.24+)
- Helm (v3.8+)
- kubectl configured
- Docker registry access (ghcr.io/akshaydubey29)

### 1. Deploy to Production

```bash
# Clone the repository
git clone https://github.com/AkshayDubey29/mimirInsights.git
cd mimirInsights

# Run the production deployment script
./deploy-production-final.sh
```

### 2. Verify Deployment

```bash
# Check pod status
kubectl get pods -n mimir-insights

# Check services
kubectl get svc -n mimir-insights

# Check ingress (if enabled)
kubectl get ingress -n mimir-insights
```

## 📋 Configuration

### Production Values File

The deployment uses `deployments/helm-chart/values-production-final.yaml` with:

- **Monitoring**: Disabled (`monitoring.enabled: false`)
- **LLM Features**: Disabled (`llm.enabled: false`)
- **Latest Images**: `ghcr.io/akshaydubey29/mimir-insights-*:latest`
- **High Availability**: 2 replicas each for frontend and backend
- **Security**: Non-root containers, read-only filesystems
- **Resource Limits**: Optimized for production workloads

### Key Features Disabled

```yaml
# Monitoring disabled for production stability
monitoring:
  enabled: false
  prometheus:
    enabled: false
  grafana:
    enabled: false

# LLM features disabled for production
llm:
  enabled: false
  openai:
    enabled: false
  anthropic:
    enabled: false
```

## 🔧 Customization

### Update Domain

Edit `deployments/helm-chart/values-production-final.yaml`:

```yaml
ingress:
  hosts:
    - host: your-domain.com  # Change this
      paths:
        - path: /
          pathType: Prefix
```

### Resource Allocation

Adjust resource limits based on your cluster capacity:

```yaml
backend:
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "512Mi"
      cpu: "500m"

frontend:
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"
```

### Scaling

Configure Horizontal Pod Autoscaler:

```yaml
hpa:
  backend:
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70
  frontend:
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70
```

## 🔒 Security Features

### Pod Security

- Non-root containers
- Read-only root filesystem
- Dropped capabilities
- Security contexts enforced

### Network Policies

- Ingress/egress traffic control
- Restricted port access
- Namespace isolation

### RBAC

- Service accounts with minimal permissions
- Role-based access control
- Namespace-scoped resources

## 📊 Monitoring & Logging

### Application Logs

```bash
# Backend logs
kubectl logs -f deployment/mimir-insights-backend -n mimir-insights

# Frontend logs
kubectl logs -f deployment/mimir-insights-frontend -n mimir-insights
```

### Health Checks

```bash
# Check backend health
kubectl exec -n mimir-insights deployment/mimir-insights-backend -- curl http://localhost:8080/health

# Check frontend health
kubectl exec -n mimir-insights deployment/mimir-insights-frontend -- curl http://localhost:80
```

## 🌐 Access

### Port Forwarding

```bash
# Frontend
kubectl port-forward svc/mimir-insights-frontend 8081:80 -n mimir-insights

# Backend
kubectl port-forward svc/mimir-insights-backend 8080:8080 -n mimir-insights
```

### Ingress Access

If ingress is configured, access via:
- Frontend: `https://your-domain.com`
- Backend API: `https://your-domain.com/api`

## 🔄 Updates

### Update Images

```bash
# Build and push new images
docker build -t ghcr.io/akshaydubey29/mimir-insights-backend:latest -f Dockerfile.backend .
docker build -t ghcr.io/akshaydubey29/mimir-insights-frontend:latest -f Dockerfile.frontend .
docker push ghcr.io/akshaydubey29/mimir-insights-backend:latest
docker push ghcr.io/akshaydubey29/mimir-insights-frontend:latest

# Update deployment
helm upgrade mimir-insights deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production-final.yaml
```

### Rollback

```bash
# List releases
helm list -n mimir-insights

# Rollback to previous version
helm rollback mimir-insights -n mimir-insights
```

## 🗑️ Cleanup

### Uninstall

```bash
# Remove Helm release
helm uninstall mimir-insights -n mimir-insights

# Remove namespace
kubectl delete namespace mimir-insights
```

### Complete Cleanup

```bash
# Remove all resources
kubectl delete all --all -n mimir-insights
kubectl delete namespace mimir-insights
```

## 🚨 Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Check image availability
   docker pull ghcr.io/akshaydubey29/mimir-insights-backend:latest
   docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:latest
   ```

2. **Resource Constraints**
   ```bash
   # Check node resources
   kubectl describe nodes
   
   # Check pod events
   kubectl describe pod -n mimir-insights <pod-name>
   ```

3. **Port Forwarding Issues**
   ```bash
   # Kill existing port forwards
   pkill -f "kubectl port-forward"
   
   # Check port usage
   lsof -i :8080
   lsof -i :8081
   ```

### Debug Commands

```bash
# Check pod status
kubectl get pods -n mimir-insights -o wide

# Check events
kubectl get events -n mimir-insights --sort-by='.lastTimestamp'

# Check logs with timestamps
kubectl logs -f deployment/mimir-insights-backend -n mimir-insights --timestamps

# Check resource usage
kubectl top pods -n mimir-insights
```

## 📈 Performance

### Resource Monitoring

```bash
# Monitor resource usage
kubectl top pods -n mimir-insights

# Check HPA status
kubectl get hpa -n mimir-insights

# Monitor scaling events
kubectl describe hpa -n mimir-insights
```

### Optimization Tips

1. **Memory**: Monitor memory usage and adjust limits
2. **CPU**: Watch CPU utilization for scaling decisions
3. **Network**: Monitor ingress/egress traffic
4. **Storage**: Check persistent volume usage

## 🔐 Security Checklist

- [ ] Non-root containers enabled
- [ ] Read-only filesystems configured
- [ ] Network policies applied
- [ ] RBAC configured
- [ ] TLS/SSL certificates installed
- [ ] Resource limits set
- [ ] Security contexts enforced
- [ ] Image vulnerability scanning completed

## 📞 Support

For issues or questions:

1. Check the troubleshooting section
2. Review application logs
3. Check Kubernetes events
4. Verify configuration values
5. Test connectivity and health endpoints

---

**Note**: This production deployment disables monitoring and LLM features for stability. Re-enable these features only after thorough testing in a staging environment. 