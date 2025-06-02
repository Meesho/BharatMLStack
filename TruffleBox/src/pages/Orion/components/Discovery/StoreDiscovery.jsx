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
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';

const StoreDiscovery = () => {
  const [storeData, setStoreData] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(true);
  const { user } = useAuth();

  useEffect(() => {
    const fetchStores = async () => {
      try {
        setLoading(true);
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/get-store`, {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        });
        if (!response.ok) throw new Error('Failed to fetch stores');
        const data = await response.json();
        
        const formattedData = Object.entries(data).map(([id, details]) => ({
          StoreId: id,
          ConfId: details['conf-id'],
          DbType: details['db-type'],
          Table: details['table'],
          MaxColumnSize: details['max-column-size-in-bytes'],
          MaxRowSize: details['max-row-size-in-bytes'],
          PrimaryKeys: details['primary-keys'] ? details['primary-keys'].join(', ') : '-',
          TableTTL: details['table-ttl']
        }));

        setStoreData(formattedData);
      } catch (error) {
        console.error('Error fetching stores:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchStores();
  }, [user.token]);

  const filteredStores = storeData.filter(store => 
    store.ConfId.toString().includes(searchTerm) ||
    store.DbType.toLowerCase().includes(searchTerm.toLowerCase()) ||
    store.Table.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
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
        <Typography
          variant="h6"
          style={{
            fontWeight: 'bold',
            color: '#24031e',
            marginBottom: '20px',
          }}
        >
          Store Discovery
        </Typography>

        {/* Search Input */}
        <div style={{ marginBottom: '20px' }}>
          <TextField
            placeholder="Search Stores"
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
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Store ID</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Config ID</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Database Type</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Table</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Max Column Size</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Max Row Size</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>Primary Keys</TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>TTL</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={7} align="center">Loading...</TableCell>
                </TableRow>
              ) : filteredStores.length > 0 ? (
                filteredStores.map((store, index) => (
                  <TableRow key={index}>
                    <TableCell>{store.StoreId || '-'}</TableCell>
                    <TableCell>{store.ConfId || '-'}</TableCell>
                    <TableCell>{store.DbType || '-'}</TableCell>
                    <TableCell>{store.Table || '-'}</TableCell>
                    <TableCell>{store.MaxColumnSize || '-'}</TableCell>
                    <TableCell>{store.MaxRowSize || '-'}</TableCell>
                    <TableCell>{store.PrimaryKeys || '-'}</TableCell>
                    <TableCell>{store.TableTTL || '-'}</TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={7} align="center">No stores found</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </CardContent>
    </Card>
  );
};

export default StoreDiscovery;
