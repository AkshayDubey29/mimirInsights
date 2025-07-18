import React, { useState, useEffect } from 'react';
import {
  Box, Grid, Paper, Typography, Card, CardContent, CardHeader, 
  Chip, Alert, CircularProgress, IconButton, Badge, Tooltip,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Accordion, AccordionSummary, AccordionDetails, Stack, Divider,
  List, ListItem, ListItemIcon, ListItemText
} from '@mui/material';
import {
  CloudQueue, DataUsage, Storage, Security, AutoAwesome,
  Refresh, Warning, CheckCircle, Error, Info, ExpandMore,
  Visibility, Code, Settings, Timeline, Assessment,
  Computer, Dns, AccountTree, Lock, Public, Business
} from '@mui/icons-material';
import { apiClient } from '../api/client';

interface EnvironmentInfo {
  cluster_info: {
    is_production: boolean;
    cluster_name: string;
    cluster_version: string;
    mimir_namespace: string;
    total_namespaces: number;
    total_nodes: number;
    data_source: string;
    detected_tenants: DetectedTenant[];
    mimir_components: string[];
    environment_details: any;
  };
  auto_discovered: {
    global_limits: Record<string, any>;
    tenant_limits: Record<string, TenantLimit>;
    config_sources: ConfigSource[];
  };
  mimir_components: string[];
  detected_tenants: DetectedTenant[];
  data_source_status: string;
  is_production: boolean;
  total_config_sources: number;
  last_updated: string;
}

interface DetectedTenant {
  name: string;
  namespace: string;
  source: string;
  org_id: string;
  has_real_data: boolean;
  last_seen: string;
  metrics_volume: number;
  component_status: Record<string, string>;
}

interface TenantLimit {
  tenant_id: string;
  limits: Record<string, any>;
  source: string;
  last_updated: string;
}

interface ConfigSource {
  name: string;
  namespace: string;
  type: string;
  keys: string[];
  last_seen: string;
}

const EnvironmentStatus: React.FC = () => {
  const [environmentData, setEnvironmentData] = useState<EnvironmentInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  const fetchEnvironmentData = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiClient.get('/api/environment');
      setEnvironmentData(response.data as EnvironmentInfo);
      setLastRefresh(new Date());
    } catch (err: any) {
      setError(err.message || 'Failed to fetch environment data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEnvironmentData();
    // Auto-refresh every 5 minutes
    const interval = setInterval(fetchEnvironmentData, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, []);

  const getDataSourceColor = (dataSource: string) => {
    switch (dataSource) {
      case 'production': return 'success';
      case 'mock': return 'warning';
      case 'mixed': return 'info';
      default: return 'default';
    }
  };

  const getDataSourceIcon = (dataSource: string) => {
    switch (dataSource) {
      case 'production': return <Business />;
      case 'mock': return <Code />;
      case 'mixed': return <Assessment />;
      default: return <Info />;
    }
  };

  const getComponentStatusColor = (status: string) => {
    switch (status) {
      case 'running': return 'success';
      case 'pending': return 'warning';
      case 'error': return 'error';
      default: return 'default';
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
        <IconButton size="small" onClick={fetchEnvironmentData} sx={{ ml: 2 }}>
          <Refresh />
        </IconButton>
      </Alert>
    );
  }

  if (!environmentData) {
    return (
      <Alert severity="info" sx={{ m: 2 }}>
        No environment data available
      </Alert>
    );
  }

  const { cluster_info, auto_discovered, detected_tenants, is_production, total_config_sources } = environmentData;

  return (
    <Box sx={{ p: 3 }}>
      <Box display="flex" justifyContent="between" alignItems="center" mb={3}>
        <Typography variant="h4" gutterBottom>
          Environment Status
        </Typography>
        <Box display="flex" alignItems="center" gap={2}>
          <Typography variant="body2" color="textSecondary">
            Last updated: {new Date(lastRefresh).toLocaleTimeString()}
          </Typography>
          <IconButton onClick={fetchEnvironmentData} size="small">
            <Refresh />
          </IconButton>
        </Box>
      </Box>

      <Grid container spacing={3}>
        {/* Cluster Overview */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader 
              title="Cluster Information"
              avatar={<Computer />}
            />
            <CardContent>
              <Stack spacing={2}>
                <Box display="flex" justifyContent="between" alignItems="center">
                  <Typography variant="body2" color="textSecondary">Environment Type</Typography>
                  <Chip 
                    icon={is_production ? <Business /> : <Code />}
                    label={is_production ? 'Production' : 'Development'}
                    color={is_production ? 'success' : 'warning'}
                    variant="outlined"
                  />
                </Box>
                <Box display="flex" justifyContent="between" alignItems="center">
                  <Typography variant="body2" color="textSecondary">Data Source</Typography>
                  <Chip 
                    icon={getDataSourceIcon(cluster_info.data_source)}
                    label={cluster_info.data_source.toUpperCase()}
                    color={getDataSourceColor(cluster_info.data_source)}
                    variant="outlined"
                  />
                </Box>
                <Divider />
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Cluster Name</Typography>
                  <Typography variant="body2">{cluster_info.cluster_name}</Typography>
                </Box>
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Version</Typography>
                  <Typography variant="body2">{cluster_info.cluster_version}</Typography>
                </Box>
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Nodes</Typography>
                  <Typography variant="body2">{cluster_info.total_nodes}</Typography>
                </Box>
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Namespaces</Typography>
                  <Typography variant="body2">{cluster_info.total_namespaces}</Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        {/* Auto-Discovery Summary */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader 
              title="Auto-Discovery Status"
              avatar={<AutoAwesome />}
            />
            <CardContent>
              <Stack spacing={2}>
                <Box display="flex" justifyContent="between" alignItems="center">
                  <Typography variant="body2" color="textSecondary">Detection Status</Typography>
                  <Chip 
                    icon={<CheckCircle />}
                    label="Active"
                    color="success"
                    variant="outlined"
                  />
                </Box>
                <Divider />
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Detected Tenants</Typography>
                  <Badge badgeContent={detected_tenants.length} color="primary">
                    <Typography variant="body2">{detected_tenants.length}</Typography>
                  </Badge>
                </Box>
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Config Sources</Typography>
                  <Badge badgeContent={total_config_sources} color="secondary">
                    <Typography variant="body2">{total_config_sources}</Typography>
                  </Badge>
                </Box>
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Global Limits</Typography>
                  <Typography variant="body2">{Object.keys(auto_discovered.global_limits).length}</Typography>
                </Box>
                <Box display="flex" justifyContent="between">
                  <Typography variant="body2" color="textSecondary">Tenant Limits</Typography>
                  <Typography variant="body2">{Object.keys(auto_discovered.tenant_limits).length}</Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        {/* Mimir Components */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader 
              title="Mimir Components"
              avatar={<Storage />}
            />
            <CardContent>
              <Stack spacing={1}>
                {cluster_info.mimir_components.map((component, index) => (
                  <Box key={index} display="flex" justifyContent="between" alignItems="center">
                    <Typography variant="body2">{component}</Typography>
                    <Chip 
                      label="Running"
                      color="success"
                      size="small"
                      variant="outlined"
                    />
                  </Box>
                ))}
                {cluster_info.mimir_components.length === 0 && (
                  <Typography variant="body2" color="textSecondary" style={{ fontStyle: 'italic' }}>
                    No Mimir components detected
                  </Typography>
                )}
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        {/* Detected Tenants */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader 
              title="Detected Tenants"
              avatar={<AccountTree />}
            />
            <CardContent>
              <Stack spacing={1} sx={{ maxHeight: 300, overflow: 'auto' }}>
                {detected_tenants.map((tenant, index) => (
                  <Box key={index} sx={{ p: 1, border: 1, borderColor: 'divider', borderRadius: 1 }}>
                    <Box display="flex" justifyContent="between" alignItems="center" mb={1}>
                      <Typography variant="body2" fontWeight="bold">{tenant.name}</Typography>
                      <Stack direction="row" spacing={0.5}>
                        <Chip 
                          label={tenant.source}
                          size="small"
                          variant="outlined"
                          color="primary"
                        />
                        <Chip 
                          label={tenant.has_real_data ? 'Real Data' : 'Mock Data'}
                          size="small"
                          variant="outlined"
                          color={tenant.has_real_data ? 'success' : 'warning'}
                        />
                      </Stack>
                    </Box>
                    <Typography variant="caption" color="textSecondary">
                      Namespace: {tenant.namespace}
                    </Typography>
                    {tenant.org_id && (
                      <Typography variant="caption" color="textSecondary" sx={{ ml: 2 }}>
                        Org ID: {tenant.org_id}
                      </Typography>
                    )}
                  </Box>
                ))}
                {detected_tenants.length === 0 && (
                  <Typography variant="body2" color="textSecondary" style={{ fontStyle: 'italic' }}>
                    No tenants detected
                  </Typography>
                )}
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        {/* Configuration Sources */}
        <Grid item xs={12}>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMore />}>
              <Typography variant="h6">Configuration Sources ({total_config_sources})</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Name</TableCell>
                      <TableCell>Namespace</TableCell>
                      <TableCell>Type</TableCell>
                      <TableCell>Keys Count</TableCell>
                      <TableCell>Last Seen</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {auto_discovered.config_sources.map((source, index) => (
                      <TableRow key={index}>
                        <TableCell>{source.name}</TableCell>
                        <TableCell>{source.namespace}</TableCell>
                        <TableCell>
                          <Chip 
                            label={source.type}
                            size="small"
                            variant="outlined"
                            color={source.type === 'runtime-override' ? 'primary' : 'default'}
                          />
                        </TableCell>
                        <TableCell>{source.keys.length}</TableCell>
                        <TableCell>
                          <Typography variant="caption">
                            {new Date(source.last_seen).toLocaleString()}
                          </Typography>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </AccordionDetails>
          </Accordion>
        </Grid>

        {/* AI Insights */}
        {is_production && (
          <Grid item xs={12}>
            <Alert 
              severity="info" 
              icon={<AutoAwesome />}
              sx={{ mt: 2 }}
            >
              <Typography variant="body2">
                <strong>AI-Enabled Production Environment Detected:</strong> All limits and configurations 
                are being auto-discovered from live Mimir ConfigMaps and runtime overrides. 
                Recommendations are based on real production metrics and trends.
              </Typography>
            </Alert>
          </Grid>
        )}
      </Grid>
    </Box>
  );
};

export default EnvironmentStatus; 