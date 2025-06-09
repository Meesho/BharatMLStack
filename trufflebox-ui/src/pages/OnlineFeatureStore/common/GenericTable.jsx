import React, { useState, useMemo } from 'react';
import {
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Paper,
  Chip,
  IconButton,
  Tooltip,
  Box,
  TextField,
  Button,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import SearchIcon from '@mui/icons-material/Search';
import { useAuth } from '../../Auth/AuthContext';

const GenericTable = ({ 
  data, 
  excludeColumns = [], 
  onRowAction, 
  loading,
  actionButtons = [],
})  => {
  const { user } = useAuth();
  const [searchQuery, setSearchQuery] = useState('');
  
  const allColumns = [
    { field: 'RequestId', headerName: 'Request ID' },
    { field: 'EntityLabel', headerName: 'Entity Label' },
    { field: 'FeatureGroupLabel', headerName: 'FG Label' },
    { 
      field: 'CreatedBy', 
      headerName: 'Created By',
      roles: ['admin']
    },
    { field: 'ApprovedBy', headerName: 'Admin POC' },
    { field: 'CreatedAt', headerName: 'Created At' },
    { field: 'UpdatedAt', headerName: 'Updated At' },
    {
      field: 'Status',
      headerName: 'Status',
      render: (row) => (
        <Chip
          label={row.Status}
          color={
            row.Status === 'APPROVED'
              ? 'success'
              : row.Status === 'REJECTED'
              ? 'error'
              : undefined
          }
          sx={{
            backgroundColor: row.Status === 'APPROVED' 
              ? '#E7F6E7'
              : row.Status === 'REJECTED'
              ? '#FFEBEE'
              : row.Status === 'PENDING APPROVAL'
              ? '#FFF8E1'
              : undefined,
            color: row.Status === 'APPROVED'
              ? '#2E7D32'
              : row.Status === 'REJECTED'
              ? '#D32F2F'
              : row.Status === 'PENDING APPROVAL'
              ? '#F57C00'
              : undefined,
            fontWeight: 'bold',
          }}
        />
      ),
    },
    {
      field: 'Actions',
      headerName: 'Actions',
      render: (row) => (
        <Tooltip title="View Details">
          <IconButton onClick={() => onRowAction(row)}>
            <InfoIcon />
          </IconButton>
        </Tooltip>
      ),
    },
  ];

  const columns = allColumns.filter((col) => {
    if (excludeColumns.includes(col.field)) return false;
    if (col.roles) {
      return col.roles.includes(user?.role);
    }
    return true;
  });

  const filteredAndSortedData = useMemo(() => {
    let filtered = [...data];
    
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.RequestId).toLowerCase().includes(searchLower) ||
          (row.CreatedBy && row.CreatedBy.toLowerCase().includes(searchLower)) ||
          (row.ApprovedBy && row.ApprovedBy.toLowerCase().includes(searchLower))
        );
      });
    }
    
    return filtered.sort((a, b) => b.RequestId - a.RequestId);
  }, [data, searchQuery]);

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  // Common table cell styles with vertical divider
  const tableCellStyles = {
    borderRight: '1px solid rgba(224, 224, 224, 1)',
    '&:last-child': {
      borderRight: 'none',
    },
  };

  return (
    <Paper elevation={3} sx={{ width: '100%', height: '100vh', padding: '1rem', display: 'flex', flexDirection: 'column', marginTop: '1rem' }}>
      <Box
        sx={{ 
          marginBottom: '1rem', 
          display: 'flex', 
          gap: '1rem', 
          alignItems: 'center'
        }}>
        <TextField
          placeholder={user?.role === 'admin' 
            ? "Search by Request ID, Admin POC, or Created By"
            : "Search by Request ID or Admin POC"
          }
          variant="outlined"
          size="small"
          sx={{ flex: 1 }}
          value={searchQuery}
          onChange={handleSearchChange}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
          <Box sx={{ display: 'flex', gap: '0.5rem' }}>
            {actionButtons.map((button, index) => (
              <Button
                key={index}
                variant={button.variant || "contained"}
                onClick={button.onClick}
                sx={{
                  backgroundColor: button.color || '#450839',
                  '&:hover': {
                    backgroundColor: button.hoverColor || '#380730'
                  },
                  ...button.sx
                }}
              >
                {button.label}
              </Button>
            ))}
          </Box>
      </Box>
      <TableContainer component={Paper} elevation={3} sx={{ marginTop: '1rem' }}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              {columns.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={tableCellStyles}
                >
                  <b>{column.headerName}</b>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              [...Array(5)].map((_, index) => (
                <TableRow key={index}>
                  {columns.map((col, colIndex) => (
                    <TableCell 
                      key={colIndex}
                      sx={tableCellStyles}
                    >
                      {/* Placeholder for Skeleton */}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              filteredAndSortedData.map((row) => (
                <TableRow
                  key={row.RequestId}
                  style={{
                    cursor: 'pointer',
                  }}
                >
                  {columns.map((column) => (
                    <TableCell 
                      key={column.field}
                      sx={tableCellStyles}
                    >
                      {column.render ? column.render(row) : row[column.field]}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default GenericTable;
