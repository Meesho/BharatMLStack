import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Box,
  Chip,
  IconButton,
  Tooltip,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Button,
  Skeleton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Grid,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Switch,
  FormControlLabel,
  Popover,
  Stack,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
  Snackbar,
  Alert as MuiAlert,
} from '@mui/material';
import FactCheckIcon from '@mui/icons-material/FactCheck';
import InfoIcon from '@mui/icons-material/Info';
import SearchIcon from '@mui/icons-material/Search';
import CachedIcon from '@mui/icons-material/Cached';
import RefreshIcon from '@mui/icons-material/Refresh';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import CloseIcon from '@mui/icons-material/Close';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import FilterListIcon from '@mui/icons-material/FilterList';
import CheckIcon from '@mui/icons-material/Check';
import CircularProgress from '@mui/material/CircularProgress';
import { useAuth } from '../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';
import * as URL_CONSTANTS from '../../../../config'

const GenericModelApprovalTable = ({ onValidate, onReview }) => {
  const [modelRequests, setModelRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING APPROVAL']); // Initialize with PENDING APPROVAL only
  const [infoDialogOpen, setInfoDialogOpen] = useState(false);
  const [selectedGroupModels, setSelectedGroupModels] = useState([]);
  const [approveDialogOpen, setApproveDialogOpen] = useState(false);
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [processingAction, setProcessingAction] = useState(false);
  const [currentGroupId, setCurrentGroupId] = useState(null);

  const [validatedGroups, setValidatedGroups] = useState(new Set());
  const [validatingGroups, setValidatingGroups] = useState(new Set());
  const [groupValidationStatus, setGroupValidationStatus] = useState(new Map()); // Map<groupId, is_valid>
  
  // Toast state
  const [toastOpen, setToastOpen] = useState(false);
  const [toastMessage, setToastMessage] = useState('');
  const [toastSeverity, setToastSeverity] = useState('info'); // 'success', 'error', 'warning', 'info'

  // Deployables state for host lookup
  const [deployables, setDeployables] = useState([]);
  const [deployablesLoading, setDeployablesLoading] = useState(false);
  
  const { user, hasPermission } = useAuth();
  
  const service = SERVICES.PREDATOR;
  const screenType = SCREEN_TYPES.PREDATOR.MODEL_APPROVAL;

  const getGroupColor = (groupIndex) => {
    const colors = [
      '#ffffff',
      '#e3f2fd', // Light blue
    ];
    return colors[groupIndex % colors.length];
  };

  // Helper function to get is_valid status for a group
  const getGroupIsValid = (groupId) => {
    const groupModels = modelRequests.filter(model => model.groupId === groupId);
    if (groupModels.length > 0) {
      return groupModels[0].isValid; // All items in a group should have the same is_valid status
    }
    return false;
  };

  const fetchModelRequests = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-approval/requests`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      
      if (!response.ok) {
        console.log('Failed to fetch model requests');
      }
      
      const result = await response.json();
      
      if (result.error) {
        console.log(result.error);
      }
      
      let formattedData = [];
      
      result.data
        .sort((a, b) => b.group_id - a.group_id)
        .forEach((group, groupIndex) => {
          if (group.groups && Array.isArray(group.groups)) {
            const groupRequests = group.groups.map((item, itemIndex) => ({
              groupId: group.group_id,
              groupIndex: groupIndex,
              groupColor: getGroupColor(groupIndex),
              isFirstInGroup: itemIndex === 0,
              groupSize: group.groups.length,
              requestId: item.request_id,
              modelName: item.payload.model_name,
              payload: item.payload,
              requestType: item.request_type,
              requestStage: item.request_stage,
              status: item.status,
              reviewer: item.reviewer || 'N/A',
              createdBy: item.craeted_by || item.created_by,
              createdAt: item.created_at,
              updatedBy: item.updated_by,
              updatedAt: item.updated_at,
              rejectReason: item.reject_reason || item.rejetc_reason || 'N/A',
              isValid: item.is_valid || false
            }));
            
            formattedData = [...formattedData, ...groupRequests];
          }
        });
      
      setModelRequests(formattedData);
    } catch (error) {
      console.log('Error fetching model requests:', error);
    } finally {
      setLoading(false);
    }
  }, [user.token]);

  useEffect(() => {
    fetchModelRequests();
  }, [fetchModelRequests]);

  // Fetch deployables for host lookup
  const fetchDeployables = useCallback(async () => {
    try {
      setDeployablesLoading(true);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=PREDATOR`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      
      if (!response.ok) {
        console.log('Failed to fetch deployables');
        return;
      }
      
      const result = await response.json();
      
      if (result.error) {
        console.log(result.error);
        return;
      }
      
      if (result.data && Array.isArray(result.data)) {
        setDeployables(result.data);
      }
    } catch (error) {
      console.log('Error fetching deployables:', error);
    } finally {
      setDeployablesLoading(false);
    }
  }, [user.token]);

  const handleInfoClick = async (groupId) => {
    const groupModels = modelRequests.filter(model => model.groupId === groupId);
    setSelectedGroupModels(groupModels);
    setCurrentGroupId(groupId);
    setInfoDialogOpen(true);
    
    // Fetch deployables when opening the dialog
    if (deployables.length === 0) {
      await fetchDeployables();
    }
  };

  const handleCloseInfoDialog = () => {
    setInfoDialogOpen(false);
    setSelectedGroupModels([]);
    setCurrentGroupId(null);
  };

  const handleValidateGroup = async (groupId) => {
    if (!hasPermission(service, screenType, ACTIONS.VALIDATE)) {
      console.warn('User does not have permission to validate');
      setToastMessage('You do not have permission to validate');
      setToastSeverity('error');
      setToastOpen(true);
      return false;
    }

    // Check if already validating
    if (validatingGroups.has(groupId)) {
      console.warn('Validation already in progress for this group');
      setToastMessage('Validation already in progress for this group');
      setToastSeverity('warning');
      setToastOpen(true);
      return false;
    }

    try {
      // Add to validating set
      setValidatingGroups(prev => new Set(prev).add(groupId));

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-approval/requests/${groupId}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${user.token}`,
          'Content-Type': 'application/json',
        },
      });

      const result = await response.json();

      if (result.error) {
        console.error('Validation error:', result.error);
        setToastMessage(result.error || 'Validation failed');
        setToastSeverity('error');
        setToastOpen(true);
        // Remove from validating set
        setValidatingGroups(prev => {
          const newSet = new Set(prev);
          newSet.delete(groupId);
          return newSet;
        });
        return false;
      }

      // Check if validation was successful and get is_valid status
      if (response.ok) {
        // Add to validated set
        setValidatedGroups(prev => new Set(prev).add(groupId));
        
        // Store the is_valid status from the response
        const isValid = result.is_valid || false;
        setGroupValidationStatus(prev => new Map(prev).set(groupId, isValid));
        
        // Show success toast with the message from API
        setToastMessage(result.data || 'Request validation completed successfully');
        setToastSeverity('success');
        setToastOpen(true);
        
        // Refresh data
        await fetchModelRequests();
        
        return true;
      } else {
        console.warn('Validation failed:', result.data);
        setToastMessage(result.data || 'Validation failed');
        setToastSeverity('error');
        setToastOpen(true);
        return false;
      }

    } catch (error) {
      console.error('Error validating group:', error);
      setToastMessage('Error validating group: ' + error.message);
      setToastSeverity('error');
      setToastOpen(true);
      return false;
    } finally {
      // Remove from validating set
      setValidatingGroups(prev => {
        const newSet = new Set(prev);
        newSet.delete(groupId);
        return newSet;
      });
    }
  };

  const handleValidateClick = async (groupId) => {
    setCurrentGroupId(groupId);
    try {
      setProcessingAction(true);
      const result = await onValidate(groupId);
      if (result && result.success) {
        // Refresh the data to get updated is_valid status
        await fetchModelRequests();
      }
    } catch (error) {
      console.log('Error validating model:', error);
    } finally {
      setProcessingAction(false);
    }
  };
  
  const handleApprove = async () => {
    try {
      setProcessingAction(true);
      const success = await onReview(currentGroupId, 'Approved');
      if (success) {
        await fetchModelRequests(); // Refresh the data
        setApproveDialogOpen(false);
        setInfoDialogOpen(false);
      }
    } catch (error) {
      console.log('Error approving model:', error);
    } finally {
      setProcessingAction(false);
    }
  };
  
  const handleReject = async () => {
    try {
      setProcessingAction(true);
      const success = await onReview(currentGroupId, 'Rejected', rejectReason);
      if (success) {
        await fetchModelRequests(); // Refresh the data
        setRejectDialogOpen(false);
        setRejectReason('');
        setInfoDialogOpen(false);
      }
    } catch (error) {
      console.log('Error rejecting model:', error);
    } finally {
      setProcessingAction(false);
    }
  };



  const StatusColumnHeader = () => {
    const [anchorEl, setAnchorEl] = useState(null);
    
    const statusOptions = [
      { value: 'PENDING APPROVAL', label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
      { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
      { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' },
      { value: 'FAILED', label: 'Failed', color: '#EEEEEE', textColor: '#616161' },
      { value: 'IN PROGRESS', label: 'In Progress', color: '#E3F2FD', textColor: '#1976D2' }
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

  const columns = [
    { 
      field: 'groupId', 
      headerName: 'Group ID',
      width: '6%',
      stickyColumn: false
    },
    { 
      field: 'requestId', 
      headerName: 'Request ID',
      width: '6%',
      stickyColumn: false
    },
    { 
      field: 'modelName', 
      headerName: 'Model Name',
      width: '36%',
      stickyColumn: false
    },
    { 
      field: 'requestType', 
      headerName: 'Request Type',
      width: '6%',
      stickyColumn: false
    },
    { 
      field: 'requestStage', 
      headerName: 'Request Stage',
      width: '10%',
      stickyColumn: false
    },
    { 
      field: 'status', 
      headerName: <StatusColumnHeader />,
      width: '5%',
      stickyColumn: false,
      render: (row) => {
        const status = row.status.toUpperCase();
        let chipProps = {
          label: status,
          sx: { fontWeight: 'medium', height: '30px' }
        };

        switch (status) {
          case 'PENDING APPROVAL':
            chipProps.sx.backgroundColor = '#FFF8E1';
            chipProps.sx.color = '#F57C00';
            break;
          case 'APPROVED':
            chipProps.sx.backgroundColor = '#E7F6E7';
            chipProps.sx.color = '#2E7D32';
            break;
          case 'REJECTED':
            chipProps.sx.backgroundColor = '#FFEBEE';
            chipProps.sx.color = '#D32F2F';
            break;
          case 'FAILED':
            chipProps.sx.backgroundColor = '#EEEEEE';
            chipProps.sx.color = '#616161';
            break;
          case 'IN PROGRESS':
            chipProps.sx.backgroundColor = '#E3F2FD';
            chipProps.sx.color = '#1976D2';
            break;
          default:
            chipProps.sx.backgroundColor = '#EEEEEE';
            chipProps.sx.color = '#616161';
        }

        return <Chip {...chipProps} />;
      }
    },
    { 
      field: 'actions', 
      headerName: 'Actions',
      width: '10%',
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
          {/* View Group Details */}
          <Tooltip title="View Group Details" disableTransition>
            <IconButton 
              onClick={(e) => {
                e.stopPropagation();
                handleInfoClick(row.groupId);
              }}
              size="small"
            >
              <InfoIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Validate Group - Only show if pending approval */}
          {isGroupPendingApproval(row.groupId) && hasPermission(service, screenType, ACTIONS.VALIDATE) && (
            <Tooltip title={row.isValid ? "Group Validated" : "Validate Group"} disableTransition>
              <IconButton 
                onClick={(e) => {
                  e.stopPropagation();
                  handleValidateGroup(row.groupId);
                }}
                size="small"
                disabled={row.isValid}
              >
                <FactCheckIcon 
                  fontSize="small" 
                  sx={{ 
                    color: row.isValid ? '#4caf50' : 'inherit' // Green if valid, default color otherwise
                  }} 
                />
              </IconButton>
            </Tooltip>
          )}
        </Box>
      )
    }
  ];

  const filteredRequests = useMemo(() => {
    return modelRequests.filter(request => {
      const matchesSearch = searchQuery === '' || 
        request.modelName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        request.requestId.toString().includes(searchQuery) ||
        request.groupId.toString().includes(searchQuery);

      const matchesStatus = selectedStatuses.length === 0 || 
        selectedStatuses.includes(request.status.toUpperCase());

      return matchesSearch && matchesStatus;
    });
  }, [modelRequests, searchQuery, selectedStatuses]);

  // Helper function to get host from deployable ID
  const getHostFromDeployableId = (deployableId) => {
    const deployable = deployables.find(d => d.id === deployableId);
    return deployable ? deployable.host : 'N/A';
  };

  // Features Display Component for grouped features
  const FeaturesDisplay = ({ features }) => {
    const [expanded, setExpanded] = useState(false);

    if (!features || features.length === 0) {
      return <Typography variant="body2" color="textSecondary">No features</Typography>;
    }

    // Group features by feature type (left part before |)
    const groupedFeatures = features.reduce((acc, feature) => {
      const pipeIndex = feature.indexOf('|');
      let featureType = 'Other';
      let featureName = feature;

      if (pipeIndex !== -1) {
        featureType = feature.substring(0, pipeIndex).trim() || 'Other';
        featureName = feature.substring(pipeIndex + 1).trim();
      }

      if (!acc[featureType]) {
        acc[featureType] = [];
      }
      acc[featureType].push(featureName);
      return acc;
    }, {});

    const featureTypes = Object.keys(groupedFeatures);
    const totalFeatures = features.length;

    return (
      <Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
          <Typography variant="body2" color="textSecondary" sx={{ fontWeight: 600 }}>
            Features ({totalFeatures})
          </Typography>
          <IconButton size="small" onClick={() => setExpanded(!expanded)}>
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          </IconButton>
        </Box>
        
        {expanded ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {featureTypes.map((featureType, typeIndex) => (
              <Box key={typeIndex} sx={{ 
                border: '1px solid #e0e0e0', 
                borderRadius: 1, 
                p: 1.5,
                backgroundColor: '#fafafa'
              }}>
                <Typography variant="caption" sx={{ 
                  fontWeight: 600, 
                  color: '#450839',
                  display: 'block',
                  mb: 1
                }}>
                  {featureType} ({groupedFeatures[featureType].length})
                </Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {groupedFeatures[featureType].map((feature, index) => (
                    <Chip
                      key={index}
                      label={feature}
                      size="small"
                      sx={{
                        fontSize: '0.7rem',
                        height: 'auto',
                        backgroundColor: 'white',
                        '& .MuiChip-label': {
                          paddingTop: '4px',
                          paddingBottom: '4px',
                          paddingLeft: '8px',
                          paddingRight: '8px'
                        }
                      }}
                    />
                  ))}
                </Box>
              </Box>
            ))}
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {featureTypes.map((featureType, index) => (
              <Chip
                key={index}
                label={`${featureType} (${groupedFeatures[featureType].length})`}
                size="small"
                sx={{
                  fontSize: '0.75rem',
                  backgroundColor: '#f0f0f0',
                  fontWeight: 500
                }}
              />
            ))}
          </Box>
        )}
      </Box>
    );
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

  const lastColumnStyle = {
    ...tableCellStyles,
    borderRight: 'none',
    wordBreak: 'break-word',
    whiteSpace: 'normal',
  };

  const getStatusChipProps = (status) => {
    switch(status.toUpperCase()) {
      case 'APPROVED':
        return {
          color: 'success',
          backgroundColor: '#E8F5E9',
          textColor: '#2E7D32'
        };
      case 'REJECTED':
        return {
          color: 'error',
          backgroundColor: '#FFEBEE',
          textColor: '#D32F2F'
        };
      case 'PENDING APPROVAL':
        return {
          color: 'warning',
          backgroundColor: '#FFF3E0',
          textColor: '#E65100'
        };
      case 'CANCELLED':
        return {
          color: 'default',
          backgroundColor: '#FAFAFA',
          textColor: '#757575'
        };
      case 'INPROGRESS':
      case 'IN PROGRESS':
        return {
          color: 'info',
          backgroundColor: '#E3F2FD',
          textColor: '#1976D2'
        };
      case 'FAILED':
        return {
          color: 'error',
          backgroundColor: '#FFCDD2',
          textColor: '#C62828'
        };
      default:
        return {
          color: 'default',
          backgroundColor: '#F5F5F5',
          textColor: '#616161'
        };
    }
  };

  const isGroupPendingApproval = (groupId) => {
    const groupRequests = modelRequests.filter(req => req.groupId === groupId);
    return groupRequests.some(req => 
      req.status.toUpperCase() === 'PENDING APPROVAL' || 
      req.status.toUpperCase() === 'FAILED'
    );
  };

  const isGroupFailed = (groupId) => {
    const groupRequests = modelRequests.filter(req => req.groupId === groupId);
    return groupRequests.some(req => req.status.toUpperCase() === 'FAILED');
  };

  return (
    <Box sx={{ p: 2 }}>
      
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
        <TextField
          placeholder="Search requests..."
          variant="outlined"
          size="small"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          sx={{ flex: 1 }}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
        
        <Button
          variant="contained"
          onClick={fetchModelRequests}
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
              ) : filteredRequests.length > 0 ? (
                filteredRequests.map((row) => (
                  <TableRow
                    key={`${row.groupId}-${row.requestId}`}
                    sx={{
                      cursor: 'pointer',
                      backgroundColor: row.groupColor,
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

      {/* Group Details Dialog */}
      <Dialog 
        open={infoDialogOpen} 
        onClose={handleCloseInfoDialog}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle sx={{ m: 0, p: 2, pr: 6 }}>
          <Typography variant="h6">Group {currentGroupId} - Model Requests Details</Typography>
          <IconButton
            aria-label="close"
            onClick={handleCloseInfoDialog}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              color: (theme) => theme.palette.grey[500],
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          {selectedGroupModels.map((model, index) => (
            <Accordion key={model.requestId} defaultExpanded={index === 0}>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                  Request ID: {model.requestId} - {model.modelName}
                </Typography>
              </AccordionSummary>
              <AccordionDetails>
                <Box sx={{ py: 1 }}>
                  <Box sx={{ mb: 2 }}>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Request ID:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.requestId}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Model Name:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.modelName}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Request Type:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.requestType}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Request Stage:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.requestStage}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Status:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.status}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Reviewer:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.reviewer}
                      </Typography>
                    </Box>
                    {model.status.toUpperCase() === 'REJECTED' && (
                      <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                        <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                          Reject Reason:
                        </Typography>
                        <Typography variant="body1" sx={{ width: '60%' }}>
                          {model.rejectReason}
                        </Typography>
                      </Box>
                    )}

                    {/* Activity Details Section */}
                    <Typography variant="subtitle1" sx={{ mt: 3, mb: 1, fontWeight: 'bold', color: '#450839' }}>
                      Activity Details
                    </Typography>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Created By:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.createdBy || 'N/A'}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Created At:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.createdAt ? new Date(model.createdAt).toLocaleString() : 'N/A'}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Updated By:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.updatedBy || 'N/A'}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Updated At:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.updatedAt ? new Date(model.updatedAt).toLocaleString() : 'N/A'}
                      </Typography>
                    </Box>
                  </Box>
                  
                  <Box>
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Model Source Path:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.model_source_path}
                      </Typography>
                    </Box>                
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Instance Count:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.meta_data.instance_count || 'N/A'}
                      </Typography>
                    </Box>
                    
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Instance Type:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.meta_data.instance_type || 'N/A'}
                      </Typography>
                    </Box>
                    
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Batch Size:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.meta_data.batch_size}
                      </Typography>
                    </Box>
                    
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Backend:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.meta_data.backend || 'N/A'}
                      </Typography>
                    </Box>
                    
                    {model.payload.meta_data.platform && (
                      <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                        <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                          Platform:
                        </Typography>
                        <Typography variant="body1" sx={{ width: '60%' }}>
                          {model.payload.meta_data.platform}
                        </Typography>
                      </Box>
                    )}
                    
                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Dynamic Batching:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.meta_data.dynamic_batching_enabled ? 'Enabled' : 'Disabled'}
                      </Typography>
                    </Box>

                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Service Deployable ID:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {model.payload.config_mapping.service_deployable_id}
                      </Typography>
                    </Box>

                    <Box sx={{ display: 'flex', py: 1, borderBottom: '1px solid #f0f0f0' }}>
                      <Typography variant="body1" sx={{ fontWeight: 'bold', width: '40%' }}>
                        Host:
                      </Typography>
                      <Typography variant="body1" sx={{ width: '60%' }}>
                        {deployablesLoading ? (
                          <CircularProgress size={16} />
                        ) : (
                          getHostFromDeployableId(model.payload.config_mapping.service_deployable_id)
                        )}
                      </Typography>
                    </Box>

                    {/* Features Section - Extract from all inputs */}
                    {(() => {
                      // Collect all features from all inputs
                      const allFeatures = [];
                      if (model.payload.meta_data.inputs && Array.isArray(model.payload.meta_data.inputs)) {
                        model.payload.meta_data.inputs.forEach(input => {
                          if (input.features && Array.isArray(input.features)) {
                            allFeatures.push(...input.features);
                          }
                        });
                      }
                      
                      return allFeatures.length > 0 ? (
                        <>
                          <Typography variant="subtitle1" sx={{ mt: 3, mb: 1, fontWeight: 'bold', color: '#450839' }}>
                            Features
                          </Typography>
                          <Box sx={{ py: 1, borderBottom: '1px solid #f0f0f0' }}>
                            <FeaturesDisplay features={allFeatures} />
                          </Box>
                        </>
                      ) : null;
                    })()}
                    
                    <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2, mb: 1 }}>Inputs</Typography>
                    {model.payload.meta_data.inputs.map((input, inputIndex) => (
                      <Box key={inputIndex} sx={{ ml: 2, mb: 2, p: 1, border: '1px solid #e0e0e0', borderRadius: 1 }}>
                        <Box sx={{ display: 'flex', py: 0.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                            Name:
                          </Typography>
                          <Typography variant="body2" sx={{ width: '60%' }}>
                            {input.name}
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', py: 0.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                            Dims:
                          </Typography>
                          <Typography variant="body2" sx={{ width: '60%' }}>
                            {JSON.stringify(input.dims)}
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', py: 0.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                            Data Type:
                          </Typography>
                          <Typography variant="body2" sx={{ width: '60%' }}>
                            {input.data_type}
                          </Typography>
                        </Box>
                      </Box>
                    ))}
                    
                    <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2, mb: 1 }}>Outputs</Typography>
                    {model.payload.meta_data.outputs.map((output, outputIndex) => (
                      <Box key={outputIndex} sx={{ ml: 2, mb: 2, p: 1, border: '1px solid #e0e0e0', borderRadius: 1 }}>
                        <Box sx={{ display: 'flex', py: 0.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                            Name:
                          </Typography>
                          <Typography variant="body2" sx={{ width: '60%' }}>
                            {output.name}
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', py: 0.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                            Dims:
                          </Typography>
                          <Typography variant="body2" sx={{ width: '60%' }}>
                            {JSON.stringify(output.dims)}
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', py: 0.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                            Data Type:
                          </Typography>
                          <Typography variant="body2" sx={{ width: '60%' }}>
                            {output.data_type}
                          </Typography>
                        </Box>
                      </Box>
                    ))}

                    {model.payload.meta_data.ensemble_scheduling && model.payload.meta_data.ensemble_scheduling.step && (
                      <>
                        <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2, mb: 1 }}>Ensemble Scheduling</Typography>
                        {model.payload.meta_data.ensemble_scheduling.step.map((step, stepIndex) => (
                          <Box key={stepIndex} sx={{ ml: 2, mb: 2, p: 1, border: '1px solid #e0e0e0', borderRadius: 1 }}>
                            <Box sx={{ display: 'flex', py: 0.5 }}>
                              <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                                Step {stepIndex + 1}:
                              </Typography>
                            </Box>
                            <Box sx={{ display: 'flex', py: 0.5 }}>
                              <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                                Model Name:
                              </Typography>
                              <Typography variant="body2" sx={{ width: '60%' }}>
                                {step.model_name}
                              </Typography>
                            </Box>
                            <Box sx={{ display: 'flex', py: 0.5 }}>
                              <Typography variant="body2" sx={{ fontWeight: 'bold', width: '40%' }}>
                                Model Version:
                              </Typography>
                              <Typography variant="body2" sx={{ width: '60%' }}>
                                {step.model_version}
                              </Typography>
                            </Box>
                            <Typography variant="body2" sx={{ fontWeight: 'bold', py: 0.5 }}>
                              Input Mapping:
                            </Typography>
                            {step.input_map && (
                              <Box sx={{ ml: 2, py: 0.5 }}>
                                {Object.entries(step.input_map).map(([key, value]) => (
                                  <Typography key={key} variant="body2">
                                    {key} → {value}
                                  </Typography>
                                ))}
                              </Box>
                            )}
                            <Typography variant="body2" sx={{ fontWeight: 'bold', py: 0.5 }}>
                              Output Mapping:
                            </Typography>
                            {step.output_map && (
                              <Box sx={{ ml: 2, py: 0.5 }}>
                                {Object.entries(step.output_map).map(([key, value]) => (
                                  <Typography key={key} variant="body2">
                                    {key} → {value}
                                  </Typography>
                                ))}
                              </Box>
                            )}
                          </Box>
                        ))}
                      </>
                    )}
                  </Box>
                </Box>
              </AccordionDetails>
            </Accordion>
          ))}
        </DialogContent>
        <DialogActions>
          {selectedGroupModels.length > 0 && isGroupPendingApproval(currentGroupId) && (
            <>
              {/* Validate Button - Show if is_valid is false and not currently validating */}
              {hasPermission(service, screenType, ACTIONS.VALIDATE) && 
               !getGroupIsValid(currentGroupId) && 
               !validatingGroups.has(currentGroupId) && (
                <Button 
                  onClick={() => handleValidateGroup(currentGroupId)}
                  variant="outlined"
                  startIcon={<CheckIcon />}
                  disabled={processingAction}
                  sx={{
                    color: '#1976d2',
                    borderColor: '#1976d2',
                    textTransform: 'none',
                    '&:hover': {
                      backgroundColor: '#1976d2',
                      color: 'white',
                      borderColor: '#1976d2',
                    }
                  }}
                >
                  Validate Group
                </Button>
              )}


              {/* Validating Indicator */}
              {validatingGroups.has(currentGroupId) && (
                <Button 
                  variant="outlined"
                  disabled
                  startIcon={<CircularProgress size={16} />}
                  sx={{
                    color: '#1976d2',
                    borderColor: '#1976d2',
                    textTransform: 'none',
                  }}
                >
                  Validating...
                </Button>
              )}

              {/* Approve Button - Show if is_valid is true */}
              {hasPermission(service, screenType, ACTIONS.APPROVE) && 
               getGroupIsValid(currentGroupId) && (
                <Button 
                  onClick={() => setApproveDialogOpen(true)}
                  variant="contained"
                  color="success"
                  startIcon={isGroupFailed(currentGroupId) ? <RefreshIcon /> : <CheckCircleIcon />}
                  disabled={processingAction}
                  sx={{
                    backgroundColor: '#66bb6a',
                    textTransform: 'none',
                    '&:hover': {
                      backgroundColor: '#4caf50',
                    }
                  }}
                >
                  {isGroupFailed(currentGroupId) ? 'Retry Group' : 'Approve Group'}
                </Button>
              )}

              {/* Reject Button - Always available */}
              {hasPermission(service, screenType, ACTIONS.REJECT) && (
                <Button 
                  onClick={() => setRejectDialogOpen(true)}
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
                  Reject Group
                </Button>
              )}

            </>
          )}
        </DialogActions>
      </Dialog>

      {/* Approve/Retry Dialog */}
      <Dialog open={approveDialogOpen} onClose={() => setApproveDialogOpen(false)}>
        <DialogTitle>
          {isGroupFailed(currentGroupId) ? 'Retry Model Request' : 'Approve Model Request'}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {isGroupFailed(currentGroupId) 
              ? 'Are you sure you want to retry this failed model request?' 
              : 'Are you sure you want to approve this model request?'}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => setApproveDialogOpen(false)} 
            color="primary"
            disabled={processingAction}
          >
            Cancel
          </Button>
          <Button 
            onClick={handleApprove} 
            color="success" 
            variant="contained"
            startIcon={isGroupFailed(currentGroupId) ? <RefreshIcon /> : <CheckCircleIcon />}
            disabled={processingAction}
            sx={{
              backgroundColor: '#66bb6a',
              textTransform: 'none',
              '&:hover': {
                backgroundColor: '#4caf50',
              }
            }}
          >
            {processingAction ? 'Processing...' : (isGroupFailed(currentGroupId) ? 'Retry' : 'Approve')}
          </Button>
        </DialogActions>
      </Dialog>
      
      {/* Reject Dialog */}
      <Dialog open={rejectDialogOpen} onClose={() => setRejectDialogOpen(false)}>
        <DialogTitle sx={{ bgcolor: '#450839', color: 'white' }}>Reject Model Request</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Please provide a reason for rejecting this model request:
          </DialogContentText>
          <TextField
            autoFocus
            margin="dense"
            label="Reject Reason"
            fullWidth
            multiline
            rows={4}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            variant="outlined"
          />
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => setRejectDialogOpen(false)} 
            color="primary"
            disabled={processingAction}
          >
            Cancel
          </Button>
          <Button 
            onClick={handleReject} 
            color="error" 
            variant="contained"
            startIcon={<CancelIcon />}
            disabled={processingAction || !rejectReason.trim()}
            sx={{
              backgroundColor: '#ef5350',
              textTransform: 'none',
              '&:hover': {
                backgroundColor: '#e53935',
              }
            }}
          >
            {processingAction ? 'Processing...' : 'Reject'}
          </Button>
        </DialogActions>
      </Dialog>
      


      {/* Toast/Snackbar for validation messages */}
      <Snackbar
        open={toastOpen}
        autoHideDuration={6000}
        onClose={() => setToastOpen(false)}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <MuiAlert
          onClose={() => setToastOpen(false)}
          severity={toastSeverity}
          variant="filled"
          sx={{ width: '100%' }}
        >
          {toastMessage}
        </MuiAlert>
      </Snackbar>
    </Box>
  );
};

export default GenericModelApprovalTable;
