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
  Paper
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ScheduleIcon from '@mui/icons-material/Schedule';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const JobFrequencyDiscovery = () => {
  const [jobFrequencies, setJobFrequencies] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    fetchJobFrequencies();
  }, []);

  const fetchJobFrequencies = async () => {
    try {
      setLoading(true);
      const response = await embeddingPlatformAPI.getJobFrequencies();
      
      if (response.frequencies) {
        // Convert frequencies object to array
        // Structure: { "FREQ_1D": "FREQ_1D", "FREQ_1M": "FREQ_1M", ... }
        const frequenciesArray = Object.values(response.frequencies);
        setJobFrequencies(frequenciesArray);
      } else {
        setJobFrequencies([]);
      }
    } catch (error) {
      console.error('Error fetching job frequencies:', error);
      setError('Failed to load job frequencies. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  // Generate description from frequency pattern
  const generateDescription = (frequency) => {
    if (!frequency) return 'Unknown frequency';
    
    const match = frequency.match(/FREQ_(\d+)([HDWM])/);
    if (!match) return frequency;
    
    const [, number, unit] = match;
    const units = {
      'H': number === '1' ? 'hour' : 'hours',
      'D': number === '1' ? 'day' : 'days',
      'W': number === '1' ? 'week' : 'weeks',
      'M': number === '1' ? 'month' : 'months'
    };
    
    const unitName = units[unit] || unit;
    return `${number === '1' ? unitName.charAt(0).toUpperCase() + unitName.slice(1) : `${number}-${unitName}`} frequency - runs every ${number} ${unitName}`;
  };

  const filteredFrequencies = jobFrequencies.filter(freq => {
    if (!searchQuery) return true;
    const searchLower = searchQuery.toLowerCase();
    const description = generateDescription(freq);
    return freq.toLowerCase().includes(searchLower) || description.toLowerCase().includes(searchLower);
  });

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
      <Box sx={{ display: 'flex', alignItems: 'center', marginBottom: '2rem', gap: '1rem' }}>
        <ScheduleIcon sx={{ fontSize: 28, color: '#522b4a' }} />
        <Box>
          <Typography variant="h5" component="h1" sx={{ fontWeight: 600, color: '#031022' }}>
            Job Frequency Discovery
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Explore approved job frequencies for model training and processing schedules
          </Typography>
        </Box>
      </Box>

      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '1rem', mb: 2 }}>
        <TextField
          label="Search Job Frequencies"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: <SearchIcon sx={{ color: 'action.active', mr: 1 }} />,
          }}
          sx={{ width: 300 }}
          size="small"
        />
        <Typography variant="body2" color="text.secondary">
          {filteredFrequencies.length} of {jobFrequencies.length} frequencies
        </Typography>
      </Box>

      <Alert severity="info" sx={{ mb: 2 }}>
        <Typography variant="body2">
          <strong>Job Frequencies</strong> control how often model training, data processing, and synchronization jobs are executed.
          These frequencies are available for use when configuring models.
        </Typography>
      </Alert>

      <TableContainer component={Paper} sx={{ flexGrow: 1, border: '1px solid rgba(224, 224, 224, 1)' }}>
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022', borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                Frequency Name
              </TableCell>
              <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', color: '#031022' }}>
                Description
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredFrequencies.length === 0 ? (
              <TableRow>
                <TableCell colSpan={2} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {jobFrequencies.length === 0 ? 'No job frequencies available' : 'No job frequencies match your search'}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredFrequencies.map((frequency, index) => (
                <TableRow key={frequency || index} hover>
                  <TableCell sx={{ borderRight: '1px solid rgba(224, 224, 224, 1)' }}>
                    <Typography
                      variant="body2"
                      sx={{
                        fontFamily: 'monospace',
                        backgroundColor: '#e3f2fd',
                        padding: '2px 6px',
                        borderRadius: '4px',
                        fontSize: '0.875rem',
                        color: '#1976d2',
                        display: 'inline-block'
                      }}
                    >
                      {frequency}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {generateDescription(frequency)}
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

export default JobFrequencyDiscovery;
