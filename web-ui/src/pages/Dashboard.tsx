import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  Chip,
  LinearProgress,
  Alert,
  Button,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  TrendingUp,
  TrendingDown,
  Warning,
  CheckCircle,
  Error,
  Refresh,
  Memory,
  Storage,
  Speed,
  Timeline,
} from '@mui/icons-material';
import { useTenants } from '../api/useTenants';
import { useEnvironment } from '../api/useEnvironment';

interface DashboardMetrics {
  totalTenants: number;
  healthyTenants: number;
  warningTenants: number;
  criticalTenants: number;
  totalSeries: number;
  totalIngestionRate: number;
  totalQueryRate: number;
  averageErrorRate: number;
  systemHealth: 'healthy' | 'warning' | 'critical';
}

const Dashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { data: tenants, loading: tenantsLoading, error: tenantsError, refetch: refetchTenants } = useTenants();
  const { data: environment, loading: envLoading, error: envError, refetch: refetchEnv } = useEnvironment();

  const fetchDashboardMetrics = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/metrics/dashboard');
      if (!response.ok) {
        throw new (Error as any)(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setMetrics(data);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch dashboard metrics';
      setError(errorMessage);
      // Calculate basic metrics from tenants data if available
      if (tenants && tenants.length > 0) {
        const healthyCount = tenants.filter(t => t.status === 'healthy').length;
        const warningCount = tenants.filter(t => t.status === 'warning').length;
        const criticalCount = tenants.filter(t => t.status === 'critical').length;
        
        setMetrics({
          totalTenants: tenants.length,
          healthyTenants: healthyCount,
          warningTenants: warningCount,
          criticalTenants: criticalCount,
          totalSeries: 0, // Will be calculated from real metrics
          totalIngestionRate: 0, // Will be calculated from real metrics
          totalQueryRate: 0, // Will be calculated from real metrics
          averageErrorRate: 0, // Will be calculated from real metrics
          systemHealth: criticalCount > 0 ? 'critical' : warningCount > 0 ? 'warning' : 'healthy',
        });
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboardMetrics();
  }, [tenants]);

  const handleRefresh = () => {
    refetchTenants();
    refetchEnv();
    fetchDashboardMetrics();
  };

  const getHealthColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success';
      case 'warning': return 'warning';
      case 'critical': return 'error';
      default: return 'default';
    }
  };

  const getHealthIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircle />;
      case 'warning': return <Warning />;
      case 'critical': return <Error />;
      default: return <Warning />;
    }
  };

  if (loading || tenantsLoading || envLoading) {
    return (
      <Box sx={{ width: '100%' }}>
        <LinearProgress />
        <Typography variant="h6" sx={{ mt: 2 }}>
          Loading dashboard...
        </Typography>
      </Box>
    );
  }

  if (error || tenantsError || envError) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        Failed to load dashboard data: {error || tenantsError?.message || envError?.message}
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Dashboard
        </Typography>
        <Button
          variant="contained"
          startIcon={<Refresh />}
          onClick={handleRefresh}
        >
          Refresh
        </Button>
      </Box>

      {/* System Health Overview */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">
              System Health
            </Typography>
            <Chip
              icon={getHealthIcon(metrics?.systemHealth || 'warning')}
              label={metrics?.systemHealth || 'Unknown'}
              color={getHealthColor(metrics?.systemHealth || 'warning')}
            />
          </Box>
          
          {environment && (
            <Grid container spacing={2}>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="primary">
                    {environment.mimir_components?.length || 0}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Mimir Components
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="primary">
                    {environment.detected_tenants?.length || 0}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Detected Tenants
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="primary">
                    {environment.mimir_namespace || 'Unknown'}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Mimir Namespace
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="primary">
                    {environment.data_source || 'Production'}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Data Source
                  </Typography>
                </Box>
              </Grid>
            </Grid>
          )}
        </CardContent>
      </Card>

      {/* Tenant Overview */}
      {tenants && tenants.length > 0 && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Tenant Overview
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="success.main">
                    {metrics?.healthyTenants || 0}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Healthy
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="warning.main">
                    {metrics?.warningTenants || 0}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Warning
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="error.main">
                    {metrics?.criticalTenants || 0}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Critical
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} md={3}>
                <Box textAlign="center">
                  <Typography variant="h4" color="primary">
                    {metrics?.totalTenants || 0}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Total
                  </Typography>
                </Box>
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      )}

      {/* Recent Activity */}
      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Recent Activity
          </Typography>
          {tenants && tenants.length > 0 ? (
            <Box>
              {tenants.slice(0, 5).map((tenant) => (
                <Box key={tenant.id} sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                  <Box>
                    <Typography variant="body1" fontWeight="medium">
                      {tenant.name}
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                      Last seen: {new Date(tenant.lastSeen).toLocaleString()}
                    </Typography>
                  </Box>
                  <Chip
                    icon={getHealthIcon(tenant.status)}
                    label={tenant.status}
                    color={getHealthColor(tenant.status)}
                    size="small"
                  />
                </Box>
              ))}
            </Box>
          ) : (
            <Typography variant="body2" color="textSecondary">
              No tenant activity detected
            </Typography>
          )}
        </CardContent>
      </Card>
    </Box>
  );
};

export default Dashboard; 