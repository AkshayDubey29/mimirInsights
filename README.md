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

## 🏗️ Architecture

### Metrics Flow Overview

```
[Application Clusters] 
     └─> [Tenant Namespace: transportation, eats, etc.]
           ├── Alloy (scrapes metrics from app targets)
           │     └─> Pulls from Consul-registered endpoints
           │     └─> Pushes to local NGINX
           └── NGINX (forwards metrics to)
                  └─> Mimir Distributor in namespace `mimir`
```

### MimirInsights System Flow

```
[Dedicated Namespace: mimir-insights]
     ├── Backend (Go-based API)
     ├── Frontend (React UI)
     ├── Analyzer (Metrics logic + limit recommendations)
     ├── Auto-discovery engine
     └── Metrics API clients (for Mimir and Kubernetes)
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
- Kubernetes cluster with Mimir deployed
- Helm 3.x
- kubectl configured

### Installation
1. **Add the Helm repository:**
```bash
helm repo add mimir-insights https://ghcr.io/akshaydubey29/mimir-insights
```

2. **Install MimirInsights:**
```bash
helm install mimir-insights mimir-insights/mimir-insights \
  --namespace mimir-insights \
  --create-namespace \
  --set mimirNamespace=mimir
```

3. **Access the dashboard:**
```bash
kubectl port-forward -n mimir-insights svc/mimir-insights-ui 8080:80
```

Then visit `http://localhost:8080`

## Project Structure

```
mimir-insights/
├── cmd/                    # Application entry points
├── pkg/                    # Core packages
│   ├── discovery/          # Auto-discovery engine
│   ├── metrics/            # Metrics analysis
│   ├── limits/             # Limit recommendations
│   ├── drift/              # Configuration drift detection
│   ├── planner/            # Capacity planning
│   ├── llm/                # LLM integration
│   ├── k8s/                # Kubernetes client
│   └── api/                # REST API handlers
├── web-ui/                 # React frontend
│   ├── pages/              # UI pages
│   ├── components/         # Reusable components
│   ├── services/           # API services
│   └── hooks/              # React hooks
├── deployments/            # Deployment manifests
│   └── helm-chart/         # Helm chart
├── docs/                   # Documentation
└── Makefile                # Build automation
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MIMIR_NAMESPACE` | Mimir namespace to monitor | `mimir` |
| `MIMIR_API_URL` | Mimir API endpoint | `http://mimir-distributor:9090` |
| `K8S_CLUSTER_URL` | Kubernetes API URL | Auto-detected |
| `LOG_LEVEL` | Logging level | `info` |

### Helm Values

See `deployments/helm-chart/values.yaml` for complete configuration options.

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
- Ingress with TLS termination
- No mutations to tenant namespaces unless explicitly configured

## 🐳 Container Images

All images are hosted at `ghcr.io/akshaydubey29`
- `mimir-insights-backend:latest`
- `mimir-insights-frontend:latest`
- `mimir-insights-analyzer:latest`

## 📈 Monitoring

The application exposes Prometheus metrics at `/metrics` for monitoring its own health and performance.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.

## 🆘 Support

For issues and questions:
- GitHub Issues: [Create an issue](https://github.com/akshaydubey29/mimirInsights/issues)
- Documentation: [Project Wiki](https://github.com/akshaydubey29/mimirInsights/wiki)
