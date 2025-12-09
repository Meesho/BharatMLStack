import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Paper,
  Chip,
  IconButton,
  Tooltip,
  Box,
  TextField,
  Button,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Skeleton,
  Snackbar,
  Alert,
  Popover,
  Stack,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import ValidateIcon from '@mui/icons-material/CheckCircle';
import SearchIcon from '@mui/icons-material/Search';
import HistoryIcon from '@mui/icons-material/History';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CloseIcon from '@mui/icons-material/Close';
import DataObjectIcon from '@mui/icons-material/DataObject';
import EventAvailableIcon from '@mui/icons-material/EventAvailable';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import BlockIcon from '@mui/icons-material/Block';
import FilterListIcon from '@mui/icons-material/FilterList';
import RefreshIcon from '@mui/icons-material/Refresh';
import CachedIcon from '@mui/icons-material/Cached';
import { useAuth } from '../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../constants/permissions';
import * as URL_CONSTANTS from '../../../config';
import JsonViewer from '../../../components/JsonViewer';

const GenericModelConfigApprovalTable = ({ onReview }) => {
  const { user, hasPermission } = useAuth();
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.MP_CONFIG_APPROVAL;
  const [configData, setConfigData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [activityModalOpen, setActivityModalOpen] = useState(false);
  const [selectedActivity, setSelectedActivity] = useState(null);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const [approveDialogOpen, setApproveDialogOpen] = useState(false);
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [processingAction, setProcessingAction] = useState(false);
  const [currentRequestId, setCurrentRequestId] = useState(null);
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'info'
  });
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING APPROVAL']); // Initialize with PENDING APPROVAL only
  
  const getStatusChipStyle = (status) => {
    switch(status?.toUpperCase()) {
      case 'APPROVED':
        return {
          backgroundColor: '#E8F5E9',
          color: '#2E7D32',
          fontWeight: 'bold',
        };
      case 'REJECTED':
        return {
          backgroundColor: '#FFEBEE',
          color: '#D32F2F',
          fontWeight: 'bold',
        };
      case 'CANCELLED':
        return {
          backgroundColor: '#FAFAFA',
          color: '#757575',
          fontWeight: 'bold',
        };
      case 'PENDING APPROVAL':
        return {
          backgroundColor: '#FFF3E0',
          color: '#E65100',
          fontWeight: 'bold',
        };
      case 'INPROGRESS':
      case 'IN PROGRESS':
        return {
          backgroundColor: '#E3F2FD',
          color: '#1976D2',
          fontWeight: 'bold',
        };
      case 'FAILED':
        return {
          backgroundColor: '#FFCDD2',
          color: '#C62828',
          fontWeight: 'bold',
        };
      default:
        return {
          backgroundColor: '#F5F5F5',
          color: '#616161',
          fontWeight: 'bold',
        };
    }
  };

  const StatusColumnHeader = () => {
    const [anchorEl, setAnchorEl] = useState(null);
    
    const statusOptions = [
      { value: 'PENDING APPROVAL', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
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
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <b>Status</b>
          <IconButton
            size="small"
            onClick={handleClick}
            sx={{ 
              p: 0.25,
              color: selectedStatuses.length > 0 ? 'primary.main' : 'text.secondary',
              '&:hover': { backgroundColor: 'action.hover' }
            }}
          >
            <FilterListIcon fontSize="small" />
            {selectedStatuses.length > 0 && (
              <Box
                sx={{
                  position: 'absolute',
                  top: 2,
                  right: 2,
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  backgroundColor: 'primary.main',
                }}
              />
            )}
          </IconButton>
        </Box>

        {selectedStatuses.length > 0 && (
          <Stack direction="row" spacing={0.5} sx={{ overflow: 'scroll', gap: '0.25rem', width: '100px' }}>
            {selectedStatuses.map((status) => {
              const statusConfig = statusOptions.find(opt => opt.value === status);
              return (
                <Chip
                  key={status}
                  label={statusConfig?.label || status}
                  size="small"
                  onDelete={() => setSelectedStatuses(prev => prev.filter(s => s !== status))}
                  sx={{
                    backgroundColor: statusConfig?.color || '#f5f5f5',
                    color: statusConfig?.textColor || '#666',
                    fontWeight: 'bold',
                    fontSize: '0.5rem',
                    padding: '0.25rem',
                    height: 22,
                    '& .MuiChip-deleteIcon': {
                      color: statusConfig?.textColor || '#666',
                      fontSize: '0.875rem',
                      '&:hover': {
                        color: statusConfig?.textColor || '#666',
                        opacity: 0.7
                      }
                    }
                  }}
                />
              );
            })}
          </Stack>
        )}
        
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
            sx: { minWidth: 250, maxWidth: 300 }
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

  const allColumns = [
    { 
      field: 'RequestId', 
      headerName: 'Request ID',
      width: '15%',
      stickyColumn: false
    },
    { 
      field: 'ConfigId', 
      headerName: 'Config ID',
      width: '20%',
      stickyColumn: false
    },
    { 
      field: 'RequestType', 
      headerName: 'Request Type',
      width: '15%',
      stickyColumn: false,
      render: (row) => (
        <Chip
          label={row.RequestType}
          sx={{
            backgroundColor: '#E3F2FD',
            color: '#1565C0',
            borderRadius: '16px',
            fontWeight: 'bold',
            fontSize: '0.75rem',
            height: '28px'
          }}
        />
      ),
    },
    {
      field: 'Status',
      headerName: <StatusColumnHeader />,
      width: '12%',
      stickyColumn: false,
      render: (row) => (
        <Chip
          label={row.Status}
          sx={{
            ...getStatusChipStyle(row.Status),
            fontSize: '0.75rem',
            height: '28px'
          }}
        />
      ),
    },
    { 
      field: 'CreatedBy', 
      headerName: 'Created By',
      width: '18%',
      stickyColumn: false,
      render: (row) => (
        <Typography variant="body2" sx={{ fontSize: '0.875rem' }}>
          {row.CreatedBy || 'N/A'}
        </Typography>
      ),
    },
    {
      field: 'Actions',
      headerName: 'Actions',
      width: '20%',
      stickyColumn: true,
      render: (row) => (
        <Box sx={{ 
          display: 'flex', 
          gap: 0.5, 
          alignItems: 'center',
          justifyContent: 'flex-start',
          flexWrap: 'nowrap',
          minWidth: 'max-content'
        }}>
          {/* View Configuration Details */}
          <Tooltip title="View Configuration Details" disableTransition>
            <IconButton 
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                // Create a more complete config object for the modal
                setSelectedConfig({
                  ...row,
                  Payload: row.Payload || {
                    config_value: row.config_value,
                    config_mapping: row.config_mapping
                  }
                });
                setConfigModalOpen(true);
              }}
            >
              <InfoIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* View Activity History */}
          <Tooltip title="View Activity History" disableTransition>
            <IconButton 
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                setSelectedActivity({
                  createdBy: row.CreatedBy,
                  updatedBy: row.UpdatedBy,
                  createdAt: row.CreatedAt,
                  updatedAt: row.UpdatedAt,
                  requestId: row.RequestId,
                  configId: row.ConfigId
                });
                setActivityModalOpen(true);
              }}
            >
              <EventAvailableIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      ),
    },
  ];

  const columns = allColumns.filter((col) => {
    if (col.roles) {
      return col.roles.includes(user?.role);
    }
    return true;
  });

  const fetchConfigData = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/inferflow-config-approval/configs`, {
        headers: {
          Authorization: `Bearer ${user?.token}`,
          'Content-Type': 'application/json'
        }
      });
      const data = await response.json();
      let rawData = [];
      if (Array.isArray(data)) {
        rawData = data;
      } else if (data.data && Array.isArray(data.data)) {
        rawData = data.data;
      } else if (typeof data === 'object') {
        rawData = [data];
      }
      
      // Transform the data to match expected field names
      const transformedData = rawData.map(item => {
        const transformed = {
          RequestId: item.request_id,
          ConfigId: item.config_id,
          RequestType: item.request_type,
          Status: item.status,
          CreatedBy: item.created_by,
          UpdatedBy: item.updated_by,
          CreatedAt: item.created_at,
          UpdatedAt: item.updated_at,
          Reviewer: item.approved_by || '',
          RejectReason: item.reject_reason,
          Payload: item.payload
        };
        return transformed;
      });
      
      setConfigData(transformedData);
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to load approval requests',
        severity: 'error'
      });
    } finally {
      setLoading(false);
    }
  }, [user?.token]);

  useEffect(() => {
    if (user?.token) {
      fetchConfigData();
    }
  }, [fetchConfigData, user?.token]);

  const filteredAndSortedData = useMemo(() => {
    let filtered = [...configData];
    
    // Status filtering
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(row => 
        selectedStatuses.includes(row.Status?.toUpperCase())
      );
    }
    
    // Search filtering
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.RequestId).toLowerCase().includes(searchLower) ||
          String(row.ConfigId).toLowerCase().includes(searchLower) ||
          (row.Status && row.Status.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.CreatedAt) - new Date(a.CreatedAt);
    });
  }, [configData, searchQuery, selectedStatuses]);

  const handleApproveRequest = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.APPROVE)) {
      setSnackbar({
        open: true,
        message: 'You do not have permission to approve configurations',
        severity: 'error'
      });
      return;
    }
    
    setCurrentRequestId(row.RequestId);
    setApproveDialogOpen(true);
  };

  const handleRejectRequest = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.REJECT)) {
      setSnackbar({
        open: true,
        message: 'You do not have permission to reject configurations',
        severity: 'error'
      });
      return;
    }
    
    setCurrentRequestId(row.RequestId);
    setRejectDialogOpen(true);
  };

  const confirmApprove = async () => {
    try {
      setProcessingAction(true);
      const success = await onReview(currentRequestId, 'Approved');
      if (success) {
        setSnackbar({
          open: true,
          message: 'Configuration approved successfully',
          severity: 'success'
        });
        fetchConfigData();
        setConfigModalOpen(false); // Close the view details modal
      } else {
        setSnackbar({
          open: true,
          message: 'Failed to approve configuration',
          severity: 'error'
        });
      }
    } finally {
      setProcessingAction(false);
      setApproveDialogOpen(false);
      setCurrentRequestId(null);
    }
  };

  const confirmReject = async () => {
    if (!rejectReason.trim()) {
      setSnackbar({
        open: true,
        message: 'Please provide a reason for rejection',
        severity: 'warning'
      });
      return;
    }

    try {
      setProcessingAction(true);
      const success = await onReview(currentRequestId, 'Rejected', rejectReason);
      if (success) {
        setSnackbar({
          open: true,
          message: 'Configuration rejected successfully',
          severity: 'success'
        });
        fetchConfigData();
        setConfigModalOpen(false); // Close the view details modal
      } else {
        setSnackbar({
          open: true,
          message: 'Failed to reject configuration',
          severity: 'error'
        });
      }
    } finally {
      setProcessingAction(false);
      setRejectDialogOpen(false);
      setCurrentRequestId(null);
      setRejectReason('');
    }
  };

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  const handleCloseSnackbar = () => {
    setSnackbar({ ...snackbar, open: false });
  };

  const tableCellStyles = {
    borderRight: '1px solid rgba(224, 224, 224, 1)',
    padding: '12px 16px',
    whiteSpace: 'normal',
    wordBreak: 'break-word',
    backgroundColor: 'white',
    '&:last-child': {
      borderRight: 'none',
    },
    '&.sticky-column': {
      position: 'sticky',
      right: 0,
      backgroundColor: 'inherit',
      zIndex: 2,
      boxShadow: '-4px 0 6px -2px rgba(0, 0, 0, 0.1)',
    }
  };

  return (
    <Box sx={{ p: 2 }}>
      
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
        <TextField
          placeholder="Search requests..."
          variant="outlined"
          size="small"
          value={searchQuery}
          onChange={handleSearchChange}
          sx={{ flex: 1 }}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              onClick={fetchConfigData}
              startIcon={<CachedIcon />}
              sx={{
                backgroundColor: '#450839',
                '&:hover': {
                  backgroundColor: '#380730',
                },
                whiteSpace: 'nowrap'
              }}
            >
              Refresh
            </Button>
          </Box>
      </Box>

      <Paper elevation={3}>
        <TableContainer sx={{ 
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1200,
            borderCollapse: 'separate',
            borderSpacing: 0,
          },
          '& .MuiTableHead-root': {
            position: 'sticky',
            top: 0,
            zIndex: 1,
            backgroundColor: '#E6EBF2',
          }
        }}>
        <Table>
          <TableHead>
            <TableRow>
              {columns.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={{
                    ...tableCellStyles,
                    backgroundColor: '#E6EBF2',
                    fontWeight: 'bold',
                    color: '#031022',
                    width: column.width,
                    ...(column.stickyColumn && {
                      '&.sticky-column': {
                        backgroundColor: '#E6EBF2',
                      }
                    }),
                    ...(column.stickyColumn && {
                      className: 'sticky-column'
                    })
                  }}
                >
                  {column.headerName}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              [...Array(5)].map((_, index) => (
                <TableRow key={index}>
                  {columns.map((col, colIndex) => (
                    <TableCell 
                      key={colIndex}
                      sx={{
                        ...tableCellStyles,
                        width: col.width,
                        ...(col.stickyColumn && {
                          className: 'sticky-column'
                        })
                      }}
                    >
                      <Skeleton animation="wave" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : filteredAndSortedData.length > 0 ? (
              filteredAndSortedData.map((row) => (
                <TableRow
                  key={row.RequestId}
                  sx={{
                    cursor: 'pointer',
                    backgroundColor: 'white',
                    '&:hover': {
                      backgroundColor: 'rgba(0, 0, 0, 0.04)',
                    },
                    '&:hover td.sticky-column': {
                      backgroundColor: 'rgba(0, 0, 0, 0.04)',
                    }
                  }}
                >
                  {columns.map((column) => (
                    <TableCell 
                      key={column.field}
                      sx={{
                        ...tableCellStyles,
                        width: column.width,
                        ...(column.stickyColumn && {
                          className: 'sticky-column'
                        })
                      }}
                    >
                      {column.render ? column.render(row) : row[column.field]}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} align="center" sx={{ py: 3 }}>
                  <Typography variant="body1" color="textSecondary">
                    No data found
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      </Paper>
      
      {/* Configuration Details Modal */}
      <Dialog
        open={configModalOpen}
        onClose={() => setConfigModalOpen(false)}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          pb: 2,
          pt: 2
        }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <DataObjectIcon />
            <Typography variant="h6">
              InferFlow Config Approval Details - {selectedConfig?.RequestId || 'N/A'}
            </Typography>
          </Box>
          <IconButton 
            onClick={() => setConfigModalOpen(false)} 
            sx={{ color: 'white' }}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers sx={{ p: 3 }}>
          {selectedConfig && (
            <Box sx={{ maxHeight: '70vh', overflow: 'auto' }}>
              {/* Request Details Section */}
              <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold', borderBottom: '2px solid #450839', pb: 1 }}>
                Request Details
              </Typography>
              
              <Box sx={{ mb: 3 }}>
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Request ID:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.RequestId}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Config ID:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.ConfigId || 'N/A'}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Request Type:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.RequestType}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Status:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    <Chip 
                      label={selectedConfig.Status} 
                      size="small" 
                      sx={getStatusChipStyle(selectedConfig.Status)}
                    />
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Reviewer:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.Reviewer || 'N/A'}
                  </Typography>
                </Box>
                
                {selectedConfig.Status?.toUpperCase() === 'REJECTED' && selectedConfig.RejectReason && (
                  <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                    <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                      Reject Reason:
                    </Typography>
                    <Typography variant="body1" sx={{ width: '60%', color: '#d32f2f' }}>
                      {selectedConfig.RejectReason}
                    </Typography>
                  </Box>
                )}
              </Box>

              {/* Activity History Section */}
              <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold', borderBottom: '2px solid #450839', pb: 1 }}>
                Activity History
              </Typography>
              
              <Box sx={{ mb: 3 }}>
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Created By:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.CreatedBy || 'N/A'}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Created At:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.CreatedAt ? new Date(selectedConfig.CreatedAt).toLocaleString() : 'N/A'}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Updated By:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.UpdatedBy || 'N/A'}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                  <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                    Updated At:
                  </Typography>
                  <Typography variant="body1" sx={{ width: '60%' }}>
                    {selectedConfig.UpdatedAt ? new Date(selectedConfig.UpdatedAt).toLocaleString() : 'N/A'}
                  </Typography>
                </Box>
              </Box>

              {/* Configuration Details Section */}
              <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold', borderBottom: '2px solid #450839', pb: 1 }}>
                Configuration Details
              </Typography>
              
              {selectedConfig.Payload && selectedConfig.Payload.config_value && (
                <Box sx={{ mb: 3 }}>
                  <JsonViewer
                    data={selectedConfig.Payload.config_value}
                    title="Config Value"
                    defaultExpanded={true}
                    editable={false}
                    maxHeight={400}
                    enableCopy={true}
                    enableDiff={false}
                    sx={{
                      '& .MuiPaper-root': {
                        border: '1px solid #e0e0e0',
                        borderRadius: 1
                      }
                    }}
                  />
                </Box>
              )}
              
              {selectedConfig.Payload && selectedConfig.Payload.config_mapping && (
                <Box sx={{ mb: 3 }}>
                  <JsonViewer
                    data={selectedConfig.Payload.config_mapping}
                    title="Config Mapping"
                    defaultExpanded={true}
                    editable={false}
                    maxHeight={400}
                    enableCopy={true}
                    enableDiff={false}
                    sx={{
                      '& .MuiPaper-root': {
                        border: '1px solid #e0e0e0',
                        borderRadius: 1
                      }
                    }}
                  />
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          {selectedConfig && selectedConfig.Status?.toUpperCase() === 'PENDING APPROVAL' && (
            <>
              {hasPermission(service, screenType, ACTIONS.APPROVE) && (
                <Button 
                  onClick={() => {
                    setCurrentRequestId(selectedConfig.RequestId);
                    setApproveDialogOpen(true);
                  }}
                  variant="contained"
                  color="success"
                  startIcon={<CheckCircleIcon />}
                  disabled={processingAction}
                  sx={{
                    backgroundColor: '#66bb6a',
                    textTransform: 'none',
                    '&:hover': {
                      backgroundColor: '#4caf50',
                    }
                  }}
                >
                  Approve Request
                </Button>
              )}
              {hasPermission(service, screenType, ACTIONS.REJECT) && (
                <Button 
                  onClick={() => {
                    setCurrentRequestId(selectedConfig.RequestId);
                    setRejectDialogOpen(true);
                  }}
                  variant="contained" 
                  color="error"
                  startIcon={<CancelIcon />}
                  disabled={processingAction}
                  sx={{
                    backgroundColor: '#ef5350',
                    textTransform: 'none',
                    '&:hover': {
                      backgroundColor: '#e53935',
                    }
                  }}
                >
                  Reject Request
                </Button>
              )}
            </>
          )}
        </DialogActions>
      </Dialog>

      {/* Activity History Modal */}
      <Dialog
        open={activityModalOpen}
        onClose={() => setActivityModalOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <HistoryIcon />
            <Typography variant="h6">Activity History</Typography>
          </Box>
          <IconButton 
            onClick={() => setActivityModalOpen(false)} 
            sx={{ color: 'white' }}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ p: 3 }}>
          {selectedActivity && (
            <Box>
              <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 500 }}>
                Request ID: {selectedActivity.requestId} | Config ID: {selectedActivity.configId}
              </Typography>
              
              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, mb: 3 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  CREATED
                </Typography>
                
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <PersonIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>By</Typography>
                    </Box>
                    <Typography variant="body2">{selectedActivity.createdBy}</Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {new Date(selectedActivity.createdAt).toLocaleDateString(undefined, { 
                        year: 'numeric', 
                        month: 'long', 
                        day: 'numeric'
                      })}
                    </Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {new Date(selectedActivity.createdAt).toLocaleTimeString(undefined, { 
                        hour: '2-digit', 
                        minute: '2-digit',
                        hour12: true
                      })}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
              
              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  LAST UPDATED
                </Typography>
                
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <PersonIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>By</Typography>
                    </Box>
                    <Typography variant="body2">{selectedActivity.updatedBy}</Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {new Date(selectedActivity.updatedAt).toLocaleDateString(undefined, { 
                        year: 'numeric', 
                        month: 'long', 
                        day: 'numeric'
                      })}
                    </Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {new Date(selectedActivity.updatedAt).toLocaleTimeString(undefined, { 
                        hour: '2-digit', 
                        minute: '2-digit',
                        hour12: true
                      })}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setActivityModalOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Approve Confirmation Dialog */}
      <Dialog
        open={approveDialogOpen}
        onClose={() => setApproveDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          alignItems: 'center',
          gap: 1
        }}>
          <CheckCircleIcon />
          Confirm Approval
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <Typography>
            Are you sure you want to approve this configuration request?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setApproveDialogOpen(false)} disabled={processingAction}>
            Cancel
          </Button>
          <Button 
            variant="contained" 
            color="success"
            onClick={confirmApprove}
            disabled={processingAction}
            sx={{
              backgroundColor: '#4CAF50',
              '&:hover': { backgroundColor: '#45a049' }
            }}
          >
            {processingAction ? 'Processing...' : 'Approve'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Reject Reason Dialog */}
      <Dialog 
        open={rejectDialogOpen} 
        onClose={() => setRejectDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          alignItems: 'center',
          gap: 1
        }}>
          <CancelIcon />
          Provide Rejection Reason
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <TextField
            autoFocus
            margin="dense"
            label="Reason for Rejection"
            type="text"
            fullWidth
            multiline
            rows={4}
            variant="outlined"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            inputProps={{ maxLength: 255 }}
            helperText={`${rejectReason.length}/255 characters`}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRejectDialogOpen(false)} disabled={processingAction}>
            Cancel
          </Button>
          <Button 
            variant="contained" 
            color="error"
            onClick={confirmReject}
            disabled={processingAction}
            sx={{
              backgroundColor: '#f44336',
              '&:hover': { backgroundColor: '#d32f2f' }
            }}
          >
            {processingAction ? 'Processing...' : 'Reject'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar for notifications */}
      <Snackbar 
        open={snackbar.open} 
        autoHideDuration={6000} 
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert 
          onClose={handleCloseSnackbar} 
          severity={snackbar.severity} 
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default GenericModelConfigApprovalTable;
