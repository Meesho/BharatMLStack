import React, { useState, useEffect, useCallback } from 'react';
import { Box, Typography, Snackbar, Alert, CircularProgress } from '@mui/material';
import NumerixApprovalTable from '../shared/NumerixApprovalTable';
import { useAuth } from '../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../constants/permissions';
import axios from 'axios';
import useFormatDate from '../../../hooks/useFormatDate';
import * as URL_CONSTANTS from '../../../config';

const NumerixConfigApproval = () => {
  const { user, hasPermission, permissions } = useAuth();
  const { formatDateToIST } = useFormatDate();
  
  const service = SERVICES.NUMERIX;
  const screenType = SCREEN_TYPES.NUMERIX.CONFIG_APPROVAL;
  const isPermissionsLoaded = permissions !== null;
  const [loading, setLoading] = useState(true);
  const [configData, setConfigData] = useState([]);
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  const fetchApprovalConfigs = useCallback(async () => {
    try {
      setLoading(true);
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-approval/configs`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`
          }
        }
      );
      
      if (response.data) {
        // Transform data to match the table format
        const transformedData = response.data.data.map(item => ({
          request_id: item.request_id,
          compute_id: item.compute_id,
          payload: item.payload?.config_value || item.payload,
          created_by: item.created_by,
          created_at: formatDateToIST(item.created_at),
          updated_by: item.updated_by,
          updated_at: formatDateToIST(item.updated_at),
          status: item.status,
          request_type: item.request_type,
          reviewer: item.reviewer || "",
          reject_reason: item.reject_reason || "",
          deployable_running_status: item.deployable_running_status === "true"
        }));
        setConfigData(transformedData);
      } else {
        showNotification(response.data?.error || 'Config data loading error', 'error');
      }
    } catch (error) {
      showNotification('Failed to load configuration data', 'error');
      console.log('Error fetching approval configs:', error);
    } finally {
      setLoading(false);
    }
  }, [user.token, formatDateToIST]);

  useEffect(() => {
    if (isPermissionsLoaded && hasPermission(service, screenType, ACTIONS.VIEW)) {
      fetchApprovalConfigs();
    } else if (isPermissionsLoaded) {
      setLoading(false);
    }
  }, [isPermissionsLoaded, hasPermission, service, screenType, fetchApprovalConfigs]);

  const handleApproveRequest = async (requestId) => {
    if (!hasPermission(service, screenType, ACTIONS.APPROVE)) {
      showNotification('You do not have permission to approve configurations', 'error');
      return;
    }
    
    try {
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-approval/review`,
        {
          request_id: requestId,
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
        fetchApprovalConfigs();
      } else {
        showNotification(response.data?.error || 'Error approving request', 'error');
      }
    } catch (error) {
      showNotification(error.response?.data?.error || 'Failed to approve request', 'error');
      console.log('Error approving request:', error);
    }
  };

  const handleRejectRequest = async (requestId, rejectReason) => {
    if (!hasPermission(service, screenType, ACTIONS.REJECT)) {
      showNotification('You do not have permission to reject configurations', 'error');
      return;
    }
    
    try {
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-approval/review`,
        {
          request_id: requestId,
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

      if (response.data && response.data.error === "") {
        showNotification('Request rejected successfully', 'success');
        fetchApprovalConfigs();
      } else {
        showNotification(response.data?.error || 'Error rejecting request', 'error');
      }
    } catch (error) {
      showNotification(error.response?.data?.error || 'Failed to reject request', 'error');
      console.log('Error rejecting request:', error);
    }
  };

  const handleCancelRequest = async (requestId) => {
    if (!hasPermission(service, screenType, ACTIONS.CANCEL)) {
      showNotification('You do not have permission to cancel configurations', 'error');
      return;
    }
    
    try {
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-approval/cancel`,
        {
          request_id: requestId,
          updated_by: user.email
        },
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data && response.data.error === "") {
        showNotification('Request cancelled successfully', 'success');
        fetchApprovalConfigs();
      } else {
        showNotification(response.data?.error || 'Error cancelling request', 'error');
      }
    } catch (error) {
      showNotification(error.response?.data?.error || 'Failed to cancel request', 'error');
      console.log('Error cancelling request:', error);
    }
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

  if (!isPermissionsLoaded) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!hasPermission(service, screenType, ACTIONS.VIEW)) {
    return (
      <Box sx={{ padding: '1.5rem' }}>
        <Alert severity="warning">
          <Typography variant="h6">Access Denied</Typography>
          <Typography>You do not have permission to view Numerix config approvals.</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ padding: '20px' }}>
      
      {loading && configData.length === 0 ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <CircularProgress sx={{ color: '#450839' }} />
        </Box>
      ) : (
        <NumerixApprovalTable
          data={configData}
          loading={loading}
          onRefresh={fetchApprovalConfigs}
          onApprove={handleApproveRequest}
          onReject={handleRejectRequest}
          onCancel={handleCancelRequest}
        />
      )}

      {/* Notification */}
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
    </Box>
  );
};

export default NumerixConfigApproval;