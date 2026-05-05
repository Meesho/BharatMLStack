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
  FormHelperText,
  Chip,
  IconButton,
  FormControl,
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
import ModelTrainingIcon from '@mui/icons-material/ModelTraining';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import StorageIcon from '@mui/icons-material/Storage';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { 
  validateModelPayload, 
  getDefaultFormValues,
} from '../../../../services/embeddingPlatform/utils';
import { DISTANCE_FUNCTIONS } from '../../../../services/embeddingPlatform/constants';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const formatTtlReadable = (seconds) => {
  const n = Number(seconds);
  if (!Number.isFinite(n) || n < 0) return '';
  if (n === 0) return '0 sec';
  const MIN = 60;
  const HOUR = 3600;
  const DAY = 86400;
  const MONTH = 30 * DAY;
  const YEAR = 365 * DAY;
  if (n >= YEAR) {
    const v = Math.round(n / YEAR);
    return `${v} year${v !== 1 ? 's' : ''}`;
  }
  if (n >= MONTH) {
    const v = Math.round(n / MONTH);
    return `${v} month${v !== 1 ? 's' : ''}`;
  }
  if (n >= DAY) {
    const v = Math.round(n / DAY);
    return `${v} day${v !== 1 ? 's' : ''}`;
  }
  if (n >= HOUR) {
    const v = Math.round(n / HOUR);
    return `${v} hour${v !== 1 ? 's' : ''}`;
  }
  if (n >= MIN) {
    const v = Math.round(n / MIN);
    return `${v} min${v !== 1 ? 's' : ''}`;
  }
  const v = Math.round(n);
  return `${v} sec${v !== 1 ? 's' : ''}`;
};

const ModelRegistry = () => {
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['APPROVED', 'PENDING', 'REJECTED']);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedModel, setSelectedModel] = useState(null);
  const [modelData, setModelData] = useState(getDefaultFormValues('model'));
  const [entities, setEntities] = useState([]);
  const [jobFrequencies, setJobFrequencies] = useState([]);
  const [isEditMode, setIsEditMode] = useState(false);
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();
  const [modalMessage, setModalMessage] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const [modelRequestsResponse, entitiesResponse, jobFrequenciesResponse] = await Promise.all([
        embeddingPlatformAPI.getModelRequests(),
        embeddingPlatformAPI.getEntities(),
        embeddingPlatformAPI.getJobFrequencies()
      ]);

      if (modelRequestsResponse.model_requests) {
        setModels(modelRequestsResponse.model_requests);
      } else {
        setModels([]);
      }

      if (entitiesResponse.entities) {
        const availableEntities = entitiesResponse.entities?.map(entity => ({
          name: entity.name,
          store_id: entity.store_id,
          label: entity.name
        })) || [];
        setEntities(availableEntities);
      }

      if (jobFrequenciesResponse.frequencies) {
        const frequenciesArray = Object.values(jobFrequenciesResponse.frequencies);
        setJobFrequencies(frequenciesArray);
      } else {
        setJobFrequencies([]);
      }
    } catch (error) {
      console.error('Error fetching model data:', error);
      setError('Failed to load model data. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const filteredModels = useTableFilter({
    data: models,
    searchQuery,
    selectedStatuses,
    searchFields: (model) => [
      model.request_id,
      model.model,
      model.entity,
      model.model_type,
      model.created_by,
      model.status,
    ],
    sortField: 'created_at',
    sortOrder: 'desc',
  });

  const handleViewRequest = (model) => {
    setSelectedModel(model);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedModel(null);
  };

  const handleOpen = () => {
    setOpen(true);
    setIsEditMode(false);
    setModelData(getDefaultFormValues('model'));
    setValidationErrors({});
  };

  const handleClose = () => {
    setOpen(false);
    setIsEditMode(false);
    setModelData(getDefaultFormValues('model'));
    setValidationErrors({});
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    
    if (name.startsWith('model_config.')) {
      const configField = name.replace('model_config.', '');
      setModelData(prev => ({
        ...prev,
        model_config: {
          ...prev.model_config,
          [configField]: type === 'number' ? parseInt(value, 10) || '' : value
        }
      }));
    } else {
      setModelData(prev => ({
        ...prev,
        [name]: type === 'checkbox' ? checked : (type === 'number' ? parseInt(value, 10) || '' : value)
      }));
    }
  };

  const handleSubmit = async () => {
    // Check if required data is available
    if (entities.length === 0) {
      setModalMessage('No approved entities available. Please create and approve entities first.');
      showNotification("Missing required data", "error");
      return;
    }

    if (jobFrequencies.length === 0) {
      setModalMessage('No job frequencies available. Please configure job frequencies first.');
      showNotification("Missing required data", "error");
      return;
    }

    const validation = validateModelPayload(modelData);
    if (!validation.isValid) {
      setValidationErrors(validation.errors);
      setModalMessage('Please fix the validation errors and try again.');
      showNotification("Operation failed", "error");
      return;
    }

    try {
      setLoading(true);
      
      // Parse metadata if it's a string
      let parsedMetadata = modelData.metadata;
      if (typeof modelData.metadata === 'string' && modelData.metadata.trim()) {
        try {
          parsedMetadata = JSON.parse(modelData.metadata);
        } catch (parseError) {
          setValidationErrors({ metadata: 'Invalid JSON format. Please provide valid JSON.' });
          setModalMessage('Metadata must be valid JSON format.');
          showNotification("Validation failed", "error");
          setLoading(false);
          return;
        }
      } else if (!modelData.metadata || (typeof modelData.metadata === 'string' && !modelData.metadata.trim())) {
        parsedMetadata = {};
      }
      
      const payload = {
        requestor: user?.email,
        reason: isEditMode ? 'Updating model configuration' : 'Registering new ML model',
        payload: {
          ...modelData,
          metadata: parsedMetadata
        }
      };
      
      const response = isEditMode 
        ? await embeddingPlatformAPI.editModel(payload)
        : await embeddingPlatformAPI.registerModel(payload);
      
      if (response) {
        setModalMessage(response.message || `Model ${isEditMode ? 'update' : 'registration'} request submitted successfully`);
        showNotification("Operation successful", "success");
        handleClose();
        fetchData();
      }
    } catch (error) {
      console.error('Error submitting model request:', error);
      setModalMessage(error.message || `Failed to submit model ${isEditMode ? 'update' : 'registration'} request`);
      showNotification("Operation failed", "error");
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
          <Typography variant="h6">Model Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage model registration requests
          </Typography>
        </Box>
      </Box>

      {/* Search and Register Button */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '1rem',
          mb: 2
        }}
      >
        <TextField
          label="Search Models"
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
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730'
            },
            minWidth: '200px'
          }}
        >
          Register Model
        </Button>
      </Box>

      {/* Models Table */}
      <TableContainer
        component={Paper}
        elevation={3}
        sx={{
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1000,
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
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Request ID
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Model Name
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Entity
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Model Type
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Created By
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                  position: 'relative'
                }}
              >
                <StatusFilterHeader
                  selectedStatuses={selectedStatuses}
                  onStatusChange={handleStatusChange}
                />
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                }}
              >
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredModels.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {models.length === 0 ? 'No model requests found' : 'No models match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredModels.map((model, index) => (
                <TableRow key={model.request_id || index} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {model.request_id || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {model.model || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {model.entity || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Chip 
                      label={model.model_type || 'N/A'}
                      size="small"
                      sx={{ 
                        backgroundColor: (model.model_type || '').toLowerCase() === 'delta' ? '#e8f5e8' : '#fff3e0',
                        color: (model.model_type || '').toLowerCase() === 'delta' ? '#2e7d32' : '#f57c00',
                        fontWeight: 600
                      }}
                    />
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {model.created_by || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <StatusChip status={model.status} />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      <IconButton size="small" onClick={() => handleViewRequest(model)} title="View Details">
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


      {/* Model Registration Dialog */}
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle>
          {isEditMode ? 'Edit Model' : 'Register Model'}
        </DialogTitle>
        <DialogContent>
          {(entities.length === 0 || jobFrequencies.length === 0) && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              {entities.length === 0 && jobFrequencies.length === 0 
                ? "No approved entities or job frequencies available. Please ensure both entities and job frequencies are configured before registering models."
                : entities.length === 0
                ? "No approved entities available. Please create and approve entities first before registering models."
                : "No job frequencies available. Please configure job frequencies before registering models."
              }
            </Alert>
          )}

          <Box sx={{ mt: 2 }}>
            <Grid container spacing={2}>
              {/* Row 1 */}
              <Grid item xs={6}>
                <FormControl fullWidth required error={!!validationErrors.entity} size="small">
                  <InputLabel>Entity</InputLabel>
                  <Select
                    name="entity"
                    value={modelData.entity}
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
                <TextField
                  fullWidth
                  required
                  size="small"
                  name="model"
                  label="Model Name"
                  value={modelData.model}
                  onChange={handleChange}
                  error={!!validationErrors.model}
                  helperText={validationErrors.model}
                />
              </Grid>

              {/* Row 2 */}
              <Grid item xs={6}>
                <FormControl fullWidth required error={!!validationErrors.model_type} size="small">
                  <InputLabel>Model Type</InputLabel>
                  <Select
                    name="model_type"
                    value={modelData.model_type}
                    onChange={handleChange}
                    label="Model Type"
                  >
                    <MenuItem value="RESET">RESET</MenuItem>
                    <MenuItem value="DELTA">DELTA</MenuItem>
                  </Select>
                  {validationErrors.model_type && (
                    <FormHelperText>{validationErrors.model_type}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <FormControl fullWidth required error={!!validationErrors.job_frequency} size="small">
                  <InputLabel>Job Frequency</InputLabel>
                  <Select
                    name="job_frequency"
                    value={modelData.job_frequency}
                    onChange={handleChange}
                    label="Job Frequency"
                  >
                    {jobFrequencies && jobFrequencies.map((freq, index) => (
                      <MenuItem key={freq || index} value={freq}>{freq}</MenuItem>
                    ))}
                  </Select>
                  {validationErrors.job_frequency && (
                    <FormHelperText>{validationErrors.job_frequency}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              {/* Row 3 */}
              <Grid item xs={6}>
                <FormControl fullWidth required error={!!validationErrors['model_config.distance_function']} size="small">
                  <InputLabel>Distance Function</InputLabel>
                  <Select
                    name="model_config.distance_function"
                    value={modelData.model_config?.distance_function || ''}
                    onChange={handleChange}
                    label="Distance Function"
                  >
                    {DISTANCE_FUNCTIONS && DISTANCE_FUNCTIONS.map((func) => (
                      <MenuItem key={func.id} value={func.id}>{func.label}</MenuItem>
                    ))}
                  </Select>
                  {validationErrors['model_config.distance_function'] && (
                    <FormHelperText>{validationErrors['model_config.distance_function']}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  size="small"
                  name="model_config.vector_dimension"
                  label="Vector Dimension"
                  value={modelData.model_config?.vector_dimension || ''}
                  onChange={handleChange}
                  error={!!validationErrors['model_config.vector_dimension']}
                  helperText={validationErrors['model_config.vector_dimension'] || "e.g., 128, 256, 512, 768"}
                  placeholder="768"
                />
              </Grid>

              {/* Row 5 */}
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  type="number"
                  size="small"
                  name="number_of_partitions"
                  label="Partitions"
                  value={24}
                  disabled
                  helperText="Auto-set to 24"
                />
              </Grid>

              {/* Row 6 - Embedding Store Toggle */}
              <Grid item xs={12}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={modelData.embedding_store_enabled}
                      onChange={handleChange}
                      name="embedding_store_enabled"
                    />
                  }
                  label="Enable Embedding Store"
                />
              </Grid>

              {/* Row 7 - Conditional TTL field */}
              {modelData.embedding_store_enabled && (
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    type="number"
                    size="small"
                    name="embedding_store_ttl"
                    label="TTL (seconds)"
                    value={modelData.embedding_store_ttl}
                    onChange={handleChange}
                    error={!!validationErrors.embedding_store_ttl}
                    helperText={(() => {
                      if (validationErrors.embedding_store_ttl) return validationErrors.embedding_store_ttl;
                      const sec = Number(modelData.embedding_store_ttl);
                      if (modelData.embedding_store_ttl != null && modelData.embedding_store_ttl !== '' && Number.isFinite(sec)) return '≈ ' + formatTtlReadable(sec);
                      return undefined;
                    })()}
                  />
                </Grid>
              )}

              {/* Row 8 - Training Data Path */}
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  size="small"
                  name="training_data_path"
                  label="Training Data Path"
                  value={modelData.training_data_path}
                  onChange={handleChange}
                  error={!!validationErrors.training_data_path}
                  helperText={validationErrors.training_data_path}
                  placeholder="gs://bucket/path/to/training-data"
                />
              </Grid>

              {/* Row 9 - Metadata */}
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  multiline
                  rows={4}
                  size="small"
                  name="metadata"
                  label="Metadata (JSON)"
                  value={typeof modelData.metadata === 'string' ? modelData.metadata : JSON.stringify(modelData.metadata || {}, null, 2)}
                  onChange={handleChange}
                  error={!!validationErrors.metadata}
                  helperText={validationErrors.metadata || 'Expected format: {"entity": "string", "key-type": "string", "details": {"catalog_id": {"feature-group": "string", "feature": "string"}}}'}
                  placeholder='{"entity": "catalog", "key-type": "catalog_id", "details": {"catalog_id": {"feature-group": "catalog_features", "feature": "catalog_id"}}}'
                />
              </Grid>
            </Grid>
          </Box>
        </DialogContent>
        <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
          <Button
            variant="outlined"
            onClick={handleClose}
            sx={{
              borderColor: '#522b4a',
              color: '#522b4a',
              '&:hover': {
                borderColor: '#613a5c',
                backgroundColor: 'rgba(82, 43, 74, 0.04)',
              },
            }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={loading}
            sx={{
              backgroundColor: '#522b4a',
              color: 'white',
              '&:hover': {
                backgroundColor: '#613a5c',
              },
              '&:disabled': {
                backgroundColor: '#cccccc',
              },
            }}
          >
            {loading ? 'Processing...' : (isEditMode ? 'Update Model' : 'Submit Request')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Model Details Modal */}
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedModel}
        config={{
          title: 'Model Details',
          icon: ModelTrainingIcon,
          sections: [
            {
              title: 'Model Information',
              icon: InfoIcon,
              layout: 'grid',
              fields: [
                { label: 'Request ID', key: 'request_id', type: 'monospace' },
                { label: 'Status', key: 'status', type: 'status' },
                { label: 'Model Name', key: 'model' },
                { label: 'Entity', key: 'entity' }
              ]
            },
            {
              title: 'Model Configuration',
              icon: SettingsIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              render: (data) => (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Model Type</Typography>
                      <Chip
                        label={data?.model_type || 'N/A'}
                        size="small"
                        sx={{
                          backgroundColor: data?.model_type === 'DELTA' ? '#e8f5e8' : '#fff3e0',
                          color: data?.model_type === 'DELTA' ? '#2e7d32' : '#f57c00',
                          fontWeight: 600,
                          mt: 0.5
                        }}
                      />
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Job Frequency</Typography>
                      <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', p: 1, borderRadius: 0.5, mt: 0.5 }}>
                        {data?.job_frequency || 'N/A'}
                      </Typography>
                    </Box>
                  </Box>
                  <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Vector Dimension</Typography>
                      <Typography variant="body1">{data?.vector_dimension || data?.model_config?.vector_dimension || 'N/A'}</Typography>
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Distance Function</Typography>
                      <Typography variant="body1">{data?.distance_function || data?.model_config?.distance_function || 'N/A'}</Typography>
                    </Box>
                  </Box>
                </Box>
              )
            },
            {
              title: 'Request Metadata',
              icon: PersonIcon,
              backgroundColor: 'rgba(158, 158, 158, 0.02)',
              borderColor: 'rgba(158, 158, 158, 0.1)',
              layout: 'grid',
              fields: [
                { label: 'Created By', key: 'created_by' },
                { label: 'Created At', key: 'created_at', type: 'date' }
              ]
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

export default ModelRegistry;