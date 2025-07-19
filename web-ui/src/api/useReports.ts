import { useState, useEffect } from 'react';

interface ReportData {
  tenant: string;
  capacityStatus: string;
  alloyTuning: string;
  details: string;
}

interface UseReportsReturn {
  data: ReportData[];
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

export const useReports = (): UseReportsReturn => {
  const [data, setData] = useState<ReportData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/capacity/report');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const responseData = await response.json();
      
      // Handle different response formats from the capacity endpoint
      let reportsData: ReportData[] = [];
      
      if (Array.isArray(responseData)) {
        // Direct array response
        reportsData = responseData;
      } else if (responseData && typeof responseData === 'object') {
        // Object with reports property
        if (Array.isArray(responseData.reports)) {
          reportsData = responseData.reports;
        } else if (Array.isArray(responseData.capacity)) {
          // Transform capacity data to reports format
          reportsData = responseData.capacity.map((item: any) => ({
            tenant: item.tenant || item.name || 'Unknown',
            capacityStatus: item.status || item.capacity_status || 'Unknown',
            alloyTuning: item.recommendation || item.alloy_tuning || 'No data',
            details: item.details || item.description || 'No details available'
          }));
        } else if (responseData.tenants && Array.isArray(responseData.tenants)) {
          // Transform tenants data to reports format
          if (responseData.tenants.length === 0) {
            reportsData = [{
              tenant: 'System',
              capacityStatus: 'No Tenants Found',
              alloyTuning: 'Deploy Alloy First',
              details: 'No tenant namespaces discovered. Deploy Alloy or check discovery configuration.'
            }];
          } else {
            reportsData = responseData.tenants.map((tenant: any) => ({
              tenant: tenant.name || 'Unknown',
              capacityStatus: tenant.status || 'Unknown',
              alloyTuning: 'Analysis pending',
              details: `Components: ${tenant.component_count || 0}`
            }));
          }
        } else {
          // Fallback: create a single report from the object
          reportsData = [{
            tenant: 'System',
            capacityStatus: 'Data Available',
            alloyTuning: 'Check individual tenants',
            details: 'Capacity data received, but format needs conversion'
          }];
        }
      } else {
        // No valid data, create placeholder
        reportsData = [{
          tenant: 'System',
          capacityStatus: 'No Data',
          alloyTuning: 'Setup Required',
          details: 'No capacity data available. Check if MimirInsights can discover your Mimir/Alloy deployments.'
        }];
      }
      
      setData(reportsData);
      setError(null);
    } catch (err) {
      console.error('Error fetching reports:', err);
      setError(err instanceof Error ? err.message : 'Unknown error occurred');
      // Set fallback data to prevent map error
      setData([{
        tenant: 'Error',
        capacityStatus: 'Connection Failed',
        alloyTuning: 'Unavailable',
        details: `Error: ${err instanceof Error ? err.message : 'Unknown error'}`
      }]);
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
    refetch: fetchData
  };
}; 