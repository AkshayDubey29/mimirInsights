// Declare process for React environment variables
declare const process: {
  env: {
    REACT_APP_API_BASE_URL?: string;
    REACT_APP_USE_MOCK_DATA?: string;
    NODE_ENV?: string;
  };
};

// Runtime configuration from window.APP_CONFIG (injected by Kubernetes)
const runtimeConfig = (window as any).APP_CONFIG || {};

export const config = {
  // API base URL - will be different for dev/staging/prod
  // In Kubernetes, this should point to the nginx proxy (same origin)
  apiBaseUrl: runtimeConfig.apiBaseUrl || process.env.REACT_APP_API_BASE_URL || '',
  // Feature flag for using mock data
  useMockData: runtimeConfig.useMockData || process.env.REACT_APP_USE_MOCK_DATA === 'true',
  // Environment
  environment: process.env.NODE_ENV || 'development',
  // API endpoints - no /api prefix since apiBaseUrl includes it
  endpoints: {
    tenants: '/tenants',
    metrics: '/metrics',
    limits: '/limits',
    configs: '/config',
    reports: '/capacity',  // Fixed: Use capacity endpoint for reports
    audit: '/audit',       // Added: Separate audit endpoint
  },
};

export const isProduction = config.environment === 'production';
export const isDevelopment = config.environment === 'development'; 