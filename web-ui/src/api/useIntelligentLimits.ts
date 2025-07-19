import { useState, useEffect } from 'react';

interface IntelligentLimitRecommendation {
  limit_name: string;
  category: string;
  current_value: any;
  recommended_value: any;
  observed_peak: number;
  average_usage: number;
  usage_percentile_95: number;
  usage_percentile_99: number;
  risk_level: string;
  confidence: number;
  reason: string;
  impact: string;
  priority: string;
  estimated_savings: any;
  implementation_steps: string[];
  last_updated: string;
}

interface TenantAnalysis {
  tenant_name: string;
  risk_score: number;
  reliability_score: number;
  performance_score: number;
  cost_optimization_score: number;
  recommendations: IntelligentLimitRecommendation[];
  missing_limits: string[];
  summary: any;
}

interface IntelligentLimitsData {
  tenant_recommendations: TenantAnalysis[];
  total_tenants: number;
  average_scores: {
    risk_score: number;
    reliability_score: number;
    performance_score: number;
    cost_optimization_score: number;
  };
  overall_summary: {
    total_recommendations: number;
    critical_recommendations: number;
    high_priority_recommendations: number;
    missing_limits_total: number;
    reliability_issues: number;
    performance_issues: number;
    cost_optimization_opportunities: number;
  };
  timestamp: string;
}

interface UseIntelligentLimitsReturn {
  data: IntelligentLimitsData | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export const useIntelligentLimits = (): UseIntelligentLimitsReturn => {
  const [data, setData] = useState<IntelligentLimitsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/limit-recommendations');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
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

// Hook for analyzing a specific tenant
export const useTenantAnalysis = (tenantName: string) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const analyzeTenant = async () => {
    if (!tenantName) return;

    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`/api/analyze-tenant?tenant=${encodeURIComponent(tenantName)}`);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
    } finally {
      setLoading(false);
    }
  };

  return {
    data,
    loading,
    error,
    analyzeTenant,
  };
}; 