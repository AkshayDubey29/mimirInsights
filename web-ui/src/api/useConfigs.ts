import { useState, useEffect } from 'react';
import { configs as mockConfigs } from '../mocks/configs';
import { config } from '../config/environment';

export function useConfigs() {
  const [data, setData] = useState([]);
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
        .then(setData)
        .finally(() => setLoading(false));
    }
  }, []);

  return { data, loading };
} 