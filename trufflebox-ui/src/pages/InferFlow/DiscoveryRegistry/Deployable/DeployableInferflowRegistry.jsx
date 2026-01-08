import React, { useState, useEffect } from 'react';
import { Box, Typography, Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField, CircularProgress, Snackbar, Alert, IconButton } from '@mui/material';
import GenericDeployableTable from './GenericDeployableTable';
import { useAuth } from '../../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';
import axios from 'axios';
import * as URL_CONSTANTS from '../../../../config';
import InfoIcon from '@mui/icons-material/Info';
import CloseIcon from '@mui/icons-material/Close';

const DeployableInferflowRegistry = () => {
  const { user, hasPermission, permissions } = useAuth();
  
  const service = SERVICES.InferFlow;
  const screenType = SCREEN_TYPES.InferFlow.DEPLOYABLE;
  const isPermissionsLoaded = permissions !== null;
  const [deployables, setDeployables] = useState([]);
  const [loading, setLoading] = useState(true);
  const [openOnboardDialog, setOpenOnboardDialog] = useState(false);
  const [openEditDialog, setOpenEditDialog] = useState(false);
  const [openDetailsDialog, setOpenDetailsDialog] = useState(false);
  const [formData, setFormData] = useState({
    appName: '',
    service: 'InferFlow',
  });
  const [selectedDeployable, setSelectedDeployable] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  useEffect(() => {
    if (isPermissionsLoaded && hasPermission(service, screenType, ACTIONS.VIEW)) {
      fetchDeployables();
    } else if (isPermissionsLoaded) {
      setLoading(false);
    }
  }, [isPermissionsLoaded, hasPermission, service, screenType]);

  const fetchDeployables = async () => {
    setLoading(true);
    setSnackbar({
      open: false,
      message: '',
      severity: 'success'
    });
    
    const token = user?.token;
    
    try {
      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=InferFlow`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      if (!response.data.data || !Array.isArray(response.data.data)) {
        console.error('Invalid response structure:', response.data);
        setSnackbar({
          open: true,
          message: 'Invalid response format from server',
          severity: 'error'
        });
        return;
      }

      // Transform API response to match our table component's expected format
      const formattedData = response.data.data.map(deployable => ({
        Id: deployable.id,
        Name: deployable.name,
        Host: deployable.host,
        Service: deployable.service,
        CpuRequest: deployable.cpuRequest || '',
        CpuLimit: deployable.cpuLimit || '',
        MemRequest: deployable.memoryRequest || '',
        MemLimit: deployable.memoryLimit || '',
        MinReplica: '',
        MaxReplica: '',
        DashboardLink: deployable.monitoring_url || '',
        DeployableRunningStatus: deployable.deployable_health === "DEPLOYMENT_REASON_ARGO_APP_HEALTHY",
        CreatedBy: deployable.created_by,
        UpdatedBy: deployable.updated_by || '',
        CreatedAt: deployable.created_at,
        UpdatedAt: deployable.updated_at,
        DeployableHealth: deployable.deployable_health,
        raw: deployable
      }));
      
      setDeployables(formattedData);
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to load deployables',
        severity: 'error'
      });
      console.log('Error fetching deployables:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleOnboardDeployable = () => {
    if (!hasPermission(service, screenType, ACTIONS.ONBOARD)) {
      setSnackbar({
        open: true,
        message: 'You do not have permission to onboard deployables',
        severity: 'error'
      });
      return;
    }
    
    setFormData({
      appName: '',
      service: 'InferFlow',
    });
    setOpenOnboardDialog(true);
  };

  const handleEditDeployable = (deployable) => {
    if (!hasPermission(service, screenType, ACTIONS.EDIT)) {
      setSnackbar({
        open: true,
        message: 'You do not have permission to edit deployables',
        severity: 'error'
      });
      return;
    }
    
    setSelectedDeployable(deployable);
    setFormData({
      appName: deployable.Name,
      service: deployable.Service,
    });
    setOpenEditDialog(true);
  };

  const handleViewDetails = (deployable) => {
    setSelectedDeployable(deployable);
    setOpenDetailsDialog(true);
  };

  const handleFormChange = (e) => {
    const { name, value } = e.target;
    const trimmedValue = typeof value === 'string' ? value.trim() : value;
    setFormData(prev => ({ ...prev, [name]: trimmedValue }));
  };

  const handleOnboardSubmit = async () => {
    if (!formData.appName || !formData.service) {
      setSnackbar({
        open: true,
        message: 'Please fill all required fields',
        severity: 'error'
      });
      return;
    }

    setSubmitting(true);
    const token = user?.token;
    
    try {
      const payload = {
        appName: formData.appName,
        service_name: "inferflow",
        created_by: user?.email || 'unknown'
      };

      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-registry/deployables`,
        payload,
        {
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      
      setSnackbar({
        open: true,
        message: 'Deployable registered successfully',
        severity: 'success'
      });
      
      setOpenOnboardDialog(false);
      fetchDeployables(); // Refresh data
      
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to onboard deployable',
        severity: 'error'
      });
      console.log('Error onboarding deployable:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleEditSubmit = async () => {
    if (!formData.appName || !formData.service) {
      setSnackbar({
        open: true,
        message: 'Please fill all required fields',
        severity: 'error'
      });
      return;
    }

    setSubmitting(true);
    const token = user?.token;
    
    try {
      const payload = {
        appName: formData.appName,
        service_name: "inferflow",
        updated_by: user?.email || 'unknown'
      };

      const response = await axios.put(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-registry/deployables/${selectedDeployable.Id}`,
        payload,
        {
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      
      setSnackbar({
        open: true,
        message: 'Deployable updated successfully',
        severity: 'success'
      });
      
      setOpenEditDialog(false);
      fetchDeployables(); // Refresh data
      
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to update deployable',
        severity: 'error'
      });
      console.log('Error updating deployable:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleCloseSnackbar = () => {
    setSnackbar(prev => ({ ...prev, open: false }));
  };

  if (!isPermissionsLoaded) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!hasPermission(service, screenType, ACTIONS.VIEW)) {
    return (
      <Box sx={{ padding: '1.5rem' }}>
        <Alert severity="warning">
          <Typography variant="h6">Access Denied</Typography>
          <Typography>You do not have permission to view deployables.</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ padding: '1.5rem' }}>
      <GenericDeployableTable 
        data={deployables}
        loading={loading}
        onViewDetails={handleViewDetails}
        onEditDeployable={handleEditDeployable}
        onOnboardDeployable={handleOnboardDeployable}
      />

      {/* Onboard Deployable Dialog */}
      <Dialog 
        open={openOnboardDialog} 
        onClose={() => setOpenOnboardDialog(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: '8px',
            overflow: 'hidden'
          }
        }}
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          p: 2,
        }}>
          <Typography variant="h6">Onboard New Deployable</Typography>
          <IconButton 
            onClick={() => setOpenOnboardDialog(false)} 
            sx={{ color: 'white' }}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, mt: 2 }}>
            <TextField
              label="Service"
              name="service"
              value={formData.service}
              disabled
              fullWidth
              size="small"
              InputProps={{
                readOnly: true,
              }}
              sx={{ 
                '& .MuiInputBase-input.Mui-disabled': {
                  WebkitTextFillColor: '#666',
                  backgroundColor: '#f5f5f5'
                }
              }}
            />
            <TextField
              label="App Name"
              name="appName"
              value={formData.appName}
              onChange={handleFormChange}
              fullWidth
              size="small"
              required
              placeholder="Enter application name"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => setOpenOnboardDialog(false)}
            variant="outlined"
            sx={{ 
              borderColor: '#450839',
              color: '#450839',
              '&:hover': {
                borderColor: '#380730',
                color: '#380730'
              }
            }}
          >
            Cancel
          </Button>
          <Button 
            onClick={handleOnboardSubmit} 
            variant="contained" 
            disabled={submitting}
            sx={{ backgroundColor: '#450839', '&:hover': { backgroundColor: '#380730' } }}
          >
            {submitting ? <CircularProgress size={24} /> : 'Submit'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Deployable Dialog */}
      <Dialog 
        open={openEditDialog} 
        onClose={() => setOpenEditDialog(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: '8px',
            overflow: 'hidden'
          }
        }}
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          p: 2,
        }}>
          <Typography variant="h6">Edit Deployable</Typography>
          <IconButton 
            onClick={() => setOpenEditDialog(false)} 
            sx={{ color: 'white' }}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, mt: 2 }}>
            <TextField
              label="Service"
              name="service"
              value={formData.service}
              disabled
              fullWidth
              size="small"
              InputProps={{
                readOnly: true,
              }}
              sx={{ 
                '& .MuiInputBase-input.Mui-disabled': {
                  WebkitTextFillColor: '#666',
                  backgroundColor: '#f5f5f5'
                }
              }}
            />
            <TextField
              label="App Name"
              name="appName"
              value={formData.appName}
              onChange={handleFormChange}
              fullWidth
              size="small"
              required
              placeholder="Enter application name"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => setOpenEditDialog(false)}
            variant="outlined"
            sx={{ 
              borderColor: '#450839',
              color: '#450839',
              '&:hover': {
                borderColor: '#380730',
                color: '#380730'
              }
            }}
          >
            Cancel
          </Button>
          <Button 
            onClick={handleEditSubmit} 
            variant="contained" 
            disabled={submitting}
            sx={{ backgroundColor: '#450839', '&:hover': { backgroundColor: '#380730' } }}
          >
            {submitting ? <CircularProgress size={24} /> : 'Update'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Deployable Details Dialog */}
      <Dialog 
        open={openDetailsDialog} 
        onClose={() => setOpenDetailsDialog(false)} 
        maxWidth="md" 
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: '12px',
            overflow: 'hidden'
          }
        }}
      >
        <DialogTitle sx={{ 
          bgcolor: '#450839', 
          color: 'white',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          pb: 1,
          pt: 1
        }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <InfoIcon />
            <Typography variant="h6">Deployable Details</Typography>
          </Box>
          <IconButton 
            onClick={() => setOpenDetailsDialog(false)} 
            sx={{ color: 'white' }}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ p: 3 }}>
          {selectedDeployable && (
            <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mt: 2 }}>
              <TextField label="Deployable ID" value={selectedDeployable.Id} disabled fullWidth />
              <TextField label="Name" value={selectedDeployable.Name} disabled fullWidth />
              <TextField label="Host" value={selectedDeployable.Host} disabled fullWidth />
              <TextField label="Service" value={selectedDeployable.Service} disabled fullWidth />
              <TextField label="Deployable Running Status" value={selectedDeployable.DeployableRunningStatus ? 'Running' : 'Stopped'} disabled fullWidth />
              <TextField label="CPU Request" value={selectedDeployable.CpuRequest} disabled fullWidth />
              <TextField label="CPU Limit" value={selectedDeployable.CpuLimit} disabled fullWidth />
              <TextField label="Memory Request" value={selectedDeployable.MemRequest} disabled fullWidth />
              <TextField label="Memory Limit" value={selectedDeployable.MemLimit} disabled fullWidth />
              <TextField label="Deployable Health" value={selectedDeployable.DeployableHealth || 'N/A'} disabled fullWidth />
              <TextField label="Dashboard Link" value={selectedDeployable.DashboardLink || 'N/A'} disabled fullWidth />
              <TextField label="Created By" value={selectedDeployable.CreatedBy} disabled fullWidth />
              <TextField label="Updated By" value={selectedDeployable.UpdatedBy || 'N/A'} disabled fullWidth />
              <TextField label="Created At" value={new Date(selectedDeployable.CreatedAt).toLocaleString()} disabled fullWidth />
              <TextField label="Updated At" value={new Date(selectedDeployable.UpdatedAt).toLocaleString()} disabled fullWidth />
            </Box>
          )}
        </DialogContent>
      </Dialog>

      {/* Snackbar for notifications */}
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

export default DeployableInferflowRegistry;
