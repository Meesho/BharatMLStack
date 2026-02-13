import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  TextField,
  CircularProgress,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
  IconButton,
  Snackbar
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import ScheduleIcon from '@mui/icons-material/Schedule';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import PersonIcon from '@mui/icons-material/Person';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const JobFrequencyApproval = () => {
  const [frequencyRequests, setFrequencyRequests] = useState([]);
  const [showViewModal, setShowViewModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [approvalComments, setApprovalComments] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['PENDING', 'APPROVED', 'REJECTED']);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const response = await embeddingPlatformAPI.getJobFrequencyRequests();
      
      if (response.job_frequency_requests) {
        setFrequencyRequests(response.job_frequency_requests || []);
      }
    } catch (error) {
      console.error('Error fetching job frequency requests:', error);
      setError('Failed to load job frequency requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  // Generate description from frequency pattern
  const generateDescription = (frequency) => {
    if (!frequency) return 'Unknown frequency';
    
    const match = frequency.match(/FREQ_(\d+)([HDWM])/);
    if (!match) return frequency;
    
    const [, number, unit] = match;
    const units = {
      'H': number === '1' ? 'hour' : 'hours',
      'D': number === '1' ? 'day' : 'days',
      'W': number === '1' ? 'week' : 'weeks',
      'M': number === '1' ? 'month' : 'months'
    };
    
    const unitName = units[unit] || unit;
    return `${number === '1' ? unitName.charAt(0).toUpperCase() + unitName.slice(1) : `${number}-${unitName}`} frequency - runs every ${number} ${unitName}`;
  };

  const filteredRequests = useTableFilter({
    data: frequencyRequests,
    searchQuery,
    selectedStatuses,
    searchFields: (request) => [
      request.request_id,
      request.payload?.job_frequency,
      request.reason,
      request.created_by,
      request.status,
    ],
    sortField: 'created_at',
    sortOrder: 'desc',
  });

  const handleViewRequest = (request) => {
    setSelectedRequest(request);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedRequest(null);
    setApprovalComments('');
  };

  const handleRejectModalOpen = () => {
    setShowRejectModal(true);
  };

  const handleRejectModalClose = () => {
    setShowRejectModal(false);
    setApprovalComments('');
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
      
      const response = await embeddingPlatformAPI.approveJobFrequency(payload);
      
      const message = response.data?.message || response.message || 'Job frequency request processed successfully';
      showNotification(message, 'success');
      
      handleCloseViewModal();
      handleRejectModalClose();
      fetchData();
    } catch (error) {
      console.error('Error processing job frequency approval:', error);
      showNotification(error.message || 'Failed to process job frequency request', 'error');
    } finally {
      setLoading(false);
    }
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
          <Typography variant="h6">Job Frequency Approval</Typography>
          <Typography variant="body1" color="text.secondary">
            Review and approve job frequency registration requests
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
          label="Search Job Frequency Requests"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: <SearchIcon sx={{ color: 'action.active', mr: 1 }} />,
          }}
          sx={{ width: 350 }}
          size="small"
        />
        <Typography variant="body2" color="text.secondary">
          {filteredRequests.length} of {frequencyRequests.length} requests
        </Typography>
      </Box>

      <Alert severity="info" sx={{ mb: 2 }}>
        <Typography variant="body2">
          <strong>Admin Note:</strong> Approved job frequencies will be automatically configured in the scheduler system and made available for model training configurations.
        </Typography>
      </Alert>

      <TableContainer component={Paper} sx={{ flexGrow: 1, border: '1px solid rgba(224, 224, 224, 1)' }}>
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Request ID
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Job Frequency
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Description
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Created By
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                  position: 'relative'
                }}
              >
                <StatusFilterHeader selectedStatuses={selectedStatuses} onStatusChange={handleStatusChange} />
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022' }}>
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredRequests.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {frequencyRequests.length === 0 ? 'No job frequency requests pending approval' : 'No job frequency requests match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredRequests.map((request, index) => (
                <TableRow key={request.request_id || index} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Typography
                      variant="body2"
                      sx={{
                        fontFamily: 'monospace',
                        backgroundColor: '#f5f5f5',
                        padding: '2px 6px',
                        borderRadius: '4px',
                        fontSize: '0.875rem',
                        display: 'inline-block'
                      }}
                    >
                      {request.request_id}
                    </Typography>
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Typography
                      variant="body2"
                      sx={{
                        fontFamily: 'monospace',
                        backgroundColor: '#e3f2fd',
                        padding: '2px 6px',
                        borderRadius: '4px',
                        fontSize: '0.875rem',
                        color: '#1976d2',
                        display: 'inline-block'
                      }}
                    >
                      {request.payload?.job_frequency}
                    </Typography>
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {generateDescription(request.payload?.job_frequency)}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.created_by || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <StatusChip status={request.status} />
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => handleViewRequest(request)}
                      sx={{ color: '#522b4a' }}
                    >
                      <VisibilityIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* View Job Frequency Request Details Modal */}
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedRequest}
        config={{
          title: 'Job Frequency Request Details',
          icon: ScheduleIcon,
          sections: [
            {
              title: 'Request Information',
              icon: PersonIcon,
              layout: 'grid',
              fields: [
                { label: 'Request ID', key: 'request_id', type: 'monospace' },
                { label: 'Status', key: 'status', type: 'status' },
                { label: 'Created By', key: 'created_by' },
                { label: 'Created At', key: 'created_at', type: 'date' }
              ]
            },
            {
              title: 'Job Frequency Configuration',
              icon: ScheduleIcon,
              backgroundColor: 'rgba(33, 150, 243, 0.02)',
              borderColor: 'rgba(33, 150, 243, 0.1)',
              render: (data) => (
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Frequency ID</Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#e3f2fd', padding: '4px 8px', borderRadius: '4px', color: '#1976d2', display: 'inline-block', mt: 0.5 }}>
                      {data?.payload?.job_frequency || 'N/A'}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Description</Typography>
                    <Typography variant="body1" sx={{ mt: 0.5 }}>{generateDescription(data?.payload?.job_frequency)}</Typography>
                  </Box>
                  <Box sx={{ gridColumn: 'span 2' }}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Reason</Typography>
                    <Typography variant="body1" sx={{ mt: 0.5 }}>{data?.reason || 'N/A'}</Typography>
                  </Box>
                </Box>
              )
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
                    onClick={() => handleApprovalSubmit('APPROVED')}
                    variant="contained"
                    startIcon={<CheckCircleIcon />}
                    disabled={loading}
                    sx={{ backgroundColor: '#2e7d32', '&:hover': { backgroundColor: '#1b5e20' } }}
                  >
                    {loading ? 'Processing...' : 'Approve'}
                  </Button>
                  <Button
                    onClick={handleRejectModalOpen}
                    variant="contained"
                    startIcon={<CancelIcon />}
                    disabled={loading}
                    sx={{ backgroundColor: '#d32f2f', '&:hover': { backgroundColor: '#b71c1c' } }}
                  >
                    Reject
                  </Button>
                </>
              )}
            </>
          )
        }}
      />

      {/* Reject Confirmation Modal */}
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
              Are you sure you want to reject this job frequency request?
            </Typography>
            <TextField
              label="Rejection Reason (Required)"
              multiline
              rows={3}
              value={approvalComments}
              onChange={(e) => setApprovalComments(e.target.value)}
              variant="outlined"
              fullWidth
              required
              error={!approvalComments.trim()}
              helperText={!approvalComments.trim() ? "Please provide a reason for rejection" : ""}
              placeholder="Explain why this job frequency request is being rejected..."
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
            onClick={() => handleApprovalSubmit('REJECTED')}
            variant="contained"
            startIcon={<CancelIcon />}
            disabled={!approvalComments.trim() || loading}
            sx={{
              backgroundColor: '#d32f2f',
              '&:hover': { backgroundColor: '#b71c1c' }
            }}
          >
            {loading ? 'Processing...' : 'Confirm Rejection'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Notifications */}
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

export default JobFrequencyApproval;