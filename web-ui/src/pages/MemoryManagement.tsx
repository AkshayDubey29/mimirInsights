import React, { useState, useEffect } from 'react';
import { useMemoryStats, useForceMemoryEviction, useResetMemoryStats, useMemoryHistory } from '../api/useMemory';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CardHeader,
  LinearProgress,
  Chip,
  Button,
  IconButton,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Switch,
  FormControlLabel,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Divider,
  Stack,
  Tooltip,
  Avatar,
} from '@mui/material';
import {
  Memory as MemoryIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
  Refresh as RefreshIcon,
  Settings as SettingsIcon,
  AutoAwesome as AutoAwesomeIcon,
  Timeline as TimelineIcon,
  Speed as SpeedIcon,
  Storage as StorageIcon,
  ExpandMore as ExpandMoreIcon,
  BugReport as BugReportIcon,
  Download as DownloadIcon,
  PlayArrow as PlayArrowIcon,
  Pause as PauseIcon,
  RestartAlt as RestartAltIcon,
  People as PeopleIcon,
  CloudUpload as CloudUploadIcon,
} from '@mui/icons-material';
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

const MemoryManagement: React.FC = () => {
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [evictionDialogOpen, setEvictionDialogOpen] = useState(false);

  // Use real API hooks
  const { data: memoryStats, isLoading: loading, error, refetch } = useMemoryStats();
  const { data: memoryHistory } = useMemoryHistory();
  const forceEvictionMutation = useForceMemoryEviction();
  const resetStatsMutation = useResetMemoryStats();

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatPercentage = (value: number) => {
    return `${(value * 100).toFixed(1)}%`;
  };

  const getMemoryStatusColor = (usage: number) => {
    if (usage > 90) return 'error';
    if (usage > 80) return 'warning';
    return 'success';
  };

  const getMemoryStatusIcon = (usage: number) => {
    if (usage > 90) return <ErrorIcon />;
    if (usage > 80) return <WarningIcon />;
    return <CheckCircleIcon />;
  };

  const handleRefresh = () => {
    refetch();
  };

  const handleForceEviction = () => {
    setEvictionDialogOpen(true);
  };

  const handleResetStats = () => {
    // Reset statistics
    console.log('Resetting memory statistics');
  };

  const handleSettings = () => {
    setSettingsDialogOpen(true);
  };

  // Chart data from real API
  const chartData = memoryHistory?.map(item => ({
    time: new Date(item.timestamp).toLocaleTimeString(),
    usage: item.stats.memory_usage_percent * 100,
    cacheItems: item.stats.cache_item_count,
    evictions: item.stats.eviction_count,
  })) || [];

  const cacheDistribution = [
    { name: 'Tenant Cache', value: memoryStats?.tenant_cache_count || 0, color: '#8884d8' },
    { name: 'Mimir Cache', value: memoryStats?.mimir_cache_count || 0, color: '#82ca9d' },
    { name: 'Other Cache', value: (memoryStats?.cache_item_count || 0) - (memoryStats?.tenant_cache_count || 0) - (memoryStats?.mimir_cache_count || 0), color: '#ffc658' },
  ];

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="400px">
        <LinearProgress sx={{ width: '100%' }} />
        <Typography variant="h6" sx={{ ml: 2 }}>Loading memory statistics...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" component="h1" fontWeight="bold">
            <MemoryIcon sx={{ mr: 2, verticalAlign: 'middle' }} />
            Memory Management
          </Typography>
          <Typography variant="subtitle1" color="text.secondary">
            Precise memory monitoring and cache management
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
          <Button variant="outlined" startIcon={<SettingsIcon />} onClick={handleSettings}>
            Settings
          </Button>
          <Button variant="contained" startIcon={<AutoAwesomeIcon />}>
            AI Optimization
          </Button>
        </Box>
      </Box>

      {/* Memory Status Alert */}
      {memoryStats && memoryStats.memory_usage_percent > 80 && (
        <Alert severity="warning" sx={{ mb: 3 }} action={
          <Button color="inherit" size="small" onClick={handleForceEviction}>
            Force Eviction
          </Button>
        }>
          <strong>Memory Usage High:</strong> {formatPercentage(memoryStats.memory_usage_percent)} of available memory is in use. 
          Consider evicting cache items or increasing memory limits.
        </Alert>
      )}

      {/* Key Metrics Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: getMemoryStatusColor(memoryStats?.memory_usage_percent || 0), mr: 2 }}>
                  {getMemoryStatusIcon(memoryStats?.memory_usage_percent || 0)}
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {formatPercentage(memoryStats?.memory_usage_percent || 0)}
                  </Typography>
                  <Typography color="text.secondary">Memory Usage</Typography>
                </Box>
              </Box>
              <LinearProgress
                variant="determinate"
                value={memoryStats?.memory_usage_percent || 0}
                color={getMemoryStatusColor(memoryStats?.memory_usage_percent || 0)}
                sx={{ mt: 2, height: 8, borderRadius: 4 }}
              />
              <Typography variant="caption" color="text.secondary">
                {formatBytes(memoryStats?.current_memory_bytes || 0)} / {formatBytes(memoryStats?.max_memory_bytes || 0)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'primary.main', mr: 2 }}>
                  <StorageIcon />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {memoryStats?.cache_item_count || 0}
                  </Typography>
                  <Typography color="text.secondary">Cache Items</Typography>
                </Box>
              </Box>
              <Typography variant="caption" color="text.secondary">
                Max: {memoryStats?.max_cache_size || 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'success.main', mr: 2 }}>
                  <SpeedIcon />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {memoryStats?.eviction_count || 0}
                  </Typography>
                  <Typography color="text.secondary">Evictions</Typography>
                </Box>
              </Box>
              <Typography variant="caption" color="text.secondary">
                Last: {memoryStats?.last_eviction ? new Date(memoryStats.last_eviction).toLocaleTimeString() : 'Never'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'warning.main', mr: 2 }}>
                  <WarningIcon />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {memoryStats?.memory_warnings || 0}
                  </Typography>
                  <Typography color="text.secondary">Warnings</Typography>
                </Box>
              </Box>
              <Typography variant="caption" color="text.secondary">
                Threshold: {formatPercentage(memoryStats?.memory_threshold || 0)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Charts and Detailed Information */}
      <Grid container spacing={3}>
        {/* Memory Usage Chart */}
        <Grid item xs={12} md={8}>
          <Card>
            <CardHeader
              title="Memory Usage Over Time"
              action={
                <IconButton>
                  <ExpandMoreIcon />
                </IconButton>
              }
            />
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <AreaChart data={memoryHistory}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <RechartsTooltip />
                  <Area 
                    type="monotone" 
                    dataKey="usage" 
                    stroke="#8884d8" 
                    fill="#8884d8" 
                    fillOpacity={0.3}
                    name="Memory Usage (%)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Cache Distribution */}
        <Grid item xs={12} md={4}>
          <Card>
            <CardHeader title="Cache Distribution" />
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={cacheDistribution}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {cacheDistribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <RechartsTooltip />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Cache Details Table */}
        <Grid item xs={12}>
          <Card>
            <CardHeader
              title="Cache Details"
              action={
                <Box display="flex" gap={1}>
                  <Button size="small" variant="outlined" onClick={handleForceEviction}>
                    Force Eviction
                  </Button>
                  <Button size="small" variant="outlined" onClick={handleResetStats}>
                    Reset Stats
                  </Button>
                </Box>
              }
            />
            <CardContent>
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>Cache Type</TableCell>
                      <TableCell>Current Items</TableCell>
                      <TableCell>Max Items</TableCell>
                      <TableCell>Utilization</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell>
                        <Box display="flex" alignItems="center" gap={1}>
                          <PeopleIcon />
                          Tenant Cache
                        </Box>
                      </TableCell>
                      <TableCell>{memoryStats?.tenant_cache_count || 0}</TableCell>
                      <TableCell>{memoryStats?.max_tenant_cache_size || 0}</TableCell>
                      <TableCell>
                        <Box display="flex" alignItems="center" gap={1}>
                          <LinearProgress
                            variant="determinate"
                            value={((memoryStats?.tenant_cache_count || 0) / (memoryStats?.max_tenant_cache_size || 1)) * 100}
                            sx={{ width: 60, height: 6, borderRadius: 3 }}
                          />
                          <Typography variant="body2">
                            {((memoryStats?.tenant_cache_count || 0) / (memoryStats?.max_tenant_cache_size || 1) * 100).toFixed(1)}%
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label="Healthy" 
                          color="success" 
                          size="small"
                          icon={<CheckCircleIcon />}
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton size="small">
                          <RefreshIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>
                        <Box display="flex" alignItems="center" gap={1}>
                          <CloudUploadIcon />
                          Mimir Cache
                        </Box>
                      </TableCell>
                      <TableCell>{memoryStats?.mimir_cache_count || 0}</TableCell>
                      <TableCell>{memoryStats?.max_mimir_cache_size || 0}</TableCell>
                      <TableCell>
                        <Box display="flex" alignItems="center" gap={1}>
                          <LinearProgress
                            variant="determinate"
                            value={((memoryStats?.mimir_cache_count || 0) / (memoryStats?.max_mimir_cache_size || 1)) * 100}
                            sx={{ width: 60, height: 6, borderRadius: 3 }}
                          />
                          <Typography variant="body2">
                            {((memoryStats?.mimir_cache_count || 0) / (memoryStats?.max_mimir_cache_size || 1) * 100).toFixed(1)}%
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label="Healthy" 
                          color="success" 
                          size="small"
                          icon={<CheckCircleIcon />}
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton size="small">
                          <RefreshIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Configuration and Settings */}
        <Grid item xs={12}>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">Configuration & Settings</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle1" gutterBottom>Current Settings</Typography>
                  <Stack spacing={2}>
                    <Box display="flex" justifyContent="space-between">
                      <Typography>Eviction Policy:</Typography>
                      <Chip label={memoryStats?.eviction_policy || 'hybrid'} color="primary" />
                    </Box>
                    <Box display="flex" justifyContent="space-between">
                      <Typography>Memory Threshold:</Typography>
                      <Typography>{formatPercentage(memoryStats?.memory_threshold || 0)}</Typography>
                    </Box>
                    <Box display="flex" justifyContent="space-between">
                      <Typography>Eviction Threshold:</Typography>
                      <Typography>{formatPercentage(memoryStats?.eviction_threshold || 0)}</Typography>
                    </Box>
                    <Box display="flex" justifyContent="space-between">
                      <Typography>Max Memory:</Typography>
                      <Typography>{formatBytes(memoryStats?.max_memory_bytes || 0)}</Typography>
                    </Box>
                  </Stack>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle1" gutterBottom>Quick Actions</Typography>
                  <Stack spacing={2}>
                    <Button 
                      variant="outlined" 
                      startIcon={<PlayArrowIcon />}
                      fullWidth
                    >
                      Start Memory Monitoring
                    </Button>
                    <Button 
                      variant="outlined" 
                      startIcon={<PauseIcon />}
                      fullWidth
                    >
                      Pause Memory Monitoring
                    </Button>
                    <Button 
                      variant="outlined" 
                      startIcon={<RestartAltIcon />}
                      fullWidth
                      onClick={handleResetStats}
                    >
                      Reset Statistics
                    </Button>
                    <Button 
                      variant="outlined" 
                      startIcon={<DownloadIcon />}
                      fullWidth
                    >
                      Export Memory Report
                    </Button>
                  </Stack>
                </Grid>
              </Grid>
            </AccordionDetails>
          </Accordion>
        </Grid>
      </Grid>

      {/* Settings Dialog */}
      <Dialog open={settingsDialogOpen} onClose={() => setSettingsDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Memory Management Settings</DialogTitle>
        <DialogContent>
          <Grid container spacing={3} sx={{ mt: 1 }}>
            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Eviction Policy</InputLabel>
                <Select value={memoryStats?.eviction_policy || 'hybrid'} label="Eviction Policy">
                  <MenuItem value="lru">LRU (Least Recently Used)</MenuItem>
                  <MenuItem value="lfu">LFU (Least Frequently Used)</MenuItem>
                  <MenuItem value="ttl">TTL (Time To Live)</MenuItem>
                  <MenuItem value="size">Size-based</MenuItem>
                  <MenuItem value="hybrid">Hybrid</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Memory Threshold (%)"
                type="number"
                defaultValue={memoryStats?.memory_threshold ? memoryStats.memory_threshold * 100 : 80}
                inputProps={{ min: 50, max: 95 }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Eviction Threshold (%)"
                type="number"
                defaultValue={memoryStats?.eviction_threshold ? memoryStats.eviction_threshold * 100 : 90}
                inputProps={{ min: 70, max: 98 }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Max Memory (GB)"
                type="number"
                defaultValue={memoryStats?.max_memory_bytes ? memoryStats.max_memory_bytes / (1024 * 1024 * 1024) : 1}
                inputProps={{ min: 0.1, max: 10 }}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSettingsDialogOpen(false)}>Cancel</Button>
          <Button variant="contained">Save Settings</Button>
        </DialogActions>
      </Dialog>

      {/* Eviction Dialog */}
      <Dialog open={evictionDialogOpen} onClose={() => setEvictionDialogOpen(false)}>
        <DialogTitle>Force Memory Eviction</DialogTitle>
        <DialogContent>
          <Typography>
            This will force an immediate eviction cycle to free up memory. Are you sure you want to proceed?
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            This action may temporarily impact performance as cache items are evicted.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEvictionDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" color="warning" onClick={() => setEvictionDialogOpen(false)}>
            Force Eviction
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default MemoryManagement; 