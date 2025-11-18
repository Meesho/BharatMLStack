import React, { useState, useEffect, useMemo, useCallback } from 'react';
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
  FormHelperText,
  Chip,
  IconButton,
  Tooltip,
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Snackbar,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import FilterListIcon from '@mui/icons-material/FilterList';
import CategoryIcon from '@mui/icons-material/Category';
import InfoIcon from '@mui/icons-material/Info';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import StorageIcon from '@mui/icons-material/Storage';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { 
  validateEntityPayload, 
  getDefaultFormValues
} from '../../../../services/embeddingPlatform/utils';

const EntityRegistry = () => {
  const [entities, setEntities] = useState([]);
  const [stores, setStores] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');
  const [selectedEntity, setSelectedEntity] = useState(null);
  const [showViewModal, setShowViewModal] = useState(false);
  const [open, setOpen] = useState(false);
  const [entityData, setEntityData] = useState(getDefaultFormValues('entity'));
  const [validationErrors, setValidationErrors] = useState({});
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING', 'APPROVED', 'REJECTED']);
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });
  const { user } = useAuth();

  const fetchStores = useCallback(async () => {
    try {
      const response = await embeddingPlatformAPI.getStoresFromETCD();
      if (response.stores) {
        setStores(response.stores);
      }
    } catch (error) {
      console.error('Error fetching stores from ETCD:', error);
    }
  }, []);

  const fetchEntities = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      
      const response = await embeddingPlatformAPI.getEntityRequests();
      
      if (response.entity_requests) {
        setEntities(response.entity_requests);
      } else {
        setEntities([]);
      }
    } catch (error) {
      console.error('Error fetching entity data:', error);
      setError('Failed to load entity data. Please refresh the page.');
      showNotification('Failed to load entity data. Please refresh the page.', 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStores();
    fetchEntities();
  }, [fetchStores, fetchEntities]);

  // Filtered and sorted data using useMemo
  const filteredEntities = useMemo(() => {
    let filtered = [...entities];
    
    // Status filtering
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(entity => 
        selectedStatuses.includes(entity.status?.toUpperCase())
      );
    }
    
    // Search filtering
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(entity => {
        return (
          String(entity.request_id || '').toLowerCase().includes(searchLower) ||
          String(entity.payload?.entity || '').toLowerCase().includes(searchLower) ||
          String(entity.payload?.store_id || '').toLowerCase().includes(searchLower) ||
          (entity.status && entity.status.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at) - new Date(a.created_at);
    });
  }, [entities, searchQuery, selectedStatuses]);

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
      case 'CANCELLED':
        bgcolor = '#EEEEEE';
        textColor = '#616161';
        break;
      case 'FAILED':
        bgcolor = '#FFCDD2';
        textColor = '#C62828';
        break;
      default:
        break;
    }

    return (
      <Chip
        label={status || 'N/A'}
        sx={{
          backgroundColor: bgcolor,
          color: textColor,
          fontWeight: 'bold',
          fontSize: '0.75rem'
        }}
      />
    );
  };

  // Status Column Header with filtering
  const StatusColumnHeader = () => {
    const [anchorEl, setAnchorEl] = useState(null);
    
    const statusOptions = [
      { value: 'PENDING', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
      { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
      { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
      { value: 'CANCELLED', label: 'Cancelled', color: '#EEEEEE', textColor: '#616161' },
      { value: 'FAILED', label: 'Failed', color: '#FFCDD2', textColor: '#C62828' }
    ];

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
          
          {/* Show active filters */}
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
          <Box sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
              <Typography variant="subtitle2" fontWeight="bold">
                Filter by Status
              </Typography>
              <Box sx={{ display: 'flex', gap: 0.5 }}>
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
            </Box>
            <Divider sx={{ mb: 1 }} />
            <List dense sx={{ py: 0 }}>
              {statusOptions.map((option) => (
                <ListItem key={option.value} disablePadding>
                  <ListItemButton
                    onClick={() => handleStatusToggle(option.value)}
                    sx={{ py: 0.5, px: 1 }}
                  >
                    <Checkbox
                      checked={selectedStatuses.includes(option.value)}
                      size="small"
                      sx={{ mr: 1, p: 0 }}
                    />
                    <Chip
                      label={option.label}
                      size="small"
                      sx={{
                        backgroundColor: option.color,
                        color: option.textColor,
                        fontWeight: 'bold',
                        fontSize: '0.75rem',
                        height: 24
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

  const handleViewEntity = (entity) => {
    setSelectedEntity(entity);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedEntity(null);
  };

  const handleOpen = () => {
    if (stores.length === 0) {
      showNotification('No approved stores available. Please create and approve a store first before registering entities.', 'warning');
      return;
    }
    setEntityData(getDefaultFormValues('entity'));
    setValidationErrors({});
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setEntityData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const showNotification = (message, severity) => {
    setNotification({ open: true, message, severity });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({ ...prev, open: false }));
  };

  const handleSubmit = async () => {
    const validation = validateEntityPayload(entityData);
    if (!validation.isValid) {
      setValidationErrors(validation.errors);
      showNotification('Please fix the validation errors and try again.', 'error');
      return;
    }

    try {
      setValidationErrors({});
      const payload = {
        requestor: user.email,
        reason: `Registering entity '${entityData.entity}' for data management and ML operations`,
        request_type: 'CREATE',
        payload: {
          entity: entityData.entity,
          store_id: entityData.store_id
        }
      };

      const response = await embeddingPlatformAPI.registerEntity(payload);
      const message = response.data?.message || response.message || 'Entity registration request submitted successfully';
      showNotification(message, 'success');
      setOpen(false);
      fetchEntities();
    } catch (error) {
      console.error('Error submitting entity registration:', error);
      showNotification(error.message || 'Failed to submit entity registration request', 'error');
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
          <Typography variant="h6">Entity Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage entity registration requests
          </Typography>
        </Box>
      </Box>

      {/* Search and Register Button */}
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
          label="Search Entity Requests"
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
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleOpen}
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730'
            },
            minWidth: '150px'
          }}
        >
          Register Entity
        </Button>
      </Box>

      {/* Entities Table */}
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
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
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
                Entity Name
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Store ID
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
            {filteredEntities.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {entities.length === 0 ? 'No entity requests available' : 'No entity requests match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredEntities.map((entity, index) => (
                <TableRow key={entity.request_id || index} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {entity.request_id || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {entity.payload?.entity || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {entity.payload?.store_id || 'N/A'}
                  </TableCell>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    {getStatusChip(entity.status)}
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      <Tooltip title="View Details">
                        <IconButton 
                          size="small"
                          onClick={() => handleViewEntity(entity)}
                          sx={{ color: '#1976d2' }}
                        >
                          <VisibilityIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Entity Registration Dialog */}
      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle>Register New Entity</DialogTitle>
        <DialogContent>
          <FormControl
            fullWidth
            margin="dense"
            error={!!validationErrors.store_id}
            sx={{ mb: 2 }}
          >
            <InputLabel>Select Store</InputLabel>
            <Select
              name="store_id"
              value={entityData.store_id}
              onChange={handleChange}
              label="Select Store"
            >
              {stores.map((store) => (
                <MenuItem key={store.id || store.store_id} value={store.id || store.store_id}>
                  Store {store.id || store.store_id} - {store.db} ({store.embeddings_table})
                </MenuItem>
              ))}
            </Select>
            {validationErrors.store_id && (
              <FormHelperText>{validationErrors.store_id}</FormHelperText>
            )}
          </FormControl>

          <TextField
            autoFocus
            margin="dense"
            name="entity"
            label="Entity Name"
            type="text"
            fullWidth
            variant="outlined"
            value={entityData.entity}
            onChange={handleChange}
            error={!!validationErrors.entity}
            helperText={validationErrors.entity || 'Lowercase, alphanumeric and underscores only (e.g., product, catalog, user)'}
            sx={{ mb: 2 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} color="secondary">
            Cancel
          </Button>
          <Button onClick={handleSubmit} color="primary">
            Submit Request
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Entity Details Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CategoryIcon sx={{ color: '#522b4a' }} />
            Entity Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedEntity && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 2 }}>
              {/* Entity Information Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                  Entity Information
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Request ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                      {selectedEntity.request_id || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Status
                    </Typography>
                    <Box sx={{ mt: 0.5 }}>
                      {getStatusChip(selectedEntity.status)}
                    </Box>
                  </Box>
                </Box>
              </Box>

              {/* Entity Configuration Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(255, 152, 0, 0.02)', borderRadius: 1, border: '1px solid rgba(255, 152, 0, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <CategoryIcon fontSize="small" sx={{ color: '#ff9800' }} />
                  Entity Configuration
                </Typography>
                
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Entity Name
                    </Typography>
                    <Chip 
                      label={selectedEntity.payload?.entity || 'N/A'}
                      size="small"
                      sx={{ 
                        backgroundColor: '#fff3e0',
                        color: '#ff9800',
                        fontWeight: 600,
                        mt: 0.5
                      }}
                    />
                  </Box>

                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Associated Store ID
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <StorageIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      <Typography component="span" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', p: 1, borderRadius: 0.5 }}>
                        {selectedEntity.payload?.store_id || 'N/A'}
                      </Typography>
                    </Typography>
                  </Box>
                </Box>
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
                      {selectedEntity.created_by || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created At
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      {selectedEntity.created_at ? new Date(selectedEntity.created_at).toLocaleString() : 'N/A'}
                    </Typography>
                  </Box>
                </Box>

                {selectedEntity.reason && selectedEntity.reason !== 'N/A' && (
                  <Box sx={{ mt: 2 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Additional Notes
                    </Typography>
                    <Typography variant="body1" sx={{ mt: 0.5, p: 1, backgroundColor: '#f9f9f9', borderRadius: 0.5, fontStyle: 'italic' }}>
                      {selectedEntity.reason}
                    </Typography>
                  </Box>
                )}
              </Box>
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
        </DialogActions>
      </Dialog>

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

export default EntityRegistry;