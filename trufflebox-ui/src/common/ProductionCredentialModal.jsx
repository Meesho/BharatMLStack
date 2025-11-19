import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  IconButton,
  Typography,
  Alert,
  CircularProgress,
  InputAdornment
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import SecurityIcon from '@mui/icons-material/Security';
import { jwtDecode } from 'jwt-decode';

import * as URL_CONSTANTS from '../config';

const ProductionCredentialModal = ({ 
  open, 
  onClose, 
  onSuccess, 
  title = "Production Credential Verification",
  description = "Please enter your production credentials to proceed with promotion."
}) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        throw new Error('Invalid production credentials');
      }

      const data = await response.json();
      const { email: userEmail, role, token } = data;
      
      if (token) {
        const decodedToken = jwtDecode(token);
        
        try {
          const sessionResponse = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/track-session`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ 
              email: userEmail,
              userId: decodedToken.sub || decodedToken.user_id,
              role: decodedToken.role || role,
              sessionStartTime: new Date().toISOString(),
              userAgent: navigator.userAgent
            }),
          });
          
          if (sessionResponse.ok) {
            const sessionData = await sessionResponse.json();
            // console.log('Production session tracked:', sessionData.sessionId);
          }
        } catch (sessionError) {
          console.error('Session tracking failed, but proceeding with promotion:', sessionError);
        }
        
        onSuccess({
          token,
          email: userEmail,
          role,
          decodedToken
        });
        
        setEmail('');
        setPassword('');
        setShowPassword(false);
      } else {
        throw new Error('No token received from production authentication');
      }
    } catch (err) {
      setError(err.message || 'Authentication failed');
    } finally {
      setIsLoading(false);
    }
  };

  const handleClose = () => {
    setEmail('');
    setPassword('');
    setShowPassword(false);
    setError('');
    onClose();
  };

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          boxShadow: '0 8px 32px rgba(0,0,0,0.12)'
        }
      }}
    >
      <DialogTitle
        sx={{
          bgcolor: '#450839',
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          pb: 2
        }}
      >
        <Box display="flex" alignItems="center" gap={1}>
          <SecurityIcon />
          <Typography variant="h6">{title}</Typography>
        </Box>
        <IconButton onClick={handleClose} size="small" sx={{ color: 'white' }}>
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <form onSubmit={handleSubmit}>
        <DialogContent sx={{ pt: 3, pb: 2 }}>
          <Alert severity="info" sx={{ mb: 3 }}>
            <Typography variant="body2">
              {description}
            </Typography>
          </Alert>

          <TextField
            fullWidth
            label="Production Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            margin="normal"
            variant="outlined"
            disabled={isLoading}
            sx={{
              '& .MuiOutlinedInput-root': {
                '&.Mui-focused fieldset': {
                  borderColor: '#450839',
                },
              },
              '& .MuiInputLabel-root.Mui-focused': {
                color: '#450839',
              },
            }}
          />

          <TextField
            fullWidth
            label="Production Password"
            type={showPassword ? "text" : "password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            margin="normal"
            variant="outlined"
            disabled={isLoading}
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    onClick={() => setShowPassword(!showPassword)}
                    edge="end"
                    disabled={isLoading}
                  >
                    {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
                  </IconButton>
                </InputAdornment>
              ),
            }}
            sx={{
              '& .MuiOutlinedInput-root': {
                '&.Mui-focused fieldset': {
                  borderColor: '#450839',
                },
              },
              '& .MuiInputLabel-root.Mui-focused': {
                color: '#450839',
              },
            }}
          />

          {error && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {error}
            </Alert>
          )}
        </DialogContent>

        <DialogActions sx={{ p: 3, pt: 1 }}>
          <Button 
            onClick={handleClose}
            disabled={isLoading}
            variant="outlined"
            sx={{ 
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                borderColor: '#450839',
                bgcolor: 'rgba(69, 8, 57, 0.04)'
              }
            }}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            disabled={isLoading || !email || !password}
            variant="contained"
            sx={{ 
              bgcolor: '#450839',
              '&:hover': {
                bgcolor: '#5a0a4a'
              },
              minWidth: 120
            }}
          >
            {isLoading ? (
              <>
                <CircularProgress size={20} color="inherit" sx={{ mr: 1 }} />
                Verifying...
              </>
            ) : (
              'Authenticate'
            )}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default ProductionCredentialModal; 