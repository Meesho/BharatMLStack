import React, { useState, useEffect, useMemo } from 'react';
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
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
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
import FilterListIcon from '@mui/icons-material/FilterList';
import SearchIcon from '@mui/icons-material/Search';
import Chip from '@mui/material/Chip';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const ModelApproval = () => {
  const [modelRequests, setModelRequests] = useState([]);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [showRejectModal, setShowRejectModal] = useState(false);
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

  const statusOptions = [
    { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
  ];

  useEffect(() => {
    fetchModelRequests();
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

  const filteredRequests = useMemo(() => {
    let filtered = modelRequests.filter(request => 
      selectedStatuses.includes(request.status?.toUpperCase())
    );

    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(request => {
        return (
          String(request.request_id || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.model || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.entity || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.model_type || '').toLowerCase().includes(searchLower) ||
          String(request.created_by || '').toLowerCase().includes(searchLower) ||
          (request.status && request.status.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at) - new Date(a.created_at);
    });
  }, [modelRequests, searchQuery, selectedStatuses]);

  const getStatusChip = (status) => {
    const statusUpper = (status || '').toUpperCase();
    let bgcolor = '#EEEEEE';
    let textColor = '#616161';

    switch (statusUpper) {
      case 'PENDING':
        bgcolor = '#FFF8E1';
        textColor = '#F57C00';
        break;
      case 'APPROVED':
        bgcolor = '#E7F6E7';
        textColor = '#2E7D32';
        break;
      case 'REJECTED':
        bgcolor = '#FFEBEE';
        textColor = '#D32F2F';
        break;
      default:
        bgcolor = '#EEEEEE';
        textColor = '#616161';
    }

    return (
      <Chip
        label={statusUpper || 'UNKNOWN'}
        size="small"
        sx={{
          backgroundColor: bgcolor,
          color: textColor,
          fontWeight: 'bold',
          minWidth: '80px',
        }}
      />
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
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
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
                <StatusColumnHeader />
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
                    {getStatusChip(request.status)}
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
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ModelTrainingIcon sx={{ color: '#522b4a' }} />
            Model Request Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedRequest && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 2 }}>
              {/* Request Information Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                  Request Information
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Request ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                      {selectedRequest.request_id || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Status
                    </Typography>
                    <Box sx={{ mt: 0.5 }}>
                      {getStatusChip(selectedRequest.status)}
                    </Box>
                  </Box>
                </Box>
              </Box>

              {/* Model Details Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(255, 152, 0, 0.02)', borderRadius: 1, border: '1px solid rgba(255, 152, 0, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <SmartToyIcon fontSize="small" sx={{ color: '#ff9800' }} />
                  Model Configuration
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Model Name
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <SmartToyIcon fontSize="small" sx={{ color: '#ff9800' }} />
                      {selectedRequest.payload?.model || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Entity
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <StorageIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      {selectedRequest.payload?.entity || 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Box>

              {/* Configuration Details */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                  Full Configuration
                </Typography>
                <TextField
                  multiline
                  rows={8}
                  value={selectedRequest.payload ? JSON.stringify(selectedRequest.payload, null, 2) : '{}'}
                  variant="outlined"
                  fullWidth
                  InputProps={{ 
                    readOnly: true,
                    style: { fontFamily: 'monospace', fontSize: '0.875rem' }
                  }}
                  sx={{ backgroundColor: '#fafafa' }}
                />
              </Box>

              {/* Metadata Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(158, 158, 158, 0.02)', borderRadius: 1, border: '1px solid rgba(158, 158, 158, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <PersonIcon fontSize="small" sx={{ color: '#757575' }} />
                  Request Metadata
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created By
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <PersonIcon fontSize="small" sx={{ color: '#522b4a' }} />
                      {selectedRequest.created_by || 'N/A'}
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
          )}
        </DialogActions>
      </Dialog>

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

export default ModelApproval;