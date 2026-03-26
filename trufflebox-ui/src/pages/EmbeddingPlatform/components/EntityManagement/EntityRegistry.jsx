import React, { useState, useEffect, useCallback } from 'react';
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
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Snackbar,
  Chip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CategoryIcon from '@mui/icons-material/Category';
import InfoIcon from '@mui/icons-material/Info';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import StorageIcon from '@mui/icons-material/Storage';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { validateEntityPayload, getDefaultFormValues } from '../../../../services/embeddingPlatform/utils';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

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
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['PENDING', 'APPROVED', 'REJECTED']);
  const { notification, showNotification, closeNotification } = useNotification();
  const { user } = useAuth();

  const fetchStores = useCallback(async () => {
    try {
      const response = await embeddingPlatformAPI.getStores();
      if (response.stores) {
        setStores(response.stores);
      }
    } catch (error) {
      console.error('Error fetching stores:', error);
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

  const filteredEntities = useTableFilter({
    data: entities,
    searchQuery,
    selectedStatuses,
    searchFields: (entity) => [
      entity.request_id,
      entity.payload?.entity,
      entity.payload?.store_id,
      entity.status,
    ],
    sortField: 'created_at',
    sortOrder: 'desc',
  });

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
            minWidth: 'fit-content'
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
                    <StatusChip status={entity.status} />
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
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedEntity}
        config={{
          title: 'Entity Details',
          icon: CategoryIcon,
          sections: [
            {
              title: 'Entity Information',
              icon: InfoIcon,
              layout: 'grid',
              fields: [
                { label: 'Request ID', key: 'request_id', type: 'monospace' },
                { label: 'Status', key: 'status', type: 'status' }
              ]
            },
            {
              title: 'Entity Configuration',
              icon: CategoryIcon,
              backgroundColor: 'rgba(255, 152, 0, 0.02)',
              borderColor: 'rgba(255, 152, 0, 0.1)',
              fields: [
                { label: 'Entity Name', key: 'payload.entity', type: 'chip' },
                { label: 'Associated Store ID', key: 'payload.store_id', type: 'monospace' }
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
            },
            {
              title: 'Additional Notes',
              icon: InfoIcon,
              render: (data) => {
                if (!data?.reason || data.reason === 'N/A') return null;
                return (
                  <Typography variant="body1" sx={{ mt: 0.5, p: 1, backgroundColor: '#f9f9f9', borderRadius: 0.5, fontStyle: 'italic' }}>
                    {data.reason}
                  </Typography>
                );
              }
            }
          ]
        }}
      />

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

export default EntityRegistry;