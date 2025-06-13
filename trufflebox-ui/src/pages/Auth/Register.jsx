import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import './Login.css';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';

import * as URL_CONSTANTS from '../../config';

const Register = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [error, setError] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const navigate = useNavigate();

  const handlePasswordChange = (e) => {
    setPassword(e.target.value);
    if (passwordError) {
      setPasswordError(''); // Clear password errors when user starts typing
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setPasswordError('');

    // Validate all fields are filled
    if (!firstName.trim()) {
      setError('First Name is required');
      return;
    }
    
    if (!lastName.trim()) {
      setError('Last Name is required');
      return;
    }
    
    if (!email.trim()) {
      setError('Email is required');
      return;
    }
    
    if (!password.trim()) {
      setPasswordError('Password is required');
      return;
    }

    // Comprehensive password validation
    const passwordRules = {
      minLength: password.length >= 8,
      hasUppercase: /[A-Z]/.test(password),
      hasLowercase: /[a-z]/.test(password),
      hasNumber: /\d/.test(password),
      hasSpecialChar: /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(password),
      noSpaces: !/\s/.test(password),
      noCommonPatterns: !/^(password|123456|qwerty|abc123|admin|user)$/i.test(password)
    };

    const failedRules = [];
    if (!passwordRules.minLength) failedRules.push('At least 8 characters');
    if (!passwordRules.hasUppercase) failedRules.push('One uppercase letter (A-Z)');
    if (!passwordRules.hasLowercase) failedRules.push('One lowercase letter (a-z)');
    if (!passwordRules.hasNumber) failedRules.push('One number (0-9)');
    if (!passwordRules.hasSpecialChar) failedRules.push('One special character (!@#$%^&*...)');
    if (!passwordRules.noSpaces) failedRules.push('No spaces allowed');
    if (!passwordRules.noCommonPatterns) failedRules.push('Not a common password');

    if (failedRules.length > 0) {
      setPasswordError(failedRules);
      return;
    }

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          first_name: firstName,
          last_name: lastName,
          email,
          password,
        }),
      });

      if (response.ok) {
        alert('Registration successful');
        navigate('/login'); // Redirect to login after successful registration
      } else {
        const data = await response.json();
        const errorMessage = data.error || data.message || 'Registration failed';
        
        // Check for duplicate email error
        if (errorMessage.includes('Duplicate entry') && errorMessage.includes('for key \'users.email\'')) {
          // Extract email from the error message
          const emailMatch = errorMessage.match(/'([^']+)'/);
          const email = emailMatch ? emailMatch[1] : 'This email';
          setError(`${email} is already registered. Please use a different email or try logging in.`);
        } else {
          setError(errorMessage);
        }
      }
    } catch (err) {
      setError('An error occurred. Please try again.');
    }
  };

  return (
    <div className="login-container">
      <h2>Register</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          placeholder="First Name"
          value={firstName}
          onChange={(e) => setFirstName(e.target.value)}
          required
        />
        <input
          type="text"
          placeholder="Last Name"
          value={lastName}
          onChange={(e) => setLastName(e.target.value)}
          required
        />
        <input
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <div className="password-field">
          <input
            type={showPassword ? "text" : "password"}
            placeholder="Password"
            value={password}
            onChange={handlePasswordChange}
            required
          />
          <span 
            className="password-toggle-icon"
            onClick={() => setShowPassword(!showPassword)}
          >
            {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
          </span>
        </div>
        {passwordError && (
          <div className="password-errors" style={{ 
            marginTop: '12px',
            padding: '12px 16px',
            backgroundColor: '#fef2f2',
            border: '1px solid #fecaca',
            borderRadius: '8px',
            boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)'
          }}>
            {Array.isArray(passwordError) ? (
              <div style={{ color: '#dc2626', fontSize: '14px', lineHeight: '1.5', textAlign: 'left' }}>
                <div style={{ 
                  fontWeight: '600', 
                  marginBottom: '8px',
                  display: 'flex',
                  alignItems: 'flex-start',
                  justifyContent: 'flex-start',
                  gap: '6px',
                  textAlign: 'left'
                }}>
                  <span style={{ fontSize: '16px' }}>⚠️</span>
                  Password should contain:
                </div>
                <div style={{ fontSize: '13px', textAlign: 'left' }}>
                  {passwordError.map((rule, index) => (
                    <div key={index} style={{ 
                      marginBottom: '4px',
                      paddingLeft: '12px',
                      textAlign: 'left'
                    }}>
                      {rule}
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div style={{ 
                color: '#dc2626', 
                fontSize: '14px',
                fontWeight: '500',
                display: 'flex',
                alignItems: 'center',
                gap: '6px'
              }}>
                <span style={{ fontSize: '16px' }}>⚠️</span>
                {passwordError}
              </div>
            )}
          </div>
        )}
        <button type="submit">Register</button>
      </form>
      {error && <p className="error">{error}</p>}
      <p>
        Already have an account? <a href="/login">Login here</a>.
      </p>
    </div>
  );
};

export default Register;
