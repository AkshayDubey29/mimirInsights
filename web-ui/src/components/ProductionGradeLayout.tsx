import React, { useState, useMemo } from 'react';
import {
  Box,
  AppBar,
  Toolbar,
  Typography,
  IconButton,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  ListItemButton,
  Divider,
  Chip,
  Badge,
  Avatar,
  Menu,
  MenuItem,
  Tooltip,
  Alert,
  Snackbar,
  LinearProgress,
  Paper,
  Grid,
  Card,
  CardContent,
  CardHeader,
  Tabs,
  Tab,
  FormControl,
  InputLabel,
  Select,
  TextField,
  InputAdornment,
  Button,
  Switch,
  FormControlLabel,
  Collapse,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import {
  Menu as MenuIcon,
  Dashboard as DashboardIcon,
  People as PeopleIcon,
  Settings as SettingsIcon,
  Assessment as AssessmentIcon,
  Report as ReportIcon,
  CloudQueue as CloudQueueIcon,
  Memory as MemoryIcon,
  Speed as SpeedIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
  Refresh as RefreshIcon,
  Search as SearchIcon,
  FilterList as FilterIcon,
  ViewList as ViewListIcon,
  ViewModule as ViewModuleIcon,
  ExpandMore as ExpandMoreIcon,
  Notifications as NotificationsIcon,
  AccountCircle as AccountCircleIcon,
  Help as HelpIcon,
  ExitToApp as ExitToAppIcon,
  AutoAwesome as AutoAwesomeIcon,
  DataUsage as DataUsageIcon,
  Storage as StorageIcon,
  Timeline as TimelineIcon,
  Security as SecurityIcon,
  BugReport as BugReportIcon,
} from '@mui/icons-material';
import { useNavigate, useLocation } from 'react-router-dom';

interface ProductionGradeLayoutProps {
  children: React.ReactNode;
}

interface SystemStatus {
  overall: 'healthy' | 'warning' | 'critical';
  tenants: { total: number; healthy: number; warning: number; critical: number };
  memory: { usage: number; limit: number; warnings: number };
  discovery: { lastUpdate: string; cacheStatus: string };
  alerts: { critical: number; warning: number; info: number };
}

const ProductionGradeLayout: React.FC<ProductionGradeLayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [notificationsAnchor, setNotificationsAnchor] = useState<null | HTMLElement>(null);
  const [showMemoryPanel, setShowMemoryPanel] = useState(false);

  // Mock system status - in real implementation, this would come from API
  const systemStatus: SystemStatus = {
    overall: 'healthy',
    tenants: { total: 156, healthy: 142, warning: 12, critical: 2 },
    memory: { usage: 68, limit: 100, warnings: 1 },
    discovery: { lastUpdate: '2 min ago', cacheStatus: 'healthy' },
    alerts: { critical: 2, warning: 12, info: 8 }
  };

  const navigationItems = [
    { path: '/', label: 'Dashboard', icon: <DashboardIcon />, badge: systemStatus.alerts.critical },
    { path: '/tenants', label: 'Tenants', icon: <PeopleIcon />, badge: systemStatus.tenants.critical },
    { path: '/discovery', label: 'Discovery', icon: <CloudQueueIcon /> },
    { path: '/limits', label: 'Limits', icon: <SettingsIcon /> },
    { path: '/config', label: 'Configuration', icon: <AssessmentIcon /> },
    { path: '/reports', label: 'Reports', icon: <ReportIcon /> },
    { path: '/memory', label: 'Memory Management', icon: <MemoryIcon />, badge: systemStatus.memory.warnings },
  ];

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success';
      case 'warning': return 'warning';
      case 'critical': return 'error';
      default: return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircleIcon />;
      case 'warning': return <WarningIcon />;
      case 'critical': return <ErrorIcon />;
      default: return <InfoIcon />;
    }
  };

  const handleNavigation = (path: string) => {
    navigate(path);
    setDrawerOpen(false);
  };

  const handleRefresh = () => {
    // Trigger refresh of all data
    window.location.reload();
  };

  const handleViewModeChange = () => {
    setViewMode(viewMode === 'list' ? 'grid' : 'list');
  };

  const handleSearch = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
  };

  const handleStatusFilter = (event: any) => {
    setStatusFilter(event.target.value);
  };

  const handleAutoRefresh = (event: React.ChangeEvent<HTMLInputElement>) => {
    setAutoRefresh(event.target.checked);
  };

  const currentPath = location.pathname;
  const currentNavItem = navigationItems.find(item => item.path === currentPath);

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      {/* Top App Bar */}
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
        <Toolbar>
          <IconButton
            color="inherit"
            edge="start"
            onClick={() => setDrawerOpen(!drawerOpen)}
            sx={{ mr: 2 }}
          >
            <MenuIcon />
          </IconButton>

          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            MimirInsights
            {currentNavItem && (
              <Chip
                icon={getStatusIcon(systemStatus.overall)}
                label={currentNavItem.label}
                color={getStatusColor(systemStatus.overall) as any}
                size="small"
                sx={{ ml: 2 }}
              />
            )}
          </Typography>

          {/* Search Bar */}
          <TextField
            size="small"
            placeholder="Search tenants, components, alerts..."
            value={searchTerm}
            onChange={handleSearch}
            sx={{ 
              width: 300, 
              mr: 2,
              '& .MuiOutlinedInput-root': {
                backgroundColor: 'rgba(255, 255, 255, 0.1)',
                '&:hover': {
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                },
              }
            }}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
          />

          {/* Status Filter */}
          <FormControl size="small" sx={{ minWidth: 120, mr: 2 }}>
            <InputLabel sx={{ color: 'white' }}>Status</InputLabel>
            <Select
              value={statusFilter}
              label="Status"
              onChange={handleStatusFilter}
              sx={{ 
                color: 'white',
                '& .MuiOutlinedInput-notchedOutline': {
                  borderColor: 'rgba(255, 255, 255, 0.3)',
                },
                '&:hover .MuiOutlinedInput-notchedOutline': {
                  borderColor: 'rgba(255, 255, 255, 0.5)',
                },
              }}
            >
              <MenuItem value="all">All Status</MenuItem>
              <MenuItem value="healthy">Healthy</MenuItem>
              <MenuItem value="warning">Warning</MenuItem>
              <MenuItem value="critical">Critical</MenuItem>
            </Select>
          </FormControl>

          {/* View Mode Toggle */}
          <Tooltip title={`Switch to ${viewMode === 'list' ? 'grid' : 'list'} view`}>
            <IconButton color="inherit" onClick={handleViewModeChange}>
              {viewMode === 'list' ? <ViewModuleIcon /> : <ViewListIcon />}
            </IconButton>
          </Tooltip>

          {/* Auto Refresh Toggle */}
          <FormControlLabel
            control={
              <Switch
                checked={autoRefresh}
                onChange={handleAutoRefresh}
                size="small"
                sx={{ color: 'white' }}
              />
            }
            label="Auto Refresh"
            sx={{ color: 'white', mr: 2 }}
          />

          {/* Memory Panel Toggle */}
          <Tooltip title="Memory Management">
            <IconButton 
              color="inherit" 
              onClick={() => setShowMemoryPanel(!showMemoryPanel)}
              sx={{ 
                backgroundColor: systemStatus.memory.warnings > 0 ? 'rgba(255, 152, 0, 0.2)' : 'transparent',
                mr: 1
              }}
            >
              <Badge badgeContent={systemStatus.memory.warnings} color="warning">
                <MemoryIcon />
              </Badge>
            </IconButton>
          </Tooltip>

          {/* Refresh Button */}
          <Tooltip title="Refresh Data">
            <IconButton color="inherit" onClick={handleRefresh}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>

          {/* Notifications */}
          <Tooltip title="Notifications">
            <IconButton 
              color="inherit" 
              onClick={(e) => setNotificationsAnchor(e.currentTarget)}
              sx={{ mr: 1 }}
            >
              <Badge badgeContent={systemStatus.alerts.critical + systemStatus.alerts.warning} color="error">
                <NotificationsIcon />
              </Badge>
            </IconButton>
          </Tooltip>

          {/* User Menu */}
          <IconButton
            color="inherit"
            onClick={(e) => setAnchorEl(e.currentTarget)}
          >
            <AccountCircleIcon />
          </IconButton>
        </Toolbar>

        {/* System Status Bar */}
        <Box sx={{ 
          backgroundColor: 'rgba(0, 0, 0, 0.2)', 
          px: 2, 
          py: 0.5,
          display: 'flex',
          alignItems: 'center',
          gap: 3
        }}>
          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="caption" color="text.secondary">
              Tenants:
            </Typography>
            <Chip 
              label={`${systemStatus.tenants.healthy}/${systemStatus.tenants.total}`} 
              color="success" 
              size="small" 
            />
            {systemStatus.tenants.warning > 0 && (
              <Chip 
                label={systemStatus.tenants.warning} 
                color="warning" 
                size="small" 
              />
            )}
            {systemStatus.tenants.critical > 0 && (
              <Chip 
                label={systemStatus.tenants.critical} 
                color="error" 
                size="small" 
              />
            )}
          </Box>

          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="caption" color="text.secondary">
              Memory:
            </Typography>
            <Chip 
              label={`${systemStatus.memory.usage}%`} 
              color={systemStatus.memory.usage > 80 ? 'warning' : 'success'} 
              size="small" 
            />
          </Box>

          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="caption" color="text.secondary">
              Discovery:
            </Typography>
            <Chip 
              label={systemStatus.discovery.lastUpdate} 
              color="info" 
              size="small" 
            />
          </Box>

          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="caption" color="text.secondary">
              Alerts:
            </Typography>
            {systemStatus.alerts.critical > 0 && (
              <Chip 
                label={systemStatus.alerts.critical} 
                color="error" 
                size="small" 
              />
            )}
            {systemStatus.alerts.warning > 0 && (
              <Chip 
                label={systemStatus.alerts.warning} 
                color="warning" 
                size="small" 
              />
            )}
          </Box>
        </Box>
      </AppBar>

      {/* Side Navigation Drawer */}
      <Drawer
        variant="permanent"
        sx={{
          width: 280,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: 280,
            boxSizing: 'border-box',
            marginTop: '112px', // Account for AppBar + status bar
            height: 'calc(100vh - 112px)',
          },
        }}
      >
        <List>
          {navigationItems.map((item) => (
            <ListItem key={item.path} disablePadding>
              <ListItemButton
                selected={currentPath === item.path}
                onClick={() => handleNavigation(item.path)}
                sx={{
                  '&.Mui-selected': {
                    backgroundColor: 'primary.main',
                    '&:hover': {
                      backgroundColor: 'primary.dark',
                    },
                  },
                }}
              >
                <ListItemIcon sx={{ color: currentPath === item.path ? 'white' : 'inherit' }}>
                  {item.icon}
                </ListItemIcon>
                <ListItemText primary={item.label} />
                {item.badge && item.badge > 0 && (
                  <Badge badgeContent={item.badge} color="error" />
                )}
              </ListItemButton>
            </ListItem>
          ))}
        </List>

        <Divider />

        {/* Quick Actions */}
        <List>
          <ListItem>
            <Typography variant="subtitle2" color="text.secondary">
              Quick Actions
            </Typography>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton>
              <ListItemIcon>
                <AutoAwesomeIcon />
              </ListItemIcon>
              <ListItemText primary="AI Insights" />
            </ListItemButton>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton>
              <ListItemIcon>
                <BugReportIcon />
              </ListItemIcon>
              <ListItemText primary="Debug Mode" />
            </ListItemButton>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton>
              <ListItemIcon>
                <SpeedIcon />
              </ListItemIcon>
              <ListItemText primary="Performance" />
            </ListItemButton>
          </ListItem>
        </List>
      </Drawer>

      {/* Main Content Area */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          marginTop: '112px', // Account for AppBar + status bar
          height: 'calc(100vh - 112px)',
          overflow: 'auto',
          backgroundColor: 'background.default',
        }}
      >
        {/* Memory Management Panel */}
        <Collapse in={showMemoryPanel}>
          <Paper sx={{ m: 2, p: 2, backgroundColor: 'background.paper' }}>
            <Grid container spacing={2} alignItems="center">
              <Grid item xs={12} md={3}>
                <Typography variant="h6" gutterBottom>
                  Memory Management
                </Typography>
                <LinearProgress 
                  variant="determinate" 
                  value={systemStatus.memory.usage} 
                  color={systemStatus.memory.usage > 80 ? 'warning' : 'primary'}
                  sx={{ height: 8, borderRadius: 4 }}
                />
                <Typography variant="caption" color="text.secondary">
                  {systemStatus.memory.usage}% of {systemStatus.memory.limit}GB used
                </Typography>
              </Grid>
              <Grid item xs={12} md={3}>
                <Typography variant="body2" color="text.secondary">
                  Cache Items: 1,247
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Evictions: 3
                </Typography>
              </Grid>
              <Grid item xs={12} md={3}>
                <Typography variant="body2" color="text.secondary">
                  Last Eviction: 5 min ago
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Policy: Hybrid
                </Typography>
              </Grid>
              <Grid item xs={12} md={3}>
                <Button variant="outlined" size="small" sx={{ mr: 1 }}>
                  Force Eviction
                </Button>
                <Button variant="outlined" size="small">
                  Reset Stats
                </Button>
              </Grid>
            </Grid>
          </Paper>
        </Collapse>

        {/* Main Content */}
        <Box sx={{ p: 3 }}>
          {children}
        </Box>
      </Box>

      {/* Notifications Menu */}
      <Menu
        anchorEl={notificationsAnchor}
        open={Boolean(notificationsAnchor)}
        onClose={() => setNotificationsAnchor(null)}
        PaperProps={{
          sx: { width: 400, maxHeight: 500 }
        }}
      >
        <MenuItem>
          <Typography variant="h6">Notifications</Typography>
        </MenuItem>
        <Divider />
        {systemStatus.alerts.critical > 0 && (
          <MenuItem>
            <ListItemIcon>
              <ErrorIcon color="error" />
            </ListItemIcon>
            <ListItemText 
              primary={`${systemStatus.alerts.critical} Critical Alerts`}
              secondary="Requires immediate attention"
            />
          </MenuItem>
        )}
        {systemStatus.alerts.warning > 0 && (
          <MenuItem>
            <ListItemIcon>
              <WarningIcon color="warning" />
            </ListItemIcon>
            <ListItemText 
              primary={`${systemStatus.alerts.warning} Warning Alerts`}
              secondary="Review recommended"
            />
          </MenuItem>
        )}
        <MenuItem>
          <ListItemIcon>
            <InfoIcon color="info" />
          </ListItemIcon>
          <ListItemText 
            primary={`${systemStatus.alerts.info} Info Alerts`}
            secondary="For your information"
          />
        </MenuItem>
      </Menu>

      {/* User Menu */}
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        <MenuItem>
          <ListItemIcon>
            <AccountCircleIcon />
          </ListItemIcon>
          <ListItemText primary="Profile" />
        </MenuItem>
        <MenuItem>
          <ListItemIcon>
            <SettingsIcon />
          </ListItemIcon>
          <ListItemText primary="Settings" />
        </MenuItem>
        <MenuItem>
          <ListItemIcon>
            <HelpIcon />
          </ListItemIcon>
          <ListItemText primary="Help" />
        </MenuItem>
        <Divider />
        <MenuItem>
          <ListItemIcon>
            <ExitToAppIcon />
          </ListItemIcon>
          <ListItemText primary="Logout" />
        </MenuItem>
      </Menu>
    </Box>
  );
};

export default ProductionGradeLayout; 