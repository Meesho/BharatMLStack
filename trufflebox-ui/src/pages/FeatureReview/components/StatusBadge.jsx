import React from 'react';
import { Chip } from '@mui/material';

const statusConfig = {
  pending: { label: 'Pending', color: 'warning' },
  plan_computed: { label: 'Plan Computed', color: 'info' },
  approved: { label: 'Approved', color: 'success' },
  rejected: { label: 'Rejected', color: 'error' },
  merged: { label: 'Merged', color: 'default' },
  closed: { label: 'Closed', color: 'default' },
};

const StatusBadge = ({ status }) => {
  const config = statusConfig[status] || { label: status, color: 'default' };
  return (
    <Chip
      label={config.label}
      color={config.color}
      size="small"
      variant={status === 'merged' || status === 'closed' ? 'outlined' : 'filled'}
    />
  );
};

export default StatusBadge;
