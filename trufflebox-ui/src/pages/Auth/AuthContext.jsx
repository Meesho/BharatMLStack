import React, { createContext, useState, useContext, useEffect } from 'react';
import * as URL_CONSTANTS from '../../config';
const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true); // Add loading state
  const isAuthenticated = !!user;

  useEffect(() => {
    const storedUser = JSON.parse(localStorage.getItem('user'));
    if (storedUser) {
      setUser(storedUser);
    }
    setLoading(false); // Set loading to false once user data is checked
  }, []);

  const login = (email, role, token) => {
    const userData = { email, role, token };
    setUser(userData);
    localStorage.setItem('user', JSON.stringify(userData));
    localStorage.setItem('authToken', token);
  };

  const logout = async () => {
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
          throw new Error('Failed to log out');
        }

        setUser(null);
        localStorage.removeItem('authToken');
        localStorage.removeItem('user');
      } else {
        setUser(null);
        localStorage.removeItem('authToken');
        localStorage.removeItem('user');
      }
    } catch (error) {
      console.error('Error during logout:', error);
    }
  };

  if (loading) {
    return <div>Loading...</div>; // You can customize loading message or spinner
  }

  return (
    <AuthContext.Provider value={{ user, isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
