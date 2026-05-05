import React, { useState, useEffect } from 'react';
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
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const StoreDiscovery = () => {
  const [stores, setStores] = useState([]);
  const [filteredStores, setFilteredStores] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    fetchStores();
  }, []);

  useEffect(() => {
    // Filter stores based on search query
    if (!searchQuery) {
      setFilteredStores(stores);
    } else {
      const filtered = stores.filter(store => 
        (store.id && store.id.toString().toLowerCase().includes(searchQuery.toLowerCase())) ||
        (store.conf_id && store.conf_id.toString().toLowerCase().includes(searchQuery.toLowerCase())) ||
        (store.db && store.db.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (store.embeddings_table && store.embeddings_table.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (store.aggregator_table && store.aggregator_table.toLowerCase().includes(searchQuery.toLowerCase()))
      );
      setFilteredStores(filtered);
    }
  }, [stores, searchQuery]);

  const fetchStores = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getStores();
      
      // Get stores from /data/stores API
      if (response.stores) {
        const storesData = response.stores || [];
        setStores(storesData);
      } else {
        setStores([]);
      }
    } catch (error) {
      console.error('Error fetching stores from ETCD:', error);
      setError('Failed to load stores from ETCD. Please refresh the page.');
    } finally {
      setLoading(false);
    }
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
          <Typography variant="h6">Store Discovery</Typography>
          <Typography variant="body1" color="text.secondary">
            Explore stores from ETCD configuration
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
          label="Search Stores"
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

      {/* Stores Table */}
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
                Store ID
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Conf ID
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                DB Type
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Embeddings Table
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                }}
              >
                Aggregator Table
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredStores.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {stores.length === 0 ? 'No stores available in ETCD' : 'No stores match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredStores.map((store, index) => (
                <TableRow key={store.id || index} hover>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.id || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.conf_id || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.db || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {store.embeddings_table || 'N/A'}
                  </TableCell>
                  <TableCell>
                    {store.aggregator_table || 'N/A'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default StoreDiscovery;