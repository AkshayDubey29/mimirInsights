# AWS ALB Deployment Guide for MimirInsights

## 🚀 Overview

This guide covers deploying MimirInsights to production using AWS Application Load Balancer (ALB) with proper ingress annotations and health check endpoints.

## 📋 Prerequisites

- AWS EKS cluster
- AWS Load Balancer Controller installed
- AWS Certificate Manager (ACM) certificate
- Proper VPC subnets and security groups
- kubectl and helm configured

## 🔧 AWS ALB Configuration

### Ingress Annotations

The deployment uses the following AWS ALB annotations:

```yaml
ingress:
  enabled: true
  className: alb
  annotations:
    # SSL Certificate
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:ap-northeast-2:138978013424:certificate/7b1c00f5-19ee-4e6c-9ca5-b30679eac---
    
    # Health Check Configuration
    alb.ingress.kubernetes.io/healthcheck-path: /healthz
    alb.ingress.kubernetes.io/healthcheck-port: "8081"
    alb.ingress.kubernetes.io/success-codes: "200"
    
    # Network Configuration
    alb.ingress.kubernetes.io/inbound-cidrs: 10.0.0.0/8
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP":80},{"HTTPS":443}]'
    alb.ingress.kubernetes.io/scheme: internal
    alb.ingress.kubernetes.io/target-type: ip
    
    # Security Groups and Subnets
    alb.ingress.kubernetes.io/security-groups: sg-03a537b10f8b713c3,sg-0faab6bb8700b4164
    alb.ingress.kubernetes.io/subnets: subnet-01ab33de57cfc8101,subnet-0247d97d25e7469f8,subnet-0ebfe41b055f0dec3,subnet-0971e77d71e---
    
    # Resource Tags
    alb.ingress.kubernetes.io/tags: role=couwatch_mimir
    
    # Ingress Class
    kubernetes.io/ingress.class: alb
```

### Health Check Endpoints

The frontend nginx configuration includes health check endpoints:

```nginx
# Health check endpoints
location /health {
    access_log off;
    return 200 "healthy\n";
    add_header Content-Type text/plain;
}

location /healthz {
    access_log off;
    return 200 "healthy\n";
    add_header Content-Type text/plain;
}
```

## 🚀 Deployment

### Quick Deployment

```bash
# Use the AWS ALB deployment script
./deploy-production-aws-alb.sh
```

### Manual Deployment

```bash
# Create namespace
kubectl create namespace mimir-insights

# Deploy with Helm
helm install mimir-insights deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production-aws-alb.yaml
```

### Values File

Use `deployments/helm-chart/values-production-aws-alb.yaml` which includes:

- AWS ALB ingress configuration
- Production image tags: `v1.0.0-20250718-110355`
- Health check endpoints
- Security hardening
- Resource optimization

## 🔍 Verification

### Check Deployment Status

```bash
# Check pods
kubectl get pods -n mimir-insights

# Check services
kubectl get svc -n mimir-insights

# Check ingress
kubectl get ingress -n mimir-insights
```

### Check ALB Status

```bash
# Get ALB hostname
kubectl get ingress -n mimir-insights -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}'

# Describe ingress for detailed status
kubectl describe ingress -n mimir-insights
```

### Health Checks

```bash
# Test frontend health
curl -f https://mimir-insights.yourdomain.com/healthz

# Test backend health
curl -f https://mimir-insights.yourdomain.com/api/health
```

## 🌐 Access

### URLs

- **Frontend**: `https://mimir-insights.yourdomain.com`
- **Backend API**: `https://mimir-insights.yourdomain.com/api`
- **Health Check**: `https://mimir-insights.yourdomain.com/healthz`

### Port Forwarding (for debugging)

```bash
# Frontend
kubectl port-forward svc/mimir-insights-frontend 8081:80 -n mimir-insights

# Backend
kubectl port-forward svc/mimir-insights-backend 8080:8080 -n mimir-insights
```

## 🔧 Configuration

### Update Domain

Edit `deployments/helm-chart/values-production-aws-alb.yaml`:

```yaml
ingress:
  hosts:
    - host: your-actual-domain.com  # Change this
      paths:
        - path: /
          pathType: Prefix
```

### Update Certificate ARN

Replace the certificate ARN with your actual ACM certificate:

```yaml
alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:region:account:certificate/your-certificate-id
```

### Update Security Groups and Subnets

Replace with your actual AWS resources:

```yaml
alb.ingress.kubernetes.io/security-groups: sg-your-security-group-1,sg-your-security-group-2
alb.ingress.kubernetes.io/subnets: subnet-your-subnet-1,subnet-your-subnet-2,subnet-your-subnet-3,subnet-your-subnet-4
```

## 📊 Monitoring

### ALB Metrics

Monitor ALB metrics in AWS CloudWatch:
- Request count
- Target response time
- HTTP 5XX errors
- Healthy/unhealthy host count

### Application Logs

```bash
# Frontend logs
kubectl logs -f deployment/mimir-insights-frontend -n mimir-insights

# Backend logs
kubectl logs -f deployment/mimir-insights-backend -n mimir-insights
```

### Health Check Monitoring

The ALB will automatically monitor the `/healthz` endpoint and route traffic only to healthy targets.

## 🔄 Updates

### Update Images

```bash
# Build new images
./build-production-final.sh

# Update deployment
helm upgrade mimir-insights deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production-aws-alb.yaml
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

### ALB Cleanup

The ALB will be automatically deleted when the ingress is removed, but you may need to manually clean up:
- Target groups
- Security group rules
- CloudWatch log groups

## 🚨 Troubleshooting

### Common Issues

1. **ALB Not Provisioning**
   ```bash
   # Check AWS Load Balancer Controller
   kubectl get pods -n kube-system | grep aws-load-balancer-controller
   
   # Check controller logs
   kubectl logs -n kube-system deployment/aws-load-balancer-controller
   ```

2. **Health Check Failures**
   ```bash
   # Test health endpoint directly
   kubectl exec -n mimir-insights deployment/mimir-insights-frontend -- curl -f http://localhost/healthz
   
   # Check nginx configuration
   kubectl exec -n mimir-insights deployment/mimir-insights-frontend -- nginx -t
   ```

3. **Certificate Issues**
   ```bash
   # Verify certificate exists
   aws acm describe-certificate --certificate-arn your-certificate-arn
   
   # Check certificate validation
   aws acm describe-certificate --certificate-arn your-certificate-arn --query 'Certificate.Status'
   ```

### Debug Commands

```bash
# Check ingress events
kubectl describe ingress -n mimir-insights

# Check ALB target groups
aws elbv2 describe-target-groups --query 'TargetGroups[?contains(TargetGroupName, `mimir-insights`)]'

# Check ALB listeners
aws elbv2 describe-listeners --load-balancer-arn your-alb-arn
```

## 📚 Resources

- [AWS Load Balancer Controller Documentation](https://kubernetes-sigs.github.io/aws-load-balancer-controller/)
- [AWS ALB Ingress Annotations](https://kubernetes-sigs.github.io/aws-load-balancer-controller/latest/guide/ingress/annotations/)
- [AWS Certificate Manager](https://docs.aws.amazon.com/acm/)
- [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)

---

**Note**: Ensure you have the AWS Load Balancer Controller installed and properly configured before deploying. The ALB provisioning may take 2-5 minutes after the ingress is created. 