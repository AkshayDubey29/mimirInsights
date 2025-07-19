import React from 'react';
import { useConfigs } from '../api/useConfigs';
import { Box, CircularProgress, Typography } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';

const columns: GridColDef[] = [
  { field: 'tenant', headerName: 'Tenant', width: 150 },
  { field: 'auditStatus', headerName: 'Audit Status', width: 140 },
  { field: 'configDrift', headerName: 'Config Drift', width: 120, valueFormatter: ({ value }) => value ? 'Yes' : 'No' },
  { field: 'details', headerName: 'Details', width: 300 },
];

const Config: React.FC = () => {
  const { data: configs, loading } = useConfigs();

  if (loading) return <CircularProgress />;

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Configuration Audit & Drift Detection</Typography>
      <div style={{ height: 400, width: '100%' }}>
        <DataGrid
          rows={(configs || []).map((c: any, i: number) => ({ id: i, ...c }))}
          columns={columns}
          pageSize={5}
          rowsPerPageOptions={[5]}
        />
      </div>
    </Box>
  );
};

export default Config; 