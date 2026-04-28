import React from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import Layout from './Layout';
import { Spinner } from 'react-bootstrap';
import { REACT_APP_ENVIRONMENT } from '../../config';
import { APP_ROUTES, ENV_CONFIG } from '../../constants/authConstants';

const ProtectedRoute = ({
  children,
  allowedRoles,
  service,
  screenType,
  requiredActions = [],
  requireAllActions = false
}) => {
  const { isAuthenticated, hasPermission, hasScreenAccess, permissions, loading } = useAuth();

  // Check if staging bypass is enabled (configurable via env var)
  // WARNING: This is a temporary workaround - should be disabled in production
  const isStaging = REACT_APP_ENVIRONMENT.toLowerCase() === 'staging';
  const shouldBypass = ENV_CONFIG.ENABLE_STAGING_BYPASS && isStaging;

  // If still loading auth state, show loading
  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}><Spinner /></div>;
  }
console.log('shouldBypass', shouldBypass, isAuthenticated, permissions);
  // If not authenticated, redirect to login
  if (!isAuthenticated) {
    return <Navigate to={APP_ROUTES.LOGIN} replace />;
  }

  // Skip permission checks in staging environment (only if explicitly enabled)
  if (shouldBypass) {
    return <Layout>{children}</Layout>;
  }

  if (permissions === null) {
    return <div>Loading permissions...</div>;
  }

  // Super admin bypass - has access to everything
  const userRole = permissions?.role;
  if (userRole === 'super_admin') {
    return <Layout>{children}</Layout>;
  }

  // Legacy role-based check (keep for backward compatibility)
  if (allowedRoles && allowedRoles.length > 0) {
    if (!userRole || !allowedRoles.includes(userRole)) {
      return <Navigate to={APP_ROUTES.UNAUTHORIZED} replace />;
    }
  }

  // Permission-based access control
  if (service && screenType) {
    if (!hasScreenAccess(service, screenType)) {
      return <Navigate to={APP_ROUTES.UNAUTHORIZED} replace />;
    }

    if (requiredActions && requiredActions.length > 0) {
      const hasRequiredPermissions = requireAllActions
        ? requiredActions.every(action => hasPermission(service, screenType, action))
        : requiredActions.some(action => hasPermission(service, screenType, action));

      if (!hasRequiredPermissions) {
        return <Navigate to={APP_ROUTES.UNAUTHORIZED} replace />;
      }
    }
  }

  return <Layout>{children}</Layout>;
};

export default ProtectedRoute;
