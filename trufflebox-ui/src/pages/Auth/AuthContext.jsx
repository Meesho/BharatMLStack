import React, { createContext, useState, useContext, useEffect, useCallback, useMemo, useRef } from 'react';
import * as URL_CONSTANTS from '../../config';
import { REACT_APP_ENVIRONMENT } from '../../config';
import httpInterceptor from '../../services/httpInterceptor';
import { 
  STORAGE_KEYS, 
  TOKEN_REFRESH, 
  ERROR_MESSAGES,
  API_ENDPOINTS,
  ALL_ACTIONS,
  USER_ROLES,
  ENV_CONFIG,
  DEFAULTS,
  TIMING
} from '../../constants/authConstants';

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
  
  // Token refresh tracking
  const refreshTimerRef = useRef(null);
  const isRefreshingRef = useRef(false);

  const fetchPermissions = useCallback(async (token) => {
    // Check if staging bypass is enabled (configurable via env var)
    // WARNING: This is a temporary workaround for staging environments
    // Should be disabled in production (REACT_APP_ENABLE_STAGING_BYPASS=false)
    const isStaging = REACT_APP_ENVIRONMENT.toLowerCase() === 'staging';
    const shouldBypass = ENV_CONFIG.ENABLE_STAGING_BYPASS && isStaging;
    
    if (shouldBypass) {
      // Return mock permissions for staging (only if explicitly enabled)
      const mockPermissions = {
        role: 'admin',
        permissions: []
      };
      setPermissions(mockPermissions);
      permissionsTokenRef.current = token;
      return { success: true, data: mockPermissions };
    }

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
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.PERMISSION_BY_ROLE}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        setPermissions(null);
        // Reset permission tracking refs on failure
        fetchingPermissionsRef.current = false;
        permissionsTokenRef.current = null;
        const isUnauthorized = response.status === 401;
        return { 
          success: false, 
          status: response.status, 
          isUnauthorized,
          error: isUnauthorized ? ERROR_MESSAGES.SESSION_EXPIRED : ERROR_MESSAGES.GENERIC_ERROR
        };
      }

      const result = await response.json();
      if (result.error) {
        setPermissions(null);
        return { success: false, error: result.error || ERROR_MESSAGES.GENERIC_ERROR };
      }

      setPermissions(result);
      return { success: true, data: result };
    } catch (error) {
      setPermissions(null);
      // Reset permission tracking refs on error
      fetchingPermissionsRef.current = false;
      permissionsTokenRef.current = null;
      // Check if it's a network error
      const isNetworkError = !error.response && error.message.includes('fetch');
      return { 
        success: false, 
        error: isNetworkError ? ERROR_MESSAGES.NETWORK_ERROR : (error.message || ERROR_MESSAGES.GENERIC_ERROR)
      };
    } finally {
      fetchingPermissionsRef.current = false;
      setLoadingPermissions(false);
    }
  }, [permissions]);

  const hasPermission = useCallback((service, screenType, action) => {
    // Staging bypass (only if explicitly enabled via config)
    const isStaging = REACT_APP_ENVIRONMENT.toLowerCase() === 'staging';
    if (ENV_CONFIG.ENABLE_STAGING_BYPASS && isStaging) {
      return true;
    }

    // Super admin has all permissions
    if (permissions?.role === 'super_admin') {
      return true;
    }

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
    // Staging bypass (only if explicitly enabled via config)
    const isStaging = REACT_APP_ENVIRONMENT.toLowerCase() === 'staging';
    if (ENV_CONFIG.ENABLE_STAGING_BYPASS && isStaging) {
      return true;
    }

    // Note: Removed super_admin bypass for menu visibility
    // Menu items should respect database permissions even for super_admin
    // Backend middleware still bypasses permission checks for super_admin on API access

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

    // Check if "view" action is in allowedActions
    // Menu items should only be visible if user has view permission
    return screenPermission.allowedActions.includes('view');
  }, [permissions]);

  const getAllowedActions = useCallback((service, screenType) => {
    // Super admin has all actions
    if (permissions?.role === USER_ROLES.SUPER_ADMIN) {
      return ALL_ACTIONS;
    }

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

  // Token refresh function
  const refreshToken = useCallback(async () => {
    if (isRefreshingRef.current) {
      return; // Already refreshing
    }

    const storedUser = JSON.parse(localStorage.getItem(STORAGE_KEYS.USER));
    if (!storedUser || !storedUser.refresh_token) {
      return;
    }

    isRefreshingRef.current = true;

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.REFRESH_TOKEN}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refresh_token: storedUser.refresh_token }),
      });

      if (!response.ok) {
        // Refresh token expired, logout
        console.warn('Token refresh failed: Refresh token expired or invalid');
        logout();
        return;
      }

      const data = await response.json();
      const { token, refresh_token } = data;

      if (!token) {
        console.error('Token refresh failed: No token in response');
        logout();
        return;
      }

      // Update stored user with new tokens
      const updatedUser = {
        ...storedUser,
        token,
        refresh_token,
      };
      setUser(updatedUser);
      localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(updatedUser));
      localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, token);

      // Schedule next refresh
      scheduleTokenRefresh();
    } catch (error) {
      console.error('Token refresh failed:', error);
      // Check if it's a network error
      if (!error.response && error.message.includes('fetch')) {
        console.warn('Network error during token refresh. Will retry on next request.');
        // Don't logout on network errors, allow retry
        isRefreshingRef.current = false;
        return;
      }
      logout();
    } finally {
      isRefreshingRef.current = false;
    }
  }, []);

  // Schedule token refresh
  const scheduleTokenRefresh = useCallback(() => {
    // Clear existing timer
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
    }

    // Refresh token before expiry to ensure seamless user experience
    // Check token expiry from JWT
    const storedUser = JSON.parse(localStorage.getItem(STORAGE_KEYS.USER));
    if (!storedUser || !storedUser.token) {
      return;
    }

    try {
      const tokenParts = storedUser.token.split('.');
      if (tokenParts.length === 3) {
        const payload = JSON.parse(atob(tokenParts[1]));
        const expiryTime = payload.exp * TIMING.TOKEN_EXPIRY_MULTIPLIER; // Convert JWT exp (seconds) to milliseconds
        const now = Date.now();
        const timeUntilExpiry = expiryTime - now;
        const refreshTime = timeUntilExpiry - TOKEN_REFRESH.REFRESH_BEFORE_EXPIRY;

        if (refreshTime > 0) {
          refreshTimerRef.current = setTimeout(() => {
            refreshToken();
          }, refreshTime);
        } else {
          // Token expires soon, refresh immediately
          refreshToken();
        }
      }
    } catch (error) {
      console.error('Error parsing token for refresh scheduling:', error);
    }
  }, [refreshToken]);

  useEffect(() => {
    const initializeAuth = async () => {
      try {
        const storedUser = JSON.parse(localStorage.getItem(STORAGE_KEYS.USER));
        if (storedUser && storedUser.token) {
          setUser(storedUser);
          
          // Schedule token refresh
          scheduleTokenRefresh();
          
          // Try to fetch permissions with the stored token
          const permissionsResult = await fetchPermissions(storedUser.token);
          
          // Only logout if it's specifically a 401 unauthorized response (expired token)
          if (!permissionsResult.success && permissionsResult.isUnauthorized) {
            // Try to refresh token first
            if (storedUser.refresh_token) {
              await refreshToken();
              // Retry permissions fetch after refresh
              const updatedUser = JSON.parse(localStorage.getItem(STORAGE_KEYS.USER));
              if (updatedUser?.token) {
                await fetchPermissions(updatedUser.token);
              }
            } else {
              // No refresh token, logout
              setUser(null);
              setPermissions(null);
              localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
              localStorage.removeItem(STORAGE_KEYS.USER);
              
              // Reset permission tracking refs
              fetchingPermissionsRef.current = false;
              permissionsTokenRef.current = null;
            }
          }
        }
      } catch (error) {
        console.error('Error initializing auth:', error);
        // Clear potentially corrupted data
        localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
        localStorage.removeItem(STORAGE_KEYS.USER);
      } finally {
        setLoading(false);
      }
    };

    initializeAuth();

    // Cleanup on unmount
    return () => {
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
      }
    };
  }, []);

  const login = useCallback(async (email, role, token, refreshToken = null, authProvider = DEFAULTS.AUTH_PROVIDER) => {
    if (!token) {
      console.error('Login failed: No token provided');
      throw new Error(ERROR_MESSAGES.GENERIC_ERROR);
    }

    const userData = { 
      email, 
      role, 
      token, 
      refresh_token: refreshToken,
      auth_provider: authProvider 
    };
    setUser(userData);
    localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(userData));
    localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, token);
    
    // Schedule token refresh if refresh token is available
    if (refreshToken) {
      scheduleTokenRefresh();
    }
    
    // Only fetch permissions if we don't have them or if token changed
    if (!permissions || permissionsTokenRef.current !== token) {
      await fetchPermissions(token);
    }
  }, [fetchPermissions, permissions, scheduleTokenRefresh]);

  const logout = useCallback(async () => {
    try {
      const token = user?.token;

      if (token) {
        try {
          const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.LOGOUT}`, {
            method: 'POST',
            headers: {
              'Authorization': `Bearer ${token}`,
            },
          });

          if (!response.ok) {
            console.warn('Logout API call failed, but continuing with local logout');
          }
        } catch (error) {
          console.warn('Error calling logout API, but continuing with local logout:', error);
        }
      }
    } catch (error) {
      console.error('Error during logout:', error);
    } finally {
      setUser(null);
      setPermissions(null);
      localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
      localStorage.removeItem(STORAGE_KEYS.USER);
      localStorage.removeItem(STORAGE_KEYS.SESSION_ID);
      
      // Clear refresh timer
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
        refreshTimerRef.current = null;
      }
      
      // Reset permission tracking refs
      fetchingPermissionsRef.current = false;
      permissionsTokenRef.current = null;
      isRefreshingRef.current = false;
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
    fetchPermissions,
    refreshToken
  }), [user, permissions, isAuthenticated, loading, loadingPermissions, login, logout, hasPermission, hasScreenAccess, getAllowedActions, getUserRole, fetchPermissions, refreshToken]);

  if (loading) {
    return (
      <div 
        style={{ 
          display: 'flex', 
          justifyContent: 'center', 
          alignItems: 'center', 
          height: '100vh',
          flexDirection: 'column',
          gap: '16px'
        }}
        role="status"
        aria-live="polite"
        aria-label="Initializing authentication"
      >
        <div className="spinner-border" role="status" style={{ width: '3rem', height: '3rem' }}>
          <span className="visually-hidden">Loading...</span>
        </div>
        <p style={{ color: '#666', fontSize: '14px' }}>Initializing authentication...</p>
      </div>
    );
  }

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);