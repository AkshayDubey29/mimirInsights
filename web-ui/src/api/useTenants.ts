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

interface UseTenantsReturn {
  data: Tenant[] | null;
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