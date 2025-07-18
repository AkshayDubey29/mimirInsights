import React, { useState, useEffect } from 'react';
import {
  Box, Grid, Paper, Typography, Card, CardContent, CardHeader, 
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, Button, Alert, CircularProgress, LinearProgress, IconButton,
  FormControl, InputLabel, Select, MenuItem, Tabs, Tab, Badge,
  Accordion, AccordionSummary, AccordionDetails, Avatar, Stack,
  Tooltip, Divider
} from '@mui/material';
import {
  Dashboard as DashboardIcon, Warning, CheckCircle, Error, Info,
  Refresh, Timeline, Storage, Memory, Speed, TrendingUp, TrendingDown,
  CloudQueue, DataUsage, MonitorHeart, ExpandMore, Settings,
  Assessment, Security, AutoAwesome, Notifications
} from '@mui/icons-material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, 
         Legend, ResponsiveContainer, AreaChart, Area, BarChart, Bar, 
         PieChart, Pie, Cell } from 'recharts';
import { useTenants } from '../api/useTenants';
import { useMetrics } from '../api/useMetrics';

interface MetricsSummary {
  totalTenants: number;
  healthyTenants: number;
  alertingTenants: number;
  totalIngestionRate: number;
  totalSeries: number;
  rejectedSamples: number;
}

interface TenantHealth {
  name: string;
  status: 'healthy' | 'warning' | 'critical';
  ingestionRate: number;
  rejectedSamples: number;
  limitUtilization: number;
  recommendationsCount: number;
  lastSeen: string;
  namespace: string;
}

interface SystemComponent {
  name: string;
  type: 'distributor' | 'ingester' | 'querier' | 'compactor' | 'alloy' | 'nginx';
  status: 'running' | 'pending' | 'error' | 'warning';
  replicas: { ready: number; total: number };
  cpuUsage: number;
  memoryUsage: number;
  namespace: string;
}

const Dashboard: React.FC = () => {
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedTenant, setSelectedTenant] = useState<string>('all');
  const [timeRange, setTimeRange] = useState<string>('24h');
  const { data: tenants, loading: tenantsLoading } = useTenants();
  const { data: metrics, loading: metricsLoading } = useMetrics(selectedTenant);

  // Mock data for demonstration - in real implementation, these would come from API
  const [summary, setSummary] = useState<MetricsSummary>({
    totalTenants: 12,
    healthyTenants: 8,
    alertingTenants: 4,
    totalIngestionRate: 450000,
    totalSeries: 2.8e6,
    rejectedSamples: 1234
  });

  const [tenantHealth, setTenantHealth] = useState<TenantHealth[]>([
    { name: 'transportation', status: 'warning', ingestionRate: 180000, rejectedSamples: 500, limitUtilization: 85, recommendationsCount: 3, lastSeen: '2 min ago', namespace: 'transportation' },
    { name: 'eats', status: 'critical', ingestionRate: 174000, rejectedSamples: 2300, limitUtilization: 96, recommendationsCount: 5, lastSeen: '1 min ago', namespace: 'eats' },
    { name: 'marketplace', status: 'healthy', ingestionRate: 96000, rejectedSamples: 0, limitUtilization: 45, recommendationsCount: 0, lastSeen: '30 sec ago', namespace: 'marketplace' },
  ]);

  const [components, setComponents] = useState<SystemComponent[]>([
    { name: 'mimir-distributor', type: 'distributor', status: 'running', replicas: { ready: 3, total: 3 }, cpuUsage: 65, memoryUsage: 72, namespace: 'mimir' },
    { name: 'mimir-ingester', type: 'ingester', status: 'running', replicas: { ready: 6, total: 6 }, cpuUsage: 78, memoryUsage: 85, namespace: 'mimir' },
    { name: 'mimir-querier', type: 'querier', status: 'running', replicas: { ready: 4, total: 4 }, cpuUsage: 45, memoryUsage: 55, namespace: 'mimir' },
    { name: 'alloy-transportation', type: 'alloy', status: 'running', replicas: { ready: 2, total: 2 }, cpuUsage: 35, memoryUsage: 40, namespace: 'transportation' },
    { name: 'alloy-eats', type: 'alloy', status: 'warning', replicas: { ready: 1, total: 2 }, cpuUsage: 95, memoryUsage: 88, namespace: 'eats' },
  ]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': case 'running': return 'success';
      case 'warning': case 'pending': return 'warning';
      case 'critical': case 'error': return 'error';
      default: return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': case 'running': return <CheckCircle />;
      case 'warning': case 'pending': return <Warning />;
      case 'critical': case 'error': return <Error />;
      default: return <Info />;
    }
  };

  const formatNumber = (num: number) => {
    if (num >= 1e6) return `${(num / 1e6).toFixed(1)}M`;
    if (num >= 1e3) return `${(num / 1e3).toFixed(1)}K`;
    return num.toString();
  };

  // Mock metrics data for charts
  const metricsData = Array.from({ length: 24 }, (_, i) => ({
    time: `${i}:00`,
    ingestionRate: 400000 + Math.random() * 100000,
    rejectedSamples: Math.random() * 1000,
    activeSeries: 2.5e6 + Math.random() * 500000,
    memoryUsage: 60 + Math.random() * 30,
  }));

  const tenantDistribution = [
    { name: 'transportation', value: 35, color: '#8884d8' },
    { name: 'eats', value: 30, color: '#82ca9d' },
    { name: 'marketplace', value: 20, color: '#ffc658' },
    { name: 'ads', value: 15, color: '#ff7300' },
  ];

  if (tenantsLoading || metricsLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="400px">
        <CircularProgress size={60} />
        <Typography variant="h6" sx={{ ml: 2 }}>Loading MimirInsights Dashboard...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
    <Box>
          <Typography variant="h4" component="h1" fontWeight="bold">
            <DashboardIcon sx={{ mr: 2, verticalAlign: 'middle' }} />
            MimirInsights Dashboard
          </Typography>
          <Typography variant="subtitle1" color="text.secondary">
            AI-enabled Observability Configuration Auditor & Optimizer
          </Typography>
        </Box>
        <Box display="flex" gap={2}>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Time Range</InputLabel>
            <Select value={timeRange} label="Time Range" onChange={(e) => setTimeRange(e.target.value)}>
              <MenuItem value="1h">Last Hour</MenuItem>
              <MenuItem value="24h">Last 24h</MenuItem>
              <MenuItem value="7d">Last 7 days</MenuItem>
              <MenuItem value="30d">Last 30 days</MenuItem>
        </Select>
      </FormControl>
          <Button variant="outlined" startIcon={<Refresh />}>Refresh</Button>
          <Button variant="contained" startIcon={<AutoAwesome />}>AI Insights</Button>
        </Box>
      </Box>

      {/* Critical Alerts */}
      {summary.alertingTenants > 0 && (
        <Alert severity="warning" sx={{ mb: 3 }} action={
          <Button color="inherit" size="small">View Details</Button>
        }>
          <strong>{summary.alertingTenants} tenants</strong> require immediate attention due to limit violations or configuration drift.
        </Alert>
      )}

      {/* Key Metrics Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'primary.main', mr: 2 }}>
                  <DataUsage />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {formatNumber(summary.totalIngestionRate)}/s
                  </Typography>
                  <Typography color="text.secondary">Total Ingestion Rate</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'success.main', mr: 2 }}>
                  <Timeline />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {formatNumber(summary.totalSeries)}
                  </Typography>
                  <Typography color="text.secondary">Active Series</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'warning.main', mr: 2 }}>
                  <Warning />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {formatNumber(summary.rejectedSamples)}
                  </Typography>
                  <Typography color="text.secondary">Rejected Samples</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'info.main', mr: 2 }}>
                  <MonitorHeart />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {summary.healthyTenants}/{summary.totalTenants}
                  </Typography>
                  <Typography color="text.secondary">Healthy Tenants</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Main Content Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={selectedTab} onChange={(_, newValue) => setSelectedTab(newValue)}>
          <Tab label="System Overview" />
          <Tab label="Tenant Health" />
          <Tab label="Metrics Flow" />
          <Tab label="AI Recommendations" />
        </Tabs>
      </Paper>

      {/* Tab Content */}
      {selectedTab === 0 && (
        <Grid container spacing={3}>
          {/* System Components */}
          <Grid item xs={12} lg={8}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>Mimir Components Health</Typography>
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>Component</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Replicas</TableCell>
                      <TableCell>CPU</TableCell>
                      <TableCell>Memory</TableCell>
                      <TableCell>Namespace</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {components.map((component) => (
                      <TableRow key={component.name}>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            {getStatusIcon(component.status)}
                            <Typography sx={{ ml: 1 }}>{component.name}</Typography>
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip 
                            label={component.status} 
                            color={getStatusColor(component.status) as any}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>{component.replicas.ready}/{component.replicas.total}</TableCell>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            {component.cpuUsage}%
                            <LinearProgress 
                              variant="determinate" 
                              value={component.cpuUsage} 
                              sx={{ ml: 1, width: 50 }}
                            />
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            {component.memoryUsage}%
                            <LinearProgress 
                              variant="determinate" 
                              value={component.memoryUsage} 
                              sx={{ ml: 1, width: 50 }}
                            />
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip label={component.namespace} variant="outlined" size="small" />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Paper>
          </Grid>

          {/* Tenant Distribution */}
          <Grid item xs={12} lg={4}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>Tenant Resource Distribution</Typography>
              <Box height={300}>
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={tenantDistribution}
                      cx="50%"
                      cy="50%"
                      innerRadius={60}
                      outerRadius={100}
                      paddingAngle={5}
                      dataKey="value"
                    >
                      {tenantDistribution.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </Pie>
                    <RechartsTooltip />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
              </Box>
            </Paper>
          </Grid>
        </Grid>
      )}

      {selectedTab === 1 && (
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>Tenant Health Status</Typography>
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>Tenant</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Ingestion Rate</TableCell>
                      <TableCell>Rejected Samples</TableCell>
                      <TableCell>Limit Utilization</TableCell>
                      <TableCell>Recommendations</TableCell>
                      <TableCell>Last Seen</TableCell>
                      <TableCell>Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {tenantHealth.map((tenant) => (
                      <TableRow key={tenant.name}>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            <Avatar sx={{ width: 24, height: 24, mr: 1, fontSize: 12 }}>
                              {tenant.name.charAt(0).toUpperCase()}
                            </Avatar>
                            {tenant.name}
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip 
                            label={tenant.status} 
                            color={getStatusColor(tenant.status) as any}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>{formatNumber(tenant.ingestionRate)}/s</TableCell>
                        <TableCell>
                          {tenant.rejectedSamples > 0 ? (
                            <Chip 
                              label={formatNumber(tenant.rejectedSamples)} 
                              color="error" 
                              size="small"
                            />
                          ) : (
                            <Chip label="0" color="success" size="small" />
                          )}
                        </TableCell>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            {tenant.limitUtilization}%
                            <LinearProgress 
                              variant="determinate" 
                              value={tenant.limitUtilization} 
                              sx={{ ml: 1, width: 60 }}
                              color={tenant.limitUtilization > 80 ? 'error' : 'primary'}
                            />
                          </Box>
                        </TableCell>
                        <TableCell>
                          {tenant.recommendationsCount > 0 ? (
                            <Badge badgeContent={tenant.recommendationsCount} color="warning">
                              <Notifications />
                            </Badge>
                          ) : (
                            <CheckCircle color="success" />
                          )}
                        </TableCell>
                        <TableCell>{tenant.lastSeen}</TableCell>
                        <TableCell>
                          <Button size="small" variant="outlined">View Details</Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Paper>
          </Grid>
        </Grid>
      )}

      {selectedTab === 2 && (
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>Real-time Metrics Flow</Typography>
              <Box height={400}>
        <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={metricsData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis />
                    <RechartsTooltip />
            <Legend />
                    <Area 
                      type="monotone" 
                      dataKey="ingestionRate" 
                      stroke="#8884d8" 
                      fill="#8884d8" 
                      fillOpacity={0.3} 
                      name="Ingestion Rate" 
                    />
                    <Area 
                      type="monotone" 
                      dataKey="rejectedSamples" 
                      stroke="#ff7300" 
                      fill="#ff7300" 
                      fillOpacity={0.3} 
                      name="Rejected Samples" 
                    />
                  </AreaChart>
        </ResponsiveContainer>
      </Box>
            </Paper>
          </Grid>
        </Grid>
      )}

      {selectedTab === 3 && (
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>AI-Generated Recommendations</Typography>
              <Stack spacing={2}>
                <Accordion>
                  <AccordionSummary expandIcon={<ExpandMore />}>
                    <Box display="flex" alignItems="center" width="100%">
                      <Warning color="error" sx={{ mr: 2 }} />
                      <Box>
                        <Typography variant="subtitle1">Tenant 'eats' requires immediate attention</Typography>
                        <Typography variant="body2" color="text.secondary">
                          Ingestion rate limit exceeded, 2,300 samples rejected in last 24h
                        </Typography>
                      </Box>
                    </Box>
                  </AccordionSummary>
                  <AccordionDetails>
                    <Typography paragraph>
                      <strong>Current limit:</strong> 150,000 samples/sec<br/>
                      <strong>Observed peak:</strong> 174,000 samples/sec<br/>
                      <strong>Recommended limit:</strong> 190,000 samples/sec (10% buffer)
                    </Typography>
                    <Button variant="contained" size="small">Apply Recommendation</Button>
                  </AccordionDetails>
                </Accordion>

                <Accordion>
                  <AccordionSummary expandIcon={<ExpandMore />}>
                    <Box display="flex" alignItems="center" width="100%">
                      <Info color="info" sx={{ mr: 2 }} />
                      <Box>
                        <Typography variant="subtitle1">Alloy replica optimization for 'transportation'</Typography>
                        <Typography variant="body2" color="text.secondary">
                          Current setup is over-provisioned, cost savings opportunity
                        </Typography>
                      </Box>
                    </Box>
                  </AccordionSummary>
                  <AccordionDetails>
                    <Typography paragraph>
                      Analysis shows that 1 Alloy replica can handle the current load efficiently.
                      Reducing from 2 to 1 replica could save ~$180/month while maintaining performance.
                    </Typography>
                    <Button variant="outlined" size="small">Schedule Optimization</Button>
                  </AccordionDetails>
                </Accordion>
              </Stack>
            </Paper>
          </Grid>
        </Grid>
      )}
    </Box>
  );
};

export default Dashboard; 