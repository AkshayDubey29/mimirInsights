export const config = {
  // API base URL - will be different for dev/staging/prod
  apiBaseUrl: process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080',
  // Feature flag for using mock data
  useMockData: process.env.REACT_APP_USE_MOCK_DATA === 'true',
  // Environment
  environment: process.env.NODE_ENV || 'development',
  // API endpoints
  endpoints: {
    tenants: '/api/v1/tenants',
    metrics: '/api/v1/metrics',
    limits: '/api/v1/limits',
    configs: '/api/v1/configs',
    reports: '/api/v1/reports',
  },
};

export const isProduction = config.environment === 'production';
export const isDevelopment = config.environment === 'development'; 