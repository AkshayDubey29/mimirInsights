import { useState, useEffect } from 'react';
import { metrics as mockMetrics } from '../mocks/metrics';
import { config } from '../config/environment';
import { metricsService } from './services';

interface MetricData {
  cpu: number[];
  memory: number[];
  alloyReplicas: number[];
  timestamps: number[];
}

interface RealMetricsData {
  data_source: string;
  endpoints: string[];
  metrics: Record<string, any>;
  time_range: any;
  collected_at: string;
}

export function useMetrics(tenant: string) {
  const [data, setData] = useState<MetricData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockMetrics[tenant as keyof typeof mockMetrics] || null);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.metrics}?tenant=${tenant}`)
        .then(res => res.json())
        .then(data => {
          setData(data);
          setLoading(false);
        })
        .catch(error => {
          console.error('Error fetching metrics:', error);
          setLoading(false);
        });
    }
  }, [tenant]);

  return { data, loading };
}

export function useRealMetrics(timeRange: string = '1h') {
  const [data, setData] = useState<RealMetricsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);

    metricsService.getRealMetrics(timeRange)
      .then(response => {
        if (response.success && response.data) {
          setData(response.data);
        } else {
          setError(response.error || 'Failed to fetch real metrics');
        }
        setLoading(false);
      })
      .catch(err => {
        console.error('Error fetching real metrics:', err);
        setError(err.message || 'Failed to fetch real metrics');
        setLoading(false);
      });
  }, [timeRange]);

  return { data, loading, error };
} 