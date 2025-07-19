// Declare process for React environment variables
declare const process: {
  env: {
    REACT_APP_API_BASE_URL?: string;
    REACT_APP_USE_MOCK_DATA?: string;
    NODE_ENV?: string;
  };
};

interface RuntimeConfig {
  apiUrl: string;
  useMockData: boolean;
  enableRealMetrics: boolean;
}

// Get runtime configuration from window object or use defaults
const runtimeConfig: RuntimeConfig = (window as any).__RUNTIME_CONFIG__ || {
  apiUrl: process.env.REACT_APP_API_BASE_URL || '/api',
  useMockData: false, // Always use real data in production
  enableRealMetrics: true,
};

export const config = {
  apiUrl: runtimeConfig.apiUrl,
  useMockData: false, // Always false - no mock data in production
  enableRealMetrics: runtimeConfig.enableRealMetrics,
  environment: process.env.NODE_ENV || 'development',
  endpoints: {
    tenants: '/tenants',
    metrics: '/metrics',
    limits: '/limits',
    configs: '/configs',
    reports: '/capacity/report',
  },
};

export const isProduction = config.environment === 'production';
export const isDevelopment = config.environment === 'development'; 