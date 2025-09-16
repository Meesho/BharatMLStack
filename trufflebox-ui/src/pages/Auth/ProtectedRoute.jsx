import React from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import Layout from './Layout';
import { Spinner } from 'react-bootstrap';

const ProtectedRoute = ({
  children,
  allowedRoles,
  service,
  screenType,
  requiredActions = [],
  requireAllActions = false
}) => {
  const { isAuthenticated, hasPermission, hasScreenAccess, permissions, loading } = useAuth();

  // If still loading auth state, show loading
  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}><Spinner /></div>;
  }

  // If not authenticated, redirect to login
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (permissions === null) {
    return <div>Loading permissions...</div>;
  }

  // Legacy role-based check (keep for backward compatibility)
  if (allowedRoles && allowedRoles.length > 0) {
    const userRole = permissions?.role;
    if (!userRole || !allowedRoles.includes(userRole)) {
      return <Navigate to="/unauthorized" replace />;
    }
  }

  // Permission-based access control
  if (service && screenType) {
    if (!hasScreenAccess(service, screenType)) {
      return <Navigate to="/unauthorized" replace />;
    }

    if (requiredActions && requiredActions.length > 0) {
      const hasRequiredPermissions = requireAllActions
        ? requiredActions.every(action => hasPermission(service, screenType, action))
        : requiredActions.some(action => hasPermission(service, screenType, action));

      if (!hasRequiredPermissions) {
        const actionType = requireAllActions ? 'all' : 'any';
        return <Navigate to="/unauthorized" replace />;
      }
    }
  }

  return <Layout>{children}</Layout>;
};

export default ProtectedRoute;
