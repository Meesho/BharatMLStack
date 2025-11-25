import React, { useState } from 'react';
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
  CircularProgress,
  Paper,
  Divider,
  Grid,
  Alert,
  Chip,
  Card,
  CardContent,
  CardHeader,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  LinearProgress,
  Fade,
  Zoom,
  MenuItem,
  FormControl,
  InputLabel,
  Select
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import SendIcon from '@mui/icons-material/Send';
import InfoIcon from '@mui/icons-material/Info';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SettingsIcon from '@mui/icons-material/Settings';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CodeIcon from '@mui/icons-material/Code';
import BugReportIcon from '@mui/icons-material/BugReport';

import { useAuth } from '../../../Auth/AuthContext';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';

const MPConfigTestingModal = ({ open, onClose, configData, onTestComplete }) => {
  const { user } = useAuth();
  
  // State for testing process
  const [currentStep, setCurrentStep] = useState('request');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  // State for test request generation
  const [batchSize, setBatchSize] = useState('2');
  const [entity, setEntity] = useState('');
  const [metaData, setMetaData] = useState({});
  const [defaultFeatures, setDefaultFeatures] = useState({});
  const [metaDataKey, setMetaDataKey] = useState('');
  const [metaDataValue, setMetaDataValue] = useState('');
  const [defaultFeatureKey, setDefaultFeatureKey] = useState('');
  const [defaultFeatureValue, setDefaultFeatureValue] = useState('');
  const [generatedRequest, setGeneratedRequest] = useState(null);
  const [editableRequest, setEditableRequest] = useState('');
  const [editableMetaData, setEditableMetaData] = useState('');
  
  const entityOptions = ['catalog', 'product', 'reel', 'supplier'];
  
  // State for test execution
  const [testEndpoint, setTestEndpoint] = useState('');
  const [testResponse, setTestResponse] = useState(null);

  const handleClose = () => {
    setCurrentStep('request');
    setLoading(false);
    setError('');
    setBatchSize('2');
    setEntity('');
    setMetaData({});
    setDefaultFeatures({});
    setMetaDataKey('');
    setMetaDataValue('');
    setDefaultFeatureKey('');
    setDefaultFeatureValue('');
    setGeneratedRequest(null);
    setEditableRequest('');
    setEditableMetaData('');
    setTestEndpoint('');
    setTestResponse(null);
    onClose();
  };

  const handleGenerateTestRequest = async () => {
    if (!configData || !batchSize) {
      setError('Please fill all required fields');
      return;
    }

    try {
      setLoading(true);
      setError('');
      
      const requestBody = {
        model_config_id: configData.config_id,
        batch_size: batchSize,
        entity: entity,
        meta_data: metaData,
        default_features: defaultFeatures
      };

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/mp-config-testing/functional-test/generate-request`,
        requestBody,
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

      const generatedData = response.data;
      setGeneratedRequest(generatedData);
      setEditableRequest(JSON.stringify(generatedData.request_body, null, 2));
      setEditableMetaData(JSON.stringify(generatedData.meta_data || {}, null, 2));
      setTestEndpoint(configData.host || '');
      setCurrentStep('response');
    } catch (err) {
      setError(err.response?.data?.error || err.message || 'Failed to generate test request');
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteTest = async () => {
    if (!editableRequest.trim() || !testEndpoint.trim()) {
      setError('Please provide both request data and test endpoint');
      return;
    }

    try {
      setLoading(true);
      setError('');
      
      let requestData;
      let metaDataObj = {};
      
      try {
        requestData = JSON.parse(editableRequest);
      } catch (e) {
        setError('Invalid JSON in Request Body: ' + e.message);
        setLoading(false);
        return;
      }
      
      if (editableMetaData.trim()) {
        try {
          metaDataObj = JSON.parse(editableMetaData);
        } catch (e) {
          setError('Invalid JSON in Meta Data: ' + e.message);
          setLoading(false);
          return;
        }
      }
      
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/mp-config-testing/functional-test/execute-request`,
        {
          request_body: requestData,
          end_point: testEndpoint,
          meta_data: metaDataObj
        },
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

      setTestResponse(response.data);
      if (onTestComplete) {
        onTestComplete(configData.config_id, 'success');
      }
    } catch (err) {
      const errorMessage = err.response?.data?.error || err.message || 'Failed to execute test';
      setError(errorMessage);
      setTestResponse({ error: errorMessage });
      if (onTestComplete) {
        onTestComplete(configData.config_id, 'failed');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleAddMetaData = () => {
    if (metaDataKey.trim() !== '' && metaDataValue.trim() !== '') {
      setMetaData(prev => ({
        ...prev,
        [metaDataKey.trim()]: metaDataValue.trim()
      }));
      setMetaDataKey('');
      setMetaDataValue('');
    }
  };

  const removeMetaData = (keyToRemove) => {
    setMetaData(prev => {
      const newMetaData = { ...prev };
      delete newMetaData[keyToRemove];
      return newMetaData;
    });
  };

  const handleAddDefaultFeature = () => {
    if (defaultFeatureKey.trim() !== '' && defaultFeatureValue.trim() !== '') {
      setDefaultFeatures(prev => ({
        ...prev,
        [defaultFeatureKey.trim()]: defaultFeatureValue.trim()
      }));
      setDefaultFeatureKey('');
      setDefaultFeatureValue('');
    }
  };

  const removeDefaultFeature = (keyToRemove) => {
    setDefaultFeatures(prev => {
      const newDefaultFeatures = { ...prev };
      delete newDefaultFeatures[keyToRemove];
      return newDefaultFeatures;
    });
  };

  const steps = ['Configure Test', 'Execute & Review'];
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
                Test InferFlow Config
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.8 }}>
                {configData?.config_id}
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
              
              {/* Entity */}
              <Grid item xs={12} sm={4}>
                <FormControl fullWidth size="small">
                  <InputLabel>Entity</InputLabel>
                  <Select
                    value={entity}
                    onChange={(e) => setEntity(e.target.value)}
                    label="Entity"
                  >
                    {entityOptions.map((option) => (
                      <MenuItem key={option} value={option}>
                        {option.charAt(0).toUpperCase() + option.slice(1)}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
            </Grid>

            <Divider sx={{ my: 3 }} />

            {/* Meta Data */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600, color: '#450839' }}>
                Meta Data (Key-Value Pairs)
              </Typography>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid item xs={5}>
                  <TextField
                    fullWidth
                    label="Key"
                    size="small"
                    value={metaDataKey}
                    onChange={(e) => setMetaDataKey(e.target.value)}
                    placeholder="e.g., MEESHO-USER-ID"
                    helperText="Meta data key"
                  />
                </Grid>
                <Grid item xs={5}>
                  <TextField
                    fullWidth
                    label="Value"
                    size="small"
                    value={metaDataValue}
                    onChange={(e) => setMetaDataValue(e.target.value)}
                    placeholder="e.g., 407656252"
                    helperText="Meta data value"
                  />
                </Grid>
                <Grid item xs={2}>
                  <Button
                    fullWidth
                    variant="outlined"
                    onClick={handleAddMetaData}
                    disabled={!metaDataKey.trim() || !metaDataValue.trim()}
                    sx={{
                      height: '40px',
                      borderColor: '#450839',
                      color: '#450839',
                      '&:hover': {
                        borderColor: '#5a0a4a',
                        backgroundColor: 'rgba(69, 8, 57, 0.04)'
                      }
                    }}
                  >
                    Add
                  </Button>
                </Grid>
              </Grid>
              
              {Object.keys(metaData).length > 0 && (
                <Box sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, border: '1px solid #e0e0e0' }}>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    {Object.entries(metaData).map(([key, value]) => (
                      <Box key={key} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip
                          label={`${key}: ${value}`}
                          onDelete={() => removeMetaData(key)}
                          color="primary"
                          variant="outlined"
                          size="small"
                          sx={{
                            borderColor: '#450839',
                            color: '#450839',
                            flex: 1
                          }}
                        />
                      </Box>
                    ))}
                  </Box>
                </Box>
              )}
            </Box>

            {/* Default Features */}
            <Box>
              <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600, color: '#2e7d32' }}>
                Default Features (Key-Value Pairs)
              </Typography>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid item xs={5}>
                  <TextField
                    fullWidth
                    label="Feature Key"
                    size="small"
                    value={defaultFeatureKey}
                    onChange={(e) => setDefaultFeatureKey(e.target.value)}
                    placeholder="e.g., catalog:source"
                    helperText="Feature key"
                  />
                </Grid>
                <Grid item xs={5}>
                  <TextField
                    fullWidth
                    label="Feature Value"
                    size="small"
                    value={defaultFeatureValue}
                    onChange={(e) => setDefaultFeatureValue(e.target.value)}
                    placeholder="e.g., entity_based_similarity:intent-iacg-80"
                    helperText="Feature value"
                  />
                </Grid>
                <Grid item xs={2}>
                  <Button
                    fullWidth
                    variant="outlined"
                    onClick={handleAddDefaultFeature}
                    disabled={!defaultFeatureKey.trim() || !defaultFeatureValue.trim()}
                    sx={{
                      height: '40px',
                      borderColor: '#2e7d32',
                      color: '#2e7d32',
                      '&:hover': {
                        borderColor: '#1b5e20',
                        backgroundColor: 'rgba(46, 125, 50, 0.04)'
                      }
                    }}
                  >
                    Add
                  </Button>
                </Grid>
              </Grid>
              
              {Object.keys(defaultFeatures).length > 0 && (
                <Box sx={{ p: 2, bgcolor: '#f1f8e9', borderRadius: 1, border: '1px solid #c8e6c9' }}>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    {Object.entries(defaultFeatures).map(([key, value]) => (
                      <Box key={key} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip
                          label={`${key}: ${value}`}
                          onDelete={() => removeDefaultFeature(key)}
                          color="success"
                          variant="outlined"
                          size="small"
                          sx={{
                            borderColor: '#2e7d32',
                            color: '#2e7d32',
                            flex: 1
                          }}
                        />
                      </Box>
                    ))}
                  </Box>
                </Box>
              )}
            </Box>
          </Box>
        )}

        {currentStep === 'response' && (
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 3, color: '#450839', fontWeight: 600 }}>
              Execute Test
            </Typography>
            
            {error && (
              <Alert severity="error" sx={{ mb: 3 }}>
                {error}
              </Alert>
            )}
            
            {/* Endpoint */}
            <TextField
              fullWidth
              label="InferFlow Endpoint"
              size="small"
              value={testEndpoint}
              onChange={(e) => setTestEndpoint(e.target.value)}
              placeholder="e.g., inferflow-service-experiment.int.meesho.int"
              helperText="Target endpoint for test execution"
              required
              sx={{ mb: 3 }}
            />

            {/* Request Data */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600 }}>
                Request Body (Editable)
              </Typography>
              <Paper 
                elevation={0} 
                sx={{ 
                  border: '1px solid #e0e0e0',
                  borderRadius: 1,
                  overflow: 'hidden'
                }}
              >
                <TextField
                  fullWidth
                  multiline
                  rows={10}
                  value={editableRequest}
                  onChange={(e) => setEditableRequest(e.target.value)}
                  placeholder="Generated request will appear here..."
                  variant="outlined"
                  sx={{ 
                    '& .MuiOutlinedInput-root': {
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                      backgroundColor: '#fafafa',
                      '& fieldset': { border: 'none' },
                      '&:hover fieldset': { border: 'none' },
                      '&.Mui-focused fieldset': { border: 'none' }
                    }
                  }}
                />
              </Paper>
            </Box>

            {/* Meta Data */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600 }}>
                Meta Data (Editable)
              </Typography>
              <Paper 
                elevation={0} 
                sx={{ 
                  border: '1px solid #e0e0e0',
                  borderRadius: 1,
                  overflow: 'hidden'
                }}
              >
                <TextField
                  fullWidth
                  multiline
                  rows={6}
                  value={editableMetaData}
                  onChange={(e) => setEditableMetaData(e.target.value)}
                  placeholder="Meta data will appear here..."
                  variant="outlined"
                  sx={{ 
                    '& .MuiOutlinedInput-root': {
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                      backgroundColor: '#f8f9fa',
                      '& fieldset': { border: 'none' },
                      '&:hover fieldset': { border: 'none' },
                      '&.Mui-focused fieldset': { border: 'none' }
                    }
                  }}
                />
              </Paper>
            </Box>

            {/* Response */}
            {testResponse && (
              <Box>
                <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600, color: testResponse.error ? '#f44336' : '#4caf50' }}>
                  Test Response {testResponse.error ? '(Failed)' : '(Success)'}
                </Typography>
                
                <Paper 
                  elevation={0} 
                  sx={{ 
                    p: 2, 
                    bgcolor: testResponse.error ? '#ffebee' : '#f1f8e9',
                    borderRadius: 1,
                    maxHeight: 300, 
                    overflow: 'auto',
                    border: testResponse.error ? '1px solid #ffcdd2' : '1px solid #c8e6c9'
                  }}
                >
                  <pre style={{ 
                    margin: 0, 
                    whiteSpace: 'pre-wrap',
                    fontFamily: 'monospace',
                    fontSize: '0.875rem',
                    color: testResponse.error ? '#d32f2f' : '#2e7d32'
                  }}>
                    {testResponse.error != "" ? testResponse.error : JSON.stringify(testResponse, null, 2)}
                  </pre>
                </Paper>
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
                onClick={() => setCurrentStep('request')}
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

export default MPConfigTestingModal;