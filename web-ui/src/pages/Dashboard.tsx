import React, { useState } from 'react';
import { useTenants } from '../api/useTenants';
import { useMetrics } from '../api/useMetrics';
import { FormControl, InputLabel, Select, MenuItem, Box, CircularProgress, Typography } from '@mui/material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

const Dashboard: React.FC = () => {
  const { data: tenants, loading: tenantsLoading } = useTenants();
  const [selectedTenant, setSelectedTenant] = useState<string>('tenant-a');
  const { data: metrics, loading: metricsLoading } = useMetrics(selectedTenant);

  if (tenantsLoading || metricsLoading) return <CircularProgress />;

  const chartData = metrics?.timestamps.map((t: number, i: number) => ({
    time: `T${t}`,
    cpu: metrics.cpu[i],
    memory: metrics.memory[i],
    alloyReplicas: metrics.alloyReplicas[i],
  })) || [];

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Tenant Metrics Dashboard</Typography>
      <FormControl sx={{ minWidth: 200, mb: 3 }}>
        <InputLabel>Tenant</InputLabel>
        <Select
          value={selectedTenant}
          label="Tenant"
          onChange={e => setSelectedTenant(e.target.value)}
        >
          {tenants.map((t: any) => (
            <MenuItem key={t.name} value={t.name}>{t.name}</MenuItem>
          ))}
        </Select>
      </FormControl>
      <Box sx={{ height: 300, mb: 4 }}>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 20, right: 30, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Line type="monotone" dataKey="cpu" stroke="#8884d8" name="CPU Usage" />
            <Line type="monotone" dataKey="memory" stroke="#82ca9d" name="Memory Usage" />
            <Line type="monotone" dataKey="alloyReplicas" stroke="#ffc658" name="Alloy Replicas" />
          </LineChart>
        </ResponsiveContainer>
      </Box>
    </Box>
  );
};

export default Dashboard; 