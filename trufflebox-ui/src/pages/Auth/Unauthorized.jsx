import React from 'react';
import { useAuth } from './AuthContext';
import { useNavigate } from 'react-router-dom';

const Unauthorized = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleGoBack = () => {
    navigate(-1);
  };

  const handleGoHome = () => {
    navigate('/');
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
    <div style={{ 
      display: 'flex', 
      flexDirection: 'column', 
      alignItems: 'center', 
      justifyContent: 'center', 
      minHeight: '50vh',
      textAlign: 'center',
      padding: '20px'
    }}>
      <div style={{ 
        maxWidth: '600px',
        padding: '30px',
        border: '1px solid #e0e0e0',
        borderRadius: '8px',
        backgroundColor: '#f9f9f9'
      }}>
        <h1 style={{ color: '#d32f2f', marginBottom: '20px' }}>
          Access Denied
        </h1>
        
        <p style={{ fontSize: '18px', marginBottom: '20px', color: '#555' }}>
          You don't have permission to access this page or perform this action.
        </p>
        
        {user && (
          <div style={{ marginBottom: '30px', padding: '15px', backgroundColor: '#fff', borderRadius: '4px' }}>
            <p><strong>User:</strong> {user.email}</p>
            <p><strong>Current Role:</strong> {user.role}</p>
          </div>
        )}
        
        <p style={{ marginBottom: '30px', color: '#666' }}>
          If you believe this is an error, please contact your administrator to review your permissions.
        </p>
        
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'center', flexWrap: 'wrap' }}>
          <button 
            onClick={handleGoBack}
            style={{
              padding: '10px 20px',
              backgroundColor: '#1976d2',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '16px'
            }}
          >
            Go Back
          </button>
          
          <button 
            onClick={handleGoHome}
            style={{
              padding: '10px 20px',
              backgroundColor: '#388e3c',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '16px'
            }}
          >
            Go Home
          </button>
          
          <button 
            onClick={handleLogout}
            style={{
              padding: '10px 20px',
              backgroundColor: '#d32f2f',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '16px'
            }}
          >
            Logout
          </button>
        </div>
      </div>
    </div>
  );
};

export default Unauthorized;