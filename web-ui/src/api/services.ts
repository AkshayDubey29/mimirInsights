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

export interface Limit {
  tenant: string;
  cpuRequest: number;
  cpuLimit: number;
  memoryRequest: number;
  memoryLimit: number;
  recommendedCpu: number;
  recommendedMemory: number;
}

export interface Config {
  tenant: string;
  configDrift: boolean;
  auditStatus: string;
  details: string;
}

export interface Report {
  tenant: string;
  capacityStatus: string;
  alloyTuning: string;
  details: string;
}

// Service functions
export const tenantService = {
  async getTenants(): Promise<ApiResponse<Tenant[]>> {
    return apiClient.get<Tenant[]>(config.endpoints.tenants);
  },

  async getTenant(name: string): Promise<ApiResponse<Tenant>> {
    return apiClient.get<Tenant>(`${config.endpoints.tenants}/${name}`);
  },

  async updateTenant(name: string, data: Partial<Tenant>): Promise<ApiResponse<Tenant>> {
    return apiClient.put<Tenant>(`${config.endpoints.tenants}/${name}`, data);
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

  async getLimitsForTenant(tenant: string): Promise<ApiResponse<Limit>> {
    return apiClient.get<Limit>(`${config.endpoints.limits}/${tenant}`);
  },

  async applyLimits(tenant: string, limits: Partial<Limit>): Promise<ApiResponse<Limit>> {
    return apiClient.post<Limit>(`${config.endpoints.limits}/${tenant}/apply`, limits);
  },

  async generateRecommendations(tenant: string): Promise<ApiResponse<Limit>> {
    return apiClient.post<Limit>(`${config.endpoints.limits}/${tenant}/recommendations`);
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
    return apiClient.get<Report[]>(config.endpoints.reports);
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