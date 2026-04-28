import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from './AuthContext';
import './Login.css';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { jwtDecode } from 'jwt-decode';
import { CircularProgress, Button } from '@mui/material';
import GoogleIcon from '@mui/icons-material/Google';

import * as URL_CONSTANTS from '../../config';
import ssoService from '../../services/ssoService';
import { 
  STORAGE_KEYS, 
  ERROR_MESSAGES, 
  API_ENDPOINTS, 
  APP_ROUTES, 
  JWT_CLAIMS,
  ENV_CONFIG 
} from '../../constants/authConstants';


const Login = () => {
  const [emailId, setEmailId] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [ssoStatus, setSsoStatus] = useState(null);
  const [isLoadingSSO, setIsLoadingSSO] = useState(false);
  const { login } = useAuth(); // Get login method from AuthContext
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // Fetch SSO status on component mount
  useEffect(() => {
    const fetchSSOStatus = async () => {
      try {
        const status = await ssoService.getSSOStatus();
        setSsoStatus(status);
      } catch (error) {
        console.error('Failed to fetch SSO status:', error);
        // Set default values if SSO status fetch fails (from config)
        setSsoStatus(ENV_CONFIG.DEFAULT_SSO_STATUS);
      }
    };

    fetchSSOStatus();

    // Handle OAuth callback
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    if (code && state) {
      handleGoogleCallback(code, state);
    }
  }, [searchParams]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      // First API call - authenticate user and get JWT token
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email: emailId, password }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || ERROR_MESSAGES.INVALID_CREDENTIALS);
      }

      const data = await response.json();
      const { email, role, token, refresh_token, auth_provider } = data;
      
      if (token) {
        // Decode the JWT token to get additional information
        const decodedToken = jwtDecode(token);
        
        // Store token, refresh token, and user info
        login(email, role, token, refresh_token, auth_provider);
        
        // Second API call - track session with JWT token (non-blocking)
        fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.TRACK_SESSION}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          },
          body: JSON.stringify({ 
            email,
            userId: decodedToken[JWT_CLAIMS.SUBJECT] || decodedToken[JWT_CLAIMS.USER_ID], // Extract user ID from token
            role: decodedToken[JWT_CLAIMS.ROLE] || role, // Use role from token or response
            sessionStartTime: new Date().toISOString(),
            userAgent: navigator.userAgent
          }),
        })
        .then(async (sessionResponse) => {
          if (sessionResponse.ok) {
            const sessionData = await sessionResponse.json();
            if (sessionData.sessionId) {
              localStorage.setItem(STORAGE_KEYS.SESSION_ID, sessionData.sessionId);
            }
          }
        })
        .catch((error) => {
          console.warn('Session tracking failed, but proceeding with login:', error);
        });
        
        // Navigate to default home page after successful login
        navigate(APP_ROUTES.HOME);
      } else {
        console.log('Token not received');
      }
    } catch (err) {
      // Provide user-friendly error messages
      let errorMessage = ERROR_MESSAGES.GENERIC_ERROR;
      if (err.message) {
        errorMessage = err.message;
      } else if (err instanceof TypeError && err.message.includes('fetch')) {
        errorMessage = ERROR_MESSAGES.NETWORK_ERROR;
      }
      setError(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleGoogleLogin = async () => {
    setIsLoadingSSO(true);
    setError('');

    try {
      const response = await ssoService.initiateGoogleOAuth();
      if (response.redirect_url) {
        // Store state in sessionStorage for validation
        sessionStorage.setItem(STORAGE_KEYS.OAUTH_STATE, response.state);
        // Redirect to Google OAuth
        window.location.href = response.redirect_url;
      }
    } catch (err) {
      setError(err.message || 'Failed to initiate Google login');
      setIsLoadingSSO(false);
    }
  };

  const handleGoogleCallback = async (code, state) => {
    setIsLoadingSSO(true);
    setError('');

    try {
      // Validate state
      const storedState = sessionStorage.getItem(STORAGE_KEYS.OAUTH_STATE);
      if (storedState !== state) {
        throw new Error(ERROR_MESSAGES.OAUTH_STATE_MISMATCH);
      }
      sessionStorage.removeItem(STORAGE_KEYS.OAUTH_STATE);

      const data = await ssoService.handleGoogleCallback(code, state);
      const { email, role, token, refresh_token, auth_provider, is_new_user } = data;

      if (token) {
        // Decode the JWT token
        const decodedToken = jwtDecode(token);

        // Store token, refresh token, and user info
        login(email, role, token, refresh_token, auth_provider);

        // Track session (non-blocking)
        fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.TRACK_SESSION}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({
            email,
            userId: decodedToken[JWT_CLAIMS.SUBJECT] || decodedToken[JWT_CLAIMS.USER_ID],
            role: decodedToken[JWT_CLAIMS.ROLE] || role,
            sessionStartTime: new Date().toISOString(),
            userAgent: navigator.userAgent,
          }),
        })
        .then(async (sessionResponse) => {
          if (sessionResponse.ok) {
            const sessionData = await sessionResponse.json();
            if (sessionData.sessionId) {
              localStorage.setItem(STORAGE_KEYS.SESSION_ID, sessionData.sessionId);
            }
          }
        })
        .catch((error) => {
          console.warn('Session tracking failed:', error);
        });

        // Navigate to default home page after successful login
        navigate(APP_ROUTES.HOME);
      }
    } catch (err) {
      // Provide user-friendly error messages
      let errorMessage = ERROR_MESSAGES.OAUTH_FAILED;
      if (err.message) {
        errorMessage = err.message;
      } else if (err instanceof TypeError && err.message.includes('fetch')) {
        errorMessage = ERROR_MESSAGES.NETWORK_ERROR;
      }
      setError(errorMessage);
      setIsLoadingSSO(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-header">
        <h2>Welcome to TruffleBox</h2>
        <p className="login-subtitle">Sign in to continue</p>
      </div>
      
      {/* Password Login Form - Only show if password is allowed */}
      {ssoStatus && ssoStatus.allow_password ? (
        <>
          <form onSubmit={handleSubmit} className="login-form">
            <div className="input-group">
              <input
                type="email"
                placeholder="Email address"
                value={emailId}
                onChange={(e) => setEmailId(e.target.value)}
                className="form-input"
                required
              />
            </div>
            <div className="input-group password-field">
              <input
                type={showPassword ? "text" : "password"}
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input"
                required
              />
              <span 
                className="password-toggle-icon"
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? "Hide password" : "Show password"}
              >
                {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
              </span>
            </div>
            <button 
              type="submit" 
              disabled={isLoading || isLoadingSSO} 
              className="login-button primary-button"
            >
              {isLoading ? (
                <>
                  <CircularProgress size={20} color="inherit" sx={{ marginRight: '8px' }} />
                  Signing in...
                </>
              ) : (
                'Sign in'
              )}
            </button>
          </form>

          {/* Divider */}
          {ssoStatus.sso_enabled && ssoStatus.providers.includes('google') && (
            <div className="divider">
              <span className="divider-line"></span>
              <span className="divider-text">or</span>
              <span className="divider-line"></span>
            </div>
          )}

          {/* Google SSO Button */}
          {ssoStatus.sso_enabled && ssoStatus.providers.includes('google') && (
            <div className="sso-section">
              <button
                type="button"
                onClick={handleGoogleLogin}
                disabled={isLoading || isLoadingSSO}
                className="google-signin-button"
              >
                {isLoadingSSO ? (
                  <>
                    <CircularProgress size={20} color="inherit" sx={{ marginRight: '8px' }} />
                    Connecting...
                  </>
                ) : (
                  <>
                    <GoogleIcon className="google-icon" />
                    <span>Continue with Google</span>
                  </>
                )}
              </button>
            </div>
          )}

          <div className="login-footer">
            <p>
              Don't have an account? <a href="/register" className="link">Create one</a>
            </p>
          </div>
        </>
      ) : (
        /* Google SSO Only - Show Google button */
        ssoStatus && ssoStatus.sso_enabled && ssoStatus.providers.includes('google') && (
          <div className="sso-only-section">
            <div className="sso-info">
              <div className="sso-icon-wrapper">
                <GoogleIcon className="sso-large-icon" />
              </div>
              <h3 className="sso-title">Sign in with Google</h3>
              <p className="sso-description">
                Use your Google account to securely access TruffleBox
              </p>
            </div>
            <button
              type="button"
              onClick={handleGoogleLogin}
              disabled={isLoading || isLoadingSSO}
              className="google-signin-button large"
            >
              {isLoadingSSO ? (
                <>
                  <CircularProgress size={20} color="inherit" sx={{ marginRight: '8px' }} />
                  Connecting to Google...
                </>
              ) : (
                <>
                  <GoogleIcon className="google-icon" />
                  <span>Continue with Google</span>
                </>
              )}
            </button>
          </div>
        )
      )}

      {error && (
        <div className="error-container">
          <p className="error-message">{error}</p>
        </div>
      )}
    </div>
  );
};

export default Login;
