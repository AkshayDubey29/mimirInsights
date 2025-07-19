import { useState, useEffect } from 'react';

interface Environment {
  mimir_namespace: string;
  detected_tenants: string[];
  mimir_components: Array<{
    name: string;
    type: string;
    namespace: string;
    status: string;
    lastSeen: string;
  }>;
  data_source: string;
  discovery_confidence: number;
  last_updated: string;
}

interface UseEnvironmentReturn {
  data: Environment | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export const useEnvironment = (): UseEnvironmentReturn => {
  const [data, setData] = useState<Environment | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/environment');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      // Transform the data to match the expected format
      const transformedData: Environment = {
        mimir_namespace: result.cluster_info?.mimir_namespace || 'Unknown',
        detected_tenants: result.detected_tenants || [],
        mimir_components: result.mimir_components || [],
        data_source: result.cluster_info?.data_source || 'Production',
        discovery_confidence: result.environment?.discovery_confidence || 0,
        last_updated: result.last_updated || new Date().toISOString(),
      };

      setData(transformedData);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
      setData(null);
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