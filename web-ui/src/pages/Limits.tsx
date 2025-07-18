import React, { useState } from 'react';
import {
  Box, Grid, Paper, Typography, Table, TableBody, TableCell, TableContainer, 
  TableHead, TableRow, Button, Chip, LinearProgress, Card, CardContent,
  Alert, Dialog, DialogTitle, DialogContent, DialogActions, TextField,
  FormControl, InputLabel, Select, MenuItem, Stack, Divider, Avatar,
  Tooltip, IconButton, Badge, CircularProgress
} from '@mui/material';
import {
  TrendingUp, TrendingDown, Warning, CheckCircle, AutoAwesome,
  Edit, Save, Cancel, Timeline, Speed, Memory, DataUsage,
  Insights, Assessment, Security
} from '@mui/icons-material';
import { useLimits } from '../api/useLimits';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, 
         Legend, ResponsiveContainer, BarChart, Bar } from 'recharts';

interface MimirLimit {
  tenant: string;
  limitType: string;
  currentValue: number;
  observedPeak: number;
  recommendedValue: number;
  utilization: number;
  status: 'healthy' | 'warning' | 'critical';
  rejectedSamples: number;
  trend: 'up' | 'down' | 'stable';
  lastUpdated: string;
  description: string;
}

interface LimitRecommendation {
  tenant: string;
  category: 'ingestion' | 'series' | 'memory' | 'query';
  priority: 'high' | 'medium' | 'low';
  impact: string;
  estimatedSavings?: string;
}

const Limits: React.FC = () => {
  const { data: limits, loading } = useLimits();
  const [selectedTenant, setSelectedTenant] = useState<string>('all');
  const [editingLimit, setEditingLimit] = useState<string | null>(null);
  const [showRecommendations, setShowRecommendations] = useState(false);
  const [showAuditDialog, setShowAuditDialog] = useState(false);
  const [auditLogs, setAuditLogs] = useState<any[]>([]);
  const [loadingAudit, setLoadingAudit] = useState(false);

  // Function to fetch audit logs
  const fetchAuditLogs = async () => {
    setLoadingAudit(true);
    try {
      const response = await fetch('http://localhost:8080/api/audit');
      const data = await response.json();
      setAuditLogs(data.audit_logs || []);
    } catch (error) {
      console.error('Error fetching audit logs:', error);
      setAuditLogs([]);
    } finally {
      setLoadingAudit(false);
    }
  };

  // Function to open audit dialog
  const handleAuditConfiguration = () => {
    setShowAuditDialog(true);
    fetchAuditLogs();
  };

  // Mock data for Mimir-specific limits
  const [mimirLimits, setMimirLimits] = useState<MimirLimit[]>([
    {
      tenant: 'eats',
      limitType: 'max_global_series_per_user',
      currentValue: 2000000,
      observedPeak: 2400000,
      recommendedValue: 2640000,
      utilization: 120,
      status: 'critical',
      rejectedSamples: 15000,
      trend: 'up',
      lastUpdated: '2 hours ago',
      description: 'Maximum number of series per tenant'
    },
    {
      tenant: 'eats',
      limitType: 'ingestion_rate',
      currentValue: 150000,
      observedPeak: 174000,
      recommendedValue: 190000,
      utilization: 116,
      status: 'critical',
      rejectedSamples: 2300,
      trend: 'up',
      lastUpdated: '5 min ago',
      description: 'Samples per second ingestion rate'
    },
    {
      tenant: 'transportation',
      limitType: 'max_label_names_per_series',
      currentValue: 30,
      observedPeak: 28,
      recommendedValue: 32,
      utilization: 93,
      status: 'warning',
      rejectedSamples: 0,
      trend: 'stable',
      lastUpdated: '1 hour ago',
      description: 'Maximum labels per series'
    },
    {
      tenant: 'marketplace',
      limitType: 'max_global_series_per_metric',
      currentValue: 50000,
      observedPeak: 22000,
      recommendedValue: 45000,
      utilization: 44,
      status: 'healthy',
      rejectedSamples: 0,
      trend: 'down',
      lastUpdated: '30 min ago',
      description: 'Maximum series per metric name'
    }
  ]);

  const [recommendations, setRecommendations] = useState<LimitRecommendation[]>([
    {
      tenant: 'eats',
      category: 'ingestion',
      priority: 'high',
      impact: 'Prevent 2,300+ rejected samples per day',
    },
    {
      tenant: 'transportation',
      category: 'memory',
      priority: 'medium',
      impact: 'Reduce memory usage by 15%',
      estimatedSavings: '$240/month'
    },
    {
      tenant: 'marketplace',
      category: 'series',
      priority: 'low',
      impact: 'Optimize storage allocation',
      estimatedSavings: '$120/month'
    }
  ]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success';
      case 'warning': return 'warning';
      case 'critical': return 'error';
      default: return 'default';
    }
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up': return <TrendingUp color="error" />;
      case 'down': return <TrendingDown color="success" />;
      default: return <Timeline color="disabled" />;
    }
  };

  const formatNumber = (num: number) => {
    if (num >= 1e6) return `${(num / 1e6).toFixed(1)}M`;
    if (num >= 1e3) return `${(num / 1e3).toFixed(1)}K`;
    return num.toString();
  };

  const applyRecommendation = (limit: MimirLimit) => {
    setMimirLimits(prev => 
      prev.map(l => 
        l.tenant === limit.tenant && l.limitType === limit.limitType
          ? { ...l, currentValue: l.recommendedValue, status: 'healthy', utilization: 85 }
          : l
      )
    );
  };

  // Chart data for limit trends
  const trendData = Array.from({ length: 7 }, (_, i) => ({
    day: `Day ${i + 1}`,
    utilization: Math.random() * 40 + 60,
    rejections: Math.random() * 1000,
  }));

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="400px">
        <Typography variant="h6">Loading Mimir Limits...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" component="h1" fontWeight="bold">
            <Assessment sx={{ mr: 2, verticalAlign: 'middle' }} />
            Mimir Tenant Limits
          </Typography>
          <Typography variant="subtitle1" color="text.secondary">
            AI-generated limit recommendations and configuration management
          </Typography>
        </Box>
        <Box display="flex" gap={2}>
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel>Tenant Filter</InputLabel>
            <Select value={selectedTenant} label="Tenant Filter" onChange={(e) => setSelectedTenant(e.target.value)}>
              <MenuItem value="all">All Tenants</MenuItem>
              <MenuItem value="eats">eats</MenuItem>
              <MenuItem value="transportation">transportation</MenuItem>
              <MenuItem value="marketplace">marketplace</MenuItem>
            </Select>
          </FormControl>
          <Button 
            variant="contained" 
            startIcon={<AutoAwesome />}
            onClick={() => setShowRecommendations(true)}
          >
            View AI Recommendations
          </Button>
          <Button variant="outlined" startIcon={<Security />} onClick={handleAuditConfiguration}>
            Audit Configuration
          </Button>
        </Box>
      </Box>

      {/* Summary Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Avatar sx={{ bgcolor: 'error.main', mr: 2 }}>
                  <Warning />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {mimirLimits.filter(l => l.status === 'critical').length}
                  </Typography>
                  <Typography color="text.secondary">Critical Limits</Typography>
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
                  <TrendingUp />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {mimirLimits.filter(l => l.rejectedSamples > 0).reduce((sum, l) => sum + l.rejectedSamples, 0).toLocaleString()}
                  </Typography>
                  <Typography color="text.secondary">Rejected Samples (24h)</Typography>
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
                  <Insights />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {recommendations.length}
                  </Typography>
                  <Typography color="text.secondary">AI Recommendations</Typography>
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
                  <CheckCircle />
                </Avatar>
                <Box>
                  <Typography variant="h4" fontWeight="bold">
                    {mimirLimits.filter(l => l.status === 'healthy').length}
                  </Typography>
                  <Typography color="text.secondary">Optimized Limits</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Limits Table */}
      <Paper sx={{ mb: 4 }}>
        <Box sx={{ p: 3 }}>
          <Typography variant="h6" gutterBottom>Current Limit Configuration vs Recommendations</Typography>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Tenant</TableCell>
                  <TableCell>Limit Type</TableCell>
                  <TableCell>Current Value</TableCell>
                  <TableCell>Observed Peak</TableCell>
                  <TableCell>AI Recommendation</TableCell>
                  <TableCell>Utilization</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Trend</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {mimirLimits
                  .filter(limit => selectedTenant === 'all' || limit.tenant === selectedTenant)
                  .map((limit) => (
                  <TableRow key={`${limit.tenant}-${limit.limitType}`}>
                    <TableCell>
                      <Box display="flex" alignItems="center">
                        <Avatar sx={{ width: 24, height: 24, mr: 1, fontSize: 12 }}>
                          {limit.tenant.charAt(0).toUpperCase()}
                        </Avatar>
                        {limit.tenant}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Tooltip title={limit.description}>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                          {limit.limitType}
                        </Typography>
                      </Tooltip>
                    </TableCell>
                    <TableCell>{formatNumber(limit.currentValue)}</TableCell>
                    <TableCell>
                      <Box display="flex" alignItems="center">
                        {formatNumber(limit.observedPeak)}
                        {limit.observedPeak > limit.currentValue && (
                          <Warning color="error" sx={{ ml: 1, fontSize: 16 }} />
                        )}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box display="flex" alignItems="center">
                        <Typography fontWeight="bold" color="primary.main">
                          {formatNumber(limit.recommendedValue)}
                        </Typography>
                        <Chip 
                          label={`+${(((limit.recommendedValue - limit.currentValue) / limit.currentValue) * 100).toFixed(0)}%`}
                          color="info"
                          size="small"
                          sx={{ ml: 1 }}
                        />
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box display="flex" alignItems="center">
                        {limit.utilization}%
                        <LinearProgress 
                          variant="determinate" 
                          value={Math.min(limit.utilization, 100)} 
                          sx={{ ml: 1, width: 60 }}
                          color={limit.utilization > 100 ? 'error' : limit.utilization > 80 ? 'warning' : 'primary'}
                        />
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={limit.status} 
                        color={getStatusColor(limit.status) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{getTrendIcon(limit.trend)}</TableCell>
                    <TableCell>
                      <Stack direction="row" spacing={1}>
                        <Button 
                          size="small" 
                          variant="contained" 
                          onClick={() => applyRecommendation(limit)}
                          disabled={limit.status === 'healthy'}
                        >
                          Apply
                        </Button>
                        <IconButton size="small" onClick={() => setEditingLimit(`${limit.tenant}-${limit.limitType}`)}>
                          <Edit fontSize="small" />
                        </IconButton>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      </Paper>

      {/* Trend Analysis */}
      <Grid container spacing={3}>
        <Grid item xs={12} lg={8}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>Limit Utilization Trends (7 days)</Typography>
            <Box height={300}>
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={trendData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="day" />
                  <YAxis />
                  <RechartsTooltip />
                  <Legend />
                  <Line type="monotone" dataKey="utilization" stroke="#8884d8" name="Utilization %" />
                </LineChart>
              </ResponsiveContainer>
            </Box>
          </Paper>
        </Grid>
        <Grid item xs={12} lg={4}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>Rejected Samples by Day</Typography>
            <Box height={300}>
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={trendData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="day" />
                  <YAxis />
                  <RechartsTooltip />
                  <Bar dataKey="rejections" fill="#ff7300" name="Rejections" />
                </BarChart>
              </ResponsiveContainer>
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* AI Recommendations Dialog */}
      <Dialog open={showRecommendations} onClose={() => setShowRecommendations(false)} maxWidth="md" fullWidth>
        <DialogTitle>
          <Box display="flex" alignItems="center">
            <AutoAwesome sx={{ mr: 2 }} />
            AI-Generated Limit Recommendations
          </Box>
        </DialogTitle>
        <DialogContent>
          <Stack spacing={3}>
            {recommendations.map((rec, index) => (
              <Alert 
                key={index}
                severity={rec.priority === 'high' ? 'error' : rec.priority === 'medium' ? 'warning' : 'info'}
                action={
                  <Button color="inherit" size="small">
                    Apply
                  </Button>
                }
              >
                <Typography variant="subtitle2">
                  {rec.tenant} - {rec.category} optimization
                </Typography>
                <Typography variant="body2">
                  {rec.impact}
                  {rec.estimatedSavings && ` • Potential savings: ${rec.estimatedSavings}`}
                </Typography>
              </Alert>
            ))}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowRecommendations(false)}>Close</Button>
          <Button variant="contained">Apply All Recommendations</Button>
        </DialogActions>
      </Dialog>

      {/* Audit Configuration Dialog */}
      <Dialog open={showAuditDialog} onClose={() => setShowAuditDialog(false)} maxWidth="lg" fullWidth>
        <DialogTitle>
          <Box display="flex" alignItems="center">
            <Security sx={{ mr: 2 }} />
            Audit Configuration & Logs
          </Box>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" gutterBottom>Configuration Audit Logs</Typography>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              Track all limit configuration changes and system actions
            </Typography>
          </Box>
          
          {loadingAudit ? (
            <Box display="flex" justifyContent="center" p={3}>
              <CircularProgress />
            </Box>
          ) : (
            <TableContainer component={Paper} sx={{ maxHeight: 400 }}>
              <Table stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell>Timestamp</TableCell>
                    <TableCell>Action</TableCell>
                    <TableCell>Tenant</TableCell>
                    <TableCell>User</TableCell>
                    <TableCell>Description</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {auditLogs.map((log, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <Typography variant="body2">
                          {new Date(log.timestamp).toLocaleString()}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={log.action}
                          variant="outlined"
                          size="small"
                        />
                      </TableCell>
                      <TableCell>{log.tenant}</TableCell>
                      <TableCell>{log.user}</TableCell>
                      <TableCell>
                        <Typography variant="body2">
                          {log.description}
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ))}
                  {auditLogs.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} align="center">
                        <Typography variant="body2" color="text.secondary">
                          No audit logs available
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowAuditDialog(false)}>Close</Button>
          <Button variant="outlined" onClick={fetchAuditLogs}>Refresh</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Limits; 