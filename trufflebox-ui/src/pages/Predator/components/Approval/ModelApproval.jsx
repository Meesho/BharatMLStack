import React, { useCallback } from 'react';
import { Alert, Spinner } from 'react-bootstrap';
import { useAuth } from '../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';
import GenericModelApprovalTable from './GenericModelApprovalTable';
import * as URL_CONSTANTS from '../../../../config';

const ModelApproval = () => {
  const { 
    hasPermission, 
    permissions,
    user
  } = useAuth();

  const service = SERVICES.PREDATOR;
  const screenType = SCREEN_TYPES.PREDATOR.MODEL_APPROVAL;
  const isPermissionsLoaded = permissions !== null;

  const handleValidate = useCallback(async (groupId) => {
    if (!hasPermission(service, screenType, ACTIONS.VALIDATE)) {
      console.warn('User does not have permission to validate');
      return false;
    }

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-approval/requests/${groupId}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${user.token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        console.log('Failed to validate model');
        return {
          success: false,
          is_valid: false
        };
      }

      const result = await response.json();
      if (result.error) {
        console.log(result.error);
        return {
          success: false,
          is_valid: false
        };
      }

      // Check if validation was successful
      if (result.data === "Request validation completed successfully") {
        // Return both success status and is_valid field
        return {
          success: true,
          is_valid: result.is_valid || false
        };
      } else {
        console.warn('Validation failed:', result.data);
        return {
          success: false,
          is_valid: false
        };
      }
    } catch (error) {
      console.error('Error validating model:', error);
      return {
        success: false,
        is_valid: false
      };
    }
  }, [hasPermission, service, screenType, user.token]);

  const handleReview = useCallback(async (groupId, action, reason = '') => {
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
        group_id: groupId.toString(),
        status: action,
        approved_by: user.email,
        reject_reason: reason || ''
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-approval/process-request`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${user.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        console.log(`Failed to ${action.toLowerCase()} model`);
      }

      const result = await response.json();
      if (result.error) {
        console.log(result.error);
      }

      return true;
    } catch (error) {
      console.error(`Error ${action.toLowerCase()} model:`, error);
      return false;
    }
  }, [hasPermission, service, screenType, user.token]);

  if (!isPermissionsLoaded) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <Spinner animation="border" role="status">
          <span className="visually-hidden">Loading permissions...</span>
        </Spinner>
      </div>
    );
  }

  if (!hasPermission(service, screenType, ACTIONS.VIEW)) {
    return (
      <div style={{ 
        display: 'flex', 
        flexDirection: 'column', 
        alignItems: 'center', 
        justifyContent: 'center', 
        minHeight: '300px',
        textAlign: 'center',
        padding: '20px'
      }}>
        <Alert variant="warning">
          <Alert.Heading>Access Denied</Alert.Heading>
          <p>You do not have permission to view model approvals.</p>
        </Alert>
      </div>
    );
  }

  return (
    <div>
      <GenericModelApprovalTable 
        onValidate={handleValidate}
        onReview={handleReview}
      />
    </div>
  );
};

export default ModelApproval;
