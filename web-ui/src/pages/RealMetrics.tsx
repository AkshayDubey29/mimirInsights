import React, { useState } from 'react';
import {
  Box, Grid, Paper, Typography, Card, CardContent, CardHeader,
  Alert, CircularProgress, LinearProgress, IconButton, FormControl,
  InputLabel, Select, MenuItem, Chip, Badge, Stack, Tooltip,
  Divider, Accordion, AccordionSummary, AccordionDetails
} from '@mui/material';
import {
  Refresh, Timeline, Storage, Memory, Speed, TrendingUp, TrendingDown,
  CloudQueue, DataUsage, MonitorHeart, ExpandMore, Settings,
  Assessment, Security, AutoAwesome, Notifications, CheckCircle,
  Error, Warning, Info
} from '@mui/icons-material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, 
         Legend, ResponsiveContainer, AreaChart, Area, BarChart, Bar } from 'recharts';
import { useRealMetrics } from '../api/useMetrics';

export default function RealMetrics() {
  const [timeRange, setTimeRange] = useState('1h');
  const { data, loading, error } = useRealMetrics(timeRange);

  const handleRefresh = () => {
    // The hook will automatically refetch when timeRange changes
    // This is just for visual feedback
  };

  const getStatusIcon = (dataSource: string) => {
    switch (dataSource) {
      case 'production':
        return <CheckCircle color="success" />;
      case 'mock':
        return <Warning color="warning" />;
      case 'mixed':
        return <Info color="info" />;
      default:
        return <Error color="error" />;
    }
  };

  const getStatusColor = (dataSource: string) => {
    switch (dataSource) {
      case 'production':
        return 'success';
      case 'mock':
        return 'warning';
      case 'mixed':
        return 'info';
      default:
        return 'error';
    }
  };

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <LinearProgress />
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <CircularProgress />
        </Box>
        <Typography variant="h6" align="center" sx={{ mt: 2 }}>
          Collecting real metrics from Mimir endpoints...
        </Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error" sx={{ mb: 3 }}>
          <Typography variant="h6">Failed to collect real metrics</Typography>
          <Typography variant="body2">{error}</Typography>
        </Alert>
      </Box>
    );
  }

  if (!data) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="info">
          No real metrics data available. Please check your Mimir configuration.
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Real Mimir Metrics
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Live metrics data collected from Mimir endpoints
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Time Range</InputLabel>
            <Select
              value={timeRange}
              label="Time Range"
              onChange={(e) => setTimeRange(e.target.value)}
            >
              <MenuItem value="1h">Last Hour</MenuItem>
              <MenuItem value="6h">Last 6 Hours</MenuItem>
              <MenuItem value="24h">Last 24 Hours</MenuItem>
              <MenuItem value="7d">Last 7 Days</MenuItem>
            </Select>
          </FormControl>
          <IconButton onClick={handleRefresh} disabled={loading}>
            <Refresh />
          </IconButton>
        </Box>
      </Box>

      {/* Data Source Status */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            {getStatusIcon(data.data_source)}
            <Typography variant="h6">
              Data Source: {data.data_source}
            </Typography>
            <Chip 
              label={data.data_source.toUpperCase()} 
              color={getStatusColor(data.data_source) as any}
              size="small"
            />
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            Collected at: {new Date(data.collected_at).toLocaleString()}
          </Typography>
        </CardContent>
      </Card>

      {/* Endpoints Status */}
      <Card sx={{ mb: 3 }}>
        <CardHeader
          title="Mimir Endpoints"
          avatar={<CloudQueue />}
        />
        <CardContent>
          <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
            {data.endpoints.map((endpoint: string, index: number) => (
              <Chip
                key={index}
                label={endpoint}
                color="primary"
                variant="outlined"
                size="small"
              />
            ))}
          </Stack>
        </CardContent>
      </Card>

      {/* Metrics Data */}
      <Grid container spacing={3}>
        {Object.entries(data.metrics).map(([endpoint, tenantMetrics]) => (
          <Grid item xs={12} key={endpoint}>
            <Accordion>
              <AccordionSummary expandIcon={<ExpandMore />}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Typography variant="h6">{endpoint}</Typography>
                  <Chip 
                    label={`${Object.keys(tenantMetrics.metrics || {}).length} metrics`}
                    size="small"
                    color="info"
                  />
                </Box>
              </AccordionSummary>
              <AccordionDetails>
                <Grid container spacing={2}>
                  {Object.entries(tenantMetrics.metrics || {}).map(([metricName, series]) => (
                    <Grid item xs={12} md={6} key={metricName}>
                      <Card variant="outlined">
                        <CardHeader
                          title={metricName.replace(/_/g, ' ').toUpperCase()}
                          subheader={`${Array.isArray(series) ? series.length : 0} series`}
                          size="small"
                        />
                        <CardContent>
                          {Array.isArray(series) && series.length > 0 ? (
                            <Box sx={{ height: 200 }}>
                              <ResponsiveContainer width="100%" height="100%">
                                <LineChart data={series[0]?.values?.map((v: any) => ({
                                  time: new Date(v.timestamp).toLocaleTimeString(),
                                  value: v.value
                                })) || []}>
                                  <CartesianGrid strokeDasharray="3 3" />
                                  <XAxis dataKey="time" />
                                  <YAxis />
                                  <RechartsTooltip />
                                  <Line type="monotone" dataKey="value" stroke="#8884d8" />
                                </LineChart>
                              </ResponsiveContainer>
                            </Box>
                          ) : (
                            <Typography variant="body2" color="text.secondary">
                              No data available
                            </Typography>
                          )}
                        </CardContent>
                      </Card>
                    </Grid>
                  ))}
                </Grid>
              </AccordionDetails>
            </Accordion>
          </Grid>
        ))}
      </Grid>

      {/* Time Range Info */}
      <Card sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Time Range Information
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Start: {new Date(data.time_range?.start).toLocaleString()}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            End: {new Date(data.time_range?.end).toLocaleString()}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Step: {data.time_range?.step}
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
} 