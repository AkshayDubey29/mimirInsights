# Mimir Configuration Guide

This guide explains how to configure MimirInsights to discover and connect to your Mimir deployment.

## Mimir Namespace Configuration

MimirInsights supports two modes for discovering your Mimir deployment:

### 1. Auto-Discovery Mode (Recommended)

The application will automatically discover your Mimir namespace using intelligent pattern matching:

```yaml
mimir:
  namespace: "auto"  # or leave empty
  discovery:
    autoDetect: true
    namespacePatterns:
      - "mimir.*"
      - ".*mimir.*"
      - "cortex.*"
      - ".*cortex.*"
      - "observability.*"
      - "monitoring.*"
```

**How it works:**
- Scans all namespaces in your cluster
- Matches namespace names against patterns
- Checks for Mimir-specific labels
- Looks for Mimir components (distributor, ingester, etc.)
- Scores each namespace and selects the best match

### 2. Manual Configuration Mode

If you know your Mimir namespace, you can specify it directly:

```yaml
mimir:
  namespace: "mimir"  # or "cortex", "observability", etc.
  discovery:
    autoDetect: false
```

## Discovery Patterns

### Namespace Patterns
The application searches for namespaces matching these patterns:
- `mimir.*` - Namespaces starting with "mimir"
- `.*mimir.*` - Namespaces containing "mimir"
- `cortex.*` - Namespaces starting with "cortex"
- `.*cortex.*` - Namespaces containing "cortex"
- `observability.*` - Observability namespaces
- `monitoring.*` - Monitoring namespaces

### Component Patterns
The application identifies Mimir components using these patterns:

| Component | Patterns |
|-----------|----------|
| Distributor | `.*distributor.*`, `.*dist.*` |
| Ingester | `.*ingester.*`, `.*ingest.*` |
| Querier | `.*querier.*`, `.*query.*`, `.*frontend.*` |
| Compactor | `.*compactor.*`, `.*compact.*` |
| Ruler | `.*ruler.*`, `.*rule.*` |
| Alertmanager | `.*alertmanager.*`, `.*alert.*` |
| Store Gateway | `.*store.*gateway.*`, `.*gateway.*` |

### Label Selectors
The application also checks for specific labels:
- `app.kubernetes.io/name` with values: `mimir`, `cortex`
- `app.kubernetes.io/part-of` with values: `mimir`, `cortex`, `observability`

## API Configuration

### Auto-Discovery
```yaml
mimir:
  api:
    distributorService: ""  # Auto-discovered
    port: 9090
    timeout: 30
```

### Manual Configuration
```yaml
mimir:
  api:
    distributorService: "mimir-distributor"  # Your distributor service name
    port: 9090
    timeout: 30
```

## Environment Variables

The following environment variables are automatically set based on your configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `MIMIR_NAMESPACE` | Mimir namespace | `auto` |
| `MIMIR_API_URL` | Mimir API endpoint | `auto` |
| `MIMIR_AUTO_DISCOVER` | Enable auto-discovery | `true` |
| `K8S_IN_CLUSTER` | Run in Kubernetes cluster | `true` |
| `MIMIR_DISCOVERY_INTERVAL` | Discovery refresh interval | `300` |

## Configuration Examples

### Example 1: Standard Mimir Deployment
```yaml
mimir:
  namespace: "auto"
  discovery:
    autoDetect: true
  api:
    distributorService: ""
    port: 9090
```

### Example 2: Cortex Deployment
```yaml
mimir:
  namespace: "auto"
  discovery:
    autoDetect: true
    namespacePatterns:
      - "cortex.*"
      - ".*cortex.*"
  api:
    distributorService: ""
    port: 9090
```

### Example 3: Custom Namespace
```yaml
mimir:
  namespace: "observability"
  discovery:
    autoDetect: false
  api:
    distributorService: "mimir-distributor"
    port: 9090
```

## Troubleshooting

### 1. Namespace Not Found
If the application can't find your Mimir namespace:

```bash
# Check what namespaces exist
kubectl get namespaces

# Check if Mimir components are running
kubectl get pods -n <mimir-namespace>

# Check namespace labels
kubectl get namespace <mimir-namespace> -o yaml
```

### 2. Manual Override
If auto-discovery fails, manually specify the namespace:

```yaml
mimir:
  namespace: "your-mimir-namespace"
  discovery:
    autoDetect: false
```

### 3. Check Discovery Logs
```bash
# Check backend logs for discovery information
kubectl logs -f deployment/mimir-insights-backend -n mimir-insights
```

Look for messages like:
- "Auto-discovering Mimir namespace..."
- "Auto-discovered Mimir namespace: mimir (confidence: 0.85)"

### 4. Verify API Connection
```bash
# Test connection to Mimir API
kubectl exec -it deployment/mimir-insights-backend -n mimir-insights -- curl -f http://mimir-distributor:9090/metrics
```

## Best Practices

1. **Use Auto-Discovery**: Let the application find your Mimir deployment automatically
2. **Label Your Namespaces**: Add appropriate labels to your Mimir namespace for better discovery
3. **Monitor Logs**: Check the backend logs to see what namespace was discovered
4. **Test Connectivity**: Verify that the application can reach your Mimir API
5. **Fallback Configuration**: Have a manual configuration ready in case auto-discovery fails

## Common Namespace Names

The application is designed to work with common Mimir deployment patterns:

- `mimir` - Standard Mimir namespace
- `cortex` - Legacy Cortex namespace
- `observability` - Observability stack namespace
- `monitoring` - Monitoring stack namespace
- `grafana-mimir` - Grafana Mimir namespace
- `prometheus` - Prometheus-based deployments

## Configuration Validation

You can validate your configuration by checking the Helm template output:

```bash
helm template mimir-insights ./deployments/helm-chart \
  --values deployments/helm-chart/values-production-final.yaml \
  --namespace mimir-insights | grep -A 10 -B 5 "MIMIR_"
```

This will show you how the Mimir configuration is being applied to the deployment. 