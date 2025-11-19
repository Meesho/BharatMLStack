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
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ExperimentIcon from '@mui/icons-material/Science';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const VariantDiscovery = () => {
  const [variants, setVariants] = useState([]);
  const [filteredVariants, setFilteredVariants] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [entityFilter, setEntityFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [error, setError] = useState('');

  const uniqueEntities = [...new Set(variants.map(v => v.entity))].sort();
  const uniqueModels = [...new Set(variants.map(v => v.model))].sort();
  const uniqueStatuses = [...new Set(variants.map(v => v.status))].sort();

  useEffect(() => {
    fetchVariants();
  }, []);

  useEffect(() => {
    // Filter variants based on search query and filters
    let filtered = variants;

    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(variant =>
        variant.entity.toLowerCase().includes(searchLower) ||
        variant.model.toLowerCase().includes(searchLower) ||
        variant.variant.toLowerCase().includes(searchLower) ||
        variant.type.toLowerCase().includes(searchLower) ||
        variant.status.toLowerCase().includes(searchLower)
      );
    }

    if (entityFilter) {
      filtered = filtered.filter(variant => variant.entity === entityFilter);
    }

    if (modelFilter) {
      filtered = filtered.filter(variant => variant.model === modelFilter);
    }

    if (statusFilter) {
      filtered = filtered.filter(variant => variant.status === statusFilter);
    }

    setFilteredVariants(filtered);
  }, [variants, searchQuery, entityFilter, modelFilter, statusFilter]);

  const fetchVariants = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getVariants();

      if (response.data && response.data.variants) {
        setVariants(response.data.variants);
      } else {
        setVariants([]);
      }
    } catch (error) {
      console.error('Error fetching variants:', error);
      setError('Failed to load variants. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const getStatusChip = (status) => {
    const statusLower = (status || '').toLowerCase();
    let bgcolor = '#E7F6E7';
    let textColor = '#2E7D32';

    switch (statusLower) {
      case 'active':
        bgcolor = '#E7F6E7';
        textColor = '#2E7D32';
        break;
      case 'inactive':
        bgcolor = '#EEEEEE';
        textColor = '#616161';
        break;
      case 'pending':
        bgcolor = '#FFF8E1';
        textColor = '#F57C00';
        break;
      case 'error':
        bgcolor = '#FFEBEE';
        textColor = '#D32F2F';
        break;
      default:
        bgcolor = '#E7F6E7';
        textColor = '#2E7D32';
    }

    return (
      <Chip
        label={(status || 'ACTIVE').toUpperCase()}
        size="small"
        sx={{
          backgroundColor: bgcolor,
          color: textColor,
          fontWeight: 'bold',
          minWidth: '80px',
        }}
      />
    );
  };

  const getTypeChip = (type) => {
    return (
      <Chip
        label={type || 'EXPERIMENT'}
        size="small"
        sx={{
          backgroundColor: '#e3f2fd',
          color: '#1976d2',
          fontWeight: 600
        }}
      />
    );
  };

  const getEnabledChip = (enabled) => {
    return (
      <Chip
        label={enabled ? 'ENABLED' : 'DISABLED'}
        size="small"
        sx={{
          backgroundColor: enabled ? '#e8f5e8' : '#ffebee',
          color: enabled ? '#2e7d32' : '#d32f2f',
          fontWeight: 600
        }}
      />
    );
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
          <Typography variant="h6">Variant Discovery</Typography>
          <Typography variant="body1" color="text.secondary">
            Explore approved A/B testing variants in the Embedding Platform ({variants.length} total variants)
          </Typography>
        </Box>
      </Box>

      {/* Search and Filters */}
      <Box
        sx={{
          display: 'flex',
          gap: 2,
          mb: 2,
          flexWrap: 'wrap'
        }}
      >
        <TextField
          label="Search Variants"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
          size="small"
          sx={{ flexGrow: 1, minWidth: 300 }}
        />
        
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>Entity</InputLabel>
          <Select
            value={entityFilter}
            onChange={(e) => setEntityFilter(e.target.value)}
            label="Entity"
          >
            <MenuItem value="">All</MenuItem>
            {uniqueEntities.map((entity) => (
              <MenuItem key={entity} value={entity}>{entity}</MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>Model</InputLabel>
          <Select
            value={modelFilter}
            onChange={(e) => setModelFilter(e.target.value)}
            label="Model"
          >
            <MenuItem value="">All</MenuItem>
            {uniqueModels.map((model) => (
              <MenuItem key={model} value={model}>{model}</MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl size="small" sx={{ minWidth: 120 }}>
          <InputLabel>Status</InputLabel>
          <Select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            label="Status"
          >
            <MenuItem value="">All</MenuItem>
            {uniqueStatuses.map((status) => (
              <MenuItem key={status} value={status}>{status}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      {/* Variants Table */}
      <TableContainer
        component={Paper}
        elevation={3}
        sx={{
          marginTop: '1rem',
          maxHeight: 'calc(100vh - 200px)',
          overflowX: 'auto',
          overflowY: 'auto',
          '& .MuiTable-root': {
            minWidth: 1200,
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
                Model
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Variant
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Type
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Vector DB
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Rate Limit
              </TableCell>
              <TableCell
                sx={{
                  backgroundColor: '#E6EBF2',
                  fontWeight: 'bold',
                  color: '#031022',
                  borderRight: '1px solid rgba(224, 224, 224, 1)',
                }}
              >
                Enabled
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
            {filteredVariants.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {variants.length === 0 ? 'No variants available' : 'No variants match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredVariants.map((variant, index) => (
                <TableRow key={`${variant.entity}-${variant.model}-${variant.variant}` || index} hover>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    <Chip 
                      label={variant.entity || 'N/A'}
                      size="small"
                      sx={{ 
                        backgroundColor: '#e3f2fd',
                        color: '#1976d2',
                        fontWeight: 600
                      }}
                    />
                  </TableCell>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {variant.model || 'N/A'}
                  </TableCell>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <ExperimentIcon fontSize="small" sx={{ color: '#ff9800' }} />
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          fontFamily: 'monospace', 
                          backgroundColor: '#fff3e0',
                          padding: '2px 6px',
                          borderRadius: '4px',
                          fontSize: '0.875rem',
                          color: '#f57c00'
                        }}
                      >
                        {variant.variant || 'N/A'}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {getTypeChip(variant.type)}
                  </TableCell>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    <Chip
                      label={variant.vector_db_type || 'QDRANT'}
                      size="small"
                      sx={{
                        backgroundColor: '#f3e5f5',
                        color: '#7b1fa2',
                        fontWeight: 600
                      }}
                    />
                  </TableCell>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    <Typography variant="body2">
                      {variant.rate_limiter?.rate_limit || 0}/s
                    </Typography>
                  </TableCell>
                  <TableCell
                    sx={{
                      borderRight: '1px solid rgba(224, 224, 224, 1)',
                    }}
                  >
                    {getEnabledChip(variant.enabled)}
                  </TableCell>
                  <TableCell>
                    {getStatusChip(variant.status)}
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

export default VariantDiscovery;
