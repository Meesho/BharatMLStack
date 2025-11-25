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
  Chip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const ModelDiscovery = () => {
  const [models, setModels] = useState([]);
  const [filteredModels, setFilteredModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    fetchModels();
  }, []);

  useEffect(() => {
    // Filter models based on search query
    if (!searchQuery) {
      setFilteredModels(models);
    } else {
      const filtered = models.filter(model => 
        (model.model && model.model.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (model.entity && model.entity.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (model.model_type && model.model_type.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (model.job_frequency && model.job_frequency.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (model.topic_name && model.topic_name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (model.status && model.status.toLowerCase().includes(searchQuery.toLowerCase()))
      );
      setFilteredModels(filtered);
    }
  }, [models, searchQuery]);

  const fetchModels = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getModels();
      
      if (response.models) {
        const modelsData = response.models || [];
        setModels(modelsData);
      } else {
        setModels([]);
      }
    } catch (error) {
      console.error('Error fetching models:', error);
      setError('Failed to load models. Please refresh the page.');
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
          <Typography variant="h6">Model Discovery</Typography>
          <Typography variant="body1" color="text.secondary">
            Explore approved ML models in the Embedding Platform
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
          label="Search Models"
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

      {/* Models Table */}
      <TableContainer 
        component={Paper} 
        elevation={3} 
        sx={{ 
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1000,
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
                Model Name
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Entity
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Model Type
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Vector Dimension
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Job Frequency
              </TableCell>
              <TableCell 
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                }}
              >
                Status
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredModels.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {models.length === 0 ? 'No approved models available' : 'No models match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredModels.map((model, index) => (
                <TableRow key={model.model || index} hover>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {model.model || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {model.entity || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    <Chip 
                      label={model.model_type || 'N/A'}
                      size="small"
                      sx={{ 
                        backgroundColor: (model.model_type || '').toLowerCase() === 'delta' ? '#e8f5e8' : '#fff3e0',
                        color: (model.model_type || '').toLowerCase() === 'delta' ? '#2e7d32' : '#f57c00',
                        fontWeight: 600
                      }}
                    />
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {model.vector_dimension || model.model_config?.vector_dimension || 'N/A'}
                  </TableCell>
                  <TableCell 
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {model.job_frequency || 'N/A'}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={(model.status || 'ACTIVE').toUpperCase()}
                      size="small"
                      sx={{
                        backgroundColor: 
                          (model.status || 'active').toLowerCase() === 'active' ? '#E7F6E7' : '#EEEEEE',
                        color: 
                          (model.status || 'active').toLowerCase() === 'active' ? '#2E7D32' : '#616161',
                        fontWeight: 'bold',
                        minWidth: '80px',
                      }}
                    />
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

export default ModelDiscovery;

