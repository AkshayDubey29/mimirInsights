import React from 'react';
import { useLimits } from '../api/useLimits';
import { Box, CircularProgress, Typography } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';

const columns: GridColDef[] = [
  { field: 'tenant', headerName: 'Tenant', width: 150 },
  { field: 'cpuRequest', headerName: 'CPU Request', width: 120 },
  { field: 'cpuLimit', headerName: 'CPU Limit', width: 120 },
  { field: 'memoryRequest', headerName: 'Memory Request', width: 140 },
  { field: 'memoryLimit', headerName: 'Memory Limit', width: 120 },
  { field: 'recommendedCpu', headerName: 'Recommended CPU', width: 160 },
  { field: 'recommendedMemory', headerName: 'Recommended Memory', width: 180 },
];

const Limits: React.FC = () => {
  const { data: limits, loading } = useLimits();

  if (loading) return <CircularProgress />;

  return (
    <Box>
      <Typography variant="h4" gutterBottom>AI-driven Limit Recommendations</Typography>
      <div style={{ height: 400, width: '100%' }}>
        <DataGrid
          rows={limits.map((l: any, i: number) => ({ id: i, ...l }))}
          columns={columns}
          pageSize={5}
          rowsPerPageOptions={[5]}
        />
      </div>
    </Box>
  );
};

export default Limits; 