import { useState, useEffect } from 'react';
import { tenants as enhancedMockTenants, basicTenants } from '../mocks/tenants';
import { config } from '../config/environment';

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

// Legacy interface for backward compatibility
interface TenantData {
  name: string;
  namespace: string;
  status: string;
  cpuUsage: number;
  memoryUsage: number;
  alloyReplicas: number;
}

export function useTenants(enhanced: boolean = false) {
  const [data, setData] = useState<EnhancedTenant[] | TenantData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        // Use enhanced tenant data by default, fall back to basic for legacy components
        setData(enhanced ? enhancedMockTenants : basicTenants);
        setLoading(false);
      }, 300);
    } else {
      const endpoint = config.endpoints.tenants;
      fetch(`${config.apiBaseUrl}${endpoint}`)
        .then(res => res.json())
        .then(setData)
        .catch(error => {
          console.error('Failed to fetch tenant data:', error);
          // Fallback to mock data on error
          setData(enhanced ? enhancedMockTenants : basicTenants);
        })
        .finally(() => setLoading(false));
    }
  }, [enhanced]);

  return { data, loading };
}

// Enhanced hook specifically for the new Tenants page
export function useEnhancedTenants() {
  return useTenants(true) as { data: EnhancedTenant[]; loading: boolean };
}

// Legacy hook for backward compatibility
export function useBasicTenants() {
  return useTenants(false) as { data: TenantData[]; loading: boolean };
} 