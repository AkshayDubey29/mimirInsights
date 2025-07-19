import { useQuery, useMutation, useQueryClient } from 'react-query';
import { apiClient } from './client';

export interface MemoryStats {
  current_memory_bytes: number;
  max_memory_bytes: number;
  memory_usage_percent: number;
  peak_memory_bytes: number;
  cache_item_count: number;
  max_cache_size: number;
  tenant_cache_count: number;
  max_tenant_cache_size: number;
  mimir_cache_count: number;
  max_mimir_cache_size: number;
  eviction_count: number;
  memory_warnings: number;
  last_eviction: string;
  last_memory_check: string;
  last_reset: string;
  eviction_policy: string;
  eviction_threshold: number;
  memory_threshold: number;
}

export interface MemorySettings {
  maxMemoryBytes: number;
  maxCacheSize: number;
  maxTenantCacheSize: number;
  maxMimirCacheSize: number;
  evictionPolicy: string;
  evictionThreshold: number;
  memoryThreshold: number;
}

export const useMemoryStats = () => {
  return useQuery<MemoryStats>(
    'memoryStats',
    async (): Promise<MemoryStats> => {
      const response = await apiClient.get('/api/cache/memory');
      return response.data as MemoryStats;
    },
    {
      refetchInterval: 30000, // Refresh every 30 seconds
      staleTime: 10000, // Consider data stale after 10 seconds
      retry: 3,
      retryDelay: 1000,
    }
  );
};

export const useForceMemoryEviction = () => {
  const queryClient = useQueryClient();
  
  return useMutation(
    async () => {
      const response = await apiClient.post('/api/cache/memory/evict');
      return response.data;
    },
    {
      onSuccess: () => {
        // Invalidate and refetch memory stats
        queryClient.invalidateQueries('memoryStats');
      },
    }
  );
};

export const useResetMemoryStats = () => {
  const queryClient = useQueryClient();
  
  return useMutation(
    async () => {
      const response = await apiClient.post('/api/cache/memory/reset');
      return response.data;
    },
    {
      onSuccess: () => {
        // Invalidate and refetch memory stats
        queryClient.invalidateQueries('memoryStats');
      },
    }
  );
};

export const useUpdateMemorySettings = () => {
  const queryClient = useQueryClient();
  
  return useMutation(
    async (settings: MemorySettings) => {
      const response = await apiClient.post('/api/cache/memory/settings', settings);
      return response.data;
    },
    {
      onSuccess: () => {
        // Invalidate and refetch memory stats
        queryClient.invalidateQueries('memoryStats');
      },
    }
  );
};

export const useMemoryHistory = () => {
  return useQuery<Array<{ timestamp: string; stats: MemoryStats }>>(
    'memoryHistory',
    async (): Promise<Array<{ timestamp: string; stats: MemoryStats }>> => {
      const response = await apiClient.get('/api/cache/memory/history');
      return response.data as Array<{ timestamp: string; stats: MemoryStats }>;
    },
    {
      refetchInterval: 60000, // Refresh every minute
      staleTime: 30000,
      retry: 2,
    }
  );
}; 