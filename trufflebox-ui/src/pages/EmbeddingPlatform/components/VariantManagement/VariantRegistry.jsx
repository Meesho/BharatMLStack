import React, { useState, useEffect } from 'react';
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
  FormControl,
  FormHelperText,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Snackbar,
  Collapse,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ExperimentIcon from '@mui/icons-material/Science';
import InfoIcon from '@mui/icons-material/Info';
import LockIcon from '@mui/icons-material/Lock';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const VariantRegistry = () => {
  const [variantRequests, setVariantRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['APPROVED', 'PENDING', 'REJECTED']);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [showRawJsonInViewModal, setShowRawJsonInViewModal] = useState(false);
  const [selectedVariant, setSelectedVariant] = useState(null);
  const [entities, setEntities] = useState([]);
  const [models, setModels] = useState([]);
  const [filters, setFilters] = useState([]);
  const [variantsList, setVariantsList] = useState([]);
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();
  
  const [variantData, setVariantData] = useState({
    entity: '',
    model: '',
    variant: '',
    reason: '',
    otd_training_data_path: '',
    vector_db_type: 'QDRANT',
    type: 'EXPERIMENT',
    filter_configuration: {
      criteria: [], // [{ column_name, condition: 'EQUALS' | 'NOT_EQUALS' }]
    },
    vector_db_config: {},
  });
  const [filterAddColumnName, setFilterAddColumnName] = useState('');
  const [filterAddCondition, setFilterAddCondition] = useState('EQUALS');

  useEffect(() => {
    fetchData();
    fetchVariantsList();
  }, []);

  useEffect(() => {
    if (variantData.entity) {
      fetchModelsForEntity(variantData.entity);
      fetchFiltersForEntity(variantData.entity);
    } else {
      setModels([]);
      setFilters([]);
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
      if (response.models && typeof response.models === 'object' && !Array.isArray(response.models)) {
        const modelsArray = [];
        Object.entries(response.models).forEach(([entity, entityData]) => {
          if (entityName && entity !== entityName) return;
          const modelsObj = entityData?.Models ?? entityData?.models;
          if (modelsObj) {
            Object.entries(modelsObj).forEach(([modelName, modelData]) => {
              const modelType = modelData.model_type ?? modelData.ModelType ?? '';
              modelsArray.push({
                model: modelName,
                name: modelName,
                entity,
                model_type: modelType,
                ModelType: modelType,
                ...modelData,
              });
            });
          }
        });
        setModels(modelsArray);
      } else if (Array.isArray(response.models)) {
        // Fallback for array format
        setModels(response.models);
      } else {
        setModels([]);
      }
    } catch (error) {
      console.error('Error fetching models for entity:', error);
      setModels([]);
    }
  };

  const fetchFiltersForEntity = async (entityName) => {
    try {
      const response = await embeddingPlatformAPI.getFilters({ entity: entityName });
      
      if (response.filters && typeof response.filters === 'object') {
        const filtersArray = Object.entries(response.filters).map(([filterName, filterData], index) => ({
          id: `${entityName}_${filterName}_${index}`,
          filter_id: `${entityName}_${filterName}_${index}`,
          entity: entityName,
          column_name: filterData.column_name || filterName,
          filter_value: filterData.filter_value || '',
          default_value: filterData.default_value || '',
          filter: {
            column_name: filterData.column_name || filterName,
            filter_value: filterData.filter_value || '',
            default_value: filterData.default_value || ''
          }
        }));
        
        setFilters(filtersArray);
      } else {
        setModels([]);
      }
    } catch (error) {
      console.error('Error fetching filters for entity:', error);
      setFilters([]);
    }
  };

  const filteredRequests = useTableFilter({
    data: variantRequests,
    searchQuery,
    selectedStatuses,
    searchFields: (request) => [
      request.request_id,
      request.payload?.entity,
      request.payload?.model,
      request.payload?.variant,
      request.created_by,
      request.status,
    ],
    sortField: 'created_at',
    sortOrder: 'desc',
  });

  const handleViewRequest = (variant) => {
    setSelectedVariant(variant);
      setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setShowRawJsonInViewModal(false);
    setSelectedVariant(null);
  };

  const handleOpen = () => {
    setOpen(true);
    setFilterAddColumnName('');
    setFilterAddCondition('EQUALS');
    setVariantData({
      entity: '',
      model: '',
      variant: '',
      reason: '',
      otd_training_data_path: '',
      vector_db_type: 'QDRANT',
      type: 'EXPERIMENT',
      filter_configuration: { criteria: [] },
      vector_db_config: {},
    });
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    if (name.startsWith('filter_configuration.')) {
      const configField = name.replace('filter_configuration.', '');
      setVariantData(prev => ({
        ...prev,
        filter_configuration: {
          ...prev.filter_configuration,
          [configField]: type === 'array' ? value : (type === 'number' ? parseInt(value, 10) || 0 : value),
        },
      }));
    } else {
      setVariantData(prev => ({
        ...prev,
        [name]: type === 'checkbox' ? checked : (type === 'number' ? parseInt(value, 10) || 0 : value),
      }));
    }
  };

  const handleAddFilterCriterion = () => {
    if (!filterAddColumnName) return;
    const columnName = filterAddColumnName;
    const condition = filterAddCondition;
    setVariantData(prev => ({
      ...prev,
      filter_configuration: {
        ...prev.filter_configuration,
        criteria: [...(prev.filter_configuration.criteria || []), { column_name: columnName, condition }],
      },
    }));
    setFilterAddColumnName('');
    setFilterAddCondition('EQUALS');
  };

  const handleRemoveFilterCriterion = (index) => {
    setVariantData(prev => ({
      ...prev,
      filter_configuration: {
        ...prev.filter_configuration,
        criteria: prev.filter_configuration.criteria.filter((_, i) => i !== index),
      },
    }));
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
      const selectedModel = models.find((m) => (m.model || m.name) === variantData.model);
      const modelType = selectedModel?.model_type ?? selectedModel?.ModelType ?? '';
      const isDeltaModel = String(modelType).toUpperCase() === 'DELTA';

      const { reason, ...rest } = variantData;
      const variantPayload = {
        ...rest,
        model_type: modelType,
        otd_training_data_path: isDeltaModel ? (variantData.otd_training_data_path ?? '') : '',
        filter_configuration: { criteria: variantData.filter_configuration?.criteria ?? [] },
        vector_db_config: variantData.vector_db_config ?? {},
        rate_limiter: { rate_limit: 0, burst_limit: 0 },
      };

      const payload = {
        requestor: user?.email || 'user@example.com',
        reason,
        payload: variantPayload,
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
                <StatusFilterHeader
                  selectedStatuses={selectedStatuses}
                  onStatusChange={handleStatusChange}
                />
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
                    <StatusChip status={request.status} />
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

              {(() => {
                const selectedModel = models.find((m) => (m.model || m.name) === variantData.model);
                const isDeltaModel = String(selectedModel?.model_type ?? selectedModel?.ModelType ?? '').toUpperCase() === 'DELTA';
                return isDeltaModel ? (
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      size="small"
                      name="otd_training_data_path"
                      label="OTD Training Data Path"
                      value={variantData.otd_training_data_path ?? ''}
                      onChange={handleChange}
                      placeholder="gs://bucket/path/to/otd-training-data"
                      helperText="Required for DELTA model type"
                    />
                  </Grid>
                ) : null;
              })()}

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

                {/* Filter Configuration: select filter + EQUALS/NOT_EQUALS per criterion */}
              <Grid item xs={12}>
                <Typography variant="subtitle1" sx={{ mt: 2, mb: 1, color: '#1976d2' }}>
                  Filter Configuration
                </Typography>
                <FormHelperText sx={{ mb: 1 }}>
                  Add filters; for each filter choose EQUALS or NOT_EQUALS. Caching and Vector DB config are set by admin during approval.
                </FormHelperText>
              </Grid>
              <Grid item xs={12} sm={5}>
                <FormControl fullWidth size="small">
                  <InputLabel>Filter (column)</InputLabel>
                  <Select
                    value={filterAddColumnName}
                    onChange={(e) => setFilterAddColumnName(e.target.value)}
                    label="Filter (column)"
                  >
                    <MenuItem value="">
                      <em>Select a filter</em>
                    </MenuItem>
                    {(Array.isArray(filters) ? filters : [])
                      .filter((f) => !variantData.entity || f.entity === variantData.entity)
                      .map((filter) => {
                        const col = filter.filter?.column_name || filter.column_name;
                        return (
                          <MenuItem key={filter.id || filter.filter_id} value={col}>
                            {col}
                            {filter.filter?.filter_value != null && (
                              <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 1 }}>
                                (value: {filter.filter.filter_value})
                              </Typography>
                            )}
                          </MenuItem>
                        );
                      })}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} sm={4}>
                <FormControl fullWidth size="small">
                  <InputLabel>Condition</InputLabel>
                  <Select
                    value={filterAddCondition}
                    onChange={(e) => setFilterAddCondition(e.target.value)}
                    label="Condition"
                  >
                    <MenuItem value="EQUALS">EQUALS</MenuItem>
                    <MenuItem value="NOT_EQUALS">NOT_EQUALS</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} sm={3}>
                <Button
                  variant="outlined"
                  startIcon={<AddIcon />}
                  onClick={handleAddFilterCriterion}
                  disabled={!filterAddColumnName}
                  fullWidth
                  sx={{ height: 40, borderColor: '#522b4a', color: '#522b4a', '&:hover': { borderColor: '#613a5c', backgroundColor: 'rgba(82, 43, 74, 0.04)' } }}
                >
                  Add filter
                </Button>
              </Grid>
              {variantData.filter_configuration?.criteria?.length > 0 && (
                <Grid item xs={12}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 1 }}>
                    Added criteria
                  </Typography>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                    {variantData.filter_configuration.criteria.map((c, index) => (
                      <Box
                        key={`${c.column_name}-${index}`}
                        sx={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          py: 0.75,
                          px: 1.5,
                          backgroundColor: 'action.hover',
                          borderRadius: 1,
                        }}
                      >
                        <Typography variant="body2">
                          <strong>{c.column_name}</strong> — {c.condition}
                        </Typography>
                        <IconButton
                          size="small"
                          onClick={() => handleRemoveFilterCriterion(index)}
                          aria-label="Remove filter"
                          sx={{ color: 'error.main' }}
                        >
                          <DeleteOutlineIcon />
                        </IconButton>
                      </Box>
                    ))}
                  </Box>
                </Grid>
              )}

              <Grid item xs={12}>
                <Alert severity="info" sx={{ mt: 2 }}>
                  Caching configuration, rate limiting, RT partition, and Vector DB configuration will be set by admin during approval.
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
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedVariant}
        config={{
          title: 'Variant Request Details',
          icon: ExperimentIcon,
          sections: [
            {
              title: 'Request Information',
              icon: InfoIcon,
              layout: 'grid',
              fields: [
                { label: 'Request ID', key: 'request_id', type: 'monospace' },
                { label: 'Status', key: 'status', type: 'status' },
                { label: 'Created By', key: 'created_by' },
                { label: 'Created At', key: 'created_at', type: 'date' }
              ]
            },
            {
              title: 'Variant Configuration',
              icon: ExperimentIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              render: (data) => {
                const p = data?.payload ?? {};
                const modelType = String(p.model_type ?? '').toUpperCase();
                const isDelta = modelType === 'DELTA';
                return (
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block">Entity</Typography>
                      <Typography variant="body2">{p.entity ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block">Model</Typography>
                      <Typography variant="body2">{p.model ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block">Variant</Typography>
                      <Typography variant="body2">{p.variant ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block">Type</Typography>
                      <Typography variant="body2">{p.type ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block">Vector DB type</Typography>
                      <Typography variant="body2">{p.vector_db_type ?? '—'}</Typography>
                    </Grid>
                    {isDelta && (
                      <Grid item xs={12}>
                        <Typography variant="caption" color="text.secondary" display="block">OTD Training Data Path</Typography>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
                          {p.otd_training_data_path || '—'}
                        </Typography>
                      </Grid>
                    )}
                  </Grid>
                );
              }
            },
            {
              title: 'Filter criteria',
              icon: ExperimentIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              render: (data) => {
                const criteria = data?.payload?.filter_configuration?.criteria;
                if (!criteria?.length) return null;
                return (
                  <TableContainer component={Paper} variant="outlined" sx={{ maxWidth: 400 }}>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 600 }}>Column</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Condition</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {criteria.map((c, i) => (
                          <TableRow key={i}>
                            <TableCell>{c.column_name ?? '—'}</TableCell>
                            <TableCell>{c.condition ?? '—'}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                );
              }
            },
            {
              title: 'Reason',
              icon: ExperimentIcon,
              render: (data) => {
                const reason = data?.payload?.reason;
                if (reason == null || String(reason).trim() === '') return null;
                return (
                  <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>{reason}</Typography>
                );
              }
            },
            {
              title: 'Vector DB configuration',
              icon: ExperimentIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.04)',
              borderColor: 'rgba(25, 118, 210, 0.15)',
              render: (data) => {
                const vdb = data?.payload?.vector_db_config;
                if (vdb == null || typeof vdb !== 'object' || Object.keys(vdb).length === 0) return null;
                return (
                  <Grid container spacing={2}>
                    {Object.entries(vdb).map(([key, val]) => (
                      <Grid item xs={12} sm={6} key={key}>
                        <Typography variant="caption" color="text.secondary" display="block">{key.replace(/_/g, ' ')}</Typography>
                        <Typography variant="body2" sx={{ fontFamily: typeof val === 'object' ? 'monospace' : 'inherit' }}>
                          {typeof val === 'object' ? JSON.stringify(val) : String(val)}
                        </Typography>
                      </Grid>
                    ))}
                  </Grid>
                );
              }
            },
            {
              title: 'Rate limiter',
              icon: ExperimentIcon,
              render: (data) => {
                const rl = data?.payload?.rate_limiter;
                if (rl == null || typeof rl !== 'object') return null;
                return (
                  <Grid container spacing={2}>
                    {rl.rate_limit != null && (
                      <Grid item xs={12} sm={6}>
                        <Typography variant="caption" color="text.secondary" display="block">Rate limit (req/sec)</Typography>
                        <Typography variant="body2">{rl.rate_limit}</Typography>
                      </Grid>
                    )}
                    {rl.burst_limit != null && (
                      <Grid item xs={12} sm={6}>
                        <Typography variant="caption" color="text.secondary" display="block">Burst limit</Typography>
                        <Typography variant="body2">{rl.burst_limit}</Typography>
                      </Grid>
                    )}
                  </Grid>
                );
              }
            },
            {
              title: 'Raw payload',
              icon: ExperimentIcon,
              render: (data) => (
                <Box>
                  <Button
                    size="small"
                    variant="text"
                    onClick={() => setShowRawJsonInViewModal((p) => !p)}
                    sx={{ textTransform: 'none', color: '#1976d2' }}
                  >
                    {showRawJsonInViewModal ? 'Hide raw JSON' : 'Show raw JSON'}
                  </Button>
                  <Collapse in={showRawJsonInViewModal}>
                    <TextField
                      multiline
                      rows={10}
                      value={data?.payload ? JSON.stringify(data.payload, null, 2) : '{}'}
                      variant="outlined"
                      fullWidth
                      InputProps={{ readOnly: true, style: { fontFamily: 'monospace', fontSize: '0.875rem' } }}
                      sx={{ mt: 1, backgroundColor: '#fafafa' }}
                    />
                  </Collapse>
                </Box>
              )
            }
          ]
        }}
      />

      {/* Notification Snackbar */}
        <Snackbar
          open={notification.open}
          autoHideDuration={6000}
          onClose={closeNotification}
          anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        >
          <Alert
            onClose={closeNotification}
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