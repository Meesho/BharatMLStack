import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  Typography,
  List,
  ListItem,
  ListItemText,
  Paper,
  TextField,
  CircularProgress,
  Box,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';

const ClientDiscovery = () => {
  const [clientNames, setClientNames] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(true);
  const { user } = useAuth();

  useEffect(() => {
    const fetchClients = async () => {
      try {
        setLoading(true);
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-jobs?jobType=reader`, {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        });
        if (!response.ok) throw new Error('Failed to fetch clients');
        const data = await response.json();
        setClientNames(Array.isArray(data) ? data : []);
      } catch (error) {
        console.error('Error fetching clients:', error);
        setClientNames([]);
      } finally {
        setLoading(false);
      }
    };

    fetchClients();
  }, [user.token]);

  const filteredClients = clientNames.filter(clientName => 
    clientName.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Card
      style={{
        marginTop: '20px',
        borderRadius: '12px',
        boxShadow: '0 6px 16px rgba(0, 0, 0, 0.08)',
        backgroundColor: '#ffffff',
        width: '100%',
        maxWidth: '800px',
        margin: '20px auto',
      }}
    >
      <CardContent>
        <Typography
          variant="h5"
          style={{
            fontWeight: '600',
            color: '#24031e',
            marginBottom: '20px',
            borderBottom: '2px solid #f0f4f8',
            paddingBottom: '10px',
          }}
        >
          Client Discovery
        </Typography>

        {/* Search Input */}
        <div style={{ marginBottom: '24px' }}>
          <TextField
            placeholder="Search Clients"
            variant="outlined"
            fullWidth
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
              ),
              sx: {
                borderRadius: '8px',
                '&:hover': {
                  boxShadow: '0 0 0 1px rgba(0, 0, 0, 0.1)',
                },
              }
            }}
          />
        </div>

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress sx={{ color: '#570d48' }} />
          </Box>
        ) : (
          <Paper 
            elevation={0} 
            style={{ 
              maxHeight: '500px', 
              overflowY: 'auto',
              backgroundColor: '#faf8fc',
              borderRadius: '8px',
              padding: '8px'
            }}
          >
            <List>
              {filteredClients.length > 0 ? (
                filteredClients.map((clientName, index) => (
                  <ListItem 
                    key={index}
                    sx={{
                      mb: 1,
                      backgroundColor: '#fff',
                      borderRadius: '8px',
                      boxShadow: '0 2px 4px rgba(0, 0, 0, 0.05)',
                      transition: 'background-color 0.2s ease',
                      '&:hover': {
                        backgroundColor: '#d5c5dd',
                      }
                    }}
                  >
                    <ListItemText 
                      primary={
                        <Typography variant="body1" sx={{ fontWeight: 500, color: '#24031e' }}>
                          {clientName}
                        </Typography>
                      }
                    />
                  </ListItem>
                ))
              ) : (
                <ListItem>
                  <ListItemText 
                    primary={searchTerm ? "No matching clients found" : "Something went wrong. Unable to fetch clients."} 
                    sx={{ textAlign: 'center', color: '#666' }}
                  />
                </ListItem>
              )}
            </List>
          </Paper>
        )}
      </CardContent>
    </Card>
  );
};

export default ClientDiscovery;
