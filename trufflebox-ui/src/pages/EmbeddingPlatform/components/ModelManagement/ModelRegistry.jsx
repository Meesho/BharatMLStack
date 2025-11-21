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
  FormHelperText,
  Chip,
  IconButton,
  Tooltip,
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
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
import FilterListIcon from '@mui/icons-material/FilterList';
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
import { 
  BUSINESS_RULES, 
  DISTANCE_FUNCTIONS 
} from '../../../../services/embeddingPlatform/constants';

const ModelRegistry = () => {
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['active', 'inactive']);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedModel, setSelectedModel] = useState(null);
  const [modelData, setModelData] = useState(getDefaultFormValues('model'));
  const [entities, setEntities] = useState([]);
  const [jobFrequencies, setJobFrequencies] = useState([]);
  const [isEditMode, setIsEditMode] = useState(false);
  const { user } = useAuth();
  const [notification, setNotification] = useState({ open: false, message: "", severity: "success" });
  const [modalMessage, setModalMessage] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

  const statusOptions = [
    { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
  ];

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const [modelsResponse, entitiesResponse, jobFrequenciesResponse] = await Promise.all([
        embeddingPlatformAPI.getModels(),
        embeddingPlatformAPI.getEntities(),
        embeddingPlatformAPI.getJobFrequencies()
      ]);

      if (modelsResponse.models) {
        setModels(modelsResponse.models);
      } else {
        setModels([]);
      }

      if (entitiesResponse.entities) {
        const availableEntities = entitiesResponse.entities?.map(entity => ({
          name: entity.name,
          store_id: entity.store_id,
          label: `${entity.name} (Store ${entity.store_id})`
        })) || [];
        setEntities(availableEntities);
      }

      if (jobFrequenciesResponse.job_frequencies) {
        setJobFrequencies(jobFrequenciesResponse.job_frequencies);
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

  const filteredModels = useMemo(() => {
    let filtered = [...models];
    
    // Status filtering
    // To-do: check status thing, rn its aactive
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(model => 
        selectedStatuses.includes(model.status)
      );
    }

    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(model => {
        return (
          String(model.request_id || '').toLowerCase().includes(searchLower) ||
          String(model.model || '').toLowerCase().includes(searchLower) ||
          String(model.entity || '').toLowerCase().includes(searchLower) ||
          String(model.model_type || '').toLowerCase().includes(searchLower) ||
          String(model.created_by || '').toLowerCase().includes(searchLower) ||
          (model.status && model.status.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at || 0) - new Date(a.created_at || 0);
    });
  }, [models, searchQuery, selectedStatuses]);

  const getStatusChip = (status) => {
    const statusUpper = (status || 'APPROVED').toUpperCase();
    let bgcolor = '#E7F6E7';
    let textColor = '#2E7D32';

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
      default:
        bgcolor = '#E7F6E7';
        textColor = '#2E7D32';
    }

    return (
      <Chip
        label={statusUpper}
        size="small"
        sx={{
          backgroundColor: bgcolor,
          color: textColor,
          fontWeight: 'bold',
          minWidth: '80px',
        }}
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
          anchorOrigin={{
            vertical: 'bottom',
            horizontal: 'left',
          }}
          transformOrigin={{
            vertical: 'top',
            horizontal: 'left',
          }}
          PaperProps={{
            sx: {
              width: 200,
              maxHeight: 300,
              overflow: 'auto'
            }
          }}
        >
          <Box sx={{ p: 1 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
              <Button 
                size="small" 
                onClick={handleSelectAll}
                sx={{ textTransform: 'none', fontSize: '0.75rem', minWidth: 'auto' }}
              >
                All
              </Button>
              <Button 
                size="small" 
                onClick={handleClearAll}
                sx={{ textTransform: 'none', fontSize: '0.75rem', minWidth: 'auto' }}
              >
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
      const payload = {
        requestor: user?.email || 'user@example.com',
        reason: isEditMode ? 'Updating model configuration' : 'Registering new ML model',
        payload: modelData
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


  const showNotification = (message, severity) => {
    setNotification({
      open: true,
      message,
      severity
    });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({
      ...prev,
      open: false
    }));
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
                <StatusColumnHeader />
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
                    {getStatusChip(model.status)}
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
                    {jobFrequencies && jobFrequencies.map((freq) => (
                      <MenuItem key={freq.id} value={freq.frequency}>{freq.frequency}</MenuItem>
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

              {/* Row 4 */}
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  size="small"
                  name="mq_id"
                  label="MQ ID"
                  value={modelData.mq_id}
                  onChange={handleChange}
                  error={!!validationErrors.mq_id}
                  helperText={validationErrors.mq_id || (isEditMode ? "Cannot be changed" : "")}
                  disabled={isEditMode}
                />
              </Grid>

              <Grid item xs={6}>
                <TextField
                  fullWidth
                  required
                  size="small"
                  name="topic_name"
                  label="Topic Name"
                  value={modelData.topic_name}
                  onChange={handleChange}
                  error={!!validationErrors.topic_name}
                  helperText={validationErrors.topic_name || (isEditMode ? "Cannot be changed" : "")}
                  disabled={isEditMode}
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

              <Grid item xs={6}>
                <TextField
                  fullWidth
                  type="number"
                  size="small"
                  name="failure_producer_mq_id"
                  label="Failure Producer MQ ID"
                  value={modelData.failure_producer_mq_id}
                  onChange={handleChange}
                  error={!!validationErrors.failure_producer_mq_id}
                  helperText={validationErrors.failure_producer_mq_id}
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
                    helperText={validationErrors.embedding_store_ttl}
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
                  rows={2}
                  size="small"
                  name="metadata"
                  label="Metadata (JSON)"
                  value={modelData.metadata}
                  onChange={handleChange}
                  error={!!validationErrors.metadata}
                  helperText={validationErrors.metadata}
                  placeholder='{"key": "value"}'
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
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ModelTrainingIcon sx={{ color: '#522b4a' }} />
            Model Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedModel && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 2 }}>
              {/* Model Information Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                  Model Information
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Request ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                      {selectedModel.request_id || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Status
                    </Typography>
                    <Box sx={{ mt: 0.5 }}>
                      {getStatusChip(selectedModel.status)}
                    </Box>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Model Name
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <SmartToyIcon fontSize="small" sx={{ color: '#ff9800' }} />
                      {selectedModel.model || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Entity
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <StorageIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      {selectedModel.entity || 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Box>

              {/* Model Configuration Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                  Model Configuration
                </Typography>
                
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                        Model Type
                      </Typography>
                      <Chip 
                        label={selectedModel.model_type || 'N/A'}
                        size="small"
                        sx={{ 
                          backgroundColor: selectedModel.model_type === 'DELTA' ? '#e8f5e8' : '#fff3e0',
                          color: selectedModel.model_type === 'DELTA' ? '#2e7d32' : '#f57c00',
                          fontWeight: 600,
                          mt: 0.5
                        }}
                      />
                    </Box>
                    
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                        Job Frequency
                      </Typography>
                      <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', p: 1, borderRadius: 0.5, mt: 0.5 }}>
                        {selectedModel.job_frequency || 'N/A'}
                      </Typography>
                    </Box>
                  </Box>

                  <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                        Vector Dimension
                      </Typography>
                      <Typography variant="body1">
                        {selectedModel.vector_dimension || selectedModel.model_config?.vector_dimension || 'N/A'}
                      </Typography>
                    </Box>
                    
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                        Distance Function
                      </Typography>
                      <Typography variant="body1">
                        {selectedModel.distance_function || selectedModel.model_config?.distance_function || 'N/A'}
                      </Typography>
                    </Box>
                  </Box>
                </Box>
              </Box>

              {/* Metadata Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(158, 158, 158, 0.02)', borderRadius: 1, border: '1px solid rgba(158, 158, 158, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <PersonIcon fontSize="small" sx={{ color: '#757575' }} />
                  Request Metadata
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created By
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <PersonIcon fontSize="small" sx={{ color: '#522b4a' }} />
                      {selectedModel.created_by || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created At
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      {selectedModel.created_at ? new Date(selectedModel.created_at).toLocaleString() : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2, backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button 
            onClick={handleCloseViewModal}
            sx={{ 
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
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

export default ModelRegistry;