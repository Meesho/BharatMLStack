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
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import FilterListIcon from '@mui/icons-material/FilterList';
import StorageIcon from '@mui/icons-material/Storage';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { 
  validateStorePayload, 
  getDefaultFormValues
} from '../../../../services/embeddingPlatform/utils';

const StoreRegistry = () => {
  const [stores, setStores] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');
  const [selectedStore, setSelectedStore] = useState(null);
  const [showViewModal, setShowViewModal] = useState(false);
  const [open, setOpen] = useState(false);
  const [storeData, setStoreData] = useState(getDefaultFormValues('store'));
  const [validationErrors, setValidationErrors] = useState({});
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING', 'APPROVED', 'REJECTED']); // Initialize with all statuses
  const { user } = useAuth();

  useEffect(() => {
    fetchStores();
  }, []);

  const fetchStores = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getStoreRequests();
      
      // Get all store requests from API
      if (response.store_requests) {
        const storesData = response.store_requests || [];
        setStores(storesData);
      } else {
        setStores([]);
      }
    } catch (error) {
      console.error('Error fetching store requests:', error);
      setError('Failed to load store requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const handleViewStore = (store) => {
    setSelectedStore(store);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedStore(null);
  };

  const handleOpen = () => {
    setStoreData(getDefaultFormValues('store'));
    setValidationErrors({});
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setStoreData(prev => ({
      ...prev,
      [name]: value
    }));
    
    // Clear validation error when user starts typing
    if (validationErrors[name]) {
      setValidationErrors(prev => ({
        ...prev,
        [name]: undefined
      }));
    }
  };

  const handleSubmit = async () => {
    const validation = validateStorePayload(storeData);
    if (!validation.isValid) {
      setValidationErrors(validation.errors);
      return;
    }

    try {
      setValidationErrors({});
      const payload = {
        requestor: user.email,
        reason: `Registering new store: ${storeData.conf_id}`,
        request_type: "CREATE",
        payload: {
          conf_id: parseInt(storeData.conf_id),
          db: storeData.db,
          embeddings_table: storeData.embeddings_table,
          aggregator_table: storeData.aggregator_table
        }
      };

      await embeddingPlatformAPI.registerStore(payload);
      
      setOpen(false);
      fetchStores(); // Refresh the data
    } catch (error) {
      console.error('Error submitting store registration:', error);
      setError(error.message || 'Failed to submit store registration request');
    }
  };

  // Filtered and sorted data using useMemo
  const filteredStores = useMemo(() => {
    let filtered = [...stores];
    
    // Status filtering
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(store => 
        selectedStatuses.includes(store.status?.toUpperCase())
      );
    }
    
    // Search filtering
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(store => {
        return (
          String(store.request_id || '').toLowerCase().includes(searchLower) ||
          String(store.payload?.conf_id || '').toLowerCase().includes(searchLower) ||
          String(store.payload?.db || '').toLowerCase().includes(searchLower) ||
          String(store.payload?.embeddings_table || '').toLowerCase().includes(searchLower) ||
          String(store.payload?.aggregator_table || '').toLowerCase().includes(searchLower) ||
          (store.status && store.status.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at) - new Date(a.created_at);
    });
  }, [stores, searchQuery, selectedStatuses]);

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
          <Typography variant="h6">Store Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage store registration requests
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
          label="Search Stores"
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
            minWidth: '200px'
          }}
        >
          Register Store
        </Button>
      </Box>

      {/* Stores Table */}
      <TableContainer 
        component={Paper} 
        elevation={3} 
        sx={{ 
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 900,
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
            {filteredStores.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {stores.length === 0 ? 'No store requests available' : 'No stores match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredStores.map((store, index) => (
                <TableRow key={store.id || index} hover>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.request_id || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.payload.conf_id || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.payload.db || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.payload.embeddings_table || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.payload.aggregator_table || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {getStatusChip(store.status)}
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      <Tooltip title="View Details">
                        <IconButton 
                          onClick={() => handleViewStore(store)} 
                          size="small"
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

      {/* Register Store Modal */}
      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle>Register New Store</DialogTitle>
        <DialogContent>
          <Box sx={{ mb: 2, p: 2, bgcolor: '#e3f2fd', borderRadius: 1 }}>
            <Typography variant="body2" color="text.secondary">
              Register a new data store for the Embedding Platform. All fields are required.
            </Typography>
          </Box>

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
            <TextField
              name="conf_id"
              label="Configuration ID"
              value={storeData.conf_id}
              onChange={handleChange}
              error={!!validationErrors.conf_id}
              helperText={validationErrors.conf_id}
              placeholder="e.g., 2"
              fullWidth
            />

            <TextField
              name="db"
              label="Database Type"
              value={storeData.db}
              onChange={handleChange}
              error={!!validationErrors.db}
              helperText={validationErrors.db}
              placeholder="e.g., POSTGRESQL"
              fullWidth
            />

            <TextField
              name="embeddings_table"
              label="Embeddings Table"
              value={storeData.embeddings_table}
              onChange={handleChange}
              error={!!validationErrors.embeddings_table}
              helperText={validationErrors.embeddings_table}
              placeholder="e.g., embeddings_table_name"
              fullWidth
            />

            <TextField
              name="aggregator_table"
              label="Aggregator Table"
              value={storeData.aggregator_table}
              onChange={handleChange}
              error={!!validationErrors.aggregator_table}
              helperText={validationErrors.aggregator_table}
              placeholder="e.g., aggregator_table_name"
              fullWidth
            />

            {Object.keys(validationErrors).length > 0 && (
              <Box sx={{ mt: 1 }}>
                <FormHelperText error>
                  Please correct the errors above before submitting.
                </FormHelperText>
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancel</Button>
          <Button 
            variant="contained" 
            onClick={handleSubmit}
            sx={{
              backgroundColor: '#450839',
              '&:hover': { backgroundColor: '#380730' },
            }}
          >
            Submit Request
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Store Details Modal */}
      <Dialog open={showViewModal} onClose={handleCloseViewModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <StorageIcon sx={{ color: '#522b4a' }} />
            Store Details
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedStore && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 2 }}>
              {/* Request Information Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                  Store Information
                </Typography>
                
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Request ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                      {selectedStore.request_id || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Status
                    </Typography>
                    <Box sx={{ mt: 0.5 }}>
                      {getStatusChip(selectedStore.status)}
                    </Box>
                  </Box>
                </Box>
              </Box>

              {/* Configuration Section */}
              <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                  Store Configuration
                </Typography>
                
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Configuration ID
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>
                      {selectedStore.payload?.conf_id || 'N/A'}
                    </Typography>
                  </Box>

                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Database Type
                    </Typography>
                    <Chip 
                      label={selectedStore.payload?.db || 'N/A'}
                      size="small"
                      sx={{ 
                        backgroundColor: '#e3f2fd',
                        color: '#1976d2',
                        fontWeight: 600,
                        mt: 0.5
                      }}
                    />
                  </Box>

                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Embeddings Table
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', p: 1, borderRadius: 0.5, mt: 0.5 }}>
                      {selectedStore.payload?.embeddings_table || 'N/A'}
                    </Typography>
                  </Box>

                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Aggregator Table
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', p: 1, borderRadius: 0.5, mt: 0.5 }}>
                      {selectedStore.payload?.aggregator_table || 'N/A'}
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
                      {selectedStore.created_by || 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Created At
                    </Typography>
                    <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#1976d2' }} />
                      {selectedStore.created_at ? new Date(selectedStore.created_at).toLocaleString() : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
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
    </Paper>
  );
};

export default StoreRegistry;