import React, { useState, useEffect } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Typography,
  Box,
  FormControl,
  FormHelperText,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Snackbar,
  Alert,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
  InputAdornment,
  IconButton,
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import ModelTrainingIcon from '@mui/icons-material/ModelTraining';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import StorageIcon from '@mui/icons-material/Storage';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import VisibilityIcon from '@mui/icons-material/Visibility';
import SearchIcon from '@mui/icons-material/Search';
import Chip from '@mui/material/Chip';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const ModelApproval = () => {
  const [modelRequests, setModelRequests] = useState([]);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [approvalComments, setApprovalComments] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['PENDING', 'APPROVED', 'REJECTED']);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [mqIdTopicsMapping, setMqIdTopicsMapping] = useState([]);
  const [approvalMqId, setApprovalMqId] = useState('');
  const [approvalTopicName, setApprovalTopicName] = useState('');
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();

  useEffect(() => {
    fetchModelRequests();
  }, []);

  useEffect(() => {
    const fetchMqIdTopics = async () => {
      try {
        const response = await embeddingPlatformAPI.getMQIdTopics();
        if (response?.mappings) {
          setMqIdTopicsMapping(response.mappings);
        } else {
          setMqIdTopicsMapping([]);
        }
      } catch (err) {
        console.error('Error fetching Kafka ID topics:', err);
        setMqIdTopicsMapping([]);
      }
    };
    fetchMqIdTopics();
  }, []);

  const fetchModelRequests = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getModelRequests();
      
      if (response.model_requests) {
        setModelRequests(response.model_requests);
      } else {
        setModelRequests([]);
      }
    } catch (error) {
      console.error('Error fetching model requests:', error);
      setError('Failed to load model requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const filteredRequests = useTableFilter({
    data: modelRequests,
    searchQuery,
    selectedStatuses,
    searchFields: (request) => [
      request.request_id,
      request.payload?.model,
      request.payload?.entity,
      request.payload?.model_type,
      request.created_by,
      request.status,
    ],
    sortField: 'created_at',
    sortOrder: 'desc',
  });

  const handleViewRequest = (request) => {
    setSelectedRequest(request);
    const payload = request?.payload ?? {};
    const mqId = payload.mq_id ?? '';
    setApprovalMqId(mqId);
    const mapping = mqId ? mqIdTopicsMapping.find(m => m.mq_id === parseInt(mqId, 10)) : null;
    setApprovalTopicName(mapping?.topic ?? payload.topic_name ?? '');
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedRequest(null);
    setApprovalMqId('');
    setApprovalTopicName('');
  };

  const handleApprovalMqIdChange = (mqId) => {
    setApprovalMqId(mqId);
    const mapping = mqIdTopicsMapping.find(m => m.mq_id === parseInt(mqId, 10));
    setApprovalTopicName(mapping?.topic ?? '');
  };

  const handleApprovalSubmit = async (decision) => {
    if (decision === 'REJECTED' && !approvalComments.trim()) {
      return;
    }

    try {
      setLoading(true);
      const payload = {
        request_id: selectedRequest.request_id,
        admin_id: user?.email || 'admin@example.com',
        approval_decision: decision,
        approval_comments: approvalComments
      };
      if (decision === 'APPROVED') {
        payload.admin_mq_id = approvalMqId ? parseInt(approvalMqId, 10) : undefined;
        payload.admin_topic_name = approvalTopicName || undefined;
      }
      const response = await embeddingPlatformAPI.approveModel(payload);
      
      const message = response.data?.message || response.message || 'Model request processed successfully';
      showNotification(message, 'success');
      fetchModelRequests();
      handleCloseViewModal();
      handleRejectModalClose();
      
      // Trigger update event for other components
      window.dispatchEvent(new CustomEvent('modelApprovalUpdate'));
    } catch (error) {
      console.error('Error processing model request:', error);
      showNotification(error.message || 'Failed to process model request', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleRejectModalClose = () => {
    setShowRejectModal(false);
    setApprovalComments('');
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Paper elevation={0} sx={{ width: '100%', height: '100vh', padding: '2rem', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h6">Model Approval</Typography>
          <Typography variant="body1" color="text.secondary">
            Review and approve/reject model registration requests
          </Typography>
        </Box>
      </Box>

      {/* Search */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '1rem',
          mb: 2
        }}
      >
        <TextField
          label="Search Model Requests"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
              </InputAdornment>
            ),
          }}
          size="small"
          fullWidth
        />
      </Box>

      {/* Table */}
      <TableContainer 
        component={Paper} 
        elevation={3} 
        sx={{ 
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1000,
            borderCollapse: 'separate',
            borderSpacing: 0,
          },
          '& .MuiTableHead-root': {
            position: 'sticky',
            top: 0,
            zIndex: 1,
            backgroundColor: '#E6EBF2',
          }
        }}
      >
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Request ID
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Model Name
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Entity
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Model Type
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Created By
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                <StatusFilterHeader
                  selectedStatuses={selectedStatuses}
                  onStatusChange={handleStatusChange}
                />
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                }}
              >
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredRequests.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {modelRequests.length === 0 ? 'No model requests pending approval' : 'No model requests match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredRequests.map((request, index) => (
                <TableRow key={request.request_id || index} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.request_id || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.payload?.model || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.payload?.entity || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Chip 
                      label={request.payload?.model_type || 'N/A'}
                      size="small"
                      sx={{ 
                        backgroundColor: request.payload?.model_type === 'DELTA' ? '#e8f5e8' : '#fff3e0',
                        color: request.payload?.model_type === 'DELTA' ? '#2e7d32' : '#f57c00',
                        fontWeight: 600
                      }}
                    />
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.created_by || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <StatusChip status={request.status} />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      <IconButton size="small" onClick={() => handleViewRequest(request)} title="View Details">
                        <VisibilityIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>


      {/* View Model Request Details Modal */}
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedRequest}
        config={{
          title: 'Model Request Details',
          icon: ModelTrainingIcon,
          sections: [
            {
              title: 'Request Information',
              icon: InfoIcon,
              layout: 'grid',
              fields: [
                { label: 'Request ID', key: 'request_id', type: 'monospace' },
                { label: 'Status', key: 'status', type: 'status' }
              ]
            },
            {
              title: 'Model Configuration',
              icon: SmartToyIcon,
              backgroundColor: 'rgba(255, 152, 0, 0.02)',
              borderColor: 'rgba(255, 152, 0, 0.1)',
              layout: 'grid',
              fields: [
                { label: 'Model Name', key: 'payload.model' },
                { label: 'Entity', key: 'payload.entity' }
              ]
            },
            {
              title: 'Kafka ID & Topic',
              icon: SettingsIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              render: (data) => {
                if (data?.status !== 'PENDING') {
                  return (
                    <Grid container spacing={2}>
                      <Grid item xs={12} sm={6}>
                        <Typography variant="caption" color="text.secondary" display="block">Kafka ID</Typography>
                        <Typography variant="body2">{data?.payload?.mq_id ?? '—'}</Typography>
                      </Grid>
                      <Grid item xs={12} sm={6}>
                        <Typography variant="caption" color="text.secondary" display="block">Topic Name</Typography>
                        <Typography variant="body2">{data?.payload?.topic_name ?? '—'}</Typography>
                      </Grid>
                    </Grid>
                  );
                }
                return (
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <FormControl fullWidth size="small">
                        <InputLabel>Kafka ID</InputLabel>
                        <Select
                          value={approvalMqId || ''}
                          onChange={(e) => handleApprovalMqIdChange(e.target.value)}
                          label="Kafka ID"
                        >
                          {mqIdTopicsMapping.map((mapping) => (
                            <MenuItem key={mapping.mq_id} value={mapping.mq_id}>
                              {mapping.mq_id}
                            </MenuItem>
                          ))}
                        </Select>
                        <FormHelperText>Required when approving</FormHelperText>
                      </FormControl>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Topic Name"
                        value={approvalTopicName}
                        disabled
                        helperText="Auto-populated from selected Kafka ID"
                      />
                    </Grid>
                  </Grid>
                );
              }
            },
            {
              title: 'Full Configuration',
              icon: SettingsIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              render: (data) => (
                <TextField
                  multiline
                  rows={8}
                  value={data?.payload ? JSON.stringify(data.payload, null, 2) : '{}'}
                  variant="outlined"
                  fullWidth
                  InputProps={{ readOnly: true, style: { fontFamily: 'monospace', fontSize: '0.875rem' } }}
                  sx={{ backgroundColor: '#fafafa' }}
                />
              )
            },
            {
              title: 'Request Metadata',
              icon: PersonIcon,
              backgroundColor: 'rgba(158, 158, 158, 0.02)',
              borderColor: 'rgba(158, 158, 158, 0.1)',
              layout: 'grid',
              fields: [
                { label: 'Created By', key: 'created_by' },
                { label: 'Created At', key: 'created_at', type: 'date' }
              ]
            },
            {
              title: 'Approval Decision',
              icon: CheckCircleIcon,
              backgroundColor: 'rgba(76, 175, 80, 0.02)',
              borderColor: 'rgba(76, 175, 80, 0.1)',
              render: (data) => {
                if (data?.status !== 'PENDING') return null;
                return (
                  <TextField
                    label="Approval/Rejection Comments"
                    multiline
                    rows={3}
                    value={approvalComments}
                    onChange={(e) => setApprovalComments(e.target.value)}
                    variant="outlined"
                    fullWidth
                    placeholder="Enter comments for your decision (required for rejection)..."
                  />
                );
              }
            }
          ],
          actions: (request, onClose) => (
            <>
              <Button onClick={onClose} sx={{ color: '#522b4a', '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' } }}>
                Close
              </Button>
              {request?.status === 'PENDING' && (
                <>
                  <Button
                    variant="contained"
                    startIcon={<CheckCircleIcon />}
                    onClick={() => handleApprovalSubmit('APPROVED')}
                    disabled={loading || !approvalMqId}
                    sx={{ backgroundColor: '#2e7d32', color: 'white', '&:hover': { backgroundColor: '#1b5e20' }, '&:disabled': { backgroundColor: '#c8e6c9' } }}
                  >
                    {loading ? 'Processing...' : 'Approve'}
                  </Button>
                  <Button
                    variant="contained"
                    startIcon={<CancelIcon />}
                    onClick={() => setShowRejectModal(true)}
                    disabled={loading}
                    sx={{ backgroundColor: '#d32f2f', color: 'white', '&:hover': { backgroundColor: '#b71c1c' }, '&:disabled': { backgroundColor: '#ffcdd2' } }}
                  >
                    Reject
                  </Button>
                </>
              )}
            </>
          )
        }}
      />

      {/* Rejection Modal */}
      <Dialog open={showRejectModal} onClose={handleRejectModalClose} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CancelIcon sx={{ color: '#d32f2f' }} />
            Confirm Rejection
          </Box>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <Typography variant="body1" sx={{ mb: 2 }}>
              Are you sure you want to reject this model request?
            </Typography>
            
            <TextField
              label="Rejection Reason (Required)"
              multiline
              rows={4}
              value={approvalComments}
              onChange={(e) => setApprovalComments(e.target.value)}
              variant="outlined"
              fullWidth
              placeholder="Please provide a reason for rejecting this request..."
              helperText="A detailed reason helps the requester understand and improve their submission."
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2, backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button 
            onClick={handleRejectModalClose}
            sx={{ 
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            startIcon={<CancelIcon />}
            onClick={() => handleApprovalSubmit('REJECTED')}
            disabled={loading || !approvalComments.trim()}
            sx={{
              backgroundColor: '#d32f2f',
              color: 'white',
              '&:hover': { backgroundColor: '#b71c1c' },
              '&:disabled': { backgroundColor: '#ffcdd2' }
            }}
          >
            {loading ? 'Processing...' : 'Confirm Rejection'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Notification Snackbar */}
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={closeNotification}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={closeNotification}
          severity={notification.severity}
          sx={{ width: '100%' }}
        >
          {notification.message}
        </Alert>
      </Snackbar>

    </Paper>
  );
};

export default ModelApproval;