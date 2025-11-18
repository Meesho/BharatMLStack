import React, { createContext, useState, useContext, useEffect, useCallback, useMemo, useRef } from 'react';
import * as URL_CONSTANTS from '../../config';
import httpInterceptor from '../../services/httpInterceptor';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [permissions, setPermissions] = useState(null);
  const [loading, setLoading] = useState(true);
  const [loadingPermissions, setLoadingPermissions] = useState(false);
  const isAuthenticated = !!user;
  
  // Use ref to track if permissions are being fetched to prevent duplicates
  const fetchingPermissionsRef = useRef(false);
  const permissionsTokenRef = useRef(null);

  const fetchPermissions = useCallback(async (token) => {
    // Prevent duplicate calls with same token
    if (fetchingPermissionsRef.current && permissionsTokenRef.current === token) {
      return { success: true, data: permissions };
    }
    
    // If permissions already exist for this token, don't refetch
    if (permissions && permissionsTokenRef.current === token) {
      return { success: true, data: permissions };
    }

    fetchingPermissionsRef.current = true;
    permissionsTokenRef.current = token;
    setLoadingPermissions(true);

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/permission-by-role`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        console.log(`Failed to fetch permissions: ${response.status}`);
        setPermissions(null);
        // Reset permission tracking refs on failure
        fetchingPermissionsRef.current = false;
        permissionsTokenRef.current = null;
        return { success: false, status: response.status, isUnauthorized: response.status === 401 };
      }

      const result = await response.json();
      if (result.error) {
        setPermissions(null);
        console.log(`Error fetching permissions: ${result.error}`);
        return { success: false, error: result.error };
      }

      setPermissions(result);
      return { success: true, data: result };
    } catch (error) {
      setPermissions(null);
      console.log('Error fetching permissions:', error);
      // Reset permission tracking refs on error
      fetchingPermissionsRef.current = false;
      permissionsTokenRef.current = null;
      return { success: false, error: error.message };
    } finally {
      fetchingPermissionsRef.current = false;
      setLoadingPermissions(false);
    }
  }, [permissions]);

  const hasPermission = useCallback((service, screenType, action) => {
    if (!permissions || !permissions.permissions) {
      return false;
    }

    const servicePermission = permissions.permissions.find(p => p.service === service);
    if (!servicePermission) {
      return false;
    }
    const screenPermission = servicePermission.screens.find(s => s.screenType === screenType);
    if (!screenPermission) {
      return false;
    }
    return screenPermission.allowedActions.includes(action);
  }, [permissions]);

  const hasScreenAccess = useCallback((service, screenType) => {
    if (!permissions || !permissions.permissions) {
      return false;
    }

    const servicePermission = permissions.permissions.find(p => p.service === service);
    if (!servicePermission) {
      return false;
    }

    const screenPermission = servicePermission.screens.find(s => s.screenType === screenType);
    return !!screenPermission;
  }, [permissions]);

  const getAllowedActions = useCallback((service, screenType) => {
    if (!permissions || !permissions.permissions) {
      return [];
    }

    const servicePermission = permissions.permissions.find(p => p.service === service);
    if (!servicePermission) {
      return [];
    }

    const screenPermission = servicePermission.screens.find(s => s.screenType === screenType);
    return screenPermission ? screenPermission.allowedActions : [];
  }, [permissions]);

  const getUserRole = useCallback(() => {
    return permissions?.role || null;
  }, [permissions]);

  useEffect(() => {
    const initializeAuth = async () => {
      const storedUser = JSON.parse(localStorage.getItem('user'));
      if (storedUser && storedUser.token) {
        setUser(storedUser);
        
        // Try to fetch permissions with the stored token
        const permissionsResult = await fetchPermissions(storedUser.token);
        
        // Only logout if it's specifically a 401 unauthorized response (expired token)
        if (!permissionsResult.success && permissionsResult.isUnauthorized) {
          console.log('Token expired during initialization, logging out');
          setUser(null);
          setPermissions(null);
          localStorage.removeItem('authToken');
          localStorage.removeItem('user');
          
          // Reset permission tracking refs
          fetchingPermissionsRef.current = false;
          permissionsTokenRef.current = null;
        }
      }
      setLoading(false);
    };

    initializeAuth();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const login = useCallback(async (email, role, token) => {
    const userData = { email, role, token };
    setUser(userData);
    localStorage.setItem('user', JSON.stringify(userData));
    localStorage.setItem('authToken', token);
    
    // Only fetch permissions if we don't have them or if token changed
    if (!permissions || permissionsTokenRef.current !== token) {
      await fetchPermissions(token);
    }
  }, [fetchPermissions, permissions]);

  const logout = useCallback(async () => {
    try {
      const token = user?.token;

      if (token) {
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/logout`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
          },
        });

        if (!response.ok) {
          console.log('Failed to log out');
        }
      }
    } catch (error) {
      console.log('Error during logout:', error);
    } finally {
      setUser(null);
      setPermissions(null);
      localStorage.removeItem('authToken');
      localStorage.removeItem('user');
      
      // Reset permission tracking refs
      fetchingPermissionsRef.current = false;
      permissionsTokenRef.current = null;
    }
  }, [user?.token]);

  useEffect(() => {
    httpInterceptor.init(logout);

    return () => {
      httpInterceptor.cleanup();
    };
  }, [logout]);

  const contextValue = useMemo(() => ({
    user,
    permissions,
    isAuthenticated,
    loading: loading || loadingPermissions,
    login,
    logout,
    hasPermission,
    hasScreenAccess,
    getAllowedActions,
    getUserRole,
    fetchPermissions
  }), [user, permissions, isAuthenticated, loading, loadingPermissions, login, logout, hasPermission, hasScreenAccess, getAllowedActions, getUserRole, fetchPermissions]);

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);