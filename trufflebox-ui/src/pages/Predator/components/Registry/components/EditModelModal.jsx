import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  CircularProgress,
  Typography,
  Alert,
  Switch,
  FormControlLabel,
} from '@mui/material';
import { Edit as EditIcon } from '@mui/icons-material';
import { useAuth } from '../../../../Auth/AuthContext';
import * as URL_CONSTANTS from '../../../../../config';
import JsonDiffView from '../../../../../components/JsonDiffView';

const EditModelModal = ({ 
  open, 
  onClose, 
  model, 
  onEditComplete 
}) => {
  const { user } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [modelParamsData, setModelParamsData] = useState(null);
  const [currentModelData, setCurrentModelData] = useState(null);
  const [sourceModels, setSourceModels] = useState([]);
  const [serviceDeployableId, setServiceDeployableId] = useState(null);

  useEffect(() => {
    if (open && model) {
      // Set current model data
      setCurrentModelData(model);
      console.log('currentModelData', model);
      
      // Fetch deployables first, then source models, then model params
      fetchDeployables();
    }
  }, [open, model]);

  const fetchDeployables = async () => {
    setIsLoading(true);
    setError('');
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=PREDATOR`,
        {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
          },
        }
      );

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result = await response.json();
      if (result.error) {
        throw new Error(result.error.message || result.error);
      }

      // Find deployable that matches the model's host
      if (result.data && Array.isArray(result.data)) {
        const matchingDeployable = result.data.find(deployable => 
          deployable.host === model.host
        );
        
        if (matchingDeployable) {
          setServiceDeployableId(matchingDeployable.id);
          console.log('Found matching deployable:', matchingDeployable);
          console.log('Service deployable ID:', matchingDeployable.id);
        } else {
          console.warn(`No deployable found for host: ${model.host}`);
          setServiceDeployableId(null);
        }
      } else {
        throw new Error('Invalid deployables response format');
      }

      // After fetching deployables, fetch source models
      fetchSourceModels();

    } catch (err) {
      console.error('Error fetching deployables:', err);
      setError(`Failed to fetch deployables: ${err.message}`);
      // Still try to fetch source models even if deployables fail
      fetchSourceModels();
    } finally {
      setIsLoading(false);
    }
  };

  const fetchSourceModels = async () => {
    setIsLoading(true);
    setError('');
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

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result = await response.json();
      if (result.error) {
        throw new Error(result.error.message || result.error);
      }

      if (result.data && result.data.folders && Array.isArray(result.data.folders)) {
        setSourceModels(result.data.folders);
        // After fetching source models, fetch model params
        fetchModelParams(result.data.folders);
      } else {
        throw new Error('Invalid source models response format');
      }
    } catch (err) {
      console.error('Error fetching source models:', err);
      setError(`Failed to fetch source models: ${err.message}`);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchModelParams = async (sourceModelsData = sourceModels) => {
    if (!model?.modelName) {
      setError('Model name is required');
      return;
    }

    if (!sourceModelsData || sourceModelsData.length === 0) {
      setError('Source models data not available');
      return;
    }

    setIsLoading(true);
    setError('');

    try {
      // Find the model in source models by name
      const sourceModel = sourceModelsData.find(folder => folder.name === model.modelName);
      if (!sourceModel || !sourceModel.path) {
        throw new Error(`Model ${model.modelName} not found in source models`);
      }

      // Replace gcs:// with gs:// for the model-params API
      const gcsPath = sourceModel.path;
      const gsPath = gcsPath.replace('gcs://', 'gs://');
      const pathAfterGs = gsPath.startsWith('gs://') ? gsPath.substring(4) : gsPath;
      
      const apiUrl = `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/model-params?model_path=${pathAfterGs}`;
      
      const response = await fetch(apiUrl, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${user.token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result = await response.json();
      
      if (result.error) {
        throw new Error(result.error);
      }

      setModelParamsData(result);
      console.log('modelParamsData', result);

    } catch (error) {
      console.error('Error fetching model params:', error);
      setError(`Failed to fetch model parameters: ${error.message}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmitEditRequest = async () => {
    setIsSubmitting(true);
    setError('');

    try {
      const sourceModel = sourceModels.find(folder => folder.name === model.modelName);
      
      const { model_name, ...metaDataWithoutModelName } = modelParamsData || {};
      
      const payload = {
        payload: [{
            model_name: model.modelName,
            model_source_path: sourceModel.path,
            meta_data: metaDataWithoutModelName,
            config_mapping: {
              service_deployable_id: serviceDeployableId
            }
        }],
        created_by: user.email,
      };

      console.log('Edit request payload:', payload);

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/models/edit`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${user.token}`,
          },
          body: JSON.stringify(payload),
        }
      );

      const result = await response.json();

      if (result.error) {
        throw new Error(result.error);
      }

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      onEditComplete(result.data?.message || 'Edit request submitted successfully', 'success');
      onClose();

    } catch (error) {
      console.error('Error submitting edit request:', error);
      setError(`Failed to submit edit request: ${error.message}`);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    setError('');
    setModelParamsData(null);
    setCurrentModelData(null);
    setServiceDeployableId(null);
    onClose();
  };

  // Recursive function to order object keys at all levels
  const orderObjectKeysRecursive = (obj, keyOrder = []) => {
    // Handle null or undefined
    if (obj === null || obj === undefined) return obj;
    
    // Handle arrays - recursively process each element
    if (Array.isArray(obj)) {
      return obj.map(item => orderObjectKeysRecursive(item, keyOrder));
    }
    
    // Handle non-object primitives (string, number, boolean)
    if (typeof obj !== 'object') return obj;
    
    // Handle objects - order keys and recursively process values
    const orderedObj = {};
    
    // First, add keys in the defined order
    keyOrder.forEach(key => {
      if (obj.hasOwnProperty(key)) {
        orderedObj[key] = orderObjectKeysRecursive(obj[key], keyOrder);
      }
    });
    
    // Then, add remaining keys in alphabetical order (for consistency)
    const remainingKeys = Object.keys(obj)
      .filter(key => !keyOrder.includes(key))
      .sort(); // Sort alphabetically for consistency
    
    remainingKeys.forEach(key => {
      orderedObj[key] = orderObjectKeysRecursive(obj[key], keyOrder);
    });
    
    return orderedObj;
  };

  const formatModelData = (data, isCurrentModel = false) => {
    if (!data) return '';
    
    // Define key order for consistent sorting at all levels
    const keyOrder = [
      // Top level keys
      'model_name',
      'inputs',
      'outputs',
      'instance_count',
      'instance_type',
      'batch_size',
      'backend',
      'dynamic_batching_enabled',
      'platform',
      'ensemble_scheduling',
      // Input/Output keys
      'name',
      'dims',
      'data_type',
      'features',
      // Ensemble scheduling keys
      'step',
      'model_name',
      'model_version',
      'input_map',
      'output_map'
    ];
    
    if (isCurrentModel && data.metaData) {
      // Add model_name at the beginning for current model data
      const formattedData = {
        model_name: data.modelName || data.model_name,
        ...data.metaData
      };
      
      const orderedData = orderObjectKeysRecursive(formattedData, keyOrder);
      
      return JSON.stringify(orderedData, null, 2);
    }
    
    if (data && typeof data === 'object') {
      const orderedData = orderObjectKeysRecursive(data, keyOrder);
      
      return JSON.stringify(orderedData, null, 2);
    }
    
    return JSON.stringify(data, null, 2);
  };

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="lg"
      fullWidth
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <EditIcon sx={{ color: '#450839' }} />
          <Typography variant="h6">
            Edit Model Request - {model?.modelName || 'Unknown Model'}
          </Typography>
        </Box>
      </DialogTitle>

      <DialogContent dividers>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, my: 2 }}>
          
          {/* Model Information */}
          <Box>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Model Information
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Typography variant="body2">
                <strong>Model Name:</strong> {model?.modelName || 'N/A'}
              </Typography>
              <Typography variant="body2">
                <strong>GCS Path:</strong> {model?.gcsPath || 'N/A'}
              </Typography>
              <Typography variant="body2">
                <strong>Host:</strong> {model?.host || 'N/A'}
              </Typography>
              <Typography variant="body2">
                <strong>Service Deployable ID:</strong> {serviceDeployableId ? serviceDeployableId : 'Loading...'}
              </Typography>
            </Box>
          </Box>

          {/* Error Display */}
          {error && (
            <Alert severity="error">
              {error}
            </Alert>
          )}

          {/* Warning if service deployable ID not found */}
          {!isLoading && !error && serviceDeployableId === null && (
            <Alert severity="warning">
              Warning: Could not find a matching deployable for host "{model?.host}". The edit request may fail without a valid service deployable ID.
            </Alert>
          )}

          {/* Loading State */}
          {isLoading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 4 }}>
              <CircularProgress />
              <Typography variant="body2" sx={{ ml: 2 }}>
                Fetching model parameters...
              </Typography>
            </Box>
          )}
          
          {/* Diff View */}
          {!isLoading && !error && currentModelData && modelParamsData && (
            <Box>
              <Typography variant="h6" sx={{ mb: 2 }}>
                Metadata Differences
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                The following shows the differences between the current model metadata and the suggested metadata from the model parameters API.
              </Typography>
              
              <JsonDiffView
                originalText={formatModelData(currentModelData, true)}
                changedText={formatModelData(modelParamsData)}
                title="Model Metadata Differences"
              />
            </Box>
          )}

          {/* No Differences */}
          {!isLoading && !error && currentModelData && modelParamsData && 
           formatModelData(currentModelData, true) === formatModelData(modelParamsData) && (
            <Alert severity="info">
              No differences found between current model metadata and suggested metadata. The model is already up to date.
            </Alert>
          )}

          {/* Debug: Show when data is available but no diff is shown */}

          {/* Instructions */}
          <Box>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Instructions
            </Typography>
            <Typography variant="body2" color="textSecondary">
              This modal shows the differences between your current model metadata and the suggested metadata from the model parameters API. 
              If you want to apply these changes, click "Submit Edit Request" to raise a request for approval.
            </Typography>
          </Box>

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
          onClick={handleSubmitEditRequest}
          variant="contained"
          disabled={isSubmitting || isLoading || !modelParamsData || !serviceDeployableId || error}
          startIcon={isSubmitting ? <CircularProgress size={20} /> : <EditIcon />}
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730',
            },
          }}
        >
          {isSubmitting ? 'Submitting...' : 'Submit Edit Request'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default EditModelModal;