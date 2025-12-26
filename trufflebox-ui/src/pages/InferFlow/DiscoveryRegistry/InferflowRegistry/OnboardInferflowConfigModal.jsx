import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  IconButton,
  Typography,
  Alert,
  Snackbar,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { useAuth } from '../../../Auth/AuthContext';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';
import InferflowConfigForm from './InferflowConfigForm';

const OnboardInferflowConfigModal = ({ open, onClose, onSuccess }) => {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [toastOpen, setToastOpen] = useState(false);
  const [toastMessage, setToastMessage] = useState('');
  const [toastSeverity, setToastSeverity] = useState('error');

  const [formData, setFormData] = useState({
    real_estate: '',
    tenant: '',
    config_identifier: '',
    rankers: [{
      model_name: '',
      end_point: '',
      calibration: '',
      batch_size: '',
      deadline: '110',
      entity_id: [],
      inputs: [{
        name: '',
        features: [],
        data_type: '',
        dims: ''
      }],
      outputs: [{
        name: '',
        data_type: '',
        model_scores_dims: '',
        model_scores: []
      }]
    }],
    re_rankers: [],
    response: {
      prism_logging_perc: 1,
      ranker_schema_features_in_response_perc: 0,
      response_features: [],
      log_features: false,
      log_batch_size: 1000
    },
    config_mapping: {
      deployable_id: ''
    }
  });

  const [modelsList, setModelsList] = useState([]);
  const [computeConfigs, setComputeConfigs] = useState([]);
  const [expressionVariables, setExpressionVariables] = useState({});
  const [expandedFeatures, setExpandedFeatures] = useState({});
  const [mpHosts, setMpHosts] = useState([]);
  const [featureTypes, setFeatureTypes] = useState([]);
  const [featureTypesLoading, setFeatureTypesLoading] = useState(false);
  const [expandedRankers, setExpandedRankers] = useState([0]);
  const [expandedReRankers, setExpandedReRankers] = useState([]);

  const CALIBRATION_OPTIONS = ['pctr_calibration', 'pcvr_calibration'];
  
  const DATA_TYPE_OPTIONS = [
    'DataTypeFP8E5M2',
    'DataTypeFP8E4M3',
    'DataTypeFP16',
    'DataTypeFP32',
    'DataTypeFP64',
    'DataTypeInt8',
    'DataTypeInt16',
    'DataTypeInt32',
    'DataTypeInt64',
    'DataTypeUint8',
    'DataTypeUint16',
    'DataTypeUint32',
    'DataTypeUint64',
    'DataTypeString',
    'DataTypeBool',
    'DataTypeFP8E5M2Vector',
    'DataTypeFP8E4M3Vector',
    'DataTypeFP16Vector',
    'DataTypeFP32Vector',
    'DataTypeFP64Vector',
    'DataTypeInt8Vector',
    'DataTypeInt16Vector',
    'DataTypeInt32Vector',
    'DataTypeInt64Vector',
    'DataTypeUint8Vector',
    'DataTypeUint16Vector',
    'DataTypeUint32Vector',
    'DataTypeUint64Vector',
    'DataTypeStringVector',
    'DataTypeBoolVector'
  ];

  useEffect(() => {
    fetchModelsList();
    fetchComputeConfigs();
    fetchMPHosts();
    fetchFeatureTypes();
  }, []);

  const fetchModelsList = async () => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/models`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      if (!response.data.error) {
        setModelsList(response.data.data);
      }
    } catch (error) {
      console.log('Error fetching models:', error);
    }
  };

  const fetchComputeConfigs = async () => {
    try {
      // First request to get pagination info
      const initialResponse = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-discovery/configs`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      
      if (!initialResponse.data.error) {
        const totalCount = initialResponse.data.pagination?.total_count;
        
        // If there are more items than the default limit, fetch all
        if (totalCount && totalCount > (initialResponse.data.pagination?.limit || 25)) {
          const allConfigsResponse = await axios.get(
            `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-discovery/configs?limit=${totalCount}`,
            {
              headers: {
                Authorization: `Bearer ${user?.token}`,
              },
            }
          );
          if (!allConfigsResponse.data.error) {
            setComputeConfigs(allConfigsResponse.data.data);
          }
        } else {
          // Use the data from the first request if total_count is within the default limit
          setComputeConfigs(initialResponse.data.data);
        }
      }
    } catch (error) {
      console.log('Error fetching compute configs:', error);
    }
  };

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
      if (response.data?.data && Array.isArray(response.data.data)) {
        const hosts = response.data.data.map(deployable => ({
          id: deployable.id,
          name: deployable.name,
          host: deployable.host
        }));
        setMpHosts(hosts);
      } else {
        setMpHosts([]);
      }
    } catch (error) {
      console.log('Error fetching InferFlow hosts:', error);
      setMpHosts([]);
    }
  };

  const fetchFeatureTypes = async () => {
    setFeatureTypesLoading(true);
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/feature-types`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      if (response.data?.data) {
        if (Array.isArray(response.data.data.feature_types)) {
          setFeatureTypes(response.data.data.feature_types);
        } else if (Array.isArray(response.data.data)) {
          setFeatureTypes(response.data.data);
        }
      }
    } catch (error) {
      console.log('Error fetching feature types:', error);
    } finally {
      setFeatureTypesLoading(false);
    }
  };



  const fetchExpressionVariables = async (configId, reRankerIndex) => {
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-expression/${configId}/variables`,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
          },
        }
      );
      if (!response.data.error) {
        setExpressionVariables(prev => ({
          ...prev,
          [reRankerIndex]: response.data.data
        }));
      }
    } catch (error) {
      console.log('Error fetching expression variables:', error);
    }
  };

  const handleBasicInfoChange = (field) => (event) => {
    setFormData(prev => ({
      ...prev,
      [field]: event.target.value.trim()
    }));
  };

  const handleRankerChange = (index, field, value) => {
    const newRankers = [...formData.rankers];
    newRankers[index] = {
      ...newRankers[index],
      [field]: value
    };

    if (field === 'model_name') {
      // Safety check: ensure modelsList exists and is an array
      if (Array.isArray(modelsList) && modelsList.length > 0) {
        const selectedModel = modelsList.find(model => model.model_name === value);
        if (selectedModel && selectedModel.meta_data) {
          if (selectedModel.host) {
            newRankers[index].end_point = selectedModel.host;
          }

          // Auto-fill batch_size from model's batch_size
          if (selectedModel.meta_data.batch_size) {
            newRankers[index].batch_size = selectedModel.meta_data.batch_size.toString();
            newRankers[index].max_batch_size = selectedModel.meta_data.batch_size;
          }

          // Safety check for inputs
          if (Array.isArray(selectedModel.meta_data.inputs)) {
            // Auto-fill inputs
            newRankers[index].inputs = selectedModel.meta_data.inputs.map(input => ({
              name: input.name,
              data_type: input.data_type,
              features: input.features || [],
              dims: input.dims || ''
            }));
          }

          // Safety check for outputs
          if (Array.isArray(selectedModel.meta_data.outputs)) {
            // Auto-fill outputs
            newRankers[index].outputs = selectedModel.meta_data.outputs.map(output => ({
              name: output.name,
              data_type: output.data_type,
              model_scores_dims: '',
              model_scores: []
            }));
          }
        }
      }
    }

    setFormData(prev => ({
      ...prev,
      rankers: newRankers
    }));
  };

  const handleInputChange = (rankerIndex, inputIndex, field, value) => {
    const newRankers = [...formData.rankers];
    newRankers[rankerIndex].inputs[inputIndex] = {
      ...newRankers[rankerIndex].inputs[inputIndex],
      [field]: value
    };
    setFormData(prev => ({
      ...prev,
      rankers: newRankers
    }));
  };

  const handleOutputChange = (rankerIndex, outputIndex, field, value) => {
    const newRankers = [...formData.rankers];
    newRankers[rankerIndex].outputs[outputIndex] = {
      ...newRankers[rankerIndex].outputs[outputIndex],
      [field]: value
    };
    setFormData(prev => ({
      ...prev,
      rankers: newRankers
    }));
  };

  const handleReRankerChange = (index, field, value) => {
    const newReRankers = [...formData.re_rankers];
    if (field === 'variable') {
      const { varName, varValue } = value;
      newReRankers[index] = {
        ...newReRankers[index],
        eq_variables: {
          ...newReRankers[index].eq_variables,
          [varName]: varValue
        }
      };
    } else {
      newReRankers[index] = {
        ...newReRankers[index],
        [field]: value
      };
      
      // When eq_id changes, reset eq_variables to clear old variables
      if (field === 'eq_id') {
        if (value) {
          newReRankers[index] = {
            ...newReRankers[index],
            eq_variables: {}
          };
          // Fetch expression variables for the new compute ID
          fetchExpressionVariables(value, index);
        } else {
          // Clear everything if eq_id is cleared
          newReRankers[index] = {
            ...newReRankers[index],
            eq_variables: {}
          };
        }
      }
    }
    setFormData(prev => ({
      ...prev,
      re_rankers: newReRankers
    }));
  };

  const handleResponseChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      response: {
        ...prev.response,
        [field]: value
      }
    }));
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

  // Helper functions for dimensions conversion
  const parseDimensionsFromString = (dimsString) => {
    try {
      if (!dimsString || dimsString.trim() === '') return [];
      const parsed = JSON.parse(dimsString);

      return Array.isArray(parsed) ? parsed.map(dim => {
        if (Array.isArray(dim) && dim.length > 0) {
          return String(dim[0]); // Display just the value
        }
        return String(dim);
      }) : [];
    } catch (e) {
      return [];
    }
  };

  const convertDimensionsToString = (dimsArray) => {
    try {
      // Convert each dimension to an array containing that value
      const parsedDims = dimsArray.map(dim => {
        // Try to parse as number if possible
        const numValue = Number(dim);
        return [isNaN(numValue) ? dim : numValue];
      });
      return JSON.stringify(parsedDims);
    } catch (e) {
      return '[]';
    }
  };



  const addRanker = () => {
    const newIndex = formData.rankers.length;
    setFormData(prev => ({
      ...prev,
      rankers: [...prev.rankers, {
        model_name: '',
        end_point: '',
        calibration: '',
        batch_size: '',
        deadline: '110',
        entity_id: [],
        inputs: [{
          name: '',
          features: [],
          data_type: '',
          dims: ''
        }],
        outputs: [{
          name: '',
          data_type: '',
          model_scores_dims: '',
          model_scores: []
        }]
      }]
    }));
    // Expand the newly added ranker
    setExpandedRankers(prev => [...prev, newIndex]);
  };

  const removeRanker = (index) => {
    setFormData(prev => ({
      ...prev,
      rankers: prev.rankers.filter((_, i) => i !== index)
    }));
    // Update expanded state
    setExpandedRankers(prev => 
      prev.filter(i => i !== index).map(i => i > index ? i - 1 : i)
    );
  };

  const addReRanker = () => {
    const newIndex = formData.re_rankers.length;
    setFormData(prev => ({
      ...prev,
      re_rankers: [...prev.re_rankers, {
        eq_variables: {},
        score: '',
        data_type: 'DataTypeFP32',
        eq_id: '',
        entity_id: []
      }]
    }));
    // Expand the newly added re-ranker
    setExpandedReRankers(prev => [...prev, newIndex]);
  };

  const removeReRanker = (index) => {
    setFormData(prev => ({
      ...prev,
      re_rankers: prev.re_rankers.filter((_, i) => i !== index)
    }));
    // Also clean up the expression variables for this index
    setExpressionVariables(prev => {
      const newVars = { ...prev };
      delete newVars[index];
      // Re-index remaining variables
      const reIndexed = {};
      Object.keys(newVars).forEach(key => {
        const oldIndex = parseInt(key);
        if (oldIndex > index) {
          reIndexed[oldIndex - 1] = newVars[key];
        } else {
          reIndexed[key] = newVars[key];
        }
      });
      return reIndexed;
    });
    // Update expanded state
    setExpandedReRankers(prev => 
      prev.filter(i => i !== index).map(i => i > index ? i - 1 : i)
    );
  };

  const handleRankerAccordionChange = (index) => {
    setExpandedRankers(prev => 
      prev.includes(index) 
        ? prev.filter(i => i !== index)
        : [...prev, index]
    );
  };

  const handleReRankerAccordionChange = (index) => {
    setExpandedReRankers(prev => 
      prev.includes(index) 
        ? prev.filter(i => i !== index)
        : [...prev, index]
    );
  };

  const validateForm = () => {
    const errors = [];

    if (!formData.real_estate.trim()) {
      errors.push('Real Estate is required');
    }
    if (!formData.tenant.trim()) {
      errors.push('Tenant is required');
    }
    if (!formData.config_identifier.trim()) {
      errors.push('Config Identifier is required');
    }

    // Validate rankers (all fields required except calibration)
    if (!formData.rankers || formData.rankers.length === 0) {
      errors.push('At least one ranker is required');
    } else {
      formData.rankers.forEach((ranker, rankerIndex) => {
        if (!ranker.model_name.trim()) {
          errors.push(`Ranker ${rankerIndex + 1}: Model Name is required`);
        }
        if (!ranker.end_point.trim()) {
          errors.push(`Ranker ${rankerIndex + 1}: End Point is required`);
        }
        if (!ranker.batch_size) {
          errors.push(`Ranker ${rankerIndex + 1}: Batch Size is required`);
        } else {
          const batchSize = Number(ranker.batch_size);
          if (ranker.max_batch_size && batchSize > ranker.max_batch_size) {
            errors.push(`Ranker ${rankerIndex + 1}: Batch Size (${batchSize}) cannot exceed model's maximum batch size (${ranker.max_batch_size})`);
          }
        }
        if (!ranker.deadline) {
          errors.push(`Ranker ${rankerIndex + 1}: Deadline is required`);
        }
        if (!ranker.entity_id || ranker.entity_id.length === 0) {
          errors.push(`Ranker ${rankerIndex + 1}: At least one Entity ID is required`);
        }

        // Validate inputs
        if (!ranker.inputs || ranker.inputs.length === 0) {
          errors.push(`Ranker ${rankerIndex + 1}: At least one input is required`);
        } else {
          ranker.inputs.forEach((input, inputIndex) => {
            if (!input.name.trim()) {
              errors.push(`Ranker ${rankerIndex + 1}, Input ${inputIndex + 1}: Name is required`);
            }
            if (!input.data_type) {
              errors.push(`Ranker ${rankerIndex + 1}, Input ${inputIndex + 1}: Data Type is required`);
            }
            if (!input.dims || !input.dims.toString().trim()) {
              errors.push(`Ranker ${rankerIndex + 1}, Input ${inputIndex + 1}: Dims is required`);
            }
            if (!input.features || input.features.length === 0) {
              errors.push(`Ranker ${rankerIndex + 1}, Input ${inputIndex + 1}: At least one feature is required`);
            }
          });
        }

        // Validate outputs
        if (!ranker.outputs || ranker.outputs.length === 0) {
          errors.push(`Ranker ${rankerIndex + 1}: At least one output is required`);
        } else {
          ranker.outputs.forEach((output, outputIndex) => {
            if (!output.name.trim()) {
              errors.push(`Ranker ${rankerIndex + 1}, Output ${outputIndex + 1}: Name is required`);
            }
            if (!output.data_type) {
              errors.push(`Ranker ${rankerIndex + 1}, Output ${outputIndex + 1}: Data Type is required`);
            }
            if (!output.model_scores_dims || !output.model_scores_dims.toString().trim()) {
              errors.push(`Ranker ${rankerIndex + 1}, Output ${outputIndex + 1}: Model Scores Dims is required`);
            }
            if (!output.model_scores || output.model_scores.length === 0) {
              errors.push(`Ranker ${rankerIndex + 1}, Output ${outputIndex + 1}: At least one Model Score is required`);
            }
            
            // Validate that dimensions match the number of scores
            if (output.model_scores && output.model_scores.length > 0) {
              try {
                const dims = JSON.parse(output.model_scores_dims || '[]');
                if (!Array.isArray(dims) || dims.length !== output.model_scores.length) {
                  errors.push(`Ranker ${rankerIndex + 1}, Output ${outputIndex + 1}: Number of dimensions (${Array.isArray(dims) ? dims.length : 0}) must match number of scores (${output.model_scores.length})`);
                }
              } catch (e) {
                errors.push(`Ranker ${rankerIndex + 1}, Output ${outputIndex + 1}: Invalid JSON format for Model Scores Dims`);
              }
            }
          });
        }
      });
    }

    // Validate re-rankers (if any are added, all fields are required)
    if (formData.re_rankers && formData.re_rankers.length > 0) {
      formData.re_rankers.forEach((reRanker, reRankerIndex) => {
        if (!reRanker.score.trim()) {
          errors.push(`Re-ranker ${reRankerIndex + 1}: Score Name is required`);
        }
        if (!reRanker.data_type) {
          errors.push(`Re-ranker ${reRankerIndex + 1}: Score Data Type is required`);
        }
        if (reRanker.eq_id === '' || reRanker.eq_id === null || reRanker.eq_id === undefined) {
          errors.push(`Re-ranker ${reRankerIndex + 1}: Expression ID is required`);
        }
        if (!reRanker.entity_id || reRanker.entity_id.length === 0) {
          errors.push(`Re-ranker ${reRankerIndex + 1}: At least one Entity ID is required`);
        }
        
        // Validate equation variables
        if (!reRanker.eq_variables || Object.keys(reRanker.eq_variables).length === 0) {
          errors.push(`Re-ranker ${reRankerIndex + 1}: At least one equation variable is required`);
        } else {
          Object.entries(reRanker.eq_variables).forEach(([key, value]) => {
            if (!key.trim()) {
              errors.push(`Re-ranker ${reRankerIndex + 1}: Variable name cannot be empty`);
            }
            if (!value.trim()) {
              errors.push(`Re-ranker ${reRankerIndex + 1}: Variable "${key}" value is required`);
            }
          });
        }
      });
    }

    // Validate config mapping
    if (!formData.config_mapping.deployable_id) {
      errors.push('InferFlow Host selection is required');
    }

    // Validate response entity ID (must be at 0th position of response_features)
    const entityId = formData.response.response_features?.[0] || '';
    if (!entityId.trim()) {
      errors.push('Entity ID in Selective Features is required');
    }

    return errors;
  };

  const handleSubmit = async () => {
    try {
      setLoading(true);
      setError('');

      // Validate form before submission
      const validationErrors = validateForm();
      if (validationErrors.length > 0) {
        const errorMessage = validationErrors.join('\n');
        setError(errorMessage);
        // Show toast notification
        setToastMessage('There are validation errors. Please scroll down to see the error details in the form.');
        setToastSeverity('error');
        setToastOpen(true);
        setLoading(false);
        return;
      }

      // Ensure entity ID is at 0th position of response_features
      const entityId = formData.response.response_features?.[0] || '';
      const otherFeatures = formData.response.response_features?.slice(1) || [];
      const responseFeatures = entityId ? [entityId, ...otherFeatures] : otherFeatures;

      const processedFormData = {
        ...formData,
        response: {
          ...formData.response,
          response_features: responseFeatures
        },
        rankers: formData.rankers.map(ranker => {
          const processedRanker = {
            ...ranker,
            batch_size: parseInt(ranker.batch_size, 10) || 0,
            deadline: parseInt(ranker.deadline, 10) || 0,
            outputs: ranker.outputs.map(output => {
              const processedOutput = { ...output };
              
              if (output.model_scores_dims && typeof output.model_scores_dims === 'string') {
                try {
                  processedOutput.model_scores_dims = JSON.parse(output.model_scores_dims);
                } catch (e) {
                  throw new Error(`Invalid JSON format for Model Scores Dims: ${output.model_scores_dims}`);
                }
              }
              
              return processedOutput;
            })
          };

          // Parse route_config: convert string to JSON object, or keep as object
          if (ranker.route_config) {
            if (typeof ranker.route_config === 'string' && ranker.route_config.trim() !== '') {
              try {
                processedRanker.route_config = JSON.parse(ranker.route_config);
              } catch (e) {
                throw new Error(`Invalid JSON format for Route Config: ${ranker.route_config}`);
              }
            } else if (typeof ranker.route_config === 'object') {
              // Already an object, keep as is
              processedRanker.route_config = ranker.route_config;
            } else {
              // Empty string or invalid value, remove it
              delete processedRanker.route_config;
            }
          } else {
            // No route_config, remove it
            delete processedRanker.route_config;
          }

          return processedRanker;
        })
      };

      const payload = {
        payload: processedFormData,
        created_by: user.email
      };

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/inferflow-config-registry/onboard`,
        payload,
        {
          headers: {
            Authorization: `Bearer ${user?.token}`,
            'Content-Type': 'application/json'
          },
        }
      );

      if (response.data.error) {
        setError(response.data.error);
        return;
      }

      const successMessage = response.data.data?.message || 'InferFlow Config onboarded successfully';
      onSuccess(successMessage);
      onClose();
    } catch (error) {
      setError(error.response?.data?.error || error.message || 'Failed to onboard InferFlow config');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: { maxHeight: '90vh' }
      }}
    >
      <DialogTitle sx={{ 
        bgcolor: '#450839', 
        color: 'white',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <Typography variant="h6">Onboard InferFlow Config</Typography>
        <IconButton
          edge="end"
          color="inherit"
          onClick={onClose}
          aria-label="close"
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers>
        <Box sx={{ p: 2 }}>
          <InferflowConfigForm
            formData={formData}
            isEditMode={false}
            isCloneMode={false}
            isAdmin={isAdmin}
            handleBasicInfoChange={handleBasicInfoChange}
            handleRankerChange={handleRankerChange}
            handleInputChange={handleInputChange}
            handleOutputChange={handleOutputChange}
            handleReRankerChange={handleReRankerChange}
            handleResponseChange={handleResponseChange}
            handleConfigMappingChange={handleConfigMappingChange}
            addRanker={addRanker}
            removeRanker={removeRanker}
            addReRanker={addReRanker}
            removeReRanker={removeReRanker}
            expandedRankers={expandedRankers}
            handleRankerAccordionChange={handleRankerAccordionChange}
            expandedReRankers={expandedReRankers}
            handleReRankerAccordionChange={handleReRankerAccordionChange}
            expandedFeatures={expandedFeatures}
            setExpandedFeatures={setExpandedFeatures}
            modelsList={modelsList}
            computeConfigs={computeConfigs}
            expressionVariables={expressionVariables}
            mpHosts={mpHosts}
            featureTypes={featureTypes}
            featureTypesLoading={featureTypesLoading}
            parseDimensionsFromString={parseDimensionsFromString}
            convertDimensionsToString={convertDimensionsToString}
            CALIBRATION_OPTIONS={CALIBRATION_OPTIONS}
            DATA_TYPE_OPTIONS={DATA_TYPE_OPTIONS}
          />
          {error && (
            <Alert severity="error" sx={{ mt: 2 }}>
              <Box component="pre" sx={{ whiteSpace: 'pre-line', fontFamily: 'inherit', margin: 0 }}>
                {error}
              </Box>
            </Alert>
          )}
        </Box>
      </DialogContent>

      <DialogActions sx={{ p: 2 }}>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSubmit}
          disabled={loading}
          sx={{
            bgcolor: '#450839',
            '&:hover': { bgcolor: '#380730' }
          }}
        >
          {loading ? 'Submitting...' : 'Submit'}
        </Button>
      </DialogActions>

      {/* Toast Notification */}
      <Snackbar
        open={toastOpen}
        autoHideDuration={6000}
        onClose={() => setToastOpen(false)}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={() => setToastOpen(false)}
          severity={toastSeverity}
          sx={{ width: '100%' }}
          variant="filled"
        >
          {toastMessage}
        </Alert>
      </Snackbar>
    </Dialog>
  );
};

export default OnboardInferflowConfigModal; 