import { useState, useEffect } from 'react';

interface Tenant {
  id: string;
  name: string;
  namespace: string;
  status: 'healthy' | 'warning' | 'critical' | 'inactive';
  discoveredAt: string;
  lastSeen: string;
  has_real_data: boolean;
}

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

export interface EnhancedTenant extends Tenant {
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

interface UseTenantsReturn {
  data: Tenant[] | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

interface UseEnhancedTenantsReturn {
  data: EnhancedTenant[] | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export const useTenants = (): UseTenantsReturn => {
  const [data, setData] = useState<Tenant[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/tenants');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      // Transform the data to match the expected format
      const transformedData: Tenant[] = result.tenants?.map((tenant: any) => ({
        id: tenant.name,
        name: tenant.name,
        namespace: tenant.namespace,
        status: tenant.status || 'inactive',
        discoveredAt: tenant.discovered_at || new Date().toISOString(),
        lastSeen: tenant.last_seen || new Date().toISOString(),
        has_real_data: true, // All data is real now
      })) || [];

      setData(transformedData);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
      setData([]); // Return empty array instead of null
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return {
    data,
    loading,
    error,
    refetch: fetchData,
  };
};

export const useEnhancedTenants = (): UseEnhancedTenantsReturn => {
  const { data: basicTenants, loading, error, refetch } = useTenants();

  // Transform basic tenants to enhanced tenants with default values
  const enhancedTenants: EnhancedTenant[] | null = basicTenants?.map(tenant => ({
    ...tenant,
    metrics: [],
    configuration: {
      maxGlobalSeriesPerUser: 0,
      ingestionRate: 0,
      maxLabelNamesPerSeries: 0,
      maxMetadataPerUser: 0,
      queryTimeout: '30s',
      maxQueryLength: '1000',
      maxQueryParallelism: 1,
      maxOutstandingRequestsPerTenant: 100,
    },
    alerts: [],
    components: {
      alloy: { replicas: 0, healthy: 0, status: 'unknown' },
      distributors: { count: 0, healthy: 0 },
      ingesters: { count: 0, healthy: 0 },
      queriers: { count: 0, healthy: 0 },
    },
    trends: {
      ingestionTrend: 'stable',
      errorTrend: 'stable',
      storageTrend: 'stable',
    },
    recommendations: [],
  })) || null;

  return {
    data: enhancedTenants,
    loading,
    error,
    refetch,
  };
}; 