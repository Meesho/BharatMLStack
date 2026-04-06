import React, { useState } from 'react';
import {
  Box,
  Typography,
  Popover,
  List,
  ListItem,
  ListItemButton,
  Checkbox,
  Divider,
  Button,
  Chip,
} from '@mui/material';
import FilterListIcon from '@mui/icons-material/FilterList';
import { STATUS_COLORS, REQUEST_STATUS } from '../../../../services/embeddingPlatform/constants';

/**
 * Reusable Status Filter Header Component
 * Provides filtering functionality for status columns in tables
 * 
 * @param {array} selectedStatuses - Array of selected status values
 * @param {function} onStatusChange - Callback when status selection changes
 * @param {array} availableStatuses - Optional array of available statuses (defaults to all)
 */
const StatusFilterHeader = ({ 
  selectedStatuses = [], 
  onStatusChange,
  availableStatuses = null 
}) => {
  const [anchorEl, setAnchorEl] = useState(null);

  // Default status options from constants
  const defaultStatusOptions = [
    { value: REQUEST_STATUS.PENDING, label: 'Pending' },
    { value: REQUEST_STATUS.APPROVED, label: 'Approved' },
    { value: REQUEST_STATUS.REJECTED, label: 'Rejected' },
    { value: REQUEST_STATUS.CANCELLED, label: 'Cancelled' },
    { value: REQUEST_STATUS.FAILED, label: 'Failed' },
  ];

  // Map status options with colors
  const statusOptions = (availableStatuses || defaultStatusOptions).map(status => {
    const statusUpper = typeof status === 'string' ? status.toUpperCase() : status.value;
    const statusLabel = typeof status === 'string' ? status : status.label;
    const statusValue = typeof status === 'string' ? status : status.value;
    
    const colorConfig = STATUS_COLORS[statusUpper] || {
      background: '#EEEEEE',
      text: '#616161'
    };

    return {
      value: statusValue,
      label: statusLabel,
      color: colorConfig.background,
      textColor: colorConfig.text
    };
  });

  const handleClick = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleStatusToggle = (status) => {
    const newStatuses = selectedStatuses.includes(status)
      ? selectedStatuses.filter(s => s !== status)
      : [...selectedStatuses, status];
    onStatusChange(newStatuses);
  };

  const handleSelectAll = () => {
    onStatusChange(statusOptions.map(option => option.value));
  };

  const handleClearAll = () => {
    onStatusChange([]);
  };

  const open = Boolean(anchorEl);
  const hasActiveFilters = selectedStatuses.length > 0 && selectedStatuses.length < statusOptions.length;

  return (
    <>
      <Box 
        sx={{ 
          display: 'flex', 
          flexDirection: 'column',
          alignItems: 'flex-start',
          width: '100%'
        }}
      >
        <Box 
          sx={{ 
            display: 'flex', 
            alignItems: 'center', 
            cursor: 'pointer',
            '&:hover': { backgroundColor: 'rgba(0, 0, 0, 0.04)' },
            borderRadius: 1,
            p: 0.5
          }}
          onClick={handleClick}
        >
          <Typography sx={{ fontWeight: 'bold', color: '#031022' }}>
            Status
          </Typography>
          <FilterListIcon 
            sx={{ 
              ml: 0.5, 
              fontSize: 16,
              color: hasActiveFilters ? '#1976d2' : '#666'
            }} 
          />
          {hasActiveFilters && (
            <Box
              sx={{
                position: 'absolute',
                top: 2,
                right: 2,
                width: 6,
                height: 6,
                borderRadius: '50%',
                backgroundColor: '#1976d2',
              }}
            />
          )}
        </Box>
        
        {/* Show active filters */}
        {selectedStatuses.length > 0 && (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5, maxWidth: '100%' }}>
            {selectedStatuses.slice(0, 2).map((status) => {
              const option = statusOptions.find(opt => opt.value === status);
              return option ? (
                <Chip
                  key={status}
                  label={option.label}
                  size="small"
                  sx={{
                    backgroundColor: option.color,
                    color: option.textColor,
                    fontWeight: 'bold',
                    fontSize: '0.65rem',
                    height: 18,
                    '& .MuiChip-label': { px: 0.5 }
                  }}
                />
              ) : null;
            })}
            {selectedStatuses.length > 2 && (
              <Chip
                label={`+${selectedStatuses.length - 2}`}
                size="small"
                sx={{
                  backgroundColor: '#f5f5f5',
                  color: '#666',
                  fontWeight: 'bold',
                  fontSize: '0.65rem',
                  height: 18,
                  '& .MuiChip-label': { px: 0.5 }
                }}
              />
            )}
          </Box>
        )}
      </Box>
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
          sx: {
            width: 200,
            maxHeight: 300,
            overflow: 'auto'
          }
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

export default StatusFilterHeader;
