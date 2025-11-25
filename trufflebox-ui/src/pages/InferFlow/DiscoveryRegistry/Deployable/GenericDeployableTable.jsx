import React, { useState, useMemo } from 'react';
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
  Skeleton,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import SearchIcon from '@mui/icons-material/Search';
import HistoryIcon from '@mui/icons-material/History';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CloseIcon from '@mui/icons-material/Close';
import DashboardIcon from '@mui/icons-material/Dashboard';
import MemoryIcon from '@mui/icons-material/Memory';
import EventAvailableIcon from '@mui/icons-material/EventAvailable';
import { useAuth } from '../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';
import Modal from '@mui/material/Modal';
import { OpenInNew } from '@mui/icons-material';

const GenericDeployableTable = ({ 
  data, 
  excludeColumns = [], 
  onViewDetails,
  onEditDeployable,
  onOnboardDeployable,
  loading,
  actionButtons = [],
}) => {
  const { user, hasPermission } = useAuth();
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.DEPLOYABLE;
  const [searchQuery, setSearchQuery] = useState('');
  const [activityModalOpen, setActivityModalOpen] = useState(false);
  const [selectedActivity, setSelectedActivity] = useState(null);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState(null);
  
  const allColumns = [
    { field: 'Name', headerName: 'App Name' },
    { field: 'Host', headerName: 'Host' },
    {
      field: 'RunningStatus',
      headerName: 'Deployable Running Status',
      render: (row) => (
        <Chip
          label={row.DeployableRunningStatus ? 'Running' : 'Stopped'}
          color={row.DeployableRunningStatus ? 'info' : 'error'}
          sx={{
            backgroundColor: row.DeployableRunningStatus ? '#E3F2FD' : '#FFEBEE',
            color: row.DeployableRunningStatus ? '#0288D1' : '#D32F2F',
            fontWeight: 'bold',
          }}
        />
      ),
    },
    {
      field: 'Actions',
      headerName: 'Actions',
      width: '250px',
      render: (row) => (
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', justifyContent: 'flex-start' }}>
          {/* View Details Action */}
          {hasPermission(service, screenType, ACTIONS.VIEW) && (
            <Tooltip title="View Details" disableTransition>
              <IconButton onClick={() => onViewDetails(row)} size="small">
                <InfoIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}

          {/* Activity History */}
          <Tooltip title="View Activity History" disableTransition>
            <IconButton 
              onClick={(e) => {
                e.stopPropagation();
                setSelectedActivity({
                  createdBy: row.CreatedBy,
                  updatedBy: row.UpdatedBy,
                  createdAt: row.CreatedAt,
                  updatedAt: row.UpdatedAt,
                  name: row.Name
                });
                setActivityModalOpen(true);
              }}
              size="small"
            >
              <EventAvailableIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Monitoring Dashboard */}
          <Tooltip title={row.DashboardLink ? "Open Dashboard" : "No dashboard link available"} disableTransition>
            <span>
              <IconButton 
                component={row.DashboardLink ? "a" : "button"}
                href={row.DashboardLink || undefined}
                target={row.DashboardLink ? "_blank" : undefined}
                rel={row.DashboardLink ? "noopener noreferrer" : undefined}
                disabled={!row.DashboardLink}
                size="small"
                sx={{ 
                  color: row.DashboardLink ? '#2196f3' : '#757575',
                  '&:hover': {
                    color: row.DashboardLink ? '#1976d2' : '#757575'
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

          {/* Additional Action Buttons */}
          {actionButtons.map((button, idx) => (
            <Tooltip key={idx} title={button.tooltip || ''} disableTransition>
              <IconButton onClick={() => button.onClick(row)} size="small">
                {button.icon}
              </IconButton>
            </Tooltip>
          ))}
        </Box>
      ),
    },
  ];

  const columns = allColumns.filter((col) => {
    if (excludeColumns.includes(col.field)) return false;
    if (col.roles) {
      return col.roles.includes(user?.role);
    }
    return true;
  });

  const filteredAndSortedData = useMemo(() => {
    let filtered = [...data];
    
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.Name).toLowerCase().includes(searchLower) ||
          (row.Host && row.Host.toLowerCase().includes(searchLower)) ||
          (row.Service && row.Service.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.CreatedAt) - new Date(a.CreatedAt);
    });
  }, [data, searchQuery]);

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  const tableCellStyles = {
    borderRight: '1px solid rgba(224, 224, 224, 1)',
    '&:last-child': {
      borderRight: 'none',
    },
  };

  return (
    <Paper elevation={3} sx={{ width: '100%', height: '100vh', padding: '1rem', display: 'flex', flexDirection: 'column', marginTop: '1rem' }}>
      <Box
        sx={{ 
          marginBottom: '1rem', 
          display: 'flex', 
          gap: '1rem', 
          alignItems: 'center'
        }}>
        <TextField
          placeholder="Search deployables..."
          variant="outlined"
          size="small"
          sx={{ flex: 1 }}
          value={searchQuery}
          onChange={handleSearchChange}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
        <Box sx={{ display: 'flex', gap: '0.5rem' }}>
          {hasPermission(service, screenType, ACTIONS.ONBOARD) && (
            <Button
              variant="contained"
              onClick={onOnboardDeployable}
              sx={{
                backgroundColor: '#450839',
                '&:hover': {
                  backgroundColor: '#380730'
                },
              }}
            >
              Onboard Deployable
            </Button>
          )}
          {actionButtons.map((button, index) => (
            <Button
              key={index}
              variant={button.variant || "contained"}
              onClick={button.onClick}
              sx={{
                backgroundColor: button.color || '#450839',
                '&:hover': {
                  backgroundColor: button.hoverColor || '#380730'
                },
                ...button.sx
              }}
            >
              {button.label}
            </Button>
          ))}
        </Box>
      </Box>

      <TableContainer 
        component={Paper} 
        elevation={3} 
        sx={{ 
          marginTop: '1rem',
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
        }}
      >
        <Table stickyHeader>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              {columns.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={{
                    ...tableCellStyles,
                    backgroundColor: '#E6EBF2',
                    fontWeight: 'bold',
                    color: '#031022',
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
                      sx={tableCellStyles}
                    >
                      <Skeleton animation="wave" width="100%" height={24} />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              filteredAndSortedData.map((row) => (
                <TableRow
                  key={row.Id}
                  hover
                  style={{
                    cursor: 'pointer',
                  }}
                >
                  {columns.map((column) => (
                    <TableCell 
                      key={column.field}
                      sx={tableCellStyles}
                    >
                      {column.render ? column.render(row) : row[column.field]}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
            {!loading && filteredAndSortedData.length === 0 && (
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
        onClose={() => setConfigModalOpen(false)}
        maxWidth="sm"
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
            <MemoryIcon />
            <Typography variant="h6">Resource Configuration</Typography>
          </Box>
          <IconButton 
            onClick={() => setConfigModalOpen(false)} 
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
                      {selectedConfig.cpuRequest}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      LIMIT
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.cpuLimit}
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
                      {selectedConfig.memRequest}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                      LIMIT
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                      {selectedConfig.memLimit}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
              
              {(selectedConfig.minReplica || selectedConfig.maxReplica) && (
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
                        {selectedConfig.minReplica || 'Not set'}
                      </Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        MAXIMUM REPLICAS
                      </Typography>
                      <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                        {selectedConfig.maxReplica || 'Not set'}
                      </Typography>
                    </Box>
                  </Box>
                </Paper>
              )}
            </Box>
          )}
        </DialogContent>
      </Dialog>

      {/* Activity History Modal */}
      <Modal
        open={activityModalOpen}
        onClose={() => setActivityModalOpen(false)}
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
              <HistoryIcon />
              <Typography variant="h6" component="h2">
                Activity History
              </Typography>
            </Box>
            <IconButton 
              onClick={() => setActivityModalOpen(false)} 
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
        </Box>
      </Modal>
    </Paper>
  );
};

export default GenericDeployableTable;
