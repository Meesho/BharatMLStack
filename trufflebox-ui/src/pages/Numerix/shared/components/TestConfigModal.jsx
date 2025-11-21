import React, { useState } from 'react';
import {
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Button,
  IconButton,
  CircularProgress,
  Box,
  Paper,
  Typography,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Divider,
  Alert,
  LinearProgress
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SettingsIcon from '@mui/icons-material/Settings';
import BugReportIcon from '@mui/icons-material/BugReport';
import InfoIcon from '@mui/icons-material/Info';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';

export const TestConfigModal = ({ open, onClose, selectedConfig, user, showNotification }) => {
  // State for testing process
  const [currentStep, setCurrentStep] = useState('request');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  // State for test request generation
  const [batchSize, setBatchSize] = useState('');
  const [dataType, setDataType] = useState('');
  const [generatedRequest, setGeneratedRequest] = useState(null);
  const [editableRequestData, setEditableRequestData] = useState('');
  
  // State for test execution
  const [responseData, setResponseData] = useState(null);

  const handleGenerateTestRequest = async () => {
    if (!selectedConfig || !batchSize) {
      setError('Please fill all required fields');
      return;
    }

    try {
      setLoading(true);
      setError('');
      
      const requestBody = {
        compute_id: selectedConfig.ComputeId.toString(),
        batch_size: batchSize.toString()
      };
      if (dataType) {
        requestBody.data_type = dataType;
      }

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-testing/functional-testing/generate-request`,
        requestBody,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data) {
        setGeneratedRequest(response.data);
        setEditableRequestData(JSON.stringify(response.data, null, 2));
        setCurrentStep('response');
      } else {
        setError('Request Generation Error');
      }
    } catch (error) {
      setError(error.response?.data?.error || 'Request Generation Error');
      console.log('Error generating test request:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteTest = async () => {
    if (!editableRequestData) {
      setError('No request data found');
      return;
    }

    try {
      setLoading(true);
      setError('');
      
      const deployableRes = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=NUMERIX`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      const endpoint = `${deployableRes?.data?.data[0].host}`;

      const parsedRequestData = JSON.parse(editableRequestData);
      const executeRequestBody = {
        ...parsedRequestData,
        end_point: endpoint
      };

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-testing/functional-testing/execute-request`,
        executeRequestBody,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      setResponseData(response.data);
      if (response.status === 200) {
        showNotification('Test ran successfully', 'success');
      } else {
        showNotification(
          response.data?.error?.message || 'Computation Error',
          'error'
        );
      }
    } catch (error) {
      setError(error.response?.data?.error?.message || 'Request Execution Error');
      console.log('Error executing request:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setCurrentStep('request');
    setLoading(false);
    setError('');
    setBatchSize('');
    setDataType('');
    setGeneratedRequest(null);
    setEditableRequestData('');
    setResponseData(null);
    onClose();
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
                Test Numerix Config
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.8 }}>
                {selectedConfig?.ComputeId}
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

        {error && (
          <Alert 
            severity="error" 
            sx={{ m: 2, mb: 0 }}
            onClose={() => setError('')}
          >
            <Typography variant="body2" sx={{ whiteSpace: 'pre-line' }}>
              {error}
            </Typography>
          </Alert>
        )}

        {currentStep === 'request' && (
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 3, color: '#450839', fontWeight: 600 }}>
              Configure Test Parameters
            </Typography>
            
            <Grid container spacing={3}>
              {/* Batch Size */}
              <Grid item xs={12} sm={6}>
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

              {/* Data Type */}
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth size="small">
                  <InputLabel>Data Type</InputLabel>
                  <Select
                    value={dataType}
                    label="Data Type"
                    onChange={(e) => setDataType(e.target.value)}
                  >
                    <MenuItem value="fp32">FP32</MenuItem>
                    <MenuItem value="fp64">FP64</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>

            <Divider sx={{ my: 3 }} />

            <Box sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, border: '1px solid #e0e0e0' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <InfoIcon sx={{ fontSize: 20, color: '#450839' }} />
                <Typography variant="body2" sx={{ fontWeight: 600, color: '#450839' }}>
                  Configuration Details
                </Typography>
              </Box>
              <Typography variant="body2" sx={{ color: '#666', mb: 1 }}>
                <strong>Compute ID:</strong> {selectedConfig?.ComputeId}
              </Typography>
              <Typography variant="body2" sx={{ color: '#666', wordBreak: 'break-word' }}>
                <strong>Expression:</strong> {selectedConfig?.InfixExpression}
              </Typography>
            </Box>
          </Box>
        )}

        {currentStep === 'response' && (
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 3, color: '#450839', fontWeight: 600 }}>
              Execute Test
            </Typography>

            {/* Request Data */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600 }}>
                Generated Request (Editable)
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
                  value={editableRequestData}
                  onChange={(e) => setEditableRequestData(e.target.value)}
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

            {/* Response */}
            {responseData && (
              <Box>
                <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600, color: responseData.error ? '#f44336' : '#4caf50' }}>
                  Test Response {responseData.error ? '(Failed)' : '(Success)'}
                </Typography>
                
                <Paper 
                  elevation={0} 
                  sx={{ 
                    p: 2, 
                    bgcolor: responseData.error ? '#ffebee' : '#f1f8e9',
                    borderRadius: 1,
                    maxHeight: 300, 
                    overflow: 'auto',
                    border: responseData.error ? '1px solid #ffcdd2' : '1px solid #c8e6c9'
                  }}
                >
                  <pre style={{ 
                    margin: 0, 
                    whiteSpace: 'pre-wrap',
                    fontFamily: 'monospace',
                    fontSize: '0.875rem',
                    color: responseData.error ? '#d32f2f' : '#2e7d32'
                  }}>
                    {responseData.error || JSON.stringify(responseData, null, 2)}
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
                disabled={loading || !editableRequestData.trim()}
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