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
import ExperimentIcon from '@mui/icons-material/Science';
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

const VariantApproval = () => {
  const [variantRequests, setVariantRequests] = useState([]);
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

    // Admin configuration fields
  const [adminConfig, setAdminConfig] = useState({
    admin_vector_db_config: '',
    admin_rate_limiter: {
      rate_limit: 100,
      burst_limit: 200
    },
    admin_rt_partition: 5
  });

  const [jsonValidationError, setJsonValidationError] = useState('');

  const statusOptions = [
    { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
  ];

  useEffect(() => {
    fetchVariantRequests();
  }, []);

  const fetchVariantRequests = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getVariantRequests();
      
      if (response.variant_requests) {
        setVariantRequests(response.variant_requests);
      } else {
        setVariantRequests([]);
      }
    } catch (error) {
      console.error('Error fetching variant requests:', error);
      setError('Failed to load variant requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const filteredRequests = useMemo(() => {
    let filtered = variantRequests.filter(request => 
      selectedStatuses.includes(request.status?.toUpperCase())
    );

    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(request => {
        return (
          String(request.request_id || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.entity || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.model || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.variant || '').toLowerCase().includes(searchLower) ||
          String(request.payload?.type || '').toLowerCase().includes(searchLower) ||
          String(request.created_by || '').toLowerCase().includes(searchLower) ||
          (request.status && request.status.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at) - new Date(a.created_at);
    });
  }, [variantRequests, searchQuery, selectedStatuses]);

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
    
    // Pre-populate admin config based on request
    if (request.payload) {
      const defaultVectorDbConfig = {
        read_host: "",
        write_host: "",
        port: "6333",
        api_key: "",
        timeout: 30,
        collection_name: `${request.payload.entity}_${request.payload.variant}_embeddings`
      };
      
      setAdminConfig(prev => ({
        ...prev,
        admin_vector_db_config: JSON.stringify(defaultVectorDbConfig, null, 2)
      }));
    }
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedRequest(null);
    setAdminConfig({
      admin_vector_db_config: '',
      admin_rate_limiter: {
        rate_limit: 100,
        burst_limit: 200
      },
      admin_rt_partition: 5
    });
    setJsonValidationError('');
  };

  const validateJson = (jsonString) => {
    if (!jsonString.trim()) {
      return { isValid: false, error: 'Vector DB config is required' };
    }
    
    try {
      const parsed = JSON.parse(jsonString);
      
      // Check required fields
      const requiredFields = ['read_host', 'write_host', 'port', 'collection_name'];
      const missingFields = requiredFields.filter(field => !parsed[field]);
      
      if (missingFields.length > 0) {
        return { 
          isValid: false, 
          error: `Missing required fields: ${missingFields.join(', ')}` 
        };
      }
      
      return { isValid: true, error: '', parsed };
    } catch (error) {
      return { 
        isValid: false, 
        error: `Invalid JSON format: ${error.message}` 
      };
    }
  };

  const handleAdminConfigChange = (e) => {
    const { name, value } = e.target;
    
    if (name === 'admin_vector_db_config') {
      setAdminConfig(prev => ({
        ...prev,
        admin_vector_db_config: value
      }));
      
      // Validate JSON on change
      const validation = validateJson(value);
      setJsonValidationError(validation.error);
    } else if (name.startsWith('admin_rate_limiter.')) {
      const field = name.replace('admin_rate_limiter.', '');
      setAdminConfig(prev => ({
        ...prev,
        admin_rate_limiter: {
          ...prev.admin_rate_limiter,
          [field]: parseInt(value, 10) || 0
        }
      }));
    } else if (name === 'admin_rt_partition') {
      setAdminConfig(prev => ({
        ...prev,
        admin_rt_partition: parseInt(value, 10) || 0
      }));
    }
  };

  const handleApprovalSubmit = async (decision) => {
    if (decision === 'REJECTED' && !approvalComments.trim()) {
      return;
    }

    if (decision === 'APPROVED') {
      const jsonValidation = validateJson(adminConfig.admin_vector_db_config);
      if (!jsonValidation.isValid) {
        showNotification(`Vector DB Config Error: ${jsonValidation.error}`, 'error');
        return;
      }
    }

    try {
      setLoading(true);
      // Prepare admin config with parsed JSON
      const finalAdminConfig = {
        admin_rate_limiter: adminConfig.admin_rate_limiter,
        admin_rt_partition: adminConfig.admin_rt_partition
      };

      // Parse JSON for vector DB config if approving
      if (decision === 'APPROVED') {
        const jsonValidation = validateJson(adminConfig.admin_vector_db_config);
        finalAdminConfig.admin_vector_db_config = jsonValidation.parsed;
      }

      const payload = {
        request_id: selectedRequest.request_id,
        admin_id: user?.email || 'admin@example.com',
        approval_decision: decision,
        approval_comments: approvalComments,
        ...finalAdminConfig
      };
      
      const response = await embeddingPlatformAPI.approveVariant(payload);
      
      const message = response.data?.message || response.message || 'Variant request processed successfully';
      showNotification(message, 'success');
      fetchVariantRequests();
      handleCloseViewModal();
      handleRejectModalClose();
      
      // Trigger update event for other components
      window.dispatchEvent(new CustomEvent('variantApprovalUpdate'));
    } catch (error) {
      console.error('Error processing variant request:', error);
      showNotification(error.message || 'Failed to process variant request', 'error');
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
          <Typography variant="h6">Variant Approval</Typography>
          <Typography variant="body1" color="text.secondary">
            Review and approve/reject A/B testing variant requests with infrastructure configuration
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
          label="Search Variant Requests"
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
                Model
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Variant
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Type
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
                <TableCell colSpan={9} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {variantRequests.length === 0 ? 'No variant requests pending approval' : 'No variant requests match your search'}
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
                    {request.payload?.model || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Typography 
                      variant="body2" 
                      sx={{ 
                        fontFamily: 'monospace', 
                        backgroundColor: '#fff3e0',
                        padding: '2px 6px',
                        borderRadius: '4px',
                        fontSize: '0.875rem',
                        color: '#f57c00',
                        display: 'inline-block'
                      }}
                    >
                      {request.payload?.variant || 'N/A'}
                    </Typography>
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Chip 
                      label={request.payload?.type || 'EXPERIMENT'}
                      size="small"
                      sx={{ 
                        backgroundColor: '#e3f2fd',
                        color: '#1976d2',
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


      {/* View Variant Request Details Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ExperimentIcon sx={{ color: '#522b4a' }} />
            Variant Request Details
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

              {/* Variant Details Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(255, 152, 0, 0.02)', borderRadius: 1, border: '1px solid rgba(255, 152, 0, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <ExperimentIcon fontSize="small" sx={{ color: '#ff9800' }} />
                  Variant Configuration
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Entity/Model/Variant
                    </Typography>
                    <Box sx={{ display: 'flex', gap: 1, mt: 0.5, flexWrap: 'wrap', alignItems: 'center' }}>
                      <Chip 
                        label={selectedRequest.payload?.entity || 'N/A'}
                        size="small"
                        sx={{ backgroundColor: '#e3f2fd', color: '#1976d2', fontWeight: 600 }}
                      />
                      <Typography variant="body2">→</Typography>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', px: 1, borderRadius: 1 }}>
                        {selectedRequest.payload?.model || 'N/A'}
                      </Typography>
                      <Typography variant="body2">→</Typography>
                      <Chip 
                        label={selectedRequest.payload?.variant || 'N/A'}
                        size="small"
                        sx={{ backgroundColor: '#fff3e0', color: '#f57c00', fontWeight: 600 }}
                      />
                    </Box>
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

              {/* Admin Configuration Section */}
              {selectedRequest.status === 'PENDING' && (
                <Box sx={{ p: 2, backgroundColor: 'rgba(33, 150, 243, 0.02)', borderRadius: 1, border: '1px solid rgba(33, 150, 243, 0.1)' }}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                    Admin Infrastructure Configuration
                  </Typography>

                  {/* Vector Database Configuration */}
                  <Box sx={{ mb: 3 }}>
                    <Typography variant="body2" sx={{ fontWeight: 600, mb: 1.5, color: '#1976d2' }}>
                      Vector Database Configuration (JSON)
                    </Typography>
                    <TextField
                      multiline
                      rows={8}
                      fullWidth
                      label="Vector DB Configuration"
                      name="admin_vector_db_config"
                      value={adminConfig.admin_vector_db_config}
                      onChange={handleAdminConfigChange}
                      variant="outlined"
                      required
                      error={!!jsonValidationError}
                      helperText={jsonValidationError || "Enter JSON configuration with required fields: read_host, write_host, port, collection_name"}
                      placeholder={`{
  "read_host": "qdrant-read.example.com",
  "write_host": "qdrant-write.example.com",
  "port": "6333",
  "api_key": "your-api-key",
  "timeout": 30,
  "collection_name": "entity_variant_embeddings"
}`}
                      sx={{
                        '& .MuiInputBase-input': {
                          fontFamily: 'monospace',
                          fontSize: '0.875rem'
                        }
                      }}
                    />
                  </Box>

                  {/* Rate Limiter Configuration */}
                  <Box sx={{ mb: 3 }}>
                    <Typography variant="body2" sx={{ fontWeight: 600, mb: 1.5, color: '#1976d2' }}>
                      Rate Limiter Configuration
                    </Typography>
                    <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                      <TextField
                        size="small"
                        label="Rate Limit (req/sec)"
                        name="admin_rate_limiter.rate_limit"
                        value={adminConfig.admin_rate_limiter.rate_limit}
                        onChange={handleAdminConfigChange}
                        variant="outlined"
                        type="number"
                        placeholder="100"
                        helperText="Requests per second"
                      />
                      <TextField
                        size="small"
                        label="Burst Limit"
                        name="admin_rate_limiter.burst_limit"
                        value={adminConfig.admin_rate_limiter.burst_limit}
                        onChange={handleAdminConfigChange}
                        variant="outlined"
                        type="number"
                        placeholder="200"
                        helperText="Maximum burst capacity"
                      />
                    </Box>
                  </Box>

                  {/* RT Partition Configuration */}
                  <Box>
                    <Typography variant="body2" sx={{ fontWeight: 600, mb: 1.5, color: '#1976d2' }}>
                      Real-time Partition Configuration
                    </Typography>
                    <TextField
                      size="small"
                      label="RT Partition"
                      name="admin_rt_partition"
                      value={adminConfig.admin_rt_partition}
                      onChange={handleAdminConfigChange}
                      variant="outlined"
                      type="number"
                      placeholder="5"
                      helperText="Real-time partition assignment"
                      sx={{ width: '200px' }}
                    />
                  </Box>
                </Box>
              )}

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
              Are you sure you want to reject this variant request?
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

export default VariantApproval;