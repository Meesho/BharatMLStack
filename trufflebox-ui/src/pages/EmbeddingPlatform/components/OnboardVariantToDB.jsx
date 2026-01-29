import React, { useState, useEffect, useMemo } from 'react';
import {
  Paper,
  Box,
  Typography,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Grid,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Snackbar,
  Alert,
  FormHelperText,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import StorageIcon from '@mui/icons-material/Storage';
import SearchIcon from '@mui/icons-material/Search';
import FilterListIcon from '@mui/icons-material/FilterList';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useAuth } from '../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../services/embeddingPlatform/api';
import { DATABASE_TYPE_OPTIONS, DEFAULT_DATABASE_TYPE } from '../../../constants/databaseTypes';

const OnboardVariantToDB = () => {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const { user } = useAuth();
  const [notification, setNotification] = useState({ open: false, message: "", severity: "success" });
  const [entities, setEntities] = useState([]);
  const [models, setModels] = useState([]);
  const [variants, setVariants] = useState([]);
  const [validationErrors, setValidationErrors] = useState({});
  
  // Requests table state
  const [onboardingRequests, setOnboardingRequests] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING', 'IN_PROGRESS', 'COMPLETED']);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);

  // Onboarding form state
  const [onboardingData, setOnboardingData] = useState({
    requestor: '',
    reason: '',
    entity: '',
    model: '',
    variant: '',
    vector_db_type: DEFAULT_DATABASE_TYPE,
  });

  const statusOptions = [
    { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
    { value: 'IN_PROGRESS', label: 'In Progress', color: '#E3F2FD', textColor: '#1976D2' },
    { value: 'COMPLETED', label: 'Completed', color: '#E8F5E8', textColor: '#2E7D32' },
    { value: 'FAILED', label: 'Failed', color: '#FFEBEE', textColor: '#D32F2F' },
  ];

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    if (onboardingData.entity) {
      fetchModelsForEntity(onboardingData.entity);
    } else {
      setModels([]);
      setOnboardingData(prev => ({ ...prev, model: '' }));
    }
  }, [onboardingData.entity]);

  useEffect(() => {
    if (onboardingData.model) {
      fetchVariantsForModel(onboardingData.entity, onboardingData.model);
    } else {
      setVariants([]);
      setOnboardingData(prev => ({ ...prev, variant: '' }));
    }
  }, [onboardingData.model]);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const [entitiesResponse, requestsResponse] = await Promise.all([
        embeddingPlatformAPI.getEntities(),
        embeddingPlatformAPI.getAllVariantOnboardingRequests()
      ]);
      
      if (entitiesResponse.entities) {
        const availableEntities = entitiesResponse.entities?.map(entity => ({
          name: entity.name,
          store_id: entity.store_id,
          label: `${entity.name} (Store ${entity.store_id})`
        })) || [];
        setEntities(availableEntities);
      }

      if (requestsResponse.variant_onboarding_requests) {
        setOnboardingRequests(requestsResponse.variant_onboarding_requests);
      } else {
        setOnboardingRequests([]);
      }
    } catch (error) {
      console.error('Error fetching data:', error);
      setError('Failed to load data. Please refresh the page.');
    } finally {
      setLoading(false);
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

  const fetchVariantsForModel = async (entityName, modelName) => {
    try {
      const response = await embeddingPlatformAPI.getVariants({ entity: entityName, model: modelName });
      if (response.variants) {
        // If variants is an object (map), convert to array
        let variantsList = [];
        if (typeof response.variants === 'object' && !Array.isArray(response.variants)) {
          variantsList = Object.values(response.variants).flat();
        } else {
          variantsList = response.variants;
        }
        const modelVariants = variantsList.filter(variant => 
          (variant.status || '') === 'active'
        );
        setVariants(modelVariants);
      } else {
        setVariants([]);
      }
    } catch (error) {
      console.error('Error fetching variants for model:', error);
      setVariants([]);
    }
  };

  const handleOpen = () => {
    setOpen(true);
    setOnboardingData({
      requestor: user?.email || '',
      reason: '',
      entity: '',
      model: '',
      variant: '',
      vector_db_type: DEFAULT_DATABASE_TYPE,
    });
    setValidationErrors({});
  };

  const handleClose = () => {
    setOpen(false);
    setValidationErrors({});
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setOnboardingData(prev => ({
      ...prev,
      [name]: value
    }));
    
    // Clear validation error for this field
    if (validationErrors[name]) {
      setValidationErrors(prev => ({
        ...prev,
        [name]: ''
      }));
    }
  };

  const validateForm = () => {
    const errors = {};
    
    if (!onboardingData.entity) {
      errors.entity = 'Entity is required';
    }
    if (!onboardingData.model) {
      errors.model = 'Model is required';
    }
    if (!onboardingData.variant) {
      errors.variant = 'Variant is required';
    }
    if (!onboardingData.reason || onboardingData.reason.length < 10) {
      errors.reason = 'Reason must be at least 10 characters';
    }
    if (!onboardingData.vector_db_type) {
      errors.vector_db_type = 'Vector DB type is required';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      showNotification('Please fix the validation errors before submitting.', "error");
      return;
    }

    if (entities.length === 0) {
      showNotification('No approved entities available. Please create and approve entities first.', "error");
      return;
    }

    try {
      setLoading(true);
      
      const payload = {
        requestor: onboardingData.requestor,
        reason: onboardingData.reason,
        payload: {
          entity: onboardingData.entity,
          model: onboardingData.model,
          variant: onboardingData.variant,
          vector_db_type: onboardingData.vector_db_type,
        }
      };

      const response = await embeddingPlatformAPI.onboardVariant(payload);
      
      if (response) {
        showNotification(response.message || 'Variant onboarding request submitted successfully', "success");
        handleClose();
        fetchData(); // Refresh the requests list
      }
    } catch (error) {
      console.error('Error submitting onboarding request:', error);
      showNotification(error.message || 'Failed to submit variant onboarding request', "error");
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

  const filteredRequests = useMemo(() => {
    let filtered = onboardingRequests.filter(request => 
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
  }, [onboardingRequests, searchQuery, selectedStatuses]);

  const getStatusChip = (status) => {
    const statusUpper = (status || 'PENDING').toUpperCase();
    const option = statusOptions.find(opt => opt.value === statusUpper);
    
    if (!option) {
      return (
        <Chip
          label={statusUpper}
          size="small"
          sx={{ backgroundColor: '#f5f5f5', color: '#666', fontWeight: 'bold', minWidth: '80px' }}
        />
      );
    }

    return (
      <Chip
        label={option.label}
        size="small"
        sx={{ backgroundColor: option.color, color: option.textColor, fontWeight: 'bold', minWidth: '80px' }}
      />
    );
  };

  const handleViewRequest = (request) => {
    setSelectedRequest(request);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedRequest(null);
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

  if (loading && entities.length === 0) {
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
        <StorageIcon sx={{ fontSize: 32, color: '#450839', mr: 2 }} />
        <Box>
          <Typography variant="h6">Onboard Variant to Database</Typography>
          <Typography variant="body1" color="text.secondary">
            Request to onboard approved variants to vector database for production use
          </Typography>
        </Box>
      </Box>

      {/* Search and Create Button */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '1rem', mb: 2 }}>
        <TextField
          label="Search Requests"
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
          disabled={entities.length === 0}
          sx={{ 
            backgroundColor: '#450839', 
            '&:hover': { backgroundColor: '#380730' },
            minWidth: '200px'
          }}
        >
          Create Request
        </Button>
      </Box>

      {/* Requests Table */}
      <TableContainer
        component={Paper}
        elevation={3}
        sx={{
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 300px)',
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
                Vector DB Type
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Requestor
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
                    {onboardingRequests.length === 0 ? 'No onboarding requests found' : 'No requests match your search'}
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
                      <StorageIcon fontSize="small" sx={{ color: '#ff9800' }} />
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
                      label={request.payload?.vector_db_type || 'QDRANT'}
                      size="small"
                      sx={{ backgroundColor: '#e8f5e8', color: '#2e7d32', fontWeight: 600 }}
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

      {/* Onboarding Request Dialog */}
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <StorageIcon sx={{ color: '#450839' }} />
            Onboard Variant to Database
          </Box>
        </DialogTitle>
        <DialogContent>
          {entities.length === 0 && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              No approved entities available. Please create and approve entities first before onboarding variants.
            </Alert>
          )}

          <Box sx={{ mt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <FormControl fullWidth required size="small" error={!!validationErrors.entity}>
                  <InputLabel>Entity</InputLabel>
                  <Select 
                    name="entity" 
                    value={onboardingData.entity} 
                    onChange={handleChange} 
                    label="Entity"
                  >
                    {entities.map((entity) => (
                      <MenuItem key={entity.name} value={entity.name}>
                        {entity.label}
                      </MenuItem>
                    ))}
                  </Select>
                  {validationErrors.entity && (
                    <FormHelperText>{validationErrors.entity}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required size="small" error={!!validationErrors.model}>
                  <InputLabel>Model</InputLabel>
                  <Select 
                    name="model" 
                    value={onboardingData.model} 
                    onChange={handleChange} 
                    label="Model"
                    disabled={!onboardingData.entity}
                  >
                    {models.map((model) => (
                      <MenuItem key={model.model || model.name} value={model.model || model.name}>
                        {model.model || model.name}
                      </MenuItem>
                    ))}
                  </Select>
                  {validationErrors.model && (
                    <FormHelperText>{validationErrors.model}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required size="small" error={!!validationErrors.variant}>
                  <InputLabel>Variant</InputLabel>
                  <Select 
                    name="variant" 
                    value={onboardingData.variant} 
                    onChange={handleChange} 
                    label="Variant"
                    disabled={!onboardingData.model}
                  >
                    {variants.map((variant) => (
                      <MenuItem key={variant.variant || variant.name} value={variant.variant || variant.name}>
                        {variant.variant || variant.name}
                      </MenuItem>
                    ))}
                  </Select>
                  {validationErrors.variant && (
                    <FormHelperText>{validationErrors.variant}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required size="small" error={!!validationErrors.vector_db_type}>
                  <InputLabel>Vector Database Type</InputLabel>
                  <Select 
                    name="vector_db_type" 
                    value={onboardingData.vector_db_type} 
                    onChange={handleChange} 
                    label="Vector Database Type"
                  >
                    {DATABASE_TYPE_OPTIONS.map((option) => (
                      <MenuItem key={option.value} value={option.value}>
                        <Box>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {option.label}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {option.description}
                          </Typography>
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                  {validationErrors.vector_db_type && (
                    <FormHelperText>{validationErrors.vector_db_type}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  required
                  multiline
                  rows={3}
                  size="small"
                  name="reason"
                  label="Reason for Onboarding"
                  value={onboardingData.reason}
                  onChange={handleChange}
                  error={!!validationErrors.reason}
                  helperText={validationErrors.reason || "Explain why this variant needs to be onboarded to the database (minimum 10 characters)"}
                  placeholder="Onboarding this variant to enable production vector search capabilities for personalized recommendations..."
                />
              </Grid>

              <Grid item xs={12}>
                <Alert severity="info" sx={{ mt: 2 }}>
                  <Typography variant="body2">
                    <strong>What happens next?</strong>
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 1 }}>
                    1. Your request will be submitted for admin approval<br/>
                    2. Once approved, an Airflow job will be triggered<br/>
                    3. The job will create Qdrant collections and populate them with data<br/>
                    4. You'll be notified when the onboarding process is complete
                  </Typography>
                </Alert>
              </Grid>
            </Grid>
          </Box>
        </DialogContent>
        <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
          <Button
            variant="outlined"
            onClick={handleClose}
            disabled={loading}
            sx={{ borderColor: '#522b4a', color: '#522b4a', '&:hover': { borderColor: '#613a5c', backgroundColor: 'rgba(82, 43, 74, 0.04)' } }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={loading || entities.length === 0}
            sx={{ backgroundColor: '#522b4a', color: 'white', '&:hover': { backgroundColor: '#613a5c' }, '&:disabled': { backgroundColor: '#cccccc' } }}
          >
            {loading ? 'Submitting...' : 'Submit Request'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Request Details Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <StorageIcon sx={{ color: '#522b4a' }} />
            Onboarding Request Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedRequest && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
              <TextField
                multiline
                rows={12}
                value={selectedRequest.payload ? JSON.stringify(selectedRequest.payload, null, 2) : '{}'}
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

export default OnboardVariantToDB;
