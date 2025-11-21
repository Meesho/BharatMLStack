import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  CircularProgress,
  Typography,
  Grid,
  Card,
  CardContent,
  IconButton,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormGroup,
  Checkbox,
  Chip,
  Autocomplete,
  Alert,
  Snackbar,
} from '@mui/material';
import { Add, Delete, CloudUpload } from '@mui/icons-material';
import { useAuth } from '../../../../Auth/AuthContext';
import * as URL_CONSTANTS from '../../../../../config';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../../constants/permissions';

const UploadModelModal = ({ 
  open, 
  onClose, 
  onUploadComplete 
}) => {
  const { user, hasPermission } = useAuth();
  const service = SERVICES.PREDATOR;
  const screenType = SCREEN_TYPES.PREDATOR.MODEL;
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // Feature type options
  const [featureTypes, setFeatureTypes] = useState([]);
  const [featureTypesLoading, setFeatureTypesLoading] = useState(false);
  
  const [formData, setFormData] = useState({
    gcs_path: '',
    metadata: {
      model_name: '',
      inputs: []
    }
  });
  const [errors, setErrors] = useState({});
  // Determine if user can upload based on permissions
  const canUpload = hasPermission(service, screenType, ACTIONS.UPLOAD);
  const canUploadPartial = hasPermission(service, screenType, ACTIONS.UPLOAD_PARTIAL);
  
  // Set isPartial based on UPLOAD_PARTIAL permission
  const [isPartial, setIsPartial] = useState(canUploadPartial);
  const [sourceModels, setSourceModels] = useState([]);
  const [sourceModelsLoading, setSourceModelsLoading] = useState(false);
  const [submitError, setSubmitError] = useState('');
  // Store raw text for each input's feature list to avoid parsing on every keystroke
  const [featureListTexts, setFeatureListTexts] = useState({});
  // Toast state
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success'
  });


  useEffect(() => {
    if (open) {
      // Initialize form data
      setFormData({
        gcs_path: '',
        metadata: {
          model_name: '',
          inputs: []
        }
      });
      setErrors({});
      setSubmitError('');
      setFeatureListTexts({});
      // Set isPartial based on UPLOAD_PARTIAL permission
      setIsPartial(canUploadPartial);
    }
  }, [open, canUploadPartial]);

  useEffect(() => {
    if (open) {
      fetchSourceModels();
    }
  }, [open]);

  useEffect(() => {
    if (open) {
      fetchFeatureTypes();
    }
  }, [open]);

  const fetchFeatureTypes = async () => {
    setFeatureTypesLoading(true);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/feature-types`,
        {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
          },
        }
      );

      if (response.ok) {
        const result = await response.json();
        if (result.data && Array.isArray(result.data.feature_types)) {
          setFeatureTypes(result.data.feature_types);
        } else if (result.data && Array.isArray(result.data)) {
          setFeatureTypes(result.data);
        } else {
          // Fallback to default values if API response is unexpected
          setFeatureTypes(['DEFAULT_FEATURE', 'ONLINE_FEATURE', 'OFFLINE_FEATURE', 'HYBRID_FEATURE']);
        }
      } else {
        // Fallback to default values if API call fails
        setFeatureTypes(['DEFAULT_FEATURE', 'ONLINE_FEATURE', 'OFFLINE_FEATURE', 'HYBRID_FEATURE']);
      }
    } catch (error) {
      console.error('Error fetching feature types:', error);
      // Fallback to default values on error
      setFeatureTypes(['DEFAULT_FEATURE', 'ONLINE_FEATURE', 'OFFLINE_FEATURE', 'HYBRID_FEATURE']);
    } finally {
      setFeatureTypesLoading(false);
    }
  };

  const fetchSourceModels = async () => {
    setSourceModelsLoading(true);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/source-models`,
        {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
          },
        }
      );

      if (response.ok) {
        const result = await response.json();
        if (result.data && result.data.folders && Array.isArray(result.data.folders)) {
          setSourceModels(result.data.folders);
        }
      }
    } catch (error) {
      console.error('Error fetching source models:', error);
    } finally {
      setSourceModelsLoading(false);
    }
  };

  const handleInputChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      [field]: typeof value === 'string' ? value.trim() : value
    }));
    
    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({
        ...prev,
        [field]: ''
      }));
    }
  };

  const handleMetadataChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      metadata: {
        ...prev.metadata,
        [field]: typeof value === 'string' ? value.trim() : value
      }
    }));
    
    // Clear error when user starts typing
    if (errors[`metadata.${field}`]) {
      setErrors(prev => ({
        ...prev,
        [`metadata.${field}`]: ''
      }));
    }
  };

  const handleInputOutputChange = (type, index, field, value) => {
    setFormData(prev => ({
      ...prev,
      metadata: {
        ...prev.metadata,
        [type]: prev.metadata[type].map((item, i) => {
          if (i === index) {
            // Create a new object and ensure arrays are new references
            const trimmedValue = typeof value === 'string' ? value.trim() : value;
            const updatedItem = { ...item, [field]: trimmedValue };
            // If field is 'features' and value is an array, ensure it's a new array reference
            if (field === 'features' && Array.isArray(value)) {
              updatedItem.features = [...value];
            }
            return updatedItem;
          }
          // Ensure we're not sharing references for other items
          return { ...item };
        })
      }
    }));
    
    // Clear error when user starts typing
    if (errors[`${type}_${index}_${field}`]) {
      setErrors(prev => ({
        ...prev,
        [`${type}_${index}_${field}`]: ''
      }));
    }
  };

  const addInput = () => {
    const newInput = {
      name: '',
      features: []
    };
    
    setFormData(prev => ({
      ...prev,
      metadata: {
        ...prev.metadata,
        inputs: [...prev.metadata.inputs, newInput]
      }
    }));
  };

  const removeInput = (index) => {
    setFormData(prev => ({
      ...prev,
      metadata: {
        ...prev.metadata,
        inputs: prev.metadata.inputs.filter((_, i) => i !== index)
      }
    }));
  };



  const parseFeatureList = (text, inputIndex) => {
    if (!text.trim()) {
      handleInputOutputChange('inputs', inputIndex, 'features', []);
      return;
    }

    try {
      // Parse the comma-separated list
      const features = text.split(',').map(feature => {
        const trimmed = feature.trim().replace(/^["']|["']$/g, ''); // Remove quotes
        const pipeIndex = trimmed.indexOf('|');
        if (pipeIndex === -1) {
            return {
                left: '',
                right: trimmed,
                full: trimmed
            };
        }
        
        return {
          left: trimmed.substring(0, pipeIndex).trim(),
          right: trimmed.substring(pipeIndex + 1).trim(), // Everything after the first pipe
          full: trimmed
        };
      }).filter(feature => feature && (feature.left || feature.right));

      // Update the specific input's features with parsed features
      const featureStrings = features.map(f => f.full);
      handleInputOutputChange('inputs', inputIndex, 'features', featureStrings);
    } catch (error) {
      console.error('Error parsing feature list:', error);
      handleInputOutputChange('inputs', inputIndex, 'features', []);
    }
  };

  const handleFeatureListChange = (value, inputIndex) => {
    // Store the raw text for smooth typing experience
    setFeatureListTexts(prev => ({
      ...prev,
      [inputIndex]: value
    }));
    // Parse in real-time so parsed features display updates immediately
    parseFeatureList(value, inputIndex);
  };

  // Helper function to convert features array to display text
  const getFeatureListText = (features) => {
    if (!features || features.length === 0) {
      return '';
    }
    return features.map(f => `"${f}"`).join(', ');
  };

  // Helper function to parse features array into structured format for display
  const getParsedFeatures = (features) => {
    if (!features || features.length === 0) {
      return [];
    }
    return features.map(f => {
      const pipeIndex = f.indexOf('|');
      if (pipeIndex === -1) {
        return {
          left: '',
          right: f,
          full: f
        };
      }
      return {
        left: f.substring(0, pipeIndex).trim(),
        right: f.substring(pipeIndex + 1).trim(),
        full: f
      };
    });
  };

  const handleFeatureEdit = (inputIndex, featureIndex, field, value) => {
    const currentFeatures = formData.metadata.inputs[inputIndex]?.features || [];
    const parsedFeatures = getParsedFeatures(currentFeatures);
    
    parsedFeatures[featureIndex] = {
      ...parsedFeatures[featureIndex],
      [field]: value,
      full: field === 'left' ? `${value}|${parsedFeatures[featureIndex].right}` : `${parsedFeatures[featureIndex].left}|${value}`
    };
    
    const featureStrings = parsedFeatures.map(f => f.full);
    handleInputOutputChange('inputs', inputIndex, 'features', featureStrings);
    // Update featureListTexts to reflect the change
    setFeatureListTexts(prev => ({
      ...prev,
      [inputIndex]: getFeatureListText(featureStrings)
    }));
  };

  const addFeature = (inputIndex) => {
    const currentFeatures = formData.metadata.inputs[inputIndex]?.features || [];
    const updatedFeatures = [...currentFeatures, '|'];
    handleInputOutputChange('inputs', inputIndex, 'features', updatedFeatures);
    // Update featureListTexts to reflect the change
    setFeatureListTexts(prev => ({
      ...prev,
      [inputIndex]: getFeatureListText(updatedFeatures)
    }));
  };

  const removeFeature = (inputIndex, featureIndex) => {
    const currentFeatures = formData.metadata.inputs[inputIndex]?.features || [];
    // Create a completely new array to avoid reference sharing
    const updatedFeatures = Array.from(currentFeatures).filter((_, i) => i !== featureIndex);
    handleInputOutputChange('inputs', inputIndex, 'features', updatedFeatures);
    // Update featureListTexts to reflect the change
    setFeatureListTexts(prev => ({
      ...prev,
      [inputIndex]: getFeatureListText(updatedFeatures)
    }));
  };

  const handleModelNameChange = (modelName) => {
    handleMetadataChange('model_name', modelName);
    
    if (modelName) {
      const selectedModel = sourceModels.find(model => model.name === modelName);
      if (selectedModel && selectedModel.metadata) {
        // Auto-populate metadata from selected model
        setFormData(prev => ({
          ...prev,
          metadata: {
            ...prev.metadata,
            model_name: modelName,
            inputs: selectedModel.metadata.inputs || []
          }
        }));
      }
    }
  };

  const validateForm = () => {
    const newErrors = {};

    // Validate GCS path
    if (!formData.gcs_path.trim()) {
      newErrors.gcs_path = 'GCS path is required';
    } else if (!formData.gcs_path.startsWith('gs://')) {
      newErrors.gcs_path = 'GCS path must start with gs://';
    }

    // Validate model name
    if (!formData.metadata.model_name.trim()) {
      newErrors['metadata.model_name'] = 'Model name is required';
    } else {
      // In create mode, check if model name already exists
      const modelExists = sourceModels.some(
        model => model.name.toLowerCase() === formData.metadata.model_name.trim().toLowerCase()
      );
      if (modelExists) {
        newErrors['metadata.model_name'] = 'A model with this name already exists. Please choose a different name.';
      }
    }

    // Validate inputs (optional, but if provided, name is required)
    formData.metadata.inputs.forEach((input, index) => {
      if (input.name && !input.name.trim()) {
        newErrors[`input_${index}_name`] = 'Input name cannot be empty';
      }
    });

    setErrors(newErrors);
    
    // Show validation errors in toast if there are any
    if (Object.keys(newErrors).length > 0) {
      const errorMessages = Object.values(newErrors).filter(msg => msg);
      setSnackbar({
        open: true,
        message: `Please fix the following errors: ${errorMessages.join(', ')}`,
        severity: 'error'
      });
    }
    
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    setSubmitError(''); // Clear previous errors

    try {
      const payload = {
        request_type: 'create',
        models: [{
          gcs_path: formData.gcs_path,
          metadata: formData.metadata
        }]
      };

      const queryParams = new URLSearchParams();
      if (isPartial) {
        queryParams.append('is_partial', 'true');
      } else {
        queryParams.append('is_partial', 'false');
      }

      const url = `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/upload-model-folder${queryParams.toString() ? '?' + queryParams.toString() : ''}`;

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
        body: JSON.stringify(payload),
      });

      const result = await response.json();

      if (result.error) {
        const errorMessage = result.error.message || result.error || 'Upload failed';
        setSubmitError(errorMessage);
        setSnackbar({
          open: true,
          message: errorMessage,
          severity: 'error'
        });
        return;
      }

      if (!response.ok) {
        const errorMsg = `HTTP ${response.status}: ${response.statusText}`;
        setSubmitError(errorMsg);
        setSnackbar({
          open: true,
          message: errorMsg,
          severity: 'error'
        });
        return;
      }

      const successMessage = result.data?.message || 'Model uploaded successfully';
      setSnackbar({
        open: true,
        message: successMessage,
        severity: 'success'
      });
      
      // Call onUploadComplete and close after a short delay to show the toast
      setTimeout(() => {
        onUploadComplete(successMessage, 'success');
        onClose();
      }, 1000);

    } catch (error) {
      console.error('Error during upload operation:', error);
      const errorMessage = `Upload failed: ${error.message}`;
      setSubmitError(errorMessage);
      setSnackbar({
        open: true,
        message: errorMessage,
        severity: 'error'
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSnackbarClose = (event, reason) => {
    if (reason === 'clickaway') {
      return;
    }
    setSnackbar(prev => ({ ...prev, open: false }));
  };

  const handleClose = () => {
    setFormData({
      gcs_path: '',
      metadata: {
        model_name: '',
        inputs: []
      }
    });
      setErrors({});
      setSubmitError('');
      setFeatureListTexts({});
      // Set isPartial based on UPLOAD_PARTIAL permission
      setIsPartial(canUploadPartial);
      onClose();
  };

  // Check if user has permission to use this modal
  if (!canUpload) {
    return (
      <Dialog
        open={open}
        onClose={handleClose}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CloudUpload sx={{ color: '#450839' }} />
            <Typography variant="h6">
              Access Denied
            </Typography>
          </Box>
        </DialogTitle>
        <DialogContent>
          <Alert severity="error">
            You do not have permission to upload models. Please contact your administrator.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} variant="contained">
            Close
          </Button>
        </DialogActions>
      </Dialog>
    );
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="lg"
      fullWidth
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <CloudUpload sx={{ color: '#450839' }} />
          <Typography variant="h6">
            Create Model
          </Typography>
        </Box>
      </DialogTitle>

      <DialogContent dividers>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, my: 2 }}>
          {/* Error Alert */}
          {submitError && (
            <Alert 
              severity="error" 
              onClose={() => setSubmitError('')}
              sx={{ mb: 2 }}
            >
              {submitError}
            </Alert>
          )}

          {/* Validation Errors Summary */}
          {Object.keys(errors).length > 0 && (
            <Alert 
              severity="error" 
              sx={{ mb: 2 }}
            >
              <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 'bold' }}>
                Please fix the following errors:
              </Typography>
              <Box component="ul" sx={{ m: 0, pl: 2 }}>
                {Object.entries(errors).map(([key, value]) => (
                  value && (
                    <li key={key}>
                      <Typography variant="body2">{value}</Typography>
                    </li>
                  )
                ))}
              </Box>
            </Alert>
          )}


          {/* Basic Information */}
          <Card variant="outlined">
            <CardContent>
              <Typography variant="h6" sx={{ mb: 2 }}>
                Model Information
              </Typography>
              
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <TextField
                    label="GCS Path"
                    value={formData.gcs_path}
                    onChange={(e) => handleInputChange('gcs_path', e.target.value)}
                    error={!!errors.gcs_path}
                    helperText={errors.gcs_path || 'Must start with gs:// and contain valid bucket/path structure'}
                    required
                    fullWidth
                    size="small"
                    placeholder="gs://your-bucket/path/to/model"
                  />
                </Grid>
                <Grid item xs={12}>
                  <TextField
                    label="Model Name"
                    value={formData.metadata.model_name}
                    onChange={(e) => handleMetadataChange('model_name', e.target.value)}
                    error={!!errors['metadata.model_name']}
                    helperText={errors['metadata.model_name']}
                    required
                    fullWidth
                    size="small"
                  />
                </Grid>
              </Grid>
            </CardContent>
          </Card>


          {/* Inputs */}
          <Card variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6">
                  Input Features
                </Typography>
                <Button
                  startIcon={<Add />}
                  onClick={() => addInput()}
                  variant="outlined"
                  size="small"
                >
                  Add Input
                </Button>
              </Box>

              {/* Ensemble Model Note */}
              <Alert 
                severity="info" 
                sx={{ mb: 2 }}
              >
                <strong>Note:</strong> If you are uploading an Ensemble model, input features must only be specified for the ensemble model and not for child models.
              </Alert>

              {formData.metadata.inputs.map((input, index) => (
                <Card key={index} variant="outlined" sx={{ mb: 2, p: 2 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                    <Typography variant="subtitle1">
                      Input {index + 1}
                    </Typography>
                    <IconButton
                      onClick={() => removeInput(index)}
                      color="error"
                      size="small"
                    >
                      <Delete />
                    </IconButton>
                  </Box>
                  
                  <Grid container spacing={2}>
                    <Grid item xs={12}>
                      <TextField
                        label="Input Name"
                        value={input.name}
                        onChange={(e) => handleInputOutputChange('inputs', index, 'name', e.target.value)}
                        error={!!errors[`input_${index}_name`]}
                        helperText={errors[`input_${index}_name`]}
                        fullWidth
                        size="small"
                      />
                    </Grid>
                    
                    {/* Feature List Input */}
                    <Grid item xs={12}>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>
                          Features
                        </Typography>
                        <TextField
                          label="Enter Feature List"
                          value={featureListTexts[index] !== undefined ? featureListTexts[index] : getFeatureListText(input.features)}
                          onChange={(e) => handleFeatureListChange(e.target.value, index)}
                          fullWidth
                          multiline
                          rows={3}
                          size="small"
                          placeholder='Enter features like: "DEFAULT_FEATURE|reel:cg_score", "ONLINE_FEATURE|reel_user:derived_int32:user__reel_active_days_90_days"'
                        />
                        
                        {/* Parsed Features Display */}
                        {(() => {
                          const parsedFeatures = getParsedFeatures(input.features);
                          return parsedFeatures.length > 0 && (
                            <Box sx={{ mt: 2 }}>
                              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                                <Typography variant="subtitle2">
                                  Parsed Features:
                                </Typography>
                                <Button
                                  startIcon={<Add />}
                                  onClick={() => addFeature(index)}
                                  variant="outlined"
                                  size="small"
                                >
                                  Add Feature
                                </Button>
                              </Box>
                              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                                {parsedFeatures.map((feature, featureIndex) => (
                                  <Box key={featureIndex} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                    <Autocomplete
                                      options={[...(featureTypes || [])].sort((a, b) => a.localeCompare(b))}
                                      value={feature.left || null}
                                      onChange={(event, newValue) => handleFeatureEdit(index, featureIndex, 'left', newValue || '')}
                                      size="small"
                                      sx={{ flex: 1 }}
                                      loading={featureTypesLoading}
                                      disableClearable={true}
                                      renderInput={(params) => (
                                        <TextField
                                          {...params}
                                          label="Feature Type"
                                          size="small"
                                          placeholder="Search and select feature type"
                                          InputProps={{
                                            ...params.InputProps,
                                            endAdornment: (
                                              <>
                                                {featureTypesLoading ? <CircularProgress color="inherit" size={20} /> : null}
                                                {params.InputProps.endAdornment}
                                              </>
                                            ),
                                          }}
                                        />
                                      )}
                                    />
                                    <Typography variant="body2" sx={{ minWidth: 'fit-content' }}>
                                      |
                                    </Typography>
                                    <TextField
                                      label="Feature"
                                      value={feature.right}
                                      onChange={(e) => handleFeatureEdit(index, featureIndex, 'right', e.target.value)}
                                      size="small"
                                      sx={{ flex: 1 }}
                                    />
                                    <IconButton
                                      onClick={() => removeFeature(index, featureIndex)}
                                      color="error"
                                      size="small"
                                    >
                                      <Delete />
                                    </IconButton>
                                  </Box>
                                ))}
                              </Box>
                            </Box>
                          );
                        })()}
                    </Grid>
                  </Grid>
                </Card>
              ))}
            </CardContent>
          </Card>

        </Box>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button
          onClick={handleClose}
          variant="outlined"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={isSubmitting}
          startIcon={isSubmitting ? <CircularProgress size={20} /> : <CloudUpload />}
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730',
            },
          }}
        >
          {isSubmitting ? 'Uploading...' : 'Create Model'}
        </Button>
      </DialogActions>

      {/* Toast Notification */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={handleSnackbarClose}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Dialog>
  );
};

export default UploadModelModal;