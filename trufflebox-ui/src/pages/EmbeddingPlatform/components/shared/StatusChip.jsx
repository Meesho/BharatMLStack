import React from 'react';
import { Chip } from '@mui/material';
import { STATUS_COLORS } from '../../../../services/embeddingPlatform/constants';

/**
 * Reusable Status Chip Component
 * Displays status with appropriate color coding
 * 
 * @param {string} status - The status value to display
 * @param {object} sx - Additional Material-UI sx props
 */
const StatusChip = ({ status, sx = {} }) => {
  const statusUpper = (status || '').toUpperCase();
  
  // Get color from constants if available, otherwise use defaults
  const statusConfig = STATUS_COLORS[statusUpper] || {
    background: '#EEEEEE',
    text: '#616161',
    label: status || 'N/A'
  };

  return (
    <Chip
      label={statusConfig.label || status || 'N/A'}
      sx={{
        backgroundColor: statusConfig.background,
        color: statusConfig.text,
        fontWeight: 'bold',
        fontSize: '0.75rem',
        ...sx
      }}
    />
  );
};

export default StatusChip;
