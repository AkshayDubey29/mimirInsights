# MimirInsights Production Deployment Guide

## 🚀 Enhanced Production Features

This production deployment includes all the latest enhancements:

### ✅ Enhanced Workload Discovery
- **Universal Support**: Deployments, StatefulSets, DaemonSets, ReplicaSets
- **Smart Detection**: Multiple label-based patterns for Alloy discovery
- **Production Ready**: Handles real-world Alloy deployment scenarios

### ✅ Latest Images
- **Backend**: `ghcr.io/akshaydubey29/mimir-insights-backend:20250118-enhanced-workload`
- **Frontend**: `ghcr.io/akshaydubey29/mimir-insights-frontend:20250718-070712`

### ✅ Advanced Features
- Drift Detection with ConfigMap comparison
- Capacity Planning with trend analysis  
- LLM Integration for metrics interpretation
- Alloy Tuning for all workload types
- Production Monitoring & Alerting
- Security Hardening & Audit Logging

## 📋 Pre-Deployment Checklist

### 1. Cluster Requirements
- [ ] Kubernetes 1.20+ cluster
- [ ] Sufficient resources (2+ nodes recommended)
- [ ] RBAC enabled
- [ ] Ingress controller deployed (nginx recommended)
- [ ] cert-manager deployed (for SSL certificates)

### 2. Required Tools
- [ ] `kubectl` configured for your cluster
- [ ] `helm` v3.8+ installed
- [ ] `openssl` for generating secrets

### 3. Domain & SSL Configuration
- [ ] Domain name configured (update `values-production.yaml`)
- [ ] DNS pointing to your cluster's ingress
- [ ] SSL certificate issuer configured (Let's Encrypt recommended)

### 4. External Dependencies
- [ ] OpenAI API key (for LLM features)
- [ ] S3 bucket for backups (optional)
- [ ] Monitoring stack (Prometheus/Grafana) for advanced monitoring

## 🔧 Configuration Steps

### 1. Update Production Values

Edit `deployments/helm-chart/values-production.yaml`:

```yaml
# Update your domain
ingress:
  hosts:
    - host: mimir-insights.yourdomain.com  # <- Change this
      
# Update cluster information
cluster:
  name: your-production-cluster  # <- Change this
  region: your-region           # <- Change this
  
# Configure node selectors (optional)
nodeSelector:
  kubernetes.io/arch: amd64
  node-type: your-node-type     # <- Update based on your labels
```

### 2. Create Required Secrets

The deployment script will create placeholder secrets. Update them with real values:

```bash
# Update LLM credentials
kubectl create secret generic mimir-insights-llm-credentials \
  --from-literal=api-key="your-actual-openai-api-key" \
  --namespace=mimir-insights \
  --dry-run=client -o yaml | kubectl apply -f -

# Update S3 credentials (if using backups)
kubectl create secret generic mimir-insights-s3-credentials \
  --from-literal=access-key="your-s3-access-key" \
  --from-literal=secret-key="your-s3-secret-key" \
  --namespace=mimir-insights \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 3. Configure Monitoring (Optional)

If you have Prometheus/Grafana:

```yaml
# In values-production.yaml
monitoring:
  serviceMonitor:
    enabled: true
    labels:
      prometheus: your-prometheus-instance  # <- Update this
  
  prometheusRule:
    enabled: true
```

## 🚀 Deployment Commands

### Option 1: Automated Production Deployment

```bash
# Full production deployment with all checks
./deploy-production.sh deploy
```

### Option 2: Manual Helm Deployment

```bash
# Create namespace
kubectl create namespace mimir-insights

# Deploy with Helm
helm install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values ./deployments/helm-chart/values-production.yaml \
  --timeout 10m \
  --wait
```

### Option 3: Upgrade Existing Deployment

```bash
# Upgrade with new images
./deploy-production.sh upgrade

# Or manually
helm upgrade mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values ./deployments/helm-chart/values-production.yaml \
  --timeout 10m \
  --wait
```

## 🔍 Verification Steps

### 1. Check Pod Status
```bash
kubectl get pods -n mimir-insights -l app.kubernetes.io/name=mimir-insights
```

### 2. Test Enhanced Workload Discovery
```bash
# Port forward to test locally
kubectl port-forward -n mimir-insights service/mimir-insights-backend 8080:8080 &

# Test new workload discovery endpoint
curl http://localhost:8080/api/alloy/workloads | jq .

# Expected response:
{
  "workloads_by_type": {
    "deployments": [...],
    "statefulsets": [...],
    "daemonsets": [...]
  },
  "type_counts": {
    "deployments": 1,
    "statefulsets": 1,
    "daemonsets": 1
  },
  "total_count": 3
}
```

### 3. Test Frontend Access
```bash
# Port forward frontend
kubectl port-forward -n mimir-insights service/mimir-insights-frontend 8081:80 &

# Open in browser
open http://localhost:8081
```

### 4. Check Monitoring
```bash
# View metrics
curl http://localhost:8080/metrics

# Check Prometheus targets (if ServiceMonitor enabled)
# Navigate to Prometheus UI and check for mimir-insights targets
```

## 📊 Production Features Verification

### Enhanced Workload Discovery
- [ ] API responds to `/api/alloy/workloads` 
- [ ] Discovers Deployments, StatefulSets, DaemonSets
- [ ] Shows correct type counts and metadata

### Drift Detection
- [ ] ConfigMap comparison working
- [ ] Hash tracking functional
- [ ] Alerts configured (if monitoring enabled)

### Capacity Planning
- [ ] Monthly reports generated
- [ ] Trend analysis working
- [ ] Resource recommendations available

### LLM Integration
- [ ] Natural language queries working
- [ ] Metrics interpretation functional
- [ ] API key configured correctly

### Alloy Tuning
- [ ] All workload types detected
- [ ] Resource recommendations working
- [ ] Patching APIs functional

## 🛠️ Troubleshooting

### Common Issues

#### 1. Pods Not Starting
```bash
# Check pod logs
kubectl logs -n mimir-insights -l app.kubernetes.io/name=mimir-insights --tail=100

# Check events
kubectl get events -n mimir-insights --sort-by='.lastTimestamp'

# Check resource constraints
kubectl describe pods -n mimir-insights
```

#### 2. Image Pull Issues
```bash
# Check if images are accessible
docker pull ghcr.io/akshaydubey29/mimir-insights-backend:20250118-enhanced-workload
docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:20250718-070712

# Check image pull secrets (if needed)
kubectl get secrets -n mimir-insights
```

#### 3. RBAC Issues
```bash
# Check service account permissions
kubectl auth can-i --list --as=system:serviceaccount:mimir-insights:mimir-insights

# Check cluster role bindings
kubectl get clusterrolebindings | grep mimir-insights
```

#### 4. API Connection Issues
```bash
# Test backend health
kubectl exec -n mimir-insights deployment/mimir-insights-backend -- curl http://localhost:8080/api/health

# Check backend logs for auto-discovery
kubectl logs -n mimir-insights deployment/mimir-insights-backend | grep -i discovery
```

### Performance Tuning

#### Resource Allocation
```yaml
# For high-load environments, increase resources:
backend:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 2Gi

frontend:
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 1Gi
```

#### Scaling Configuration
```yaml
# For large clusters, adjust HPA settings:
hpa:
  minReplicas: 3
  maxReplicas: 20
  targetCPUUtilizationPercentage: 60
  targetMemoryUtilizationPercentage: 70
```

## 🔒 Security Considerations

### 1. Network Policies
The production values include network policies. Ensure they match your cluster's networking setup.

### 2. RBAC Permissions
Review and adjust RBAC rules based on your security requirements:
```yaml
rbac:
  rules:
    # Add or remove permissions as needed
    - apiGroups: ["apps"]
      resources: ["deployments"]
      verbs: ["get", "list", "watch", "patch"]  # Remove "patch" if not needed
```

### 3. Security Context
The production deployment runs with restricted security context:
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1001
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
```

## 📈 Monitoring & Alerting

### Prometheus Metrics
Key metrics to monitor:
- `mimir_insights_workloads_discovered_total`
- `mimir_insights_api_requests_total`
- `mimir_insights_drift_detections_total`
- `mimir_insights_capacity_recommendations_total`

### Custom Alerts
The production values include basic alerts. Customize them for your environment:
```yaml
monitoring:
  prometheusRule:
    rules:
      - alert: MimirInsightsWorkloadDiscoveryFailed
        expr: increase(mimir_insights_discovery_errors_total[5m]) > 5
        for: 2m
        labels:
          severity: warning
```

## 🔄 Maintenance

### Regular Tasks
- [ ] Monitor pod health and logs
- [ ] Review capacity recommendations
- [ ] Update secrets rotation
- [ ] Backup configuration (if enabled)
- [ ] Update images regularly

### Upgrade Process
```bash
# 1. Update values file with new image tags
# 2. Run upgrade
./deploy-production.sh upgrade

# 3. Verify functionality
./deploy-production.sh test

# 4. Monitor for issues
kubectl logs -n mimir-insights -l app.kubernetes.io/name=mimir-insights -f
```

## 📞 Support

### Logs Collection
```bash
# Collect all logs for troubleshooting
kubectl logs -n mimir-insights -l app.kubernetes.io/name=mimir-insights --previous > mimir-insights-logs.txt

# Get cluster information
kubectl get all -n mimir-insights -o yaml > mimir-insights-resources.yaml
```

### Key API Endpoints for Testing
- `GET /api/health` - Health check
- `GET /api/alloy/workloads` - Enhanced workload discovery
- `GET /api/alloy/deployments` - Legacy deployment discovery (backward compatibility)
- `GET /api/metrics` - Prometheus metrics
- `GET /api/discovery/namespaces` - Namespace discovery
- `GET /api/capacity/recommendations` - Capacity planning

## 🎉 Success Criteria

Your production deployment is successful when:
- [ ] All pods are running and healthy
- [ ] Enhanced workload discovery finds all Alloy instances
- [ ] Web UI is accessible via configured domain
- [ ] API endpoints respond correctly
- [ ] Monitoring metrics are being collected
- [ ] SSL certificates are working
- [ ] Alerts are configured (if monitoring enabled)

---

**🚀 Ready for Production!**

Your MimirInsights deployment now includes comprehensive workload discovery, advanced monitoring capabilities, and production-grade security features. 