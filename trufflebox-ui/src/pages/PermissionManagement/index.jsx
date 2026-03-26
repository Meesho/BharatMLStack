import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  TextField,
  Button,
  Alert,
  Snackbar,
  Skeleton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
  IconButton,
  Checkbox,
  ListItemText,
  Tooltip,
  Divider,
  Grid,
  TableSortLabel,
  Pagination,
  CircularProgress,
  Autocomplete,
  Fade,
  Slide,
  Switch,
  FormControlLabel,
  Tabs,
  Tab,
  Badge,
  Avatar,
  alpha,
  useTheme,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import SaveIcon from '@mui/icons-material/Save';
import CloseIcon from '@mui/icons-material/Close';
import SecurityIcon from '@mui/icons-material/Security';
import FilterListIcon from '@mui/icons-material/FilterList';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import RefreshIcon from '@mui/icons-material/Refresh';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningIcon from '@mui/icons-material/Warning';
import PersonIcon from '@mui/icons-material/Person';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import SupervisorAccountIcon from '@mui/icons-material/SupervisorAccount';
import VisibilityIcon from '@mui/icons-material/Visibility';
import BuildIcon from '@mui/icons-material/Build';
import LayersIcon from '@mui/icons-material/Layers';
import SettingsIcon from '@mui/icons-material/Settings';
import AppsIcon from '@mui/icons-material/Apps';
import TuneIcon from '@mui/icons-material/Tune';
import { useAuth } from '../Auth/AuthContext';
import * as URL_CONSTANTS from '../../config';

// Role config with colors and icons
const ROLE_CONFIG = {
  super_admin: {
    label: 'Super Admin',
    color: '#dc2626',
    bgColor: '#fef2f2',
    icon: <SupervisorAccountIcon fontSize="small" />,
  },
  admin: {
    label: 'Admin',
    color: '#d97706',
    bgColor: '#fffbeb',
    icon: <AdminPanelSettingsIcon fontSize="small" />,
  },
  user: {
    label: 'User',
    color: '#2563eb',
    bgColor: '#eff6ff',
    icon: <PersonIcon fontSize="small" />,
  },
};

// Action icons
const ACTION_ICONS = {
  view: <VisibilityIcon fontSize="small" />,
  edit: <EditIcon fontSize="small" />,
  onboard: <AddIcon fontSize="small" />,
  delete: <DeleteIcon fontSize="small" />,
  approve: <CheckCircleIcon fontSize="small" />,
};

const PermissionManagement = () => {
  const theme = useTheme();
  const [permissions, setPermissions] = useState([]);
  const [filteredPermissions, setFilteredPermissions] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [roleFilter, setRoleFilter] = useState('all');
  const [serviceFilter, setServiceFilter] = useState('all');
  const [loading, setLoading] = useState(true);
  const [updateStatus, setUpdateStatus] = useState({ message: '', type: '', show: false });
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [bulkUpdateDialogOpen, setBulkUpdateDialogOpen] = useState(false);
  const [selectedPermission, setSelectedPermission] = useState(null);
  const [editingPermission, setEditingPermission] = useState(null);
  const [bulkUpdateRole, setBulkUpdateRole] = useState('user');
  const [bulkUpdatePermissions, setBulkUpdatePermissions] = useState({});
  const [saving, setSaving] = useState(false);
  const [sortConfig, setSortConfig] = useState({ field: 'service_name', direction: 'asc' });
  const [page, setPage] = useState(1);
  const [rowsPerPage] = useState(10);
  const [activeTab, setActiveTab] = useState(0);
  
  // Metadata state
  const [services, setServices] = useState([]);
  const [allScreenTypes, setAllScreenTypes] = useState([]);
  const [actions, setActions] = useState([]);
  const [loadingMetadata, setLoadingMetadata] = useState(true);
  
  const { user } = useAuth();
  const isSuperAdmin = user?.role === 'super_admin';

  // Fetch metadata
  useEffect(() => {
    const fetchMetadata = async () => {
      if (!isSuperAdmin || !user?.token) {
        setLoadingMetadata(false);
        return;
      }

      try {
        setLoadingMetadata(true);
        const [servicesRes, screenTypesRes, actionsRes] = await Promise.all([
          fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/metadata/services`, {
            headers: { 'Authorization': `Bearer ${user.token}` }
          }),
          fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/metadata/screen-types`, {
            headers: { 'Authorization': `Bearer ${user.token}` }
          }),
          fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/metadata/actions`, {
            headers: { 'Authorization': `Bearer ${user.token}` }
          })
        ]);

        if (servicesRes.ok) {
          const data = await servicesRes.json();
          setServices(data.services || []);
        }

        if (screenTypesRes.ok) {
          const data = await screenTypesRes.json();
          setAllScreenTypes(data.screen_types || []);
        }

        if (actionsRes.ok) {
          const data = await actionsRes.json();
          setActions(data.actions || []);
        }
      } catch (error) {
        console.error('Error fetching metadata:', error);
        setUpdateStatus({
          message: 'Failed to fetch metadata. Please refresh the page.',
          type: 'error',
          show: true
        });
      } finally {
        setLoadingMetadata(false);
      }
    };

    if (user?.token && isSuperAdmin) {
      fetchMetadata();
    }
  }, [user?.token, isSuperAdmin]);

  // Filter screen types based on selected service
  const availableScreenTypes = useMemo(() => {
    if (!editingPermission?.service_id) {
      return [];
    }
    return allScreenTypes.filter(st => st.service_id === editingPermission.service_id);
  }, [editingPermission?.service_id, allScreenTypes]);

  // Fetch permissions
  const fetchPermissions = async () => {
    if (!isSuperAdmin) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/permissions`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch permissions');
      }

      const data = await response.json();
      setPermissions(data);
    } catch (error) {
      setUpdateStatus({
        message: 'Failed to fetch permissions. Please try again.',
        type: 'error',
        show: true
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (user?.token && isSuperAdmin) {
      fetchPermissions();
    }
  }, [user?.token, isSuperAdmin]);

  // Filter and sort permissions
  useEffect(() => {
    let filtered = [...permissions];

    if (roleFilter !== 'all') {
      filtered = filtered.filter(p => p.role === roleFilter);
    }

    if (serviceFilter !== 'all') {
      filtered = filtered.filter(p => p.service_id === parseInt(serviceFilter));
    }

    if (searchTerm) {
      const searchLower = searchTerm.toLowerCase();
      filtered = filtered.filter(p =>
        p.service_name?.toLowerCase().includes(searchLower) ||
        p.screen_type_name?.toLowerCase().includes(searchLower) ||
        p.role?.toLowerCase().includes(searchLower) ||
        p.allowed_action_names?.some(action => action.toLowerCase().includes(searchLower))
      );
    }

    if (sortConfig.field) {
      filtered.sort((a, b) => {
        let aVal = a[sortConfig.field] || '';
        let bVal = b[sortConfig.field] || '';

        if (typeof aVal === 'string') {
          aVal = aVal.toLowerCase();
          bVal = bVal.toLowerCase();
        }

        if (sortConfig.direction === 'asc') {
          return aVal > bVal ? 1 : -1;
        } else {
          return aVal < bVal ? 1 : -1;
        }
      });
    }

    setFilteredPermissions(filtered);
    setPage(1);
  }, [permissions, roleFilter, serviceFilter, searchTerm, sortConfig]);

  // Pagination
  const paginatedPermissions = useMemo(() => {
    const startIndex = (page - 1) * rowsPerPage;
    return filteredPermissions.slice(startIndex, startIndex + rowsPerPage);
  }, [filteredPermissions, page, rowsPerPage]);

  // Statistics
  const stats = useMemo(() => {
    return {
      total: permissions.length,
      super_admin: permissions.filter(p => p.role === 'super_admin').length,
      admin: permissions.filter(p => p.role === 'admin').length,
      user: permissions.filter(p => p.role === 'user').length,
      services: [...new Set(permissions.map(p => p.service_id))].length,
    };
  }, [permissions]);

  const handleSort = (field) => {
    setSortConfig(prev => ({
      field,
      direction: prev.field === field && prev.direction === 'asc' ? 'desc' : 'asc'
    }));
  };

  const handleCreate = () => {
    setEditingPermission({
      role: 'user',
      service_id: null,
      screen_type_id: null,
      allowed_actions: [],
    });
    setEditDialogOpen(true);
  };

  const handleEdit = (permission) => {
    setEditingPermission({
      id: permission.id,
      role: permission.role,
      service_id: permission.service_id,
      screen_type_id: permission.screen_type_id,
      allowed_actions: permission.allowed_actions || [],
    });
    setEditDialogOpen(true);
  };

  const handleDuplicate = (permission) => {
    setEditingPermission({
      role: permission.role,
      service_id: permission.service_id,
      screen_type_id: permission.screen_type_id,
      allowed_actions: permission.allowed_actions || [],
    });
    setEditDialogOpen(true);
  };

  const handleDelete = (permission) => {
    setSelectedPermission(permission);
    setDeleteDialogOpen(true);
  };

  const handleSave = async () => {
    if (!editingPermission.role || !editingPermission.service_id || !editingPermission.screen_type_id || !editingPermission.allowed_actions?.length) {
      setUpdateStatus({
        message: 'Please fill in all required fields.',
        type: 'error',
        show: true
      });
      return;
    }

    try {
      setSaving(true);
      const url = editingPermission.id
        ? `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/permissions/${editingPermission.id}`
        : `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/permissions`;

      const method = editingPermission.id ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
        body: JSON.stringify({
          role: editingPermission.role,
          service_id: editingPermission.service_id,
          screen_type_id: editingPermission.screen_type_id,
          allowed_actions: editingPermission.allowed_actions,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to save permission');
      }

      await fetchPermissions();
      setEditDialogOpen(false);
      setEditingPermission(null);
      setUpdateStatus({
        message: 'Permission saved successfully!',
        type: 'success',
        show: true
      });
    } catch (error) {
      setUpdateStatus({
        message: error.message || 'Failed to save permission. Please try again.',
        type: 'error',
        show: true
      });
    } finally {
      setSaving(false);
    }
  };

  const handleConfirmDelete = async () => {
    try {
      setSaving(true);
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/permissions/${selectedPermission.id}`,
        {
          method: 'DELETE',
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to delete permission');
      }

      await fetchPermissions();
      setDeleteDialogOpen(false);
      setSelectedPermission(null);
      setUpdateStatus({
        message: 'Permission deleted successfully!',
        type: 'success',
        show: true
      });
    } catch (error) {
      setUpdateStatus({
        message: error.message || 'Failed to delete permission. Please try again.',
        type: 'error',
        show: true
      });
    } finally {
      setSaving(false);
    }
  };

  // Pre-populate bulk update permissions based on selected role
  const prePopulateBulkPermissions = (role) => {
    if (!permissions || !Array.isArray(permissions)) {
      return {};
    }
    
    const rolePermissions = permissions.filter(p => p.role === role);
    const prePopulated = {};
    
    rolePermissions.forEach(perm => {
      if (perm.service_id && perm.screen_type_id && perm.allowed_actions && perm.allowed_actions.length > 0) {
        const key = `${perm.service_id}-${perm.screen_type_id}`;
        prePopulated[key] = perm.allowed_actions;
      }
    });
    
    return prePopulated;
  };

  const handleBulkUpdate = () => {
    setBulkUpdateRole('user');
    const prePopulated = prePopulateBulkPermissions('user');
    setBulkUpdatePermissions(prePopulated);
    setBulkUpdateDialogOpen(true);
  };

  const handleBulkPermissionToggle = (serviceId, screenTypeId, actionId) => {
    const key = `${serviceId}-${screenTypeId}`;
    setBulkUpdatePermissions(prev => {
      const current = prev[key] || [];
      const newActions = current.includes(actionId)
        ? current.filter(id => id !== actionId)
        : [...current, actionId];
      
      if (newActions.length === 0) {
        const { [key]: removed, ...rest } = prev;
        return rest;
      }
      
      return { ...prev, [key]: newActions };
    });
  };

  const handleBulkSelectAllForScreen = (serviceId, screenTypeId) => {
    const key = `${serviceId}-${screenTypeId}`;
    const allActionIds = actions.map(a => a.id);
    setBulkUpdatePermissions(prev => ({
      ...prev,
      [key]: allActionIds,
    }));
  };

  const handleBulkClearScreen = (serviceId, screenTypeId) => {
    const key = `${serviceId}-${screenTypeId}`;
    setBulkUpdatePermissions(prev => {
      const { [key]: removed, ...rest } = prev;
      return rest;
    });
  };

  const handleBulkUpdateSave = async () => {
    if (!bulkUpdateRole) {
      setUpdateStatus({ message: 'Please select a role.', type: 'error', show: true });
      return;
    }

    const permissionsArray = Object.entries(bulkUpdatePermissions)
      .filter(([key, actionIds]) => actionIds && actionIds.length > 0)
      .map(([key, actionIds]) => {
        const [serviceId, screenTypeId] = key.split('-').map(Number);
        return { 
          role: bulkUpdateRole,
          service_id: serviceId, 
          screen_type_id: screenTypeId, 
          allowed_actions: actionIds 
        };
      });

    if (permissionsArray.length === 0) {
      setUpdateStatus({ message: 'Please select at least one permission.', type: 'error', show: true });
      return;
    }

    try {
      setSaving(true);
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/permissions/role/${bulkUpdateRole}/bulk`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${user.token}`,
          },
          body: JSON.stringify(permissionsArray),
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to update permissions');
      }

      await fetchPermissions();
      setBulkUpdateDialogOpen(false);
      setBulkUpdatePermissions({});
      setUpdateStatus({
        message: `Updated ${permissionsArray.length} permission(s) for ${ROLE_CONFIG[bulkUpdateRole]?.label}!`,
        type: 'success',
        show: true
      });
    } catch (error) {
      setUpdateStatus({
        message: error.message || 'Failed to update permissions.',
        type: 'error',
        show: true
      });
    } finally {
      setSaving(false);
    }
  };

  const handleServiceChange = (event, newValue) => {
    setEditingPermission({
      ...editingPermission,
      service_id: newValue ? newValue.id : null,
      screen_type_id: null,
    });
  };

  const handleScreenTypeChange = (event, newValue) => {
    setEditingPermission({
      ...editingPermission,
      screen_type_id: newValue ? newValue.id : null,
    });
  };

  const handleActionToggle = (actionId) => {
    setEditingPermission(prev => ({
      ...prev,
      allowed_actions: prev.allowed_actions.includes(actionId)
        ? prev.allowed_actions.filter(id => id !== actionId)
        : [...prev.allowed_actions, actionId],
    }));
  };

  const handleCloseSnackbar = () => {
    setUpdateStatus(prev => ({ ...prev, show: false }));
  };

  const selectedService = useMemo(() => {
    if (!editingPermission?.service_id) return null;
    return services.find(s => s.id === editingPermission.service_id);
  }, [editingPermission?.service_id, services]);

  const selectedScreenType = useMemo(() => {
    if (!editingPermission?.screen_type_id) return null;
    return availableScreenTypes.find(st => st.id === editingPermission.screen_type_id);
  }, [editingPermission?.screen_type_id, availableScreenTypes]);

  if (!isSuperAdmin) {
    return (
      <Box sx={{ p: 4, display: 'flex', justifyContent: 'center' }}>
        <Alert severity="error" sx={{ maxWidth: 500 }}>
          <Typography variant="h6">Access Denied</Typography>
          <Typography variant="body2">Only super administrators can access permission management.</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ minHeight: '100vh', backgroundColor: '#f8fafc' }}>
      {/* Header */}
      <Box 
        sx={{ 
          background: 'linear-gradient(135deg, rgb(120, 40, 98) 0%, rgb(101, 34, 81) 50%, rgb(92, 7, 66) 100%)',
          color: 'white',
          py: 4,
          px: { xs: 2, sm: 4 },
        }}
      >
        <Box sx={{ maxWidth: 1400, mx: 'auto' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
            <Avatar sx={{ bgcolor: 'rgba(255,255,255,0.2)', width: 48, height: 48 }}>
              <SecurityIcon />
            </Avatar>
            <Box>
              <Typography variant="h4" sx={{ fontWeight: 700, letterSpacing: '-0.5px' }}>
                Permission Management
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.8 }}>
                Configure role-based access control for your platform
              </Typography>
            </Box>
          </Box>

          {/* Stats Row */}
          <Grid container spacing={2}>
            {[
              { label: 'Total Permissions', value: stats.total, icon: <LayersIcon /> },
              { label: 'Super Admin', value: stats.super_admin, icon: <SupervisorAccountIcon />, color: '#f87171' },
              { label: 'Admin', value: stats.admin, icon: <AdminPanelSettingsIcon />, color: '#fbbf24' },
              { label: 'User', value: stats.user, icon: <PersonIcon />, color: '#60a5fa' },
              { label: 'Services', value: stats.services, icon: <AppsIcon /> },
            ].map((stat, idx) => (
              <Grid item xs={6} sm={4} md={2.4} key={idx}>
                <Box 
                  sx={{ 
                    p: 2, 
                    borderRadius: 2, 
                    backgroundColor: 'rgba(255,255,255,0.1)',
                    backdropFilter: 'blur(10px)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1.5,
                  }}
                >
                  <Box sx={{ opacity: 0.8, color: stat.color || 'inherit' }}>{stat.icon}</Box>
                  <Box>
                    <Typography variant="h5" sx={{ fontWeight: 700, lineHeight: 1 }}>
                      {stat.value}
                    </Typography>
                    <Typography variant="caption" sx={{ opacity: 0.7 }}>
                      {stat.label}
                    </Typography>
                  </Box>
                </Box>
              </Grid>
            ))}
          </Grid>
        </Box>
      </Box>

      {/* Main Content */}
      <Box sx={{ maxWidth: 1400, mx: 'auto', p: { xs: 2, sm: 4 }, mt: -2 }}>
        {/* Toolbar */}
        <Paper 
          elevation={0} 
          sx={{ 
            p: 2, 
            mb: 3, 
            borderRadius: 3,
            border: '1px solid',
            borderColor: 'divider',
            display: 'flex',
            flexWrap: 'wrap',
            gap: 2,
            alignItems: 'center',
          }}
        >
          <TextField
            placeholder="Search permissions..."
            size="small"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: <SearchIcon sx={{ color: 'text.secondary', mr: 1 }} />,
            }}
            sx={{ 
              minWidth: 280,
              flex: 1,
              '& .MuiOutlinedInput-root': {
                borderRadius: 2,
                backgroundColor: '#f8fafc',
              }
            }}
          />

          <Box sx={{ display: 'flex', gap: 1 }}>
            {['all', 'super_admin', 'admin', 'user'].map((role) => (
              <Chip
                key={role}
                label={role === 'all' ? 'All Roles' : ROLE_CONFIG[role]?.label}
                onClick={() => setRoleFilter(role)}
                variant={roleFilter === role ? 'filled' : 'outlined'}
                sx={{
                  fontWeight: 500,
                  ...(roleFilter === role && role !== 'all' && {
                    backgroundColor: ROLE_CONFIG[role]?.color,
                    color: 'white',
                  }),
                }}
              />
            ))}
          </Box>

          <Box sx={{ display: 'flex', gap: 1, ml: 'auto' }}>
            <Tooltip title="Refresh">
              <IconButton onClick={fetchPermissions} disabled={loading}>
                <RefreshIcon />
              </IconButton>
            </Tooltip>
            <Button
              variant="outlined"
              startIcon={<TuneIcon />}
              onClick={handleBulkUpdate}
              sx={{ borderRadius: 2 }}
            >
              Bulk Update
            </Button>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleCreate}
              disabled={loadingMetadata}
              sx={{ 
                borderRadius: 2,
                background: 'linear-gradient(135deg, #8B4578 0%, #9C4D85 100%)',
                boxShadow: '0 4px 14px 0 rgba(139, 69, 120, 0.3)',
                color: 'white',
                '&:hover': {
                  background: 'linear-gradient(135deg, #7A3D6B 0%, #8B4578 100%)',
                }
              }}
            >
              Add Permission
            </Button>
          </Box>
        </Paper>

        {/* Service Filter Tabs */}
        <Box sx={{ mb: 3 }}>
          <Tabs
            value={serviceFilter}
            onChange={(e, val) => setServiceFilter(val)}
            variant="scrollable"
            scrollButtons="auto"
              sx={{
                '& .MuiTab-root': {
                  textTransform: 'none',
                  fontWeight: 500,
                  minHeight: 40,
                  borderRadius: 2,
                  mr: 1,
                },
                '& .Mui-selected': {
                  backgroundColor: alpha('#8B4578', 0.12),
                  color: '#6B1E5A',
                },
              }}
          >
            <Tab label="All Services" value="all" />
            {services.map(service => (
              <Tab 
                key={service.id} 
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    {service.display_name}
                    <Chip 
                      label={permissions.filter(p => p.service_id === service.id).length} 
                      size="small" 
                      sx={{ height: 20, fontSize: '0.7rem' }}
                    />
                  </Box>
                } 
                value={service.id.toString()} 
              />
            ))}
          </Tabs>
        </Box>

        {/* Table */}
        <Paper 
          elevation={0} 
          sx={{ 
            borderRadius: 3,
            border: '1px solid',
            borderColor: 'divider',
            overflow: 'hidden',
          }}
        >
          <TableContainer sx={{ maxHeight: 'calc(100vh - 450px)' }}>
            <Table stickyHeader>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ backgroundColor: '#f8fafc', fontWeight: 600 }}>
                    <TableSortLabel
                      active={sortConfig.field === 'role'}
                      direction={sortConfig.field === 'role' ? sortConfig.direction : 'asc'}
                      onClick={() => handleSort('role')}
                    >
                      Role
                    </TableSortLabel>
                  </TableCell>
                  <TableCell sx={{ backgroundColor: '#f8fafc', fontWeight: 600 }}>
                    <TableSortLabel
                      active={sortConfig.field === 'service_name'}
                      direction={sortConfig.field === 'service_name' ? sortConfig.direction : 'asc'}
                      onClick={() => handleSort('service_name')}
                    >
                      Service
                    </TableSortLabel>
                  </TableCell>
                  <TableCell sx={{ backgroundColor: '#f8fafc', fontWeight: 600 }}>
                    <TableSortLabel
                      active={sortConfig.field === 'screen_type_name'}
                      direction={sortConfig.field === 'screen_type_name' ? sortConfig.direction : 'asc'}
                      onClick={() => handleSort('screen_type_name')}
                    >
                      Screen Type
                    </TableSortLabel>
                  </TableCell>
                  <TableCell sx={{ backgroundColor: '#f8fafc', fontWeight: 600, minWidth: 250 }}>
                    Allowed Actions
                  </TableCell>
                  <TableCell sx={{ backgroundColor: '#f8fafc', fontWeight: 600, width: 130 }}>
                    Actions
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {loading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell><Skeleton variant="rounded" width={100} height={28} /></TableCell>
                      <TableCell><Skeleton variant="text" /></TableCell>
                      <TableCell><Skeleton variant="text" /></TableCell>
                      <TableCell><Skeleton variant="text" width="60%" /></TableCell>
                      <TableCell><Skeleton variant="rounded" width={80} height={32} /></TableCell>
                    </TableRow>
                  ))
                ) : paginatedPermissions.length > 0 ? (
                  paginatedPermissions.map((perm) => (
                    <TableRow key={perm.id} hover>
                      <TableCell>
                        <Box 
                          sx={{ 
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 0.5,
                            px: 1.5,
                            py: 0.5,
                            borderRadius: 2,
                            backgroundColor: ROLE_CONFIG[perm.role]?.bgColor,
                            color: ROLE_CONFIG[perm.role]?.color,
                            fontWeight: 600,
                            fontSize: '0.8rem',
                          }}
                        >
                          {ROLE_CONFIG[perm.role]?.icon}
                          {ROLE_CONFIG[perm.role]?.label}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" fontWeight={500}>
                          {perm.service_name}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {perm.screen_type_name}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                          {perm.allowed_action_names?.map((action, idx) => (
                            <Chip 
                              key={idx} 
                              label={action}
                              size="small"
                              icon={ACTION_ICONS[action] || <SettingsIcon fontSize="small" />}
                              sx={{ 
                                fontSize: '0.75rem',
                                height: 26,
                                backgroundColor: alpha('#8B4578', 0.1),
                                color: '#6B1E5A',
                                '& .MuiChip-icon': { color: '#8B4578' }
                              }}
                            />
                          ))}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 0.5 }}>
                          <Tooltip title="Edit">
                            <IconButton size="small" onClick={() => handleEdit(perm)}>
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Duplicate">
                            <IconButton size="small" onClick={() => handleDuplicate(perm)}>
                              <ContentCopyIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Delete">
                            <IconButton size="small" onClick={() => handleDelete(perm)} sx={{ color: '#ef4444' }}>
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Box>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={5} sx={{ py: 8, textAlign: 'center' }}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
                        <Avatar sx={{ width: 64, height: 64, bgcolor: '#f1f5f9' }}>
                          <SecurityIcon sx={{ color: '#94a3b8' }} />
                        </Avatar>
                        <Typography variant="h6" color="text.secondary">No permissions found</Typography>
                        <Typography variant="body2" color="text.secondary">
                          Try adjusting your filters or add a new permission
                        </Typography>
                        <Button variant="outlined" startIcon={<AddIcon />} onClick={handleCreate}>
                          Add Permission
                        </Button>
                      </Box>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>

          {filteredPermissions.length > rowsPerPage && (
            <Box sx={{ p: 2, display: 'flex', justifyContent: 'center', borderTop: '1px solid', borderColor: 'divider' }}>
              <Pagination
                count={Math.ceil(filteredPermissions.length / rowsPerPage)}
                page={page}
                onChange={(e, val) => setPage(val)}
                shape="rounded"
              />
            </Box>
          )}
        </Paper>
      </Box>

      {/* Create/Edit Dialog */}
      <Dialog 
        open={editDialogOpen} 
        onClose={() => !saving && setEditDialogOpen(false)} 
        maxWidth="sm" 
        fullWidth
        TransitionComponent={Slide}
        TransitionProps={{ direction: 'up' }}
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
              <Avatar sx={{ bgcolor: alpha('#8B4578', 0.12), color: '#6B1E5A' }}>
                {editingPermission?.id ? <EditIcon /> : <AddIcon />}
              </Avatar>
              <Typography variant="h6" fontWeight={600}>
                {editingPermission?.id ? 'Edit Permission' : 'Create Permission'}
              </Typography>
            </Box>
            <IconButton onClick={() => setEditDialogOpen(false)} disabled={saving}>
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 3 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* Role Selection */}
            <Box>
              <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>Role *</Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                {['user', 'admin', 'super_admin'].map((role) => (
                  <Box
                    key={role}
                    onClick={() => setEditingPermission({ ...editingPermission, role })}
                    sx={{
                      flex: 1,
                      p: 2,
                      borderRadius: 2,
                      border: '2px solid',
                      borderColor: editingPermission?.role === role ? ROLE_CONFIG[role].color : 'divider',
                      backgroundColor: editingPermission?.role === role ? ROLE_CONFIG[role].bgColor : 'transparent',
                      cursor: 'pointer',
                      textAlign: 'center',
                      transition: 'all 0.2s',
                      '&:hover': {
                        borderColor: ROLE_CONFIG[role].color,
                        transform: 'translateY(-2px)',
                      }
                    }}
                  >
                    <Box sx={{ color: ROLE_CONFIG[role].color, mb: 0.5 }}>
                      {ROLE_CONFIG[role].icon}
                    </Box>
                    <Typography variant="body2" fontWeight={600} sx={{ color: ROLE_CONFIG[role].color }}>
                      {ROLE_CONFIG[role].label}
                    </Typography>
                  </Box>
                ))}
              </Box>
            </Box>

            {/* Service Selection */}
            <Autocomplete
              options={services}
              getOptionLabel={(opt) => opt.display_name || opt.name}
              value={selectedService}
              onChange={handleServiceChange}
              loading={loadingMetadata}
              renderInput={(params) => (
                <TextField {...params} label="Service" required placeholder="Search service..." />
              )}
              renderOption={(props, opt) => (
                <Box component="li" {...props}>
                  <Box>
                    <Typography fontWeight={500}>{opt.display_name}</Typography>
                    {opt.description && <Typography variant="caption" color="text.secondary">{opt.description}</Typography>}
                  </Box>
                </Box>
              )}
            />

            {/* Screen Type Selection */}
            <Autocomplete
              options={availableScreenTypes}
              getOptionLabel={(opt) => opt.display_name || opt.name}
              value={selectedScreenType}
              onChange={handleScreenTypeChange}
              disabled={!editingPermission?.service_id}
              renderInput={(params) => (
                <TextField 
                  {...params} 
                  label="Screen Type" 
                  required 
                  placeholder={editingPermission?.service_id ? "Search screen type..." : "Select a service first"}
                  helperText={!editingPermission?.service_id ? "Select a service first" : undefined}
                />
              )}
            />

            {/* Actions Selection */}
            <Box>
              <Typography variant="subtitle2" sx={{ mb: 1.5, fontWeight: 600 }}>
                Allowed Actions *
              </Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {actions.map((action) => {
                  const isSelected = editingPermission?.allowed_actions?.includes(action.id);
                  return (
                    <Chip
                      key={action.id}
                      label={action.display_name}
                      icon={ACTION_ICONS[action.name] || <SettingsIcon fontSize="small" />}
                      onClick={() => handleActionToggle(action.id)}
                      variant={isSelected ? 'filled' : 'outlined'}
                      sx={{
                        cursor: 'pointer',
                        fontWeight: 500,
                        transition: 'all 0.2s',
                        ...(isSelected && {
                          backgroundColor: '#8B4578',
                          color: 'white',
                          '& .MuiChip-icon': { color: 'white' },
                        }),
                        '&:hover': {
                          transform: 'scale(1.05)',
                        }
                      }}
                    />
                  );
                })}
              </Box>
              {editingPermission?.allowed_actions?.length > 0 && (
                <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                  {editingPermission.allowed_actions.length} action(s) selected
                </Typography>
              )}
            </Box>
          </Box>
        </DialogContent>
        <Divider />
        <DialogActions sx={{ p: 2.5 }}>
          <Button onClick={() => setEditDialogOpen(false)} disabled={saving}>Cancel</Button>
          <Button 
            onClick={handleSave} 
            variant="contained" 
            startIcon={saving ? <CircularProgress size={16} /> : <SaveIcon />}
            disabled={saving || !editingPermission?.service_id || !editingPermission?.screen_type_id || !editingPermission?.allowed_actions?.length}
            sx={{ 
              borderRadius: 2,
              background: 'linear-gradient(135deg, #8B4578 0%, #9C4D85 100%)',
              color: 'white',
              '&:hover': {
                background: 'linear-gradient(135deg, #7A3D6B 0%, #8B4578 100%)',
              }
            }}
          >
            {saving ? 'Saving...' : 'Save Permission'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => !saving && setDeleteDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Avatar sx={{ bgcolor: '#fef2f2', color: '#dc2626' }}>
              <WarningIcon />
            </Avatar>
            <Typography variant="h6" fontWeight={600}>Delete Permission</Typography>
          </Box>
        </DialogTitle>
        <DialogContent>
          <Typography color="text.secondary">
            Are you sure you want to delete this permission? This action cannot be undone.
          </Typography>
          {selectedPermission && (
            <Box sx={{ mt: 2, p: 2, bgcolor: '#f8fafc', borderRadius: 2 }}>
              <Typography variant="body2"><strong>Role:</strong> {ROLE_CONFIG[selectedPermission.role]?.label}</Typography>
              <Typography variant="body2"><strong>Service:</strong> {selectedPermission.service_name}</Typography>
              <Typography variant="body2"><strong>Screen:</strong> {selectedPermission.screen_type_name}</Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2.5 }}>
          <Button onClick={() => setDeleteDialogOpen(false)} disabled={saving}>Cancel</Button>
          <Button 
            onClick={handleConfirmDelete} 
            color="error" 
            variant="contained"
            disabled={saving}
            startIcon={saving ? <CircularProgress size={16} /> : <DeleteIcon />}
          >
            {saving ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Bulk Update Dialog */}
      <Dialog 
        open={bulkUpdateDialogOpen} 
        onClose={() => !saving && setBulkUpdateDialogOpen(false)} 
        maxWidth="lg" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3, maxHeight: '90vh' } }}
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
              <Avatar sx={{ bgcolor: alpha('#8B4578', 0.12), color: '#6B1E5A' }}>
                <TuneIcon />
              </Avatar>
              <Box>
                <Typography variant="h6" fontWeight={600}>Bulk Update Permissions</Typography>
                <Typography variant="caption" color="text.secondary">
                  Configure multiple permissions for a role at once
                </Typography>
              </Box>
            </Box>
            <IconButton onClick={() => setBulkUpdateDialogOpen(false)} disabled={saving}>
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ p: 0 }}>
          {/* Role Selection */}
          <Box sx={{ p: 3, bgcolor: '#f8fafc', borderBottom: '1px solid', borderColor: 'divider' }}>
            <Typography variant="subtitle2" sx={{ mb: 1.5, fontWeight: 600 }}>Select Role</Typography>
            <Box sx={{ display: 'flex', gap: 2 }}>
              {['user', 'admin', 'super_admin'].map((role) => (
                <Box
                  key={role}
                  onClick={() => { 
                    setBulkUpdateRole(role); 
                    const prePopulated = prePopulateBulkPermissions(role);
                    setBulkUpdatePermissions(prePopulated);
                  }}
                  sx={{
                    flex: 1,
                    p: 2,
                    borderRadius: 2,
                    border: '2px solid',
                    borderColor: bulkUpdateRole === role ? ROLE_CONFIG[role].color : 'divider',
                    backgroundColor: bulkUpdateRole === role ? ROLE_CONFIG[role].bgColor : 'white',
                    cursor: 'pointer',
                    textAlign: 'center',
                    transition: 'all 0.2s',
                    '&:hover': { borderColor: ROLE_CONFIG[role].color }
                  }}
                >
                  <Box sx={{ color: ROLE_CONFIG[role].color, mb: 0.5 }}>{ROLE_CONFIG[role].icon}</Box>
                  <Typography variant="body2" fontWeight={600} sx={{ color: ROLE_CONFIG[role].color }}>
                    {ROLE_CONFIG[role].label}
                  </Typography>
                </Box>
              ))}
            </Box>
            <Alert severity="warning" sx={{ mt: 2 }}>
              This will <strong>replace all existing permissions</strong> for {ROLE_CONFIG[bulkUpdateRole]?.label}
            </Alert>
          </Box>

          {/* Permission Matrix */}
          <Box sx={{ p: 3, maxHeight: 'calc(90vh - 350px)', overflowY: 'auto' }}>
            {services.map((service) => {
              const serviceScreenTypes = allScreenTypes.filter(st => st.service_id === service.id);
              if (serviceScreenTypes.length === 0) return null;

              return (
                <Box key={service.id} sx={{ mb: 4 }}>
                  <Typography variant="h6" sx={{ mb: 2, fontWeight: 600, color: '#6B1E5A' }}>
                    {service.display_name}
                  </Typography>
                  
                  <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
                    <Table size="small">
                      <TableHead>
                        <TableRow sx={{ bgcolor: '#f8fafc' }}>
                          <TableCell sx={{ fontWeight: 600, width: 200 }}>Screen Type</TableCell>
                          {actions.map(action => (
                            <TableCell key={action.id} align="center" sx={{ fontWeight: 600, fontSize: '0.75rem' }}>
                              {action.display_name}
                            </TableCell>
                          ))}
                          <TableCell align="center" sx={{ fontWeight: 600, width: 120 }}>Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {serviceScreenTypes.map((st) => {
                          const key = `${service.id}-${st.id}`;
                          const selected = bulkUpdatePermissions[key] || [];
                          return (
                            <TableRow key={st.id} hover>
                              <TableCell>
                                <Typography variant="body2" fontWeight={500}>{st.display_name}</Typography>
                              </TableCell>
                              {actions.map(action => (
                                <TableCell key={action.id} align="center">
                                  <Checkbox
                                    size="small"
                                    checked={selected.includes(action.id)}
                                    onChange={() => handleBulkPermissionToggle(service.id, st.id, action.id)}
                                    sx={{ 
                                      p: 0.5,
                                      '&.Mui-checked': { color: '#8B4578' }
                                    }}
                                  />
                                </TableCell>
                              ))}
                              <TableCell align="center">
                                <Button size="small" onClick={() => handleBulkSelectAllForScreen(service.id, st.id)}>All</Button>
                                <Button size="small" color="inherit" onClick={() => handleBulkClearScreen(service.id, st.id)}>Clear</Button>
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Box>
              );
            })}
          </Box>
        </DialogContent>
        <Divider />
        <DialogActions sx={{ p: 2.5, bgcolor: '#f8fafc' }}>
          <Box sx={{ flex: 1 }}>
            <Typography variant="body2" color="text.secondary">
              {Object.keys(bulkUpdatePermissions).length} screen type(s) selected, {Object.values(bulkUpdatePermissions).reduce((sum, a) => sum + a.length, 0)} action(s) total
            </Typography>
          </Box>
          <Button onClick={() => setBulkUpdateDialogOpen(false)} disabled={saving}>Cancel</Button>
          <Button 
            onClick={handleBulkUpdateSave} 
            variant="contained"
            disabled={saving || Object.keys(bulkUpdatePermissions).length === 0}
            startIcon={saving ? <CircularProgress size={16} /> : <SaveIcon />}
            sx={{ 
              borderRadius: 2,
              background: 'linear-gradient(135deg, #8B4578 0%, #9C4D85 100%)',
              color: 'white',
              '&:hover': {
                background: 'linear-gradient(135deg, #7A3D6B 0%, #8B4578 100%)',
              }
            }}
          >
            {saving ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar */}
      <Snackbar
        open={updateStatus.show}
        autoHideDuration={4000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert 
          onClose={handleCloseSnackbar} 
          severity={updateStatus.type}
          variant="filled"
          sx={{ borderRadius: 2 }}
        >
          {updateStatus.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default PermissionManagement;
