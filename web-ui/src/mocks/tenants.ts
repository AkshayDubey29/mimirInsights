interface TenantMetrics {
  timestamp: string;
  ingestionRate: number;
  queryRate: number;
  seriesCount: number;
  samplesPerSecond: number;
  storageUsageGB: number;
  cpuUsage: number;
  memoryUsage: number;
  errorRate: number;
}

interface TenantConfiguration {
  maxGlobalSeriesPerUser: number;
  ingestionRate: number;
  maxLabelNamesPerSeries: number;
  maxMetadataPerUser: number;
  queryTimeout: string;
  maxQueryLength: string;
  maxQueryParallelism: number;
  maxOutstandingRequestsPerTenant: number;
}

interface TenantAlert {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  title: string;
  description: string;
  timestamp: string;
  resolved: boolean;
}

interface EnhancedTenant {
  id: string;
  name: string;
  namespace: string;
  status: 'healthy' | 'warning' | 'critical' | 'inactive';
  discoveredAt: string;
  lastSeen: string;
  metrics: TenantMetrics[];
  configuration: TenantConfiguration;
  alerts: TenantAlert[];
  components: {
    alloy: { replicas: number; healthy: number; status: string };
    distributors: { count: number; healthy: number };
    ingesters: { count: number; healthy: number };
    queriers: { count: number; healthy: number };
  };
  trends: {
    ingestionTrend: 'up' | 'down' | 'stable';
    errorTrend: 'up' | 'down' | 'stable';
    storageTrend: 'up' | 'down' | 'stable';
  };
  recommendations: Array<{
    type: 'optimization' | 'scaling' | 'configuration';
    title: string;
    impact: 'high' | 'medium' | 'low';
    description: string;
  }>;
}

export const tenants: EnhancedTenant[] = [
  {
    id: 'eats-tenant',
    name: 'eats',
    namespace: 'eats-production',
    status: 'critical',
    discoveredAt: '2024-07-15T08:00:00Z',
    lastSeen: '2024-07-17T19:30:00Z',
    metrics: [
      { timestamp: '19:25', ingestionRate: 174000, queryRate: 1200, seriesCount: 2800000, samplesPerSecond: 4500, storageUsageGB: 1250, cpuUsage: 0.85, memoryUsage: 0.78, errorRate: 0.015 },
      { timestamp: '19:26', ingestionRate: 176000, queryRate: 1250, seriesCount: 2820000, samplesPerSecond: 4600, storageUsageGB: 1252, cpuUsage: 0.87, memoryUsage: 0.80, errorRate: 0.018 },
      { timestamp: '19:27', ingestionRate: 178000, queryRate: 1300, seriesCount: 2850000, samplesPerSecond: 4700, storageUsageGB: 1255, cpuUsage: 0.89, memoryUsage: 0.82, errorRate: 0.022 },
      { timestamp: '19:28', ingestionRate: 180000, queryRate: 1350, seriesCount: 2880000, samplesPerSecond: 4800, storageUsageGB: 1258, cpuUsage: 0.91, memoryUsage: 0.84, errorRate: 0.025 },
      { timestamp: '19:29', ingestionRate: 182000, queryRate: 1400, seriesCount: 2900000, samplesPerSecond: 4900, storageUsageGB: 1260, cpuUsage: 0.93, memoryUsage: 0.86, errorRate: 0.028 },
    ],
    configuration: {
      maxGlobalSeriesPerUser: 3000000,
      ingestionRate: 150000,
      maxLabelNamesPerSeries: 30,
      maxMetadataPerUser: 8000,
      queryTimeout: '2m',
      maxQueryLength: '32KB',
      maxQueryParallelism: 14,
      maxOutstandingRequestsPerTenant: 100,
    },
    alerts: [
      { id: '1', severity: 'critical', title: 'Ingestion Rate Exceeded', description: 'Current rate 182K/s exceeds limit 150K/s', timestamp: '2024-07-17T19:29:00Z', resolved: false },
      { id: '2', severity: 'warning', title: 'High Memory Usage', description: 'Memory usage at 86% of allocated resources', timestamp: '2024-07-17T19:28:00Z', resolved: false },
      { id: '3', severity: 'warning', title: 'Error Rate Increasing', description: 'Error rate trending upward: 2.8%', timestamp: '2024-07-17T19:27:00Z', resolved: false },
    ],
    components: {
      alloy: { replicas: 6, healthy: 5, status: 'degraded' },
      distributors: { count: 4, healthy: 4 },
      ingesters: { count: 8, healthy: 7 },
      queriers: { count: 6, healthy: 6 },
    },
    trends: { ingestionTrend: 'up', errorTrend: 'up', storageTrend: 'up' },
    recommendations: [
      { type: 'scaling', title: 'Increase Ingestion Limits', impact: 'high', description: 'Recommend increasing ingestion rate limit to 200K/s with 20% buffer' },
      { type: 'optimization', title: 'Optimize Series Cardinality', impact: 'medium', description: 'High cardinality detected in payment metrics - consider label optimization' },
      { type: 'configuration', title: 'Increase Query Parallelism', impact: 'medium', description: 'Query latency could be improved by increasing parallelism to 20' },
    ],
  },
  {
    id: 'transportation-tenant',
    name: 'transportation',
    namespace: 'transportation-prod',
    status: 'warning',
    discoveredAt: '2024-07-15T08:00:00Z',
    lastSeen: '2024-07-17T19:30:00Z',
    metrics: [
      { timestamp: '19:25', ingestionRate: 95000, queryRate: 800, seriesCount: 1800000, samplesPerSecond: 2800, storageUsageGB: 850, cpuUsage: 0.65, memoryUsage: 0.58, errorRate: 0.008 },
      { timestamp: '19:26', ingestionRate: 96000, queryRate: 820, seriesCount: 1820000, samplesPerSecond: 2850, storageUsageGB: 852, cpuUsage: 0.66, memoryUsage: 0.59, errorRate: 0.009 },
      { timestamp: '19:27', ingestionRate: 94000, queryRate: 790, seriesCount: 1810000, samplesPerSecond: 2780, storageUsageGB: 851, cpuUsage: 0.64, memoryUsage: 0.57, errorRate: 0.007 },
      { timestamp: '19:28', ingestionRate: 97000, queryRate: 850, seriesCount: 1830000, samplesPerSecond: 2900, storageUsageGB: 853, cpuUsage: 0.67, memoryUsage: 0.60, errorRate: 0.010 },
      { timestamp: '19:29', ingestionRate: 98000, queryRate: 870, seriesCount: 1840000, samplesPerSecond: 2950, storageUsageGB: 855, cpuUsage: 0.68, memoryUsage: 0.61, errorRate: 0.011 },
    ],
    configuration: {
      maxGlobalSeriesPerUser: 2000000,
      ingestionRate: 120000,
      maxLabelNamesPerSeries: 25,
      maxMetadataPerUser: 6000,
      queryTimeout: '90s',
      maxQueryLength: '24KB',
      maxQueryParallelism: 12,
      maxOutstandingRequestsPerTenant: 80,
    },
    alerts: [
      { id: '4', severity: 'warning', title: 'Query Latency High', description: 'P95 query latency at 8.5s, above target of 5s', timestamp: '2024-07-17T19:25:00Z', resolved: false },
      { id: '5', severity: 'info', title: 'Storage Growth Rate', description: 'Storage growing at 2GB/day, within expected range', timestamp: '2024-07-17T19:20:00Z', resolved: true },
    ],
    components: {
      alloy: { replicas: 4, healthy: 4, status: 'healthy' },
      distributors: { count: 3, healthy: 3 },
      ingesters: { count: 6, healthy: 6 },
      queriers: { count: 4, healthy: 4 },
    },
    trends: { ingestionTrend: 'stable', errorTrend: 'stable', storageTrend: 'up' },
    recommendations: [
      { type: 'optimization', title: 'Optimize Query Performance', impact: 'high', description: 'Implement query result caching to reduce latency' },
      { type: 'configuration', title: 'Adjust Query Timeout', impact: 'low', description: 'Consider increasing query timeout to 2m for complex queries' },
    ],
  },
  {
    id: 'marketplace-tenant',
    name: 'marketplace',
    namespace: 'marketplace-production',
    status: 'healthy',
    discoveredAt: '2024-07-15T08:00:00Z',
    lastSeen: '2024-07-17T19:30:00Z',
    metrics: [
      { timestamp: '19:25', ingestionRate: 65000, queryRate: 450, seriesCount: 1200000, samplesPerSecond: 1800, storageUsageGB: 580, cpuUsage: 0.45, memoryUsage: 0.38, errorRate: 0.003 },
      { timestamp: '19:26', ingestionRate: 66000, queryRate: 460, seriesCount: 1210000, samplesPerSecond: 1820, storageUsageGB: 581, cpuUsage: 0.46, memoryUsage: 0.39, errorRate: 0.003 },
      { timestamp: '19:27', ingestionRate: 64000, queryRate: 440, seriesCount: 1205000, samplesPerSecond: 1780, storageUsageGB: 580, cpuUsage: 0.44, memoryUsage: 0.37, errorRate: 0.002 },
      { timestamp: '19:28', ingestionRate: 67000, queryRate: 470, seriesCount: 1215000, samplesPerSecond: 1850, storageUsageGB: 582, cpuUsage: 0.47, memoryUsage: 0.40, errorRate: 0.003 },
      { timestamp: '19:29', ingestionRate: 68000, queryRate: 480, seriesCount: 1220000, samplesPerSecond: 1870, storageUsageGB: 583, cpuUsage: 0.48, memoryUsage: 0.41, errorRate: 0.003 },
    ],
    configuration: {
      maxGlobalSeriesPerUser: 1500000,
      ingestionRate: 100000,
      maxLabelNamesPerSeries: 20,
      maxMetadataPerUser: 5000,
      queryTimeout: '60s',
      maxQueryLength: '16KB',
      maxQueryParallelism: 10,
      maxOutstandingRequestsPerTenant: 60,
    },
    alerts: [
      { id: '6', severity: 'info', title: 'Under-provisioned Resources', description: 'CPU usage consistently below 50% - consider scaling down', timestamp: '2024-07-17T19:00:00Z', resolved: false },
    ],
    components: {
      alloy: { replicas: 3, healthy: 3, status: 'healthy' },
      distributors: { count: 2, healthy: 2 },
      ingesters: { count: 4, healthy: 4 },
      queriers: { count: 3, healthy: 3 },
    },
    trends: { ingestionTrend: 'stable', errorTrend: 'down', storageTrend: 'stable' },
    recommendations: [
      { type: 'scaling', title: 'Optimize Resource Allocation', impact: 'medium', description: 'Resources are over-provisioned - consider reducing replicas by 25%' },
      { type: 'optimization', title: 'Enable Compaction', impact: 'low', description: 'Enable advanced compaction strategies for better storage efficiency' },
    ],
  },
  {
    id: 'analytics-tenant',
    name: 'analytics',
    namespace: 'analytics-prod',
    status: 'healthy',
    discoveredAt: '2024-07-16T10:30:00Z',
    lastSeen: '2024-07-17T19:30:00Z',
    metrics: [
      { timestamp: '19:25', ingestionRate: 45000, queryRate: 320, seriesCount: 800000, samplesPerSecond: 1200, storageUsageGB: 420, cpuUsage: 0.35, memoryUsage: 0.28, errorRate: 0.001 },
      { timestamp: '19:26', ingestionRate: 46000, queryRate: 330, seriesCount: 810000, samplesPerSecond: 1220, storageUsageGB: 421, cpuUsage: 0.36, memoryUsage: 0.29, errorRate: 0.001 },
      { timestamp: '19:27', ingestionRate: 44000, queryRate: 310, seriesCount: 805000, samplesPerSecond: 1180, storageUsageGB: 420, cpuUsage: 0.34, memoryUsage: 0.27, errorRate: 0.001 },
      { timestamp: '19:28', ingestionRate: 47000, queryRate: 340, seriesCount: 815000, samplesPerSecond: 1240, storageUsageGB: 422, cpuUsage: 0.37, memoryUsage: 0.30, errorRate: 0.001 },
      { timestamp: '19:29', ingestionRate: 48000, queryRate: 350, seriesCount: 820000, samplesPerSecond: 1260, storageUsageGB: 423, cpuUsage: 0.38, memoryUsage: 0.31, errorRate: 0.002 },
    ],
    configuration: {
      maxGlobalSeriesPerUser: 1000000,
      ingestionRate: 80000,
      maxLabelNamesPerSeries: 15,
      maxMetadataPerUser: 3000,
      queryTimeout: '45s',
      maxQueryLength: '12KB',
      maxQueryParallelism: 8,
      maxOutstandingRequestsPerTenant: 40,
    },
    alerts: [],
    components: {
      alloy: { replicas: 2, healthy: 2, status: 'healthy' },
      distributors: { count: 2, healthy: 2 },
      ingesters: { count: 3, healthy: 3 },
      queriers: { count: 2, healthy: 2 },
    },
    trends: { ingestionTrend: 'stable', errorTrend: 'stable', storageTrend: 'stable' },
    recommendations: [
      { type: 'optimization', title: 'Implement Long-term Storage', impact: 'low', description: 'Consider tiered storage for historical analytics data' },
    ],
  }
];

// Legacy compatibility export for basic tenant data
export const basicTenants = [
  {
    name: 'eats',
    namespace: 'eats-production',
    status: 'Critical',
    cpuUsage: 0.93,
    memoryUsage: 0.86,
    alloyReplicas: 6,
  },
  {
    name: 'transportation',
    namespace: 'transportation-prod',
    status: 'Warning',
    cpuUsage: 0.68,
    memoryUsage: 0.61,
    alloyReplicas: 4,
  },
  {
    name: 'marketplace',
    namespace: 'marketplace-production',
    status: 'Healthy',
    cpuUsage: 0.48,
    memoryUsage: 0.41,
    alloyReplicas: 3,
  },
  {
    name: 'analytics',
    namespace: 'analytics-prod',
    status: 'Healthy',
    cpuUsage: 0.38,
    memoryUsage: 0.31,
    alloyReplicas: 2,
  },
]; 