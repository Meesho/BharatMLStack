import React from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import Layout from './Layout';

const ProtectedRoute = ({ children, allowedRoles }) => {
  const { isAuthenticated, hasRole } = useAuth();

  if (!isAuthenticated) {
    console.log(isAuthenticated);
    return <Navigate to="/login" replace />;
  }

  if (allowedRoles && !allowedRoles.some(hasRole)) {
    return <Navigate to="/unauthorized" replace />;
  }

  return <Layout>{children}</Layout>;
};

export default ProtectedRoute;
