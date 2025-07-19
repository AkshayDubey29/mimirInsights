# Memory Management System

## Overview

The MimirInsights application implements a comprehensive, production-grade memory management system designed to prevent unbounded memory footprint growth during deployments. This system ensures precise control over cache memory usage while maintaining optimal performance for production-grade clusters with potentially limitless tenants, limits, and configurations.

## Architecture

### Backend Memory Management

#### Core Components

1. **MemoryManager** (`pkg/cache/memory_manager.go`)
   - Precise memory limits and monitoring
   - Configurable eviction policies
   - Real-time memory statistics
   - Automatic memory threshold management

2. **Cache Manager** (`pkg/cache/manager.go`)
   - Integration with MemoryManager
   - Background data collection with memory checks
   - Cache TTL management
   - Memory-aware data storage

3. **API Endpoints** (`pkg/api/server.go`)
   - `/api/cache/memory` - Get memory statistics
   - `/api/cache/memory/history` - Get memory usage history
   - `/api/cache/memory/evict` - Force memory eviction
   - `/api/cache/memory/reset` - Reset memory statistics
   - `/api/cache/memory/settings` - Update memory settings

#### Memory Management Features

```go
// Memory limits configuration
type MemoryManager struct {
    maxMemoryBytes     int64  // Maximum memory usage in bytes
    maxCacheSize       int    // Maximum total cache items
    maxTenantCacheSize int    // Maximum tenant cache items
    maxMimirCacheSize  int    // Maximum Mimir cache items
    
    // Monitoring and thresholds
    memoryThreshold     float64 // Warning threshold (80%)
    evictionThreshold   float64 // Eviction threshold (90%)
    evictionPolicy      EvictionPolicy // LRU, LFU, TTL, Size, Hybrid
}
```

#### Eviction Policies

1. **LRU (Least Recently Used)**
   - Evicts least recently accessed items
   - Best for temporal locality patterns

2. **LFU (Least Frequently Used)**
   - Evicts least frequently accessed items
   - Best for stable access patterns

3. **TTL (Time To Live)**
   - Evicts items based on expiration time
   - Best for time-sensitive data

4. **Size-based**
   - Evicts largest items first
   - Best for memory-constrained environments

5. **Hybrid**
   - Combines multiple policies
   - Adaptive based on usage patterns

### Frontend Memory Management

#### Components

1. **MemoryManagement Page** (`web-ui/src/pages/MemoryManagement.tsx`)
   - Real-time memory statistics display
   - Memory usage charts and visualizations
   - Cache distribution analysis
   - Memory control operations

2. **Memory API Hooks** (`web-ui/src/api/useMemory.ts`)
   - React Query integration for efficient data fetching
   - Real-time updates with configurable intervals
   - Error handling and retry logic
   - Optimistic updates for better UX

3. **DataGrid Component** (`web-ui/src/components/DataGridWithPagination.tsx`)
   - Production-grade data handling
   - Efficient pagination for large datasets
   - Advanced filtering and sorting
   - Memory-aware rendering

## Memory Management Strategy

### 1. Precise Memory Limits

```go
// Automatic memory limit calculation
func NewMemoryManager() *MemoryManager {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    totalMemory := int64(m.Sys)
    maxMemoryBytes := totalMemory / 4 // Use 25% of system memory
    
    // Ensure bounds
    if maxMemoryBytes < 100*1024*1024 { // 100MB minimum
        maxMemoryBytes = 100 * 1024 * 1024
    }
    if maxMemoryBytes > 2*1024*1024*1024 { // 2GB maximum
        maxMemoryBytes = 2 * 1024 * 1024 * 1024
    }
}
```

### 2. Real-time Monitoring

- **Memory Usage Tracking**: Continuous monitoring of current memory usage
- **Peak Memory Recording**: Track highest memory usage for analysis
- **Cache Item Counting**: Monitor number of cached items by type
- **Eviction Statistics**: Track eviction frequency and patterns

### 3. Proactive Eviction

```go
func (mm *MemoryManager) CheckMemoryUsage() error {
    usagePercent := float64(mm.currentMemoryBytes) / float64(mm.maxMemoryBytes)
    
    if usagePercent >= mm.evictionThreshold {
        return mm.triggerEviction()
    } else if usagePercent >= mm.memoryThreshold {
        mm.memoryWarnings++
        logrus.Warnf("Memory warning: %.1f%% >= %.1f%%", 
            usagePercent*100, mm.memoryThreshold*100)
    }
    return nil
}
```

### 4. Memory-aware Data Collection

```go
func (m *Manager) collectAllData(ctx context.Context) error {
    // Check memory before collection
    if err := m.memoryManager.CheckMemoryUsage(); err != nil {
        logrus.Warnf("Memory check failed: %v", err)
    }
    
    // Estimate memory usage for new data
    estimatedSize := m.estimateCollectionSize()
    if canAdd, err := m.memoryManager.CanAddItem("collection", estimatedSize); !canAdd {
        logrus.Warnf("Cannot add collection data: %v", err)
        return err
    }
    
    // Proceed with collection
    // ...
}
```

## UI Enhancements for Production Scale

### 1. Efficient Data Handling

#### Virtual Scrolling
- Render only visible rows for large datasets
- Implement windowing for optimal performance
- Lazy load data as needed

#### Pagination Strategy
```typescript
// Efficient pagination with configurable page sizes
const paginatedData = useMemo(() => {
  if (!enablePagination) return filteredAndSortedData;
  const startIndex = page * rowsPerPage;
  return filteredAndSortedData.slice(startIndex, startIndex + rowsPerPage);
}, [filteredAndSortedData, page, rowsPerPage, enablePagination]);
```

#### Debounced Search
```typescript
const handleSearch = useCallback(
  debounce((event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
    setPage(0); // Reset to first page when searching
  }, 300),
  []
);
```

### 2. Memory Management UI

#### Real-time Statistics Display
```typescript
const { data: memoryStats, isLoading, refetch } = useMemoryStats();

// Auto-refresh every 30 seconds
{
  refetchInterval: 30000,
  staleTime: 10000,
  retry: 3,
  retryDelay: 1000,
}
```

#### Memory Usage Visualization
- Line charts showing memory usage over time
- Pie charts for cache distribution
- Progress bars for memory thresholds
- Color-coded status indicators

#### Control Operations
```typescript
const forceEvictionMutation = useForceMemoryEviction();
const resetStatsMutation = useResetMemoryStats();

// Optimistic updates
{
  onSuccess: () => {
    queryClient.invalidateQueries('memoryStats');
  },
}
```

### 3. Production-Grade Data Grid

#### Features
- **Dual View Modes**: Table and card views for different use cases
- **Advanced Filtering**: Multi-column filtering with complex criteria
- **Smart Sorting**: Multi-column sorting with visual indicators
- **Bulk Operations**: Select multiple items for batch operations
- **Export Capabilities**: CSV and JSON export for data analysis
- **Responsive Design**: Adapts to different screen sizes

#### Performance Optimizations
```typescript
// Memoized filtering and sorting
const filteredAndSortedData = useMemo(() => {
  let filtered = data;
  
  // Apply search filter
  if (searchTerm) {
    const searchLower = searchTerm.toLowerCase();
    filtered = filtered.filter((row) =>
      columns.some((column) => {
        if (!column.searchable) return false;
        const value = row[column.id];
        return String(value).toLowerCase().includes(searchLower);
      })
    );
  }
  
  // Apply sorting
  if (sortBy && enableSorting) {
    filtered = [...filtered].sort((a, b) => {
      // Efficient comparison logic
    });
  }
  
  return filtered;
}, [data, searchTerm, sortBy, sortOrder, columns, enableSorting]);
```

## Configuration

### Backend Configuration

```yaml
# Memory management settings
memory:
  max_memory_bytes: 1073741824  # 1GB
  max_cache_size: 1000
  max_tenant_cache_size: 500
  max_mimir_cache_size: 500
  memory_threshold: 0.8         # 80%
  eviction_threshold: 0.9       # 90%
  eviction_policy: "hybrid"
  memory_check_interval: 30s
```

### Frontend Configuration

```typescript
// Memory management settings
const memoryConfig = {
  refreshInterval: 30000,    // 30 seconds
  staleTime: 10000,         // 10 seconds
  retryAttempts: 3,
  retryDelay: 1000,
  chartUpdateInterval: 60000, // 1 minute
};
```

## Monitoring and Alerting

### Memory Metrics

1. **Current Memory Usage**: Real-time memory consumption
2. **Peak Memory Usage**: Highest recorded memory usage
3. **Cache Hit Rate**: Efficiency of cache utilization
4. **Eviction Rate**: Frequency of cache evictions
5. **Memory Warnings**: Number of threshold warnings

### Alerting Rules

```yaml
# Memory alerting configuration
alerts:
  memory_usage_high:
    threshold: 80%
    severity: warning
    message: "Memory usage is high"
  
  memory_usage_critical:
    threshold: 90%
    severity: critical
    message: "Memory usage is critical"
  
  eviction_frequency_high:
    threshold: 10/minute
    severity: warning
    message: "High eviction frequency detected"
```

## Best Practices

### 1. Memory Management

- **Set Appropriate Limits**: Configure memory limits based on system resources
- **Monitor Regularly**: Check memory usage patterns and adjust settings
- **Use Appropriate Eviction Policies**: Choose policies based on data access patterns
- **Implement Graceful Degradation**: Handle memory pressure gracefully

### 2. UI Performance

- **Implement Virtual Scrolling**: For datasets with thousands of items
- **Use Debounced Search**: Prevent excessive API calls during typing
- **Optimize Re-renders**: Use React.memo and useMemo for expensive operations
- **Lazy Load Components**: Load heavy components only when needed

### 3. Data Handling

- **Implement Efficient Pagination**: Use cursor-based pagination for large datasets
- **Cache Strategically**: Cache frequently accessed data, not everything
- **Use Compression**: Compress data when possible to reduce memory usage
- **Implement Data Archiving**: Move old data to cheaper storage

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   - Check eviction policy effectiveness
   - Review cache size limits
   - Analyze memory usage patterns

2. **Slow UI Performance**
   - Implement virtual scrolling
   - Optimize component rendering
   - Use efficient data structures

3. **Frequent Evictions**
   - Adjust memory limits
   - Review eviction policy
   - Optimize data collection frequency

### Debugging Tools

1. **Memory Statistics API**: `/api/cache/memory`
2. **Memory History API**: `/api/cache/memory/history`
3. **Force Eviction API**: `/api/cache/memory/evict`
4. **Memory Settings API**: `/api/cache/memory/settings`

## Future Enhancements

### Planned Features

1. **Predictive Memory Management**
   - Machine learning-based memory usage prediction
   - Proactive eviction based on usage patterns

2. **Advanced Analytics**
   - Memory usage trend analysis
   - Performance impact assessment
   - Capacity planning recommendations

3. **Distributed Memory Management**
   - Multi-node memory coordination
   - Load balancing across instances

4. **Enhanced UI Features**
   - Real-time memory usage alerts
   - Interactive memory usage charts
   - Advanced filtering and search capabilities

This memory management system ensures that MimirInsights can handle production-grade clusters with limitless tenants, limits, and configurations while maintaining optimal performance and preventing memory-related issues. 