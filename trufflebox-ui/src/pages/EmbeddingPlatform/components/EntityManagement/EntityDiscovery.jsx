import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  TextField,
  CircularProgress,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Snackbar,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const EntityDiscovery = () => {
  const [entities, setEntities] = useState([]);
  const [filteredEntities, setFilteredEntities] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  const fetchEntities = useCallback(async () => {
    try {
      setLoading(true);
      const response = await embeddingPlatformAPI.getEntities();
      
      if (response.entities) {
        const entitiesData = response.entities;
        setEntities(entitiesData);
        setFilteredEntities(entitiesData);
      } else {
        setEntities([]);
        setFilteredEntities([]);
      }
    } catch (error) {
      console.error('Error fetching entities:', error);
      setError('Failed to load entities. Please refresh the page.');
      showNotification('Failed to load entities. Please refresh the page.', 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchEntities();
    const handleEntityUpdate = () => {
      setTimeout(() => fetchEntities(), 1000);
    };
    window.addEventListener('entityApprovalUpdate', handleEntityUpdate);
    return () => {
      window.removeEventListener('entityApprovalUpdate', handleEntityUpdate);
    };
  }, [fetchEntities]);

  useEffect(() => {
    const filtered = entities.filter(entity => {
      const lowerCaseQuery = searchQuery.toLowerCase();
      return (
        (entity.name && entity.name.toLowerCase().includes(lowerCaseQuery)) ||
        (entity.store_id?.toString().includes(lowerCaseQuery)) ||
        (entity.store_db && entity.store_db.toLowerCase().includes(lowerCaseQuery))
      );
    });
    setFilteredEntities(filtered);
  }, [searchQuery, entities]);

  const showNotification = (message, severity) => {
    setNotification({ open: true, message, severity });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({ ...prev, open: false }));
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Paper elevation={0} sx={{ width: '100%', height: '100vh', padding: '2rem', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h6">Entity Discovery</Typography>
          <Typography variant="body1" color="text.secondary">
            Explore approved logical entities
          </Typography>
        </Box>
      </Box>

      {/* Search */}
      <Box 
        sx={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          gap: '1rem',
          mb: 2 
        }}
      >
        <TextField
          label="Search Entities"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
          size="small"
          fullWidth
        />
      </Box>

      {/* Entities Table */}
      <TableContainer 
        component={Paper} 
        elevation={3} 
        sx={{ 
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 800,
            borderCollapse: 'separate',
            borderSpacing: 0,
          },
          '& .MuiTableHead-root': {
            position: 'sticky',
            top: 0,
            zIndex: 1,
            backgroundColor: '#E6EBF2',
          }
        }}
      >
        <Table stickyHeader>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Entity Name
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Store ID
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredEntities.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {entities.length === 0 ? 'No approved entities available' : 'No entities match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredEntities.map((entity, index) => (
                <TableRow key={entity.name || index} hover>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {entity.name || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {entity.store_id || 'N/A'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleCloseNotification}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={handleCloseNotification}
          severity={notification.severity}
          sx={{ width: '100%' }}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </Paper>
  );
};

export default EntityDiscovery;
