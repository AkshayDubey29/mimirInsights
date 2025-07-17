import React from 'react';
import { useReports } from '../api/useReports';
import { Box, CircularProgress, Typography } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';

const columns: GridColDef[] = [
  { field: 'tenant', headerName: 'Tenant', width: 150 },
  { field: 'capacityStatus', headerName: 'Capacity Status', width: 160 },
  { field: 'alloyTuning', headerName: 'Alloy Tuning', width: 160 },
  { field: 'details', headerName: 'Details', width: 300 },
];

const Reports: React.FC = () => {
  const { data: reports, loading } = useReports();

  if (loading) return <CircularProgress />;

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Tenant Capacity Planning Reports</Typography>
      <div style={{ height: 400, width: '100%' }}>
        <DataGrid
          rows={reports.map((r: any, i: number) => ({ id: i, ...r }))}
          columns={columns}
          pageSize={5}
          rowsPerPageOptions={[5]}
        />
      </div>
    </Box>
  );
};

export default Reports; 