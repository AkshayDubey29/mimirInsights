import { apiClient, ApiResponse } from './client';
import { config } from '../config/environment';

// Types for API responses
export interface Tenant {
  name: string;
  namespace: string;
  status: string;
  cpuUsage: number;
  memoryUsage: number;
  alloyReplicas: number;
}

export interface Metrics {
  cpu: number[];
  memory: number[];
  alloyReplicas: number[];
  timestamps: number[];
}

export interface Config {
  tenant: string;
  configDrift: boolean;
  auditStatus: string;
  details: string;
}

export interface Limit {
  tenant: string;
  cpuRequest: number;
  cpuLimit: number;
  memoryRequest: number;
  memoryLimit: number;
  recommendedCpu: number;
  recommendedMemory: number;
}

export interface Report {
  tenant: string;
  capacityStatus: string;
  alloyTuning: string;
  details: string;
}

export const tenantsService = {
  async getTenants(): Promise<ApiResponse<Tenant[]>> {
    return apiClient.get<Tenant[]>(config.endpoints.tenants);
  },

  async createTenant(tenant: Omit<Tenant, 'status'>): Promise<ApiResponse<Tenant>> {
    return apiClient.post<Tenant>(config.endpoints.tenants, tenant);
  },
};

export const metricsService = {
  async getMetrics(tenant: string, timeRange?: string): Promise<ApiResponse<Metrics>> {
    const params = new URLSearchParams({ tenant });
    if (timeRange) params.append('timeRange', timeRange);
    return apiClient.get<Metrics>(`${config.endpoints.metrics}?${params.toString()}`);
  },

  async getMetricsSummary(): Promise<ApiResponse<any>> {
    return apiClient.get(`${config.endpoints.metrics}/summary`);
  },
};

export const limitsService = {
  async getLimits(): Promise<ApiResponse<Limit[]>> {
    return apiClient.get<Limit[]>(config.endpoints.limits);
  },

  async getTenantLimits(tenant: string): Promise<ApiResponse<Limit>> {
    return apiClient.get<Limit>(`${config.endpoints.limits}/${tenant}`);
  },

  async updateLimits(tenant: string, limits: Partial<Limit>): Promise<ApiResponse<Limit>> {
    return apiClient.put<Limit>(`${config.endpoints.limits}/${tenant}`, limits);
  },
};

export const configService = {
  async getConfigs(): Promise<ApiResponse<Config[]>> {
    return apiClient.get<Config[]>(config.endpoints.configs);
  },

  async auditConfigs(): Promise<ApiResponse<Config[]>> {
    return apiClient.post<Config[]>(`${config.endpoints.configs}/audit`);
  },

  async fixDrift(tenant: string): Promise<ApiResponse<Config>> {
    return apiClient.post<Config>(`${config.endpoints.configs}/${tenant}/fix-drift`);
  },

  async exportAuditReport(): Promise<ApiResponse<string>> {
    return apiClient.get<string>(`${config.endpoints.configs}/export`);
  },
};

export const reportsService = {
  async getReports(): Promise<ApiResponse<Report[]>> {
    try {
      // Try the capacity endpoint first
      const capacityResponse = await apiClient.get<any>(config.endpoints.reports);
      
      if (capacityResponse.success && capacityResponse.data) {
        // Transform capacity data to reports format
        let reportsData: Report[] = [];
        
        if (Array.isArray(capacityResponse.data)) {
          reportsData = capacityResponse.data.map((item: any) => ({
            tenant: item.tenant || item.name || 'Unknown',
            capacityStatus: item.status || item.capacity_status || 'Unknown',
            alloyTuning: item.recommendation || item.alloy_tuning || 'No data',
            details: item.details || item.description || 'No details available'
          }));
        } else if (capacityResponse.data.tenants && Array.isArray(capacityResponse.data.tenants)) {
          reportsData = capacityResponse.data.tenants.map((tenant: any) => ({
            tenant: tenant.name || 'Unknown',
            capacityStatus: tenant.status || 'Unknown',
            alloyTuning: 'Analysis pending',
            details: `Components: ${tenant.component_count || 0}`
          }));
        }
        
        return {
          data: reportsData,
          success: true
        };
      }
      
      // Fallback to tenants endpoint if capacity fails
      const tenantsResponse = await this.getTenantsAsReports();
      return tenantsResponse;
      
    } catch (error) {
      // Final fallback
      return {
        data: [{
          tenant: 'System',
          capacityStatus: 'Error',
          alloyTuning: 'Unavailable',
          details: `Failed to fetch reports: ${error}`
        }],
        success: false,
        error: `Failed to fetch reports: ${error}`
      };
    }
  },

  async getTenantsAsReports(): Promise<ApiResponse<Report[]>> {
    try {
      const tenantsResponse = await tenantsService.getTenants();
      
      if (tenantsResponse.success && Array.isArray(tenantsResponse.data)) {
        const reportsData: Report[] = tenantsResponse.data.map((tenant: Tenant) => ({
          tenant: tenant.name,
          capacityStatus: tenant.status || 'Unknown',
          alloyTuning: `${tenant.alloyReplicas || 0} replicas`,
          details: `CPU: ${tenant.cpuUsage || 0}%, Memory: ${tenant.memoryUsage || 0}%`
        }));
        
        return {
          data: reportsData,
          success: true
        };
      }
      
      return {
        data: [],
        success: false,
        error: 'No tenant data available'
      };
    } catch (error) {
      return {
        data: [],
        success: false,
        error: `Failed to fetch tenant data: ${error}`
      };
    }
  },

  async generateReport(tenant: string): Promise<ApiResponse<Report>> {
    return apiClient.post<Report>(`${config.endpoints.reports}/${tenant}/generate`);
  },

  async exportReport(tenant: string, format: 'pdf' | 'csv' = 'pdf'): Promise<ApiResponse<string>> {
    return apiClient.get<string>(`${config.endpoints.reports}/${tenant}/export?format=${format}`);
  },

  async getCapacityPlanning(): Promise<ApiResponse<any>> {
    return apiClient.get(`${config.endpoints.reports}/capacity-planning`);
  },
}; 