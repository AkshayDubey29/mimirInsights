import React, { useState, useMemo, useCallback } from 'react';
import {
  Box,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  TextField,
  InputAdornment,
  IconButton,
  Chip,
  Tooltip,
  LinearProgress,
  Typography,
  Stack,
  Button,
  Menu,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
  Checkbox,
  FormControlLabel,
  Switch,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Card,
  CardContent,
  Grid,
  Avatar,
  Badge,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterIcon,
  Sort as SortIcon,
  ViewList as ViewListIcon,
  ViewModule as ViewModuleIcon,
  Download as DownloadIcon,
  Settings as SettingsIcon,
  Refresh as RefreshIcon,
  MoreVert as MoreVertIcon,
  Visibility as VisibilityIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
} from '@mui/icons-material';

export interface Column<T> {
  id: keyof T;
  label: string;
  minWidth?: number;
  align?: 'left' | 'right' | 'center';
  format?: (value: any) => React.ReactNode;
  sortable?: boolean;
  filterable?: boolean;
  searchable?: boolean;
  render?: (value: any, row: T) => React.ReactNode;
}

export interface DataGridProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  error?: string | null;
  title?: string;
  subtitle?: string;
  searchPlaceholder?: string;
  enableSearch?: boolean;
  enableFilters?: boolean;
  enableSorting?: boolean;
  enablePagination?: boolean;
  enableExport?: boolean;
  enableBulkActions?: boolean;
  enableViewMode?: boolean;
  defaultViewMode?: 'table' | 'grid' | 'card';
  pageSize?: number;
  pageSizeOptions?: number[];
  onRowClick?: (row: T) => void;
  onBulkAction?: (action: string, selectedRows: T[]) => void;
  onExport?: (format: 'csv' | 'json') => void;
  onRefresh?: () => void;
  getRowId?: (row: T) => string | number;
  getRowStatus?: (row: T) => 'success' | 'warning' | 'error' | 'info' | 'default';
  getRowActions?: (row: T) => Array<{
    label: string;
    icon: React.ReactNode;
    action: () => void;
    color?: 'primary' | 'secondary' | 'error' | 'warning' | 'info' | 'success';
  }>;
}

export function DataGridWithPagination<T extends Record<string, any>>({
  data,
  columns,
  loading = false,
  error = null,
  title,
  subtitle,
  searchPlaceholder = 'Search...',
  enableSearch = true,
  enableFilters = true,
  enableSorting = true,
  enablePagination = true,
  enableExport = true,
  enableBulkActions = false,
  enableViewMode = true,
  defaultViewMode = 'table',
  pageSize = 25,
  pageSizeOptions = [10, 25, 50, 100],
  onRowClick,
  onBulkAction,
  onExport,
  onRefresh,
  getRowId = (row) => row.id || row.name || JSON.stringify(row),
  getRowStatus,
  getRowActions,
}: DataGridProps<T>) {
  // State management
  const [searchTerm, setSearchTerm] = useState('');
  const [sortBy, setSortBy] = useState<keyof T | null>(null);
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(pageSize);
  const [viewMode, setViewMode] = useState<'table' | 'grid' | 'card'>(defaultViewMode);
  const [selectedRows, setSelectedRows] = useState<Set<string | number>>(new Set());
  const [filterAnchorEl, setFilterAnchorEl] = useState<null | HTMLElement>(null);
  const [sortAnchorEl, setSortAnchorEl] = useState<null | HTMLElement>(null);
  const [actionsAnchorEl, setActionsAnchorEl] = useState<null | HTMLElement>(null);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);

  // Filter and sort data
  const filteredAndSortedData = useMemo(() => {
    let filtered = data;

    // Apply search filter
    if (searchTerm) {
      const searchLower = searchTerm.toLowerCase();
      filtered = filtered.filter((row) =>
        columns.some((column) => {
          if (!column.searchable) return false;
          const value = row[column.id];
          if (value == null) return false;
          return String(value).toLowerCase().includes(searchLower);
        })
      );
    }

    // Apply sorting
    if (sortBy && enableSorting) {
      filtered = [...filtered].sort((a, b) => {
        const aValue = a[sortBy];
        const bValue = b[sortBy];

        if (aValue == null && bValue == null) return 0;
        if (aValue == null) return sortOrder === 'asc' ? -1 : 1;
        if (bValue == null) return sortOrder === 'asc' ? 1 : -1;

        let comparison = 0;
        if (typeof aValue === 'string' && typeof bValue === 'string') {
          comparison = aValue.localeCompare(bValue);
        } else if (typeof aValue === 'number' && typeof bValue === 'number') {
          comparison = aValue - bValue;
        } else {
          comparison = String(aValue).localeCompare(String(bValue));
        }

        return sortOrder === 'asc' ? comparison : -comparison;
      });
    }

    return filtered;
  }, [data, searchTerm, sortBy, sortOrder, columns, enableSorting]);

  // Paginate data
  const paginatedData = useMemo(() => {
    if (!enablePagination) return filteredAndSortedData;
    const startIndex = page * rowsPerPage;
    return filteredAndSortedData.slice(startIndex, startIndex + rowsPerPage);
  }, [filteredAndSortedData, page, rowsPerPage, enablePagination]);

  // Event handlers
  const handleSearch = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
    setPage(0); // Reset to first page when searching
  }, []);

  const handleSort = useCallback((columnId: keyof T) => {
    if (sortBy === columnId) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(columnId);
      setSortOrder('asc');
    }
    setPage(0);
  }, [sortBy, sortOrder]);

  const handlePageChange = useCallback((event: unknown, newPage: number) => {
    setPage(newPage);
  }, []);

  const handleRowsPerPageChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  }, []);

  const handleRowSelection = useCallback((rowId: string | number) => {
    const newSelected = new Set(selectedRows);
    if (newSelected.has(rowId)) {
      newSelected.delete(rowId);
    } else {
      newSelected.add(rowId);
    }
    setSelectedRows(newSelected);
  }, [selectedRows]);

  const handleSelectAll = useCallback(() => {
    if (selectedRows.size === paginatedData.length) {
      setSelectedRows(new Set());
    } else {
      setSelectedRows(new Set(paginatedData.map(getRowId)));
    }
  }, [selectedRows, paginatedData, getRowId]);

  const handleExport = useCallback((format: 'csv' | 'json') => {
    if (onExport) {
      onExport(format);
    }
  }, [onExport]);

  const handleBulkAction = useCallback((action: string) => {
    if (onBulkAction) {
      const selectedData = data.filter(row => selectedRows.has(getRowId(row)));
      onBulkAction(action, selectedData);
    }
    setActionsAnchorEl(null);
  }, [onBulkAction, selectedRows, data, getRowId]);

  // Render functions
  const renderCell = useCallback((column: Column<T>, value: any, row: T) => {
    if (column.render) {
      return column.render(value, row);
    }
    if (column.format) {
      return column.format(value);
    }
    return value != null ? String(value) : '';
  }, []);

  const renderStatusIcon = useCallback((status: string) => {
    switch (status) {
      case 'success': return <CheckCircleIcon color="success" />;
      case 'warning': return <WarningIcon color="warning" />;
      case 'error': return <ErrorIcon color="error" />;
      case 'info': return <InfoIcon color="info" />;
      default: return <InfoIcon color="action" />;
    }
  }, []);

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Paper sx={{ width: '100%', overflow: 'hidden' }}>
      {/* Header */}
      {(title || subtitle || enableSearch || enableFilters || enableExport || enableViewMode) && (
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" spacing={2}>
            <Box>
              {title && <Typography variant="h6">{title}</Typography>}
              {subtitle && <Typography variant="body2" color="text.secondary">{subtitle}</Typography>}
            </Box>
            
            <Stack direction="row" spacing={1}>
              {enableSearch && (
                <TextField
                  size="small"
                  placeholder={searchPlaceholder}
                  value={searchTerm}
                  onChange={handleSearch}
                  InputProps={{
                    startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment>,
                  }}
                  sx={{ minWidth: 200 }}
                />
              )}
              
              {enableFilters && (
                <Tooltip title="Filters">
                  <IconButton onClick={(e) => setFilterAnchorEl(e.currentTarget)}>
                    <FilterIcon />
                  </IconButton>
                </Tooltip>
              )}
              
              {enableSorting && (
                <Tooltip title="Sort">
                  <IconButton onClick={(e) => setSortAnchorEl(e.currentTarget)}>
                    <SortIcon />
                  </IconButton>
                </Tooltip>
              )}
              
              {enableViewMode && (
                <Tooltip title="View Mode">
                  <IconButton onClick={() => setViewMode(viewMode === 'table' ? 'grid' : 'table')}>
                    {viewMode === 'table' ? <ViewModuleIcon /> : <ViewListIcon />}
                  </IconButton>
                </Tooltip>
              )}
              
              {enableExport && (
                <Tooltip title="Export">
                  <IconButton onClick={(e) => setActionsAnchorEl(e.currentTarget)}>
                    <DownloadIcon />
                  </IconButton>
                </Tooltip>
              )}
              
              {onRefresh && (
                <Tooltip title="Refresh">
                  <IconButton onClick={onRefresh} disabled={loading}>
                    <RefreshIcon />
                  </IconButton>
                </Tooltip>
              )}
            </Stack>
          </Stack>
        </Box>
      )}

      {/* Loading indicator */}
      {loading && <LinearProgress />}

      {/* Data display */}
      {viewMode === 'table' ? (
        <>
          <TableContainer sx={{ maxHeight: 600 }}>
            <Table stickyHeader>
              <TableHead>
                <TableRow>
                  {enableBulkActions && (
                    <TableCell padding="checkbox">
                      <Checkbox
                        indeterminate={selectedRows.size > 0 && selectedRows.size < paginatedData.length}
                        checked={selectedRows.size === paginatedData.length && paginatedData.length > 0}
                        onChange={handleSelectAll}
                      />
                    </TableCell>
                  )}
                  {columns.map((column) => (
                    <TableCell
                      key={String(column.id)}
                      align={column.align}
                      style={{ minWidth: column.minWidth }}
                      sx={{
                        cursor: column.sortable ? 'pointer' : 'default',
                        '&:hover': column.sortable ? { backgroundColor: 'action.hover' } : {},
                      }}
                      onClick={() => column.sortable && handleSort(column.id)}
                    >
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <Typography variant="subtitle2">{column.label}</Typography>
                        {column.sortable && sortBy === column.id && (
                          <SortIcon sx={{ fontSize: 16, transform: sortOrder === 'desc' ? 'rotate(180deg)' : 'none' }} />
                        )}
                      </Stack>
                    </TableCell>
                  ))}
                  {getRowActions && <TableCell align="center">Actions</TableCell>}
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedData.map((row) => {
                  const rowId = getRowId(row);
                  const isSelected = selectedRows.has(rowId);
                  const status = getRowStatus ? getRowStatus(row) : 'default';
                  
                  return (
                    <TableRow
                      key={rowId}
                      hover
                      selected={isSelected}
                      onClick={() => onRowClick?.(row)}
                      sx={{ cursor: onRowClick ? 'pointer' : 'default' }}
                    >
                      {enableBulkActions && (
                        <TableCell padding="checkbox">
                          <Checkbox
                            checked={isSelected}
                            onChange={() => handleRowSelection(rowId)}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </TableCell>
                      )}
                      {columns.map((column) => (
                        <TableCell key={String(column.id)} align={column.align}>
                          {renderCell(column, row[column.id], row)}
                        </TableCell>
                      ))}
                      {getRowActions && (
                        <TableCell align="center">
                          <Stack direction="row" spacing={1}>
                            {getRowActions(row).map((action, index) => (
                              <Tooltip key={index} title={action.label}>
                                <IconButton
                                  size="small"
                                  color={action.color}
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    action.action();
                                  }}
                                >
                                  {action.icon}
                                </IconButton>
                              </Tooltip>
                            ))}
                          </Stack>
                        </TableCell>
                      )}
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
          
          {enablePagination && (
            <TablePagination
              rowsPerPageOptions={pageSizeOptions}
              component="div"
              count={filteredAndSortedData.length}
              rowsPerPage={rowsPerPage}
              page={page}
              onPageChange={handlePageChange}
              onRowsPerPageChange={handleRowsPerPageChange}
            />
          )}
        </>
      ) : (
        <Box sx={{ p: 2 }}>
          <Grid container spacing={2}>
            {paginatedData.map((row) => {
              const rowId = getRowId(row);
              const status = getRowStatus ? getRowStatus(row) : 'default';
              
              return (
                <Grid item xs={12} sm={6} md={4} lg={3} key={rowId}>
                  <Card
                    sx={{
                      cursor: onRowClick ? 'pointer' : 'default',
                      '&:hover': onRowClick ? { boxShadow: 4 } : {},
                    }}
                    onClick={() => onRowClick?.(row)}
                  >
                    <CardContent>
                      <Stack direction="row" alignItems="center" spacing={1} mb={1}>
                        {renderStatusIcon(status)}
                        <Typography variant="h6" noWrap>
                          {row.name || row.title || String(rowId)}
                        </Typography>
                      </Stack>
                      
                      {columns.slice(0, 3).map((column) => (
                        <Box key={String(column.id)} sx={{ mb: 1 }}>
                          <Typography variant="caption" color="text.secondary">
                            {column.label}:
                          </Typography>
                          <Typography variant="body2">
                            {renderCell(column, row[column.id], row)}
                          </Typography>
                        </Box>
                      ))}
                      
                      {getRowActions && (
                        <Stack direction="row" spacing={1} mt={1}>
                          {getRowActions(row).slice(0, 3).map((action, index) => (
                            <Tooltip key={index} title={action.label}>
                              <IconButton
                                size="small"
                                color={action.color}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  action.action();
                                }}
                              >
                                {action.icon}
                              </IconButton>
                            </Tooltip>
                          ))}
                        </Stack>
                      )}
                    </CardContent>
                  </Card>
                </Grid>
              );
            })}
          </Grid>
        </Box>
      )}

      {/* Menus */}
      <Menu
        anchorEl={filterAnchorEl}
        open={Boolean(filterAnchorEl)}
        onClose={() => setFilterAnchorEl(null)}
      >
        <MenuItem>Filter options will be implemented here</MenuItem>
      </Menu>

      <Menu
        anchorEl={sortAnchorEl}
        open={Boolean(sortAnchorEl)}
        onClose={() => setSortAnchorEl(null)}
      >
        {columns.filter(col => col.sortable).map((column) => (
          <MenuItem
            key={String(column.id)}
            onClick={() => {
              handleSort(column.id);
              setSortAnchorEl(null);
            }}
          >
            {column.label}
          </MenuItem>
        ))}
      </Menu>

      <Menu
        anchorEl={actionsAnchorEl}
        open={Boolean(actionsAnchorEl)}
        onClose={() => setActionsAnchorEl(null)}
      >
        <MenuItem onClick={() => handleExport('csv')}>Export as CSV</MenuItem>
        <MenuItem onClick={() => handleExport('json')}>Export as JSON</MenuItem>
        {enableBulkActions && selectedRows.size > 0 && (
          <>
            <MenuItem onClick={() => handleBulkAction('delete')}>Delete Selected</MenuItem>
            <MenuItem onClick={() => handleBulkAction('update')}>Update Selected</MenuItem>
          </>
        )}
      </Menu>
    </Paper>
  );
} 