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
  Checkbox,
  Typography,
  Divider,
  Stack,
  Popover,
  List,
  ListItem,
  ListItemButton,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import SearchIcon from '@mui/icons-material/Search';
import FilterListIcon from '@mui/icons-material/FilterList';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useAuth } from '../../Auth/AuthContext';
import ErrorBoundary from '../../../common/ErrorBoundary';

const StatusColumnHeader = ({ selectedStatuses, setSelectedStatuses }) => {
  const [anchorEl, setAnchorEl] = useState(null);
  
  const statusOptions = [
    { value: 'PENDING APPROVAL', label: 'Pending Approval', color: '#FFF8E1', textColor: '#F57C00' },
    { value: 'APPROVED', label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
    { value: 'REJECTED', label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' }
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
          <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', gap: '0.25rem' }}>
          {selectedStatuses.map((status) => {
            const statusConfig = {
              'PENDING APPROVAL': { label: 'Pending', color: '#FFF8E1', textColor: '#F57C00' },
              'APPROVED': { label: 'Approved', color: '#E7F6E7', textColor: '#2E7D32' },
              'REJECTED': { label: 'Rejected', color: '#FFEBEE', textColor: '#D32F2F' }
            };
            const config = statusConfig[status] || { label: status, color: '#f5f5f5', textColor: '#666' };
            return (
              <Chip
                key={status}
                label={config.label}
                size="small"
                onDelete={() => setSelectedStatuses(prev => prev.filter(s => s !== status))}
                sx={{
                  backgroundColor: config.color,
                  color: config.textColor,
                  fontWeight: 'bold',
                  fontSize: '0.5rem',
                  padding: '0.25rem',
                  height: 22,
                  '& .MuiChip-deleteIcon': {
                    color: config.textColor,
                    fontSize: '0.875rem',
                    '&:hover': {
                      color: config.textColor,
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

const TableContent = ({ 
  data, 
  excludeColumns = [], 
  onRowAction, 
  loading,
  actionButtons = [],
  columns,
  searchQuery,
  setSearchQuery,
  selectedStatuses,
  setSelectedStatuses
}) => {
  const { user } = useAuth();

  const filteredAndSortedData = useMemo(() => {
    const safeData = Array.isArray(data) ? data : [];
    let filtered = [...safeData];
    
    // Filter by status
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(row => selectedStatuses.includes(row.Status));
    }
    
    // Filter by search query
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.RequestId).toLowerCase().includes(searchLower) ||
          (row.CreatedBy && row.CreatedBy.toLowerCase().includes(searchLower)) ||
          (row.ApprovedBy && row.ApprovedBy.toLowerCase().includes(searchLower)) ||
          (row.EntityLabel && row.EntityLabel.toLowerCase().includes(searchLower)) ||
          (row.FeatureGroupLabel && row.FeatureGroupLabel.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => b.RequestId - a.RequestId);
  }, [data, searchQuery, selectedStatuses]);

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  // Common table cell styles with vertical divider
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
          alignItems: 'center',
          flexWrap: 'wrap'
        }}>
        <TextField
          placeholder={user?.role === 'admin' 
            ? "Search by Request ID, Admin POC, or Created By"
            : "Search by Request ID or Admin POC"
          }
          variant="outlined"
          size="small"
          sx={{ flex: 1, minWidth: '300px' }}
          value={searchQuery}
          onChange={handleSearchChange}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />

        <Box sx={{ display: 'flex', gap: '0.5rem' }}>
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
      <TableContainer component={Paper} elevation={3} sx={{ marginTop: '1rem', flex: 1 }}>
        <Table stickyHeader>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              {columns.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={tableCellStyles}
                >
                  {column.field === 'Status' ? (
                    <StatusColumnHeader 
                      selectedStatuses={selectedStatuses}
                      setSelectedStatuses={setSelectedStatuses}
                    />
                  ) : (
                    <b>{column.headerName}</b>
                  )}
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
                      {/* Placeholder for Skeleton */}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : filteredAndSortedData.length === 0 ? (
              <TableRow>
                <TableCell 
                  colSpan={columns.length} 
                  sx={{ 
                    textAlign: 'center', 
                    py: 4, 
                    color: 'text.secondary',
                    fontStyle: 'italic'
                  }}
                >
                  No records found matching the current filters
                </TableCell>
              </TableRow>
            ) : (
              filteredAndSortedData.map((row) => (
                <TableRow
                  key={row.RequestId}
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
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

const GenericTable = ({ 
  data, 
  excludeColumns = [], 
  onRowAction, 
  loading,
  actionButtons = [],
  flowType = 'default'
})  => {
  const { user } = useAuth();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatuses, setSelectedStatuses] = useState(['PENDING APPROVAL']); // Initialize with PENDING APPROVAL only
  
  const allColumns = [
    { field: 'RequestId', headerName: 'Request ID' },
    { field: 'EntityLabel', headerName: 'Entity Label' },
    { field: 'FeatureGroupLabel', headerName: 'FG Label' },
    { 
      field: 'CreatedBy', 
      headerName: 'Created By',
      roles: ['admin']
    },
    { field: 'ApprovedBy', headerName: 'Admin POC' },
    { field: 'CreatedAt', headerName: 'Created At' },
    { field: 'UpdatedAt', headerName: 'Updated At' },
    {
      field: 'Status',
      headerName: 'Status',
      render: (row) => (
        <Chip
          label={row.Status}
          color={
            row.Status === 'APPROVED'
              ? 'success'
              : row.Status === 'REJECTED'
              ? 'error'
              : undefined
          }
          sx={{
            backgroundColor: row.Status === 'APPROVED' 
              ? '#E7F6E7'
              : row.Status === 'REJECTED'
              ? '#FFEBEE'
              : row.Status === 'PENDING APPROVAL'
              ? '#FFF8E1'
              : undefined,
            color: row.Status === 'APPROVED'
              ? '#2E7D32'
              : row.Status === 'REJECTED'
              ? '#D32F2F'
              : row.Status === 'PENDING APPROVAL'
              ? '#F57C00'
              : undefined,
            fontWeight: 'bold',
          }}
        />
      ),
    },
    {
      field: 'Actions',
      headerName: 'Actions',
      render: (row) => {
        const showActionIcon = flowType === 'approval' && row.Status === 'PENDING APPROVAL';
        
        return (
          <Tooltip title={showActionIcon ? "Take Action" : "View Details"}>
            <IconButton 
              onClick={() => onRowAction(row)}
              sx={{
                color: showActionIcon ? '#450839' : '#666',
                // backgroundColor: showActionIcon ? 'rgba(69, 8, 57, 0.08)' : 'rgba(0, 0, 0, 0.04)',
                '&:hover': {
                  backgroundColor: showActionIcon ? 'rgba(69, 8, 57, 0.12)' : 'rgba(0, 0, 0, 0.08)',
                  color: showActionIcon ? '#380730' : '#333',
                  transform: 'scale(1.05)',
                },
                transition: 'all 0.2s ease-in-out',
              }}
            >
              {showActionIcon ? (
                <EditIcon fontSize="small" />
              ) : (
                <VisibilityIcon fontSize="small" />
              )}
            </IconButton>
          </Tooltip>
        );
      },
    },
  ];

  const columns = allColumns.filter((col) => {
    if (excludeColumns.includes(col.field)) return false;
    if (col.roles) {
      return col.roles.includes(user?.role);
    }
    return true;
  });

  return (
    // Error boundary catches errors from TableContent data processing
    // Uses table-specific fallback message instead of app-level error handling
    <ErrorBoundary 
      fallbackMessage="Table failed to load data. Please refresh the page."
    >
      <TableContent 
        data={data}
        excludeColumns={excludeColumns}
        onRowAction={onRowAction}
        loading={loading}
        actionButtons={actionButtons}
        columns={columns}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        selectedStatuses={selectedStatuses}
        setSelectedStatuses={setSelectedStatuses}
      />
    </ErrorBoundary>
  );
};

export default GenericTable;
