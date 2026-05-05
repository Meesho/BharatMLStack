import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  TextField,
  CircularProgress,
  Alert,
  Paper,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import FilterListIcon from '@mui/icons-material/FilterList';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const FilterDiscovery = () => {
  const [filterGroups, setFilterGroups] = useState([]);
  const [filteredGroups, setFilteredGroups] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    fetchFilters();
  }, []);

  useEffect(() => {
    // Filter groups and filters based on search query
    if (!searchQuery) {
      setFilteredGroups(filterGroups);
    } else {
      const filtered = filterGroups
        .map(group => ({
          ...group,
          filters: group.filters.filter(filter =>
            filter.column_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            filter.filter_value.toLowerCase().includes(searchQuery.toLowerCase()) ||
            filter.default_value.toLowerCase().includes(searchQuery.toLowerCase()) ||
            group.entity.toLowerCase().includes(searchQuery.toLowerCase())
          )
        }))
        .filter(group => group.filters.length > 0);
      setFilteredGroups(filtered);
    }
  }, [filterGroups, searchQuery]);

  const normalizeFiltersResponse = (response) => {
    if (response.filter_groups && Array.isArray(response.filter_groups)) {
      return response.filter_groups;
    }
    const filtersByEntity = response.filters ?? response.Filters ?? {};
    if (typeof filtersByEntity !== 'object') return [];
    return Object.entries(filtersByEntity).map(([entity, filtersObj]) => ({
      entity,
      filters: typeof filtersObj === 'object' && filtersObj !== null
        ? Object.values(filtersObj).filter(f => f && (f.column_name != null || f.filter_value != null))
        : [],
    }));
  };

  const fetchFilters = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await embeddingPlatformAPI.getAllFilters();
      const groups = normalizeFiltersResponse(response);
      setFilterGroups(groups);
    } catch (error) {
      console.error('Error fetching filters:', error);
      setError('Failed to load filters. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const getTotalFilterCount = () => {
    return filterGroups.reduce((total, group) => total + group.filters.length, 0);
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
          <Typography variant="h6">Filter Discovery</Typography>
          <Typography variant="body1" color="text.secondary">
            Explore approved filters grouped by entity ({getTotalFilterCount()} total filters across {filterGroups.length} entities)
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
          label="Search Filters"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
          size="small"
          fullWidth
          placeholder="Search by entity, column name, filter value..."
        />
      </Box>

      {/* Filter Groups */}
      <Box sx={{ flexGrow: 1, overflowY: 'auto' }}>
        {filteredGroups.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 8 }}>
            <FilterListIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
            <Typography variant="h6" color="text.secondary" gutterBottom>
              {filterGroups.length === 0 ? 'No filters available' : 'No filters match your search'}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {filterGroups.length === 0 
                ? 'Filters will appear here once they are created and approved'
                : 'Try adjusting your search terms'
              }
            </Typography>
          </Box>
        ) : (
          filteredGroups.map((group, groupIndex) => (
            <Accordion 
              key={group.entity} 
              defaultExpanded={filteredGroups.length <= 3}
              sx={{ 
                mb: 2,
                '&:before': { display: 'none' },
                boxShadow: 'none',
                border: '1px solid rgba(224, 224, 224, 1)',
                borderRadius: '4px !important',
                '&.Mui-expanded': {
                  margin: '0 0 16px 0',
                }
              }}
            >
              <AccordionSummary 
                expandIcon={<ExpandMoreIcon />}
                sx={{
                  backgroundColor: 'rgba(0, 0, 0, 0.02)',
                  borderBottom: '1px solid rgba(224, 224, 224, 1)',
                  '&.Mui-expanded': {
                    minHeight: '48px',
                  },
                  '& .MuiAccordionSummary-content.Mui-expanded': {
                    margin: '12px 0',
                  }
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, width: '100%' }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <FilterListIcon sx={{ color: '#1976d2', fontSize: 20 }} />
                    <Typography variant="h6" sx={{ fontWeight: 600 }}>
                      {group.entity}
                    </Typography>
                  </Box>
                  <Chip
                    label={`${group.filters.length} filter${group.filters.length !== 1 ? 's' : ''}`}
                    size="small"
                    sx={{
                      backgroundColor: '#e3f2fd',
                      color: '#1976d2',
                      fontWeight: 600
                    }}
                  />
                </Box>
              </AccordionSummary>
              <AccordionDetails sx={{ p: 0 }}>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow sx={{ backgroundColor: '#fafafa' }}>
                        <TableCell sx={{ fontWeight: 'bold', color: '#031022' }}>
                          Column Name
                        </TableCell>
                        <TableCell sx={{ fontWeight: 'bold', color: '#031022' }}>
                          Filter Value
                        </TableCell>
                        <TableCell sx={{ fontWeight: 'bold', color: '#031022' }}>
                          Default Value
                        </TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {group.filters.map((filter, filterIndex) => (
                        <TableRow key={filter.column_name} hover>
                          <TableCell>
                            <Typography 
                              variant="body2" 
                              sx={{ 
                                fontFamily: 'monospace', 
                                backgroundColor: '#f5f5f5',
                                padding: '2px 6px',
                                borderRadius: '4px',
                                fontSize: '0.875rem'
                              }}
                            >
                              {filter.column_name}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={filter.filter_value}
                              size="small"
                              sx={{
                                backgroundColor: '#e8f5e8',
                                color: '#2e7d32',
                                fontWeight: 600
                              }}
                            />
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={filter.default_value}
                              size="small"
                              variant="outlined"
                              sx={{
                                borderColor: '#d32f2f',
                                color: '#d32f2f',
                                fontWeight: 600
                              }}
                            />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </AccordionDetails>
            </Accordion>
          ))
        )}
      </Box>
    </Paper>
  );
};

export default FilterDiscovery;