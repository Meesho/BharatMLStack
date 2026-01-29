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
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
  Snackbar,
  Grid
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import VisibilityIcon from '@mui/icons-material/Visibility';
import FilterListIcon from '@mui/icons-material/FilterList';
import CloudIcon from '@mui/icons-material/Cloud';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunch';
import LaunchIcon from '@mui/icons-material/Launch';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const DeploymentApproval = () => {
  const [deploymentRequests, setDeploymentRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['APPROVED', 'PENDING', 'REJECTED']);
  const [error, setError] = useState('');
  const [showViewModal, setShowViewModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [approvalComments, setApprovalComments] = useState('');
  const { user } = useAuth();
  const [notification, setNotification] = useState({ open: false, message: "", severity: "success" });
  const [processingRequest, setProcessingRequest] = useState(null);

  const statusOptions = [
    { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
  ];

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const onboardingResponse = await embeddingPlatformAPI.getAllVariantOnboardingRequests().catch(() => ({ variant_onboarding_requests: [] }));
      
      // Transform requests to deployment format (payload already normalized by API)
      const allRequests = (onboardingResponse.variant_onboarding_requests || []).map(req => ({
        ...req,
        request_type: 'variant_onboarding'
      }));
      
      setDeploymentRequests(allRequests);
    } catch (error) {
      console.error('Error fetching deployment requests:', error);
      setError('Failed to load deployment requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  // Search and filter logic
  const filteredRequests = deploymentRequests.filter(request => {
    const searchTerm = searchQuery.toLowerCase();
    const matchesSearch = !searchQuery || 
      request.request_id?.toString().toLowerCase().includes(searchTerm) ||
      request.requestor?.toLowerCase().includes(searchTerm) ||
      request.payload?.dns_subdomain?.toLowerCase().includes(searchTerm) ||
      request.payload?.entity?.toLowerCase().includes(searchTerm) ||
      request.payload?.model?.toLowerCase().includes(searchTerm) ||
      request.payload?.variant?.toLowerCase().includes(searchTerm) ||
      request.request_type?.toLowerCase().includes(searchTerm);
    
    const matchesStatus = selectedStatuses.includes(request.status || 'PENDING');
    
    return matchesSearch && matchesStatus;
  });

  // Status Column Header with filtering
  const StatusColumnHeader = () => {
    const [anchorEl, setAnchorEl] = useState(null);
    
    const handleClick = (event) => {
      setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
      setAnchorEl(null);
    };

    const handleStatusToggle = (status) => {
      setSelectedStatuses(prev => 
        prev.includes(status) 
          ? prev.filter(s => s !== status)
          : [...prev, status]
      );
    };

    const handleSelectAll = () => {
      setSelectedStatuses(statusOptions.map(option => option.value));
    };

    const handleClearAll = () => {
      setSelectedStatuses([]);
    };

    const open = Boolean(anchorEl);

    return (
      <>
        <Box 
          sx={{ 
            display: 'flex', 
            flexDirection: 'column',
            alignItems: 'flex-start',
            width: '100%'
          }}
        >
          <Box 
            sx={{ 
              display: 'flex', 
              alignItems: 'center', 
              cursor: 'pointer',
              '&:hover': { backgroundColor: 'rgba(0, 0, 0, 0.04)' },
              borderRadius: 1,
              p: 0.5
            }}
            onClick={handleClick}
          >
            <Typography sx={{ fontWeight: 'bold', color: '#031022' }}>
              Status
            </Typography>
            <FilterListIcon 
              sx={{ 
                ml: 0.5, 
                fontSize: 16,
                color: selectedStatuses.length < statusOptions.length ? '#1976d2' : '#666'
              }} 
            />
            {selectedStatuses.length > 0 && selectedStatuses.length < statusOptions.length && (
              <Box
                sx={{
                  position: 'absolute',
                  top: 2,
                  right: 2,
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  backgroundColor: '#1976d2',
                }}
              />
            )}
          </Box>

          {selectedStatuses.length > 0 && (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5, maxWidth: '100%' }}>
              {selectedStatuses.slice(0, 2).map((status) => {
                const option = statusOptions.find(opt => opt.value === status);
                return option ? (
                  <Chip
                    key={status}
                    label={option.label}
                    size="small"
                    sx={{
                      backgroundColor: option.color,
                      color: option.textColor,
                      fontWeight: 'bold',
                      fontSize: '0.65rem',
                      height: 18,
                      '& .MuiChip-label': { px: 0.5 }
                    }}
                  />
                ) : null;
              })}
              {selectedStatuses.length > 2 && (
                <Chip
                  label={`+${selectedStatuses.length - 2}`}
                  size="small"
                  sx={{
                    backgroundColor: '#f5f5f5',
                    color: '#666',
                    fontWeight: 'bold',
                    fontSize: '0.65rem',
                    height: 18,
                    '& .MuiChip-label': { px: 0.5 }
                  }}
                />
              )}
            </Box>
          )}
        </Box>
        <Popover
          open={open}
          anchorEl={anchorEl}
          onClose={handleClose}
          anchorOrigin={{
            vertical: 'bottom',
            horizontal: 'left',
          }}
          transformOrigin={{
            vertical: 'top',
            horizontal: 'left',
          }}
          PaperProps={{
            sx: {
              width: 200,
              maxHeight: 300,
              overflow: 'auto'
            }
          }}
        >
          <Box sx={{ p: 1 }}>
            <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
              <Button 
                size="small" 
                onClick={handleSelectAll}
                sx={{ textTransform: 'none', fontSize: '0.75rem', minWidth: 'auto' }}
              >
                All
              </Button>
              <Button 
                size="small" 
                onClick={handleClearAll}
                sx={{ textTransform: 'none', fontSize: '0.75rem', minWidth: 'auto' }}
              >
                Clear
              </Button>
            </Box>
            <Divider sx={{ mb: 1 }} />
            <List dense>
              {statusOptions.map((option) => (
                <ListItem key={option.value} disablePadding>
                  <ListItemButton onClick={() => handleStatusToggle(option.value)} sx={{ py: 0.5 }}>
                    <Checkbox
                      edge="start"
                      checked={selectedStatuses.includes(option.value)}
                      size="small"
                      sx={{ mr: 1 }}
                    />
                    <Chip
                      label={option.label}
                      size="small"
                      sx={{
                        backgroundColor: option.color,
                        color: option.textColor,
                        fontWeight: 'bold',
                        minWidth: '80px'
                      }}
                    />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          </Box>
        </Popover>
      </>
    );
  };

  const handleViewRequest = (request) => {
    setSelectedRequest(request);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedRequest(null);
  };

  const handleApproveReject = async (decision) => {
    if (!selectedRequest) return;

    try {
      setProcessingRequest(selectedRequest.request_id);
      
      // Determine the appropriate API call based on request type
      let apiCall;
      const payload = {
        request_id: selectedRequest.request_id,
        admin_id: user?.email || 'admin@example.com',
        approval_decision: decision,
        approval_comments: approvalComments || (decision === 'APPROVED' ? 'Request approved' : 'Request rejected'),
      };

      if (selectedRequest.request_type === 'variant_onboarding') {
        apiCall = () => embeddingPlatformAPI.approveVariantOnboarding(payload);
      } else {
        throw new Error('Unsupported request type');
      }

      await apiCall();
      
      setNotification({
        open: true,
        message: `Deployment request ${decision.toLowerCase()} successfully!`,
        severity: 'success'
      });
      
      fetchData(); // Refresh the data
      setShowViewModal(false);
      setShowRejectModal(false);
      setApprovalComments('');
      
    } catch (error) {
      console.error(`Error ${decision.toLowerCase()}ing request:`, error);
      setNotification({
        open: true,
        message: error.message || `Failed to ${decision.toLowerCase()} request. Please try again.`,
        severity: 'error'
      });
    } finally {
      setProcessingRequest(null);
    }
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({ ...prev, open: false }));
  };

  const getRequestTypeIcon = (type) => {
    switch (type) {
      case 'cluster_creation': return <CloudIcon />;
      case 'variant_promotion': return <RocketLaunchIcon />;
      case 'variant_onboarding': return <LaunchIcon />;
      default: return <LaunchIcon />;
    }
  };

  const getRequestTypeLabel = (type) => {
    switch (type) {
      case 'cluster_creation': return 'Cluster Creation';
      case 'variant_promotion': return 'Variant Promotion';
      case 'variant_onboarding': return 'Variant Onboarding';
      default: return 'Unknown';
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
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
        <Button onClick={fetchData} variant="contained">
          Retry
        </Button>
      </Box>
    );
  }

  return (
    <Paper elevation={0} sx={{ width: '100%', height: '100vh', padding: '2rem', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h6">Deployment Approval</Typography>
          <Typography variant="body1" color="text.secondary">
            Review and approve deployment operation requests
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
          label="Search Deployment Requests"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: <SearchIcon sx={{ color: 'action.active', mr: 1 }} />,
          }}
          sx={{ width: 350 }}
          size="small"
        />
        <Typography variant="body2" color="text.secondary">
          {filteredRequests.length} of {deploymentRequests.length} requests
        </Typography>
      </Box>

      <Alert severity="info" sx={{ mb: 2 }}>
        <Typography variant="body2">
          <strong>Admin Note:</strong> Approved deployment requests will initiate infrastructure provisioning and configuration changes in production systems.
        </Typography>
      </Alert>

      {/* Table */}
      <TableContainer component={Paper} sx={{ flexGrow: 1, border: '1px solid rgba(224, 224, 224, 1)' }}>
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Request ID
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Type
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Details
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Requestor
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
                <StatusColumnHeader />
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
                    {searchQuery || selectedStatuses.length < statusOptions.length ? 'No requests match your search and filters' : 'No deployment requests available'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredRequests.map((request) => (
                <TableRow key={request.request_id} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', px: 1, borderRadius: 1, display: 'inline-block' }}>
                      {request.request_id}
                    </Typography>
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {getRequestTypeIcon(request.request_type)}
                      <Typography variant="body2">
                        {getRequestTypeLabel(request.request_type)}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.request_type === 'cluster_creation' && (
                      <Typography variant="body2">
                        {request.payload?.dns_subdomain} ({request.payload?.node_conf?.count} nodes)
                      </Typography>
                    )}
                    {(request.request_type === 'variant_promotion' || request.request_type === 'variant_onboarding') && (
                      <Typography variant="body2">
                        {request.payload?.entity}/{request.payload?.model}/{request.payload?.variant}
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {request.requestor || request.created_by || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Chip
                      label={request.status || 'PENDING'}
                      size="small"
                      sx={{
                        backgroundColor: request.status === 'APPROVED' ? '#E7F6E7' : request.status === 'REJECTED' ? '#FFEBEE' : '#FFF8E1',
                        color: request.status === 'APPROVED' ? '#2E7D32' : request.status === 'REJECTED' ? '#D32F2F' : '#F57C00',
                        fontWeight: 600
                      }}
                    />
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

      {/* View Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {selectedRequest && getRequestTypeIcon(selectedRequest.request_type)}
            Deployment Request Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedRequest && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
              {/* Request Info Section */}
              <Box sx={{ backgroundColor: '#f8f4f6', p: 2, borderRadius: 1, border: '1px solid #e4d5db' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                  {getRequestTypeIcon(selectedRequest.request_type)}
                  <Typography variant="h6" sx={{ color: '#522b4a', fontWeight: 600 }}>
                    Request Information
                  </Typography>
                </Box>
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Request ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>
                      {selectedRequest.request_id}
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Type
                    </Typography>
                    <Typography variant="body1">
                      {getRequestTypeLabel(selectedRequest.request_type)}
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Status
                    </Typography>
                    <Typography variant="body1">
                      <Chip
                        label={selectedRequest.status || 'PENDING'}
                        size="small"
                        sx={{
                          backgroundColor: selectedRequest.status === 'APPROVED' ? '#E7F6E7' : selectedRequest.status === 'REJECTED' ? '#FFEBEE' : '#FFF8E1',
                          color: selectedRequest.status === 'APPROVED' ? '#2E7D32' : selectedRequest.status === 'REJECTED' ? '#D32F2F' : '#F57C00',
                          fontWeight: 600
                        }}
                      />
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Requestor
                    </Typography>
                    <Typography variant="body1">
                      {selectedRequest.requestor || selectedRequest.created_by || 'N/A'}
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Reason
                    </Typography>
                    <Typography variant="body1">
                      {selectedRequest.reason || 'N/A'}
                    </Typography>
                  </Grid>
                </Grid>
              </Box>

              {/* Configuration Details Section */}
              <Box sx={{ backgroundColor: '#f0f7ff', p: 2, borderRadius: 1, border: '1px solid #d1e7ff' }}>
                <Typography variant="h6" sx={{ color: '#1976d2', fontWeight: 600, mb: 2 }}>
                  Configuration Details
                </Typography>
                <pre style={{ whiteSpace: 'pre-wrap', fontSize: '0.875rem', backgroundColor: '#f5f5f5', padding: '1rem', borderRadius: '4px' }}>
                  {JSON.stringify(selectedRequest.payload, null, 2)}
                </pre>
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button
            onClick={handleCloseViewModal}
            sx={{
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
          >
            Close
          </Button>
          {selectedRequest && selectedRequest.status === 'PENDING' && (
            <>
              <Button
                onClick={() => setShowRejectModal(true)}
                variant="contained"
                sx={{
                  backgroundColor: '#d32f2f',
                  '&:hover': { backgroundColor: '#b71c1c' }
                }}
                disabled={processingRequest === selectedRequest.request_id}
              >
                {processingRequest === selectedRequest.request_id ? 'Processing...' : 'Reject'}
              </Button>
              <Button
                onClick={() => handleApproveReject('APPROVED')}
                variant="contained"
                sx={{
                  backgroundColor: '#2e7d32',
                  '&:hover': { backgroundColor: '#1b5e20' }
                }}
                disabled={processingRequest === selectedRequest.request_id}
              >
                {processingRequest === selectedRequest.request_id ? 'Processing...' : 'Approve'}
              </Button>
            </>
          )}
        </DialogActions>
      </Dialog>

      {/* Reject Modal */}
      <Dialog open={showRejectModal} onClose={() => setShowRejectModal(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ color: '#d32f2f', fontWeight: 600 }}>
          Reject Deployment Request
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1" sx={{ mb: 2 }}>
            Are you sure you want to reject this deployment request? Please provide a reason for rejection.
          </Typography>
          <TextField
            fullWidth
            multiline
            rows={3}
            label="Rejection Comments"
            value={approvalComments}
            onChange={(e) => setApprovalComments(e.target.value)}
            placeholder="Please provide a reason for rejecting this deployment request..."
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowRejectModal(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => handleApproveReject('REJECTED')}
            variant="contained"
            color="error"
            disabled={processingRequest === selectedRequest?.request_id}
          >
            {processingRequest === selectedRequest?.request_id ? 'Processing...' : 'Reject Request'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Toast Notification */}
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

export default DeploymentApproval;