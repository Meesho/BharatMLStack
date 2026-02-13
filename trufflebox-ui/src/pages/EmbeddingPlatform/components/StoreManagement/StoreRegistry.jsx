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
  FormHelperText,
  IconButton,
  Tooltip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import StorageIcon from '@mui/icons-material/Storage';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import PersonIcon from '@mui/icons-material/Person';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { 
  validateStorePayload, 
  getDefaultFormValues
} from '../../../../services/embeddingPlatform/utils';
import { 
  StatusChip, 
  StatusFilterHeader, 
  ViewDetailModal,
  useTableFilter,
  useStatusFilter
} from '../shared';

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
  const { user } = useAuth();
  
  const { selectedStatuses, handleStatusChange } = useStatusFilter(['PENDING', 'APPROVED', 'REJECTED']);

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
    const defaultValues = getDefaultFormValues('store');
    // Hardcode conf_id to 1 and db to 'scylla'
    setStoreData({
      ...defaultValues,
      conf_id: '1',
      db: 'scylla'
    });
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
        reason: `Registering new store: 1`,
        request_type: "CREATE",
        payload: {
          conf_id: parseInt(storeData.conf_id, 10) || 1, // Ensure it's sent as integer
          db: 'scylla',
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

  // Use shared table filter hook
  const filteredStores = useTableFilter({
    data: stores,
    searchQuery,
    selectedStatuses,
    searchFields: (store) => [
      store.request_id,
      store.payload?.conf_id,
      store.payload?.db,
      store.payload?.embeddings_table,
      store.payload?.aggregator_table,
      store.status
    ],
    sortField: 'created_at',
    sortOrder: 'desc'
  });


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
                    <StatusChip status={store.status} />
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
              value="1"
              disabled
              fullWidth
            />

            <TextField
              name="db"
              label="Database Type"
              value="scylla"
              disabled
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
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedStore}
        config={{
          title: 'Store Details',
          icon: StorageIcon,
          sections: [
            {
              title: 'Store Information',
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
          ]
        }}
      />
    </Paper>
  );
};

export default StoreRegistry;