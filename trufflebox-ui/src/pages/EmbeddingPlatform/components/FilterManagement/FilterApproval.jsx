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
  FormHelperText,
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
  Chip,
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import FilterListIcon from '@mui/icons-material/FilterList';
import InfoIcon from '@mui/icons-material/Info';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CategoryIcon from '@mui/icons-material/Category';
import VisibilityIcon from '@mui/icons-material/Visibility';
import SearchIcon from '@mui/icons-material/Search';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const FilterApproval = () => {
  const [filterRequests, setFilterRequests] = useState([]);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [approvalComments, setApprovalComments] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['PENDING', 'APPROVED', 'REJECTED']);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();

  useEffect(() => {
    fetchFilterRequests();
  }, []);

  const fetchFilterRequests = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getFilterRequests();
      
      if (response.filter_requests) {
        setFilterRequests(response.filter_requests);
      } else {
        setFilterRequests([]);
      }
    } catch (error) {
      console.error('Error fetching filter requests:', error);
      setError('Failed to load filter requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const filteredRequests = useTableFilter({
    data: filterRequests,
    searchQuery,
    selectedStatuses,
    searchFields: (request) => [
      request.request_id,
      request.payload?.entity,
      request.payload?.filter?.column_name,
      request.payload?.filter?.filter_value,
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
      
      const response = await embeddingPlatformAPI.approveFilter(payload);
      
      const message = response.data?.message || response.message || 'Filter request processed successfully';
      showNotification(message, 'success');
      fetchFilterRequests();
      handleCloseViewModal();
      handleRejectModalClose();
      
      // Trigger update event for other components
      window.dispatchEvent(new CustomEvent('filterApprovalUpdate'));
    } catch (error) {
      console.error('Error processing filter request:', error);
      showNotification(error.message || 'Failed to process filter request', 'error');
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
        <Typography variant="h6">Filter Approval</Typography>
        <Typography variant="body1" color="text.secondary">
          Review and approve/reject filter registration requests
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
        label="Search Filter Requests"
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
              Column Name
            </TableCell>
            <TableCell
              sx={{
                backgroundColor: '#E6EBF2',
                fontWeight: 'bold',
                color: '#031022',
                borderRight: '1px solid rgba(224, 224, 224, 1)',
              }}
            >
              Filter Value
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
              <StatusFilterHeader selectedStatuses={selectedStatuses} onStatusChange={handleStatusChange} />
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
                  {filterRequests.length === 0 ? 'No filter requests pending approval' : 'No filter requests match your search'}
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
                  <Chip 
                    label={request.payload?.entity || 'N/A'}
                    size="small"
                    sx={{ 
                      backgroundColor: '#e3f2fd',
                      color: '#1976d2',
                      fontWeight: 600
                    }}
                  />
                </TableCell>
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
                    {request.payload?.filter?.column_name || 'N/A'}
                  </Typography>
                </TableCell>
                <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                  <Chip
                    label={request.payload?.filter?.filter_value || 'N/A'}
                    size="small"
                    sx={{
                      backgroundColor: '#e8f5e8',
                      color: '#2e7d32',
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

    {/* View Filter Request Details Modal */}
    <ViewDetailModal
      open={showViewModal}
      onClose={handleCloseViewModal}
      data={selectedRequest}
      config={{
        title: 'Filter Request Details',
        icon: FilterListIcon,
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
            title: 'Filter Configuration',
            icon: CategoryIcon,
            backgroundColor: 'rgba(25, 118, 210, 0.02)',
            borderColor: 'rgba(25, 118, 210, 0.1)',
            layout: 'grid',
            fields: [
              { label: 'Entity', key: 'payload.entity' },
              { label: 'Column Name', key: 'payload.filter.column_name', type: 'monospace' },
              { label: 'Filter Value', key: 'payload.filter.filter_value', type: 'chip' },
              { label: 'Default Value', key: 'payload.filter.default_value' }
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
              { label: 'Created At', key: 'created_at', type: 'date' },
              { label: 'Reason', key: 'reason' }
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
                  disabled={loading}
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
            Are you sure you want to reject this filter request?
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

export default FilterApproval;