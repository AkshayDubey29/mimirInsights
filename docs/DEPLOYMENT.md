# MimirInsights Deployment Guide

## Prerequisites

### 1. Kubernetes Cluster Requirements

- **Kubernetes Version**: 10.24higher
- **Cluster Type**: EKS, GKE, AKS, or any CNCF-compliant cluster
- **Resources**: Minimum2 CPU cores and 4GB RAM available
- **Storage**: No persistent storage required (stateless application)

###2. Required Tools

- **kubectl**: Configured to access your cluster
- **Helm**: Version 30.8 higher
- **Docker**: For local development and testing
- **AWS CLI**: If using EKS (for kubeconfig setup)

### 3. Cluster Components

- **Mimir**: Grafana Mimir deployed and running
- **ALB Controller**: AWS Load Balancer Controller (for EKS)
- **Cert Manager**: For TLS certificate management (optional)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/akshaydubey29/mimirInsights.git
cd mimirInsights
```

###2ld and Push Images

```bash
# Build Docker images
make docker-build

# Push to registry (requires authentication)
make docker-push
```

### 3. Deploy with Helm

```bash
# Add the Helm repository
helm repo add mimir-insights https://ghcr.io/akshaydubey29/mimir-insights

# Install MimirInsights
helm install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --create-namespace \
  --set backend.image.tag=latest \
  --set frontend.image.tag=latest
```

### 4. Access the Application

```bash
# Port forward to access locally
kubectl port-forward -n mimir-insights svc/mimir-insights-frontend 3000access via ingress (if configured)
# https://insights.yourdomain.com
```

## Detailed Deployment

### 1onfiguration

#### Environment Variables

Create a `values-custom.yaml` file:

```yaml
# Mimir configuration
backend:
  config:
    mimir:
      namespace: mimir
      api_url: http://mimir-distributor:9090
      timeout:30   
    k8s:
      in_cluster: true
      tenant_label: team
      tenant_prefix: tenant-
    
    log:
      level: info
      format: json

# Ingress configuration
ingress:
  enabled: true
  className: alb
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-west-2:123456789012tificate/your-cert-id
    alb.ingress.kubernetes.io/ssl-redirect: 443
  
  hosts:
    - host: insights.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: mimir-insights-frontend
              port:
                number: 80
        - path: /api
          pathType: Prefix
          backend:
            service:
              name: mimir-insights-backend
              port:
                number: 8080

# Resource configuration
backend:
  resources:
    requests:
      cpu: 250
      memory: 512
    limits:
      cpu: 500
      memory: 1Gi

frontend:
  resources:
    requests:
      cpu: 10
      memory: 128
    limits:
      cpu: 20
      memory: 256RBAC Configuration

The Helm chart creates the necessary RBAC resources:

```yaml
rbac:
  create: true
  rules:
    - apiGroups: ["]
      resources: ["namespaces",pods", "services, configmaps",persistentvolumeclaims]      verbs: ["get",list", watch]
    - apiGroups: ["apps]
      resources: ["deployments,statefulsets, eplicasets]      verbs: ["get",list", watch]
    - apiGroups: ["]
      resources: ["events]      verbs: ["get",list", watch]
    - apiGroups: ["]
      resources: ["resourcequotas, imitranges]      verbs: ["get",list", watch"]
```

### 2. Deployment Steps

#### Step 1: Verify Prerequisites

```bash
# Check cluster access
kubectl cluster-info

# Verify Mimir is running
kubectl get pods -n mimir

# Check available resources
kubectl top nodes
```

#### Step 2: Create Namespace (if not using Helm)

```bash
kubectl create namespace mimir-insights
```

#### Step 3: Install with Custom Values

```bash
helm install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --create-namespace \
  -f values-custom.yaml \
  --wait \
  --timeout=10m
```

#### Step 4: Verify Deployment

```bash
# Check pod status
kubectl get pods -n mimir-insights

# Check services
kubectl get svc -n mimir-insights

# Check ingress
kubectl get ingress -n mimir-insights

# Check logs
kubectl logs -n mimir-insights -l app.kubernetes.io/component=backend
```

### 3ment Configuration

#### Configure DNS

If using a custom domain:

```bash
# Get the ALB DNS name
kubectl get ingress -n mimir-insights -o jsonpath={.items0tatus.loadBalancer.ingress[0].hostname}'

# Add CNAME record in your DNS provider
# insights.yourdomain.com -> ALB-DNS-NAME
```

#### SSL Certificate

For production deployments, configure SSL:

```bash
# Install cert-manager (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v10.12/cert-manager.yaml

# Create ClusterIssuer for Lets Encrypt
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1ind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v2pi.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01
        ingress:
          class: alb
EOF
```

## Production Deployment

### 1. Security Considerations

#### Network Policies

Create network policies for additional security:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mimir-insights-network-policy
  namespace: mimir-insights
spec:
  podSelector: [object Object]}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: mimir
    ports:
    - protocol: TCP
      port: 9090
#### Pod Security Standards

Enable Pod Security Standards:

```bash
# Label namespace for restricted policy
kubectl label namespace mimir-insights pod-security.kubernetes.io/enforce=restricted
kubectl label namespace mimir-insights pod-security.kubernetes.io/audit=restricted
kubectl label namespace mimir-insights pod-security.kubernetes.io/warn=restricted
```

### 2. Monitoring Setup

#### ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: mimir-insights
  namespace: mimir-insights
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: mimir-insights
      app.kubernetes.io/component: backend
  endpoints:
  - port: http
    path: /metrics
    interval: 30`

#### PrometheusRule

```yaml
apiVersion: monitoring.coreos.com/v1nd: PrometheusRule
metadata:
  name: mimir-insights-alerts
  namespace: mimir-insights
spec:
  groups:
  - name: mimir-insights
    rules:
    - alert: MimirInsightsDown
      expr: up{job="mimir-insights} == 0
      for:5   labels:
        severity: critical
      annotations:
        summary:MimirInsights is down"
        description: MimirInsights has been down for more than 5 minutes"
```

### 3. Backup and Recovery

#### Configuration Backup

```bash
# Backup Helm values
helm get values mimir-insights -n mimir-insights > mimir-insights-backup.yaml

# Backup RBAC
kubectl get clusterrole,clusterrolebinding -l app.kubernetes.io/name=mimir-insights -o yaml > rbac-backup.yaml
```

#### Disaster Recovery

```bash
# Restore from backup
helm install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --create-namespace \
  -f mimir-insights-backup.yaml

kubectl apply -f rbac-backup.yaml
```

## Troubleshooting

### 1ommon Issues

#### Pods Not Starting

```bash
# Check pod events
kubectl describe pod -n mimir-insights <pod-name>

# Check logs
kubectl logs -n mimir-insights <pod-name>

# Check resource limits
kubectl top pods -n mimir-insights
```

#### Service Connectivity

```bash
# Test service connectivity
kubectl run test-pod --image=busybox -it --rm --restart=Never -- nslookup mimir-insights-backend

# Test API endpoint
kubectl run test-pod --image=curlimages/curl -it --rm --restart=Never -- curl http://mimir-insights-backend:8080i/health
```

#### Ingress Issues

```bash
# Check ingress status
kubectl describe ingress -n mimir-insights

# Check ALB controller logs
kubectl logs -n kube-system deployment.apps/aws-load-balancer-controller
```

###2. Debug Commands

```bash
# Get all resources
kubectl get all -n mimir-insights

# Check events
kubectl get events -n mimir-insights --sort-by=.lastTimestamp'

# Check configuration
kubectl get configmap -n mimir-insights

# Check secrets
kubectl get secrets -n mimir-insights
```

### 3. Performance Issues

```bash
# Check resource usage
kubectl top pods -n mimir-insights

# Check HPA status
kubectl get hpa -n mimir-insights

# Check metrics
kubectl port-forward -n mimir-insights svc/mimir-insights-backend 880:8080
curl http://localhost:8080metrics
```

## Upgrades

###1 Helm Upgrade

```bash
# Update Helm repository
helm repo update

# Upgrade deployment
helm upgrade mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --reuse-values \
  --set backend.image.tag=new-version \
  --set frontend.image.tag=new-version
```

### 2. Rollback

```bash
# List releases
helm list -n mimir-insights

# Rollback to previous version
helm rollback mimir-insights 1 -n mimir-insights
```

## Uninstallation

### 1. Remove Application

```bash
# Uninstall Helm release
helm uninstall mimir-insights -n mimir-insights

# Delete namespace
kubectl delete namespace mimir-insights
```

### 2Clean Up RBAC

```bash
# Delete ClusterRole and ClusterRoleBinding
kubectl delete clusterrole,clusterrolebinding -l app.kubernetes.io/name=mimir-insights
```

### 3Remove Images

```bash
# Remove from registry (if needed)
docker rmi ghcr.io/akshaydubey29/mimir-insights-backend:latest
docker rmi ghcr.io/akshaydubey29/mimir-insights-ui:latest
```

## Support

For issues and questions:

- **GitHub Issues**: [Create an issue](https://github.com/akshaydubey29/mimirInsights/issues)
- **Documentation**: [Project Wiki](https://github.com/akshaydubey29/mimirInsights/wiki)
- **Discussions**: [GitHub Discussions](https://github.com/akshaydubey29mimirInsights/discussions) 