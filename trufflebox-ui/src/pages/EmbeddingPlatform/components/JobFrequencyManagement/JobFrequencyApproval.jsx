import React, { useState, useEffect, useMemo } from 'react';
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
  Snackbar
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import FilterListIcon from '@mui/icons-material/FilterList';
import ScheduleIcon from '@mui/icons-material/Schedule';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import PersonIcon from '@mui/icons-material/Person';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const JobFrequencyApproval = () => {
  const [frequencyRequests, setFrequencyRequests] = useState([]);
  const [showViewModal, setShowViewModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [approvalComments, setApprovalComments] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING', 'APPROVED', 'REJECTED']);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { user } = useAuth();
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  // Filter popover state
  const [statusAnchorEl, setStatusAnchorEl] = useState(null);

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

  const filteredRequests = useMemo(() => {
    let filtered = frequencyRequests.filter(request =>
      selectedStatuses.includes(request.status?.toUpperCase())
    );

    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(request => {
        return (
          String(request.request_id || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.job_frequency || '').toLowerCase().includes(searchLower) ||
          String(request.reason || '').toLowerCase().includes(searchLower) ||
          String(request.created_by || '').toLowerCase().includes(searchLower) ||
          (request.status && request.status.toLowerCase().includes(searchLower))
        );
      });
    }

    return filtered.sort((a, b) => {
      return new Date(b.created_at) - new Date(a.created_at);
    });
  }, [frequencyRequests, searchQuery, selectedStatuses]);

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

  const showNotification = (message, severity) => {
    setNotification({ open: true, message, severity });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({ ...prev, open: false }));
  };

  // Status filter functions
  const handleStatusFilterClick = (event) => {
    setStatusAnchorEl(event.currentTarget);
  };

  const handleStatusFilterClose = () => {
    setStatusAnchorEl(null);
  };

  const handleStatusToggle = (status) => {
    setSelectedStatuses(prev =>
      prev.includes(status)
        ? prev.filter(s => s !== status)
        : [...prev, status]
    );
  };

  // Status Column Header with filtering (same pattern as ModelRegistry)
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

      {/* View Job Frequency Request Details Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ScheduleIcon sx={{ color: '#522b4a' }} />
            Job Frequency Request Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedRequest && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 2 }}>
              {/* Request Information Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <PersonIcon fontSize="small" sx={{ color: '#522b4a' }} />
                  Request Information
                </Typography>

                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Request ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>
                      {selectedRequest.request_id}
                    </Typography>
                  </Box>

                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created By
                    </Typography>
                    <Typography variant="body1">
                      {selectedRequest.created_by || 'N/A'}
                    </Typography>
                  </Box>

                  <Box>
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
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created At
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      {selectedRequest.created_at ? new Date(selectedRequest.created_at).toLocaleString() : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Box>

              {/* Job Frequency Details Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(33, 150, 243, 0.02)', borderRadius: 1, border: '1px solid rgba(33, 150, 243, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <ScheduleIcon fontSize="small" sx={{ color: '#1976d2' }} />
                  Job Frequency Configuration
                </Typography>

                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Frequency ID
                    </Typography>
                    <Typography variant="body1" sx={{ 
                      fontFamily: 'monospace',
                      backgroundColor: '#e3f2fd',
                      padding: '4px 8px',
                      borderRadius: '4px',
                      color: '#1976d2',
                      display: 'inline-block',
                      mt: 0.5
                    }}>
                      {selectedRequest.payload?.job_frequency}
                    </Typography>
                  </Box>

                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Description
                    </Typography>
                    <Typography variant="body1" sx={{ mt: 0.5 }}>
                      {generateDescription(selectedRequest.payload?.job_frequency)}
                    </Typography>
                  </Box>

                  <Box sx={{ gridColumn: 'span 2' }}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Reason
                    </Typography>
                    <Typography variant="body1" sx={{ mt: 0.5 }}>
                      {selectedRequest.reason || 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Box>

              {/* Approval Section */}
              {selectedRequest.status === 'PENDING' && (
                <Box sx={{ p: 2, backgroundColor: 'rgba(76, 175, 80, 0.02)', borderRadius: 1, border: '1px solid rgba(76, 175, 80, 0.1)' }}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <CheckCircleIcon fontSize="small" sx={{ color: '#4caf50' }} />
                    Approval Decision
                  </Typography>
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
                </Box>
                )}
              </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2, backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button 
            onClick={handleCloseViewModal}
            sx={{ 
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
          >
            Close
          </Button>
          {selectedRequest?.status === 'PENDING' && (
            <>
              <Button 
                onClick={() => handleApprovalSubmit('APPROVED')}
                variant="contained"
                startIcon={<CheckCircleIcon />}
                disabled={loading}
                sx={{
                  backgroundColor: '#2e7d32',
                  '&:hover': { backgroundColor: '#1b5e20' }
                }}
              >
                {loading ? 'Processing...' : 'Approve'}
              </Button>
              <Button 
                onClick={handleRejectModalOpen}
                variant="contained"
                startIcon={<CancelIcon />}
                disabled={loading}
                sx={{
                  backgroundColor: '#d32f2f',
                  '&:hover': { backgroundColor: '#b71c1c' }
                }}
              >
                Reject
              </Button>
            </>
          )}
        </DialogActions>
      </Dialog>

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

export default JobFrequencyApproval;