# MimirInsights Architecture

## Overview

MimirInsights is a Kubernetes-native, AI-driven application designed to automatically discover, analyze, audit, and optimize per-tenant limit configurations for Grafana Mimir in multi-tenant observability environments.

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    MimirInsights Platform                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   React Frontend │  │   Go Backend    │  │   Helm Chart    │  │
│  │                 │  │                 │  │                 │  │
│  │ • Dashboard     │  │ • API Server    │  │ • K8s Resources │  │
│  │ • Tenant Views  │  │ • Discovery     │  │ • RBAC          │  │
│  │ • Limit Config  │  │ • Metrics       │  │ • Ingress       │  │
│  │ • Reports       │  │ • Analysis      │  │ • HPA           │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                           │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   Mimir Namespace│  │ Tenant Namespaces│  │ MimirInsights   │  │
│  │                 │  │                 │  │ Namespace       │  │
│  │ • Distributor   │  │ • Alloy         │  │ • Backend Pods  │  │
│  │ • Ingester      │  │ • Consul        │  │ • Frontend Pods │  │
│  │ • Querier       │  │ • NGINX         │  │ • Services      │  │
│  │ • Compactor     │  │ • App Metrics   │  │ • Ingress       │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

###1 Backend (Go)

The backend is built in Go and provides the core functionality:

#### Core Packages

- **`pkg/config`**: Configuration management with environment variable support
- **`pkg/discovery`**: Auto-discovery engine for Mimir and tenant components
- **`pkg/metrics`**: Metrics client for querying Mimir Prometheus endpoints
- **`pkg/limits`**: AI-driven limit analysis and recommendations
- **`pkg/k8s`**: Kubernetes client for cluster interactions
- **`pkg/api`**: REST API server with Gin framework

#### Key Features

- **Auto-Discovery**: Discovers Mimir components and tenant namespaces
- **Metrics Analysis**: Queries Mimir for tenant-specific metrics
- **Limit Recommendations**: AI-driven analysis with 10-20% safety buffers
- **Configuration Audit**: Identifies missing or misconfigured limits
- **Prometheus Metrics**: Exposes application metrics for monitoring

### 2Frontend (React)

The frontend is built with React and Material-UI:

#### Key Components

- **Dashboard**: Overview of all tenants and system health
- **Tenant Views**: Per-tenant analysis and configuration
- **Limit Management**: Side-by-side current vs recommended limits
- **Reports**: Capacity planning and audit reports
- **Configuration**: System settings and feature toggles

#### Features

- **Dark Mode**: Modern dark theme for better UX
- **Real-time Updates**: Live data refresh and WebSocket support
- **Responsive Design**: Works on desktop and mobile devices
- **Export Capabilities**: CSV and JSON export for reports

### 3. Kubernetes Deployment

#### Helm Chart Structure

```
deployments/helm-chart/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default configuration
├── templates/
│   ├── namespace.yaml      # Namespace creation
│   ├── rbac.yaml          # RBAC configuration
│   ├── deployment.yaml    # Backend deployment
│   ├── service.yaml       # Service definitions
│   ├── ingress.yaml       # Ingress configuration
│   ├── hpa.yaml          # Horizontal Pod Autoscaler
│   └── _helpers.tpl      # Template helpers
```

#### Key Features

- **Multi-Component**: Separate deployments for backend and frontend
- **RBAC**: Read-only access to monitored namespaces
- **HPA**: Automatic scaling based on CPU/memory usage
- **Ingress**: ALB-based ingress with TLS termination
- **Security**: Non-root containers and security contexts

## Data Flow

###1. Discovery Flow

```
1. Application starts
2. Discovery engine queries K8s API
3. Identifies Mimir namespace components
4. Discovers tenant namespaces5 Parses ConfigMaps and configurations
6. Updates internal state
```

###2Metrics Flow

```1ics client queries Mimir API
2. Executes Prometheus queries for each tenant
3. Collects data over multiple time ranges (48h, 7d, 30604ulates peak values and trends5Stores results for analysis
```

### 3nalysis Flow

```
1analyzer processes tenant metrics
2. Compares current limits with observed peaks3ies safety buffers (10-204ulates risk levels and recommendations
5tes human-readable explanations
```

### 4. API Flow

```1Frontend makes API requests
2. Backend processes requests with middleware
3. Queries discovery, metrics, and analysis engines
4. Returns structured JSON responses
5. Frontend renders data in UI components
```

## Security Architecture

###1. Authentication & Authorization

- **RBAC**: Kubernetes Role-Based Access Control
- **Service Accounts**: Dedicated service accounts with minimal permissions
- **Namespace Isolation**: Separate namespace for MimirInsights
- **Read-Only Access**: No mutations to tenant namespaces by default

### 2. Network Security

- **Ingress TLS**: HTTPS termination at ALB
- **Internal Communication**: Service-to-service communication within cluster
- **Network Policies**: Optional network policies for additional isolation

### 3. Container Security

- **Non-Root**: Containers run as non-root users
- **Security Contexts**: Pod and container security contexts
- **Image Scanning**: Trivy vulnerability scanning in CI/CD
- **Minimal Base Images**: Alpine Linux for smaller attack surface

## Monitoring & Observability

### 1. Application Metrics

The application exposes Prometheus metrics:

- **Request Counters**: API endpoint request counts
- **Request Duration**: Response time histograms
- **Error Counters**: Error tracking by type
- **Business Metrics**: Tenant discovery, analysis success rates

### 2. Health Checks

- **Liveness Probe**: `/api/health` endpoint
- **Readiness Probe**: Dependency checks
- **Startup Probe**: Initial startup validation

### 3ng

- **Structured Logging**: JSON format with correlation IDs
- **Log Levels**: Configurable verbosity
- **Centralized Logging**: Integration with cluster logging

## Scalability

### 1. Horizontal Scaling

- **HPA**: Automatic scaling based on resource usage
- **Multiple Replicas**: Configurable replica counts
- **Load Balancing**: Service-level load balancing

### 2. Performance Optimization

- **Caching**: React Query for frontend caching
- **Connection Pooling**: HTTP client connection reuse
- **Async Processing**: Non-blocking API operations
- **Resource Limits**: CPU and memory limits

### 3 Data Management

- **Stateless Design**: No persistent storage requirements
- **Configuration**: Environment-based configuration
- **Caching Strategy**: In-memory caching for discovery results

## Deployment Architecture

###1. CI/CD Pipeline

```
GitHub Actions Workflow:1 Push/PR → Trigger Pipeline
2. Run Tests → Validate Code3ld Images → Docker Build
4. Push Images → Container Registry
5. Deploy → Helm Upgrade
6an → Vulnerability Check
```

### 2. Environment Strategy

- **Development**: Local development with hot reload
- **Staging**: Pre-production testing environment
- **Production**: Live environment with monitoring

### 3. Rollback Strategy

- **Helm Rollback**: Quick rollback to previous versions
- **Image Tags**: Immutable image tags for versioning
- **Health Checks**: Automatic rollback on health check failures

## Integration Points

### 1. Mimir Integration

- **Prometheus API**: Query metrics from Mimir
- **Tenant Isolation**: Multi-tenant metric queries
- **Configuration**: Runtime-overrides ConfigMap parsing

### 2. Kubernetes Integration

- **API Server**: Direct K8cess
- **Discovery**: Automatic resource discovery
- **RBAC**: Permission-based access control

###3ernal Integrations

- **LLM Providers**: OpenAI GPT-4 integration (optional)
- **GitOps**: Git-based configuration management
- **Monitoring**: Prometheus/Grafana integration

## Future Enhancements

### 1. Planned Features

- **Drift Detection**: Configuration drift monitoring
- **Capacity Planning**: Advanced capacity forecasting
- **GitOps Integration**: Automated PR creation
- **LLM Assistant**: Natural language querying

### 2. Scalability Improvements

- **Microservices**: Service decomposition
- **Event-Driven**: Event sourcing architecture
- **Distributed Tracing**: OpenTelemetry integration
- **Advanced Caching**: Redis-based caching

### 3. Security Enhancements

- **OIDC Integration**: OpenID Connect authentication
- **Network Policies**: Advanced network isolation
- **Secret Management**: External secrets integration
- **Audit Logging**: Comprehensive audit trails 