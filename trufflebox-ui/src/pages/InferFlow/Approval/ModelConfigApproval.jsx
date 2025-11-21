import React, { useCallback } from 'react';
import { Box, Typography, CircularProgress, Alert } from '@mui/material';
import { useAuth } from '../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../constants/permissions';
import GenericModelConfigApprovalTable from './GenericModelConfigApprovalTable';
import * as URL_CONSTANTS from '../../../config';

const ModelConfigApproval = () => {
  const { 
    hasPermission, 
    permissions,
    user
  } = useAuth();
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.MP_CONFIG_APPROVAL;
  const isPermissionsLoaded = permissions !== null;

  const handleReview = useCallback(async (requestId, action, reason = '') => {
    const actionMap = {
      'Approved': ACTIONS.APPROVE,
      'Rejected': ACTIONS.REJECT
    };

    const requiredAction = actionMap[action];
    if (!hasPermission(service, screenType, requiredAction)) {
      console.warn(`User does not have permission to ${action.toLowerCase()}`);
      return false;
    }

    try {
      const payload = {
        request_id: requestId,
        status: action.toLowerCase(),
        reviewer: user.email,
        reject_reason: reason || ''
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/mp-config-approval/review`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${user.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        console.log(`Failed to ${action.toLowerCase()} configuration`);
      }

      const result = await response.json();
      if (result.error) {
        console.log(result.error);
      }

      return true;
    } catch (error) {
      console.error(`Error ${action.toLowerCase()} configuration:`, error);
      return false;
    }
  }, [hasPermission, service, screenType, user.token]);

  if (!isPermissionsLoaded) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!hasPermission(service, screenType, ACTIONS.VIEW)) {
    return (
      <Box sx={{ 
        display: 'flex', 
        flexDirection: 'column', 
        alignItems: 'center', 
        justifyContent: 'center', 
        minHeight: '300px',
        textAlign: 'center',
        padding: '20px'
      }}>
        <Alert severity="warning">
          <Typography variant="h6">Access Denied</Typography>
          <Typography>You do not have permission to view InferFlow config approvals.</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box>
        <GenericModelConfigApprovalTable
        onReview={handleReview}
      />
    </Box>
  );
};

export default ModelConfigApproval;
