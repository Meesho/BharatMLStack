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
  Snackbar,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { useAuth } from '../../../Auth/AuthContext';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';

const PromoteMPConfigModal = ({ open, onClose, onSuccess, configData }) => {
  const { user } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  const [formData, setFormData] = useState({
    config_id: '',
    config_value: '',
    selectedHost: '',
    config_mapping: {
      deployable_id: '',
      response_default_values: null
    },
    proposed_model_end_point: []
  });

  const [promoteHosts, setPromoteHosts] = useState([]);
  const [models, setModels] = useState([]);
  const [productionModels, setProductionModels] = useState([]);
  const [latestRequest, setLatestRequest] = useState(null);
  const [showConfirmation, setShowConfirmation] = useState(false);

  useEffect(() => {
    if (configData) {
      const predatorComponents = configData.config_value?.component_config?.predator_components || [];
      const modelsList = predatorComponents.map(component => ({
        component: component.component,
        model_name: component.model_name,
        model_end_point: component.model_end_point
      }));
      setModels(modelsList);
      
      setFormData({
        config_id: configData.config_id || '',
        config_value: JSON.stringify(configData.config_value, null, 2) || '',
        selectedHost: '',
        config_mapping: {
          deployable_id: '',
          response_default_values: null
        },
        proposed_model_end_point: modelsList.map(model => ({
          component: model.component,
          model_name: model.model_name,
          current_model_end_point: model.model_end_point,
          new_deployable_id: ''
        }))
      });
    }
  }, [configData]);

  useEffect(() => {
    // Fetch production hosts with prod credentials when modal opens
    if (open && window.prodCredentials) {
      fetchPromoteHosts(window.prodCredentials.token);
      fetchProductionModels(window.prodCredentials.token);
    }
  }, [open]);

  useEffect(() => {
    // Fetch latest request when configData is available
    if (open && configData?.config_id && user?.token) {
      fetchLatestRequest(configData.config_id);
    }
  }, [open, configData, user?.token]);

  const fetchLatestRequest = async (configId) => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/mp-config-registry/latestRequest/${configId}`,
        {
          headers: {
            Authorization: `Bearer ${user.token}`,
          },
        }
      );

      const data = response.data?.data;
      setLatestRequest(data);
    } catch (error) {
      console.log('Error fetching latest request:', error);
      setLatestRequest(null);
    }
  };

  const fetchProductionModels = async (prodToken) => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/horizon/predator-config-discovery/models`,
        {
          headers: {
            Authorization: `Bearer ${prodToken}`,
          },
        }
      );
      
      const data = response.data?.data || response.data;
      if (Array.isArray(data)) {
        setProductionModels(data);
      } else {
        setProductionModels([]);
      }
    } catch (error) {
      console.log('Error fetching production models:', error);
      setProductionModels([]);
    }
  };

  const getProductionHostForModel = (modelName) => {
    const productionModel = productionModels.find(model => model.model_name === modelName);
    return productionModel ? productionModel.host : '';
  };

  const isModelNotFoundInProduction = (modelName) => {
    const productionModel = productionModels.find(model => model.model_name === modelName);
    return !productionModel;
  };

  const fetchPromoteHosts = async (prodToken) => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=InferFlow`,
        {
          headers: {
            Authorization: `Bearer ${prodToken}`,
          },
        }
      );
      
      const data = response.data?.data || response.data;
      if (Array.isArray(data) && data.length > 0) {
        // Extract unique hosts from deployables
        const hostMap = new Map();
        
        data.forEach(deployable => {
          if (deployable.host && deployable.active) {
            const hostKey = deployable.host;
            if (!hostMap.has(hostKey)) {
              hostMap.set(hostKey, {
                id: deployable.id,
                host: deployable.host,
                name: deployable.name || deployable.host.split('.')[0],
                active: deployable.active
              });
            }
          }
        });
        
        const hosts = Array.from(hostMap.values());
        setPromoteHosts(hosts);
        return true;
      } else {
        setPromoteHosts([]);
        setError('No active deployables found in production');
        return false;
      }
    } catch (error) {
      setError('Failed to fetch production deployables. Please try again.');
      setPromoteHosts([]);
      return false;
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

  const handleModelHostChange = (index, value) => {
    setFormData(prev => {
      const newProposedEndpoints = [...prev.proposed_model_end_point];
      newProposedEndpoints[index] = {
        ...newProposedEndpoints[index],
        new_deployable_id: value
      };
      return {
        ...prev,
        proposed_model_end_point: newProposedEndpoints
      };
    });
  };

  const handleCloseSnackbar = () => {
    setSnackbar(prev => ({ ...prev, open: false }));
  };

  const handleFormChange = (event) => {
    const { name, value } = event.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const validateForm = () => {
    const errors = [];

    // Validate required fields
    if (!formData.config_id.trim()) {
      errors.push('Config ID is required');
    }
    if (!formData.config_value.trim()) {
      errors.push('Config Value is required');
    }
    if (!formData.selectedHost) {
      errors.push('Please select a InferFlow production host for promotion');
    }

    // Validate proposed model endpoints if they exist
    if (formData.proposed_model_end_point && formData.proposed_model_end_point.length > 0) {
      formData.proposed_model_end_point.forEach((endpoint, index) => {
        if (!endpoint.component || !endpoint.component.trim()) {
          errors.push(`Model Endpoint ${index + 1}: Component is required`);
        }
        if (!endpoint.model_name || !endpoint.model_name.trim()) {
          errors.push(`Model Endpoint ${index + 1}: Model Name is required`);
        }
        if (!endpoint.current_model_end_point || !endpoint.current_model_end_point.trim()) {
          errors.push(`Model Endpoint ${index + 1}: Current Model End Point is required`);
        }
      });
    }

    return errors;
  };

  const handleSubmit = () => {
    const validationErrors = validateForm();
    if (validationErrors.length > 0) {
      setError(validationErrors.join('\n'));
      return;
    }

    const prodCredentials = window.prodCredentials;
    if (!prodCredentials) {
      setError('Production credentials not found. Please try again.');
      return;
    }

    // Show confirmation modal
    setShowConfirmation(true);
  };

  const handleConfirmSubmit = async () => {
    setShowConfirmation(false);
    
    const prodCredentials = window.prodCredentials;
    if (!prodCredentials) {
      setError('Production credentials not found. Please try again.');
      return;
    }
    
    try {
      setLoading(true);
      setError('');

      const proposedModelEndpoints = formData.proposed_model_end_point
        .filter(endpoint => endpoint.model_name)
        .map(endpoint => ({
          model_name: endpoint.model_name,
          end_point_id: getProductionHostForModel(endpoint.model_name) || ''
        }));

      const payload = {
        payload: {
          config_id: formData.config_id,
          config_value: JSON.parse(formData.config_value),
          config_mapping: {
            deployable_id: parseInt(formData.selectedHost),
            response_default_values: formData.config_mapping.response_default_values
          },
          latest_request: latestRequest,
          proposed_model_end_point: proposedModelEndpoints
        },
        created_by: prodCredentials.email
      };
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/horizon/mp-config-registry/promote`,
        payload,
        {
          headers: {
            Authorization: `Bearer ${prodCredentials.token}`,
            'Content-Type': 'application/json'
          },
        }
      );

      if (response.data.error) {
        setError(response.data.error || 'Promotion failed');
        return;
      }

      const successMessage = response.data.data?.message || 'Configuration promoted successfully';
      setSnackbar({
        open: true,
        message: successMessage,
        severity: 'success'
      });

      delete window.prodCredentials;
      onClose();
      onSuccess(successMessage);
    } catch (err) {
      setError(err.response?.data?.error || 'An error occurred while promoting the configuration');
      setSnackbar({
        open: true,
        message: err.response?.data?.error || 'An error occurred while promoting the configuration',
        severity: 'error'
      });
    } finally {
      setLoading(false);
    }
  };

  const getSelectedHostDetails = () => {
    return promoteHosts.find(host => host.id === parseInt(formData.selectedHost));
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
            <Typography variant="h6">Promote InferFlow Config</Typography>
          </Box>
          <IconButton onClick={onClose} size="small" sx={{ color: 'white' }}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 1, p: 1 }}>
            {/* Config ID - Read Only */}
            <TextField
              fullWidth
              label="InferFlow Config ID"
              value={formData.config_id}
              disabled
              sx={{ 
                mb: 3,
                '& .MuiInputBase-input.Mui-disabled': {
                  bgcolor: '#f5f5f5',
                  WebkitTextFillColor: '#666'
                }
              }}
            />

            {/* Config Value - Read Only */}
            <TextField
              fullWidth
              label="Config Value"
              value={formData.config_value}
              disabled
              multiline
              rows={8}
              sx={{ 
                mb: 3,
                '& .MuiInputBase-input.Mui-disabled': {
                  bgcolor: '#f5f5f5',
                  WebkitTextFillColor: '#666'
                },
                '& .MuiInputBase-root': {
                  fontFamily: 'monospace',
                  fontSize: '0.875rem'
                }
              }}
            />

            {/* Production Model Host Selection */}
            <FormControl fullWidth sx={{ mb: 3 }}>
              <InputLabel required>InferFlow Production Host</InputLabel>
              <Select
                name="selectedHost"
                value={formData.selectedHost}
                onChange={handleFormChange}
                label="Production Model Host *"
                required
                sx={{ bgcolor: 'white' }}
              >
                {promoteHosts.map((item) => (
                  <MenuItem key={item.id} value={item.id}>
                    <Box>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {item.name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Host: {item.host}
                      </Typography>
                    </Box>
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {/* Model Endpoint Config Section */}
            <Divider sx={{ my: 3 }} />
            <Typography variant="h6" sx={{ color: '#450839', fontWeight: 600, mb: 2 }}>
              Model Endpoint Configuration
            </Typography>
            
            {formData.proposed_model_end_point.map((endpoint, index) => (
              <Box key={index} sx={{ mb: 2, p: 2, border: '1px solid #e0e0e0', borderRadius: 1 }}>
                <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Model Name"
                    value={endpoint.model_name}
                    disabled
                    sx={{
                      '& .MuiInputBase-input.Mui-disabled': {
                        bgcolor: '#f5f5f5',
                        WebkitTextFillColor: '#666'
                      }
                    }}
                  />
                  <TextField
                    fullWidth
                    size="small"
                    label="Predator Production Host"
                    value={getProductionHostForModel(endpoint.model_name)}
                    disabled
                    error={isModelNotFoundInProduction(endpoint.model_name)}
                    sx={{
                      '& .MuiInputBase-input.Mui-disabled': {
                        bgcolor: isModelNotFoundInProduction(endpoint.model_name) ? '#ffebee' : '#f5f5f5',
                        WebkitTextFillColor: '#666'
                      }
                    }}
                  />
                </Box>
                {isModelNotFoundInProduction(endpoint.model_name) && (
                  <Alert severity="error" sx={{ mt: 1, py: 0.5 }}>
                    <Typography variant="body2">
                      Please onboard the model "{endpoint.model_name}" on production first
                    </Typography>
                  </Alert>
                )}
              </Box>
            ))}

            {/* Info Alert */}
            <Alert severity="info" sx={{ mb: 2 }}>
              <Typography variant="body2">
                This configuration will be promoted to the selected production model host with your production credentials. Make sure the model is already onboarded on production.
              </Typography>
            </Alert>

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
            {loading ? 'Promoting...' : 'Promote'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Confirmation Modal */}
      <Dialog 
        open={showConfirmation} 
        onClose={() => setShowConfirmation(false)} 
        maxWidth="sm" 
        fullWidth
      >
        <DialogTitle
          sx={{
            bgcolor: '#450839',
            color: 'white',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
          }}
        >
          <Typography variant="h6">Confirm Promotion</Typography>
          <IconButton onClick={() => setShowConfirmation(false)} size="small" sx={{ color: 'white' }}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ mt: 2 }}>
          <Alert severity="warning" sx={{ mb: 3 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              Please review the following details before confirming the promotion:
            </Typography>
          </Alert>

          {/* Selected Host */}
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: '#450839' }}>
              InferFlow Production Host:
            </Typography>
            {getSelectedHostDetails() ? (
              <Box sx={{ p: 2, bgcolor: '#f5f5f5', borderRadius: 1 }}>
                <Typography variant="body1" sx={{ fontWeight: 500 }}>
                  {getSelectedHostDetails().name}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Host: {getSelectedHostDetails().host}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  ID: {getSelectedHostDetails().id}
                </Typography>
              </Box>
            ) : (
              <Typography variant="body2" color="error">
                No host selected
              </Typography>
            )}
          </Box>

          {/* Model Endpoint Configuration */}
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: '#450839' }}>
              Model Endpoint Configuration:
            </Typography>
            {formData.proposed_model_end_point && formData.proposed_model_end_point.length > 0 ? (
              <Box sx={{ maxHeight: '300px', overflowY: 'auto' }}>
                {formData.proposed_model_end_point.map((endpoint, index) => {
                  const productionHost = getProductionHostForModel(endpoint.model_name);
                  const isNotFound = isModelNotFoundInProduction(endpoint.model_name);
                  
                  return (
                    <Box 
                      key={index} 
                      sx={{ 
                        mb: 1.5, 
                        p: 2, 
                        border: '1px solid #e0e0e0', 
                        borderRadius: 1,
                        bgcolor: isNotFound ? '#ffebee' : '#f5f5f5'
                      }}
                    >
                      <Typography variant="body2" sx={{ fontWeight: 500, mb: 0.5 }}>
                        Model: {endpoint.model_name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Predator Production Host: {productionHost || 'Not found'}
                      </Typography>
                      {isNotFound && (
                        <Alert severity="error" sx={{ mt: 1, py: 0.5 }}>
                          <Typography variant="caption">
                            Model not found in production
                          </Typography>
                        </Alert>
                      )}
                    </Box>
                  );
                })}
              </Box>
            ) : (
              <Typography variant="body2" color="text.secondary">
                No model endpoints configured
              </Typography>
            )}
          </Box>
        </DialogContent>
        <Divider />
        <DialogActions sx={{ p: 2, bgcolor: '#fafafa' }}>
          <Button 
            onClick={() => setShowConfirmation(false)}
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
            onClick={handleConfirmSubmit}
            variant="contained"
            sx={{ 
              bgcolor: '#450839',
              '&:hover': {
                bgcolor: '#5a0a4a'
              }
            }}
          >
            Confirm & Promote
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseSnackbar} severity={snackbar.severity} sx={{ width: '100%' }}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </>
  );
};

export default PromoteMPConfigModal; 