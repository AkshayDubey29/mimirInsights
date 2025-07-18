import { useState, useEffect } from 'react';
import { reports as mockReports } from '../mocks/reports';
import { config } from '../config/environment';

interface ReportData {
  tenant: string;
  capacityStatus: string;
  alloyTuning: string;
  details: string;
}

export function useReports() {
  const [data, setData] = useState<ReportData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (config.useMockData) {
      setTimeout(() => {
        setData(mockReports);
        setLoading(false);
      }, 300);
    } else {
      fetch(`${config.apiBaseUrl}${config.endpoints.reports}`)
        .then(res => {
          if (!res.ok) {
            throw new Error(`HTTP ${res.status}: ${res.statusText}`);
          }
          return res.json();
        })
        .then(responseData => {
          // Handle error responses from the API
          if (responseData && responseData.error) {
            console.warn('API returned error:', responseData.error);
            setData([{
              tenant: 'System',
              capacityStatus: 'No Data',
              alloyTuning: 'Setup Required',
              details: responseData.error || 'No capacity data available'
            }]);
            setError(null); // Don't treat this as a hard error
            return;
          }

          // Handle different response formats from the capacity endpoint
          let reportsData = [];
          
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
        })
        .catch(err => {
          console.error('Error fetching reports:', err);
          setError(err.message);
          // Set fallback data to prevent map error
          setData([{
            tenant: 'Error',
            capacityStatus: 'Connection Failed',
            alloyTuning: 'Unavailable',
            details: `Error: ${err.message}`
          }]);
        })
        .finally(() => setLoading(false));
    }
  }, []);

  return { data, loading, error };
} 