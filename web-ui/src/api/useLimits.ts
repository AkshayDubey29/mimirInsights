import { useState, useEffect } from 'react';

interface Limit {
  name: string;
  category: string;
  currentValue: any;
  recommendedValue: any;
  status: string;
  lastUpdated: string;
  description: string;
}

interface UseLimitsReturn {
  data: Limit[] | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export const useLimits = (tenant: string): UseLimitsReturn => {
  const [data, setData] = useState<Limit[] | null>(null);
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

      const response = await fetch(`/api/limits?tenant=${encodeURIComponent(tenant)}`);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      // Transform the data to match the expected format
      const transformedData: Limit[] = result.limits?.map((limit: any) => ({
        name: limit.name,
        category: limit.category,
        currentValue: limit.current_value,
        recommendedValue: limit.recommended_value,
        status: limit.status || 'Active',
        lastUpdated: limit.last_updated || new Date().toISOString(),
        description: limit.description || `Limit for ${limit.name}`,
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