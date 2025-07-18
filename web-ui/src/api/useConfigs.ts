import { useState, useEffect } from 'react';
import { configs as mockConfigs } from '../mocks/configs';
import { config } from '../config/environment';

interface ConfigData {
  tenant: string;
  configDrift: boolean;
  auditStatus: string;
  details: string;
}

export function useConfigs() {
  const [data, setData] = useState<ConfigData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockConfigs);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.configs}`)
        .then(res => res.json())
        .then(response => {
          // Transform backend response to expected format
          const tenants = response?.environment?.detected_tenants || [];
          const configData: ConfigData[] = tenants.map((tenant: any) => ({
            tenant: tenant.name,
            configDrift: false, // TODO: Implement drift detection
            auditStatus: tenant.has_real_data ? 'Active' : 'Mock Data',
            details: tenant.has_real_data 
              ? `Active tenant with ${tenant.metrics_volume} metrics`
              : 'Mock tenant for development'
          }));
          setData(configData);
        })
        .catch(error => {
          console.error('Error fetching configs:', error);
          // Fallback to mock data on error
          setData(mockConfigs);
        })
        .finally(() => setLoading(false));
    }
  }, []);

  return { data, loading };
} 