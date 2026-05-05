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
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Skeleton,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import EditIcon from '@mui/icons-material/Edit';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import KeyboardDoubleArrowUpIcon from '@mui/icons-material/KeyboardDoubleArrowUp';
import OutboxIcon from '@mui/icons-material/Outbox';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import SearchIcon from '@mui/icons-material/Search';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import HistoryIcon from '@mui/icons-material/History';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CloseIcon from '@mui/icons-material/Close';

import LinkIcon from '@mui/icons-material/Link';
import { useAuth } from '../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';
import Modal from '@mui/material/Modal';
import { EventAvailable as EventAvailableIcon, OpenInNew } from '@mui/icons-material';

const GenericInferflowConfigRegistryTable = ({ 
  data, 
  excludeColumns = [], 
  onViewDetails,
  onEditConfig,
  onDeleteConfig,
  onPromoteConfig,
  onScaleUpConfig,
  onCloneConfig,
  onTestConfig,
  onOnboard,
  loading,
  actionButtons = [],
}) => {
  const { user, hasPermission } = useAuth();
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.MP_CONFIG;
  const [searchQuery, setSearchQuery] = useState('');
  
  const allColumns = [
    { 
      field: 'config_id', 
      headerName: 'Inferpipe ID',
      width: '35%',
    },
    {
      field: 'host', 
      headerName: 'Host',
      width: '35%',
    },
    {
      field: 'Actions',
      headerName: 'Actions',
      width: '250px',
      stickyColumn: true,
      render: (row) => (
        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', justifyContent: 'flex-start' }}>
          {/* View Details Action */}
          <Tooltip title="View Details" disableTransition>
            <IconButton 
              onClick={() => onViewDetails(row)}
              size="small"
            >
              <InfoIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Edit Action */}
          {hasPermission(service, screenType, ACTIONS.EDIT) && (
            <Tooltip title="Edit" disableTransition>
              <IconButton 
                onClick={() => onEditConfig(row)}
                size="small"
              >
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}

          {/* Test Action */}
          {hasPermission(service, screenType, ACTIONS.TEST) && onTestConfig && (
            <Tooltip title="Test Inferpipe" disableTransition>
              <IconButton 
                onClick={() => onTestConfig(row)}
                size="small"
                sx={{ color: '#1976d2' }}
              >
                <PlayArrowIcon fontSize="small" sx={{ color: '#450839' }} />
              </IconButton>
            </Tooltip>
          )}

          {/* Monitoring URL */}
          {row.monitoring_url ? (
            <Tooltip title="Open Monitoring Dashboard" disableTransition>
              <IconButton 
                component="a" 
                href={row.monitoring_url} 
                target="_blank" 
                rel="noopener noreferrer"
                size="small"
                sx={{
                  color: '#2196f3',
                  '&:hover': {
                    color: '#1976d2'
                  },
                }}
              >
                <OpenInNew fontSize="small" />
              </IconButton>
            </Tooltip>
          ) : (
            <IconButton disabled sx={{ color: 'rgba(0, 0, 0, 0.26)' }} size="small">
              <LinkIcon fontSize="small" />
            </IconButton>
          )}

          {/* Promote Action */}
          {hasPermission(service, screenType, ACTIONS.PROMOTE) && (
            <Tooltip 
              title={!row.test_results?.tested ? "Testing is yet to be done" : "Promote"} 
              disableTransition
            >
              <span>
                <IconButton 
                  onClick={() => onPromoteConfig(row)}
                  size="small"
                  disabled={!row.test_results?.tested}
                  sx={{ 
                    color: row.test_results?.tested ? 'inherit' : 'rgba(0, 0, 0, 0.26)'
                  }}
                >
                  <KeyboardDoubleArrowUpIcon fontSize="small" />
                </IconButton>
              </span>
            </Tooltip>
          )}

          {/* Scale Up Action */}
          {hasPermission(service, screenType, ACTIONS.SCALE_UP) && (
            <Tooltip 
              title={!row.test_results?.tested ? "Testing is yet to be done" : "Scale Up"} 
              disableTransition
            >
              <span>
                <IconButton 
                  onClick={() => onScaleUpConfig(row)}
                  size="small"
                  disabled={!row.test_results?.tested}
                  sx={{ 
                    color: row.test_results?.tested ? '#2e7d32' : 'rgba(0, 0, 0, 0.26)'
                  }}
                >
                  <OutboxIcon fontSize="small" />
                </IconButton>
              </span>
            </Tooltip>
          )}

          {/* Clone Action */}
          {hasPermission(service, screenType, ACTIONS.CLONE) && (
            <Tooltip title="Clone" disableTransition>
              <IconButton 
                onClick={() => onCloneConfig(row)}
                size="small"
                sx={{ color: '#ed6c02' }}
              >
                <ContentCopyIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}

          {/* Delete Action */}
          {hasPermission(service, screenType, ACTIONS.DEACTIVATE) && (
            <Tooltip title="Delete" disableTransition>
              <IconButton 
                onClick={() => onDeleteConfig(row)}
                size="small"
                sx={{ color: '#d32f2f' }}
              >
                <DeleteForeverIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </Box>
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
          String(row.config_id || '').toLowerCase().includes(searchLower) ||
          String(row.host || '').toLowerCase().includes(searchLower)
        );
      });
    }
    
    return filtered.sort((a, b) => {
      return new Date(b.created_at) - new Date(a.created_at);
    });
  }, [data, searchQuery]);

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  const tableCellStyles = {
    borderRight: '1px solid rgba(224, 224, 224, 1)',
    padding: '12px 16px',
    whiteSpace: 'normal',
    wordBreak: 'break-word',
    backgroundColor: 'white',
    '&:last-child': {
      borderRight: 'none',
    },
    '&.sticky-column': {
      position: 'sticky',
      right: 0,
      backgroundColor: 'inherit',
      zIndex: 2,
      boxShadow: '-4px 0 6px -2px rgba(0, 0, 0, 0.1)',
    }
  };

  return (
    <Paper elevation={3} sx={{ width: '100%', height: '100vh', padding: '1rem', display: 'flex', flexDirection: 'column', marginTop: '1rem' }}>
      <Box sx={{ marginBottom: '1rem', display: 'flex', gap: '1rem', alignItems: 'center' }}>
        <TextField
          placeholder="Search by Inferpipe ID or Host"
          variant="outlined"
          size="small"
          sx={{ 
            flex: 1,
            '& .MuiOutlinedInput-root': {
              borderRadius: '8px',
            }
          }}
          value={searchQuery}
          onChange={handleSearchChange}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />
        {hasPermission(service, screenType, ACTIONS.ONBOARD) && (
          <Button
            variant="contained"
            color="primary"
            onClick={onOnboard}
            sx={{
              backgroundColor: '#450839',
              '&:hover': { backgroundColor: '#380730' },
            }}
          >
            Onboard Inferpipe
          </Button>
        )}
        <Box sx={{ display: 'flex', gap: '0.5rem' }}>
          {actionButtons?.map((button, index) => (
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
            <TableRow>
              {columns?.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={{
                    ...tableCellStyles,
                    backgroundColor: '#E6EBF2',
                    fontWeight: 'bold',
                    fontSize: '0.875rem',
                    color: '#031022',
                    width: column.width,
                    minWidth: column.width,
                    ...(column.stickyColumn && {
                      '&.sticky-column': {
                        backgroundColor: '#E6EBF2',
                      }
                    }),
                    ...(column.stickyColumn && {
                      className: 'sticky-column'
                    })
                  }}
                >
                  {column.headerName}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              [...Array(5)].map((_, index) => (
                <TableRow key={index}>
                  {columns?.map((col, colIndex) => (
                    <TableCell 
                      key={colIndex}
                      sx={{
                        ...tableCellStyles,
                        width: columns[colIndex]?.width,
                        minWidth: columns[colIndex]?.width,
                        ...(columns[colIndex]?.stickyColumn && {
                          className: 'sticky-column'
                        })
                      }}
                    >
                      <Skeleton animation="wave" width="100%" height={24} />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              filteredAndSortedData.map((row) => (
                <TableRow
                  key={row.config_id}
                  hover
                  sx={{
                    cursor: 'pointer',
                    '&:hover td': {
                      backgroundColor: 'rgba(0, 0, 0, 0.04)',
                    },
                    '&:hover td.sticky-column': {
                      backgroundColor: 'rgba(0, 0, 0, 0.04)',
                    }
                  }}
                >
                  {columns?.map((column) => (
                    <TableCell 
                      key={column.field}
                      sx={{
                        ...tableCellStyles,
                        width: column.width,
                        minWidth: column.width,
                        ...(column.stickyColumn && {
                          className: 'sticky-column'
                        })
                      }}
                    >
                      {column.render ? column.render(row) : row[column.field]}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
            {!loading && filteredAndSortedData.length === 0 && (
              <TableRow>
                <TableCell colSpan={columns.length} align="center" sx={{ py: 3 }}>
                  <Typography variant="body1" color="textSecondary">
                    No data found
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default GenericInferflowConfigRegistryTable;
