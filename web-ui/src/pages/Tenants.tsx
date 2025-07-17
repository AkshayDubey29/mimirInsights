import React, { useState } from 'react';
import { useTenants } from '../api/useTenants';
import { Box, CircularProgress, Typography, TextField } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';

const columns: GridColDef[] = [
  { field: 'name', headerName: 'Tenant', width: 150 },
  { field: 'namespace', headerName: 'Namespace', width: 180 },
  { field: 'status', headerName: 'Status', width: 120 },
  { field: 'cpuUsage', headerName: 'CPU Usage', width: 120, valueFormatter: ({ value }) => `${(value * 100).toFixed(0)}%` },
  { field: 'memoryUsage', headerName: 'Memory Usage', width: 140, valueFormatter: ({ value }) => `${(value * 100).toFixed(0)}%` },
  { field: 'alloyReplicas', headerName: 'Alloy Replicas', width: 140 },
];

const Tenants: React.FC = () => {
  const { data: tenants, loading } = useTenants();
  const [search, setSearch] = useState('');

  if (loading) return <CircularProgress />;

  const filtered = tenants.filter((t: any) =>
    t.name.toLowerCase().includes(search.toLowerCase()) ||
    t.namespace.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Tenants</Typography>
      <TextField
        label="Search tenants"
        variant="outlined"
        size="small"
        sx={{ mb: 2 }}
        value={search}
        onChange={e => setSearch(e.target.value)}
      />
      <div style={{ height: 400, width: '100%' }}>
        <DataGrid
          rows={filtered.map((t: any, i: number) => ({ id: i, ...t }))}
          columns={columns}
          pageSize={5}
          rowsPerPageOptions={[5]}
        />
      </div>
    </Box>
  );
};

export default Tenants; 