import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Alert,
  Snackbar,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  CircularProgress,
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import VisibilityIcon from '@mui/icons-material/Visibility';
import SearchIcon from '@mui/icons-material/Search';
import StorageIcon from '@mui/icons-material/Storage';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import PersonIcon from '@mui/icons-material/Person';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { 
  StatusChip, 
  StatusFilterHeader, 
  ViewDetailModal,
  useTableFilter,
  useStatusFilter
} from '../shared';
const StoreApproval = () => {
  const [showViewModal, setShowViewModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [approvalComments, setApprovalComments] = useState('');
  const [loading, setLoading] = useState(true);
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });
  const [storeRequests, setStoreRequests] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');
  const { user } = useAuth();
  
  // Use shared status filter hook - Initialize with PENDING only for approval page
  const { selectedStatuses, handleStatusChange } = useStatusFilter(['PENDING']);

  const fetchStoreRequests = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      
      const response = await embeddingPlatformAPI.getStoreRequests();
      
      if (response.store_requests) {
        const requests = (response.store_requests || []).filter(request => request != null);
        
        setStoreRequests(requests);
      } else {
        setStoreRequests([]);
      }
    } catch (error) {
      console.error('Error fetching store requests:', error);
      setError('Failed to load store requests. Please refresh the page.');
      showNotification('Failed to load store requests. Please refresh the page.', 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStoreRequests();
  }, [fetchStoreRequests]);

  // Use shared table filter hook
  const filteredRequests = useTableFilter({
    data: storeRequests,
    searchQuery,
    selectedStatuses,
    searchFields: (request) => [
      request.request_id,
      request.payload?.conf_id,
      request.payload?.db,
      request.payload?.embeddings_table,
      request.payload?.aggregator_table,
      request.created_by,
      request.status
    ],
    sortField: 'created_at',
    sortOrder: 'desc'
  });


  const handleViewRequest = (request) => {
    setSelectedRequest(request);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedRequest(null);
  };


  const handleRejectModalClose = () => {
    setShowRejectModal(false);
    setSelectedRequest(null);
    setApprovalComments('');
  };

  const handleApprovalSubmit = async (decision) => {
    if (!selectedRequest) return;

    // For rejection, require comments
    if (decision === 'REJECTED' && !approvalComments.trim()) {
      showNotification('Please provide a reason for rejection.', 'error');
      return;
    }

    try {
      setLoading(true);

      const payload = {
        request_id: selectedRequest.request_id,
        approval_decision: decision,
        approval_comments: decision === 'REJECTED' ? approvalComments : '',
        admin_id: user.email,
      };

      const response = await embeddingPlatformAPI.approveStore(payload);
      
      const message = response.data?.message || response.message || 'Store request processed successfully';
      showNotification(message, 'success');
      fetchStoreRequests(); // Refresh the requests
      handleCloseViewModal();
      handleRejectModalClose();
      
      // Trigger update event for other components
      window.dispatchEvent(new CustomEvent('storeApprovalUpdate'));
    } catch (error) {
      console.error('Error processing store request:', error);
      showNotification(error.message || 'Failed to process store request', 'error');
    } finally {
      setLoading(false);
    }
  };

  const showNotification = (message, severity) => {
    setNotification({
      open: true,
      message,
      severity
    });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({
      ...prev,
      open: false
    }));
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
          <Typography variant="h6">Store Approval</Typography>
          <Typography variant="body1" color="text.secondary">
            Review and approve/reject store registration requests
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
          label="Search Store Requests"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
          size="small"
          fullWidth
        />
      </Box>

      {/* Store Requests Table */}
      <TableContainer
        component={Paper}
        elevation={3}
        sx={{
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1200,
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
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Req ID
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Conf ID
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                DB Type
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Embeddings Table
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Aggregator Table
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
                <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {storeRequests.length === 0 ? 'No store requests pending approval' : 'No store requests match your search'}
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
                    {request.payload?.conf_id || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.payload?.db || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.payload?.embeddings_table || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.payload?.aggregator_table || 'N/A'}
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

      {/* View Store Request Details Modal */}
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedRequest}
        config={{
          title: 'Store Request Details',
          icon: StorageIcon,
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
              title: 'Store Configuration',
              icon: SettingsIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              fields: [
                { label: 'Configuration ID', key: 'payload.conf_id', type: 'monospace' },
                { label: 'Database Type', key: 'payload.db', type: 'chip' },
                { label: 'Embeddings Table', key: 'payload.embeddings_table', type: 'monospace' },
                { label: 'Aggregator Table', key: 'payload.aggregator_table', type: 'monospace' }
              ]
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
            }
          ],
          actions: (request) => {
            if (request?.status === 'PENDING') {
              return (
                <>
                  <Button 
                    onClick={handleCloseViewModal}
                    sx={{ 
                      color: '#522b4a',
                      '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
                    }}
                  >
                    Close
                  </Button>
                  <Button 
                    variant="contained" 
                    startIcon={<CheckCircleIcon />}
                    onClick={() => handleApprovalSubmit('APPROVED')}
                    disabled={loading}
                    sx={{
                      backgroundColor: '#2e7d32',
                      color: 'white',
                      '&:hover': { backgroundColor: '#1b5e20' },
                      '&:disabled': { backgroundColor: '#c8e6c9' }
                    }}
                  >
                    {loading ? 'Processing...' : 'Approve'}
                  </Button>
                  <Button 
                    variant="contained" 
                    startIcon={<CancelIcon />}
                    onClick={() => setShowRejectModal(true)}
                    disabled={loading}
                    sx={{
                      backgroundColor: '#d32f2f',
                      color: 'white',
                      '&:hover': { backgroundColor: '#b71c1c' },
                      '&:disabled': { backgroundColor: '#ffcdd2' }
                    }}
                  >
                    Reject
                  </Button>
                </>
              );
            }
            return (
              <Button 
                onClick={handleCloseViewModal}
                sx={{ 
                  color: '#522b4a',
                  '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
                }}
              >
                Close
              </Button>
            );
          }
        }}
      />

      {/* Rejection Reason Modal */}
      <Dialog open={showRejectModal} onClose={handleRejectModalClose} maxWidth="sm" fullWidth>
        <DialogTitle>Reject Store Request</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Please provide a reason for rejecting this store request:
          </Typography>
          <TextField
            autoFocus
            margin="dense"
            label="Rejection Reason"
            fullWidth
            multiline
            rows={3}
            variant="outlined"
            value={approvalComments}
            onChange={(e) => setApprovalComments(e.target.value)}
            placeholder="Enter the reason for rejection..."
          />
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

      {/* Toast Notifications */}
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleCloseNotification}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={handleCloseNotification}
          severity={notification.severity}
          sx={{ width: '100%' }}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </Paper>
  );
};

export default StoreApproval;