import { useState, useEffect } from 'react';
import { config } from '../config/environment';

interface Metric {
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

interface UseMetricsReturn {
  data: Metric[] | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export const useMetrics = (tenant: string): UseMetricsReturn => {
  const [data, setData] = useState<Metric[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = async () => {
    if (!tenant) {
      setData(null);
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`/api/metrics?tenant=${encodeURIComponent(tenant)}`);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      // Transform the data to match the expected format
      const transformedData: Metric[] = result.metrics?.map((metric: any) => ({
        timestamp: metric.timestamp,
        ingestionRate: metric.ingestion_rate || 0,
        queryRate: metric.query_rate || 0,
        seriesCount: metric.series_count || 0,
        samplesPerSecond: metric.samples_per_second || 0,
        storageUsageGB: metric.storage_usage_gb || 0,
        cpuUsage: metric.cpu_usage || 0,
        memoryUsage: metric.memory_usage || 0,
        errorRate: metric.error_rate || 0,
      })) || [];

      setData(transformedData);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
      setData([]); // Return empty array instead of mock data
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [tenant]);

  return {
    data,
    loading,
    error,
    refetch: fetchData,
  };
};

export function useRealMetrics(timeRange: string = '1h') {
  const [data, setData] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);

    // This function is not provided in the new_code, so it will be removed.
    // If it's meant to be kept, it needs to be re-added or the new_code needs to be updated.
    // For now, removing it as it's not part of the new_code's useRealMetrics.
  }, [timeRange]);

  return { data, loading, error };
} 