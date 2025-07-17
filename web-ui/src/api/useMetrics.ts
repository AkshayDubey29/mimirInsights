import { useState, useEffect } from 'react';
import { metrics as mockMetrics } from '../mocks/metrics';
import { config } from '../config/environment';

export function useMetrics(tenant: string) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockMetrics[tenant]);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.metrics}?tenant=${tenant}`)
        .then(res => res.json())
        .then(setData)
        .finally(() => setLoading(false));
    }
  }, [tenant]);

  return { data, loading };
} 