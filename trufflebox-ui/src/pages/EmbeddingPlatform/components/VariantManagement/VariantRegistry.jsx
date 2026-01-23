import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  TextField,
  CircularProgress,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
  IconButton,
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
  FormControl,
  FormHelperText,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Switch,
  FormControlLabel,
  Snackbar,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import FilterListIcon from '@mui/icons-material/FilterList';
import ExperimentIcon from '@mui/icons-material/Science';
import LockIcon from '@mui/icons-material/Lock';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const VariantRegistry = () => {
  const [variantRequests, setVariantRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['APPROVED', 'PENDING', 'REJECTED']);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedVariant, setSelectedVariant] = useState(null);
  const [entities, setEntities] = useState([]);
  const [models, setModels] = useState([]);
  const [filters, setFilters] = useState([]);
  const [variantsList, setVariantsList] = useState([]);
  const { user } = useAuth();
  const [notification, setNotification] = useState({ open: false, message: "", severity: "success" });
  
  // Simplified variant data structure
  const [variantData, setVariantData] = useState({
    entity: '',
    model: '',
    variant: '',
    reason: '',
    vector_db_type: 'QDRANT',
    type: 'EXPERIMENT',
    caching_configuration: {
      in_memory_caching_enabled: true,
      in_memory_cache_ttl_seconds: 300,
      distributed_caching_enabled: false,
      distributed_cache_ttl_seconds: 600,
      embedding_retrieval_in_memory_config: { enabled: true, ttl: 60 },
      embedding_retrieval_distributed_config: { enabled: false, ttl: 300 },
      dot_product_in_memory_config: { enabled: true, ttl: 30 },
      dot_product_distributed_config: { enabled: false, ttl: 120 },
    },
    filter_configuration: {
      criteria: [],
      enabled: false,
      selected_filters: []
    },
    vector_db_config: {},
    rate_limiter: { rate_limit: 0, burst_limit: 0 },
    rt_partition: 0
  });

  const statusOptions = [
    { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
  ];

  useEffect(() => {
    fetchData();
    fetchFilters();
    fetchVariantsList();
  }, []);

  useEffect(() => {
    if (variantData.entity) {
      fetchModelsForEntity(variantData.entity);
      fetchFiltersForEntity(variantData.entity);
    } else {
      setModels([]);
      setVariantData(prev => ({ ...prev, model: '' }));
    }
  }, [variantData.entity]);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const [variantRequestsResponse, entitiesResponse] = await Promise.all([
        embeddingPlatformAPI.getVariantRequests(),
        embeddingPlatformAPI.getEntities()
      ]);

      if (variantRequestsResponse.variant_requests) {
        setVariantRequests(variantRequestsResponse.variant_requests);
      } else {
        setVariantRequests([]);
      }

      if (entitiesResponse.entities) {
        const availableEntities = entitiesResponse.entities?.map(entity => ({
          name: entity.name,
          store_id: entity.store_id,
          label: entity.name
        })) || [];
        setEntities(availableEntities);
      }
    } catch (error) {
      console.error('Error fetching variant data:', error);
      setError('Failed to load variant data. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const fetchVariantsList = async () => {
    try {
      const response = await embeddingPlatformAPI.getVariantsList();
      if (response.variants) {
        setVariantsList(response.variants);
      } else {
        setVariantsList([]);
      }
    } catch (error) {
      console.error('Error fetching variants list:', error);
      setVariantsList([]);
    }
  };

  const fetchModelsForEntity = async (entityName) => {
    try {
      const response = await embeddingPlatformAPI.getModels({ entity: entityName });
      if (response.models) {
        const approvedModels = response.models.filter(model => 
          (model.status || '') === 'active'
        );
        setModels(approvedModels);
      }
    } catch (error) {
      console.error('Error fetching models for entity:', error);
      setModels([]);
    }
  };

  const fetchFilters = async () => {
    try {
      const response = await embeddingPlatformAPI.getFilters();
      
      // Transform filter_groups response to flat array format
      if (response.filter_groups && Array.isArray(response.filter_groups)) {
        const transformedFilters = response.filter_groups.flatMap((group, groupIndex) => 
          group.filters.map((filter, filterIndex) => ({
            id: `${group.entity}_${filterIndex}`,
            filter_id: `${group.entity}_${filterIndex}`,
            entity: group.entity,
            column_name: filter.column_name,
            filter_value: filter.filter_value,
            default_value: filter.default_value,
            filter: {
              column_name: filter.column_name,
              filter_value: filter.filter_value,
              default_value: filter.default_value
            }
          }))
        );
        setFilters(transformedFilters);
      } else if (response.filters) {
        // Fallback for old response format
        setFilters(response.filters);
      } else {
        setFilters([]);
      }
    } catch (error) {
      console.error('Error fetching filters:', error);
      setFilters([]);
    }
  };

  const fetchFiltersForEntity = async (entityName) => {
    try {
      const response = await embeddingPlatformAPI.getFilters({ entity: entityName });
      
      // Transform filter_groups response to flat array format
      if (response.filter_groups && Array.isArray(response.filter_groups)) {
        const transformedFilters = response.filter_groups.flatMap((group, groupIndex) => 
          group.filters.map((filter, filterIndex) => ({
            id: `${group.entity}_${filterIndex}`,
            filter_id: `${group.entity}_${filterIndex}`,
            entity: group.entity,
            column_name: filter.column_name,
            filter_value: filter.filter_value,
            default_value: filter.default_value,
            filter: {
              column_name: filter.column_name,
              filter_value: filter.filter_value,
              default_value: filter.default_value
            }
          }))
        );
        // Store entity-specific filters separately or filter existing ones
        setFilters(prevFilters => [
          ...prevFilters.filter(f => f.entity !== entityName),
          ...transformedFilters
        ]);
      } else if (response.filters) {
        // Fallback for old response format
        setFilters(prevFilters => [
          ...prevFilters.filter(f => f.entity !== entityName),
          ...response.filters
        ]);
      }
    } catch (error) {
      console.error('Error fetching filters for entity:', error);
    }
  };

  const filteredRequests = useMemo(() => {
    let filtered = variantRequests.filter(request => 
      selectedStatuses.includes((request.status || 'PENDING').toUpperCase())
    );

    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(request => {
        return (
          String(request.request_id || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.entity || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.model || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.variant || '').toLowerCase().includes(searchLower) ||
          String(request.created_by || '').toLowerCase().includes(searchLower)
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at || 0) - new Date(a.created_at || 0);
    });
  }, [variantRequests, searchQuery, selectedStatuses]);

  const getStatusChip = (status) => {
    const statusUpper = (status || 'PENDING').toUpperCase();
    let bgcolor = '#FFF8E1';
    let textColor = '#F57C00';

    switch (statusUpper) {
      case 'PENDING':
        bgcolor = '#FFF8E1';
        textColor = '#F57C00';
        break;
      case 'APPROVED':
        bgcolor = '#E7F6E7';
        textColor = '#2E7D32';
        break;
      case 'REJECTED':
        bgcolor = '#FFEBEE';
        textColor = '#D32F2F';
        break;
    }

    return (
      <Chip
        label={statusUpper}
        size="small"
        sx={{ backgroundColor: bgcolor, color: textColor, fontWeight: 'bold', minWidth: '80px' }}
      />
    );
  };

  // Status Column Header with filtering
  const StatusColumnHeader = () => {
    const [anchorEl, setAnchorEl] = useState(null);
    
    const handleClick = (event) => {
      setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
      setAnchorEl(null);
    };

    const handleStatusToggle = (status) => {
      setSelectedStatuses(prev => 
        prev.includes(status) 
          ? prev.filter(s => s !== status)
          : [...prev, status]
      );
    };

    const handleSelectAll = () => {
      setSelectedStatuses(statusOptions.map(option => option.value));
    };

    const handleClearAll = () => {
      setSelectedStatuses([]);
    };

    const open = Boolean(anchorEl);

    return (
      <>
        <Box 
          sx={{ 
            display: 'flex', 
            flexDirection: 'column',
            alignItems: 'flex-start',
            width: '100%'
          }}
        >
          <Box 
            sx={{ 
              display: 'flex', 
              alignItems: 'center', 
              cursor: 'pointer',
              '&:hover': { backgroundColor: 'rgba(0, 0, 0, 0.04)' },
              borderRadius: 1,
              p: 0.5
            }}
            onClick={handleClick}
          >
            <Typography sx={{ fontWeight: 'bold', color: '#031022' }}>
              Status
            </Typography>
            <FilterListIcon 
              sx={{ 
                ml: 0.5, 
                fontSize: 16,
                color: selectedStatuses.length < statusOptions.length ? '#1976d2' : '#666'
              }} 
            />
            {selectedStatuses.length > 0 && selectedStatuses.length < statusOptions.length && (
              <Box
                sx={{
                  position: 'absolute',
                  top: 2,
                  right: 2,
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  backgroundColor: '#1976d2',
                }}
              />
            )}
          </Box>

          {selectedStatuses.length > 0 && (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5, maxWidth: '100%' }}>
              {selectedStatuses.slice(0, 2).map((status) => {
                const option = statusOptions.find(opt => opt.value === status);
                return option ? (
                  <Chip
                    key={status}
                    label={option.label}
                    size="small"
                    sx={{
                      backgroundColor: option.color,
                      color: option.textColor,
                      fontWeight: 'bold',
                      fontSize: '0.65rem',
                      height: 18,
                      '& .MuiChip-label': { px: 0.5 }
                    }}
                  />
                ) : null;
              })}
              {selectedStatuses.length > 2 && (
                <Chip
                  label={`+${selectedStatuses.length - 2}`}
                  size="small"
                  sx={{
                    backgroundColor: '#f5f5f5',
                    color: '#666',
                    fontWeight: 'bold',
                    fontSize: '0.65rem',
                    height: 18,
                    '& .MuiChip-label': { px: 0.5 }
                  }}
                />
              )}
            </Box>
          )}
        </Box>
        <Popover
          open={open}
          anchorEl={anchorEl}
          onClose={handleClose}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
          transformOrigin={{ vertical: 'top', horizontal: 'left' }}
          PaperProps={{ sx: { width: 200, maxHeight: 300, overflow: 'auto' } }}
        >
          <Box sx={{ p: 1 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
              <Button size="small" onClick={handleSelectAll} sx={{ textTransform: 'none', fontSize: '0.75rem', minWidth: 'auto' }}>
                All
              </Button>
              <Button size="small" onClick={handleClearAll} sx={{ textTransform: 'none', fontSize: '0.75rem', minWidth: 'auto' }}>
                Clear
              </Button>
            </Box>
            <Divider sx={{ mb: 1 }} />
            <List dense>
              {statusOptions.map((option) => (
                <ListItem key={option.value} disablePadding>
                  <ListItemButton onClick={() => handleStatusToggle(option.value)} sx={{ py: 0.5 }}>
                    <Checkbox
                      edge="start"
                      checked={selectedStatuses.includes(option.value)}
                      size="small"
                      sx={{ mr: 1 }}
                    />
                    <Chip
                      label={option.label}
                      size="small"
                      sx={{
                        backgroundColor: option.color,
                        color: option.textColor,
                        fontWeight: 'bold',
                        minWidth: '80px'
                      }}
                    />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          </Box>
        </Popover>
      </>
    );
  };

  const handleViewRequest = (variant) => {
    setSelectedVariant(variant);
      setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedVariant(null);
  };

  const handleOpen = () => {
    setOpen(true);
    setVariantData({
      entity: '',
      model: '',
      variant: '',
      reason: '',
      vector_db_type: 'QDRANT',
      type: 'EXPERIMENT',
      caching_configuration: {
        in_memory_caching_enabled: true,
        in_memory_cache_ttl_seconds: 300,
        distributed_caching_enabled: false,
        distributed_cache_ttl_seconds: 600,
        embedding_retrieval_in_memory_config: { enabled: true, ttl: 60 },
        embedding_retrieval_distributed_config: { enabled: false, ttl: 300 },
        dot_product_in_memory_config: { enabled: true, ttl: 30 },
        dot_product_distributed_config: { enabled: false, ttl: 120 },
      },
      filter_configuration: {
        criteria: [],
        enabled: false,
        selected_filters: []
      },
      vector_db_config: {},
      rate_limiter: { rate_limit: 0, burst_limit: 0 },
      rt_partition: 0
    });
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    
    if (name.startsWith('caching_configuration.')) {
      const configField = name.replace('caching_configuration.', '');
      
      // Handle nested config objects
      if (configField.includes('.')) {
        const [configType, field] = configField.split('.');
    setVariantData(prev => ({
      ...prev,
          caching_configuration: {
            ...prev.caching_configuration,
            [configType]: {
              ...prev.caching_configuration[configType],
              [field]: type === 'checkbox' ? checked : (type === 'number' ? parseInt(value, 10) || 0 : value)
            }
      }
    }));
      } else {
    setVariantData(prev => ({
      ...prev,
          caching_configuration: {
            ...prev.caching_configuration,
            [configField]: type === 'checkbox' ? checked : (type === 'number' ? parseInt(value, 10) || 0 : value)
      }
    }));
      }
    } else if (name.startsWith('filter_configuration.')) {
      const configField = name.replace('filter_configuration.', '');
    setVariantData(prev => ({
      ...prev,
      filter_configuration: {
        ...prev.filter_configuration,
          [configField]: type === 'checkbox' ? checked : (type === 'array' ? value : (type === 'number' ? parseInt(value, 10) || 0 : value))
      }
    }));
    } else {
      setVariantData(prev => ({
        ...prev,
        [name]: type === 'checkbox' ? checked : (type === 'number' ? parseInt(value, 10) || 0 : value)
      }));
    }
  };

  const handleSubmit = async () => {
    if (entities.length === 0) {
      showNotification('No approved entities available. Please create and approve entities first.', "error");
      return;
    }

    if (!variantData.entity || !variantData.model || !variantData.variant || !variantData.reason) {
      showNotification('Please fill in all required fields.', "error");
      return;
    }

    if (variantData.reason.length < 10) {
      showNotification('Reason should be at least 10 characters.', "error");
      return;
    }

    try {
      setLoading(true);
      // Extract reason from variantData for the top-level payload
      const { reason, ...variantPayload } = variantData;
      
      const payload = {
        requestor: user?.email || 'user@example.com',
        reason: reason,
        payload: variantPayload
      };

      const response = await embeddingPlatformAPI.registerVariant(payload);
      
      if (response) {
        showNotification(response.message || 'Variant registration request submitted successfully', "success");
        handleClose();
        fetchData();
      }
    } catch (error) {
      console.error('Error submitting variant request:', error);
      showNotification(error.message || 'Failed to submit variant registration request', "error");
    } finally {
      setLoading(false);
    }
  };

  const showNotification = (message, severity) => {
    setNotification({ open: true, message, severity });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({ ...prev, open: false }));
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Paper elevation={0} sx={{ width: '100%', height: '100vh', padding: '2rem', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h6">Variant Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage A/B testing variant registration requests
          </Typography>
        </Box>
      </Box>

      {/* Search and Register Button */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '1rem', mb: 2 }}>
        <TextField
          label="Search Variants"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
          size="small"
          fullWidth
        />
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleOpen}
          sx={{ backgroundColor: '#450839', '&:hover': { backgroundColor: '#380730' }, minWidth: '200px' }}
        >
          Register Variant
        </Button>
      </Box>

      {/* Variants Table */}
      <TableContainer
        component={Paper}
        elevation={3}
          sx={{
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1200,
            borderCollapse: 'separate',
            borderSpacing: 0,
          },
          '& .MuiTableHead-root': {
            position: 'sticky',
            top: 0,
            zIndex: 1,
            backgroundColor: '#E6EBF2',
          }
        }}
      >
        <Table stickyHeader>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Request ID
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Entity
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Model
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Variant
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Type
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Created By
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', borderRight: '1px solid rgba(224, 224, 224, 1)', position: 'relative' }}>
                <StatusColumnHeader />
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022' }}>
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredRequests.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {variantRequests.length === 0 ? 'No variant requests found' : 'No variants match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredRequests.map((request, index) => (
                <TableRow key={request.request_id || index} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.request_id || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Chip 
                      label={request.payload?.entity || 'N/A'}
                      size="small"
                      sx={{ backgroundColor: '#e3f2fd', color: '#1976d2', fontWeight: 600 }}
                    />
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.payload?.model || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <ExperimentIcon fontSize="small" sx={{ color: '#ff9800' }} />
                      <Typography 
                        variant="body2" 
          sx={{
                          fontFamily: 'monospace', 
                          backgroundColor: '#fff3e0',
                          padding: '2px 6px',
                          borderRadius: '4px',
                          fontSize: '0.875rem',
                          color: '#f57c00'
                        }}
                      >
                        {request.payload?.variant || 'N/A'}
                      </Typography>
      </Box>
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Chip 
                      label={request.payload?.type || 'EXPERIMENT'}
                      size="small"
                      sx={{ backgroundColor: '#e3f2fd', color: '#1976d2', fontWeight: 600 }}
                    />
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.created_by || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {getStatusChip(request.status)}
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      <IconButton size="small" onClick={() => handleViewRequest(request)} title="View Details">
                        <VisibilityIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Variant Registration Dialog */}
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle>Register Variant</DialogTitle>
        <DialogContent>
          {entities.length === 0 && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              No approved entities available. Please create and approve entities first before registering variants.
            </Alert>
          )}

          <Box sx={{ mt: 2 }}>
              <Grid container spacing={2}>
              <Grid item xs={6}>
                <FormControl fullWidth required size="small">
                  <InputLabel>Entity</InputLabel>
                  <Select name="entity" value={variantData.entity} onChange={handleChange} label="Entity">
                    {entities.map((entity) => (
                      <MenuItem key={entity.name} value={entity.name}>
                        {entity.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required size="small">
                  <InputLabel>Model</InputLabel>
                  <Select name="model" value={variantData.model} onChange={handleChange} label="Model" disabled={!variantData.entity}>
                    {models.map((model) => (
                      <MenuItem key={model.model || model.name} value={model.model || model.name}>
                        {model.model || model.name}
                          </MenuItem>
                    ))}
                    </Select>
                  </FormControl>
                </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required size="small">
                  <InputLabel>Variant Name</InputLabel>
                  <Select
                    name="variant"
                    value={variantData.variant}
                    onChange={handleChange}
                    label="Variant Name"
                  >
                    {variantsList.map((variant) => (
                      <MenuItem key={variant} value={variant}>
                        {variant}
                      </MenuItem>
                    ))}
                  </Select>
                  <FormHelperText>Select a variant from the available list</FormHelperText>
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required size="small">
                  <InputLabel>Type</InputLabel>
                  <Select 
                    name="type" 
                    value={variantData.type} 
                    onChange={handleChange} 
                    label="Type"
                    disabled
                  >
                    <MenuItem value="EXPERIMENT">EXPERIMENT</MenuItem>
                  </Select>
                  <FormHelperText>Automatically set to EXPERIMENT</FormHelperText>
                </FormControl>
                </Grid>

              <Grid item xs={12}>
                  <TextField
                    fullWidth
                  required
                  multiline
                  rows={2}
                  size="small"
                  name="reason"
                  label="Reason for Registration"
                  value={variantData.reason}
                        onChange={handleChange}
                  helperText="Explain why this variant is needed (minimum 10 characters)"
                  placeholder="Creating A/B test variant for personalized recommendations to improve user engagement by 15%"
                  />
                </Grid>

              <Grid item xs={6}>
                  <TextField
                    fullWidth
                  size="small"
                  name="vector_db_type"
                  label="Vector DB Type"
                  value={variantData.vector_db_type}
                    disabled
                  helperText="Automatically set to QDRANT"
                  InputProps={{
                    startAdornment: <LockIcon fontSize="small" sx={{ color: 'action.disabled', mr: 1 }} />
                  }}
                  />
                </Grid>

                <Grid item xs={12}>
                <Typography variant="subtitle1" sx={{ mt: 2, mb: 1, color: '#1976d2' }}>
                  Caching Configuration
                  </Typography>
              </Grid>

              <Grid item xs={6}>
                  <FormControlLabel
                    control={
                      <Switch
                      checked={variantData.caching_configuration.in_memory_caching_enabled}
                        onChange={handleChange}
                        name="caching_configuration.in_memory_caching_enabled"
                    />
                  }
                  label="In-Memory Caching"
                />
              </Grid>
              <Grid item xs={6}>
                    <TextField
                      fullWidth
                  type="number"
                      size="small"
                  name="caching_configuration.in_memory_cache_ttl_seconds"
                  label="In-Memory TTL (seconds)"
                  value={variantData.caching_configuration.in_memory_cache_ttl_seconds}
                  onChange={handleChange}
                  disabled={!variantData.caching_configuration.in_memory_caching_enabled}
                  />
                </Grid>

              <Grid item xs={6}>
                  <FormControlLabel
                    control={
                      <Switch
                      checked={variantData.caching_configuration.distributed_caching_enabled}
                        onChange={handleChange}
                        name="caching_configuration.distributed_caching_enabled"
                      />
                    }
                  label="Distributed Caching"
                  />
                </Grid>
              <Grid item xs={6}>
                    <TextField
                  fullWidth
                      type="number"
                  size="small"
                  name="caching_configuration.distributed_cache_ttl_seconds"
                  label="Distributed TTL (seconds)"
                  value={variantData.caching_configuration.distributed_cache_ttl_seconds}
                      onChange={handleChange}
                  disabled={!variantData.caching_configuration.distributed_caching_enabled}
                    />
                  </Grid>

              {/* Embedding Retrieval Caching */}
              <Grid item xs={12}>
                <Typography variant="body2" sx={{ mt: 1, mb: 1, color: '#1976d2', fontWeight: 500 }}>
                  Embedding Retrieval Caching
                </Typography>
              </Grid>
              
              <Grid item xs={3}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={variantData.caching_configuration.embedding_retrieval_in_memory_config.enabled}
                      onChange={handleChange}
                      name="caching_configuration.embedding_retrieval_in_memory_config.enabled"
                    />
                  }
                  label="In-Memory"
                />
              </Grid>
              <Grid item xs={3}>
                    <TextField
                  fullWidth
                      type="number"
                  size="small"
                  name="caching_configuration.embedding_retrieval_in_memory_config.ttl"
                  label="TTL (seconds)"
                  value={variantData.caching_configuration.embedding_retrieval_in_memory_config.ttl}
                      onChange={handleChange}
                  disabled={!variantData.caching_configuration.embedding_retrieval_in_memory_config.enabled}
                />
              </Grid>
              <Grid item xs={3}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={variantData.caching_configuration.embedding_retrieval_distributed_config.enabled}
                      onChange={handleChange}
                      name="caching_configuration.embedding_retrieval_distributed_config.enabled"
                    />
                  }
                  label="Distributed"
                />
              </Grid>
              <Grid item xs={3}>
                <TextField
                      fullWidth
                  type="number"
                  size="small"
                  name="caching_configuration.embedding_retrieval_distributed_config.ttl"
                  label="TTL (seconds)"
                  value={variantData.caching_configuration.embedding_retrieval_distributed_config.ttl}
                  onChange={handleChange}
                  disabled={!variantData.caching_configuration.embedding_retrieval_distributed_config.enabled}
                    />
                  </Grid>

              {/* Dot Product Caching */}
                <Grid item xs={12}>
                <Typography variant="body2" sx={{ mt: 1, mb: 1, color: '#1976d2', fontWeight: 500 }}>
                  Dot Product Caching
                  </Typography>
                </Grid>

              <Grid item xs={3}>
                  <FormControlLabel
                    control={
                      <Switch
                      checked={variantData.caching_configuration.dot_product_in_memory_config.enabled}
                      onChange={handleChange}
                      name="caching_configuration.dot_product_in_memory_config.enabled"
                    />
                  }
                  label="In-Memory"
                />
              </Grid>
              <Grid item xs={3}>
                    <TextField
                      fullWidth
                  type="number"
                      size="small"
                  name="caching_configuration.dot_product_in_memory_config.ttl"
                  label="TTL (seconds)"
                  value={variantData.caching_configuration.dot_product_in_memory_config.ttl}
                  onChange={handleChange}
                  disabled={!variantData.caching_configuration.dot_product_in_memory_config.enabled}
                />
                </Grid>
              <Grid item xs={3}>
                  <FormControlLabel
                    control={
                      <Switch
                      checked={variantData.caching_configuration.dot_product_distributed_config.enabled}
                      onChange={handleChange}
                      name="caching_configuration.dot_product_distributed_config.enabled"
                    />
                  }
                  label="Distributed"
                />
              </Grid>
              <Grid item xs={3}>
                    <TextField
                      fullWidth
                  type="number"
                      size="small"
                  name="caching_configuration.dot_product_distributed_config.ttl"
                  label="TTL (seconds)"
                  value={variantData.caching_configuration.dot_product_distributed_config.ttl}
                  onChange={handleChange}
                  disabled={!variantData.caching_configuration.dot_product_distributed_config.enabled}
                />
                </Grid>

          {/* Filter Configuration */}
              <Grid item xs={12}>
                <Typography variant="subtitle1" sx={{ mt: 2, mb: 1, color: '#1976d2' }}>
                  Filter Configuration
                </Typography>
              </Grid>
              
              <Grid item xs={6}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={variantData.filter_configuration.enabled}
                      onChange={handleChange}
                      name="filter_configuration.enabled"
                    />
                  }
                  label="Enable Filtering"
                />
              </Grid>
              <Grid item xs={6}>
                <FormControl fullWidth size="small" disabled={!variantData.filter_configuration.enabled}>
                  <InputLabel>Available Filters</InputLabel>
                  <Select
                    multiple
                    name="filter_configuration.selected_filters"
                    value={variantData.filter_configuration.selected_filters}
                    onChange={(e) => {
                      const selectedFilters = e.target.value;
                      // Update selected filters
                      handleChange({
                        target: {
                          name: 'filter_configuration.selected_filters',
                          value: selectedFilters,
                          type: 'array'
                        }
                      });
                      
                      // Convert selected filters to criteria format
                      const criteria = selectedFilters.map(filterId => {
                        const filter = filters.find(f => f.id === filterId || f.filter_id === filterId);
                        return filter ? {
                          column_name: filter.filter?.column_name || filter.column_name,
                          operator: 'equals',
                          filter_value: filter.filter?.filter_value || filter.filter_value,
                          default_value: filter.filter?.default_value || filter.default_value
                        } : null;
                      }).filter(Boolean);
                      
                      // Update criteria as well
                      handleChange({
                        target: {
                          name: 'filter_configuration.criteria',
                          value: criteria,
                          type: 'array'
                        }
                      });
                    }}
                    label="Available Filters"
                    renderValue={(selected) => (
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                        {selected.map((filterId) => {
                          const filter = filters.find(f => f.id === filterId || f.filter_id === filterId);
                          return (
                            <Chip 
                              key={filterId} 
                              label={filter ? `${filter.entity}: ${filter.filter?.column_name || filter.column_name}` : filterId}
                      size="small"
                            />
                          );
                        })}
                  </Box>
                    )}
                  >
                    {filters
                      .filter(filter => !variantData.entity || filter.entity === variantData.entity)
                      .map((filter) => (
                        <MenuItem 
                          key={filter.id || filter.filter_id} 
                          value={filter.id || filter.filter_id}
                        >
                          <Box>
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>
                              {filter.entity}: {filter.filter?.column_name || filter.column_name}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              Value: {filter.filter?.filter_value || filter.filter_value} 
                              (Default: {filter.filter?.default_value || filter.default_value})
                            </Typography>
                          </Box>
                            </MenuItem>
                          ))}
                        </Select>
                  <FormHelperText>
                    Select filters to apply to this variant
                    {variantData.entity && ` (showing filters for ${variantData.entity})`}
                  </FormHelperText>
                      </FormControl>
                    </Grid>

              <Grid item xs={12}>
                <Alert severity="info" sx={{ mt: 2 }}>
                  Rate limiting, RT partition, and Vector DB configuration will be set by admin during approval.
                </Alert>
                    </Grid>
                  </Grid>
                </Box>
        </DialogContent>
        <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
          <Button
            variant="outlined"
            onClick={handleClose}
            sx={{ borderColor: '#522b4a', color: '#522b4a', '&:hover': { borderColor: '#613a5c', backgroundColor: 'rgba(82, 43, 74, 0.04)' } }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={loading}
            sx={{ backgroundColor: '#522b4a', color: 'white', '&:hover': { backgroundColor: '#613a5c' }, '&:disabled': { backgroundColor: '#cccccc' } }}
          >
            {loading ? 'Processing...' : 'Submit Request'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Variant Details Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ExperimentIcon sx={{ color: '#522b4a' }} />
            Variant Request Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedVariant && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
              <TextField
                multiline
                rows={12}
                value={selectedVariant.payload ? JSON.stringify(selectedVariant.payload, null, 2) : '{}'}
                variant="outlined"
                fullWidth
                InputProps={{ 
                  readOnly: true,
                  style: { fontFamily: 'monospace', fontSize: '0.875rem' }
                }}
                sx={{ backgroundColor: '#fafafa' }}
              />
          </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2, backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button 
            onClick={handleCloseViewModal}
            sx={{ color: '#522b4a', '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' } }}
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    
      {/* Notification Snackbar */}
        <Snackbar
          open={notification.open}
          autoHideDuration={6000}
          onClose={handleCloseNotification}
          anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        >
          <Alert
            onClose={handleCloseNotification}
            severity={notification.severity}
            sx={{ width: '100%' }}
          >
            {notification.message}
          </Alert>
        </Snackbar>
    </Paper>
  );
};

export default VariantRegistry;