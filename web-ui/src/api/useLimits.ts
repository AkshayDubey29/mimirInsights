import { useState, useEffect } from 'react';
import { limits as mockLimits } from '../mocks/limits';
import { config } from '../config/environment';

interface LimitData {
  tenant: string;
  cpuRequest: number;
  cpuLimit: number;
  memoryRequest: number;
  memoryLimit: number;
  recommendedCpu: number;
  recommendedMemory: number;
}

export function useLimits() {
  const [data, setData] = useState<LimitData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockLimits);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.limits}`)
        .then(res => res.json())
        .then(setData)
        .finally(() => setLoading(false));
    }
  }, []);

  return { data, loading };
} 