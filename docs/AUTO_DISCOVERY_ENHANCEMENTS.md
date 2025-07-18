# Auto-Discovery Enhancements for MimirInsights

## Overview

This document describes the comprehensive enhancements made to the MimirInsights auto-discovery system to intelligently detect Mimir deployments and components without manual configuration.

## Key Enhancements Implemented

### 1. Configurable Mimir Namespace in values.yaml

**Problem**: The Mimir namespace was hardcoded, making the application less flexible for different deployment scenarios.

**Solution**: Enhanced `values.yaml` with comprehensive Mimir configuration:

```yaml
# Mimir configuration - Where the actual Mimir deployment is running
mimir:
  # Namespace where Mimir is deployed (auto-discovered if empty)
  namespace: ""
  # Auto-discovery configuration
  discovery:
    # Enable automatic Mimir namespace detection
    autoDetect: true
    # Namespace patterns to search for Mimir (regex supported)
    namespacePatterns:
      - "mimir.*"
      - ".*mimir.*"
      - "cortex.*"
      - ".*cortex.*"
      - "observability.*"
      - "monitoring.*"
    # Label selectors to identify Mimir namespaces
    namespaceLabels:
      - key: "app.kubernetes.io/name"
        values: ["mimir", "cortex"]
      - key: "app.kubernetes.io/part-of"
        values: ["mimir", "cortex", "observability"]
    # Component discovery patterns
    componentPatterns:
      distributor: [".*distributor.*", ".*dist.*"]
      ingester: [".*ingester.*", ".*ingest.*"]
      querier: [".*querier.*", ".*query.*", ".*frontend.*"]
      compactor: [".*compactor.*", ".*compact.*"]
      ruler: [".*ruler.*", ".*rule.*"]
      alertmanager: [".*alertmanager.*", ".*alert.*"]
      store_gateway: [".*store.*gateway.*", ".*gateway.*"]
    # Service discovery patterns
    servicePatterns:
      - "mimir-.*"
      - "cortex-.*"
      - ".*-mimir-.*"
      - ".*-cortex-.*"
    # ConfigMap patterns
    configMapPatterns:
      - ".*mimir.*config.*"
      - ".*cortex.*config.*"
      - ".*runtime.*overrides.*"
      - ".*limits.*config.*"
  # API configuration
  api:
    # Service name pattern for distributor (auto-discovered if empty)
    distributorService: ""
    # Default port for Mimir components
    port: 9090
    # Timeout for API calls
    timeout: 30
    # Paths to try for metrics endpoints
    metricsPaths:
      - "/metrics"
      - "/api/v1/query"
      - "/prometheus/api/v1/query"
```

### 2. Enhanced Discovery Engine with Regex Patterns

**Problem**: The original discovery relied on simple string matching, missing many valid Mimir components.

**Solution**: Implemented comprehensive pattern matching:

- **Pre-compiled regex patterns** for performance
- **Multiple component types** with flexible naming patterns
- **Label and annotation matching** for comprehensive discovery
- **Service and ConfigMap correlation** for validation

Key features:
- Namespace pattern matching with regex support
- Component type detection using multiple naming conventions
- Cross-reference validation using services, ConfigMaps, and deployments

### 3. Multi-Validation Approach

**Problem**: Single-source validation could lead to false positives or missed components.

**Solution**: Implemented cross-validation using multiple approaches:

#### Validation Sources:
1. **Deployment/StatefulSet Analysis**: Names, labels, annotations, images
2. **Service Correlation**: Related services and endpoints  
3. **ConfigMap Cross-Reference**: Configuration mentions and relationships
4. **Metrics Endpoint Validation**: Accessible metrics endpoints
5. **Port Pattern Matching**: Expected ports for component types

#### Confidence Scoring:
- Each validation source contributes to a confidence score
- Components with higher confidence scores are prioritized
- Detailed validation information tracked for debugging

```go
type ValidationResult struct {
    ConfidenceScore float64            `json:"confidence_score"`
    MatchedBy       []string           `json:"matched_by"`
    ValidationInfo  map[string]interface{} `json:"validation_info"`
}
```

### 4. Automatic Metrics Endpoint Discovery

**Problem**: Manual configuration of metrics endpoints was error-prone and limited.

**Solution**: Intelligent metrics endpoint discovery:

#### Discovery Methods:
1. **Service-based Discovery**: 
   - Scan all services in Mimir namespace
   - Identify metrics ports by port numbers and names
   - Generate cluster-internal endpoints

2. **Ingress-based Discovery**:
   - Scan ingresses for metrics paths
   - Support both HTTP and HTTPS schemes
   - Handle custom metrics endpoints

3. **Component Port Analysis**:
   - Common Mimir ports: 9090, 8080, 3100, 9093, 9094
   - Port name patterns: "metrics", "http-metrics", "prometheus"
   - Auto-generate endpoint URLs

#### Features:
- **Automatic endpoint generation**: `http://service.namespace.svc.cluster.local:port`
- **Multiple metrics paths**: `/metrics`, `/api/v1/query`, `/prometheus/api/v1/query`
- **Validation**: Check endpoint patterns for likelihood of serving metrics
- **New API endpoint**: `/api/metrics/discovery` to list discovered endpoints

### 5. Intelligent Namespace Auto-Discovery

**Problem**: Required manual specification of Mimir namespace.

**Solution**: Multi-criteria namespace evaluation:

#### Evaluation Criteria:
1. **Namespace name patterns** (regex-based)
2. **Namespace labels** with configurable selectors
3. **Mimir deployments count** in namespace
4. **Mimir services count** in namespace  
5. **Mimir ConfigMaps presence** in namespace

#### Scoring Algorithm:
- Namespace name match: +20 points
- Label match: +15 points per match
- Each Mimir deployment: +10 points
- Each Mimir service: +8 points
- Each Mimir ConfigMap: +5 points

The namespace with the highest confidence score is selected as the Mimir namespace.

### 6. Enhanced Component Discovery

**Problem**: Limited component type detection and metadata.

**Solution**: Comprehensive component analysis:

#### Enhanced MimirComponent Structure:
```go
type MimirComponent struct {
    Name             string                 `json:"name"`
    Type             string                 `json:"type"`
    Namespace        string                 `json:"namespace"`
    Status           string                 `json:"status"`
    Replicas         int32                  `json:"replicas"`
    Labels           map[string]string      `json:"labels"`
    Annotations      map[string]string      `json:"annotations"`
    Image            string                 `json:"image"`
    Version          string                 `json:"version"`
    ServiceEndpoints []string               `json:"service_endpoints"`
    MetricsEndpoints []string               `json:"metrics_endpoints"`
    ConfigMaps       []string               `json:"config_maps"`
    Validation       ValidationResult       `json:"validation"`
}
```

#### Component Type Detection:
- **Distributor**: `.*distributor.*`, `.*dist.*`
- **Ingester**: `.*ingester.*`, `.*ingest.*`
- **Querier**: `.*querier.*`, `.*query.*`, `.*frontend.*`
- **Compactor**: `.*compactor.*`, `.*compact.*`
- **Ruler**: `.*ruler.*`, `.*rule.*`
- **Alertmanager**: `.*alertmanager.*`, `.*alert.*`
- **Store Gateway**: `.*store.*gateway.*`, `.*gateway.*`

### 7. Configuration Structure Updates

**Problem**: Configuration structure didn't support the new discovery features.

**Solution**: Enhanced configuration with new types:

```go
type DiscoveryConfig struct {
    AutoDetect         bool                     `mapstructure:"auto_detect"`
    NamespacePatterns  []string                 `mapstructure:"namespace_patterns"`
    NamespaceLabels    []LabelSelector          `mapstructure:"namespace_labels"`
    ComponentPatterns  map[string][]string      `mapstructure:"component_patterns"`
    ServicePatterns    []string                 `mapstructure:"service_patterns"`
    ConfigMapPatterns  []string                 `mapstructure:"config_map_patterns"`
}

type LabelSelector struct {
    Key    string   `mapstructure:"key"`
    Values []string `mapstructure:"values"`
}

type APIConfig struct {
    DistributorService string   `mapstructure:"distributor_service"`
    Port               int      `mapstructure:"port"`
    Timeout            int      `mapstructure:"timeout"`
    MetricsPaths       []string `mapstructure:"metrics_paths"`
}
```

## API Enhancements

### New Endpoint: `/api/metrics/discovery`

Returns auto-discovered metrics endpoints:

```json
{
    "endpoints": [
        "http://mimir-distributor.mimir.svc.cluster.local:9090/metrics",
        "http://mimir-ingester.mimir.svc.cluster.local:9090/metrics",
        "http://mimir-querier.mimir.svc.cluster.local:9090/api/v1/query"
    ],
    "count": 3,
    "timestamp": "2025-07-17T18:20:00Z"
}
```

### Enhanced Environment Endpoint

The `/api/environment` endpoint now includes:
- Auto-discovered namespace information
- Enhanced component discovery results with validation scores
- Cross-referenced component relationships

## Benefits

### 1. Zero-Configuration Deployment
- Automatically detects Mimir namespace and components
- No manual endpoint configuration required
- Works across different Mimir deployment patterns

### 2. High Accuracy Discovery
- Multi-source validation reduces false positives
- Confidence scoring prioritizes reliable discoveries
- Comprehensive pattern matching catches edge cases

### 3. Flexible Configuration
- Regex-based patterns adapt to custom naming conventions
- Label selectors support various labeling strategies
- Configurable discovery settings via Helm values

### 4. Performance Optimized
- Pre-compiled regex patterns for fast matching
- Efficient Kubernetes API usage
- Caching of discovery results

### 5. Debugging Support
- Detailed validation information for troubleshooting
- Confidence scores help identify weak discoveries
- Comprehensive logging of discovery process

## Usage Examples

### Auto-Discovery Mode (Default)
```yaml
mimir:
  namespace: ""  # Auto-discover
  discovery:
    autoDetect: true
```

### Custom Patterns
```yaml
mimir:
  discovery:
    namespacePatterns:
      - "my-mimir-.*"
      - "prod-monitoring"
    componentPatterns:
      distributor: ["my-dist-.*", "distributor-.*"]
```

### Label-Based Discovery
```yaml
mimir:
  discovery:
    namespaceLabels:
      - key: "monitoring.io/system"
        values: ["mimir"]
      - key: "env"
        values: ["production", "staging"]
```

## Future Enhancements

1. **HTTP Endpoint Validation**: Actually ping metrics endpoints to verify availability
2. **Dynamic Pattern Learning**: Learn patterns from successful discoveries
3. **Multi-Cluster Support**: Discover Mimir across multiple Kubernetes clusters
4. **Advanced Correlation**: Use Grafana dashboards and Prometheus rules for validation
5. **Performance Metrics**: Track discovery accuracy and performance metrics

## Configuration Migration

For existing deployments, the enhanced discovery is backward compatible:
- If `mimir.namespace` is set, auto-discovery is skipped
- Default patterns match common Mimir deployments
- Existing functionality remains unchanged

The enhancements provide a solid foundation for intelligent, zero-configuration Mimir component discovery while maintaining flexibility for custom deployment scenarios. 