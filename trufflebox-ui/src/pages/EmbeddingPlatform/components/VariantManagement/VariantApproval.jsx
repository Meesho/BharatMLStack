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
  Collapse,
  Switch,
  FormControlLabel,
  Grid,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
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

const statusOptions = [
  { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
  { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
  { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
];

const StatusColumnHeader = ({ selectedStatuses, setSelectedStatuses }) => {
  const [anchorEl, setAnchorEl] = useState(null);

  const handleClick = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleStatusToggle = (status) => {
    setSelectedStatuses(prev =>
      prev.includes(status) ? prev.filter(s => s !== status) : [...prev, status]
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
          width: '100%',
        }}
      >
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            cursor: 'pointer',
            '&:hover': { backgroundColor: 'rgba(0, 0, 0, 0.04)' },
            borderRadius: 1,
            p: 0.5,
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
              color: selectedStatuses.length < statusOptions.length ? '#1976d2' : '#666',
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
                    '& .MuiChip-label': { px: 0.5 },
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
                  '& .MuiChip-label': { px: 0.5 },
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
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        PaperProps={{ sx: { width: 200, maxHeight: 300, overflow: 'auto' } }}
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
                      minWidth: '80px',
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

  // Admin configuration fields (vector DB, rate limiter, caching)
  const [adminConfig, setAdminConfig] = useState({
    admin_vector_db_config: '',
    admin_rate_limiter: {
      rate_limit: 100,
      burst_limit: 200
    },
    admin_caching_configuration: {
      in_memory_caching_enabled: true,
      in_memory_cache_ttl_seconds: 300,
      distributed_caching_enabled: false,
      distributed_cache_ttl_seconds: 600,
      embedding_retrieval_in_memory_config: { enabled: true, ttl: 60 },
      embedding_retrieval_distributed_config: { enabled: false, ttl: 300 },
      dot_product_in_memory_config: { enabled: false, ttl: 1 },
      dot_product_distributed_config: { enabled: false, ttl: 1 },
    },
  });

  const [jsonValidationError, setJsonValidationError] = useState('');
  const [showRawPayload, setShowRawPayload] = useState(false);
  const [showRawVectorDb, setShowRawVectorDb] = useState(false);

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
        port: "6334",
        http2config: {
          deadline: 200,
          write_deadline: 5000,
          keep_alive_time: "30000",
          thread_pool_size: "32",
          is_plain_text: true
        },
        http1config: {
          timeout_in_ms: 0,
          dial_timeout_in_ms: 0,
          max_idle_connections: 0,
          max_idle_connections_per_host: 0,
          idle_conn_timeout_in_ms: 0,
          keep_alive_timeout_in_ms: 0
        },
        params: {
          default_indexing_threshold: "100",
          indexing_threshold: "100",
          max_indexing_threads: "2",
          max_segment_size_in_mb: "100",
          on_disk_payload: "false",
          replication_factor: "3",
          search_indexed_only: "true",
          segment_number: "16",
          shard_number: "1",
          write_consistency_factor: "2"
        },
        payload: {
          sc: {
            field_schema: "keyword",
            default_value: ""
          }
        }
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
      admin_rate_limiter: { rate_limit: 100, burst_limit: 200 },
      admin_caching_configuration: {
        in_memory_caching_enabled: true,
        in_memory_cache_ttl_seconds: 300,
        distributed_caching_enabled: false,
        distributed_cache_ttl_seconds: 600,
        embedding_retrieval_in_memory_config: { enabled: true, ttl: 60 },
        embedding_retrieval_distributed_config: { enabled: false, ttl: 300 },
        dot_product_in_memory_config: { enabled: false, ttl: 1 },
        dot_product_distributed_config: { enabled: false, ttl: 1 },
      },
    });
    setJsonValidationError('');
    setShowRawPayload(false);
    setShowRawVectorDb(false);
  };

  const validateJson = (jsonString) => {
    if (!jsonString.trim()) {
      return { isValid: false, error: 'Vector DB config is required' };
    }
    
    try {
      const parsed = JSON.parse(jsonString);
      
      // Check required fields
      const requiredFields = ['read_host', 'write_host', 'port'];
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
    const { name, value, type, checked } = e.target;
    
    if (name === 'admin_vector_db_config') {
      setAdminConfig(prev => ({ ...prev, admin_vector_db_config: value }));
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
    } else if (name.startsWith('admin_caching_configuration.')) {
      const rest = name.replace('admin_caching_configuration.', '');
      if (rest.includes('.')) {
        const [configType, field] = rest.split('.');
        setAdminConfig(prev => ({
          ...prev,
          admin_caching_configuration: {
            ...prev.admin_caching_configuration,
            [configType]: {
              ...(prev.admin_caching_configuration[configType] || {}),
              [field]: type === 'checkbox' ? checked : (parseInt(value, 10) || 0)
            }
          }
        }));
      } else {
        setAdminConfig(prev => ({
          ...prev,
          admin_caching_configuration: {
            ...prev.admin_caching_configuration,
            [rest]: type === 'checkbox' ? checked : (parseInt(value, 10) || 0)
          }
        }));
      }
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
      };

      // Parse JSON for vector DB config if approving
      if (decision === 'APPROVED') {
        const jsonValidation = validateJson(adminConfig.admin_vector_db_config);
        finalAdminConfig.admin_vector_db_config = jsonValidation.parsed;
        finalAdminConfig.admin_caching_configuration = adminConfig.admin_caching_configuration;
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
                <StatusColumnHeader selectedStatuses={selectedStatuses} setSelectedStatuses={setSelectedStatuses} />
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
        <DialogContent sx={{ pt: 3 }}>
          {selectedRequest && (
            <Grid container spacing={3}>
              {/* Request Information and Variant Configuration side by side */}
              <Grid item xs={12} md={6}>
                <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)', height: '100%' }}>
                  <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                    Request Information
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600 }}>Request ID</Typography>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>{selectedRequest.request_id || 'N/A'}</Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600 }}>Status</Typography>
                      <Box sx={{ mt: 0.25 }}>{getStatusChip(selectedRequest.status)}</Box>
                    </Grid>
                  </Grid>
                </Box>
              </Grid>

              <Grid item xs={12} md={6}>
                <Box sx={{ p: 2, backgroundColor: 'rgba(255, 152, 0, 0.02)', borderRadius: 1, border: '1px solid rgba(255, 152, 0, 0.1)', height: '100%' }}>
                  <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <ExperimentIcon fontSize="small" sx={{ color: '#ff9800' }} />
                    Variant Configuration
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                    <Chip label={selectedRequest.payload?.entity || 'N/A'} size="small" sx={{ backgroundColor: '#e3f2fd', color: '#1976d2', fontWeight: 600 }} />
                    <Typography variant="body2" color="text.secondary">→</Typography>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', px: 1, borderRadius: 1 }}>{selectedRequest.payload?.model || 'N/A'}</Typography>
                    <Typography variant="body2" color="text.secondary">→</Typography>
                    <Chip label={selectedRequest.payload?.variant || 'N/A'} size="small" sx={{ backgroundColor: '#fff3e0', color: '#f57c00', fontWeight: 600 }} />
                  </Box>
                </Box>
              </Grid>

              {/* Request configuration (fields + filter criteria) */}
              <Grid item xs={12}>
                <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)' }}>
                  <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                    Request Configuration
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={6} sm={4}>
                      <Typography variant="caption" color="text.secondary" display="block">Entity</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>{selectedRequest.payload?.entity ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={6} sm={4}>
                      <Typography variant="caption" color="text.secondary" display="block">Model</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>{selectedRequest.payload?.model ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={6} sm={4}>
                      <Typography variant="caption" color="text.secondary" display="block">Variant</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>{selectedRequest.payload?.variant ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={6} sm={4}>
                      <Typography variant="caption" color="text.secondary" display="block">Type</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>{selectedRequest.payload?.type ?? '—'}</Typography>
                    </Grid>
                    <Grid item xs={6} sm={4}>
                      <Typography variant="caption" color="text.secondary" display="block">Vector DB type</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>{selectedRequest.payload?.vector_db_type ?? '—'}</Typography>
                    </Grid>
                  </Grid>
                  {selectedRequest.payload?.filter_configuration?.criteria?.length > 0 && (
                    <Box sx={{ mt: 2 }}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>Filter criteria</Typography>
                      <TableContainer component={Paper} variant="outlined" sx={{ maxWidth: 400 }}>
                        <Table size="small">
                          <TableHead>
                            <TableRow>
                              <TableCell sx={{ fontWeight: 600 }}>Column</TableCell>
                              <TableCell sx={{ fontWeight: 600 }}>Condition</TableCell>
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {selectedRequest.payload.filter_configuration.criteria.map((c, i) => (
                              <TableRow key={i}>
                                <TableCell>{c.column_name ?? '—'}</TableCell>
                                <TableCell>{c.condition ?? '—'}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </TableContainer>
                    </Box>
                  )}
                  <Button size="small" startIcon={showRawPayload ? <ExpandLessIcon /> : <ExpandMoreIcon />} onClick={() => setShowRawPayload((p) => !p)} sx={{ mt: 2, textTransform: 'none', color: '#1976d2' }}>
                    {showRawPayload ? 'Hide raw payload' : 'Show raw payload (JSON)'}
                  </Button>
                  <Collapse in={showRawPayload}>
                    <TextField multiline rows={8} value={selectedRequest.payload ? JSON.stringify(selectedRequest.payload, null, 2) : '{}'} variant="outlined" fullWidth InputProps={{ readOnly: true, style: { fontFamily: 'monospace', fontSize: '0.875rem' } }} sx={{ mt: 1, backgroundColor: '#fafafa' }} />
                  </Collapse>
                </Box>
              </Grid>

              {/* Metadata */}
              <Grid item xs={12}>
                <Box sx={{ p: 2, backgroundColor: 'rgba(158, 158, 158, 0.02)', borderRadius: 1, border: '1px solid rgba(158, 158, 158, 0.1)' }}>
                  <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <PersonIcon fontSize="small" sx={{ color: '#757575' }} />
                    Request Metadata
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600 }}>Created By</Typography>
                      <Typography variant="body2" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}><PersonIcon fontSize="small" sx={{ color: '#522b4a', fontSize: 16 }} />{selectedRequest.created_by || 'N/A'}</Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600 }}>Created At</Typography>
                      <Typography variant="body2" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}><AccessTimeIcon fontSize="small" sx={{ color: '#1976d2', fontSize: 16 }} />{selectedRequest.created_at ? new Date(selectedRequest.created_at).toLocaleString() : 'N/A'}</Typography>
                    </Grid>
                  </Grid>
                </Box>
              </Grid>

              {/* Admin Configuration */}
              {selectedRequest.status === 'PENDING' && (
                <Grid item xs={12}>
                  <Box sx={{ p: 2, backgroundColor: 'rgba(33, 150, 243, 0.02)', borderRadius: 1, border: '1px solid rgba(33, 150, 243, 0.1)' }}>
                    <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                      <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      Admin Infrastructure Configuration
                    </Typography>

                    {/* Vector Database Configuration */}
                    <Box sx={{ mb: 3 }}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600, mb: 1 }}>Vector Database Configuration</Typography>
                    {(() => {
                      const parsed = validateJson(adminConfig.admin_vector_db_config);
                      if (parsed.isValid && parsed.parsed) {
                        const p = parsed.parsed;
                        return (
                          <Box sx={{ mb: 1.5, p: 1.5, backgroundColor: 'rgba(25, 118, 210, 0.04)', borderRadius: 1 }}>
                            <Typography variant="caption" color="text.secondary">Summary (verify before approval)</Typography>
                            <Grid container spacing={1} sx={{ mt: 0.5 }}>
                              <Grid item xs={6} sm={4}><Typography variant="body2"><strong>read_host:</strong> {p.read_host ?? '—'}</Typography></Grid>
                              <Grid item xs={6} sm={4}><Typography variant="body2"><strong>write_host:</strong> {p.write_host ?? '—'}</Typography></Grid>
                              <Grid item xs={6} sm={4}><Typography variant="body2"><strong>port:</strong> {p.port ?? '—'}</Typography></Grid>
                            </Grid>
                            <Button size="small" startIcon={showRawVectorDb ? <ExpandLessIcon /> : <ExpandMoreIcon />} onClick={() => setShowRawVectorDb((v) => !v)} sx={{ mt: 1, textTransform: 'none', color: '#1976d2' }}>
                              {showRawVectorDb ? 'Hide' : 'Edit JSON'}
                            </Button>
                          </Box>
                        );
                      }
                      return null;
                    })()}
                    <Collapse in={showRawVectorDb || !adminConfig.admin_vector_db_config?.trim() || !validateJson(adminConfig.admin_vector_db_config).isValid}>
                      <TextField
                        multiline
                        rows={8}
                        fullWidth
                        label="Vector DB Configuration (JSON)"
                        name="admin_vector_db_config"
                        value={adminConfig.admin_vector_db_config}
                        onChange={handleAdminConfigChange}
                        variant="outlined"
                        required
                        error={!!jsonValidationError}
                        helperText={jsonValidationError || 'Required: read_host, write_host, port'}
                        placeholder='{"read_host":"...","write_host":"...","port":"6333",...}'
                        sx={{ '& .MuiInputBase-input': { fontFamily: 'monospace', fontSize: '0.875rem' } }}
                      />
                    </Collapse>
                  </Box>

                    {/* Rate Limiter */}
                    <Box sx={{ mb: 3 }}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600, mb: 1 }}>Rate Limiter</Typography>
                      <Grid container spacing={2}>
                        <Grid item xs={12} sm={6} md={4}>
                          <TextField size="small" fullWidth label="Rate Limit (req/sec)" name="admin_rate_limiter.rate_limit" value={adminConfig.admin_rate_limiter.rate_limit} onChange={handleAdminConfigChange} variant="outlined" type="number" />
                        </Grid>
                        <Grid item xs={12} sm={6} md={4}>
                          <TextField size="small" fullWidth label="Burst Limit" name="admin_rate_limiter.burst_limit" value={adminConfig.admin_rate_limiter.burst_limit} onChange={handleAdminConfigChange} variant="outlined" type="number" />
                        </Grid>
                      </Grid>
                    </Box>

                    {/* Caching Configuration */}
                    <Box sx={{ mb: 2 }}>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 600, mb: 1 }}>Caching Configuration</Typography>
                    <TableContainer component={Paper} variant="outlined">
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell sx={{ fontWeight: 600, width: 180 }}>Setting</TableCell>
                            <TableCell sx={{ fontWeight: 600 }} align="center">In-memory</TableCell>
                            <TableCell sx={{ fontWeight: 600 }} align="center">Distributed</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          <TableRow>
                            <TableCell sx={{ verticalAlign: 'middle' }}>Caching</TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'center' }}>
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={adminConfig.admin_caching_configuration?.in_memory_caching_enabled ?? false}
                                      onChange={handleAdminConfigChange}
                                      name="admin_caching_configuration.in_memory_caching_enabled"
                                    />
                                  }
                                  label=""
                                />
                                <TextField
                                  size="small"
                                  type="number"
                                  label="TTL (sec)"
                                  name="admin_caching_configuration.in_memory_cache_ttl_seconds"
                                  value={adminConfig.admin_caching_configuration?.in_memory_cache_ttl_seconds ?? 0}
                                  onChange={handleAdminConfigChange}
                                  sx={{ width: 100, '& .MuiInputBase-input': { height: '1em', minHeight: '1em' } }}
                                  inputProps={{ min: 0 }}
                                />
                              </Box>
                            </TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'center' }}>
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={adminConfig.admin_caching_configuration?.distributed_caching_enabled ?? false}
                                      onChange={handleAdminConfigChange}
                                      name="admin_caching_configuration.distributed_caching_enabled"
                                    />
                                  }
                                  label=""
                                />
                                <TextField
                                  size="small"
                                  type="number"
                                  label="TTL (sec)"
                                  name="admin_caching_configuration.distributed_cache_ttl_seconds"
                                  value={adminConfig.admin_caching_configuration?.distributed_cache_ttl_seconds ?? 0}
                                  onChange={handleAdminConfigChange}
                                  sx={{ width: 100, '& .MuiInputBase-input': { height: '1em', minHeight: '1em' } }}
                                  inputProps={{ min: 0 }}
                                />
                              </Box>
                            </TableCell>
                          </TableRow>
                          <TableRow>
                            <TableCell sx={{ verticalAlign: 'middle' }}>Embedding retrieval</TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'center' }}>
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={adminConfig.admin_caching_configuration?.embedding_retrieval_in_memory_config?.enabled ?? false}
                                      onChange={handleAdminConfigChange}
                                      name="admin_caching_configuration.embedding_retrieval_in_memory_config.enabled"
                                    />
                                  }
                                  label=""
                                />
                                <TextField
                                  size="small"
                                  type="number"
                                  label="TTL"
                                  name="admin_caching_configuration.embedding_retrieval_in_memory_config.ttl"
                                  value={adminConfig.admin_caching_configuration?.embedding_retrieval_in_memory_config?.ttl ?? 0}
                                  onChange={handleAdminConfigChange}
                                  sx={{ width: 100, '& .MuiInputBase-input': { height: '1em', minHeight: '1em' } }}
                                  inputProps={{ min: 0 }}
                                />
                              </Box>
                            </TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'center' }}>
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={adminConfig.admin_caching_configuration?.embedding_retrieval_distributed_config?.enabled ?? false}
                                      onChange={handleAdminConfigChange}
                                      name="admin_caching_configuration.embedding_retrieval_distributed_config.enabled"
                                    />
                                  }
                                  label=""
                                />
                                <TextField
                                  size="small"
                                  type="number"
                                  label="TTL"
                                  name="admin_caching_configuration.embedding_retrieval_distributed_config.ttl"
                                  value={adminConfig.admin_caching_configuration?.embedding_retrieval_distributed_config?.ttl ?? 0}
                                  onChange={handleAdminConfigChange}
                                  sx={{ width: 100, '& .MuiInputBase-input': { height: '1em', minHeight: '1em' } }}
                                  inputProps={{ min: 0 }}
                                />
                              </Box>
                            </TableCell>
                          </TableRow>
                          <TableRow>
                            <TableCell sx={{ verticalAlign: 'middle' }}>Dot product</TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'center' }}>
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={adminConfig.admin_caching_configuration?.dot_product_in_memory_config?.enabled ?? false}
                                      onChange={handleAdminConfigChange}
                                      name="admin_caching_configuration.dot_product_in_memory_config.enabled"
                                    />
                                  }
                                  label=""
                                />
                                <TextField
                                  size="small"
                                  type="number"
                                  label="TTL"
                                  name="admin_caching_configuration.dot_product_in_memory_config.ttl"
                                  value={adminConfig.admin_caching_configuration?.dot_product_in_memory_config?.ttl ?? 0}
                                  onChange={handleAdminConfigChange}
                                  sx={{ width: 100, '& .MuiInputBase-input': { height: '1em', minHeight: '1em' } }}
                                  inputProps={{ min: 0 }}
                                />
                              </Box>
                            </TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'center' }}>
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={adminConfig.admin_caching_configuration?.dot_product_distributed_config?.enabled ?? false}
                                      onChange={handleAdminConfigChange}
                                      name="admin_caching_configuration.dot_product_distributed_config.enabled"
                                    />
                                  }
                                  label=""
                                />
                                <TextField
                                  size="small"
                                  type="number"
                                  label="TTL"
                                  name="admin_caching_configuration.dot_product_distributed_config.ttl"
                                  value={adminConfig.admin_caching_configuration?.dot_product_distributed_config?.ttl ?? 0}
                                  onChange={handleAdminConfigChange}
                                  sx={{ width: 100, '& .MuiInputBase-input': { height: '1em', minHeight: '1em' } }}
                                  inputProps={{ min: 0 }}
                                />
                              </Box>
                            </TableCell>
                          </TableRow>
                        </TableBody>
                      </Table>
                    </TableContainer>
                    </Box>
                  </Box>
                </Grid>
              )}

              {/* Approval Decision */}
              {selectedRequest?.status === 'PENDING' && (
                <Grid item xs={12}>
                  <Box sx={{ p: 2, backgroundColor: 'rgba(76, 175, 80, 0.02)', borderRadius: 1, border: '1px solid rgba(76, 175, 80, 0.1)' }}>
                    <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CheckCircleIcon fontSize="small" sx={{ color: '#4caf50' }} />
                      Approval Decision
                    </Typography>
                    <TextField label="Approval/Rejection Comments" multiline rows={3} value={approvalComments} onChange={(e) => setApprovalComments(e.target.value)} variant="outlined" fullWidth placeholder="Enter comments for your decision (required for rejection)..." />
                  </Box>
                </Grid>
              )}
            </Grid>
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