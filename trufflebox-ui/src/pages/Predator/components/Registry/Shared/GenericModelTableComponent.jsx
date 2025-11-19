import React, { useState, useEffect } from 'react';
import {
  Box,
  Chip,
  IconButton,
  Tooltip,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Button,
  Skeleton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Checkbox,
  Grid,
  Snackbar,
  Alert as MuiAlert,
} from '@mui/material';
import JsonViewer from '../../../../../components/JsonViewer';
import InfoIcon from '@mui/icons-material/Info';
import SearchIcon from '@mui/icons-material/Search';
import LinkIcon from '@mui/icons-material/Link';
import KeyboardDoubleArrowUpIcon from '@mui/icons-material/KeyboardDoubleArrowUp';
import OutboxIcon from '@mui/icons-material/Outbox';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import SummarizeIcon from '@mui/icons-material/Summarize';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { useAuth } from '../../../../Auth/AuthContext';
import * as URL_CONSTANTS from '../../../../../config';
import OpenInNew from '@mui/icons-material/OpenInNew';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import EditIcon from '@mui/icons-material/Edit';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../../constants/permissions';

const GenericModelTableComponent = ({
  handleCloneAction,
  handlePromoteAction,
  handleRemoveAction,
  handleOnboardModel,
  handleTestModel,
  handleEditModel,
  handleUploadModel,
  handleEditModelForUpload,
  models: propModels,
  loading: propLoading
}) => {
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedModels, setSelectedModels] = useState([]);
  const [deleteConfirmDialogOpen, setDeleteConfirmDialogOpen] = useState(false);
  const [modelsToDelete, setModelsToDelete] = useState([]);
  const [fullDataDialogOpen, setFullDataDialogOpen] = useState(false);
  const [selectedFullData, setSelectedFullData] = useState(null);
  
  // Toast state
  const [toastOpen, setToastOpen] = useState(false);
  const [toastMessage, setToastMessage] = useState('');
  const [toastSeverity, setToastSeverity] = useState('error');
  
  const { user, hasPermission } = useAuth();
  const service = SERVICES.PREDATOR;
  const screenType = SCREEN_TYPES.PREDATOR.MODEL;

  useEffect(() => {
    setModels(propModels || []);
    setLoading(propLoading !== undefined ? propLoading : false);
  }, [propModels, propLoading]);

  const formatDate = (dateString) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const handleFullDataClick = (row) => {
    setSelectedFullData(row);
    setFullDataDialogOpen(true);
  };

  const handleCloseFullDataDialog = () => {
    setFullDataDialogOpen(false);
    setSelectedFullData(null);
  };

  const handleRowCheckboxToggle = (id) => {
    setSelectedModels(prev => {
      if (prev.includes(id)) {
        return prev.filter(modelId => modelId !== id);
      } else {
        return [...prev, id];
      }
    });
  };

  const validateModelsForPromotion = (modelsToValidate) => {
    const invalidModels = modelsToValidate.filter(model => {
      // Check if test_results is null or undefined
      const hasValidTestResults = model.testResults !== null && model.testResults !== undefined;
      // Check if has_nil_data is false (or doesn't exist, which we treat as false)
      const hasNoNilData = model.hasNilData === false || model.hasNilData === undefined;
      
      return !hasValidTestResults || !hasNoNilData;
    });

    return {
      isValid: invalidModels.length === 0,
      invalidModels: invalidModels
    };
  };

  const handleBulkPromote = () => {
    const selectedModelData = models.filter(model => selectedModels.includes(model.id));
    
    // Validate models before promotion
    const validation = validateModelsForPromotion(selectedModelData);
    
    if (!validation.isValid) {
      const modelNames = validation.invalidModels.map(m => m.modelName).join(', ');
      setToastMessage(`Testing is yet to be done for: ${modelNames}`);
      setToastSeverity('error');
      setToastOpen(true);
      return;
    }
    
    handlePromoteAction(selectedModelData);
    setSelectedModels([]);
  };

  const handleBulkScaleUp = () => {
    const selectedModelData = models.filter(model => selectedModels.includes(model.id));
    handleCloneAction(selectedModelData);
    setSelectedModels([]);
  };

  const handleBulkDelete = () => {
    setModelsToDelete(selectedModels);
    setDeleteConfirmDialogOpen(true);
  };

  const confirmDelete = () => {
    handleRemoveAction(modelsToDelete);
    setDeleteConfirmDialogOpen(false);
    setModelsToDelete([]);
    setSelectedModels([]);
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

  const columns = [
    {
      field: 'checkbox',
      headerName: '',
      width: '48px',
      align: 'center',
      stickyColumn: false
    },
    {
      field: 'modelName',
      headerName: 'Model Name',
      width: '32%',
      stickyColumn: false
    },
    {
      field: 'host',
      headerName: 'Host',
      width: '22%',
      stickyColumn: false
    },
    {
      field: 'gcsPath',
      headerName: 'GCS Path',
      width: '22%',
      stickyColumn: false
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: '23%',
      stickyColumn: true,
      render: (row) => (
        <Box sx={{
          display: 'flex',
          gap: 0.5,
          alignItems: 'center',
          justifyContent: 'flex-start',
          flexWrap: 'nowrap',
          minWidth: 'max-content'
        }}>
          {/* View All Data */}
          <Tooltip title="View All Data" disableTransition>
            <IconButton size="small" onClick={() => handleFullDataClick(row)}>
              <InfoIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* Edit Model */}
          {hasPermission(service, screenType, ACTIONS.EDIT) && (
            <Tooltip title="Edit Model" disableTransition>
              <IconButton
                size="small"
                onClick={() => handleEditModel(row)}
                sx={{
                  color: '#1976d2',
                  '&:hover': {
                    backgroundColor: 'rgba(25, 118, 210, 0.04)'
                  }
                }}
              >
                <EditIcon fontSize="small" sx={{ color: '#450839' }} />
              </IconButton>
            </Tooltip>
          )}

          {/* Test Model */}
          {hasPermission(service, screenType, ACTIONS.TEST) && (
            <Tooltip title="Test Model" disableTransition>
              <IconButton
                size="small"
                onClick={() => handleTestModel(row)}
                sx={{
                  color: '#1976d2',
                  '&:hover': {
                    backgroundColor: 'rgba(25, 118, 210, 0.04)'
                  }
                }}
              >
                <PlayArrowIcon fontSize="small" sx={{ color: '#450839' }} />
              </IconButton>
            </Tooltip>
          )}

          {/* View Dashboard */}
          <Tooltip title="View Dashboard" disableTransition>
            <IconButton
              size="small"
              href={row.monitoringUrl}
              target="_blank"
              rel="noopener noreferrer"
              sx={{
                color: '#2196f3',
                '&:hover': {
                  color: '#1976d2',
                  backgroundColor: 'rgba(33, 150, 243, 0.04)'
                }
              }}
            >
              <OpenInNew fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      )
    },
  ];

  const getTableCellStyles = (column) => {
    const baseStyles = {
      px: 2,
      py: 1.5,
      fontWeight: 'medium',
      fontSize: '0.875rem',
      borderRight: '1px solid #e0e0e0',
    };

    if (!column.autoWidth) {
      baseStyles.maxWidth = column.width || 200;
      baseStyles.width = column.width;
    } else {
      baseStyles.whiteSpace = 'normal';
      baseStyles.wordBreak = 'break-word';
    }

    return baseStyles;
  };

  const lastColumnStyle = {
    px: 2,
    py: 1.5,
    fontWeight: 'medium',
    fontSize: '0.875rem',
    maxWidth: 200,
    borderRight: 'none',
  };

  const filteredModels = models.filter(model => {
    const searchTermLower = searchQuery.toLowerCase();
    return (
      model.modelName?.toLowerCase().includes(searchTermLower) ||
      model.host?.toLowerCase().includes(searchTermLower) ||
      model.createdBy?.toLowerCase().includes(searchTermLower) ||
      model.updatedBy?.toLowerCase().includes(searchTermLower) ||
      model.gcsPath?.toLowerCase().includes(searchTermLower)
    );
  });

  const FeaturesDisplay = ({ features }) => {
    const [expanded, setExpanded] = useState(false);

    if (!features || features.length === 0) {
      return <Typography variant="body2" color="textSecondary">No features</Typography>;
    }

    // Group features by feature type (left part before |)
    const groupedFeatures = features.reduce((acc, feature) => {
      const pipeIndex = feature.indexOf('|');
      let featureType = 'Other';
      let featureName = feature;

      if (pipeIndex !== -1) {
        featureType = feature.substring(0, pipeIndex).trim() || 'Other';
        featureName = feature.substring(pipeIndex + 1).trim();
      }

      if (!acc[featureType]) {
        acc[featureType] = [];
      }
      acc[featureType].push(featureName);
      return acc;
    }, {});

    const featureTypes = Object.keys(groupedFeatures);
    const totalFeatures = features.length;

    return (
      <Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
          <Typography variant="body2" color="textSecondary" sx={{ fontWeight: 600 }}>
            Features ({totalFeatures})
          </Typography>
          <IconButton size="small" onClick={() => setExpanded(!expanded)}>
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          </IconButton>
        </Box>
        
        {expanded ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {featureTypes.map((featureType, typeIndex) => (
              <Box key={typeIndex} sx={{ 
                border: '1px solid #e0e0e0', 
                borderRadius: 1, 
                p: 1.5,
                backgroundColor: '#fafafa'
              }}>
                <Typography variant="caption" sx={{ 
                  fontWeight: 600, 
                  color: '#450839',
                  display: 'block',
                  mb: 1
                }}>
                  {featureType} ({groupedFeatures[featureType].length})
                </Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {groupedFeatures[featureType].map((feature, index) => (
                    <Chip
                      key={index}
                      label={feature}
                      size="small"
                      sx={{
                        fontSize: '0.7rem',
                        height: 'auto',
                        backgroundColor: 'white',
                        '& .MuiChip-label': {
                          paddingTop: '4px',
                          paddingBottom: '4px',
                          paddingLeft: '8px',
                          paddingRight: '8px'
                        }
                      }}
                    />
                  ))}
                </Box>
              </Box>
            ))}
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {featureTypes.map((featureType, index) => (
              <Chip
                key={index}
                label={`${featureType} (${groupedFeatures[featureType].length})`}
                size="small"
                sx={{
                  fontSize: '0.75rem',
                  backgroundColor: '#f0f0f0',
                  fontWeight: 500
                }}
              />
            ))}
          </Box>
        )}
      </Box>
    );
  };

  return (
    <Box sx={{ p: 2 }}>

      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <TextField
          placeholder="Search models..."
          variant="outlined"
          size="small"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          sx={{ flex: 1, mr: 2 }}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
        />

        <Box sx={{ display: 'flex', gap: 2 }}>
          {hasPermission(service, screenType, ACTIONS.UPLOAD) && (
            <Button
              variant="outlined"
              startIcon={<CloudUploadIcon />}
              onClick={handleUploadModel}
              sx={{
                color: '#450839',
                borderColor: '#450839',
                '&:hover': {
                  backgroundColor: '#450839',
                  color: 'white',
                  borderColor: '#450839',
                },
                whiteSpace: 'nowrap'
              }}
            >
              Upload Model
            </Button>
          )}
          {hasPermission(service, screenType, ACTIONS.UPLOAD_EDIT) && (
            <Button
              variant="outlined"
              startIcon={<EditIcon />}
              onClick={handleEditModelForUpload}
              sx={{
                color: '#450839',
                borderColor: '#450839',
                '&:hover': {
                  backgroundColor: '#450839',
                  color: 'white',
                  borderColor: '#450839',
                },
                whiteSpace: 'nowrap'
              }}
            >
              Edit Model
            </Button>
          )}
          <Button
            variant="contained"
            onClick={handleOnboardModel}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              },
              whiteSpace: 'nowrap'
            }}
          >
            Deploy Model
          </Button>
        </Box>
      </Box>

      {/* Bulk Action Buttons */}
      {selectedModels.length > 0 && (
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2, mb: 2 }}>
          {hasPermission(service, screenType, ACTIONS.PROMOTE) && (
            <Button
              variant="outlined"
              startIcon={<KeyboardDoubleArrowUpIcon />}
              onClick={handleBulkPromote}
              sx={{
                color: '#450839',
                borderColor: '#450839',
                '&:hover': {
                  borderColor: '#380730',
                  backgroundColor: 'rgba(69, 8, 57, 0.04)'
                }
              }}
            >
              Promote
            </Button>
          )}
          {hasPermission(service, screenType, ACTIONS.SCALE_UP) && (
            <Button
              variant="outlined"
              startIcon={<OutboxIcon />}
              onClick={handleBulkScaleUp}
              sx={{
                color: '#450839',
                borderColor: '#450839',
                '&:hover': {
                  borderColor: '#380730',
                  backgroundColor: 'rgba(69, 8, 57, 0.04)'
                }
              }}
            >
              Scale Up
            </Button>
          )}
          {hasPermission(service, screenType, ACTIONS.DELETE) && (
            <Button
              variant="outlined"
              color="error"
              startIcon={<DeleteForeverIcon />}
              onClick={handleBulkDelete}
            >
              Delete
            </Button>
          )}
        </Box>
      )}

      <Paper elevation={3} sx={{ width: '100%', overflow: 'hidden', boxShadow: 3 }}>
        <TableContainer
          component={Paper}
          elevation={3}
          sx={{
            maxHeight: 'calc(100vh - 200px)',
            overflowX: 'auto',
            overflowY: 'auto',
            '& .MuiTable-root': {
              minWidth: 1200, // Ensures table doesn't compress too much
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
                {columns.map((column) => (
                  <TableCell
                    key={column.field}
                    align={column.align || 'left'}
                    sx={{
                      ...tableCellStyles,
                      backgroundColor: '#E6EBF2',
                      fontWeight: 'bold',
                      color: '#031022',
                      width: column.width,
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
                    {columns.map((col, colIndex) => (
                      <TableCell
                        key={colIndex}
                        align={col.align || 'left'}
                        sx={{
                          ...tableCellStyles,
                          width: col.width,
                          ...(col.stickyColumn && {
                            className: 'sticky-column'
                          })
                        }}
                      >
                        <Skeleton animation="wave" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : filteredModels.length > 0 ? (
                filteredModels.map((row) => (
                  <TableRow
                    key={row.id}
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
                    {columns.map((column, index) => (
                      <TableCell
                        key={column.field}
                        align={column.align || 'left'}
                        sx={{
                          ...tableCellStyles,
                          width: column.width,
                          ...(column.stickyColumn && {
                            className: 'sticky-column'
                          })
                        }}
                      >
                        {column.field === 'checkbox' ? (
                          <Checkbox
                            checked={selectedModels.includes(row.id)}
                            onChange={() => handleRowCheckboxToggle(row.id)}
                            onClick={(e) => e.stopPropagation()}
                          />
                        ) : column.render ? (
                          column.render(row)
                        ) : (
                          row[column.field]
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
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

      {/* Full Data Details Dialog */}
      <Dialog
        open={fullDataDialogOpen}
        onClose={handleCloseFullDataDialog}
        maxWidth="lg"
        fullWidth
        PaperProps={{
          sx: {
            maxHeight: '90vh'
          }
        }}
      >
        <DialogTitle
          sx={{
            backgroundColor: '#450839',
            color: 'white',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '4px 24px',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <InfoIcon />
            <Typography variant="h6">Model Information</Typography>
          </Box>
          <IconButton
            onClick={handleCloseFullDataDialog}
            sx={{ color: 'white' }}
            edge="end"
          >
            <Box sx={{ fontSize: '1.5rem' }}>×</Box>
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ px: 4, py: 3 }}>
          {selectedFullData && (
            <Box>
              {/* Basic Information */}
              <Paper elevation={0} sx={{ p: 3, mb: 3, backgroundColor: '#f8f8f8', borderRadius: 2 }}>
                <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 'bold' }}>
                  Basic Information
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Model Name</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.modelName || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Host</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.host || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Machine Type</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.machineType || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">GCS Path</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium', wordBreak: 'break-all' }}>
                        {selectedFullData.gcsPath || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">App Token</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.appToken || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Deployable Running Status</Typography>
                      <Chip
                        label={selectedFullData.deployableRunningStatus ? 'Running' : 'Stopped'}
                        color={selectedFullData.deployableRunningStatus ? 'success' : 'default'}
                        size="small"
                        sx={{
                          alignSelf: 'flex-start',
                          mt: 0.5,
                          backgroundColor: selectedFullData.deployableRunningStatus ? '#E7F6E7' : '#F5F5F5',
                          color: selectedFullData.deployableRunningStatus ? '#2E7D32' : '#757575',
                          fontWeight: 'bold',
                        }}
                      />
                    </Box>
                  </Grid>
                </Grid>
              </Paper>

              {/* Metadata Section */}
              {selectedFullData.metaData && (
                <Paper elevation={0} sx={{ p: 3, mb: 3, backgroundColor: '#f8f8f8', borderRadius: 2 }}>
                  <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 'bold' }}>
                    Model Metadata
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Backend</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.metaData.backend || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Batch Size</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.metaData.batch_size || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Instance Count</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.metaData.instance_count || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    {selectedFullData.metaData.platform && (
                      <Grid item xs={6}>
                        <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                          <Typography variant="body2" color="textSecondary">Platform</Typography>
                          <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                            {selectedFullData.metaData.platform}
                          </Typography>
                        </Box>
                      </Grid>
                    )}
                  </Grid>

                  {/* Inputs */}
                  <Typography variant="h6" sx={{ mt: 3, mb: 2, color: '#450839', fontWeight: 'bold' }}>
                    Inputs
                  </Typography>
                  {selectedFullData.metaData.inputs && selectedFullData.metaData.inputs.length > 0 ? (
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      {selectedFullData.metaData.inputs.map((input, index) => (
                        <Paper key={index} elevation={0} sx={{ p: 2, backgroundColor: '#ffffff', borderRadius: 2, borderLeft: '4px solid #450839' }}>
                          <Grid container spacing={2}>
                            <Grid item xs={3}>
                              <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                <Typography variant="body2" color="textSecondary">Name</Typography>
                                <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                                  {input.name || 'Unnamed'}
                                </Typography>
                              </Box>
                            </Grid>
                            <Grid item xs={3}>
                              <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                <Typography variant="body2" color="textSecondary">Dims</Typography>
                                <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                                  {Array.isArray(input.dims) ? JSON.stringify(input.dims) : (input.dims || 'Not specified')}
                                </Typography>
                              </Box>
                            </Grid>
                            <Grid item xs={3}>
                              <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                <Typography variant="body2" color="textSecondary">Data Type</Typography>
                                <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                                  {input.data_type || 'Not specified'}
                                </Typography>
                              </Box>
                            </Grid>
                            <Grid item xs={3}>
                              <FeaturesDisplay features={input.features} />
                            </Grid>
                          </Grid>
                        </Paper>
                      ))}
                    </Box>
                  ) : (
                    <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                      No input data available
                    </Typography>
                  )}

                  {/* Outputs */}
                  <Typography variant="h6" sx={{ mt: 3, mb: 2, color: '#450839', fontWeight: 'bold' }}>
                    Outputs
                  </Typography>
                  {selectedFullData.metaData.outputs && selectedFullData.metaData.outputs.length > 0 ? (
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      {selectedFullData.metaData.outputs.map((output, index) => (
                        <Paper key={index} elevation={0} sx={{ p: 2, backgroundColor: '#ffffff', borderRadius: 2, borderLeft: '4px solid #450839' }}>
                          <Grid container spacing={2}>
                            <Grid item xs={4}>
                              <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                <Typography variant="body2" color="textSecondary">Name</Typography>
                                <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                                  {output.name || 'Unnamed'}
                                </Typography>
                              </Box>
                            </Grid>
                            <Grid item xs={4}>
                              <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                <Typography variant="body2" color="textSecondary">Dims</Typography>
                                <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                                  {typeof output.dims === 'object' ? JSON.stringify(output.dims) : (output.dims || 'Not specified')}
                                </Typography>
                              </Box>
                            </Grid>
                            <Grid item xs={4}>
                              <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                <Typography variant="body2" color="textSecondary">Data Type</Typography>
                                <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                                  {output.data_type || 'Not specified'}
                                </Typography>
                              </Box>
                            </Grid>
                          </Grid>
                        </Paper>
                      ))}
                    </Box>
                  ) : (
                    <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                      No output data available
                    </Typography>
                  )}

                  {/* Ensemble Scheduling */}
                  {selectedFullData.metaData.ensemble_scheduling && (
                    <>
                      <Typography variant="h6" sx={{ mt: 3, mb: 2, color: '#450839', fontWeight: 'bold' }}>
                        Ensemble Scheduling
                      </Typography>
                      <JsonViewer
                        data={selectedFullData.metaData.ensemble_scheduling}
                        title="Ensemble Scheduling Configuration"
                        defaultExpanded={true}
                        maxHeight={300}
                        enableCopy={true}
                        enableDiff={false}
                        editable={false}
                        sx={{ mt: 2, mb: 2 }}
                      />
                    </>
                  )}
                </Paper>
              )}

              {/* Deployment Configuration */}
              {selectedFullData.deploymentConfig && (
                <Paper elevation={0} sx={{ p: 3, mb: 3, backgroundColor: '#f8f8f8', borderRadius: 2 }}>
                  <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 'bold' }}>
                    Deployment Configuration
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">CPU Request</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.cpu_request || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">CPU Limit</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.cpu_limit || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Memory Request</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.mem_request || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Memory Limit</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.mem_limit || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">GPU Request</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.gpu_request || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">GPU Limit</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.gpu_limit || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Min Replicas</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.min_replica || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Max Replicas</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.max_replica || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={12}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Node Selector</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.deploymentConfig.node_selector || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                  </Grid>
                </Paper>
              )}

              {/* Test Results */}
              {selectedFullData.testResults && (
                <Paper elevation={0} sx={{ p: 3, mb: 3, backgroundColor: '#f8f8f8', borderRadius: 2 }}>
                  <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 'bold' }}>
                    Test Results
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Functionally Tested</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.testResults.functionally_tested ? 'Yes' : 'No'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Load Testing Stage</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.testResults.load_testing_stage || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid item xs={12}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                        <Typography variant="body2" color="textSecondary">Load Testing Result</Typography>
                        <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                          {selectedFullData.testResults.load_testing_result || 'Not specified'}
                        </Typography>
                      </Box>
                    </Grid>
                  </Grid>
                </Paper>
              )}

              {/* Connection Configuration */}
              {selectedFullData.connectionConfig && (
                <Paper elevation={0} sx={{ p: 3, mb: 3, backgroundColor: '#f8f8f8', borderRadius: 2 }}>
                  <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 'bold' }}>
                    Connection Configuration
                  </Typography>
                  <JsonViewer
                    data={selectedFullData.connectionConfig}
                    title="Connection Configuration"
                    defaultExpanded={true}
                    maxHeight={300}
                    enableCopy={true}
                    enableDiff={false}
                    editable={false}
                    sx={{ mt: 2, mb: 2 }}
                  />
                </Paper>
              )}

              {/* Activity Information */}
              <Paper elevation={0} sx={{ p: 3, mb: 3, backgroundColor: '#f8f8f8', borderRadius: 2 }}>
                <Typography variant="h6" sx={{ mb: 2, color: '#450839', fontWeight: 'bold' }}>
                  Activity Information
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Created By</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.createdBy || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Created At</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.createdAt || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Updated By</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.updatedBy || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={6}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', mb: 2 }}>
                      <Typography variant="body2" color="textSecondary">Updated At</Typography>
                      <Typography variant="body1" sx={{ fontWeight: 'medium' }}>
                        {selectedFullData.updatedAt || 'Not specified'}
                      </Typography>
                    </Box>
                  </Grid>
                </Grid>
              </Paper>

              {/* Monitoring URL */}
              {selectedFullData.monitoringUrl && (
                <Box>
                  <Typography variant="subtitle2" color="textSecondary">Monitoring Dashboard</Typography>
                  <Button
                    startIcon={<OpenInNew />}
                    onClick={() => window.open(selectedFullData.monitoringUrl, '_blank')}
                    sx={{ color: '#2196f3', textTransform: 'none' }}
                  >
                    Open Dashboard
                  </Button>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleCloseFullDataDialog}
            variant="contained"
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              }
            }}
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteConfirmDialogOpen}
        onClose={() => setDeleteConfirmDialogOpen(false)}
        aria-labelledby="alert-dialog-title"
        aria-describedby="alert-dialog-description"
      >
        <DialogTitle
          sx={{
            backgroundColor: '#450839',
            color: 'white',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '4px 24px',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="h6">Confirm Deletion</Typography>
          </Box>
          <IconButton
            onClick={() => setDeleteConfirmDialogOpen(false)}
            sx={{ color: 'white' }}
            edge="end"
          >
            <Box sx={{ fontSize: '1.5rem' }}>×</Box>
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ px: 4, py: 3, mt: 1 }}>
          <Typography variant="body1">
            Are you sure you want to delete {modelsToDelete.length > 1 ? `these ${modelsToDelete.length} models` : 'this model'}?
          </Typography>
          <Typography variant="body2" color="error" sx={{ mt: 2 }}>
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={() => setDeleteConfirmDialogOpen(false)}
            variant="outlined"
            sx={{
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                borderColor: '#380730'
              }
            }}
          >
            Cancel
          </Button>
          <Button onClick={confirmDelete} variant="contained" sx={{ backgroundColor: '#450839', '&:hover': { backgroundColor: '#380730' } }} autoFocus>
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Toast/Snackbar for validation messages */}
      <Snackbar
        open={toastOpen}
        autoHideDuration={6000}
        onClose={() => setToastOpen(false)}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <MuiAlert
          onClose={() => setToastOpen(false)}
          severity={toastSeverity}
          variant="filled"
          sx={{ width: '100%' }}
        >
          {toastMessage}
        </MuiAlert>
      </Snackbar>
    </Box>
  );
};

export default GenericModelTableComponent;