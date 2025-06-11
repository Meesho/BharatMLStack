import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  TextField,
  Switch,
  FormControl,
  Select,
  MenuItem,
  Alert,
  Snackbar,
  Box,
  Skeleton,
  Tooltip,
  Fade,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import PersonIcon from '@mui/icons-material/Person';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import GroupIcon from '@mui/icons-material/Group';
import { useAuth } from '../Auth/AuthContext';
import * as URL_CONSTANTS from '../../config';

const UserManagement = () => {
  const [users, setUsers] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(true);
  const [updateStatus, setUpdateStatus] = useState({ message: '', type: '', show: false });
  const { user } = useAuth();

  // Fetch users
  useEffect(() => {
    const fetchUsers = async () => {
      try {
        setLoading(true);
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/users`, {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        });

      if (!response.ok) {
        throw new Error('Failed to fetch users');
      }

        const data = await response.json();
        setUsers(data);
      } catch (error) {
        setUpdateStatus({
          message: 'Failed to fetch users. Please try again.',
          type: 'error',
          show: true
        });
      } finally {
        setLoading(false);
      }
    };

    if (user?.token) {
      fetchUsers();
    }
  }, [user?.token]);

  // Handle role update
  const handleRoleUpdate = async (email, newRole) => {
    try {
      const userToUpdate = users.find(u => u.email === email);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/update-user`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
        body: JSON.stringify({
          email: email,
          is_active: userToUpdate.is_active,
          role: newRole
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to update user role');
      }

      setUsers(prevUsers =>
        prevUsers.map(u =>
          u.email === email ? { ...u, role: newRole } : u
        )
      );

      setUpdateStatus({
        message: 'User role updated successfully!',
        type: 'success',
        show: true
      });
    } catch (error) {
      setUpdateStatus({
        message: 'Failed to update user role. Please try again.',
        type: 'error',
        show: true
      });
    }
  };

  // Handle status update
  const handleStatusUpdate = async (email, newStatus) => {
    try {
      const userToUpdate = users.find(u => u.email === email);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/update-user`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
        body: JSON.stringify({
          email: email,
          is_active: newStatus,
          role: userToUpdate.role
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to update user status');
      }

      setUsers(prevUsers =>
        prevUsers.map(u =>
          u.email === email ? { ...u, is_active: newStatus } : u
        )
      );

      setUpdateStatus({
        message: 'User status updated successfully!',
        type: 'success',
        show: true
      });
    } catch (error) {
      setUpdateStatus({
        message: 'Failed to update user status. Please try again.',
        type: 'error',
        show: true
      });
    }
  };

  // Filter users based on search term
  const filteredUsers = users.filter(user => 
    `${user.first_name} ${user.last_name}`.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.role.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleCloseSnackbar = () => {
    setUpdateStatus(prev => ({ ...prev, show: false }));
  };

  return (
    <Box sx={{ p: 3, backgroundColor: '#f8f9fa', minHeight: '100vh' }}>
      <Box sx={{ mb: 4 }}>
        <Typography
          variant="h4"
          sx={{
            fontWeight: 'bold',
            color: '#24031e',
            mb: 1,
            display: 'flex',
            alignItems: 'center',
            gap: 2
          }}
        >
          <GroupIcon sx={{ fontSize: 32, color: '#450839' }} />
          User Management
        </Typography>
        <Typography variant="body1" color="text.secondary">
          Manage user roles and access permissions
        </Typography>
            </Box>

      <Card
        style={{
          marginTop: '20px',
          borderRadius: '8px',
          boxShadow: '0 4px 10px rgba(0, 0, 0, 0.1)',
          backgroundColor: '#ffffff',
          width: '100%',
        }}
      >
        <CardContent>
          {/* Search Input */}
          <div style={{ marginBottom: '20px' }}>
            <TextField
              placeholder="Search Users by Name, Email, or Role"
              variant="outlined"
              fullWidth
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
                ),
              }}
            />
          </div>

          <TableContainer component={Paper} style={{ maxHeight: '100%' }}>
            <Table stickyHeader>
              <TableHead>
                <TableRow>
                  <TableCell 
                    sx={{ 
                      backgroundColor: '#450839', 
                      color: 'white', 
                      fontWeight: 'bold',
                      fontSize: '0.875rem'
                    }}
                  >
                    User
                  </TableCell>
                  <TableCell 
                    sx={{ 
                      backgroundColor: '#450839', 
                      color: 'white', 
                      fontWeight: 'bold',
                      fontSize: '0.875rem'
                    }}
                  >
                    Email
                  </TableCell>
                  <TableCell 
                    sx={{ 
                      backgroundColor: '#450839', 
                      color: 'white', 
                      fontWeight: 'bold',
                      fontSize: '0.875rem'
                    }}
                  >
                    Role
                  </TableCell>
                  <TableCell 
                    sx={{ 
                      backgroundColor: '#450839', 
                      color: 'white', 
                      fontWeight: 'bold',
                      fontSize: '0.875rem'
                    }}
                  >
                    Status
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {loading ? (
                  // Loading Skeletons
                  Array.from({ length: 5 }).map((_, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <Box display="flex" alignItems="center" gap={2}>
                          <Skeleton variant="circular" width={40} height={40} />
                          <Skeleton variant="text" width={120} height={20} />
                        </Box>
                      </TableCell>
                      <TableCell><Skeleton variant="text" width={180} height={20} /></TableCell>
                      <TableCell><Skeleton variant="text" width={80} height={20} /></TableCell>
                      <TableCell><Skeleton variant="text" width={100} height={20} /></TableCell>
                    </TableRow>
                  ))
                ) : filteredUsers.length > 0 ? (
                  filteredUsers.map((userData, index) => (
                    <Fade in={true} timeout={300 + index * 100} key={index}>
                      <TableRow 
                        sx={{ 
                          '&:hover': { 
                            backgroundColor: '#f8f9fa',
                            cursor: 'pointer'
                          },
                          transition: 'background-color 0.2s ease'
                        }}
                      >
                        <TableCell>
                          <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                            {`${userData.first_name || ''} ${userData.last_name || ''}`.trim() || 'Unknown User'}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" color="text.secondary">
                            {userData.email || '-'}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <FormControl size="small" sx={{ minWidth: 120 }}>
                            <Select
                              value={userData.role || 'user'}
                              onChange={(e) => handleRoleUpdate(userData.email, e.target.value)}
                              variant="outlined"
                              sx={{
                                '& .MuiSelect-select': {
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 1,
                                },
                              }}
                            >
                              <MenuItem value="user">
                                <Box display="flex" alignItems="center" gap={1}>
                                  <PersonIcon fontSize="small" />
                                  User
                                </Box>
                              </MenuItem>
                              <MenuItem value="admin">
                                <Box display="flex" alignItems="center" gap={1}>
                                  <AdminPanelSettingsIcon fontSize="small" />
                                  Admin
                                </Box>
                              </MenuItem>
                            </Select>
                          </FormControl>
                        </TableCell>
                        <TableCell>
                          <Box display="flex" alignItems="center" gap={2}>
                            <Tooltip title={userData.is_active ? 'Click to deactivate' : 'Click to activate'}>
                              <Switch
                                checked={userData.is_active || false}
                                onChange={(e) => handleStatusUpdate(userData.email, e.target.checked)}
                                size="medium"
                                color='success'
                              />
                            </Tooltip>
                          </Box>
                        </TableCell>
                      </TableRow>
                    </Fade>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={4} align="center" sx={{ py: 8 }}>
                      <Box textAlign="center">
                        <GroupIcon sx={{ fontSize: 48, color: '#ccc', mb: 2 }} />
                        <Typography variant="h6" color="text.secondary" gutterBottom>
                          {searchTerm ? 'No users found' : 'No users available'}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {searchTerm 
                            ? 'Try adjusting your search criteria' 
                            : 'Users will appear here once they are added to the system'
                          }
                        </Typography>
                      </Box>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* Success/Error Messages */}
      <Snackbar
        open={updateStatus.show}
        autoHideDuration={4000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Alert 
          onClose={handleCloseSnackbar} 
          severity={updateStatus.type}
          sx={{ width: '100%' }}
          elevation={6}
          variant="filled"
        >
          {updateStatus.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default UserManagement; 