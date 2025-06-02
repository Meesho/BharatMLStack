import React, { useState } from 'react';
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
  Modal,
  Box,
  Button,
  IconButton,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import InfoIcon from '@mui/icons-material/Info';
import EditIcon from '@mui/icons-material/Edit';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import axios from 'axios';
import { useAuth } from '../../../Auth/AuthContext';
import * as URL_CONSTANTS from '../../../../config';

const FeatureList = ({
  features, 
  activeVersion, 
  entityLabel,
  featureGroupLabel
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [selectedFeature, setSelectedFeature] = useState(null);
  const [editFeatures, setEditFeatures] = useState([]);
  const [error, setError] = useState(null);
  const [validationErrors, setValidationErrors] = useState([]);
  const [showSuccessNotification, setShowSuccessNotification] = useState(false);
  const [showErrorDialog, setShowErrorDialog] = useState(false);

  const { user } = useAuth();
  
  const versionNumber = parseInt(activeVersion, 10);
  const featureData = features[versionNumber];
  const labels = featureData ? featureData.labels.split(',') : [];
  const defaultValues = featureData ? featureData['default-values'] : [];
  const sourceBasePaths = featureData ? featureData['source-base-paths'] || [] : [];
  const sourceDataColumns = featureData ? featureData['source-data-columns'] || [] : [];
  const storageProviders = featureData ? featureData['storage-providers'] || [] : [];
  const stringLengths = featureData ? featureData['string-lengths'] || [] : [];
  const vectorLengths = featureData ? featureData['vector-lengths'] || [] : [];

  // Filter the labels and defaultValues based on the search term
  const filteredData = labels
    .map((label, index) => ({
      label,
      defaultValue: defaultValues[index],
      sourceBasePath: sourceBasePaths[index]|| 'N/A',
      sourceDataColumn: sourceDataColumns[index]|| 'N/A',
      storageProvider: storageProviders[index] || 'N/A',
      stringLength: stringLengths[index],
      vectorLength: vectorLengths[index],
    }))
    .filter(item => 
      item.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
      item.defaultValue.toLowerCase().includes(searchTerm.toLowerCase())
    );

  const handleInfoClick = (feature) => {
    setSelectedFeature(feature);
    setShowModal(true);
  };

  const handleCloseModal = () => {
    setShowModal(false);
  };

  const handleOpenEditModal = () => {
    const initialEditFeatures = filteredData.map(feature => ({
      labels: feature.label,
      'default-values': feature.defaultValue === '-' ? '' : feature.defaultValue,
      'source-base-paths': feature.sourceBasePath === '-' ? '' : feature.sourceBasePath,
      'source-data-columns': feature.sourceDataColumn === '-' ? '' : feature.sourceDataColumn,
      'storage-providers': feature.storageProvider === '-' ? '' : feature.storageProvider,
      'string-lengths': feature.stringLength === '-' ? '' : feature.stringLength,
      'vector-lengths': feature.vectorLength === '-' ? '' : feature.vectorLength,
    }));
    setEditFeatures(initialEditFeatures);
    setShowEditModal(true);
  };

  const handleEditChange = (index, field, value) => {
    const updatedFeatures = [...editFeatures];
    updatedFeatures[index][field] = value;
    setEditFeatures(updatedFeatures);
  };

  const handleCloseSuccessNotification = () => {
    setShowSuccessNotification(false);
  };

  const handleCloseErrorDialog = () => {
    setShowErrorDialog(false);
    setError(null);
  };

  const handleSubmitEdit = async () => {
    // Validate required fields
    const errors = [];
    
    editFeatures.forEach((feature, index) => {
      // Check if string-lengths is undefined, null, or empty string, but allow '0'
      if (feature['string-lengths'] === undefined || 
          feature['string-lengths'] === null || 
          feature['string-lengths'].toString().trim() === '') {
        errors.push({ index, field: 'string-lengths', message: 'String Length is required' });
      }
      
      // Check if vector-lengths is undefined, null, or empty string, but allow '0'
      if (feature['vector-lengths'] === undefined || 
          feature['vector-lengths'] === null || 
          feature['vector-lengths'].toString().trim() === '') {
        errors.push({ index, field: 'vector-lengths', message: 'Vector Length is required' });
      }
    });
    
    if (errors.length > 0) {
      setValidationErrors(errors);
      return;
    }
    
    // Clear any existing validation errors
    setValidationErrors([]);
    
    const token = user.token;
    try {
        const payload = {
          'entity-label': entityLabel,
          'feature-group-label': featureGroupLabel,
          'features': editFeatures.map(feature => ({
            labels: feature.labels,
            'default-values': feature['default-values'],
            'source-base-path': feature['source-base-paths'],
            'source-data-column': feature['source-data-columns'],
            'storage-provider': feature['storage-providers'],
            'string-length': feature['string-lengths'].toString(),
            'vector-length': feature['vector-lengths'].toString(),
          }))
        };
        
        console.log('Edit features payload:', payload);
      
      await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/edit-features`, 
        payload, 
        {
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      setShowEditModal(false);
      setShowSuccessNotification(true);
    } catch (err) {
      console.error('Edit features error:', err);
      setError(
        err.response?.data?.message || 
        'Failed to edit features. Please try again later.'
      );
      setShowErrorDialog(true);
    }
  };

  // Helper function to check if a field has validation error
  const hasError = (index, field) => {
    return validationErrors.some(error => error.index === index && error.field === field);
  };
  
  // Helper function to get error message
  const getErrorMessage = (index, field) => {
    const error = validationErrors.find(error => error.index === index && error.field === field);
    return error ? error.message : '';
  };

  const modalStyle = {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    bgcolor: 'background.paper',
    boxShadow: 24,
    p: 4,
    borderRadius: '8px',
    width: '80%',
    maxWidth: '1400px',
    maxHeight: '80%',
    overflowY: 'auto',
  };
  const viewModalStyle = {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    bgcolor: 'background.paper',
    boxShadow: 24,
    p: 4,
    borderRadius: '8px',
    minWidth: '40%',
    maxWidth: '800px',
    maxHeight: '80%',
    overflowY: 'auto',
  };

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
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <Typography
            variant="h6"
            style={{
              fontWeight: 'bold',
              color: '#24031e',
            }}
          >
            Features
          </Typography>
          <Button 
            variant="contained" 
            sx={{
              backgroundColor: '#35072c',
              '&:hover': {
                backgroundColor: '#35072c',
              },
            }}
            startIcon={<EditIcon />}
            onClick={handleOpenEditModal}
          >
             Edit Features
          </Button>
        </div>

        {/* Search Input */}
        <div style={{ marginBottom: '20px' }}>
          <TextField
            placeholder="Search Features"
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

        <TableContainer component={Paper} style={{ maxHeight: '800px', overflowY: 'auto' }}>
          <Table stickyHeader>
            <TableHead>
              <TableRow>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>
                  Feature Name
                </TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold' }}>
                  Feature Default Value
                </TableCell>
                <TableCell sx={{ backgroundColor: '#E6EBF2', fontWeight: 'bold', width: '60px' }}>
                  Info
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredData.map((item, index) => (
                <TableRow key={index}>
                  <TableCell>{item.label}</TableCell>
                  <TableCell>{item.defaultValue}</TableCell>
                  <TableCell>
                    <InfoIcon
                      onClick={() => handleInfoClick(item)}
                      style={{
                        color: '#24031e',
                        cursor: 'pointer',
                        transition: 'color 0.2s ease',
                      }}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </CardContent>

      {/* Info Modal */}
      <Modal
        open={showModal}
        onClose={handleCloseModal}
        aria-labelledby="feature-info-modal"
        aria-describedby="feature-details"
      >
        <Box sx={viewModalStyle}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
            <Typography id="feature-info-modal" variant="h6" component="h2" sx={{ fontWeight: 'bold' }}>
              Feature Details
            </Typography>
            <IconButton onClick={handleCloseModal}>
              <CloseIcon />
            </IconButton>
          </div>
          
          {selectedFeature && (
            <TableContainer component={Paper} sx={{ mb: 2 }}>
              <Table>
                <TableBody>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Feature Name</TableCell>
                    <TableCell>{selectedFeature.label}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Default Value</TableCell>
                    <TableCell>{selectedFeature.defaultValue}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Source Base Path</TableCell>
                    <TableCell>{selectedFeature.sourceBasePath}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Source Data Column</TableCell>
                    <TableCell>{selectedFeature.sourceDataColumn}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Storage Provider</TableCell>
                    <TableCell>{selectedFeature.storageProvider}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Vector Length</TableCell>
                    <TableCell>{selectedFeature.vectorLength}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>String Length</TableCell>
                    <TableCell>{selectedFeature.stringLength}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Box>
      </Modal>

      {/*  Edit Modal */}
      <Modal
        open={showEditModal}
        onClose={() => setShowEditModal(false)}
        aria-labelledby="feature-edit-modal"
      >
        <Box sx={modalStyle}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
            <Typography id="feature-edit-modal" variant="h6" component="h2" sx={{ fontWeight: 'bold' }}>
               Edit Features
            </Typography>
            <IconButton onClick={() => setShowEditModal(false)}>
              <CloseIcon />
            </IconButton>
          </div>
          
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Feature Name</TableCell>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Default Value</TableCell>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Source Base Path</TableCell>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Source Data Column</TableCell>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Storage Provider</TableCell>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>String Length *</TableCell>
                  <TableCell sx={{ fontWeight: 'bold', backgroundColor: '#f5f5f5' }}>Vector Length *</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {editFeatures.map((feature, index) => (
                  <TableRow key={index}>
                    <TableCell>{feature.labels}</TableCell>
                    <TableCell>
                      <TextField
                        value={feature['default-values']}
                        onChange={(e) => handleEditChange(index, 'default-values', e.target.value)}
                        fullWidth
                        variant="outlined"
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <TextField
                        value={feature['source-base-paths']}
                        onChange={(e) => handleEditChange(index, 'source-base-paths', e.target.value)}
                        fullWidth
                        variant="outlined"
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <TextField
                        value={feature['source-data-columns']}
                        onChange={(e) => handleEditChange(index, 'source-data-columns', e.target.value)}
                        fullWidth
                        variant="outlined"
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <TextField
                        select
                        value={feature['storage-providers']}
                        onChange={(e) => handleEditChange(index, 'storage-providers', e.target.value)}
                        fullWidth
                        variant="outlined"
                        size="small"
                      >
                        <MenuItem value="PARQUET_GCS">PARQUET_GCS</MenuItem>
                        <MenuItem value="PARQUET_S3">PARQUET_S3</MenuItem>
                        <MenuItem value="PARQUET_ADLS">PARQUET_ADLS</MenuItem>
                        <MenuItem value="DELTA_GCS">DELTA_GCS</MenuItem>
                        <MenuItem value="DELTA_S3">DELTA_S3</MenuItem>
                        <MenuItem value="DELTA_ADLS">DELTA_ADLS</MenuItem>
                        <MenuItem value="TABLE">TABLE</MenuItem>
                      </TextField>
                    </TableCell>
                    <TableCell>
                      <TextField
                        value={feature['string-lengths']}
                        onChange={(e) => handleEditChange(index, 'string-lengths', e.target.value)}
                        fullWidth
                        variant="outlined"
                        size="small"
                        required
                        error={hasError(index, 'string-lengths')}
                        helperText={getErrorMessage(index, 'string-lengths')}
                      />
                    </TableCell>
                    <TableCell>
                      <TextField
                        value={feature['vector-lengths']}
                        onChange={(e) => handleEditChange(index, 'vector-lengths', e.target.value)}
                        fullWidth
                        variant="outlined"
                        size="small"
                        required
                        error={hasError(index, 'vector-lengths')}
                        helperText={getErrorMessage(index, 'vector-lengths')}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          
          <Box sx={{ mt: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography sx={{ color: 'text.secondary' }}>* Required fields</Typography>
            <Button 
              variant="contained" 
              sx={{
                backgroundColor: '#35072c',
              }}
              onClick={handleSubmitEdit}
            >
              Save Changes
            </Button>
          </Box>
        </Box>
      </Modal>

      {/* Success Notification Dialog */}
      <Dialog
        open={showSuccessNotification}
        onClose={handleCloseSuccessNotification}
        maxWidth="sm"
      >
        <DialogTitle>
          Success
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CheckCircleOutlineIcon sx={{ color: 'green' }} />
            <Typography>
              Your features edit request has been successfully processed.
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={handleCloseSuccessNotification}
            sx={{
              backgroundColor: '#35072c',
              color: 'white',
              '&:hover': {
                backgroundColor: '#24031e',
              },
            }}
          >
            OK
          </Button>
        </DialogActions>
      </Dialog>

      {/* Error Dialog */}
      <Dialog
        open={showErrorDialog}
        onClose={handleCloseErrorDialog}
        maxWidth="sm"
      >
        <DialogTitle sx={{ color: '#d32f2f' }}>
          Error
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CloseIcon sx={{ color: '#d32f2f' }} />
            <Typography>
              {error}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={handleCloseErrorDialog}
            sx={{
              backgroundColor: '#d32f2f',
              color: 'white',
              '&:hover': {
                backgroundColor: '#b71c1c',
              },
            }}
          >
            OK
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  );
};

export default FeatureList;
