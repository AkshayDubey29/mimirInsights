import { useState, useEffect } from 'react';
import { config } from '../config/environment';

interface Config {
  id: string;
  name: string;
  namespace: string;
  type: string;
  status: string;
  lastModified: string;
  auditStatus: string;
  description: string;
}

interface UseConfigsReturn {
  data: Config[] | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export const useConfigs = (): UseConfigsReturn => {
  const [data, setData] = useState<Config[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/config');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      // Transform the data to match the expected format
      const transformedData: Config[] = result.configs?.map((config: any) => ({
        id: config.name,
        name: config.name,
        namespace: config.namespace,
        type: config.kind,
        status: config.status || 'Active',
        lastModified: config.lastModified || new Date().toISOString(),
        auditStatus: 'Active',
        description: config.description || `Configuration for ${config.name}`,
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
  }, []);

  return {
    data,
    loading,
    error,
    refetch: fetchData,
  };
}; 