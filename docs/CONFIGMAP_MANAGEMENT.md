# ConfigMap Management for Tenant Limits

This document describes which ConfigMaps the MimirInsights backend modifies when editing tenant limits and how the auto-discovery system works.

## Overview

MimirInsights manages tenant limits through various Kubernetes ConfigMaps in the Mimir namespace. The system uses an auto-discovery mechanism to identify and parse limit configurations from multiple sources.

## ConfigMap Types and Patterns

### 1. Global Configuration ConfigMaps

#### `mimir-config` or `mimir`
- **Purpose**: Main Mimir configuration containing global limits
- **Location**: `mimir` namespace (configurable)
- **Content**: YAML configuration with global limit settings
- **Auto-discovery Pattern**: Exact name match for `mimir-config` or `mimir`

#### `runtime-overrides` or `mimir-runtime-overrides`
- **Purpose**: Runtime overrides for dynamic limit adjustments
- **Location**: `mimir` namespace (configurable)
- **Content**: YAML format with runtime limit overrides
- **Auto-discovery Pattern**: Exact name match for override ConfigMaps

### 2. Tenant-Specific ConfigMaps

The system auto-discovers tenant ConfigMaps using the following patterns:

#### Tenant ConfigMap Naming Patterns
- `tenant-{tenant-name}`
- `{tenant-name}-tenant`
- `{tenant-name}-limits`
- `{tenant-name}-overrides`
- `user-{tenant-name}`
- `org-{tenant-name}`

#### Examples
- `tenant-eats` - Limits for "eats" tenant
- `transportation-limits` - Limits for "transportation" tenant
- `user-delivery` - Limits for "delivery" tenant
- `marketplace-overrides` - Overrides for "marketplace" tenant

### 3. Monitoring Configuration ConfigMaps

#### Alloy/Grafana Agent ConfigMaps
- **Patterns**: `alloy`, `grafana-agent`, `agent-config`, `monitoring`, `scrape`
- **Purpose**: Monitoring and metrics collection configuration
- **Auto-discovery**: Pattern-based matching for monitoring-related names

## ConfigMap Data Structure

### Global Limits Format
```yaml
# In mimir-config ConfigMap
limits:
  max_global_series_per_user: 5000000
  ingestion_rate: 10000
  max_label_names_per_series: 30
  max_label_value_length: 2048
  ingestion_burst_size: 20000
```

### Runtime Overrides Format
```yaml
# In runtime-overrides ConfigMap
overrides:
  tenant-1:
    max_global_series_per_user: 2000000
    ingestion_rate: 5000
  tenant-2:
    max_global_series_per_user: 10000000
    ingestion_rate: 20000
```

### Tenant-Specific Format
```yaml
# In tenant-{name} ConfigMap
limits:
  max_global_series_per_user: 3000000
  ingestion_rate: 8000
  max_label_names_per_series: 25
```

## Auto-Discovery Process

### 1. ConfigMap Discovery
The auto-discovery system (`pkg/limits/auto_discovery.go`) performs the following steps:

1. **Scan Mimir Namespace**: Gets all ConfigMaps in the configured Mimir namespace
2. **Pattern Matching**: Identifies ConfigMaps using predefined patterns
3. **Content Parsing**: Parses YAML content to extract limit configurations
4. **Source Tracking**: Records source information for audit purposes

### 2. Discovery Order and Precedence
1. **Global Limits**: From `mimir-config` ConfigMap
2. **Runtime Overrides**: From `runtime-overrides` ConfigMap
3. **Tenant-Specific**: From tenant ConfigMaps (highest precedence)

### 3. Supported Limit Types
The system recognizes and manages these Mimir limit types:

- `max_global_series_per_user` - Maximum active series per tenant
- `max_label_names_per_series` - Maximum label names per series
- `max_label_value_length` - Maximum label value length
- `ingestion_rate` - Samples per second limit
- `ingestion_burst_size` - Burst size for ingestion
- `max_global_series_per_metric` - Series per metric limit
- `max_global_exemplars_per_user` - Exemplars per tenant
- `max_global_metadata_per_user` - Metadata entries per tenant
- `max_global_metadata_per_metric` - Metadata per metric
- `max_global_exemplars_per_metric` - Exemplars per metric

## Backend API Integration

### Configuration Sources Discovery
The backend exposes discovered ConfigMaps through:
- **GET /api/config** - Returns all discovered ConfigMaps
- **GET /api/environment** - Includes auto-discovered limits and sources

### Limit Analysis
The limits analyzer (`pkg/limits/analyzer.go`) uses auto-discovery to:
1. Get current tenant configurations
2. Analyze limit utilization
3. Generate recommendations
4. Track configuration sources

## ConfigMap Modification Workflow

### When Editing Tenant Limits

1. **Target ConfigMap Selection**:
   - First priority: Existing tenant-specific ConfigMap (`tenant-{name}`)
   - Second priority: Runtime overrides ConfigMap
   - Fallback: Create new tenant ConfigMap

2. **Modification Process**:
   - Read current ConfigMap content
   - Parse YAML structure
   - Update specific limit values
   - Preserve other configuration
   - Apply changes with proper formatting

3. **Validation**:
   - Validate YAML syntax
   - Check limit value ranges
   - Ensure tenant name consistency
   - Verify required fields

## Example ConfigMap Modifications

### Updating Ingestion Rate for "eats" Tenant

**Before** (`tenant-eats` ConfigMap):
```yaml
limits:
  max_global_series_per_user: 2000000
  ingestion_rate: 5000
```

**After** (increased ingestion rate):
```yaml
limits:
  max_global_series_per_user: 2000000
  ingestion_rate: 8000  # Updated
  max_label_names_per_series: 30  # Added new limit
```

### Creating New Tenant Limits

**New ConfigMap** (`tenant-delivery`):
```yaml
limits:
  max_global_series_per_user: 1500000
  ingestion_rate: 6000
  max_label_names_per_series: 25
  ingestion_burst_size: 12000
```

## Monitoring and Audit

### ConfigMap Change Tracking
- All ConfigMap modifications are logged
- Audit logs include source ConfigMap name
- Changes tracked with timestamps
- User attribution when available

### Discovery Status
- Real-time discovery of new/modified ConfigMaps
- Source validation and health checks
- Configuration drift detection
- Missing limit identification

## Best Practices

### 1. ConfigMap Naming
- Use consistent naming patterns
- Include tenant identifier clearly
- Avoid special characters
- Use lowercase names

### 2. Configuration Management
- Keep tenant-specific overrides minimal
- Use runtime overrides for temporary changes
- Document limit changes with annotations
- Regular backup of critical ConfigMaps

### 3. Limit Setting Guidelines
- Set conservative initial limits
- Monitor tenant usage patterns
- Adjust based on observed peaks
- Include safety buffers (10-20%)

## Troubleshooting

### Common Issues
1. **ConfigMap Not Discovered**: Check naming patterns and namespace
2. **Limits Not Applied**: Verify YAML syntax and Mimir restart
3. **Permission Errors**: Ensure proper RBAC for ConfigMap access
4. **Override Conflicts**: Check precedence order and source priority

### Debug Commands
```bash
# List all ConfigMaps in Mimir namespace
kubectl get configmaps -n mimir

# View specific tenant ConfigMap
kubectl get configmap tenant-eats -n mimir -o yaml

# Check runtime overrides
kubectl get configmap runtime-overrides -n mimir -o yaml
```

## API Endpoints

### Discovery Endpoints
- `GET /api/config` - All discovered ConfigMaps
- `GET /api/environment` - Environment with auto-discovery data
- `GET /api/audit` - ConfigMap modification audit logs

### Analysis Endpoints
- `GET /api/limits` - Current limits analysis
- `POST /api/analyze` - Tenant-specific analysis
- `GET /api/drift` - Configuration drift detection

---

**Note**: This documentation reflects the current auto-discovery patterns implemented in `pkg/limits/auto_discovery.go` and `pkg/discovery/engine.go`. The system is designed to be extensible for additional ConfigMap patterns as needed. 