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
  MenuItem
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';

export const TestConfigModal = ({ open, onClose, selectedConfig, user, showNotification }) => {
  const [batchSize, setBatchSize] = useState('');
  const [dataType, setDataType] = useState('');
  const [testingSubmitting, setTestingSubmitting] = useState(false);
  const [executeSubmitting, setExecuteSubmitting] = useState(false);
  const [openExecuteModal, setOpenExecuteModal] = useState(false);
  const [requestData, setRequestData] = useState(null);
  const [editableRequestData, setEditableRequestData] = useState('');
  const [jsonValidationError, setJsonValidationError] = useState('');
  const [responseData, setResponseData] = useState(null);

  const handleGenerateTestRequest = async () => {
    if (!selectedConfig || !batchSize) {
      showNotification('Please fill all required fields', 'warning');
      return;
    }

    try {
      setTestingSubmitting(true);
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
        setRequestData(response.data);
        setEditableRequestData(formatJsonForDisplay(response.data));
        setJsonValidationError('');
        handleCloseTestModal();
        setOpenExecuteModal(true);
      } else {
        showNotification('Request Generation Error', 'error');
      }
    } catch (error) {
      showNotification(
        error.response?.data?.error || 'Request Generation Error',
        'error'
      );
      console.log('Error generating test request:', error);
    } finally {
      setTestingSubmitting(false);
    }
  };

  const validateJsonFormat = (jsonString) => {
    try {
      JSON.parse(jsonString);
      setJsonValidationError('');
      return true;
    } catch (error) {
      setJsonValidationError(`Invalid JSON format: ${error.message}`);
      return false;
    }
  };

  const handleRequestDataChange = (event) => {
    const newValue = event.target.value;
    setEditableRequestData(newValue);
    validateJsonFormat(newValue);
  };

  const handleExecuteRequest = async () => {
    if (!editableRequestData) {
      showNotification('No request data found', 'error');
      return;
    }

    if (!validateJsonFormat(editableRequestData)) {
      showNotification('Please fix JSON format errors before sending', 'error');
      return;
    }

    try {
      setExecuteSubmitting(true);
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
      showNotification(
        error.response?.data?.error?.message || 'Request Execution Error',
        'error'
      );
      console.log('Error executing request:', error);
    } finally {
      setExecuteSubmitting(false);
    }
  };

  const handleCloseTestModal = () => {
    setBatchSize('');
    setDataType('');
    onClose();
  };

  const handleCloseExecuteModal = () => {
    setOpenExecuteModal(false);
    setResponseData(null);
    setEditableRequestData('');
    setJsonValidationError('');
  };

  const formatJsonForDisplay = (json) => {
    return JSON.stringify(json, null, 2);
  };

  if (openExecuteModal) {
    return (
      <Dialog
        open={openExecuteModal}
        onClose={handleCloseExecuteModal}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          Testing
          <IconButton
            aria-label="close"
            onClick={handleCloseExecuteModal}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box>
              <Typography variant="subtitle1" gutterBottom>
                Request Body (Editable):
              </Typography>
              <TextField
                multiline
                fullWidth
                rows={12}
                value={editableRequestData}
                onChange={handleRequestDataChange}
                error={!!jsonValidationError}
                helperText={jsonValidationError}
                sx={{
                  '& .MuiInputBase-input': {
                    fontFamily: 'monospace',
                    fontSize: '0.875rem'
                  }
                }}
                placeholder="Request body JSON..."
              />
            </Box>

            {responseData && (
              <Box>
                <Typography variant="subtitle1" gutterBottom>
                  Response:
                </Typography>
                <Paper
                  elevation={0}
                  sx={{
                    p: 2,
                    maxHeight: '300px',
                    overflow: 'auto',
                    backgroundColor: '#f5f5f5',
                    fontFamily: 'monospace'
                  }}
                >
                  <pre>{formatJsonForDisplay(responseData)}</pre>
                </Paper>
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={handleExecuteRequest}
            variant="contained"
            disabled={executeSubmitting || !!jsonValidationError || !editableRequestData.trim()}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730'
              },
            }}
          >
            {executeSubmitting ? <CircularProgress size={24} /> : 'Send'}
          </Button>
        </DialogActions>
      </Dialog>
    );
  }

  return (
    <Dialog
      open={open}
      onClose={handleCloseTestModal}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>
        Test Config
        <IconButton
          aria-label="close"
          onClick={handleCloseTestModal}
          sx={{
            position: 'absolute',
            right: 8,
            top: 8,
          }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers>
        <TextField
          label="Enter Batch Size"
          fullWidth
          margin="normal"
          value={batchSize}
          onChange={(e) => setBatchSize(e.target.value)}
          type="number"
        />
        <FormControl fullWidth margin="normal">
          <InputLabel>Data Type</InputLabel>
          <Select
            value={dataType}
            label="Data Type"
            onChange={(e) => setDataType(e.target.value)}
          >
            <MenuItem value="fp32">fp32</MenuItem>
            <MenuItem value="fp64">fp64</MenuItem>
          </Select>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button
          onClick={handleCloseTestModal}
          sx={{
            color: '#450839',
            borderColor: '#450839',
            '&:hover': {
              backgroundColor: 'rgba(69, 8, 57, 0.04)',
              borderColor: '#380730'
            },
          }}
        >
          Cancel
        </Button>
        <Button
          onClick={handleGenerateTestRequest}
          variant="contained"
          disabled={testingSubmitting}
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730'
            },
          }}
        >
          {testingSubmitting ? <CircularProgress size={24} /> : 'Generate'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}; 