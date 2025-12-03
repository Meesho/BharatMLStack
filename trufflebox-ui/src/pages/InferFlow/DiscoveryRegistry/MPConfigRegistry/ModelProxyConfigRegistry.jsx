import React, { useState, useEffect } from 'react';
import { Box, Snackbar, Alert, Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography } from '@mui/material';
import GenericMPConfigRegistryTable from './GenericMPConfigRegistryTable';
import OnboardMPConfigModal from './OnboardMPConfigModal';
import EditMPConfigModal from './EditMPConfigModal';
import CloneMPConfigModal from './CloneMPConfigModal';
import PromoteMPConfigModal from './PromoteMPConfigModal';
import ScaleUpMPConfigModal from './ScaleUpMPConfigModal';
import MPConfigTestingModal from './MPConfigTestingModal';
import ViewConfigDetailsModal from './ViewConfigDetailsModal';
import ProductionCredentialModal from '../../../../common/ProductionCredentialModal';
import { useAuth } from '../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';
import * as URL_CONSTANTS from '../../../../config';
import axios from 'axios';

const ModelProxyConfigRegistry = () => {
  const { user, hasPermission, permissions } = useAuth();
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.MP_CONFIG;
  const isPermissionsLoaded = permissions !== null;
  const [loading, setLoading] = useState(true);
  const [configData, setConfigData] = useState([]);
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success'
  });
  const [onboardModalOpen, setOnboardModalOpen] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState({
    open: false,
    config: null
  });
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const [cloneModalOpen, setCloneModalOpen] = useState(false);
  const [promoteModalOpen, setPromoteModalOpen] = useState(false);
  const [scaleUpModalOpen, setScaleUpModalOpen] = useState(false);
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [showProdCredentialModal, setShowProdCredentialModal] = useState(false);
  const [selectedConfigForPromotion, setSelectedConfigForPromotion] = useState(null);
  const [viewDetailsModal, setViewDetailsModal] = useState({
    open: false,
    config: null
  });

  useEffect(() => {
    if (isPermissionsLoaded && hasPermission(service, screenType, ACTIONS.VIEW)) {
      fetchConfigs();
    } else if (isPermissionsLoaded) {
      setLoading(false);
    }
  }, [isPermissionsLoaded, hasPermission, service, screenType]);

  const fetchConfigs = async () => {
    try {
      setLoading(true);
      const response = await axios.get(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/inferflow-config-discovery/configs`, {
        headers: {
          Authorization: `Bearer ${user?.token}`,
          'Content-Type': 'application/json'
        },
      });
      
      if (response.data.error) {
        console.log(response.data.error);
      }
      
      // Transform the API response to match table expectations
      const transformedData = Array.isArray(response.data.data) 
        ? response?.data?.data?.map(item => ({
            ...item,
            // Add missing fields that table expects
            active: item.deployable_running_status || false,
            created_by: item.created_by || 'Unknown',
            updated_by: item.updated_by || 'Unknown',
            created_at: item.created_at || new Date().toISOString(),
            updated_at: item.updated_at || new Date().toISOString(),
            // Transform test_results to match expected format
            test_results: {
              tested: item.test_results?.tested || false,
              status: item.test_results?.tested ? 'PASSED' : 'NOT_TESTED',
              message: item.test_results?.message || 'Not Tested'
            }
          }))
        : [];
        
      setConfigData(transformedData);
    } catch (error) {
      setConfigData([]);
      showSnackbar(error.message || 'Failed to fetch configurations', 'error');
      console.log('Error fetching configurations:', error);
    } finally {
      setLoading(false);
    }
  };

  const showSnackbar = (message, severity = 'success') => {
    setSnackbar({
      open: true,
      message,
      severity
    });
  };

  const handleCloseSnackbar = () => {
    setSnackbar(prev => ({ ...prev, open: false }));
  };

  const handleOnboard = () => {
    if (!hasPermission(service, screenType, ACTIONS.ONBOARD)) {
      showSnackbar('You do not have permission to onboard configurations', 'error');
      return;
    }
    setOnboardModalOpen(true);
  };

  const handleOnboardSuccess = (message) => {
    showSnackbar(message);
    fetchConfigs();
  };

  const handleEditConfig = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.EDIT)) {
      showSnackbar('You do not have permission to edit configurations', 'error');
      return;
    }
    setSelectedConfig(row);
    setEditModalOpen(true);
  };

  const handleDeleteConfig = async (row) => {
    try {
      const response = await axios.patch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/inferflow-config-registry/delete?id=${row.config_id}`, {}, {
        headers: {
          Authorization: `Bearer ${user?.token}`,
          'Content-Type': 'application/json'
        },
      });
      if (response.data.error) {
        console.log(response.data.error);
      }
      
      const successMessage = typeof response.data.data === 'string' 
        ? response.data.data 
        : response.data.data?.message || 'Configuration deleted successfully';
      
      showSnackbar(successMessage);
      fetchConfigs();
      setDeleteConfirmation({ open: false, config: null });
    } catch (error) {
      showSnackbar(error.message, 'error');
    }
  };

  const handleDeleteConfirmationOpen = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.DEACTIVATE)) {
      showSnackbar('You do not have permission to delete configurations', 'error');
      return;
    }
    setDeleteConfirmation({ open: true, config: row });
  };

  const handleDeleteConfirmationClose = () => {
    setDeleteConfirmation({ open: false, config: null });
  };

  const handlePromoteConfig = async (row) => {
    if (!hasPermission(service, screenType, ACTIONS.PROMOTE)) {
      showSnackbar('You do not have permission to promote configurations', 'error');
      return;
    }
    
    // Check if configuration has been tested
    if (!row.test_results?.tested) {
      showSnackbar(`Testing is yet to be done for: ${row.config_id}`, 'error');
      return;
    }
    
    setSelectedConfigForPromotion(row);
    setShowProdCredentialModal(true);
  };

  const handleProductionCredentialSuccess = async (prodCredentials) => {
    setShowProdCredentialModal(false);
    
    if (!selectedConfigForPromotion) {
      showSnackbar('No configuration selected for promotion', 'error');
      return;
    }

    window.prodCredentials = prodCredentials;
    setSelectedConfig(selectedConfigForPromotion);
    setPromoteModalOpen(true);
  };

  const handleProductionCredentialClose = () => {
    setShowProdCredentialModal(false);
    setSelectedConfigForPromotion(null);
    delete window.prodCredentials;
  };

  const handleScaleUpConfig = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.SCALE_UP)) {
      showSnackbar('You do not have permission to scale up configurations', 'error');
      return;
    }
    
    // Check if configuration has been tested
    if (!row.test_results?.tested) {
      showSnackbar(`Testing is yet to be done for: ${row.config_id}`, 'error');
      return;
    }
    
    setSelectedConfig(row);
    setScaleUpModalOpen(true);
  };

  const handleCloneConfig = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.CLONE)) {
      showSnackbar('You do not have permission to clone configurations', 'error');
      return;
    }
    setSelectedConfig(row);
    setCloneModalOpen(true);
  };

  const handleTestConfig = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.TEST)) {
      showSnackbar('You do not have permission to test configurations', 'error');
      return;
    }
    setSelectedConfig(row);
    setTestModalOpen(true);
  };

  const handleTestComplete = (configId, status) => {
    fetchConfigs();
    showSnackbar(`Test ${status} for config: ${configId}`, status === 'success' ? 'success' : 'error');
  };

  const handleViewDetails = (row) => {
    setViewDetailsModal({
      open: true,
      config: row
    });
  };


  if (!isPermissionsLoaded) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <Alert severity="info">Loading permissions...</Alert>
      </Box>
    );
  }

  if (!hasPermission(service, screenType, ACTIONS.VIEW)) {
    return (
      <Box sx={{ padding: '1.5rem' }}>
        <Alert severity="warning">
          <Typography variant="h6">Access Denied</Typography>
          <Typography>You do not have permission to view InferFlow configurations.</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <GenericMPConfigRegistryTable
        data={configData}
        loading={loading}
        onViewDetails={handleViewDetails}
        onEditConfig={handleEditConfig}
        onDeleteConfig={handleDeleteConfirmationOpen}
        onPromoteConfig={handlePromoteConfig}
        onScaleUpConfig={handleScaleUpConfig}
        onCloneConfig={handleCloneConfig}
        onTestConfig={handleTestConfig}
        onOnboard={handleOnboard}
      />

      <OnboardMPConfigModal
        open={onboardModalOpen}
        onClose={() => setOnboardModalOpen(false)}
        onSuccess={handleOnboardSuccess}
      />

      <EditMPConfigModal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        onSuccess={(message) => {
          showSnackbar(message);
          fetchConfigs();
        }}
        configData={selectedConfig}
      />

      <CloneMPConfigModal
        open={cloneModalOpen}
        onClose={() => setCloneModalOpen(false)}
        onSuccess={(message, type = 'success') => {
          showSnackbar(message, type);
          fetchConfigs();
        }}
        configData={selectedConfig}
      />

      <PromoteMPConfigModal
        open={promoteModalOpen}
        onClose={() => setPromoteModalOpen(false)}
        onSuccess={(message) => {
          showSnackbar(message);
          fetchConfigs();
        }}
        configData={selectedConfig}
      />

      <ScaleUpMPConfigModal
        open={scaleUpModalOpen}
        onClose={() => setScaleUpModalOpen(false)}
        onSuccess={(message) => {
          showSnackbar(message);
          fetchConfigs();
        }}
        configData={selectedConfig}
      />

      <ViewConfigDetailsModal
        open={viewDetailsModal.open}
        onClose={() => setViewDetailsModal({ open: false, config: null })}
        configData={viewDetailsModal.config}
      />

      <MPConfigTestingModal
        open={testModalOpen}
        onClose={() => setTestModalOpen(false)}
        configData={selectedConfig}
        onTestComplete={handleTestComplete}
      />

      <Dialog
        open={deleteConfirmation.open}
        onClose={handleDeleteConfirmationClose}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          Confirm Delete
        </DialogTitle>
        <DialogContent sx={{ mt: 2 }}>
          <Typography>
            Are you sure you want to delete the configuration with ID: {deleteConfirmation.config?.config_id}?
          </Typography>
          <Typography variant="body2" color="error" sx={{ mt: 1 }}>
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button 
            onClick={handleDeleteConfirmationClose}
            variant="outlined"
          >
            Cancel
          </Button>
          <Button 
            onClick={() => handleDeleteConfig(deleteConfirmation.config)}
            variant="contained" 
            color="error"
            autoFocus
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Production Credential Modal */}
      <ProductionCredentialModal
        open={showProdCredentialModal}
        onClose={handleProductionCredentialClose}
        onSuccess={handleProductionCredentialSuccess}
        title="Production Credential Verification"
        description="Please enter your production credentials to proceed with the InferFlow configuration promotion."
      />

      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseSnackbar} severity={snackbar.severity} sx={{ width: '100%' }}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default ModelProxyConfigRegistry;