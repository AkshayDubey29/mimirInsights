import { useState, useEffect } from 'react';
import { reports as mockReports } from '../mocks/reports';
import { config } from '../config/environment';

export function useReports() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockReports);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.reports}`)
        .then(res => res.json())
        .then(setData)
        .finally(() => setLoading(false));
    }
  }, []);

  return { data, loading };
} 