import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  IconButton,
  Typography,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Alert,
  Divider,
  Tooltip,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import ErrorIcon from '@mui/icons-material/Error';
import { useAuth } from '../../../Auth/AuthContext';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';

const ScaleUpInferflowConfigModal = ({ open, onClose, onSuccess, configData }) => {
  const { user } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [responseModal, setResponseModal] = useState({
    open: false,
    type: '', // 'success' or 'error'
    message: ''
  });

  // Form Data States
  const [formData, setFormData] = useState({
    config_id: '',
    config_value: '',
    config_mapping: {
      deployable_id: '',
      response_default_values: null
    },
    proposed_model_endpoints: [],
    logging_ttl: '',
    logging_perc: ''
  });

  // Dropdown Options States
  const [mpHosts, setMpHosts] = useState([]);
  const [predatorModels, setPredatorModels] = useState([]);
  const [predatorHosts, setPredatorHosts] = useState([]);
  const [components, setComponents] = useState([]);
  const [loggingTtlOptions, setLoggingTtlOptions] = useState([30, 60, 90]); // Fallback values

  // Initialize form data when modal opens
  useEffect(() => {
    if (configData) {
      const predatorComponents = configData.config_value?.component_config?.predator_components || [];
      setComponents(predatorComponents);

      // Extract logging_perc from response_config
      const loggingPerc = configData.config_value?.response_config?.logging_perc ?? '';

      setFormData({
        config_id: configData.config_id || '',
        config_value: JSON.stringify(configData.config_value, null, 2) || '',
        config_mapping: {
          deployable_id: '',
          response_default_values: null
        },
        proposed_model_endpoints: predatorComponents.map(c => ({
          component: c.component,
          model_name: c.model_name,
          model_id: '',
          host: ''
        })),
        logging_ttl: '',
        logging_perc: loggingPerc !== '' && loggingPerc !== null && loggingPerc !== undefined ? String(loggingPerc) : ''
      });
    }
  }, [configData]);

  // Fetch MP Hosts on component mount
  useEffect(() => {
    fetchMPHosts();
    fetchPredatorModels();
    fetchPredatorHosts();
    fetchLoggingTtlOptions();
  }, []);

  // API Calls
  const fetchMPHosts = async () => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=InferFlow`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      const data = response.data?.data || response.data;
      setMpHosts(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Error fetching InferFlow hosts:', error);
      setMpHosts([]);
    }
  };

  const fetchPredatorModels = async () => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/models`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      setPredatorModels(Array.isArray(response.data.data) ? response.data.data : []);
    } catch (error) {
      console.error('Error fetching predator models:', error);
      setPredatorModels([]);
    }
  };

  const fetchPredatorHosts = async () => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=PREDATOR`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      setPredatorHosts(Array.isArray(response.data) ? response.data : []);
    } catch (error) {
      console.error('Error fetching predator hosts:', error);
      setPredatorHosts([]);
    }
  };

  const fetchLoggingTtlOptions = async () => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/inferflow-config-registry/logging-ttl`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      const options = Array.isArray(response.data.data) && response.data.data.length > 0 
        ? response.data.data 
        : [30, 60, 90]; // Fallback to default values
      setLoggingTtlOptions(options);
    } catch (error) {
      console.error('Error fetching logging TTL options:', error);
    }
  };

  // Handle form changes
  const handleConfigIdChange = (event) => {
    setFormData(prev => ({
      ...prev,
      config_id: event.target.value.trim()
    }));
  };

  const handleLoggingTtlChange = (event) => {
    setFormData(prev => ({
      ...prev,
      logging_ttl: event.target.value
    }));
  };

  const handleLoggingPercChange = (event) => {
    const value = event.target.value;
    // Allow empty string or valid number between 0-100
    if (value === '' || (Number(value) >= 0 && Number(value) <= 100)) {
      setFormData(prev => ({
        ...prev,
        logging_perc: value
      }));
    }
  };

  const handleConfigMappingChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      config_mapping: {
        ...prev.config_mapping,
        [field]: value
      }
    }));
  };

  const handleEndpointChange = (index, field, value) => {
    setFormData(prev => {
      const newEndpoints = [...prev.proposed_model_endpoints];
      newEndpoints[index] = { ...newEndpoints[index], [field]: value };

      if (field === 'model_id') {
        const selectedModel = predatorModels.find(model => model.id === parseInt(value));
        if (selectedModel && selectedModel.host) {
          newEndpoints[index].host = selectedModel.host;
        }
      }

      return { ...prev, proposed_model_endpoints: newEndpoints };
    });
  };

  const handleResponseModalClose = () => {
    setResponseModal(prev => ({ ...prev, open: false }));
    onClose();
  };

  const validateForm = () => {
    const errors = [];

    // Validate required basic fields
    if (!formData.config_id.trim()) {
      errors.push('Inferpipe ID is required');
    }
    if (!formData.config_value.trim()) {
      errors.push('Inferpipe Value is required');
    }

    // Validate config mapping
    if (!formData.config_mapping.deployable_id) {
      errors.push('InferFlow Host selection is required');
    }

    // Validate logging TTL
    if (!formData.logging_ttl) {
      errors.push('Logging TTL selection is required');
    }

    // Validate logging percentage
    if (formData.logging_perc === '' || formData.logging_perc === null || formData.logging_perc === undefined) {
      errors.push('Logging Percentage is required');
    } else {
      const loggingPerc = Number(formData.logging_perc);
      if (isNaN(loggingPerc) || loggingPerc < 0 || loggingPerc > 100) {
        errors.push('Logging Percentage must be a number between 0 and 100');
      }
    }

    // Validate proposed model endpoints
    if (!formData.proposed_model_endpoints || formData.proposed_model_endpoints.length === 0) {
      errors.push('At least one model endpoint is required');
    } else {
      formData.proposed_model_endpoints.forEach((endpoint, index) => {
        if (!endpoint.model_id) {
          errors.push(`Model Endpoint ${index + 1}: Model selection is required`);
        }
        if (!endpoint.host) {
          errors.push(`Model Endpoint ${index + 1}: Host is required`);
        }
      });
    }

    // Validate config ID cannot be the same as the original config ID
    if (formData.config_id.trim() === configData.config_id.trim()) {
      errors.push('Inferpipe ID cannot be the same as the original inferpipe ID');
    }
    
    return errors;
  };

  // Form Submission
  const handleSubmit = async () => {
    try {
      setLoading(true);
      setError('');

      const validationErrors = validateForm();
      if (validationErrors.length > 0) {
        setError(validationErrors.join('\n'));
        setLoading(false);
        return;
      }

      const transformedEndpoints = formData.proposed_model_endpoints.map((endpoint, index) => {
        const currentComponent = components[index];
        const newModel = predatorModels.find(m => m.id === endpoint.model_id);
        
        return {
          current_model_name: currentComponent.model_name,
          new_model_name: newModel ? newModel.model_name : '',
          end_point_id: endpoint.host || '',
        };
      });

      const payload = {
        payload: {
          config_id: formData.config_id,
          config_value: JSON.parse(formData.config_value),
          config_mapping: formData.config_mapping,
          proposed_model_endpoints: transformedEndpoints,
          logging_ttl: formData.logging_ttl,
          logging_perc: Number(formData.logging_perc)
        },
        created_by: user.email
      };

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/inferflow-config-registry/scale-up`,
        payload,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
            'Content-Type': 'application/json'
          },
        }
      );

      if (response.data.error) {
        console.log(response.data.error);
      }
      
      const successMessage = typeof response.data.data === 'string' 
        ? response.data.data 
        : response.data.data?.message || 'Inferpipe scaled up successfully';
      
      onSuccess(successMessage);
      onClose();
    } catch (err) {
      setError(err.message || 'An error occurred while scaling up the inferpipe');
      setResponseModal({
        open: true,
        type: 'error',
        message: err.message || 'An error occurred while scaling up the inferpipe'
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
        <DialogTitle
          sx={{
            bgcolor: '#450839',
            color: 'white',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            mb: 2
          }}
        >
          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="h6">Scale Up Inferpipe</Typography>
          </Box>
          <IconButton onClick={onClose} size="small" sx={{ color: 'white' }}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 1, p: 1 }}>
            <Box sx={{ mb: 3, borderRadius: 1 }}>
              <TextField
                fullWidth
                label="Inferpipe ID"
                value={formData.config_id}
                onChange={handleConfigIdChange}
                sx={{ mb: 2 }}
              />

              <TextField
                fullWidth
                label="Inferpipe Value"
                value={formData.config_value}
                disabled
                multiline
                rows={8}
                sx={{ 
                  mb: 2,
                  '& .MuiInputBase-input.Mui-disabled': {
                    bgcolor: 'white',
                    WebkitTextFillColor: '#000000',
                    fontFamily: 'monospace'
                  }
                }}
              />

              {/* MP Host */}
              <FormControl fullWidth sx={{ mb: 2 }}>
                <InputLabel required>InferFlow Host</InputLabel>
                <Select
                  value={formData.config_mapping.deployable_id}
                  onChange={(e) => handleConfigMappingChange('deployable_id', e.target.value)}
                  label="InferFlow Host *"
                  required
                  sx={{ bgcolor: 'white' }}
                >
                  {mpHosts.filter(host => host?.host !== configData?.host).map((host) => (
                    <MenuItem key={host.id} value={host.id}>
                      {`${host.name} (${host.host})`}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              {/* Logging TTL */}
              <FormControl fullWidth sx={{ mb: 2 }}>
                <InputLabel required>Logging TTL (Days)</InputLabel>
                <Select
                  value={formData.logging_ttl}
                  onChange={handleLoggingTtlChange}
                  label="Logging TTL (Days) *"
                  required
                  sx={{ bgcolor: 'white' }}
                >
                  {loggingTtlOptions.map((ttl) => (
                    <MenuItem key={ttl} value={ttl}>
                      {ttl} days
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              {/* Logging Percentage */}
              <TextField
                fullWidth
                label="Logging Percentage"
                type="number"
                value={formData.logging_perc}
                onChange={handleLoggingPercChange}
                required
                inputProps={{ 
                  min: 0, 
                  max: 100,
                  step: 0.1
                }}
                helperText="Enter a value between 0 and 100"
                sx={{ mb: 2 }}
                error={formData.logging_perc !== '' && (Number(formData.logging_perc) < 0 || Number(formData.logging_perc) > 100)}
              />
            </Box>

            {components.length > 0 && (
              <Box sx={{ mb: 2, mt: 2, p: 2, border: '1px solid #ccc', borderRadius: 1 }}>
                <Typography variant="h6" sx={{ mb: 2, color: '#450839' }}>
                  Model Endpoint Configuration
                </Typography>
                {components.map((component, index) => (
                  <Box key={index} sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
                    <TextField
                      label="Model Name"
                      value={component.model_name}
                      disabled
                      sx={{ 
                        flex: 1,
                        '& .MuiInputBase-input.Mui-disabled': {
                          bgcolor: '#f5f5f5',
                          WebkitTextFillColor: '#666'
                        }
                      }}
                    />
                    <FormControl sx={{ flex: 1 }}>
                      <InputLabel required>Model</InputLabel>
                      <Select
                        value={formData.proposed_model_endpoints[index]?.model_id || ''}
                        onChange={(e) => handleEndpointChange(index, 'model_id', e.target.value)}
                        label="Model *"
                        required
                      >
                        {predatorModels.map((model) => (
                          <MenuItem key={model.id} value={model.id}>
                            <Tooltip title={model.model_name} placement="left" arrow>
                              <Typography noWrap>{model.model_name}</Typography>
                            </Tooltip>
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <TextField
                      label="Host"
                      value={formData.proposed_model_endpoints[index]?.host || ''}
                      disabled
                      sx={{ 
                        flex: 1,
                        '& .MuiInputBase-input.Mui-disabled': {
                          bgcolor: '#f5f5f5',
                          WebkitTextFillColor: '#666'
                        }
                      }}
                    />
                  </Box>
                ))}
              </Box>
            )}

            {error && (
              <Alert severity="error" sx={{ mb: 2 }}>
                <Box component="pre" sx={{ whiteSpace: 'pre-line', fontFamily: 'inherit', margin: 0 }}>
                  {error}
                </Box>
              </Alert>
            )}
          </Box>
        </DialogContent>
        <Divider />
        <DialogActions sx={{ p: 2 }}>
          <Button 
            onClick={onClose}
            variant="outlined"
            sx={{ 
              mr: 1,
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                borderColor: '#450839',
                bgcolor: 'rgba(69, 8, 57, 0.04)'
              }
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            disabled={loading}
            sx={{ 
              bgcolor: '#450839',
              '&:hover': {
                bgcolor: '#5a0a4a'
              }
            }}
          >
            {loading ? 'Scaling Up...' : 'Scale Up'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Error Modal */}
      <Dialog open={responseModal.open && responseModal.type === 'error'} onClose={handleResponseModalClose}>
        <DialogTitle>
          <Box display="flex" alignItems="center" gap={1}>
            <ErrorIcon color="error" />
            Error
          </Box>
        </DialogTitle>
        <DialogContent>
          <Typography>{responseModal.message}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleResponseModalClose}>Okay, Try Again</Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default ScaleUpInferflowConfigModal; 