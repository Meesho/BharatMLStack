import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import './Login.css';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { jwtDecode } from 'jwt-decode';
import { CircularProgress } from '@mui/material';

import * as URL_CONSTANTS from '../../config';


const Login = () => {
  const [emailId, setEmailId] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const { login } = useAuth(); // Get login method from AuthContext
  const navigate = useNavigate();

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
        console.log('Invalid credentials');
      }

      const data = await response.json();
      const { email, role, token } = data;
      
      if (token) {
        // Decode the JWT token to get additional information
        const decodedToken = jwtDecode(token);
        
        // Store token and user info
        login(email, role, token);
        
        // Second API call - track session with JWT token
        const sessionResponse = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/track-session`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          },
          body: JSON.stringify({ 
            email,
            userId: decodedToken.sub || decodedToken.user_id, // Extract user ID from token
            role: decodedToken.role || role, // Use role from token or response
            sessionStartTime: new Date().toISOString(),
            userAgent: navigator.userAgent
          }),
        });
        
        if (!sessionResponse.ok) {
          console.error('Session tracking failed, but proceeding with login');
        } else {
          const sessionData = await sessionResponse.json();
          // You can store the session ID if needed
          localStorage.setItem('sessionId', sessionData.sessionId);
        }
        
        // Navigate to dashboard regardless of session tracking success
        navigate('/feature-discovery');
      } else {
        console.log('Token not received');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="login-container">
      <h2>Login To TruffleBox</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="email"
          placeholder="Email"
          value={emailId}
          onChange={(e) => setEmailId(e.target.value)}
          required
        />
        <div className="password-field">
          <input
            type={showPassword ? "text" : "password"}
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <span 
            className="password-toggle-icon"
            onClick={() => setShowPassword(!showPassword)}
          >
            {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
          </span>
        </div>
        <button type="submit" disabled={isLoading} className="login-button">
          {isLoading ? (
            <CircularProgress size={24} color="inherit" sx={{ marginRight: '8px' }} />
          ) : (
            'Log in'
          )}
        </button>
      </form>
      {error && <p className="error">{error}</p>}
      <p>
        Don't have an account? <a href="/register">Register here</a>.
      </p>
    </div>
  );
};

export default Login;
