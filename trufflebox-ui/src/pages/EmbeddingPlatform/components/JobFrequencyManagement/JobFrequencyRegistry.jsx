import React, { useState, useEffect } from 'react';
import {
  Paper,
  Box,
  Typography,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Snackbar,
  Alert,
  CircularProgress,
  IconButton,
  Grid,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormControlLabel,
  Switch,
  FormHelperText,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ScheduleIcon from '@mui/icons-material/Schedule';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { JOB_FREQUENCIES } from '../../../../services/embeddingPlatform/constants';
import { useNotification } from '../shared/hooks/useNotification';
import { useStatusFilter, useTableFilter, StatusChip, StatusFilterHeader, ViewDetailModal } from '../shared';

const JobFrequencyRegistry = () => {
  const [frequencyRequests, setFrequencyRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const { selectedStatuses, setSelectedStatuses, handleStatusChange } = useStatusFilter(['APPROVED', 'PENDING', 'REJECTED']);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedFrequency, setSelectedFrequency] = useState(null);
  const { user } = useAuth();
  const { notification, showNotification, closeNotification } = useNotification();

  const [frequencyData, setFrequencyData] = useState({
    job_frequency: '',
    reason: '',
    use_custom: false // New state for custom frequency toggle
  });
  const [validationErrors, setValidationErrors] = useState({});

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const response = await embeddingPlatformAPI.getJobFrequencyRequests();
      
      if (response.job_frequency_requests) {
        setFrequencyRequests(response.job_frequency_requests);
      } else {
        setFrequencyRequests([]);
      }
    } catch (error) {
      console.error('Error fetching job frequency requests:', error);
      setError('Failed to load job frequency requests. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const filteredRequests = useTableFilter({
    data: frequencyRequests,
    searchQuery,
    selectedStatuses,
    searchFields: (request) => [
      request.request_id,
      request.payload?.job_frequency,
      request.created_by,
      request.status,
    ],
    sortField: 'created_at',
    sortOrder: 'desc',
  });

  const handleViewFrequency = (frequency) => {
    setSelectedFrequency(frequency);
    setShowViewModal(true);
  };

  const handleCloseViewModal = () => {
    setShowViewModal(false);
    setSelectedFrequency(null);
  };

  const handleOpen = () => {
    setFrequencyData({
      job_frequency: '',
      reason: '',
      use_custom: false
    });
    setValidationErrors({});
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
    setValidationErrors({});
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    
    if (name === 'use_custom') {
      const useCustom = e.target.checked;
      setFrequencyData(prev => ({
        ...prev,
        use_custom: useCustom,
        job_frequency: useCustom ? '' : prev.job_frequency // Clear job_frequency if switching to custom
      }));
      setValidationErrors(prev => ({ ...prev, job_frequency: '' })); // Clear validation error
    } else {
      setFrequencyData(prev => ({
        ...prev,
        [name]: value
      }));
    }
    
    // Clear validation error when user types
    if (validationErrors[name]) {
      setValidationErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const validateFrequencyFormat = (frequency) => {
    // Validate FREQ_{number}{unit} format where unit can be D, H, W, M
    const regex = /^FREQ_\d+[DHWM]$/;
    return regex.test(frequency);
  };

  const validateForm = () => {
    const errors = {};
    
    if (!frequencyData.job_frequency) {
      errors.job_frequency = 'Job frequency is required';
    } else if (frequencyData.use_custom && !validateFrequencyFormat(frequencyData.job_frequency)) {
      errors.job_frequency = 'Invalid format. Use FREQ_{number}{unit} (e.g., FREQ_1D, FREQ_2W, FREQ_6H)';
    }
    
    if (!frequencyData.reason) {
      errors.reason = 'Reason is required';
    }
    
    return errors;
  };

  const handleSubmit = async () => {
    const errors = validateForm();
    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors);
      return;
    }

    try {
      const payload = {
        requestor: user?.email,
        reason: frequencyData.reason,
        request_type: "CREATE",
        payload: {
          job_frequency: frequencyData.job_frequency
        }
      };

      await embeddingPlatformAPI.registerJobFrequency(payload);
      
      showNotification('Job frequency registration submitted successfully!', 'success');
      
        handleClose();
      fetchData(); // Refresh the data
    } catch (error) {
      console.error('Error registering job frequency:', error);
      showNotification(error.message || 'Failed to register job frequency. Please try again.', 'error');
    }
  };

  const generateDescription = (frequency) => {
    if (!frequency) return 'N/A';
    
    const match = frequency.match(/^FREQ_(\d+)([DHWM])$/);
    if (!match) return 'Custom frequency pattern';
    
    const [, number, unit] = match;
    const unitMap = {
      'D': 'day(s)',
      'H': 'hour(s)', 
      'W': 'week(s)',
      'M': 'month(s)'
    };
    
    return `Runs every ${number} ${unitMap[unit] || unit}`;
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
          <Typography variant="h6">Job Frequency Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage job frequency registration requests
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
          label="Search Job Frequency Requests"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: <SearchIcon sx={{ color: 'action.active', mr: 1 }} />,
          }}
          sx={{ width: 300 }}
          size="small"
        />
            <Button
              variant="contained"
          startIcon={<AddIcon />}
              onClick={handleOpen}
              sx={{
                backgroundColor: '#522b4a',
                '&:hover': { backgroundColor: '#613a5c' },
              }}
            >
          Register Job Frequency
            </Button>
          </Box>

      {/* Table */}
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
                <StatusFilterHeader selectedStatuses={selectedStatuses} onStatusChange={handleStatusChange} />
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
                    {frequencyRequests.length === 0 ? 'No job frequency requests available' : 'No requests match your search and filters'}
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
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', px: 1, borderRadius: 1, display: 'inline-block' }}>
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
                      onClick={() => handleViewFrequency(request)}
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

      {/* Registration Modal */}
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle sx={{ color: '#522b4a', fontWeight: 600 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ScheduleIcon />
            Register Job Frequency
                        </Box>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <Grid container spacing={2}>
              {/* Custom frequency toggle */}
              <Grid item xs={12}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={frequencyData.use_custom}
                      onChange={(e) => {
                        const useCustom = e.target.checked;
                        setFrequencyData(prev => ({
                          ...prev,
                          use_custom: useCustom,
                          job_frequency: useCustom ? '' : prev.job_frequency
                        }));
                        setValidationErrors(prev => ({ ...prev, job_frequency: '' }));
                      }}
                      name="use_custom"
                    />
                  }
                  label="Use Custom Frequency Pattern"
                  sx={{ mb: 2 }}
                />
              </Grid>

              {/* Job Frequency Input */}
              <Grid item xs={12}>
                {!frequencyData.use_custom ? (
          <FormControl 
            fullWidth 
                    required 
                    size="small" 
            error={!!validationErrors.job_frequency}
                    sx={{ mb: 2 }}
          >
                    <InputLabel>Job Frequency</InputLabel>
            <Select
              name="job_frequency"
              value={frequencyData.job_frequency}
              onChange={handleChange}
                      label="Job Frequency"
            >
              {JOB_FREQUENCIES.map((freq) => (
                <MenuItem key={freq.id} value={freq.id}>
                          <Box>
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>
                              {freq.label} ({freq.id})
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {freq.description}
                            </Typography>
                          </Box>
                </MenuItem>
              ))}
            </Select>
            {validationErrors.job_frequency && (
              <FormHelperText>{validationErrors.job_frequency}</FormHelperText>
            )}
          </FormControl>
                ) : (
                  <TextField
                    fullWidth
                    required
                    size="small"
                    name="job_frequency"
                    label="Custom Job Frequency"
                    value={frequencyData.job_frequency}
                    onChange={handleChange}
                    error={!!validationErrors.job_frequency}
                    helperText={validationErrors.job_frequency || "Enter frequency in format FREQ_{number}{unit} (e.g., FREQ_3D, FREQ_8H)"}
                    placeholder="FREQ_3D"
                    sx={{ mb: 2 }}
                  />
                )}
              </Grid>

              {/* Reason */}
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  required
                  multiline
                  rows={3}
                  size="small"
                  name="reason"
                  label="Reason for Registration"
                  value={frequencyData.reason}
                  onChange={handleChange}
                  error={!!validationErrors.reason}
                  helperText={validationErrors.reason || "Explain why this frequency is needed"}
                  placeholder="Adding this frequency for high-priority models requiring specialized training schedules"
                />
              </Grid>
            </Grid>

            {/* Preview */}
            {frequencyData.job_frequency && (
              <Alert severity="success" sx={{ mt: 2 }}>
            <Typography variant="body2">
                  <strong>Preview:</strong> {generateDescription(frequencyData.job_frequency)}
            </Typography>
          </Alert>
          )}
          </Box>
        </DialogContent>
        <DialogActions sx={{ backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button
            onClick={handleClose}
            sx={{
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            sx={{
              backgroundColor: '#522b4a',
              '&:hover': { backgroundColor: '#613a5c' }
            }}
          >
            Register Job Frequency
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Modal */}
      <ViewDetailModal
        open={showViewModal}
        onClose={handleCloseViewModal}
        data={selectedFrequency}
        config={{
          title: 'Job Frequency Request Details',
          icon: ScheduleIcon,
          sections: [
            {
              title: 'Request Information',
              icon: ScheduleIcon,
              layout: 'grid',
              fields: [
                { label: 'Request ID', key: 'request_id', type: 'monospace' },
                { label: 'Status', key: 'status', type: 'status' },
                { label: 'Created At', key: 'created_at', type: 'date' },
                { label: 'Created By', key: 'created_by' }
              ]
            },
            {
              title: 'Frequency Configuration',
              icon: ScheduleIcon,
              backgroundColor: 'rgba(25, 118, 210, 0.02)',
              borderColor: 'rgba(25, 118, 210, 0.1)',
              render: (data) => (
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Job Frequency</Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', px: 1, borderRadius: 1, display: 'inline-block', mt: 0.5 }}>
                      {data?.payload?.job_frequency || 'N/A'}
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Description</Typography>
                    <Typography variant="body1" sx={{ mt: 0.5 }}>{generateDescription(data?.payload?.job_frequency)}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Reason</Typography>
                    <Typography variant="body1" sx={{ mt: 0.5 }}>{data?.payload?.reason || 'N/A'}</Typography>
                  </Grid>
                </Grid>
              )
            }
          ]
        }}
      />

        {/* Toast Notification */}
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

export default JobFrequencyRegistry;