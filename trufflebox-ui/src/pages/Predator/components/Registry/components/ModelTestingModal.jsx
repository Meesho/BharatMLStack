import React, { useState, useEffect, memo, useCallback, useMemo } from 'react';
import {
  Box,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  CircularProgress,
  Paper,
  Divider,
  IconButton,
  Checkbox,
  FormControlLabel,
  Grid,
  LinearProgress,
  Chip,
  Card,
  CardContent,
  CardHeader,
  Tabs,
  Tab,
  Autocomplete,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Alert,
  AlertTitle
} from '@mui/material';
import JsonViewer from '../../../../../components/JsonViewer';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CodeIcon from '@mui/icons-material/Code';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SettingsIcon from '@mui/icons-material/Settings';
import BugReportIcon from '@mui/icons-material/BugReport';
import ViewListIcon from '@mui/icons-material/ViewList';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import SaveIcon from '@mui/icons-material/Save';
import CancelIcon from '@mui/icons-material/Cancel';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../../config';
import { useAuth } from '../../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../../constants/permissions';

const ModelTestingModal = ({
  open,
  onClose,
  model,
  onTestComplete
}) => {
  const { user, hasPermission } = useAuth();
  const service = SERVICES.PREDATOR;
  const screenType = SCREEN_TYPES.PREDATOR.MODEL;

  // State for testing process
  const [currentStep, setCurrentStep] = useState('request');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // State for test request generation
  const [batchSize, setBatchSize] = useState('2');
  const [generatedRequest, setGeneratedRequest] = useState(null);
  const [editableRequest, setEditableRequest] = useState('');

  // State for request body tabs and UI view
  const [requestTabValue, setRequestTabValue] = useState(0); // 0: JSON, 1: UI
  const [parsedRequestBody, setParsedRequestBody] = useState(null);
  const [syncError, setSyncError] = useState('');

  // State for test execution
  const [testEndpoint, setTestEndpoint] = useState('');
  const [testResponse, setTestResponse] = useState(null);

  // Load test configuration
  const [isLoadTestEnabled, setIsLoadTestEnabled] = useState(false);
  const [loadTestConfig, setLoadTestConfig] = useState({
    total_requests: '',
    concurrency: '',
    duration: '',
    load_start: '',
    load_step: '',
    load_end: '',
    connections: '',
    rps: ''
  });

  const validateJsonFormat = (jsonString) => {
    try {
      JSON.parse(jsonString);
      return true;
    } catch (error) {
      return false;
    }
  };

  const syncFromJsonToUI = () => {
    try {
      const parsed = JSON.parse(editableRequest);
      setParsedRequestBody(parsed);
      setSyncError('');
    } catch (error) {
      setSyncError('Invalid JSON format');
    }
  };

  const syncFromUIToJson = () => {
    if (parsedRequestBody) {
      setEditableRequest(JSON.stringify(parsedRequestBody, null, 2));
      setSyncError('');
    }
  };

  // Effect to handle initial sync when request is generated
  useEffect(() => {
    if (editableRequest && requestTabValue === 1) {
      syncFromJsonToUI();
    }
  }, [editableRequest, requestTabValue]);

  const handleTabChange = (event, newValue) => {
    if (newValue === 1 && editableRequest) {
      syncFromJsonToUI();
    } else if (newValue === 0 && parsedRequestBody) {
      syncFromUIToJson();
    }
    setRequestTabValue(newValue);
  };

  // Handle UI form changes - memoized to prevent RequestUIEditor re-renders
  const updateParsedRequestBody = useCallback((newData) => {
    setParsedRequestBody(newData);
  }, []);

  // Memoize the request body data to prevent unnecessary re-renders and maintain stable reference for editing
  const memoizedRequestBody = useMemo(() => {
    if (parsedRequestBody) {
      return parsedRequestBody;
    }
    if (editableRequest) {
      try {
        return JSON.parse(editableRequest);
      } catch {
        return {};
      }
    }
    return null;
  }, [parsedRequestBody, editableRequest]);

  const handleGenerateTestRequest = async () => {
    if (!hasPermission(service, screenType, ACTIONS.TEST)) {
      setError('You do not have permission to test models');
      return;
    }

    if (!model || !batchSize) {
      setError('Please fill all required fields');
      return;
    }

    try {
      setLoading(true);
      setError('');

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-testing/generate-request`,
        {
          batch_size: batchSize,
          model_name: model.modelName
        },
        {
          headers: {
            Authorization: `Bearer ${user.token}`,
          }
        }
      );

      const generatedData = response.data;
      setGeneratedRequest(generatedData);
      const requestBodyString = JSON.stringify(generatedData.request_body, null, 2);
      setEditableRequest(requestBodyString);
      setParsedRequestBody(generatedData.request_body);
      setTestEndpoint(generatedData['host-name'] || model.host || '');
      setCurrentStep('response');
    } catch (error) {
      setError(error.response?.data?.error || error.message || 'Failed to generate test request');
      console.error('Error generating test request:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteTest = async () => {
    let requestToUse = editableRequest;
    if (requestTabValue === 1 && parsedRequestBody) {
      requestToUse = JSON.stringify(parsedRequestBody, null, 2);
      setEditableRequest(requestToUse);
    }

    if (!requestToUse.trim() || !testEndpoint.trim()) {
      setError('Please provide both request data and test endpoint');
      return;
    }

    if (!hasPermission(service, screenType, ACTIONS.TEST)) {
      setError('You do not have permission to test models');
      return;
    }

    if (!validateJsonFormat(requestToUse)) {
      setError('Please fix JSON format errors before sending');
      return;
    }

    try {
      setLoading(true);
      setError('');

      const parsedRequestBody = JSON.parse(requestToUse);

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-testing/functional-testing/execute-request`,
        {
          end_point: testEndpoint,
          model_name: model.modelName,
          request_body: parsedRequestBody,
          "meta-data": generatedRequest?.["meta-data"] || {}
        },
        {
          headers: {
            Authorization: `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      setTestResponse(response.data);

      // If load test is enabled, send load test request after functional test
      if (isLoadTestEnabled) {
        handleExecuteLoadTest();
      }
    } catch (error) {
      const errorMessage = error.response?.data?.error || error.message || 'Failed to execute test';
      setError(errorMessage);
      setTestResponse({ error: errorMessage });
      console.error('Error sending test request:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleLoadTestConfigChange = (e) => {
    const { name, value } = e.target;
    setLoadTestConfig(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleLoadTestCheckboxChange = (event) => {
    if (!hasPermission(service, screenType, ACTIONS.LOAD_TEST)) {
      setError('You do not have permission to perform load testing');
      return;
    }
    setIsLoadTestEnabled(event.target.checked);
  };

  const handleExecuteLoadTest = async () => {
    if (!generatedRequest) return;

    if (!hasPermission(service, screenType, ACTIONS.LOAD_TEST)) {
      setError('You do not have permission to perform load testing');
      return;
    }

    if (!validateJsonFormat(editableRequest)) {
      setError('Please fix JSON format errors before sending');
      return;
    }

    try {
      setLoading(true);
      setError('');

      const parsedRequestBody = JSON.parse(editableRequest);

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-testing/load-test-request`,
        {
          host_name: testEndpoint,
          model_name: model.modelName,
          request_body: parsedRequestBody,
          load_test_config: loadTestConfig,
          "meta-data": generatedRequest?.["meta-data"] || {}
        },
        {
          headers: {
            Authorization: `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );


    } catch (error) {
      const errorMessage = error.response?.data?.error || error.message || 'Failed to execute load test';
      setError(errorMessage);
      console.error('Error sending load test request:', error);
    } finally {
      setLoading(false);
    }
  };

  const CodeBlock = ({ data, title = "Data" }) => {
    return (
      <JsonViewer
        data={data}
        title={title}
        defaultExpanded={true}
        maxHeight={300}
        enableCopy={true}
        enableDiff={false}
        editable={false}
      />
    );
  };

  // Component for UI-based request editing
  const RequestUIEditor = memo(({ requestBody, onUpdate }) => {
    const [isEditing, setIsEditing] = useState(false);
    const [localRequestBody, setLocalRequestBody] = useState(null);

    if (!requestBody || !requestBody.inputs) {
      return (
        <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
          <Typography>Generate a request to see the UI editor</Typography>
        </Box>
      );
    }

    const handleStartEdit = () => {
      setLocalRequestBody(JSON.parse(JSON.stringify(requestBody))); // Deep copy
      setIsEditing(true);
    };

    const handleSaveChanges = () => {
      onUpdate(localRequestBody);
      setIsEditing(false);
      setLocalRequestBody(null);
    };

    const handleCancelEdit = () => {
      setLocalRequestBody(null);
      setIsEditing(false);
    };

    const currentRequestBody = isEditing ? localRequestBody : requestBody;

    const updateInputField = (inputIndex, field, value) => {
      if (!isEditing) return;
      const updatedBody = { ...localRequestBody };
      updatedBody.inputs[inputIndex] = {
        ...updatedBody.inputs[inputIndex],
        [field]: typeof value === 'string' ? value.trim() : value
      };
      setLocalRequestBody(updatedBody);
    };

    const updateFeatureValue = (inputIndex, featureIndex, batchIndex, newValue) => {
      if (!isEditing) return;
      const updatedBody = { ...localRequestBody };
      if (!updatedBody.inputs[inputIndex].data[batchIndex]) {
        updatedBody.inputs[inputIndex].data[batchIndex] = [];
      }
      updatedBody.inputs[inputIndex].data[batchIndex][featureIndex] = newValue;
      setLocalRequestBody(updatedBody);
    };

    const updateFeatureName = (inputIndex, featureIndex, newFeature) => {
      if (!isEditing) return;
      const updatedBody = { ...localRequestBody };
      updatedBody.inputs[inputIndex].features[featureIndex] = typeof newFeature === 'string' ? newFeature.trim() : newFeature;
      setLocalRequestBody(updatedBody);
    };

    const addBatch = (inputIndex) => {
      if (!isEditing) return;
      const updatedBody = { ...localRequestBody };
      const currentInput = updatedBody.inputs[inputIndex];
      const numFeatures = currentInput.dims[1] || currentInput.features?.length || 1;
      const newBatch = new Array(numFeatures).fill("0.0");
      updatedBody.inputs[inputIndex].data.push(newBatch);
      updatedBody.inputs[inputIndex].dims[0] = updatedBody.inputs[inputIndex].data.length;
      setLocalRequestBody(updatedBody);
    };

    const removeBatch = (inputIndex, batchIndex) => {
      if (!isEditing) return;
      const updatedBody = { ...localRequestBody };
      updatedBody.inputs[inputIndex].data.splice(batchIndex, 1);
      updatedBody.inputs[inputIndex].dims[0] = updatedBody.inputs[inputIndex].data.length;
      setLocalRequestBody(updatedBody);
    };

    return (
      <Box>
        {/* Edit/Save/Cancel Controls - Sticky */}
        <Box sx={{ 
          position: 'sticky', 
          top: -2,
          zIndex: 100,
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center', 
          mb: 2, 
          mx: -2,
          mt: -2,
          p: 2, 
          backgroundColor: '#f8f9fa', 
          borderRadius: 0,
          boxShadow: isEditing ? '0 2px 8px rgba(0,0,0,0.1)' : 'none',
          border: '1px solid #e0e0e0',
          borderBottom: isEditing ? '2px solid #450839' : '1px solid #e0e0e0'
        }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, color: '#450839' }}>
            UI Editor {isEditing && <Chip label="Editing" size="small" color="warning" sx={{ ml: 1 }} />}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            {!isEditing ? (
              <Button
                size="small"
                variant="outlined"
                startIcon={<EditIcon />}
                onClick={handleStartEdit}
                sx={{
                  borderColor: '#450839',
                  color: '#450839',
                  '&:hover': {
                    borderColor: '#5a0a4a',
                    backgroundColor: 'rgba(69, 8, 57, 0.04)'
                  }
                }}
              >
                Edit
              </Button>
            ) : (
              <>
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<CancelIcon />}
                  onClick={handleCancelEdit}
                  sx={{ mr: 1 }}
                >
                  Cancel
                </Button>
                <Button
                  size="small"
                  variant="contained"
                  startIcon={<SaveIcon />}
                  onClick={handleSaveChanges}
                  sx={{
                    bgcolor: '#4caf50',
                    '&:hover': { bgcolor: '#45a049' }
                  }}
                >
                  Save Changes
                </Button>
              </>
            )}
          </Box>
        </Box>

        {currentRequestBody.inputs.map((input, inputIndex) => (
          <Accordion
            key={inputIndex}
            defaultExpanded
            sx={{
              boxShadow: 'none',
              '&:before': { display: 'none' },
              mb: 2
            }}
          >
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Input {inputIndex}: {input.name}
              </Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Grid container spacing={2}>
                {/* Basic Input Properties */}
                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Input Name"
                    value={input.name || ''}
                    onChange={(e) => updateInputField(inputIndex, 'name', e.target.value)}
                    size="small"
                    disabled={!isEditing}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Autocomplete
                    options={['BYTES', 'FP32', 'FP64', 'INT32', 'INT64']}
                    value={input.data_type || 'BYTES'}
                    onChange={(event, newValue) => {
                      updateInputField(inputIndex, 'data_type', newValue || 'BYTES');
                    }}
                    disabled={!isEditing}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Data Type"
                        size="small"
                        fullWidth
                      />
                    )}
                  />
                </Grid>

                {/* Batch Size Info & Controls */}
                <Grid item xs={12}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3, p: 2, backgroundColor: '#f8f9fa', borderRadius: 1 }}>
                    <Typography variant="subtitle1" sx={{ fontWeight: 600, color: '#555' }}>
                      Batch Size: {input.data?.length || 0} | Features: {input.features?.length || 0}
                    </Typography>
                    <Box>
                      <Button
                        size="small"
                        startIcon={<AddIcon />}
                        onClick={() => addBatch(inputIndex)}
                        sx={{ mr: 1 }}
                        disabled={!isEditing}
                      >
                        Add Batch
                      </Button>
                    </Box>
                  </Box>
                </Grid>

                {/* Feature-Based Data Editing */}
                <Grid item xs={12}>
                  {input.features && input.features.map((feature, featureIndex) => (
                    <Card key={featureIndex} sx={{
                      mb: 3,
                      border: '1px solid #e0e0e0',
                      boxShadow: 'none',
                      borderRadius: 2,
                      '&:hover': {
                        borderColor: '#ccc'
                      }
                    }}>
                      <CardHeader
                        title={
                          <Box>
                            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: '#333' }}>
                              Feature {featureIndex + 1}
                            </Typography>
                            <TextField
                              fullWidth
                              size="small"
                              value={feature || ''}
                              onChange={(e) => updateFeatureName(inputIndex, featureIndex, e.target.value)}
                              placeholder="Feature name..."
                              disabled={!isEditing}
                              sx={{
                                '& .MuiInputBase-input': { fontSize: '0.875rem' }
                              }}
                            />
                          </Box>
                        }
                        sx={{ pb: 2 }}
                      />
                      <CardContent sx={{ pt: 0, pb: 3 }}>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                          Values across batches:
                        </Typography>
                        <Grid container spacing={2}>
                          {input.data && input.data.map((batch, batchIndex) => (
                            <Grid item xs={12} sm={6} md={4} lg={3} key={batchIndex}>
                              <Box sx={{ position: 'relative' }}>
                                <TextField
                                  fullWidth
                                  size="small"
                                  label={`Batch ${batchIndex + 1}`}
                                  value={batch[featureIndex] || ''}
                                  onChange={(e) => updateFeatureValue(inputIndex, featureIndex, batchIndex, e.target.value)}
                                  disabled={!isEditing}
                                  sx={{
                                    '& .MuiInputBase-input': { fontSize: '0.875rem' },
                                    '& .MuiInputLabel-root': { fontSize: '0.875rem' }
                                  }}
                                />
                                {input.data.length > 1 && isEditing && (
                                  <IconButton
                                    size="small"
                                    onClick={() => removeBatch(inputIndex, batchIndex)}
                                    sx={{
                                      position: 'absolute',
                                      top: -6,
                                      right: -6,
                                      backgroundColor: '#fff',
                                      color: '#d32f2f',
                                      width: 24,
                                      height: 24,
                                      boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
                                      '&:hover': {
                                        backgroundColor: '#ffebee',
                                        color: '#b71c1c'
                                      }
                                    }}
                                  >
                                    <DeleteIcon sx={{ fontSize: 16 }} />
                                  </IconButton>
                                )}
                              </Box>
                            </Grid>
                          ))}
                        </Grid>

                        {/* Show data mapping info */}
                        <Box sx={{
                          mt: 2,
                          p: 2,
                          backgroundColor: '#f8f9fa',
                          borderRadius: 1,
                          border: '1px solid #e9ecef'
                        }}>
                          <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                            JSON: {input.data && input.data.map((_, batchIndex) =>
                              `data[${batchIndex}][${featureIndex}]`
                            ).join(' • ')}
                          </Typography>
                        </Box>
                      </CardContent>
                    </Card>
                  ))}
                </Grid>
              </Grid>
            </AccordionDetails>
          </Accordion>
        ))}
      </Box>
    );
  });
  RequestUIEditor.displayName = 'RequestUIEditor';

  const handleClose = () => {
    if (requestTabValue === 1 && parsedRequestBody) {
      setEditableRequest(JSON.stringify(parsedRequestBody, null, 2));
    }
    
    if (testResponse) {
      if (testResponse.error) {
        onTestComplete('Test failed: ' + testResponse.error, 'error');
      } else {
        onTestComplete('Test completed successfully', 'success');
      }
    }

    setCurrentStep('request');
    setLoading(false);
    setError('');
    setBatchSize('2');
    setGeneratedRequest(null);
    setEditableRequest('');
    setTestEndpoint('');
    setTestResponse(null);
    setIsLoadTestEnabled(false);
    setRequestTabValue(0);
    setParsedRequestBody(null);
    setSyncError('');
    setLoadTestConfig({
      total_requests: '',
      concurrency: '',
      duration: '',
      load_start: '',
      load_step: '',
      load_end: '',
      connections: '',
      rps: ''
    });
    onClose();
  };

  const handleBackToRequest = () => {
    if (requestTabValue === 1 && parsedRequestBody) {
      setEditableRequest(JSON.stringify(parsedRequestBody, null, 2));
    }
    setCurrentStep('request');
  };

  const activeStep = currentStep === 'request' ? 0 : 1;

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          maxHeight: '90vh'
        }
      }}
    >
      <DialogTitle
        sx={{
          bgcolor: '#450839',
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          py: 2
        }}
      >
        <Box>
          <Box display="flex" alignItems="center" gap={1.5}>
            <BugReportIcon sx={{ fontSize: 24 }} />
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Test Predator Model
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.8 }}>
                {model?.modelName}
              </Typography>
            </Box>
          </Box>

          {/* Progress Indicator */}
          <Box sx={{ mt: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Box sx={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                bgcolor: activeStep >= 0 ? '#fff' : 'rgba(255,255,255,0.3)'
              }} />
              <Typography variant="caption" sx={{ color: activeStep === 0 ? '#fff' : 'rgba(255,255,255,0.7)' }}>
                Configure
              </Typography>
            </Box>
            <Box sx={{ width: 16, height: 1, bgcolor: activeStep >= 1 ? '#fff' : 'rgba(255,255,255,0.3)' }} />
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Box sx={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                bgcolor: activeStep >= 1 ? '#fff' : 'rgba(255,255,255,0.3)'
              }} />
              <Typography variant="caption" sx={{ color: activeStep === 1 ? '#fff' : 'rgba(255,255,255,0.7)' }}>
                Execute
              </Typography>
            </Box>
          </Box>
        </Box>
        <IconButton
          onClick={handleClose}
          sx={{
            color: 'white',
            '&:hover': { backgroundColor: 'rgba(255,255,255,0.1)' }
          }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers sx={{ p: 0 }}>
        {loading && (
          <LinearProgress
            sx={{
              '& .MuiLinearProgress-bar': {
                backgroundColor: '#450839'
              }
            }}
          />
        )}

        {currentStep === 'request' && (
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 3, color: '#450839', fontWeight: 600 }}>
              Configure Test Parameters
            </Typography>

            {/* Info Panel for Recently Onboarded Models */}
            <Alert 
              severity="info" 
              sx={{ 
                mb: 3,
                '& .MuiAlert-icon': {
                  color: '#1976d2'
                }
              }}
            >
              <AlertTitle sx={{ fontWeight: 600 }}>Model Readiness</AlertTitle>
              If this model was recently onboarded, it may take a few minutes to become ready for testing. 
              Please allow some time for the model deployment to complete before running tests.
            </Alert>

            <Grid container spacing={3}>
              {/* Batch Size */}
              <Grid item xs={12} sm={4}>
                <TextField
                  fullWidth
                  label="Batch Size"
                  type="number"
                  size="small"
                  value={batchSize}
                  onChange={(e) => setBatchSize(e.target.value)}
                  required
                  inputProps={{ min: 1, max: 100 }}
                  helperText="Number of test samples"
                />
              </Grid>
            </Grid>

            <Divider sx={{ my: 3 }} />

            {/* Load Test Configuration */}
            {hasPermission(service, screenType, ACTIONS.LOAD_TEST) && (
              <Box sx={{ mb: 3 }}>
                <FormControlLabel
                  control={
                    <Checkbox
                      checked={isLoadTestEnabled}
                      onChange={handleLoadTestCheckboxChange}
                      sx={{
                        color: '#450839',
                        '&.Mui-checked': {
                          color: '#450839',
                        },
                      }}
                    />
                  }
                  label="Enable Load Testing"
                  sx={{ mb: 2 }}
                />

                {isLoadTestEnabled && (
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Total Requests"
                        name="total_requests"
                        value={loadTestConfig.total_requests}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Concurrency"
                        name="concurrency"
                        value={loadTestConfig.concurrency}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Duration (seconds)"
                        name="duration"
                        value={loadTestConfig.duration}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Load Start"
                        name="load_start"
                        value={loadTestConfig.load_start}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Load Step"
                        name="load_step"
                        value={loadTestConfig.load_step}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Load End"
                        name="load_end"
                        value={loadTestConfig.load_end}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Connections"
                        name="connections"
                        value={loadTestConfig.connections}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="RPS"
                        name="rps"
                        value={loadTestConfig.rps}
                        onChange={handleLoadTestConfigChange}
                        type="number"
                        size="small"
                      />
                    </Grid>
                  </Grid>
                )}
              </Box>
            )}
          </Box>
        )}

        {currentStep === 'response' && (
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 3, color: '#450839', fontWeight: 600 }}>
              Execute Test
            </Typography>

            {/* Endpoint */}
            <TextField
              fullWidth
              label="Model Endpoint"
              size="small"
              value={testEndpoint}
              onChange={(e) => setTestEndpoint(e.target.value)}
              placeholder="e.g., predator-service.int.meesho.int"
              helperText="Target endpoint for test execution"
              required
              sx={{ mb: 3 }}
              disabled={true}
            />

            {/* Request Data with Tabs */}
            <Box sx={{ mb: 3 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                  Generated Request (Editable)
                </Typography>
                {syncError && (
                  <Chip
                    label={syncError}
                    color="error"
                    size="small"
                    sx={{ fontSize: '0.75rem' }}
                  />
                )}
              </Box>

              <Paper
                elevation={0}
                sx={{
                  border: '1px solid #e0e0e0',
                  borderRadius: 1,
                  overflow: 'hidden'
                }}
              >
                {/* Tabs */}
                <Tabs
                  value={requestTabValue}
                  onChange={handleTabChange}
                  sx={{
                    borderBottom: '1px solid #e0e0e0',
                    minHeight: 42,
                    '& .MuiTab-root': {
                      minHeight: 42,
                      textTransform: 'none',
                      fontSize: '0.875rem'
                    }
                  }}
                >
                  <Tab
                    icon={<CodeIcon sx={{ fontSize: 18 }} />}
                    iconPosition="start"
                    label="JSON View"
                    sx={{ gap: 1 }}
                  />
                  <Tab
                    icon={<ViewListIcon sx={{ fontSize: 18 }} />}
                    iconPosition="start"
                    label="UI Editor"
                    sx={{ gap: 1 }}
                  />
                </Tabs>

                {/* Tab Content */}
                <Box sx={{ minHeight: 300 }}>
                  {requestTabValue === 0 && (
                    <Box sx={{ p: 2 }}>
                      {memoizedRequestBody ? (
                        <JsonViewer
                          data={memoizedRequestBody}
                          title="Request Configuration"
                          defaultExpanded={true}
                          editable={true}
                          originalData={generatedRequest?.request_body}
                          maxHeight={300}
                          enableCopy={true}
                          enableDiff={true}
                          onDataChange={(newData) => {
                            setParsedRequestBody(newData);
                            setEditableRequest(JSON.stringify(newData, null, 2));
                            setSyncError('');
                          }}
                          sx={{ border: 'none' }}
                        />
                      ) : (
                        <Box sx={{ 
                          p: 3, 
                          textAlign: 'center', 
                          color: 'text.secondary',
                          backgroundColor: '#fafafa',
                          borderRadius: 1,
                          border: '1px solid #e0e0e0'
                        }}>
                          <Typography variant="body2">
                            Generate a test request to see the JSON configuration
                          </Typography>
                        </Box>
                      )}
                    </Box>
                  )}

                  {requestTabValue === 1 && (
                    <Box sx={{ maxHeight: 400, overflow: 'auto', p: 2, paddingTop: 0 }}>
                      <RequestUIEditor
                        requestBody={parsedRequestBody}
                        onUpdate={updateParsedRequestBody}
                      />
                    </Box>
                  )}
                </Box>
              </Paper>
            </Box>

            {/* Response */}
            {testResponse && (
              <Box>
                <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600, color: testResponse.error ? '#f44336' : '#4caf50' }}>
                  Test Response {testResponse.error ? '(Failed)' : '(Success)'}
                </Typography>

                <JsonViewer
                  data={testResponse}
                  title={testResponse.error ? "Error Response" : "Test Response"}
                  defaultExpanded={true}
                  editable={false}
                  maxHeight={300}
                  enableCopy={true}
                  enableDiff={false}
                  sx={{
                    border: testResponse.error ? '1px solid #ffcdd2' : '1px solid #c8e6c9',
                    backgroundColor: testResponse.error ? '#ffebee' : '#f1f8e9'
                  }}
                />
              </Box>
            )}
          </Box>
        )}
      </DialogContent>

      <DialogActions sx={{ p: 2, justifyContent: 'space-between' }}>
        <Button
          onClick={handleClose}
          variant="outlined"
          sx={{
            color: '#666',
            borderColor: '#ddd',
            '&:hover': {
              borderColor: '#999',
              backgroundColor: 'rgba(0,0,0,0.04)'
            }
          }}
        >
          Close
        </Button>

        <Box sx={{ display: 'flex', gap: 1 }}>
          {currentStep === 'request' && (
            <Button
              onClick={handleGenerateTestRequest}
              variant="contained"
              disabled={loading}
              startIcon={loading ? <CircularProgress size={18} color="inherit" /> : <SettingsIcon />}
              sx={{
                bgcolor: '#450839',
                '&:hover': { bgcolor: '#5a0a4a' }
              }}
            >
              {loading ? 'Generating...' : 'Generate Request'}
            </Button>
          )}

          {currentStep === 'response' && (
            <>
              <Button
                onClick={handleBackToRequest}
                variant="outlined"
                sx={{
                  color: '#450839',
                  borderColor: '#450839',
                  '&:hover': {
                    borderColor: '#5a0a4a',
                    backgroundColor: 'rgba(69, 8, 57, 0.04)'
                  }
                }}
              >
                ← Back
              </Button>
              <Button
                onClick={handleExecuteTest}
                variant="contained"
                disabled={loading || !testEndpoint.trim() || !editableRequest.trim()}
                startIcon={loading ? <CircularProgress size={18} color="inherit" /> : <PlayArrowIcon />}
                sx={{
                  bgcolor: '#450839',
                  '&:hover': { bgcolor: '#5a0a4a' }
                }}
              >
                {loading ? 'Testing...' : 'Execute Test'}
              </Button>
            </>
          )}
        </Box>
      </DialogActions>
    </Dialog>
  );
};

export default ModelTestingModal; 