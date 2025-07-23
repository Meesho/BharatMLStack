import React, { useState, useMemo, useEffect } from 'react';
import {
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Paper,
  IconButton,
  Tooltip,
  Box,
  TextField,
  Button,
  Skeleton,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import KeyboardDoubleArrowUpSharpIcon from '@mui/icons-material/KeyboardDoubleArrowUpSharp';
import EditIcon from '@mui/icons-material/Edit';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import ActivityIcon from '@mui/icons-material/EventAvailable';
import CloseIcon from '@mui/icons-material/Close';
import VisibilityIcon from '@mui/icons-material/Visibility';
import InfoIcon from '@mui/icons-material/Info';
import { OpenInNew } from '@mui/icons-material';
import Modal from '@mui/material/Modal';
import HistoryIcon from '@mui/icons-material/History';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import { useAuth } from '../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../constants/permissions';
import { ExpressionViewModal } from '../../../components/ExpressionViewModal';
import { convertInfixToLatex } from '../../../utils/infixToPostfix';
import { ConfigDetailsModal } from './components/ConfigDetailsModal';

const GenericNumerixTable = ({ 
  data, 
  excludeColumns = [], 
  onRowAction, 
  onTestAction,
  onEditAction,
  loading,
  actionButtons = [],
  pageName = '',
  searchPlaceholder = '',
  buttonName = '',
}) => {
  const { hasPermission } = useAuth();
  
  const service = SERVICES.NUMERIX;
  const screenType = pageName === 'numerix_testing' ? SCREEN_TYPES.NUMERIX.CONFIG_TESTING : SCREEN_TYPES.NUMERIX.CONFIG;

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRow, setSelectedRow] = useState(null);
  const [activityModalOpen, setActivityModalOpen] = useState(false);
  const [expressionModalOpen, setExpressionModalOpen] = useState(false);
  const [selectedExpression, setSelectedExpression] = useState(null);
  const [openDetailsModal, setOpenDetailsModal] = useState(false);
  const [selectedConfigForDetails, setSelectedConfigForDetails] = useState(null);
  


  const handleExpressionClick = (row) => {
    setSelectedExpression(row);
    setExpressionModalOpen(true);
  };

  const handleCloseExpressionModal = () => {
    setExpressionModalOpen(false);
    setSelectedExpression(null);
  };
  
  const handleViewDetails = (row) => {
    setSelectedConfigForDetails(row);
    setOpenDetailsModal(true);
  };

  const handleCloseDetailsModal = () => {
    setOpenDetailsModal(false);
    setSelectedConfigForDetails(null);
  };

  const allColumns = [
    { 
      field: 'ComputeId', 
      headerName: 'Compute Id',
      width: '100px',
    },
    { 
      field: 'InfixExpression', 
      headerName: 'Infix Expression',
      render: (row) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
          <Box 
            className="math-expression"
            sx={{ 
              flex: 1, 
              minWidth: 0,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              '& .MathJax': {
                fontSize: '0.85rem !important',
                maxWidth: '100%',
                overflow: 'hidden'
              }
            }}
          >
            <span dangerouslySetInnerHTML={{
              __html: `\\(${convertInfixToLatex(row.InfixExpression)}\\)`
            }} />
          </Box>
          <Tooltip title="View Expression Details" disableTransition>
            <IconButton 
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                handleExpressionClick(row);
              }}
              sx={{ color: '#450839', flexShrink: 0 }}
            >
              <VisibilityIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      )
    },
    {
      field: 'Actions',
      headerName: 'Actions',
      width: '250px',
      render: (row) => (
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', justifyContent: 'flex-end' }}>
          {/* View Details Action */}
          <Tooltip title="View Details" disableTransition>
            <IconButton onClick={() => handleViewDetails(row)} size="small">
              <InfoIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Edit Action */}
          {hasPermission(service, screenType, ACTIONS.EDIT) && onEditAction && (
            <Tooltip title="Edit" disableTransition>
              <IconButton onClick={() => onEditAction(row)} size="small">
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
          
          {/* Test Action */}
          {hasPermission(service, screenType, ACTIONS.TEST) && onTestAction && (
            <Tooltip title="Test Config" disableTransition>
              <IconButton onClick={() => onTestAction(row)} size="small">
                <PlayArrowIcon fontSize="small" sx={{ color: '#450839' }} />
              </IconButton>
            </Tooltip>
          )}

          {/* Activity History */}
          <Tooltip title="View Activity History" disableTransition>
            <IconButton 
              onClick={(e) => {
                e.stopPropagation();
                handleActivityClick(row);
              }}
              size="small"
            >
              <ActivityIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Monitoring URL */}
          {row.dashboard_link ? (
            <Tooltip title="Open Monitoring Dashboard" disableTransition>
              <IconButton 
                onClick={() => window.open(row.dashboard_link, '_blank')}
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
              <OpenInNew fontSize="small" />
            </IconButton>
          )}

          {/* Promote Action */}
          {hasPermission(service, screenType, ACTIONS.PROMOTE) && onRowAction && (
            <Tooltip title="Promote" disableTransition>
              <IconButton onClick={() => onRowAction(row)} size="small">
                <KeyboardDoubleArrowUpSharpIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </Box>
      ),
    },
  ];

  const columns = allColumns.filter((col) => {
    if (excludeColumns.includes(col.field)) return false;
    if (col.hideFromTable) return false;
    return true;
  });

  const filteredAndSortedData = useMemo(() => {
    let filtered = [...data];
    
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(row => {
        return (
          String(row.ComputeId || '').toLowerCase().includes(searchLower) ||
          String(row.InfixExpression || '').toLowerCase().includes(searchLower) ||
          String(row.Created_by || '').toLowerCase().includes(searchLower)
        );
      });
    }
    
    return filtered.sort((a, b) => new Date(b.Created_at) - new Date(a.Created_at));
  }, [data, searchQuery]);

  // Re-render MathJax when data changes
  useEffect(() => {
    const timer = setTimeout(() => {
      if (window.MathJax && filteredAndSortedData.length > 0) {
        window.MathJax.typesetPromise().catch((err) => {
          console.error('MathJax typesetting failed:', err);
        });
      }
    }, 100);
    
    return () => clearTimeout(timer);
  }, [filteredAndSortedData]);

  const handleSearchChange = (event) => {
    setSearchQuery(event.target.value);
  };

  const handleActivityClick = (row) => {
    setSelectedRow(row);
    setActivityModalOpen(true);
  };

  const handleCloseActivityModal = () => {
    setActivityModalOpen(false);
  };

  const tableCellStyles = {
    borderRight: '1px solid rgba(224, 224, 224, 1)',
    padding: '8px 16px',
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
          placeholder={searchPlaceholder || "Search expressions..."}
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
          {pageName === 'numerix_registry' && hasPermission(service, screenType, ACTIONS.ONBOARD) && (
            <Button
              variant="contained"
              onClick={actionButtons[0]?.onClick}
              sx={{
                backgroundColor: '#450839',
                '&:hover': {
                  backgroundColor: '#380730'
                },
              }}
            >
              {buttonName || "Add Compute"}
            </Button>
          )}
          {actionButtons.map((button, index) => (
            (!(pageName === 'numerix_registry' && index === 0)) && (
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
            )
          ))}
        </Box>
      </Box>
      
      <TableContainer>
        <Table stickyHeader sx={{ tableLayout: 'fixed' }}>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#E6EBF2' }}>
              {columns.map((column) => (
                <TableCell 
                  key={column.field}
                  sx={{
                    ...tableCellStyles,
                    backgroundColor: '#f5f5f5',
                    width: column.width || 'auto',
                    ...(column.field === 'ComputeId' && { 
                      width: '100px',
                      minWidth: '100px',
                      maxWidth: '100px',
                    }),
                    ...(column.field === 'Actions' && { 
                      width: '250px',
                      minWidth: '250px',
                      maxWidth: '250px',
                    }),
                    ...(column.field === 'InfixExpression' && { 
                      width: 'auto',
                    })
                  }}
                >
                  <b>{column.headerName}</b>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              [...Array(12)].map((_, index) => (
                <TableRow key={index}>
                  {columns.map((col, colIndex) => (
                    <TableCell 
                      key={colIndex}
                      sx={{
                        ...tableCellStyles,
                        width: col.width || 'auto',
                        ...(col.field === 'ComputeId' && { 
                          width: '100px',
                          minWidth: '100px',
                          maxWidth: '100px',
                        }),
                        ...(col.field === 'Actions' && { 
                          width: '250px',
                          minWidth: '250px',
                          maxWidth: '250px',
                        }),
                        ...(col.field === 'InfixExpression' && { 
                          width: 'auto',
                        })
                      }}
                    >
                      <Skeleton 
                        variant="rectangular" 
                        width="100%" 
                        height={20} 
                        animation="wave"
                      />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              filteredAndSortedData.map((row, rowIndex) => (
                <TableRow
                  key={row.ComputeId || rowIndex}
                  style={{
                    cursor: 'pointer',
                  }}
                >
                  {columns.map((column) => (
                    <TableCell 
                      key={column.field}
                      sx={{
                        ...tableCellStyles,
                        width: column.width || 'auto',
                        ...(column.field === 'ComputeId' && { 
                          width: '100px',
                          minWidth: '100px',
                          maxWidth: '100px',
                        }),
                        ...(column.field === 'Actions' && { 
                          width: '250px',
                          minWidth: '250px',
                          maxWidth: '250px',
                        }),
                        ...(column.field === 'InfixExpression' && { 
                          width: 'auto',
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

      {/* Activity Modal */}
      <Modal 
        open={activityModalOpen} 
        onClose={handleCloseActivityModal} 
        aria-labelledby="activity-history-modal"
      >
        <Box sx={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: 600, maxWidth: '90vw', bgcolor: 'background.paper', boxShadow: 24, borderRadius: 2, overflow: 'hidden' }}>
          <Box sx={{ bgcolor: '#450839', color: 'white', p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <HistoryIcon />
              <Typography variant="h6" component="h2">
                Activity History
              </Typography>
            </Box>
            <IconButton onClick={handleCloseActivityModal} sx={{ color: 'white' }} aria-label="close">
              <CloseIcon />
            </IconButton>
          </Box>
          {selectedRow && (
            <Box sx={{ p: 3 }}>
              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1, mb: 3 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  CREATED
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <PersonIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>By</Typography>
                    </Box>
                    <Typography variant="body2">{selectedRow.Created_by || 'N/A'}</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRow.Created_at ? new Date(selectedRow.Created_at).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' }) : 'N/A'}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRow.Created_at ? new Date(selectedRow.Created_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: true }) : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
              <Paper elevation={0} sx={{ p: 2, bgcolor: '#f8f9fa', borderRadius: 1 }}>
                <Typography variant="subtitle2" sx={{ color: '#666', mb: 2 }}>
                  LAST UPDATED
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <PersonIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>By</Typography>
                    </Box>
                    <Typography variant="body2">{selectedRow.Updated_by || 'N/A'}</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CalendarTodayIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Date</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRow.Updated_at ? new Date(selectedRow.Updated_at).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' }) : 'N/A'}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AccessTimeIcon fontSize="small" sx={{ color: '#450839' }} />
                      <Typography variant="body2" fontWeight={500}>Time</Typography>
                    </Box>
                    <Typography variant="body2">
                      {selectedRow.Updated_at ? new Date(selectedRow.Updated_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: true }) : 'N/A'}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
            </Box>
          )}
        </Box>
      </Modal>

      {/* Expression View Modal */}
      <ExpressionViewModal
        open={expressionModalOpen}
        onClose={handleCloseExpressionModal}
        infixExpression={selectedExpression?.InfixExpression || ''}
        postfixExpression={selectedExpression?.PostfixExpression || ''}
      />

      {/* Details Modal */}
      <ConfigDetailsModal
        open={openDetailsModal}
        onClose={handleCloseDetailsModal}
        config={selectedConfigForDetails}
      />
    </Paper>
  );
};

export default GenericNumerixTable;
