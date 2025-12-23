import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  IconButton,
  Box,
  Typography,
  Paper,
  Chip,
  Grid,
  Tooltip
} from '@mui/material';
import JsonViewer from '../../../../components/JsonViewer';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';

const ViewInferflowConfigDetailsModal = ({ open, onClose, configData }) => {
  const getStatusChip = (status) => {
    if (status) {
      return (
        <Chip
          icon={<CheckCircleIcon />}
          label="Running"
          size="small"
          sx={{
            backgroundColor: '#E8F5E9',
            color: '#2E7D32',
            fontWeight: 'bold'
          }}
        />
      );
    } else {
      return (
        <Chip
          icon={<CancelIcon />}
          label="Stopped"
          size="small"
          sx={{
            backgroundColor: '#FFEBEE',
            color: '#D32F2F',
            fontWeight: 'bold'
          }}
        />
      );
    }
  };

  const getTestStatusChip = (testResults) => {
    const status = testResults?.tested ? 'PASSED' : 'NOT_TESTED';
    switch (status) {
      case 'PASSED':
        return <Chip label="Passed" size="small" sx={{ backgroundColor: '#E8F5E9', color: '#2E7D32', fontWeight: 'bold' }} />;
      case 'FAILED':
        return <Chip label="Failed" size="small" sx={{ backgroundColor: '#FFEBEE', color: '#D32F2F', fontWeight: 'bold' }} />;
      default:
        return <Chip label="Not Tested" size="small" sx={{ backgroundColor: '#FFF3E0', color: '#E65100', fontWeight: 'bold' }} />;
    }
  };

  const renderTestMessage = (message) => {
    if (!message) return null;
    
    const MAX_LENGTH = 100;
    const isTruncated = message.length > MAX_LENGTH;
    const displayMessage = isTruncated ? `${message.substring(0, MAX_LENGTH)}...` : message;

    if (isTruncated) {
      return (
        <Tooltip title={message} arrow placement="top">
          <Typography 
            variant="caption" 
            sx={{ 
              display: 'block', 
              mt: 0.5, 
              color: '#666',
              cursor: 'pointer',
              '&:hover': {
                color: '#450839'
              }
            }}
          >
            {displayMessage}
          </Typography>
        </Tooltip>
      );
    }

    return (
      <Typography variant="caption" sx={{ display: 'block', mt: 0.5, color: '#666' }}>
        {displayMessage}
      </Typography>
    );
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
    >
      <DialogTitle sx={{
        bgcolor: '#450839',
        color: 'white',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        pb: 2,
        pt: 2
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="h6">
          InferFlow Config Details - {configData?.config_id || 'N/A'}
          </Typography>
        </Box>
        <IconButton
          aria-label="close"
          onClick={onClose}
          sx={{
            color: 'white',
            '&:hover': {
              backgroundColor: 'rgba(255, 255, 255, 0.1)'
            }
          }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers sx={{ p: 3 }}>
        {configData && (
          <Box sx={{ maxHeight: '70vh', overflow: 'auto' }}>
            {/* Basic Information Section */}
            <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold', borderBottom: '2px solid #450839', pb: 1 }}>
              Basic Information
            </Typography>

            <Grid container spacing={3} sx={{ mb: 3 }}>
              <Grid item xs={12} md={6}>
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Config ID:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {configData.config_id || 'N/A'}
                  </Typography>
                </Box>

                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Host:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {configData.host || 'N/A'}
                  </Typography>
                </Box>

                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Status:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {getStatusChip(configData.deployable_running_status)}
                  </Typography>
                </Box>
              </Grid>

              <Grid item xs={12} md={6}>
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Test Results:
                  </Typography>
                  <Box sx={{ width: '60%' }}>
                    {getTestStatusChip(configData.test_results)}
                    {renderTestMessage(configData.test_results?.message)}
                  </Box>
                </Box>

                {configData.monitoring_url && (
                  <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                    <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                      Monitoring:
                    </Typography>
                    <Typography variant="body1" sx={{ width: '60%' }}>
                      <a
                        href={configData.monitoring_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ color: '#450839', textDecoration: 'none' }}
                      >
                        View Dashboard
                      </a>
                    </Typography>
                  </Box>
                )}
              </Grid>
            </Grid>

            {/* Activity History Section */}
            <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold', borderBottom: '2px solid #450839', pb: 1 }}>
              Activity History
            </Typography>

            <Grid container spacing={3} sx={{ mb: 3 }}>
              <Grid item xs={12} md={6}>
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Created By:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {configData.created_by || 'N/A'}
                  </Typography>
                </Box>

                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Created At:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {configData.created_at ? new Date(configData.created_at).toLocaleString() : 'N/A'}
                  </Typography>
                </Box>
              </Grid>

              <Grid item xs={12} md={6}>
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Updated By:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {configData.updated_by || 'N/A'}
                  </Typography>
                </Box>

                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Updated At:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {configData.updated_at ? new Date(configData.updated_at).toLocaleString() : 'N/A'}
                  </Typography>
                </Box>
              </Grid>
            </Grid>

            {/* Configuration Details Section */}
            <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold', borderBottom: '2px solid #450839', pb: 1 }}>
              Configuration Details
            </Typography>

            {/* Config Value */}
            {configData.config_value && (
              <Paper elevation={1} sx={{ p: 2, mb: 3, bgcolor: '#f8f9fa', borderRadius: 1 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2, fontWeight: 'bold' }}>
                  CONFIG VALUE
                </Typography>
                <JsonViewer
                  data={configData.config_value}
                  title="Configuration Value"
                  defaultExpanded={true}
                  maxHeight={300}
                  enableCopy={true}
                  editable={false}
                  sx={{ mb: 2 }}
                />
              </Paper>
            )}


          </Box>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default ViewInferflowConfigDetailsModal; 