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
  Chip,
  IconButton,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Snackbar,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import FilterListIcon from '@mui/icons-material/FilterList';
import InfoIcon from '@mui/icons-material/Info';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CategoryIcon from '@mui/icons-material/Category';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const FilterRegistry = () => {
  const [filterRequests, setFilterRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['APPROVED', 'PENDING', 'REJECTED']);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedFilter, setSelectedFilter] = useState(null);
  const [filterData, setFilterData] = useState({
    entity: '',
    reason: '',
    column_name: '',
    filter_value: '',
    default_value: ''
  });
  const [entities, setEntities] = useState([]);
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();
  const [validationErrors, setValidationErrors] = useState({});

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const [filterRequestsResponse, entitiesResponse] = await Promise.all([
        embeddingPlatformAPI.getFilterRequests(),
        embeddingPlatformAPI.getEntities()
      ]);

      if (filterRequestsResponse.filter_requests) {
        setFilterRequests(filterRequestsResponse.filter_requests);
      } else {
        setFilterRequests([]);
      }

      if (entitiesResponse.entities) {
        const availableEntities = entitiesResponse.entities?.map(entity => ({
          name: entity.name,
          store_id: entity.store_id,
          label: `${entity.name} (Store ${entity.store_id})`
        })) || [];
        setEntities(availableEntities);
      }
    } catch (error) {
      console.error('Error fetching filter data:', error);
      setError('Failed to load filter data. Please refresh the page.');
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

  const handleViewRequest = (filter) => {
    setSelectedFilter(filter);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedFilter(null);
  };

  const handleOpen = () => {
    setOpen(true);
    setFilterData({
      entity: '',
      reason: '',
      column_name: '',
      filter_value: '',
      default_value: ''
    });
    setValidationErrors({});
  };

  const handleClose = () => {
    setOpen(false);
    setFilterData({
      entity: '',
      reason: '',
      column_name: '',
      filter_value: '',
      default_value: ''
    });
    setValidationErrors({});
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFilterData(prev => ({
      ...prev,
      [name]: value
    }));
    
    // Clear validation error for this field
    if (validationErrors[name]) {
      setValidationErrors(prev => ({
        ...prev,
        [name]: ''
      }));
    }
  };

  const validateForm = () => {
    const errors = {};
    
    if (!filterData.entity) {
      errors.entity = 'Entity is required';
    }
    
    if (!filterData.reason) {
      errors.reason = 'Reason is required';
    } else if (filterData.reason.length < 10) {
      errors.reason = 'Reason should be at least 10 characters';
    }
    
    if (!filterData.column_name) {
      errors.column_name = 'Column name is required';
    } else if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(filterData.column_name)) {
      errors.column_name = 'Column name should start with a letter and contain only letters, numbers, and underscores';
    }
    
    if (!filterData.filter_value) {
      errors.filter_value = 'Filter value is required';
    }
    
    if (!filterData.default_value) {
      errors.default_value = 'Default value is required';
    }
    
    return errors;
  };

  const handleSubmit = async () => {
    // Check if required data is available
    if (entities.length === 0) {
      showNotification('No approved entities available. Please create and approve entities first.', "error");
      return;
    }

    const errors = validateForm();
    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors);
      showNotification('Please fix the validation errors and try again.', "error");
      return;
    }

    try {
      setLoading(true);
      const payload = {
        requestor: user?.email || 'user@example.com',
        reason: filterData.reason,
        payload: {
          entity: filterData.entity,
          filter: {
            column_name: filterData.column_name,
            filter_value: filterData.filter_value,
            default_value: filterData.default_value
          }
        }
      };
      
      const response = await embeddingPlatformAPI.registerFilter(payload);
      
      if (response) {
        showNotification(response.message || 'Filter registration request submitted successfully', "success");
        handleClose();
        fetchData();
      }
    } catch (error) {
      console.error('Error submitting filter request:', error);
      showNotification(error.message || 'Failed to submit filter registration request', "error");
    } finally {
      setLoading(false);
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
          <Typography variant="h6">Filter Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage filter registration requests
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
          label="Search Filters"
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
          Register Filter
        </Button>
      </Box>

      {/* Filters Table */}
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
                  position: 'relative'
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
                    {filterRequests.length === 0 ? 'No filter requests found' : 'No filters match your search'}
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

      {/* Filter Registration Dialog */}
      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle>Register Filter</DialogTitle>
        <DialogContent>
          {entities.length === 0 && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              No approved entities available. Please create and approve entities first before registering filters.
            </Alert>
          )}

          <Box sx={{ mt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <FormControl fullWidth required error={!!validationErrors.entity} size="small">
                  <InputLabel>Entity</InputLabel>
                  <Select
                    name="entity"
                    value={filterData.entity}
                    onChange={handleChange}
                    label="Entity"
                  >
                    {entities.map((entity) => (
                      <MenuItem key={entity.name} value={entity.name}>
                        {entity.label}
                      </MenuItem>
                    ))}
                  </Select>
                  {validationErrors.entity && (
                    <FormHelperText>{validationErrors.entity}</FormHelperText>
                  )}
                </FormControl>
              </Grid>

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  required
                  multiline
                  rows={2}
                  size="small"
                  name="reason"
                  label="Reason for Registration"
                  value={filterData.reason}
                  onChange={handleChange}
                  error={!!validationErrors.reason}
                  helperText={validationErrors.reason || "Explain why this filter is needed (e.g., Adding GST filter for catalog products to support tax-aware recommendations)"}
                  placeholder="Adding GST filter for catalog products to support tax-aware recommendations"
                />
              </Grid>

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  required
                  size="small"
                  name="column_name"
                  label="Column Name"
                  value={filterData.column_name}
                  onChange={handleChange}
                  error={!!validationErrors.column_name}
                  helperText={validationErrors.column_name || "Use snake_case (e.g., is_active, category_id)"}
                  placeholder="is_active"
                />
              </Grid>

              <Grid item xs={6}>
                <TextField
                  fullWidth
                  required
                  size="small"
                  name="filter_value"
                  label="Filter Value"
                  value={filterData.filter_value}
                  onChange={handleChange}
                  error={!!validationErrors.filter_value}
                  helperText={validationErrors.filter_value || "Value when filter is applied"}
                  placeholder="true"
                />
              </Grid>

              <Grid item xs={6}>
                <TextField
                  fullWidth
                  required
                  size="small"
                  name="default_value"
                  label="Default Value"
                  value={filterData.default_value}
                  onChange={handleChange}
                  error={!!validationErrors.default_value}
                  helperText={validationErrors.default_value || "Value when filter is not applied"}
                  placeholder="false"
                />
              </Grid>
            </Grid>
          </Box>
        </DialogContent>
        <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
          <Button
            variant="outlined"
            onClick={handleClose}
            sx={{
              borderColor: '#522b4a',
              color: '#522b4a',
              '&:hover': {
                borderColor: '#613a5c',
                backgroundColor: 'rgba(82, 43, 74, 0.04)',
              },
            }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={loading}
            sx={{
              backgroundColor: '#522b4a',
              color: 'white',
              '&:hover': {
                backgroundColor: '#613a5c',
              },
              '&:disabled': {
                backgroundColor: '#cccccc',
              },
            }}
          >
            {loading ? 'Processing...' : 'Submit Request'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Filter Details Modal */}
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedFilter}
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
            }
          ]
        }}
      />

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

export default FilterRegistry;