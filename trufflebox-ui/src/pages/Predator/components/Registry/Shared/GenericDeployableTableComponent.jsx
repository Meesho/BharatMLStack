import React, { useState } from 'react';
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
  Skeleton,
  Dialog,
  DialogTitle,
  DialogContent,
  Modal,
  Grid,
  Divider,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import TuneIcon from '@mui/icons-material/Tune';
import SettingsIcon from '@mui/icons-material/Settings';
import EventAvailableIcon from '@mui/icons-material/EventAvailable';
import CloseIcon from '@mui/icons-material/Close';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import { useAuth } from '../../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../../constants/permissions';
import { OpenInNew } from '@mui/icons-material';

const GenericDeployableTableComponent = ({
  handleEditAction,
  handleRefreshAction,
  handleTuningAction,
  handleOnboardDeployable,
  deployables,
  loading,
}) => {
  const { user, hasPermission } = useAuth();
  const [searchQuery, setSearchQuery] = useState('');

  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [activityModalOpen, setActivityModalOpen] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const [selectedActivity, setSelectedActivity] = useState(null);
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.DEPLOYABLE;

  const handleConfigClick = (row) => {
    setSelectedConfig({
      name: row.deployableName,
      cpuRequest: row.originalData?.cpuRequest,
      cpuLimit: row.originalData?.cpuLimit,
      cpuRequestUnit: row.originalData?.cpuRequestUnit,
      cpuLimitUnit: row.originalData?.cpuLimitUnit,
      memoryRequest: row.originalData?.memoryRequest,
      memoryLimit: row.originalData?.memoryLimit,
      memoryRequestUnit: row.originalData?.memoryRequestUnit,
      memoryLimitUnit: row.originalData?.memoryLimitUnit,
      gpu_request: row.originalData?.gpu_request,
      gpu_limit: row.originalData?.gpu_limit,
      min_replica: row.originalData?.min_replica,
      max_replica: row.originalData?.max_replica,
      machine_type: row.originalData?.machine_type,
      nodeSelectorValue: row.originalData?.nodeSelectorValue,
      triton_image_tag: row.originalData?.triton_image_tag,
      gcs_bucket_path: row.originalData?.gcs_bucket_path,
      serviceAccount: row.originalData?.serviceAccount,
      cpu_threshold: row.originalData?.cpu_threshold,
      gpu_threshold: row.originalData?.gpu_threshold,
    });
    setConfigModalOpen(true);
  };

  const handleActivityClick = (row) => {
    setSelectedActivity({
      name: row.deployableName,
      createdBy: row.createdBy,
      createdAt: row.createdAt,
      updatedBy: row.updatedBy,
      updatedAt: row.updatedAt,
    });
    setActivityModalOpen(true);
  };

  const handleCloseConfigModal = () => {
    setConfigModalOpen(false);
    setSelectedConfig(null);
  };

  const handleCloseActivityModal = () => {
    setActivityModalOpen(false);
    setSelectedActivity(null);
  };

  const columns = [
    { field: 'deployableName', headerName: 'App Name' },
    { field: 'host', headerName: 'Host' },
    {
      field: 'deployableRunningStatus',
      headerName: 'Deployable Running Status',
      render: (row) => (
        <Chip
          label={row.deployableRunningStatus ? 'Running' : 'Stopped'}
          color={row.deployableRunningStatus ? 'info' : 'error'}
          sx={{
            backgroundColor: row.deployableRunningStatus ? '#E3F2FD' : '#FFEBEE',
            color: row.deployableRunningStatus ? '#0288D1' : '#D32F2F',
            fontWeight: 'bold',
          }}
        />
      ),
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: '250px',
      render: (row) => (
        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', justifyContent: 'flex-start' }}>
          {/* View Resource Configuration */}
          <Tooltip title="View Resource Configuration" disableTransition>
            <IconButton 
              size="small"
              onClick={() => handleConfigClick(row)}
            >
              <SettingsIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Edit Action */}
          {hasPermission(service, screenType, ACTIONS.EDIT) && (
            <Tooltip title="Edit Deployable" disableTransition>
              <IconButton onClick={() => handleEditAction(row)} size="small">
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}

          {/* Activity History */}
          <Tooltip title="View Activity History" disableTransition>
            <IconButton 
              onClick={() => handleActivityClick(row)}
              size="small"
            >
              <EventAvailableIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Monitoring URL */}
          <Tooltip title={row.dashboardLink ? "Open Monitoring Dashboard" : "No dashboard link available"} disableTransition>
            <span>
              <IconButton 
                component={row.dashboardLink ? "a" : "button"}
                href={row.dashboardLink || undefined}
                target={row.dashboardLink ? "_blank" : undefined}
                rel={row.dashboardLink ? "noopener noreferrer" : undefined}
                disabled={!row.dashboardLink}
                size="small"
                sx={{ 
                  color: row.active && row.dashboardLink ? '#2196f3' : '#757575',
                  '&:hover': {
                    color: row.active && row.dashboardLink ? '#1976d2' : '#757575'
                  },
                  '&.Mui-disabled': {
                    color: '#757575'
                  }
                }}
              >
                <OpenInNew fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>

          {/* Refresh Action */}
          <Tooltip title="Refresh Deployable" disableTransition>
            <IconButton onClick={() => handleRefreshAction(row)} size="small">
              <RefreshIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Tuning Action */}
          {hasPermission(service, screenType, ACTIONS.EDIT) && (<Tooltip title="Tune Thresholds" disableTransition>
            <IconButton onClick={() => handleTuningAction(row)} size="small">
              <TuneIcon fontSize="small" />
            </IconButton>
          </Tooltip>)}
        </Box>
      ),
    },
  ];

  const filteredData = React.useMemo(() => {
    if (!deployables) return [];
    
    let filtered = [...deployables];
    
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.deployableName).toLowerCase().includes(searchLower) ||
          String(row.host).toLowerCase().includes(searchLower) ||
          String(row.service).toLowerCase().includes(searchLower)
        );
      });
    }
    
    return filtered;
  }, [deployables, searchQuery]);

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  const parseDate = (dateString) => {
    if (!dateString) return null;
    
    let parsedDate;
    
    if (dateString instanceof Date) {
      parsedDate = dateString;
    }
    else if (typeof dateString === 'string' && dateString.includes(',')) {
      const [datePart, timePart] = dateString.split(', ');
      const [day, month, year] = datePart.split('/');
      parsedDate = new Date(`${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}T${timePart}`);
    }
    else {
      parsedDate = new Date(dateString);
    }
    
    return isNaN(parsedDate.getTime()) ? null : parsedDate;
  };

  const tableCellStyles = {
    borderRight: '1px solid rgba(224, 224, 224, 1)',
    padding: '8px 16px',
    '&:last-child': {
      borderRight: 'none',
    },
  };

  return (
    <Paper elevation={3} sx={{ width: '100%', overflow: 'hidden', boxShadow: 3 }}>
      <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <TextField
          placeholder="Search deployables..."
          variant="outlined"
          size="small"
          value={searchQuery}
          onChange={handleSearchChange}
          sx={{ flex: 1, mr: 2 }}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
        
        {/* {hasPermission(service, screenType, ACTIONS.ONBOARD) && (
          <Button
            variant="contained"
            onClick={handleOnboardDeployable}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730'
              },
            }}
          >
            Onboard Deployable
          </Button>
        )} */}
        <Button
            variant="contained"
            onClick={handleOnboardDeployable}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730'
              },
            }}
          >
            Onboard Deployable
          </Button>
      </Box>

      <TableContainer sx={{ maxHeight: 'calc(100vh - 250px)' }}>
        <Table stickyHeader>
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
                    width: column.width || 'auto',
                    minWidth: column.width || 'auto',
                    maxWidth: column.width || 'none',
                  }}
                >
                  {column.headerName}
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
                      }}
                    >
                      <Skeleton animation="wave" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : filteredData.length > 0 ? (
              filteredData.map((row) => (
                <TableRow key={row.id}>
                  {columns.map((column) => (
                    <TableCell 
                      key={column.field}
                      sx={{
                        ...tableCellStyles,
                        width: column.width || 'auto',
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

      {/* Resource Configuration Modal */}
      <Dialog
        open={configModalOpen}
        onClose={handleCloseConfigModal}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          pb: 1,
          pt: 1
        }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <SettingsIcon />
            <Typography variant="h6">Resource Configuration</Typography>
          </Box>
          <IconButton 
            onClick={handleCloseConfigModal} 
            sx={{ color: 'white' }}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ p: 3 }}>
          {selectedConfig && (
            <Box>
              <Typography variant="h5" sx={{ mb: 3, color: '#450839', fontWeight: 500 }}>
                {selectedConfig.name}
              </Typography>
              
              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, mb: 3 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  CPU RESOURCES
                </Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      REQUEST
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.cpuRequest ? 
                        `${selectedConfig.cpuRequest}${selectedConfig.cpuRequestUnit || ''}` : 'Not Set'}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      LIMIT
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.cpuLimit ? 
                        `${selectedConfig.cpuLimit}${selectedConfig.cpuLimitUnit || ''}` : 'Not Set'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
              
              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, mb: 3 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  MEMORY RESOURCES
                </Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      REQUEST
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.memoryRequest ? 
                        `${selectedConfig.memoryRequest}${selectedConfig.memoryRequestUnit || ''}` : 'Not Set'}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      LIMIT
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.memoryLimit ? 
                        `${selectedConfig.memoryLimit}${selectedConfig.memoryLimitUnit || ''}` : 'Not Set'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>

              {selectedConfig.machine_type === 'GPU' && (
                <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, mb: 3 }}>
                  <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                    GPU RESOURCES
                  </Typography>
                  <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                    <Box>
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        REQUEST
                      </Typography>
                      <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                        {selectedConfig.gpu_request || 'Not Set'}
                      </Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        LIMIT
                      </Typography>
                      <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                        {selectedConfig.gpu_limit || 'Not Set'}
                      </Typography>
                    </Box>
                  </Box>
                </Paper>
              )}

              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  SCALING
                </Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      MINIMUM REPLICAS
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.min_replica || 'Not Set'}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      MAXIMUM REPLICAS
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.max_replica || 'Not Set'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>

              <Divider sx={{ my: 3 }} />

              <Grid container spacing={2}>
                <Grid item xs={12} md={6}>
                  <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1 }}>
                    <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                      CONFIGURATION
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      <Box>
                        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                          MACHINE TYPE
                        </Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedConfig.machine_type || 'Not Set'}
                        </Typography>
                      </Box>
                      <Box>
                        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                          NODE SELECTOR
                        </Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedConfig.nodeSelectorValue || 'Not Set'}
                        </Typography>
                      </Box>
                      <Box>
                        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                          TRITON IMAGE TAG
                        </Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedConfig.triton_image_tag || 'Not Set'}
                        </Typography>
                      </Box>
                    </Box>
                  </Paper>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1 }}>
                    <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                      THRESHOLDS
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      <Box>
                        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                          CPU THRESHOLD
                        </Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedConfig.cpu_threshold ? `${selectedConfig.cpu_threshold}%` : 'Not Set'}
                        </Typography>
                      </Box>
                      {selectedConfig.machine_type === 'GPU' && (
                        <Box>
                          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                            GPU THRESHOLD
                          </Typography>
                          <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                            {selectedConfig.gpu_threshold ? `${selectedConfig.gpu_threshold}%` : 'Not Set'}
                          </Typography>
                        </Box>
                      )}
                    </Box>
                  </Paper>
                </Grid>
              </Grid>
            </Box>
          )}
        </DialogContent>
      </Dialog>

      {/* Activity History Modal */}
      <Modal
        open={activityModalOpen}
        onClose={handleCloseActivityModal}
        aria-labelledby="activity-history-modal"
      >
        <Box sx={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 600,
          maxWidth: '90vw',
          bgcolor: 'background.paper',
          boxShadow: 24,
          borderRadius: 2,
          overflow: 'hidden'
        }}>
          <Box sx={{ 
            bgcolor: '#450839', 
            color: 'white', 
            p: 2,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between'
          }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <EventAvailableIcon />
              <Typography variant="h6" component="h2">
                Activity History
              </Typography>
            </Box>
            <IconButton 
              onClick={handleCloseActivityModal} 
              sx={{ color: 'white' }}
              aria-label="close"
            >
              <CloseIcon />
            </IconButton>
          </Box>
          
          {selectedActivity && (
            <Box sx={{ p: 3 }}>
              <Typography variant="h5" sx={{ mb: 2, color: '#450839', fontWeight: 500 }}>
                {selectedActivity.name}
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
                      {parseDate(selectedActivity.createdAt)?.toLocaleDateString(undefined, { 
                        year: 'numeric', 
                        month: 'long', 
                        day: 'numeric'
                      }) || 'Invalid Date'}
                    </Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {parseDate(selectedActivity.createdAt)?.toLocaleTimeString(undefined, { 
                        hour: '2-digit', 
                        minute: '2-digit',
                        hour12: true
                      }) || 'Invalid Time'}
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
                    <Typography variant="body2">{selectedActivity.updatedBy || 'N/A'}</Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedActivity.updatedAt ? (parseDate(selectedActivity.updatedAt)?.toLocaleDateString(undefined, { 
                        year: 'numeric', 
                        month: 'long', 
                        day: 'numeric'
                      }) || 'Invalid Date') : 'N/A'}
                    </Typography>
                  </Box>
                  
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedActivity.updatedAt ? (parseDate(selectedActivity.updatedAt)?.toLocaleTimeString(undefined, { 
                        hour: '2-digit', 
                        minute: '2-digit',
                        hour12: true
                      }) || 'Invalid Time') : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
            </Box>
          )}
        </Box>
      </Modal>
    </Paper>
  );
};

export default GenericDeployableTableComponent;
