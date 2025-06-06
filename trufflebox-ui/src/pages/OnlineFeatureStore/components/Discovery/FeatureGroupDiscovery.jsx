import React, { useState, useEffect, useMemo, useCallback } from 'react';
import axios from 'axios';
import { useAuth } from '../../../Auth/AuthContext';
import { 
  Box, 
  Typography, 
  CircularProgress, 
  TextField, 
  InputAdornment, 
  IconButton, 
  Paper, 
  Button, 
  Dialog, 
  DialogTitle, 
  DialogContent, 
  DialogActions, 
  FormControl, 
  InputLabel, 
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  MenuItem,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import SearchIcon from '@mui/icons-material/Search';
import ViewInArIcon from '@mui/icons-material/ViewInAr';
import EditIcon from '@mui/icons-material/Edit';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';

import * as URL_CONSTANTS from '../../../../config';

const FeatureGroupDiscovery = ({ entityId, onFeatureGroupClick }) => {
  const [selectedFeatureGroup, setSelectedFeatureGroup] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [featureGroups, setFeatureGroups] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const { user } = useAuth();
  // New state for success notification
  const [showSuccessNotification, setShowSuccessNotification] = useState(false);

  const [editFormData, setEditFormData] = useState({
    ttlInSeconds: '',
    inMemoryCacheEnabled: '',
    distributedCacheEnabled: '',
    layoutVersion: ''
  });

  const fetchFeatureGroups = useCallback(() => {
    if (entityId) {
      setLoading(true);
      setError(null);
      const token = user.token;
      axios
        .get(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/retrieve-feature-groups?entity=${entityId}`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        })
        .then((response) => {
          const normalizedFeatureGroups = response.data.map((featureGroup) => ({
            ...featureGroup,
            activeVersion: featureGroup['active-version'],
            features: featureGroup.features,
          }));
          setFeatureGroups(normalizedFeatureGroups);
        })
        .catch((error) => {
          console.error('Error fetching feature groups:', error);
          setError('Failed to fetch feature groups. Please try again later.');
        })
        .finally(() => setLoading(false));
    }
  }, [entityId, user.token]);

  useEffect(() => {
    fetchFeatureGroups();
  }, [fetchFeatureGroups]);

  const handleFeatureGroupClick = useCallback((featureGroup) => {
    setSelectedFeatureGroup(featureGroup);
    onFeatureGroupClick(featureGroup);
  }, [onFeatureGroupClick]);

  const handleInfoClick = (featureGroup) => {
    setSelectedFeatureGroup(featureGroup);
    setShowModal(true);
  };

  const handleEditClick = (featureGroup) => {
    setSelectedFeatureGroup(featureGroup);
    setEditFormData({
      ttlInSeconds: featureGroup['ttl-in-seconds'],
      inMemoryCacheEnabled: !!featureGroup['in-memory-cache-enabled'],
      distributedCacheEnabled: !!featureGroup['distributed-cache-enabled'],
      layoutVersion: featureGroup['layout-version']
    });
    
    setShowEditModal(true);
  };

  const handleEditSubmit = () => {
    const token = user.token;
    const payload = {
      'entity-label': selectedFeatureGroup['entity-label'],
      'fg-label': selectedFeatureGroup['feature-group-label'],
      'ttl-in-seconds': parseInt(editFormData.ttlInSeconds),
      'in-memory-cache-enabled': editFormData.inMemoryCacheEnabled,
      'distributed-cache-enabled': editFormData.distributedCacheEnabled,
      'layout-version': 1,
    };

    axios
      .post(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/edit-feature-group`, payload, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
      })
      .then(() => {
        fetchFeatureGroups();
        setShowEditModal(false);
        // Show success notification
        setShowSuccessNotification(true);
      })
      .catch((error) => {
        console.error('Error editing feature group:', error);
        setError('Failed to edit feature group. Please try again later.');
      });
  };

  const closeModal = () => setShowModal(false);
  const closeEditModal = () => setShowEditModal(false);
  // Handler to close success notification
  const handleCloseSuccessNotification = () => {
    setShowSuccessNotification(false);
  };

  // Memoize feature group list rendering to optimize performance
  const renderedFeatureGroups = useMemo(() => {
    // Filter feature groups based on the search term
    const filteredFeatureGroups = featureGroups.filter((featureGroup) =>
      featureGroup['feature-group-label'].toLowerCase().includes(searchTerm.toLowerCase())
    );

    return filteredFeatureGroups.map((featureGroup, index) => (
      <Paper
        key={index}
        elevation={selectedFeatureGroup?.id === featureGroup.id ? 3 : 1}
        sx={{
          height: '50px',
          backgroundColor: selectedFeatureGroup?.id === featureGroup.id ? '#e6f7ff' : '#ffffff',
          margin: '5px 0',
          padding: '10px',
          position: 'relative',
          border: selectedFeatureGroup?.id === featureGroup.id ? '2px solid #24031e' : '1px solid #ddd',
          borderRadius: '5px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          cursor: 'pointer',
          transition: 'box-shadow 0.3s ease, transform 0.2s ease',
        }}
        onClick={() => handleFeatureGroupClick(featureGroup)}
      >
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <ViewInArIcon sx={{ marginRight: '10px', color: '#24031e' }} />
          <Typography sx={{ 
            fontWeight: '500', 
            fontSize: '16px',
            whiteSpace: 'normal',
            wordBreak: 'break-all',
            wordWrap: 'break-word',
          }}>
            {featureGroup['feature-group-label']}
          </Typography>
        </Box>
        <Box sx={{
          display: 'flex',
          alignItems: 'center'
        }}>
          <IconButton 
            onClick={(e) => {
              e.stopPropagation();
              handleInfoClick(featureGroup);
            }}
            size="small"
          >
            <InfoIcon sx={{
              color: '#24031e',
              transition: 'color 0.2s ease',
            }} />
          </IconButton>
          <IconButton
            onClick={(e) => {
              e.stopPropagation();
              handleEditClick(featureGroup);
            }}
            size="small"
          >
            <EditIcon sx={{
              color: '#24031e',
              transition: 'color 0.2s ease',
            }} />
          </IconButton>
        </Box>
      </Paper>
    ));
  }, [featureGroups, selectedFeatureGroup, searchTerm, handleFeatureGroupClick]);

  return (
    <Box sx={{ position: 'relative', height: '100%' }}>
      <Box
        sx={{
          position: 'sticky',
          top: 0,
          backgroundColor: '#35072c',
          width: '435px',
          height: '50px',
          margin: '5px',
          padding: '10px',
          fontSize: '20px',
          fontWeight: 'bold',
          color: '#ffffff',
          zIndex: 1000,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderRadius: '5px',
        }}
      >
        Feature Groups
      </Box>

      {/* Search Input */}
      <Box sx={{ padding: '10px' }}>
        <TextField
          fullWidth
          variant="outlined"
          placeholder="Search Feature Groups"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      {/* Feature Group List */}
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          overflowY: 'auto',
          width: '450px',
          height: 'calc(100vh - 120px)',
          backgroundColor: '#f5f5f5',
          borderRadius: '5px',
          padding: '5px',
        }}
      >
        {loading ? (
          <Box sx={{ textAlign: 'center', padding: '20px' }}>
            <CircularProgress />
          </Box>
        ) : error ? (
          <Typography color="error" sx={{ textAlign: 'center', padding: '20px' }}>
            {error}
          </Typography>
        ) : (
          renderedFeatureGroups
        )}
      </Box>

      {/* Modal for Feature Group Details */}
      <Dialog 
        open={showModal} 
        onClose={closeModal} 
        maxWidth="md" 
        fullWidth
      >
        <DialogTitle>
          Feature Group Details
          <IconButton
            aria-label="close"
            onClick={closeModal}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          {selectedFeatureGroup && (
            <TableContainer component={Paper}>
              <Table>
                <TableBody>
                  <TableRow>
                    <TableCell><strong>Label</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['feature-group-label']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Active Version</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['active-version']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Entity Label</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['entity-label']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Store ID</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['store-id']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Data Type</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['data-type']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>TTL (Seconds)</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['ttl-in-seconds']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Job Name</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['job-id']}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>In-Memory Cache Enabled</strong></TableCell>
                    <TableCell>
                      {selectedFeatureGroup['in-memory-cache-enabled'] ? 'Yes' : 'No'}
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Distributed Cache Enabled</strong></TableCell>
                    <TableCell>
                      {selectedFeatureGroup['distributed-cache-enabled'] ? 'Yes' : 'No'}
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Layout Version</strong></TableCell>
                    <TableCell>{selectedFeatureGroup['layout-version']}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </DialogContent>
      </Dialog>

      {/* Edit Modal */}
      <Dialog 
        open={showEditModal} 
        onClose={closeEditModal} 
        maxWidth="sm" 
        fullWidth
      >
        <DialogTitle>
          Edit Feature Group: {selectedFeatureGroup?.['feature-group-label']}
          <IconButton
            aria-label="close"
            onClick={closeEditModal}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label="TTL (Seconds)"
              type="number"
              fullWidth
              value={editFormData.ttlInSeconds}
              onChange={(e) => setEditFormData(prev => ({
                ...prev,
                ttlInSeconds: e.target.value
              }))}
            />

            <FormControl fullWidth>
              <InputLabel>In-Memory Cache Enabled</InputLabel>
              <Select
                value={editFormData.inMemoryCacheEnabled}
                label="In-Memory Cache Enabled"
                onChange={(e) => setEditFormData(prev => ({
                  ...prev,
                  inMemoryCacheEnabled: e.target.value === 'true'
                }))}
              >
                <MenuItem value="true">True</MenuItem>
                <MenuItem value="false">False</MenuItem>
              </Select>
            </FormControl>

            <FormControl fullWidth>
              <InputLabel>Distributed Cache Enabled</InputLabel>
              <Select
                value={editFormData.distributedCacheEnabled}
                label="Distributed Cache Enabled"
                onChange={(e) => setEditFormData(prev => ({
                  ...prev,
                  distributedCacheEnabled: e.target.value === 'true'
                }))}
              >
                <MenuItem value="true">True</MenuItem>
                <MenuItem value="false">False</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Layout Version"
              value="1"
              fullWidth
              disabled
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={handleEditSubmit}
            sx={{
              backgroundColor: '#35072c',
              '&:hover': {
                backgroundColor: '#35072c',
              },
            }}
            variant="contained"
          >
            Save Changes
          </Button>
        </DialogActions>
      </Dialog>

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
              Your feature group edit request has been successfully registered.
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
    </Box>
  );
};

export default FeatureGroupDiscovery;
