# 📊 MimirInsights

> AI-enabled Observability Configuration Auditor & Optimizer for Multi-Tenant Grafana Mimir Deployments

## 🚀 Executive Summary

**MimirInsights** is a Kubernetes-native, AI-driven application designed to automatically discover, analyze, audit, and optimize per-tenant limit configurations for Grafana Mimir in multi-tenant observability environments.

It bridges the gap between metrics ingestion trends and Mimir's hard-coded limits by:

* Continuously analyzing ingestion and rejection patterns over time
* Suggesting optimal values for Mimir limits with a 10–20% safety buffer
* Auditing all tenant-level configurations and surfacing missing or misconfigured limits
* Visualizing full metrics flow, system health, and tenant pipelines
* Auto-discovering infrastructure setup across namespaces
* Providing a production-grade UI with metrics dashboard and limit recommendations

## ��️ Architecture

### Production Simulation Environment

```
┌─────────────────────────────────────────────────────────────┐
│                    KIND CLUSTER                             │
│                                                             │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │  mimir-insights │    │     mimir       │                │
│  │   namespace     │    │   namespace     │                │
│  │                 │    │                 │                │
│  │ • Frontend      │    │ • Distributor   │                │
│  │ • Backend       │    │ • Ingester      │                │
│  │                 │    │ • Querier       │                │
│  │                 │    │ • Compactor     │                │
│  │                 │    │ • Ruler         │                │
│  │                 │    │ • Alertmanager  │                │
│  │                 │    │ • Store Gateway │                │
│  └─────────────────┘    └─────────────────┘                │
│                                                             │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │   tenant-prod   │    │  tenant-staging │                │
│  │   namespace     │    │   namespace     │                │
│  └─────────────────┘    └─────────────────┘                │
│                                                             │
│  ┌─────────────────┐                                       │
│  │   tenant-dev    │                                       │
│  │   namespace     │                                       │
│  └─────────────────┘                                       │
└─────────────────────────────────────────────────────────────┘
```

### CI/CD Workflow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Local Dev     │    │  GitHub Actions  │    │   Local Deploy  │
│                 │    │                  │    │                 │
│ • Code Changes  │───▶│ • Build React    │───▶│ • Update Values │
│ • Git Push      │    │ • Build Go       │    │ • Deploy to     │
│                 │    │ • Multi-arch     │    │   Kind Cluster  │
│                 │    │ • Push to GHCR   │    │ • Port Forward  │
│                 │    │ • Timestamp Tags │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 🎯 Key Features

### 🧠 AI-Based Limit Suggestion
- Analyzes 48h, 30d, 60d metrics trend per tenant
- Calculates observed peaks and buffers by 10–20%
- Suggests updated limit values for supported Mimir limits
- Highlights under-provisioned and missing values

### 🔍 Auto-Discovery Engine
- Discovers Mimir components (Deployments, PVCs, ConfigMaps)
- Identifies tenant namespaces and their Alloy/NGINX/Consul setup
- Parses Alloy scrape endpoints, DNS rules, NGINX upstream config

### 📈 Metrics Analyzer
- Pulls metrics from Mimir Prometheus endpoints
- Computes trends for ingestion rates, rejections, active series, memory usage
- Per-tenant and per-limit type analysis

### 📊 UI Dashboard
- Health overview of pods, PVCs, and components
- Visual metrics pipeline representation
- Side-by-side current vs observed vs recommended limits
- Tenant configuration audit views

## 🚀 Quick Start

### Prerequisites
- Docker with Buildx support
- kubectl
- helm
- kind cluster (for local development)
- jq (for JSON parsing)

### 1. Setup Local Environment

```bash
# Create kind cluster
kind create cluster --name mimir-insights

# Verify cluster is running
kubectl cluster-info
```

### 2. Deploy Mimir Production Stack

```bash
# Deploy Mimir production stack and services
kubectl apply -f mimir-production-stack.yaml
kubectl apply -f mimir-services.yaml

# Verify Mimir is running
kubectl get pods -n mimir
```

### 3. Deploy MimirInsights

```bash
# Deploy MimirInsights to interact with Mimir
./deploy-local.sh
```

### 4. Access the Application

```bash
# Check status of all components
./check-status.sh

# Access URLs:
# Frontend: http://localhost:8081
# Backend API: http://localhost:8080/api/tenants
# Mimir API: http://localhost:9009/api/v1/status/buildinfo
```

## 📁 Project Structure

```
mimirInsights/
├── .github/workflows/     # CI/CD pipeline
│   └── build-multiarch.yml
├── cmd/                   # Application entry points
├── pkg/                   # Core packages
│   ├── discovery/         # Auto-discovery engine
│   ├── metrics/           # Metrics analysis
│   ├── limits/            # Limit recommendations
│   ├── drift/             # Configuration drift detection
│   ├── planner/           # Capacity planning
│   ├── llm/               # LLM integration
│   ├── k8s/               # Kubernetes client
│   └── api/               # REST API handlers
├── web-ui/                # React frontend
├── deployments/           # Deployment manifests
│   └── helm-chart/        # Helm chart
├── build-multi-arch.sh    # Multi-architecture build script
├── deploy-local.sh        # Local deployment script
├── check-status.sh        # Status monitoring script
├── Dockerfile.backend     # Backend Dockerfile
├── Dockerfile.frontend    # Frontend Dockerfile
├── mimir-production-stack.yaml  # Mimir production stack
├── mimir-services.yaml    # Mimir services
└── README.md              # This file
```

## 🔧 CI/CD Workflow

### Building Images

The CI/CD pipeline builds multi-architecture images in GitHub Actions:

```bash
# Trigger CI/CD pipeline by pushing to main/develop
git push origin main

# Or manually trigger workflow dispatch
# Go to GitHub Actions → Build Multi-Architecture Docker Images → Run workflow
```

### Image Naming Convention
- **Frontend**: `ghcr.io/akshaydubey29/mimir-insights-frontend`
- **Backend**: `ghcr.io/akshaydubey29/mimir-insights-backend`
- **Tags**: Timestamp-based (e.g., `20250719-141012`) for production deployments

### Deploying from CI/CD

After the CI/CD pipeline completes:

```bash
# Update values file and deploy with timestamp
./deploy-from-ci.sh 20250719-141012

# Or just update values file
./update-values.sh 20250719-141012
```

### Manual CI Trigger
1. Go to GitHub Actions tab
2. Select "Build Multi-Architecture Docker Images"
3. Click "Run workflow"
4. Enter version (e.g., "v1.0.0")

## 📊 API Endpoints

| Endpoint | Description |
|----------|-------------|
| `/api/limits` | Fetch current + recommended limits |
| `/api/tenants` | List discovered tenants |
| `/api/config` | Dump Mimir + Alloy config |
| `/api/health` | Health check |
| `/metrics` | Prometheus metrics |
| `/dashboard` | UI dashboard |

## 🔐 Security

- RBAC configured for read-only access to monitored namespaces
- ServiceAccount with minimal required permissions
- Images built with non-root users
- Read-only root filesystems
- Dropped capabilities

## 🐳 Container Images

All images are hosted at `ghcr.io/akshaydubey29`
- `mimir-insights-frontend:{timestamp}`
- `mimir-insights-backend:{timestamp}`

## 📈 Monitoring

The application exposes Prometheus metrics at `/metrics` for monitoring its own health and performance.

## 📋 Useful Commands

```bash
# Check overall status
./check-status.sh

# View MimirInsights logs
kubectl logs -f -l app.kubernetes.io/name=mimir-insights -n mimir-insights

# View Mimir logs
kubectl logs -f -l app.kubernetes.io/part-of=mimir -n mimir

# Port forward Mimir API
kubectl port-forward -n mimir svc/mimir-api 9009:9009

# Uninstall MimirInsights
helm uninstall mimir-insights -n mimir-insights
```

## 📄 Documentation

- [CI/CD Workflow](CI_CD_WORKFLOW.md) - Detailed CI/CD pipeline documentation
- [Production Simulation](PRODUCTION_SIMULATION_READY.md) - Production environment setup
- [Project Summary](PROJECT_SUMMARY.md) - Comprehensive project overview

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test locally using the production simulation environment
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.

## 🆘 Support

For issues and questions:
- GitHub Issues: [Create an issue](https://github.com/akshaydubey29/mimirInsights/issues)
- Documentation: [Project Wiki](https://github.com/akshaydubey29/mimirInsights/wiki)
