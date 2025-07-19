import React, { useState, useEffect, useMemo } from 'react';
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
  Stack,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Collapse,
  Snackbar,
  CircularProgress,
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
  ViewList as ViewListIcon,
  ViewModule as ViewModuleIcon,
  Download as DownloadIcon,
  ExpandMore as ExpandMoreIcon,
  Info as InfoIcon,
  People as PeopleIcon,
  AutoAwesome as AutoAwesomeIcon,
  BugReport as BugReportIcon,
  Sort as SortIcon,
} from '@mui/icons-material';
import { useEnhancedTenants } from '../api/useTenants';
import { config } from '../config/environment';
import { DataGridWithPagination, Column } from '../components/DataGridWithPagination';

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
      default: return <InfoIcon />;
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
  const [namespaceFilter, setNamespaceFilter] = useState('all');
  const [selectedTenant, setSelectedTenant] = useState<EnhancedTenant | null>(null);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [viewMode, setViewMode] = useState<'table' | 'grid'>('table');
  const [selectedTenants, setSelectedTenants] = useState<string[]>([]);
  const [addTenantDialogOpen, setAddTenantDialogOpen] = useState(false);
  const [newTenantData, setNewTenantData] = useState({
    name: '',
    namespace: '',
    description: ''
  });
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error' | 'info' | 'warning';
  }>({ open: false, message: '', severity: 'info' });
  const [sortField, setSortField] = useState<string>('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Use enhanced tenant data from API
  const { data: tenants, loading } = useEnhancedTenants();

  // Auto-refresh effect
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      // Trigger a manual refresh by refetching data
      window.location.reload();
    }, 30000); // Refresh every 30 seconds

    return () => clearInterval(interval);
  }, [autoRefresh]);

  // Filter and sort tenants
  const filteredAndSortedTenants = useMemo(() => {
    let filtered = tenants || [];

    // Apply search filter
    if (searchTerm) {
      filtered = filtered.filter((tenant) =>
        tenant.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        tenant.namespace.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    // Apply status filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter((tenant) => tenant.status === statusFilter);
    }

    // Apply namespace filter
    if (namespaceFilter !== 'all') {
      filtered = filtered.filter((tenant) => tenant.namespace === namespaceFilter);
    }

    // Apply sorting
    filtered.sort((a, b) => {
      let aValue: any = a[sortField as keyof EnhancedTenant];
      let bValue: any = b[sortField as keyof EnhancedTenant];

      if (typeof aValue === 'string') {
        aValue = aValue.toLowerCase();
        bValue = bValue.toLowerCase();
      }

      if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    return filtered;
  }, [tenants, searchTerm, statusFilter, namespaceFilter, sortField, sortDirection]);

  // Get unique namespaces for filter
  const uniqueNamespaces = useMemo(() => {
    if (!tenants) return [];
    return Array.from(new Set(tenants.map(tenant => tenant.namespace)));
  }, [tenants]);

  // Statistics
  const stats = useMemo(() => {
    if (!tenants) return { total: 0, healthy: 0, warning: 0, critical: 0, inactive: 0 };
    
    return tenants.reduce((acc, tenant) => {
      acc.total++;
      acc[tenant.status]++;
      return acc;
    }, { total: 0, healthy: 0, warning: 0, critical: 0, inactive: 0 });
  }, [tenants]);

  // Data grid columns
  const columns: Column<EnhancedTenant>[] = [
    {
      id: 'name',
      label: 'Tenant Name',
      minWidth: 150,
      sortable: true,
      searchable: true,
      render: (value, row) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main' }}>
            <PeopleIcon />
          </Avatar>
          <Box>
            <Typography variant="body2" fontWeight="medium">
              {row.name}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {row.namespace}
            </Typography>
          </Box>
        </Box>
      ),
    },
    {
      id: 'status',
      label: 'Status',
      minWidth: 120,
      sortable: true,
      render: (value, row) => <StatusChip status={row.status} />,
    },
    {
      id: 'components',
      label: 'Components',
      minWidth: 150,
      render: (value, row) => (
        <Stack direction="row" spacing={1}>
          <Chip 
            label={`Alloy: ${row.components.alloy.healthy}/${row.components.alloy.replicas}`}
            size="small"
            color={row.components.alloy.healthy === row.components.alloy.replicas ? 'success' : 'warning'}
          />
          <Chip 
            label={`Ingesters: ${row.components.ingesters.healthy}/${row.components.ingesters.count}`}
            size="small"
            color={row.components.ingesters.healthy === row.components.ingesters.count ? 'success' : 'warning'}
          />
        </Stack>
      ),
    },
    {
      id: 'trends',
      label: 'Trends',
      minWidth: 120,
      render: (value, row) => (
        <Stack direction="row" spacing={1}>
          <Tooltip title={`Ingestion: ${row.trends.ingestionTrend}`}>
            <TrendIcon trend={row.trends.ingestionTrend} />
          </Tooltip>
          <Tooltip title={`Errors: ${row.trends.errorTrend}`}>
            <TrendIcon trend={row.trends.errorTrend} />
          </Tooltip>
        </Stack>
      ),
    },
    {
      id: 'alerts',
      label: 'Alerts',
      minWidth: 100,
      render: (value, row) => {
        const criticalAlerts = row.alerts.filter(a => a.severity === 'critical' && !a.resolved).length;
        const warningAlerts = row.alerts.filter(a => a.severity === 'warning' && !a.resolved).length;
        
        return (
          <Stack direction="row" spacing={0.5}>
            {criticalAlerts > 0 && (
              <Badge badgeContent={criticalAlerts} color="error">
                <ErrorIcon color="error" />
              </Badge>
            )}
            {warningAlerts > 0 && (
              <Badge badgeContent={warningAlerts} color="warning">
                <WarningIcon color="warning" />
              </Badge>
            )}
            {criticalAlerts === 0 && warningAlerts === 0 && (
              <CheckCircleIcon color="success" />
            )}
          </Stack>
        );
      },
    },
    {
      id: 'lastSeen',
      label: 'Last Seen',
      minWidth: 120,
      sortable: true,
      render: (value, row) => (
        <Typography variant="body2">
          {new Date(row.lastSeen).toLocaleDateString()}
        </Typography>
      ),
    },
  ];

  // Event handlers
  const handleSearch = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
  };

  const handleStatusFilter = (event: any) => {
    setStatusFilter(event.target.value);
  };

  const handleNamespaceFilter = (event: any) => {
    setNamespaceFilter(event.target.value);
  };

  const handleSort = (field: string) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const handleRefresh = () => {
    window.location.reload();
    setSnackbar({
      open: true,
      message: 'Tenant data refreshed successfully',
      severity: 'success'
    });
  };

  const handleExport = (format: 'csv' | 'json') => {
    const data = filteredAndSortedTenants.map(tenant => ({
      name: tenant.name,
      namespace: tenant.namespace,
      status: tenant.status,
      lastSeen: tenant.lastSeen,
      alerts: tenant.alerts.filter(a => !a.resolved).length,
    }));

    if (format === 'csv') {
      const csvContent = [
        'Name,Namespace,Status,Last Seen,Active Alerts',
        ...data.map(row => `${row.name},${row.namespace},${row.status},${row.lastSeen},${row.alerts}`)
      ].join('\n');
      
      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tenants-${new Date().toISOString().split('T')[0]}.csv`;
      a.click();
      window.URL.revokeObjectURL(url);
    } else {
      const jsonContent = JSON.stringify(data, null, 2);
      const blob = new Blob([jsonContent], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tenants-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
      window.URL.revokeObjectURL(url);
    }

    setSnackbar({
      open: true,
      message: `Tenant data exported as ${format.toUpperCase()} successfully`,
      severity: 'success'
    });
  };

  const handleBulkAction = (action: string, selectedRows: EnhancedTenant[]) => {
    console.log(`Bulk action: ${action}`, selectedRows);
    setSnackbar({
      open: true,
      message: `${action} applied to ${selectedRows.length} tenants`,
      severity: 'info'
    });
  };

  const handleTenantClick = (tenant: EnhancedTenant) => {
    setSelectedTenant(tenant);
    setDetailsDialogOpen(true);
  };

  const handleAddTenant = () => {
    setAddTenantDialogOpen(true);
    setNewTenantData({ name: '', namespace: '', description: '' });
  };

  const handleSubmitNewTenant = async () => {
    if (!newTenantData.name.trim() || !newTenantData.namespace.trim()) {
      setSnackbar({
        open: true,
        message: 'Please fill in all required fields',
        severity: 'error'
      });
      return;
    }

    try {
      const response = await fetch(`${config.apiUrl}/api/tenants`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newTenantData),
      });

      if (response.ok) {
        setSnackbar({
          open: true,
          message: `Tenant "${newTenantData.name}" created successfully!`,
          severity: 'success'
        });
        setAddTenantDialogOpen(false);
        setNewTenantData({ name: '', namespace: '', description: '' });
        window.location.reload(); // Reload to show new tenant in grid
      } else {
        const error = await response.json();
        setSnackbar({
          open: true,
          message: `Failed to create tenant: ${error.error || 'Unknown error'}`,
          severity: 'error'
        });
      }
    } catch (error) {
      setSnackbar({
        open: true,
        message: `Error creating tenant: ${error}`,
        severity: 'error'
      });
    }
  };

  const getTenantActions = (tenant: EnhancedTenant) => [
    {
      label: 'View Details',
      icon: <VisibilityIcon />,
      action: () => handleTenantClick(tenant),
      color: 'primary' as const,
    },
    {
      label: 'Configure',
      icon: <SettingsIcon />,
      action: () => console.log('Configure tenant:', tenant.name),
      color: 'secondary' as const,
    },
    {
      label: 'Restart',
      icon: <RestartAltIcon />,
      action: () => console.log('Restart tenant:', tenant.name),
      color: 'warning' as const,
    },
  ];

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="400px">
        <CircularProgress size={40} />
        <Typography variant="h6" sx={{ ml: 2 }}>Loading tenants...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" component="h1" fontWeight="bold">
            <PeopleIcon sx={{ mr: 2, verticalAlign: 'middle' }} />
            Tenant Management
          </Typography>
          <Typography variant="subtitle1" color="text.secondary">
            Manage and monitor tenant configurations across your Mimir cluster
          </Typography>
        </Box>
        <Box display="flex" gap={2}>
          <FormControlLabel
            control={
              <Switch
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
              />
            }
            label="Auto Refresh"
          />
          <Button variant="outlined" startIcon={<RefreshIcon />} onClick={handleRefresh}>
            Refresh
          </Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={handleAddTenant}>
            Add Tenant
          </Button>
        </Box>
      </Box>

      {/* Statistics Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography variant="h4" fontWeight="bold" color="primary">
                    {stats.total}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Total Tenants
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'primary.main' }}>
                  <PeopleIcon />
                </Avatar>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography variant="h4" fontWeight="bold" color="success.main">
                    {stats.healthy}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Healthy
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'success.main' }}>
                  <CheckCircleIcon />
                </Avatar>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography variant="h4" fontWeight="bold" color="warning.main">
                    {stats.warning}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Warning
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'warning.main' }}>
                  <WarningIcon />
                </Avatar>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography variant="h4" fontWeight="bold" color="error.main">
                    {stats.critical}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Critical
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'error.main' }}>
                  <ErrorIcon />
                </Avatar>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Filters and Controls */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              size="small"
              placeholder="Search tenants..."
              value={searchTerm}
              onChange={handleSearch}
              InputProps={{
                startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment>,
              }}
            />
          </Grid>
          <Grid item xs={12} md={2}>
            <FormControl fullWidth size="small">
              <InputLabel>Status</InputLabel>
              <Select value={statusFilter} label="Status" onChange={handleStatusFilter}>
                <MenuItem value="all">All Status</MenuItem>
                <MenuItem value="healthy">Healthy</MenuItem>
                <MenuItem value="warning">Warning</MenuItem>
                <MenuItem value="critical">Critical</MenuItem>
                <MenuItem value="inactive">Inactive</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={2}>
            <FormControl fullWidth size="small">
              <InputLabel>Namespace</InputLabel>
              <Select value={namespaceFilter} label="Namespace" onChange={handleNamespaceFilter}>
                <MenuItem value="all">All Namespaces</MenuItem>
                {uniqueNamespaces.map((ns) => (
                  <MenuItem key={ns} value={ns}>{ns}</MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={5}>
            <Stack direction="row" spacing={1} justifyContent="flex-end">
              <Tooltip title="Sort by Name">
                <IconButton 
                  onClick={() => handleSort('name')}
                  color={sortField === 'name' ? 'primary' : 'default'}
                >
                  <SortIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Table View">
                <IconButton 
                  onClick={() => setViewMode('table')}
                  color={viewMode === 'table' ? 'primary' : 'default'}
                >
                  <ViewListIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Grid View">
                <IconButton 
                  onClick={() => setViewMode('grid')}
                  color={viewMode === 'grid' ? 'primary' : 'default'}
                >
                  <ViewModuleIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Export CSV">
                <IconButton onClick={() => handleExport('csv')}>
                  <DownloadIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Export JSON">
                <IconButton onClick={() => handleExport('json')}>
                  <DownloadIcon />
                </IconButton>
              </Tooltip>
            </Stack>
          </Grid>
        </Grid>
      </Paper>

      {/* Data Grid */}
      <DataGridWithPagination
        data={filteredAndSortedTenants}
        columns={columns}
        loading={loading}
        title="Tenants"
        subtitle={`Showing ${filteredAndSortedTenants.length} of ${tenants?.length || 0} tenants`}
        enableSearch={false} // We have our own search
        enableFilters={false} // We have our own filters
        enableSorting={true}
        enablePagination={true}
        enableExport={true}
        enableBulkActions={true}
        enableViewMode={false} // We have our own view mode
        pageSize={25}
        pageSizeOptions={[10, 25, 50, 100]}
        onRowClick={handleTenantClick}
        onBulkAction={handleBulkAction}
        onExport={handleExport}
        onRefresh={handleRefresh}
        getRowId={(row) => row.id}
        getRowStatus={(row) => {
          switch (row.status) {
            case 'healthy': return 'success';
            case 'warning': return 'warning';
            case 'critical': return 'error';
            case 'inactive': return 'default';
            default: return 'default';
          }
        }}
        getRowActions={getTenantActions}
      />

      {/* Tenant Details Dialog */}
      <Dialog
        open={detailsDialogOpen}
        onClose={() => setDetailsDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          <Box display="flex" alignItems="center" gap={2}>
            <Avatar sx={{ bgcolor: 'primary.main' }}>
              <PeopleIcon />
            </Avatar>
            <Box>
              <Typography variant="h6">{selectedTenant?.name}</Typography>
              <Typography variant="body2" color="text.secondary">
                {selectedTenant?.namespace}
              </Typography>
            </Box>
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedTenant && (
            <Grid container spacing={3}>
              <Grid item xs={12}>
                <Tabs value={selectedTab} onChange={(e: React.SyntheticEvent, newValue: number) => setSelectedTab(newValue)}>
                  <Tab label="Overview" />
                  <Tab label="Metrics" />
                  <Tab label="Configuration" />
                  <Tab label="Alerts" />
                  <Tab label="Recommendations" />
                </Tabs>
              </Grid>
              
              <Grid item xs={12}>
                {selectedTab === 0 && (
                  <Box>
                    <Typography variant="h6" mb={2}>Overview</Typography>
                    <Grid container spacing={2}>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="text.secondary">Status</Typography>
                        <StatusChip status={selectedTenant.status} />
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="text.secondary">Discovered</Typography>
                        <Typography variant="body2">
                          {new Date(selectedTenant.discoveredAt).toLocaleDateString()}
                        </Typography>
                      </Grid>
                      <Grid item xs={12}>
                        <Typography variant="body2" color="text.secondary" mb={1}>Components Health</Typography>
                        <Stack direction="row" spacing={1}>
                          <Chip 
                            label={`Alloy: ${selectedTenant.components.alloy.healthy}/${selectedTenant.components.alloy.replicas}`}
                            color={selectedTenant.components.alloy.healthy === selectedTenant.components.alloy.replicas ? 'success' : 'warning'}
                          />
                          <Chip 
                            label={`Distributors: ${selectedTenant.components.distributors.healthy}/${selectedTenant.components.distributors.count}`}
                            color={selectedTenant.components.distributors.healthy === selectedTenant.components.distributors.count ? 'success' : 'warning'}
                          />
                          <Chip 
                            label={`Ingesters: ${selectedTenant.components.ingesters.healthy}/${selectedTenant.components.ingesters.count}`}
                            color={selectedTenant.components.ingesters.healthy === selectedTenant.components.ingesters.count ? 'success' : 'warning'}
                          />
                          <Chip 
                            label={`Queriers: ${selectedTenant.components.queriers.healthy}/${selectedTenant.components.queriers.count}`}
                            color={selectedTenant.components.queriers.healthy === selectedTenant.components.queriers.count ? 'success' : 'warning'}
                          />
                        </Stack>
                      </Grid>
                    </Grid>
                  </Box>
                )}
                
                {selectedTab === 1 && (
                  <Box>
                    <Typography variant="h6" mb={2}>Metrics</Typography>
                    <Typography variant="body2" color="text.secondary">
                      Metrics visualization will be implemented with charts in a future update
                    </Typography>
                    <Box mt={2}>
                      <Typography variant="subtitle2" color="text.secondary">Recent Metrics:</Typography>
                      <List dense>
                        {selectedTenant.metrics.slice(-3).map((metric, index) => (
                          <ListItem key={index}>
                            <ListItemText
                              primary={`${metric.timestamp} - Ingestion: ${metric.ingestionRate.toLocaleString()}/s, Queries: ${metric.queryRate}/s`}
                              secondary={`CPU: ${(metric.cpuUsage * 100).toFixed(1)}%, Memory: ${(metric.memoryUsage * 100).toFixed(1)}%, Errors: ${(metric.errorRate * 100).toFixed(2)}%`}
                            />
                          </ListItem>
                        ))}
                      </List>
                    </Box>
                  </Box>
                )}
                
                {selectedTab === 2 && (
                  <Box>
                    <Typography variant="h6" mb={2}>Configuration</Typography>
                    <Grid container spacing={2}>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="text.secondary">Max Global Series Per User</Typography>
                        <Typography variant="body2">{selectedTenant.configuration.maxGlobalSeriesPerUser.toLocaleString()}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="text.secondary">Ingestion Rate Limit</Typography>
                        <Typography variant="body2">{selectedTenant.configuration.ingestionRate.toLocaleString()}/s</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="text.secondary">Query Timeout</Typography>
                        <Typography variant="body2">{selectedTenant.configuration.queryTimeout}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="text.secondary">Max Query Parallelism</Typography>
                        <Typography variant="body2">{selectedTenant.configuration.maxQueryParallelism}</Typography>
                      </Grid>
                    </Grid>
                  </Box>
                )}
                
                {selectedTab === 3 && (
                  <Box>
                    <Typography variant="h6" mb={2}>Alerts</Typography>
                    {selectedTenant.alerts.length > 0 ? (
                      <List>
                        {selectedTenant.alerts.map((alert) => (
                          <ListItem key={alert.id} divider>
                            <ListItemIcon>
                              {alert.severity === 'critical' && <ErrorIcon color="error" />}
                              {alert.severity === 'warning' && <WarningIcon color="warning" />}
                              {alert.severity === 'info' && <InfoIcon color="info" />}
                            </ListItemIcon>
                            <ListItemText
                              primary={alert.title}
                              secondary={
                                <Box>
                                  <Typography variant="body2">{alert.description}</Typography>
                                  <Typography variant="caption" color="text.secondary">
                                    {new Date(alert.timestamp).toLocaleString()}
                                  </Typography>
                                </Box>
                              }
                            />
                            <Chip 
                              label={alert.resolved ? 'Resolved' : 'Active'} 
                              color={alert.resolved ? 'success' : 'error'}
                              size="small"
                            />
                          </ListItem>
                        ))}
                      </List>
                    ) : (
                      <Typography variant="body2" color="text.secondary">No alerts for this tenant</Typography>
                    )}
                  </Box>
                )}

                {selectedTab === 4 && (
                  <Box>
                    <Typography variant="h6" mb={2}>Recommendations</Typography>
                    {selectedTenant.recommendations.length > 0 ? (
                      <List>
                        {selectedTenant.recommendations.map((rec, index) => (
                          <ListItem key={index} divider>
                            <ListItemIcon>
                              {rec.type === 'optimization' && <AutoAwesomeIcon color="primary" />}
                              {rec.type === 'scaling' && <TrendingUpIcon color="secondary" />}
                              {rec.type === 'configuration' && <SettingsIcon color="info" />}
                            </ListItemIcon>
                            <ListItemText
                              primary={
                                <Box display="flex" alignItems="center" gap={1}>
                                  {rec.title}
                                  <Chip 
                                    label={rec.impact} 
                                    color={rec.impact === 'high' ? 'error' : rec.impact === 'medium' ? 'warning' : 'success'}
                                    size="small"
                                  />
                                </Box>
                              }
                              secondary={rec.description}
                            />
                          </ListItem>
                        ))}
                      </List>
                    ) : (
                      <Typography variant="body2" color="text.secondary">No recommendations for this tenant</Typography>
                    )}
                  </Box>
                )}
              </Grid>
            </Grid>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsDialogOpen(false)}>Close</Button>
          <Button variant="contained">Configure</Button>
        </DialogActions>
      </Dialog>

      {/* Add Tenant Dialog */}
      <Dialog
        open={addTenantDialogOpen}
        onClose={() => setAddTenantDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Add New Tenant</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Tenant Name"
                value={newTenantData.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewTenantData({ ...newTenantData, name: e.target.value })}
                required
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Namespace"
                value={newTenantData.namespace}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewTenantData({ ...newTenantData, namespace: e.target.value })}
                required
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                value={newTenantData.description}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewTenantData({ ...newTenantData, description: e.target.value })}
                multiline
                rows={3}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddTenantDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSubmitNewTenant}>
            Create Tenant
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert 
          onClose={() => setSnackbar({ ...snackbar, open: false })} 
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default Tenants; 