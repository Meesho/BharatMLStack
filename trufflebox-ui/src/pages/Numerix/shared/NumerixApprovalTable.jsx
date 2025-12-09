import React, { useState } from 'react';
import {
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Paper,
  Box,
  TextField,
  IconButton,
  Tooltip,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Skeleton,
  Snackbar,
  Alert,
  Divider,
  InputAdornment,
  Modal,
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Stack
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CloseIcon from '@mui/icons-material/Close';
import DoDisturbAlt from '@mui/icons-material/DoDisturbAlt';
import SearchIcon from '@mui/icons-material/Search';
import FilterListIcon from '@mui/icons-material/FilterList';
import HistoryIcon from '@mui/icons-material/History';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import VisibilityIcon from '@mui/icons-material/Visibility';
import EventAvailableIcon from '@mui/icons-material/EventAvailable';
import { Edit } from '@mui/icons-material';
import { useAuth } from '../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../constants/permissions';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../config';

const StatusColumnHeader = ({ selectedStatuses, setSelectedStatuses }) => {
  const [anchorEl, setAnchorEl] = useState(null);
  
  const statusOptions = [
    { value: 'PENDING APPROVAL', label: 'Pending', color: '#FFF3E0', textColor: '#E65100' },
    { value: 'APPROVED', label: 'Approved', color: '#E8F5E9', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
    { value: 'CANCELLED', label: 'Cancelled', color: '#FAFAFA', textColor: '#757575' }
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
        <Stack direction="row" spacing={0.5} sx={{ overflow: 'scroll', gap: '0.25rem' }}>
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

const formatPayload = (payload) => {
  if (!payload) return 'No payload';
  
  const configValue = payload.config_value || payload;

  const infix = configValue.infix_expression.substring(0, 100) + (configValue.infix_expression.length > 100 ? '...' : '');
  const postfix = configValue.postfix_expression?.substring(0, 100) + (configValue.postfix_expression?.length > 100 ? '...' : '');
  return `infix_exp: "${infix}"\npostfix_exp: "${postfix}"`;
};

const formatStatus = (status) => {
  if (!status) return '';
  const statusLower = status.toLowerCase();
  if (statusLower === 'pending_approval' || statusLower === 'pending approval') {
    return 'PENDING';
  }
  return status;
};

const NumerixApprovalTable = ({ data, loading, onRefresh, onApprove, onReject, onCancel }) => {
  const { user, hasPermission } = useAuth();
  
  const service = SERVICES.NUMERIX;
  const screenType = SCREEN_TYPES.NUMERIX.CONFIG_APPROVAL;
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING APPROVAL']); // Initialize with PENDING APPROVAL only
  
  const [openDetailModal, setOpenDetailModal] = useState(false);
  const [openRejectModal, setOpenRejectModal] = useState(false);
  const [openCancelModal, setOpenCancelModal] = useState(false);
  const [openActivityModal, setOpenActivityModal] = useState(false);
  const [openPayloadModal, setOpenPayloadModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [rejectReason, setRejectReason] = useState('');
  const [actionLoading, setActionLoading] = useState(false);
  
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  const columns = [
    { 
      field: 'request_id', 
      headerName: 'Request ID',
      width: '100px',
    },
    { 
      field: 'compute_id', 
      headerName: 'Compute ID',
      width: '100px',
    },
    { 
      field: 'payload', 
      headerName: 'Payload',
      width: 'auto',
      render: (row) => (
        <Box sx={{ 
          display: 'flex', 
          alignItems: 'flex-start',
          gap: 1, 
          width: '100%',
          overflow: 'hidden',
          py: 1
        }}>
          <Box sx={{ 
            flex: 1,
            minWidth: 0,
            overflow: 'hidden',
            whiteSpace: 'pre',
            fontSize: '0.75rem',
            color: '#1a1a1a',
            fontFamily: 'monospace',
            lineHeight: '1.4',
            maxHeight: '2.8em',
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            '& .expression-line': {
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis'
            }
          }}>
            {formatPayload(row.payload).split('\n').map((line, index) => (
              <div key={index} className="expression-line">{line}</div>
            ))}
          </Box>
          <Tooltip title="View Full Payload" disableTransition>
            <IconButton 
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                handleOpenPayloadModal(row);
              }}
              sx={{ 
                color: '#450839', 
                flexShrink: 0,
                mt: -0.5
              }}
            >
              <VisibilityIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      )
    },
    {
      field: 'status',
      headerName: 'Status',
      width: '120px',
      render: (row) => (
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <Chip
            label={formatStatus(row.status)}
            color={
              row.status === 'APPROVED' || row.status === 'Approved'
                ? 'success'
                : row.status === 'REJECTED' || row.status === 'Rejected'
                ? 'error'
                : row.status === 'CANCELLED' || row.status === 'Cancelled'
                ? 'default'
                : 'warning'
            }
            sx={{
              backgroundColor: getStatusColor(row.status).bg,
              color: getStatusColor(row.status).text,
              fontWeight: 'bold',
              maxWidth: '90px',
              height: '24px',
              '& .MuiChip-label': {
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                fontSize: '0.75rem',
                padding: '0 8px'
              }
            }}
          />
        </Box>
      ),
    },
    { 
      field: 'request_type', 
      headerName: 'Request Type',
      width: '100px',
      render: (row) => (
        <Tooltip title={row.request_type} disableTransition>
          <Box sx={{ 
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            maxWidth: '90px',
            padding: '6px 0',
            fontSize: '0.75rem',
            color: '#1a1a1a'
          }}>
            {row.request_type}
          </Box>
        </Tooltip>
      )
    },
    { 
      field: 'reviewer', 
      headerName: 'Reviewer',
      width: '100px',
      render: (row) => (
        <Tooltip title={row.reviewer || 'Not reviewed yet'} disableTransition>
          <Box sx={{ 
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            maxWidth: '90px',
            padding: '6px 0',
            fontSize: '0.875rem',
            color: row.reviewer ? '#1a1a1a' : '#666'
          }}>
            {row.reviewer || 'Not reviewed'}
          </Box>
        </Tooltip>
      )
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: '150px',
      render: (row) => (
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', justifyContent: 'flex-start' }}>
          {/* View Details/Take Action */}
          <Tooltip title="View Details" disableTransition>
            <IconButton onClick={() => handleViewDetails(row)} size="small">
              {row.status === 'PENDING APPROVAL' ? <Edit fontSize="small" sx={{ color: '#450839' }} /> : <InfoIcon fontSize="small" />}
            </IconButton>
          </Tooltip>

          {/* Activity History */}
          <Tooltip title="View Activity History" disableTransition>
            <IconButton 
              onClick={(e) => {
                e.stopPropagation();
                handleOpenActivityModal(row);
              }}
              size="small"
            >
              <EventAvailableIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Cancel Action */}
          {(row.status === 'PENDING APPROVAL' || row.status === 'Pending Approval') && hasPermission(service, screenType, ACTIONS.CANCEL) && (
            <Tooltip title="Cancel Request" disableTransition>
              <IconButton 
                onClick={() => handleCancelRequest(row)}
                size="small"
                sx={{ color: '#f57c00' }}
              >
                <DoDisturbAlt fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </Box>
      ),
    },
  ];

  const filteredData = React.useMemo(() => {
    if (!data) return [];
    
    let filtered = [...data];
    
    // Filter by status
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(row => {
        const rowStatus = row.status.toUpperCase();
        return selectedStatuses.includes(rowStatus);
      });
    }
    
    // Filter by search query
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.request_id).toLowerCase().includes(searchLower) ||
          String(row.compute_id).toLowerCase().includes(searchLower) ||
          (row.created_by && row.created_by.toLowerCase().includes(searchLower)) ||
          (row.reviewer && row.reviewer.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => b.request_id - a.request_id);
  }, [data, searchQuery, selectedStatuses]);

  const getStatusColor = (status) => {
    const statusLower = status?.toLowerCase() || '';
    
    if (statusLower.includes('approved')) {
      return { bg: '#E7F6E7', text: '#2E7D32' };
    } else if (statusLower.includes('rejected')) {
      return { bg: '#FFEBEE', text: '#D32F2F' };
    } else if (statusLower.includes('cancelled')) {
      return { bg: '#EEEEEE', text: '#616161' };
    } else {
      return { bg: '#FFF8E1', text: '#F57C00' };
    }
  };

  const handleViewDetails = (row) => {
    setSelectedRequest(row);
    setOpenDetailModal(true);
  };

  const handleApproveRequest = async () => {
    if (!selectedRequest) return;
    
    setActionLoading(true);
    try {
      if (onApprove) {
        await onApprove(selectedRequest.request_id);
        handleCloseDetailModal();
      } else {
        const response = await axios.post(
          `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-approval/review`,
          {
            request_id: selectedRequest.request_id,
            status: "APPROVED",
            reviewer: user.email,
            reject_reason: ""
          },
          {
            headers: {
              'Authorization': `Bearer ${user.token}`,
              'Content-Type': 'application/json'
            }
          }
        );
        
        if (response.data &&  response.data.error === "") {
          showNotification('Request approved successfully', 'success');
          onRefresh && onRefresh();
          handleCloseDetailModal();
        } else {
          showNotification(response.data?.error || 'Error approving request', 'error');
        }
      }
    } catch (error) {
      showNotification(error.response?.data?.error || 'Error approving request', 'error');
      console.log('Error approving request:', error);
    } finally {
      setActionLoading(false);
    }
  };

  const handleRejectRequest = async () => {
    if (!selectedRequest || !rejectReason) {
      showNotification('Please provide a reason for rejection', 'warning');
      return;
    }
    
    setActionLoading(true);
    try {
      if (onReject) {
        await onReject(selectedRequest.request_id, rejectReason);
        handleCloseRejectModal();
        handleCloseDetailModal();
      } else {
        const response = await axios.post(
          `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-approval/review`,
          {
            request_id: selectedRequest.request_id,
            status: "REJECTED",
            reviewer: user.email,
            reject_reason: rejectReason
          },
          {
            headers: {
              'Authorization': `Bearer ${user.token}`,
              'Content-Type': 'application/json'
            }
          }
        );
        
        if (response.data &&  response.data.error === "") {
          showNotification('Request rejected successfully', 'success');
          onRefresh && onRefresh();
          handleCloseRejectModal();
          handleCloseDetailModal();
        } else {
          showNotification(response.data?.error || 'Error rejecting request', 'error');
        }
      }
    } catch (error) {
      showNotification(error.response?.data?.error || 'Error rejecting request', 'error');
      console.log('Error rejecting request:', error);
    } finally {
      setActionLoading(false);
    }
  };

  const handleCancelRequest = (row) => {
    setSelectedRequest(row);
    setOpenCancelModal(true);
  };

  const handleConfirmCancel = async () => {
    if (!selectedRequest) return;
    
    setActionLoading(true);
    try {
      if (onCancel) {
        await onCancel(selectedRequest.request_id);
      } else {
        console.log('Cancel handler not provided');
      }
    } catch (error) {
      showNotification('Failed to cancel request', 'error');
      console.log('Error cancelling request:', error);
    } finally {
      setActionLoading(false);
      setOpenCancelModal(false);
      if (openDetailModal) {
        handleCloseDetailModal();
      }
    }
  };

  const handleCloseCancelModal = () => {
    setOpenCancelModal(false);
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

  const handleOpenRejectModal = () => {
    setOpenDetailModal(false);
    setRejectReason('');
    setOpenRejectModal(true);
  };

  const handleCloseRejectModal = () => {
    setOpenRejectModal(false);
    setRejectReason('');
    if (selectedRequest) {
      setOpenDetailModal(true);
    }
  };

  const handleCloseDetailModal = () => {
    setOpenDetailModal(false);
    setSelectedRequest(null);
  };

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  const handleOpenActivityModal = (row) => {
    setSelectedRequest(row);
    setOpenActivityModal(true);
  };

  const handleCloseActivityModal = () => {
    setOpenActivityModal(false);
  };

  const handleOpenPayloadModal = (row) => {
    setSelectedRequest(row);
    setOpenPayloadModal(true);
  };

  const handleClosePayloadModal = () => {
    setOpenPayloadModal(false);
  };

  const tableCellStyles = {
    borderRight: '1px solid rgb(204, 195, 195)',
    '&:last-child': {
      borderRight: 'none',
    },
  };

  return (
    <Paper sx={{ width: '100%', overflow: 'hidden', boxShadow: 3 }}>
      {/* Search bar */}
      <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <TextField
          placeholder="Search requests..."
          variant="outlined"
          size="small"
          value={searchQuery}
          onChange={handleSearchChange}
          fullWidth
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
      </Box>
      
      {/* Table */}
      <TableContainer sx={{ maxHeight: 'calc(100vh - 250px)' }}>
        <Table stickyHeader sx={{ tableLayout: 'fixed' }}>
          <TableHead>
            <TableRow>
              {columns.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={{
                    ...tableCellStyles,
                    backgroundColor: '#f5f5f5',
                    fontWeight: 'bold',
                    color: '#031022',
                    width: column.width || 'auto',
                    minWidth: column.width || 'auto',
                    maxWidth: column.width || 'none',
                    ...(column.field === 'payload' && {
                      width: 'auto',
                      minWidth: '300px'
                    })
                  }}
                >
                  {column.field === 'status' ? (
                    <StatusColumnHeader 
                      selectedStatuses={selectedStatuses}
                      setSelectedStatuses={setSelectedStatuses}
                    />
                  ) : (
                    column.headerName
                  )}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              Array.from(new Array(5)).map((_, index) => (
                <TableRow key={index}>
                  {columns.map((column, cellIndex) => (
                    <TableCell 
                      key={cellIndex} 
                      sx={{
                        ...tableCellStyles,
                        width: column.width || 'auto',
                        minWidth: column.width || 'auto',
                        maxWidth: column.width || 'none',
                        ...(column.field === 'payload' && {
                          width: 'auto',
                          minWidth: '300px'
                        })
                      }}
                    >
                      <Skeleton animation="wave" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : filteredData.length > 0 ? (
              filteredData.map((row) => (
                <TableRow key={row.request_id}>
                  {columns.map((column) => (
                    <TableCell 
                      key={column.field} 
                      sx={{
                        ...tableCellStyles,
                        width: column.width || 'auto',
                        minWidth: column.width || 'auto',
                        maxWidth: column.width || 'none',
                        ...(column.field === 'payload' && {
                          width: 'auto',
                          minWidth: '300px'
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
                <TableCell colSpan={columns.length} align="center">
                  No records found matching the current filters.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      
      {/* Detail Modal */}
      <Dialog 
        open={openDetailModal} 
        onClose={handleCloseDetailModal}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h5" color="#450839">Numerix Configuration Request Details</Typography>
          <IconButton onClick={handleCloseDetailModal}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider sx={{ bgcolor: '#450839' }} />
        {selectedRequest && (
          <>
            <DialogContent>
              <Typography><strong>Request ID:</strong> {selectedRequest.request_id}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Compute ID:</strong> {selectedRequest.compute_id}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Status:</strong> {selectedRequest.status}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Request Type:</strong> {selectedRequest.request_type}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Created By:</strong> {selectedRequest.created_by}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Created At:</strong> {selectedRequest.created_at}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Updated By:</strong> {selectedRequest.updated_by}</Typography>
              <Divider sx={{ my: 1 }}/>
              <Typography><strong>Updated At:</strong> {selectedRequest.updated_at}</Typography>
              {selectedRequest.reviewer && (
                <>
                  <Divider sx={{ my: 1 }}/>
                  <Typography><strong>Reviewer:</strong> {selectedRequest.reviewer}</Typography>
                </>
              )}

              <Divider sx={{ my: 2 }} />
              
              <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                Configuration Payload:
              </Typography>
              <Box sx={{ bgcolor: '#f5f5f5', p: 2, borderRadius: 1, overflow: 'auto', maxHeight: '200px' }}>
                <pre style={{ margin: 0 }}>
                  {JSON.stringify(selectedRequest.payload, null, 2)}
                </pre>
              </Box>
              
              {selectedRequest.reject_reason && (
                <>
                  <Divider sx={{ my: 2 }} />
                  <Typography><strong>Rejection Reason:</strong> {selectedRequest.reject_reason}</Typography>
                </>
              )}
            </DialogContent>
            <Divider sx={{ bgcolor: '#450839' }} />
            <DialogActions sx={{ p: 2, justifyContent: 'flex-end' }}>
              {(selectedRequest.status === 'PENDING APPROVAL' || selectedRequest.status === 'Pending Approval') && (
                <>
                    <Box sx={{ display: 'flex', gap: 1}}>
                    {hasPermission(service, screenType, ACTIONS.APPROVE) && (
                      <Button 
                        variant="contained" 
                        color="success"
                        startIcon={<CheckCircleIcon />}
                        sx={{
                          '&:hover': {
                            backgroundColor: '#70c476',
                          },
                        }}
                        onClick={handleApproveRequest}
                        disabled={actionLoading}
                      >
                        Approve
                      </Button>
                    )}
                    {hasPermission(service, screenType, ACTIONS.REJECT) && (
                      <Button 
                        variant="contained" 
                        color="error"
                        sx={{
                          '&:hover': {
                            backgroundColor: '#f07b89',
                          },
                        }}
                        startIcon={<CancelIcon />}
                        onClick={handleOpenRejectModal}
                        disabled={actionLoading}

                      >
                        Reject
                      </Button>
                    )}
                    {hasPermission(service, screenType, ACTIONS.CANCEL) && (
                      <Button 
                        variant="contained" 
                        color="warning"
                        sx={{
                          '&:hover': {
                            backgroundColor: '#ed9f60',
                          },
                        }} 
                        startIcon={<DoDisturbAlt />}
                        onClick={() => handleCancelRequest(selectedRequest)}
                        disabled={actionLoading}
                      >
                        Cancel Request
                      </Button>
                    )}
                    </Box>
                </>
              )}
            </DialogActions>
          </>
        )}
      </Dialog>
      
      {/* Reject Modal */}
      <Dialog
        open={openRejectModal}
        onClose={handleCloseRejectModal}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h6">Reject Request</Typography>
          <IconButton onClick={handleCloseRejectModal}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 3 }}>
          <Typography gutterBottom>
            Please provide a reason for rejecting this request:
          </Typography>
          <TextField
            margin="dense"
            label="Reject Reason"
            fullWidth
            multiline
            rows={4}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            inputProps={{ maxLength: 255 }}
            helperText={`${rejectReason.length}/255 characters`}
            required
            error={rejectReason.trim() === ''}
          />
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button 
            variant="outlined"
            onClick={handleCloseRejectModal}
            sx={{
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                backgroundColor: 'rgba(60, 9, 50, 0.04)',
                borderColor: '#380730'
              },
            }}
          >
            Cancel
          </Button>
          <Button 
            onClick={handleRejectRequest} 
            variant="contained" 
            disabled={actionLoading || !rejectReason.trim()}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#5A0A4B',
              },
            }}
          >
            Submit
          </Button>
        </DialogActions>
      </Dialog>

      {/* Cancel Confirmation Modal */}
      <Dialog 
        open={openCancelModal} 
        onClose={handleCloseCancelModal}
        maxWidth="sm"
      >
        <DialogTitle sx={{display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h6">Cancel Request</Typography>
          <IconButton onClick={handleCloseCancelModal}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 3 }}>
          <Typography>
            Are you sure you want to cancel this request?
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button 
            variant="outlined"
            onClick={handleCloseCancelModal}
            sx={{
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                backgroundColor: 'rgba(60, 9, 50, 0.04)',
                borderColor: '#380730'
              },
            }}
          >
            NO
          </Button>
          <Button 
            onClick={handleConfirmCancel} 
            variant="contained" 
            color="warning"
            disabled={actionLoading}
            sx={{
              '&:hover': {
                backgroundColor: '#ed9f60',
              },
            }}
          >
            YES
          </Button>
        </DialogActions>
      </Dialog>

      {/* Activity Modal */}
      <Modal 
        open={openActivityModal} 
        onClose={handleCloseActivityModal} 
        aria-labelledby="activity-history-modal"
      >
        <Box sx={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: 600, maxWidth: '90vw', bgcolor: 'background.paper', boxShadow: 24, borderRadius: 2, overflow: 'hidden' }}>
          <Box sx={{ bgcolor: '#450839', color: 'white', p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <HistoryIcon />
              <Typography variant="h6" component="h2">
                Activity History
              </Typography>
            </Box>
            <IconButton onClick={handleCloseActivityModal} sx={{ color: 'white' }} aria-label="close">
              <CloseIcon />
            </IconButton>
          </Box>
          {selectedRequest && (
            <Box sx={{ p: 3 }}>
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
                    <Typography variant="body2">{selectedRequest.created_by || 'N/A'}</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRequest.created_at ? new Date(selectedRequest.created_at).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' }) : 'N/A'}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRequest.created_at ? new Date(selectedRequest.created_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: true }) : 'N/A'}
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
                    <Typography variant="body2">{selectedRequest.updated_by || 'N/A'}</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRequest.updated_at ? new Date(selectedRequest.updated_at).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' }) : 'N/A'}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRequest.updated_at ? new Date(selectedRequest.updated_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: true }) : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
            </Box>
          )}
        </Box>
      </Modal>

      {/* Payload Modal */}
      <Dialog 
        open={openPayloadModal} 
        onClose={handleClosePayloadModal}
        maxWidth="md"
        fullWidth
        sx={{
          '& .MuiDialog-paper': {
            borderRadius: '10px',
          },
        }}
      >
        <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', bgcolor: '#450839', color: 'white', borderRadius: '10px 10px 0 0' }}>
          <Typography variant="h6">Configuration Payload</Typography>
          <IconButton onClick={handleClosePayloadModal} sx={{ color: 'white' }}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider sx={{ bgcolor: '#450839' }} />
        {selectedRequest && (
          <DialogContent>
            {selectedRequest.payload?.config_value && (
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle1"><strong>Infix Expression:</strong></Typography>
                <Box sx={{ bgcolor: '#f5f5f5', p: 2, borderRadius: 1, mt: 1 }}>
                  <Typography 
                    variant="body1"
                    sx={{
                      fontFamily: 'monospace',
                      wordBreak: 'break-all',
                      whiteSpace: 'pre-wrap'
                    }}
                  >
                    {selectedRequest.payload.config_value?.infix_expression || 'N/A'}
                  </Typography>
                </Box>
                
                <Typography variant="subtitle1" sx={{ mt: 2 }}><strong>Postfix Expression:</strong></Typography>
                <Box sx={{ bgcolor: '#f5f5f5', p: 2, borderRadius: 1, mt: 1 }}>
                  <Typography 
                    variant="body1"
                    sx={{
                      fontFamily: 'monospace',
                      wordBreak: 'break-all',
                      whiteSpace: 'pre-wrap'
                    }}
                  >
                    {selectedRequest.payload.config_value?.postfix_expression || 'N/A'}
                  </Typography>
                </Box>
              </Box>
            )}

            <Box sx={{ bgcolor: '#f5f5f5', p: 2, borderRadius: 1, overflow: 'auto', mt: 1, maxHeight: '300px' }}>
              <pre style={{ margin: 0 }}>
                {JSON.stringify(selectedRequest.payload, null, 2)}
              </pre>
            </Box>
          </DialogContent>
        )}
      </Dialog>

      {/* Notification snackbar */}
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleCloseNotification}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
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

export default NumerixApprovalTable;
