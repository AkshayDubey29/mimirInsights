import React from 'react';
import { useReports } from '../api/useReports';
import { Box, CircularProgress, Typography, Alert, Button } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { Refresh } from '@mui/icons-material';

const columns: GridColDef[] = [
  { field: 'tenant', headerName: 'Tenant', width: 150 },
  { field: 'capacityStatus', headerName: 'Capacity Status', width: 160 },
  { field: 'alloyTuning', headerName: 'Alloy Tuning', width: 160 },
  { field: 'details', headerName: 'Details', width: 300 },
];

const Reports: React.FC = () => {
  const { data: reports, loading, error } = useReports();

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
        <Typography variant="body1" sx={{ ml: 2 }}>
          Loading capacity reports...
        </Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Box>
        <Typography variant="h4" gutterBottom>Tenant Capacity Planning Reports</Typography>
        <Alert 
          severity="error" 
          sx={{ mb: 2 }}
          action={
            <Button 
              color="inherit" 
              size="small" 
              startIcon={<Refresh />}
              onClick={() => window.location.reload()}
            >
              Retry
            </Button>
          }
        >
          Failed to load reports: {error}
        </Alert>
      </Box>
    );
  }

  // Ensure reports is always an array
  const reportsData = Array.isArray(reports) ? reports : [];

  if (reportsData.length === 0) {
    return (
      <Box>
        <Typography variant="h4" gutterBottom>Tenant Capacity Planning Reports</Typography>
        <Alert severity="info" sx={{ mb: 2 }}>
          No reports available. This might be because:
          <ul>
            <li>No tenants have been discovered yet</li>
            <li>Capacity analysis is still in progress</li>
            <li>The backend is using mock data mode</li>
          </ul>
        </Alert>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Tenant Capacity Planning Reports</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Showing {reportsData.length} capacity report{reportsData.length !== 1 ? 's' : ''} • 
        Last updated: {new Date().toLocaleTimeString()}
      </Typography>
      <div style={{ height: 400, width: '100%' }}>
        <DataGrid
          rows={reportsData.map((r: any, i: number) => ({ id: i, ...r }))}
          columns={columns}
          pageSize={5}
          rowsPerPageOptions={[5, 10, 25]}
          disableSelectionOnClick
          sx={{
            '& .MuiDataGrid-cell': {
              borderBottom: '1px solid rgba(255, 255, 255, 0.12)',
            },
            '& .MuiDataGrid-columnHeaders': {
              borderBottom: '1px solid rgba(255, 255, 255, 0.12)',
            },
          }}
        />
      </div>
    </Box>
  );
};

export default Reports; 