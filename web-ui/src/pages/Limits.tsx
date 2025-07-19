import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Chip,
  Button,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Edit as EditIcon,
  Save as SaveIcon,
  Cancel as CancelIcon,
  Warning,
  CheckCircle,
  Error,
  Refresh,
} from '@mui/icons-material';
import { useLimits } from '../api/useLimits';
import { useTenants } from '../api/useTenants';

const Limits: React.FC = () => {
  const [selectedTenant, setSelectedTenant] = useState<string>('');
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingLimit, setEditingLimit] = useState<any>(null);
  const [newValue, setNewValue] = useState<string>('');

  const { data: tenants, loading: tenantsLoading, error: tenantsError } = useTenants();
  const { data: limits, loading: limitsLoading, error: limitsError, refetch: refetchLimits } = useLimits(selectedTenant);

  const handleEditLimit = (limit: any) => {
    setEditingLimit(limit);
    setNewValue(String(limit.currentValue || ''));
    setEditDialogOpen(true);
  };

  const handleSaveLimit = async () => {
    if (!editingLimit || !selectedTenant) return;

    try {
      const response = await fetch(`/api/tenants/${selectedTenant}/limits`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          limit_name: editingLimit.name,
          new_value: newValue,
          reason: 'Updated via UI',
          apply_now: true,
        }),
      });

      if (response.ok) {
        setEditDialogOpen(false);
        refetchLimits();
      } else {
        console.error('Failed to update limit');
      }
    } catch (error) {
      console.error('Error updating limit:', error);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success';
      case 'warning': return 'warning';
      case 'critical': return 'error';
      default: return 'default';
    }
  };

  if (tenantsLoading) {
    return (
      <Box sx={{ width: '100%' }}>
        <LinearProgress />
        <Typography variant="h6" sx={{ mt: 2 }}>
          Loading tenants...
        </Typography>
      </Box>
    );
  }

  if (tenantsError) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        Failed to load tenants: {tenantsError.message}
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Limits Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<Refresh />}
          onClick={() => refetchLimits()}
          disabled={!selectedTenant}
        >
          Refresh
        </Button>
      </Box>

      {/* Tenant Selection */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Select Tenant
          </Typography>
          <Grid container spacing={2}>
            {tenants?.map((tenant) => (
              <Grid item key={tenant.id}>
                <Chip
                  label={tenant.name}
                  color={selectedTenant === tenant.name ? 'primary' : 'default'}
                  onClick={() => setSelectedTenant(tenant.name)}
                  variant={selectedTenant === tenant.name ? 'filled' : 'outlined'}
                />
              </Grid>
            ))}
          </Grid>
        </CardContent>
      </Card>

      {/* Limits Display */}
      {selectedTenant && (
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Limits for {selectedTenant}
            </Typography>
            
            {limitsLoading ? (
              <Box sx={{ width: '100%' }}>
                <LinearProgress />
                <Typography variant="body2" sx={{ mt: 1 }}>
                  Loading limits...
                </Typography>
              </Box>
            ) : limitsError ? (
              <Alert severity="error">
                Failed to load limits: {limitsError.message}
              </Alert>
            ) : limits && limits.length > 0 ? (
              <TableContainer component={Paper}>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>Limit Name</TableCell>
                      <TableCell>Category</TableCell>
                      <TableCell>Current Value</TableCell>
                      <TableCell>Recommended Value</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Last Updated</TableCell>
                      <TableCell>Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {limits.map((limit) => (
                      <TableRow key={limit.name}>
                        <TableCell>{limit.name}</TableCell>
                        <TableCell>
                          <Chip label={limit.category} size="small" variant="outlined" />
                        </TableCell>
                        <TableCell>{String(limit.currentValue || 'Not set')}</TableCell>
                        <TableCell>{String(limit.recommendedValue || 'N/A')}</TableCell>
                        <TableCell>
                          <Chip
                            label={limit.status}
                            color={getStatusColor(limit.status)}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          {new Date(limit.lastUpdated).toLocaleString()}
                        </TableCell>
                        <TableCell>
                          <Tooltip title="Edit Limit">
                            <IconButton
                              size="small"
                              onClick={() => handleEditLimit(limit)}
                            >
                              <EditIcon />
                            </IconButton>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            ) : (
              <Typography variant="body2" color="textSecondary">
                No limits found for this tenant
              </Typography>
            )}
          </CardContent>
        </Card>
      )}

      {/* Edit Limit Dialog */}
      <Dialog open={editDialogOpen} onClose={() => setEditDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          Edit Limit: {editingLimit?.name}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 2 }}>
            <TextField
              fullWidth
              label="New Value"
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              sx={{ mb: 2 }}
            />
            <Typography variant="body2" color="textSecondary">
              Current: {editingLimit?.currentValue || 'Not set'} → Recommended: {editingLimit?.recommendedValue || 'N/A'}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialogOpen(false)}>
            Cancel
          </Button>
          <Button onClick={handleSaveLimit} variant="contained">
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Limits; 