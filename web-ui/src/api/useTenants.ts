import { useState, useEffect } from 'react';
import { tenants as mockTenants } from '../mocks/tenants';
import { config } from '../config/environment';

export function useTenants() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockTenants);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.tenants}`)
        .then(res => res.json())
        .then(setData)
        .finally(() => setLoading(false));
    }
  }, []);

  return { data, loading };
} 