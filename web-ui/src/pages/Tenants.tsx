import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  TextField,
  InputAdornment,
  Chip,
  Button,
  IconButton,
  Menu,
  MenuItem,
  Card,
  CardContent,
  CardHeader,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  Alert,
  Tooltip,
  LinearProgress,
  Avatar,
  Tabs,
  Tab,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  Switch,
  FormControlLabel,
  Badge,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
  Add as AddIcon,
  MoreVert as MoreVertIcon,
  Visibility as VisibilityIcon,
  Settings as SettingsIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  CheckCircle as CheckCircleIcon,
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  CloudUpload as CloudUploadIcon,
  Timeline as TimelineIcon,
  Storage as StorageIcon,
  Speed as SpeedIcon,
  Memory as MemoryIcon,
  Dns as DnsIcon,
  NetworkCheck as NetworkCheckIcon,
  Security as SecurityIcon,
  Assignment as AssignmentIcon,
  PlayCircleOutline as PlayCircleOutlineIcon,
  PauseCircleOutline as PauseCircleOutlineIcon,
  RestartAlt as RestartAltIcon,
} from '@mui/icons-material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import { useEnhancedTenants } from '../api/useTenants';
import { config } from '../config/environment';

interface TenantMetrics {
  timestamp: string;
  ingestionRate: number;
  queryRate: number;
  seriesCount: number;
  samplesPerSecond: number;
  storageUsageGB: number;
  cpuUsage: number;
  memoryUsage: number;
  errorRate: number;
}

interface TenantConfiguration {
  maxGlobalSeriesPerUser: number;
  ingestionRate: number;
  maxLabelNamesPerSeries: number;
  maxMetadataPerUser: number;
  queryTimeout: string;
  maxQueryLength: string;
  maxQueryParallelism: number;
  maxOutstandingRequestsPerTenant: number;
}

interface TenantAlert {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  title: string;
  description: string;
  timestamp: string;
  resolved: boolean;
}

interface EnhancedTenant {
  id: string;
  name: string;
  namespace: string;
  status: 'healthy' | 'warning' | 'critical' | 'inactive';
  discoveredAt: string;
  lastSeen: string;
  metrics: TenantMetrics[];
  configuration: TenantConfiguration;
  alerts: TenantAlert[];
  components: {
    alloy: { replicas: number; healthy: number; status: string };
    distributors: { count: number; healthy: number };
    ingesters: { count: number; healthy: number };
    queriers: { count: number; healthy: number };
  };
  trends: {
    ingestionTrend: 'up' | 'down' | 'stable';
    errorTrend: 'up' | 'down' | 'stable';
    storageTrend: 'up' | 'down' | 'stable';
  };
  recommendations: Array<{
    type: 'optimization' | 'scaling' | 'configuration';
    title: string;
    impact: 'high' | 'medium' | 'low';
    description: string;
  }>;
}



const StatusChip: React.FC<{ status: string }> = ({ status }) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success';
      case 'warning': return 'warning';
      case 'critical': return 'error';
      case 'inactive': return 'default';
      default: return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircleIcon />;
      case 'warning': return <WarningIcon />;
      case 'critical': return <ErrorIcon />;
      default: return undefined;
    }
  };

  return (
    <Chip
      icon={getStatusIcon(status)}
      label={status.charAt(0).toUpperCase() + status.slice(1)}
      color={getStatusColor(status) as any}
      size="small"
    />
  );
};

const TrendIcon: React.FC<{ trend: 'up' | 'down' | 'stable' }> = ({ trend }) => {
  switch (trend) {
    case 'up': return <TrendingUpIcon color="error" />;
    case 'down': return <TrendingDownIcon color="success" />;
    case 'stable': return <TimelineIcon color="primary" />;
    default: return null;
  }
};

const Tenants: React.FC = () => {
  const [selectedTab, setSelectedTab] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [selectedTenant, setSelectedTenant] = useState<EnhancedTenant | null>(null);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [selectedTenantId, setSelectedTenantId] = useState<string>('');
  const [addTenantDialogOpen, setAddTenantDialogOpen] = useState(false);
  const [newTenantData, setNewTenantData] = useState({
    name: '',
    namespace: '',
    description: ''
  });

  // Function to handle opening add tenant dialog
  const handleAddTenant = () => {
    setAddTenantDialogOpen(true);
    setNewTenantData({ name: '', namespace: '', description: '' });
  };

  // Function to handle adding new tenant
  const handleSubmitNewTenant = async () => {
    if (!newTenantData.name.trim() || !newTenantData.namespace.trim()) {
      alert('Please fill in all required fields');
      return;
    }

    try {
      const response = await fetch(`${config.apiBaseUrl}/api/tenants`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: newTenantData.name,
          namespace: newTenantData.namespace,
          description: newTenantData.description,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        alert(`Tenant "${newTenantData.name}" created successfully!`);
        setAddTenantDialogOpen(false);
        setNewTenantData({ name: '', namespace: '', description: '' });
        // Refresh tenants list
        window.location.reload();
      } else {
        const error = await response.json();
        alert(`Failed to create tenant: ${error.error || 'Unknown error'}`);
      }
    } catch (error) {
      alert(`Error creating tenant: ${error}`);
    }
  };

  // Use enhanced tenant data from API
  const { data: tenants, loading } = useEnhancedTenants();

  useEffect(() => {
    let interval: NodeJS.Timeout;
    if (autoRefresh) {
      interval = setInterval(() => {
        // Simulate real-time updates
        console.log('Refreshing tenant data...');
      }, 30000);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [autoRefresh]);

  const filteredTenants = tenants.filter(tenant => {
    const matchesSearch = tenant.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         tenant.namespace.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === 'all' || tenant.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const handleTenantAction = (action: string, tenantId: string) => {
    console.log(`${action} action for tenant: ${tenantId}`);
    setAnchorEl(null);
  };

  const openTenantDetails = (tenant: EnhancedTenant) => {
    setSelectedTenant(tenant);
    setDetailsDialogOpen(true);
  };

  const columns: GridColDef[] = [
    {
      field: 'name',
      headerName: 'Tenant',
      width: 120,
      renderCell: (params: GridRenderCellParams) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Avatar sx={{ width: 24, height: 24, fontSize: '0.75rem', bgcolor: 'primary.main' }}>
            {params.value.charAt(0).toUpperCase()}
          </Avatar>
          <Typography variant="body2" fontWeight="medium">{params.value}</Typography>
        </Box>
      ),
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 100,
      renderCell: (params: GridRenderCellParams) => <StatusChip status={params.value} />,
    },
    {
      field: 'ingestionRate',
      headerName: 'Ingestion Rate',
      width: 130,
      renderCell: (params: GridRenderCellParams) => {
        const tenant = params.row as EnhancedTenant;
        const currentRate = tenant.metrics[tenant.metrics.length - 1]?.ingestionRate || 0;
        const limit = tenant.configuration.ingestionRate;
        const percentage = (currentRate / limit) * 100;
        
        return (
          <Box>
            <Typography variant="body2">
              {(currentRate / 1000).toFixed(0)}K/s
            </Typography>
            <LinearProgress
              variant="determinate"
              value={Math.min(percentage, 100)}
              color={percentage > 100 ? 'error' : percentage > 80 ? 'warning' : 'primary'}
              sx={{ height: 4, borderRadius: 2 }}
            />
          </Box>
        );
      },
    },
    {
      field: 'seriesCount',
      headerName: 'Series Count',
      width: 120,
      renderCell: (params: GridRenderCellParams) => {
        const tenant = params.row as EnhancedTenant;
        const currentSeries = tenant.metrics[tenant.metrics.length - 1]?.seriesCount || 0;
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Typography variant="body2">
              {(currentSeries / 1000000).toFixed(1)}M
            </Typography>
            <TrendIcon trend={tenant.trends.ingestionTrend} />
          </Box>
        );
      },
    },
    {
      field: 'errorRate',
      headerName: 'Error Rate',
      width: 100,
      renderCell: (params: GridRenderCellParams) => {
        const tenant = params.row as EnhancedTenant;
        const errorRate = tenant.metrics[tenant.metrics.length - 1]?.errorRate || 0;
        const percentage = errorRate * 100;
        
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Typography 
              variant="body2" 
              color={percentage > 2 ? 'error' : percentage > 1 ? 'warning' : 'textSecondary'}
            >
              {percentage.toFixed(2)}%
            </Typography>
            <TrendIcon trend={tenant.trends.errorTrend} />
          </Box>
        );
      },
    },
    {
      field: 'alerts',
      headerName: 'Alerts',
      width: 80,
      renderCell: (params: GridRenderCellParams) => {
        const tenant = params.row as EnhancedTenant;
        const activeAlerts = tenant.alerts.filter(alert => !alert.resolved);
        const criticalCount = activeAlerts.filter(alert => alert.severity === 'critical').length;
        const warningCount = activeAlerts.filter(alert => alert.severity === 'warning').length;
        
        return (
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            {criticalCount > 0 && (
              <Badge badgeContent={criticalCount} color="error">
                <ErrorIcon fontSize="small" />
              </Badge>
            )}
            {warningCount > 0 && (
              <Badge badgeContent={warningCount} color="warning">
                <WarningIcon fontSize="small" />
              </Badge>
            )}
            {activeAlerts.length === 0 && (
              <CheckCircleIcon fontSize="small" color="success" />
            )}
          </Box>
        );
      },
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 120,
      sortable: false,
      renderCell: (params: GridRenderCellParams) => (
        <Box>
          <Tooltip title="View Details">
            <IconButton 
              size="small" 
              onClick={() => openTenantDetails(params.row as EnhancedTenant)}
            >
              <VisibilityIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="More Actions">
            <IconButton 
              size="small" 
              onClick={(e) => {
                setAnchorEl(e.currentTarget);
                setSelectedTenantId(params.row.id);
              }}
            >
              <MoreVertIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      ),
    },
  ];

  const renderOverviewTab = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h5" fontWeight="bold">
            Tenant Auto-Discovery & Management
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <FormControlLabel
              control={
                <Switch
                  checked={autoRefresh}
                  onChange={(e) => setAutoRefresh(e.target.checked)}
                  size="small"
                />
              }
              label="Auto Refresh"
            />
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={() => console.log('Manual refresh')}
              size="small"
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleAddTenant}
              size="small"
            >
              Add Tenant
            </Button>
          </Box>
        </Box>
      </Grid>
      
      {/* Summary Cards */}
      <Grid item xs={12} md={3}>
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography color="textSecondary" gutterBottom variant="body2">
                  Total Tenants
                </Typography>
                <Typography variant="h4" component="div">
                  {tenants.length}
                </Typography>
                <Typography variant="body2" color="success.main">
                  +2 discovered today
                </Typography>
              </Box>
              <DnsIcon color="primary" sx={{ fontSize: 40 }} />
            </Box>
          </CardContent>
        </Card>
      </Grid>

      <Grid item xs={12} md={3}>
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography color="textSecondary" gutterBottom variant="body2">
                  Healthy Tenants
                </Typography>
                <Typography variant="h4" component="div" color="success.main">
                  {tenants.filter(t => t.status === 'healthy').length}
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  {((tenants.filter(t => t.status === 'healthy').length / tenants.length) * 100).toFixed(0)}% healthy
                </Typography>
              </Box>
              <CheckCircleIcon color="success" sx={{ fontSize: 40 }} />
            </Box>
          </CardContent>
        </Card>
      </Grid>

      <Grid item xs={12} md={3}>
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography color="textSecondary" gutterBottom variant="body2">
                  Total Ingestion
                </Typography>
                <Typography variant="h4" component="div">
                  {(tenants.reduce((sum, t) => sum + (t.metrics[t.metrics.length - 1]?.ingestionRate || 0), 0) / 1000).toFixed(0)}K/s
                </Typography>
                <Typography variant="body2" color="primary">
                  Across all tenants
                </Typography>
              </Box>
              <CloudUploadIcon color="primary" sx={{ fontSize: 40 }} />
            </Box>
          </CardContent>
        </Card>
      </Grid>

      <Grid item xs={12} md={3}>
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography color="textSecondary" gutterBottom variant="body2">
                  Active Alerts
                </Typography>
                <Typography variant="h4" component="div" color="error.main">
                  {tenants.reduce((sum, t) => sum + t.alerts.filter(a => !a.resolved).length, 0)}
                </Typography>
                <Typography variant="body2" color="error.main">
                  {tenants.filter(t => t.alerts.some(a => !a.resolved && a.severity === 'critical')).length} critical
                </Typography>
              </Box>
              <WarningIcon color="error" sx={{ fontSize: 40 }} />
            </Box>
          </CardContent>
        </Card>
      </Grid>

      {/* Filters and Search */}
      <Grid item xs={12}>
        <Paper sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
            <TextField
              placeholder="Search tenants..."
              variant="outlined"
              size="small"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
              }}
              sx={{ minWidth: 250 }}
            />
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel>Status Filter</InputLabel>
              <Select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                label="Status Filter"
              >
                <MenuItem value="all">All Status</MenuItem>
                <MenuItem value="healthy">Healthy</MenuItem>
                <MenuItem value="warning">Warning</MenuItem>
                <MenuItem value="critical">Critical</MenuItem>
                <MenuItem value="inactive">Inactive</MenuItem>
              </Select>
            </FormControl>
            <Chip
              icon={<FilterIcon />}
              label={`${filteredTenants.length} tenant(s) shown`}
              variant="outlined"
            />
          </Box>
        </Paper>
      </Grid>

      {/* Tenants Data Grid */}
      <Grid item xs={12}>
        <Paper sx={{ height: 600, width: '100%' }}>
          <DataGrid
            rows={filteredTenants}
            columns={columns}
            pageSize={10}
            rowsPerPageOptions={[10, 25, 50]}
            disableSelectionOnClick
            loading={loading}
            getRowClassName={(params) => {
              switch (params.row.status) {
                case 'critical': return 'row-critical';
                case 'warning': return 'row-warning';
                default: return '';
              }
            }}
            sx={{
              '& .row-critical': {
                backgroundColor: 'rgba(244, 67, 54, 0.1)',
              },
              '& .row-warning': {
                backgroundColor: 'rgba(255, 152, 0, 0.1)',
              },
            }}
          />
        </Paper>
      </Grid>
    </Grid>
  );

  const renderAnalyticsTab = () => (
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Card>
          <CardHeader title="Tenant Distribution by Status" />
          <CardContent>
            <Box sx={{ height: 300 }}>
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={[
                      { name: 'Healthy', value: tenants.filter(t => t.status === 'healthy').length, fill: '#4caf50' },
                      { name: 'Warning', value: tenants.filter(t => t.status === 'warning').length, fill: '#ff9800' },
                      { name: 'Critical', value: tenants.filter(t => t.status === 'critical').length, fill: '#f44336' },
                      { name: 'Inactive', value: tenants.filter(t => t.status === 'inactive').length, fill: '#9e9e9e' },
                    ]}
                    cx="50%"
                    cy="50%"
                    outerRadius={80}
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  />
                  <RechartsTooltip />
                </PieChart>
              </ResponsiveContainer>
            </Box>
          </CardContent>
        </Card>
      </Grid>

      <Grid item xs={12} md={6}>
        <Card>
          <CardHeader title="Ingestion Rate Distribution" />
          <CardContent>
            <Box sx={{ height: 300 }}>
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={tenants.map(t => ({
                  name: t.name,
                  rate: t.metrics[t.metrics.length - 1]?.ingestionRate || 0,
                  limit: t.configuration.ingestionRate
                }))}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <RechartsTooltip 
                    formatter={(value, name) => [
                      `${(value as number / 1000).toFixed(0)}K/s`, 
                      name === 'rate' ? 'Current' : 'Limit'
                    ]}
                  />
                  <Bar dataKey="rate" fill="#2196f3" name="Current Rate" />
                  <Bar dataKey="limit" fill="#ff9800" name="Limit" />
                </BarChart>
              </ResponsiveContainer>
            </Box>
          </CardContent>
        </Card>
      </Grid>

      <Grid item xs={12}>
        <Card>
          <CardHeader title="Real-time Metrics Across All Tenants" />
          <CardContent>
            <Box sx={{ height: 400 }}>
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={tenants[0].metrics.map((_, index) => {
                  const timestamp = tenants[0].metrics[index].timestamp;
                  return {
                    timestamp,
                    ...tenants.reduce((acc, tenant) => {
                      acc[`${tenant.name}_ingestion`] = tenant.metrics[index]?.ingestionRate || 0;
                      acc[`${tenant.name}_errors`] = (tenant.metrics[index]?.errorRate || 0) * 10000; // Scale for visibility
                      return acc;
                    }, {} as any)
                  };
                })}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="timestamp" />
                  <YAxis />
                  <RechartsTooltip />
                  {tenants.map((tenant, index) => (
                    <Line
                      key={`${tenant.name}_ingestion`}
                      type="monotone"
                      dataKey={`${tenant.name}_ingestion`}
                      stroke={['#2196f3', '#4caf50', '#ff9800', '#9c27b0'][index % 4]}
                      strokeWidth={2}
                      name={`${tenant.name} Ingestion`}
                    />
                  ))}
                </LineChart>
              </ResponsiveContainer>
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  );

  return (
    <Box sx={{ p: 3 }}>
      <Tabs
        value={selectedTab}
        onChange={(_, newValue) => setSelectedTab(newValue)}
        sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}
      >
        <Tab label="Tenant Overview" />
        <Tab label="Analytics & Trends" />
        <Tab label="Discovery Engine" />
        <Tab label="Configuration Audit" />
      </Tabs>

      {selectedTab === 0 && renderOverviewTab()}
      {selectedTab === 1 && renderAnalyticsTab()}
      {selectedTab === 2 && (
        <Box>
          <Alert severity="info" sx={{ mb: 2 }}>
            <strong>Auto-Discovery Active:</strong> Continuously scanning Mimir metrics endpoints for new tenants. 
            Last scan: 2 minutes ago. Next scan: in 28 minutes.
          </Alert>
          <Typography variant="h6">Discovery Engine Configuration</Typography>
          {/* Additional discovery engine content would go here */}
        </Box>
      )}
      {selectedTab === 3 && (
        <Box>
          <Alert severity="warning" sx={{ mb: 2 }}>
            <strong>Configuration Drift Detected:</strong> 3 tenants have configuration mismatches. 
            Review recommended configurations to optimize performance.
          </Alert>
          <Typography variant="h6">Configuration Audit Results</Typography>
          {/* Additional audit content would go here */}
        </Box>
      )}

      {/* Action Menu */}
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        <MenuItem onClick={() => handleTenantAction('configure', selectedTenantId)}>
          <SettingsIcon sx={{ mr: 1 }} fontSize="small" />
          Configure Limits
        </MenuItem>
        <MenuItem onClick={() => handleTenantAction('restart', selectedTenantId)}>
          <RestartAltIcon sx={{ mr: 1 }} fontSize="small" />
          Restart Components
        </MenuItem>
        <MenuItem onClick={() => handleTenantAction('pause', selectedTenantId)}>
          <PauseCircleOutlineIcon sx={{ mr: 1 }} fontSize="small" />
          Pause Ingestion
        </MenuItem>
        <MenuItem onClick={() => handleTenantAction('resume', selectedTenantId)}>
          <PlayCircleOutlineIcon sx={{ mr: 1 }} fontSize="small" />
          Resume Ingestion
        </MenuItem>
        <Divider />
        <MenuItem onClick={() => handleTenantAction('export', selectedTenantId)}>
          <AssignmentIcon sx={{ mr: 1 }} fontSize="small" />
          Export Configuration
        </MenuItem>
      </Menu>

      {/* Tenant Details Dialog */}
      <Dialog
        open={detailsDialogOpen}
        onClose={() => setDetailsDialogOpen(false)}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Avatar sx={{ bgcolor: 'primary.main' }}>
                {selectedTenant?.name.charAt(0).toUpperCase()}
              </Avatar>
              <Box>
                <Typography variant="h6">{selectedTenant?.name}</Typography>
                <Typography variant="body2" color="textSecondary">
                  {selectedTenant?.namespace}
                </Typography>
              </Box>
              <StatusChip status={selectedTenant?.status || 'unknown'} />
            </Box>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button startIcon={<SettingsIcon />} size="small">
                Configure
              </Button>
              <Button startIcon={<TimelineIcon />} size="small">
                Metrics
              </Button>
            </Box>
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedTenant && (
            <Grid container spacing={3}>
              {/* Metrics Overview */}
              <Grid item xs={12} md={8}>
                <Card>
                  <CardHeader title="Real-time Metrics" />
                  <CardContent>
                    <Box sx={{ height: 300 }}>
                      <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={selectedTenant.metrics}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="timestamp" />
                          <YAxis />
                          <RechartsTooltip />
                          <Area
                            type="monotone"
                            dataKey="ingestionRate"
                            stroke="#2196f3"
                            fill="#2196f3"
                            fillOpacity={0.3}
                            name="Ingestion Rate"
                          />
                        </AreaChart>
                      </ResponsiveContainer>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>

              {/* Component Health */}
              <Grid item xs={12} md={4}>
                <Card>
                  <CardHeader title="Component Health" />
                  <CardContent>
                    <List dense>
                      <ListItem>
                        <ListItemIcon>
                          <NetworkCheckIcon color={selectedTenant.components.alloy.status === 'healthy' ? 'success' : 'error'} />
                        </ListItemIcon>
                        <ListItemText
                          primary="Alloy Agents"
                          secondary={`${selectedTenant.components.alloy.healthy}/${selectedTenant.components.alloy.replicas} healthy`}
                        />
                      </ListItem>
                      <ListItem>
                        <ListItemIcon>
                          <StorageIcon color="success" />
                        </ListItemIcon>
                        <ListItemText
                          primary="Distributors"
                          secondary={`${selectedTenant.components.distributors.healthy}/${selectedTenant.components.distributors.count} healthy`}
                        />
                      </ListItem>
                      <ListItem>
                        <ListItemIcon>
                          <MemoryIcon color="success" />
                        </ListItemIcon>
                        <ListItemText
                          primary="Ingesters"
                          secondary={`${selectedTenant.components.ingesters.healthy}/${selectedTenant.components.ingesters.count} healthy`}
                        />
                      </ListItem>
                      <ListItem>
                        <ListItemIcon>
                          <SpeedIcon color="success" />
                        </ListItemIcon>
                        <ListItemText
                          primary="Queriers"
                          secondary={`${selectedTenant.components.queriers.healthy}/${selectedTenant.components.queriers.count} healthy`}
                        />
                      </ListItem>
                    </List>
                  </CardContent>
                </Card>
              </Grid>

              {/* Active Alerts */}
              {selectedTenant.alerts.length > 0 && (
                <Grid item xs={12}>
                  <Card>
                    <CardHeader title="Active Alerts" />
                    <CardContent>
                                             {selectedTenant.alerts.filter(alert => !alert.resolved).map((alert) => (
                         <Alert
                           key={alert.id}
                           severity={alert.severity === 'critical' ? 'error' : alert.severity}
                           sx={{ mb: 1 }}
                         >
                           <strong>{alert.title}</strong>: {alert.description}
                         </Alert>
                       ))}
                    </CardContent>
                  </Card>
                </Grid>
              )}

              {/* AI Recommendations */}
              <Grid item xs={12}>
                <Card>
                  <CardHeader title="AI Recommendations" />
                  <CardContent>
                    {selectedTenant.recommendations.map((rec, index) => (
                      <Box key={index} sx={{ mb: 2, p: 2, border: 1, borderColor: 'divider', borderRadius: 1 }}>
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                          <Typography variant="subtitle2">{rec.title}</Typography>
                          <Chip
                            label={rec.impact}
                            size="small"
                            color={rec.impact === 'high' ? 'error' : rec.impact === 'medium' ? 'warning' : 'default'}
                          />
                        </Box>
                        <Typography variant="body2" color="textSecondary">
                          {rec.description}
                        </Typography>
                        <Button size="small" sx={{ mt: 1 }}>
                          Apply Recommendation
                        </Button>
                      </Box>
                    ))}
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Add Tenant Dialog */}
      <Dialog open={addTenantDialogOpen} onClose={() => setAddTenantDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          <Box display="flex" alignItems="center">
            <AddIcon sx={{ mr: 2 }} />
            Add New Tenant
          </Box>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <Alert severity="info" sx={{ mb: 3 }}>
              This dialog simulates tenant creation. In a production environment, this would create 
              the necessary Kubernetes resources and ConfigMaps for the new tenant.
            </Alert>
            
            <TextField
              fullWidth
              label="Tenant Name *"
              value={newTenantData.name}
              onChange={(e) => setNewTenantData({ ...newTenantData, name: e.target.value })}
              placeholder="e.g., eats, transportation, delivery"
              sx={{ mb: 3 }}
              helperText="Unique identifier for the tenant"
            />
            
            <TextField
              fullWidth
              label="Namespace *"
              value={newTenantData.namespace}
              onChange={(e) => setNewTenantData({ ...newTenantData, namespace: e.target.value })}
              placeholder="e.g., tenant-eats, tenant-transportation"
              sx={{ mb: 3 }}
              helperText="Kubernetes namespace for tenant resources"
            />
            
            <TextField
              fullWidth
              label="Description"
              value={newTenantData.description}
              onChange={(e) => setNewTenantData({ ...newTenantData, description: e.target.value })}
              placeholder="Optional description for this tenant"
              multiline
              rows={3}
              helperText="Optional description for documentation purposes"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddTenantDialogOpen(false)}>Cancel</Button>
          <Button 
            variant="contained" 
            onClick={handleSubmitNewTenant}
            disabled={!newTenantData.name.trim() || !newTenantData.namespace.trim()}
          >
            Create Tenant
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Tenants; 