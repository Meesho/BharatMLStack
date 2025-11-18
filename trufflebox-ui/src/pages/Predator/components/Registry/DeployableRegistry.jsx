import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Box,
  CircularProgress,
  Snackbar,
  Alert,
  Autocomplete
} from '@mui/material';
import { useAuth } from '../../../Auth/AuthContext';
import GenericDeployableTableComponent from './Shared/GenericDeployableTableComponent';
import * as URL_CONSTANTS from '../../../../config';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../../constants/permissions';

const DeployableRegistry = () => {
  const { user, hasPermission } = useAuth();
  const [openDialog, setOpenDialog] = useState(false);
  const [openTuningDialog, setOpenTuningDialog] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTuningSubmitting, setIsTuningSubmitting] = useState(false);
  const [selectedDeployable, setSelectedDeployable] = useState(null);
  const [nodeSelectors, setNodeSelectors] = useState([]);
  const [gcsBasePaths, setGcsBasePaths] = useState([]);
  const [gcsTritonPaths, setGcsTritonPaths] = useState([]);
  const [tritonImageTags, setTritonImageTags] = useState([]);
  const [serviceAccounts, setServiceAccounts] = useState([]);
  const [machineTypes, setMachineTypes] = useState([]);
  const [cpuRequestUnits, setCpuRequestUnits] = useState([]);
  const [cpuLimitUnits, setCpuLimitUnits] = useState([]);
  const [memoryRequestUnits, setMemoryRequestUnits] = useState([]);
  const [memoryLimitUnits, setMemoryLimitUnits] = useState([]);
  const [deploymentStrategies, setDeploymentStrategies] = useState([]);
  const [deployables, setDeployables] = useState([]);
  const [loading, setLoading] = useState(false);
  const service = SERVICES.PREDATOR;
  const screenType = SCREEN_TYPES.PREDATOR.DEPLOYABLE;

  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success'
  });
  const [formData, setFormData] = useState({
    service_name: 'predator',
    appName: '',
    machine_type: '',
    cpuRequest: '',
    cpuRequestUnit: 'm',
    cpuLimit: '',
    cpuLimitUnit: 'm',
    memoryRequest: '',
    memoryRequestUnit: 'M',
    memoryLimit: '',
    memoryLimitUnit: 'M',
    gpu_request: '',
    gpu_limit: '',
    min_replica: '',
    max_replica: '',
    nodeSelectorValue: '',
    triton_image_tag: '',
    gcs_bucket_path: '',
    gcs_triton_path: '',
    serviceAccount: '',
    deploymentStrategy: '',
  });
  const [tuningFormData, setTuningFormData] = useState({
    service_name: 'predator',
    appName: '',
    machine_type: '',
    cpu_threshold: '',
    gpu_threshold: '',
  });
  const [errors, setErrors] = useState({});
  const [tuningErrors, setTuningErrors] = useState({});

  const fetchMetadata = async () => {
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables/metadata`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      if (data.error !== null && data.error !== undefined && data.error !== '') {
        throw new Error(data.error);
      }

             setNodeSelectors(data.node_selectors || []);
       setGcsBasePaths(data.gcs_base_path || []);
       setGcsTritonPaths(data.gcs_triton_path || []);
       setTritonImageTags(data.triton_image_tags || []);
       setServiceAccounts(data.service_account || []);
       setMachineTypes(data.machine_type || ['CPU', 'GPU']);
       setCpuRequestUnits(data.cpu_request_unit || ['m']);
       setCpuLimitUnits(data.cpu_limit_unit || ['m']);
       setMemoryRequestUnits(data.memory_request_unit || ['M']);
       setMemoryLimitUnits(data.memory_limit_unit || ['M']);
       setDeploymentStrategies(data.deployment_strategy || ['rollingUpdate']);


    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to fetch metadata. Please try again.',
        severity: 'error'
      });
      console.log('Error fetching metadata:', error);
    }
  };

  useEffect(() => {
    fetchMetadata();
    fetchDeployables();
  }, []);

  const fetchDeployables = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=PREDATOR`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const res = await response.json();

      if (res.error !== null && res.error !== undefined && res.error !== '') {
        throw new Error(res.error);
      }

      const data = res.data;

      if (!data || !Array.isArray(data)) {
        console.warn('Data is not an array or is undefined:', data);
        setDeployables([]);
        return;
      }

      const formattedData = data.map(item => ({
        id: item.id,
        deployableName: item.name,
        host: item.host,
        service: item.service,
        status: item.active ? 'ACTIVE' : 'PENDING',
        createdBy: item.created_by,
        createdAt: formatDate(item.created_at),
        updatedBy: item.updated_by,
        updatedAt: formatDate(item.updated_at),
        active: item.active,
        dashboardLink: item.monitoring_url,
        workflowStatus: item.workflow_status,
        deployableRunningStatus: item.deployable_running_status,
                  config: {
            machine_type: item.config?.machine_type,
            cpuRequest: item.config?.cpu_request,
            cpuRequestUnit: item.config?.cpu_request_unit,
            cpuLimit: item.config?.cpu_limit,
            cpuLimitUnit: item.config?.cpu_limit_unit,
            memoryRequest: item.config?.mem_request,
            memoryRequestUnit: item.config?.mem_request_unit,
            memoryLimit: item.config?.mem_limit,
            memoryLimitUnit: item.config?.mem_limit_unit,
            gpu_request: item.config?.gpu_request,
            gpu_limit: item.config?.gpu_limit,
            min_replica: item.config?.min_replica,
            max_replica: item.config?.max_replica,
            cpu_threshold: item.config?.cpu_threshold,
            gpu_threshold: item.config?.gpu_threshold,
            nodeSelectorValue: item.config?.nodeselector,
            triton_image_tag: item.config?.triton_image_tag,
            gcs_bucket_path: item.config?.base_path,
            gcs_triton_path: item.config?.gcs_triton_path,
            serviceAccount: item.config?.service_account,
            deploymentStrategy: item.config?.deployment_strategy
          },
        // Original data preserved
        originalData: item
      }));

      setDeployables(formattedData);
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to fetch deployables. Please try again.',
        severity: 'error'
      });
      console.error('Error fetching deployables:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const handleOnboardClick = () => {
    setIsEditMode(false);
    setSelectedDeployable(null);
    resetForm();
    setOpenDialog(true);
  };

  const handleEditClick = (deployable) => {
    setIsEditMode(true);
    setSelectedDeployable(deployable);

    const data = deployable.originalData || deployable.config || {};
    
    setFormData({
      service_name: 'predator',
      appName: deployable.deployableName || deployable.name || '',
      machine_type: data.machine_type || '',
      cpuRequest: data.cpuRequest || '',
      cpuRequestUnit: data.cpuRequestUnit || 'm',
      cpuLimit: data.cpuLimit || '',
      cpuLimitUnit: data.cpuLimitUnit || 'm',
      memoryRequest: data.memoryRequest || '',
      memoryRequestUnit: data.memoryRequestUnit || 'M',
      memoryLimit: data.memoryLimit || '',
      memoryLimitUnit: data.memoryLimitUnit || 'M',
      gpu_request: data.gpu_request || '',
      gpu_limit: data.gpu_limit || '',
      min_replica: data.min_replica || '',
      max_replica: data.max_replica || '',
      nodeSelectorValue: data.nodeSelectorValue || '',
      triton_image_tag: data.triton_image_tag || '',
      gcs_bucket_path: data.gcs_bucket_path || '',
      gcs_triton_path: data.gcs_triton_path || data.gcs_triton_script_path || data.triton_script_path,
      serviceAccount: data.serviceAccount || data.service_account,
      deploymentStrategy: data.deploymentStrategy || data.deployment_strategy,
    });

    setOpenDialog(true);
  };

  const handleClose = () => {
    setOpenDialog(false);
    resetForm();
  };

  const resetForm = () => {
    setFormData({
      service_name: 'predator',
      appName: '',
      machine_type: 'CPU',
      cpuRequest: '',
      cpuRequestUnit: cpuRequestUnits[0] || 'm',
      cpuLimit: '',
      cpuLimitUnit: cpuLimitUnits[0] || 'm',
      memoryRequest: '',
      memoryRequestUnit: memoryRequestUnits[0] || 'M',
      memoryLimit: '',
      memoryLimitUnit: memoryLimitUnits[0] || 'M',
      gpu_request: '',
      gpu_limit: '',
      min_replica: '',
      max_replica: '',
      nodeSelectorValue: '',
      triton_image_tag: '',
      gcs_bucket_path: '',
      gcs_triton_path: '',
      serviceAccount: '',
      deploymentStrategy: deploymentStrategies[0] || '',
    });
    setErrors({});
  };

  const resetTuningForm = () => {
    setTuningFormData({
      service_name: 'predator',
      appName: '',
      machine_type: '',
      cpu_threshold: '',
      gpu_threshold: '',
    });
    setTuningErrors({});
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));

    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }

    if (name === 'machine_type') {
      if (value === 'CPU') {
        // Clear GPU fields when CPU is selected
        setFormData(prev => ({
          ...prev,
          gpu_request: '',
          gpu_limit: ''
        }));
      }
    }
  };

  const handleTuningChange = (e) => {
    const { name, value } = e.target;
    setTuningFormData(prev => ({
      ...prev,
      [name]: value
    }));

    if (tuningErrors[name]) {
      setTuningErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }

    // Clear GPU threshold when machine type changes to CPU
    if (name === 'machine_type' && value === 'CPU') {
      setTuningFormData(prev => ({
        ...prev,
        [name]: value,
        gpu_threshold: ''
      }));
    }
  };

  const validateForm = () => {
    const newErrors = {};

    // Required fields for all cases
    if (!formData.appName) newErrors.appName = 'App Name is required';
    if (!formData.nodeSelectorValue) newErrors.nodeSelectorValue = 'Node Selector is required';
    if (!formData.min_replica) newErrors.min_replica = 'Min Replica is required';
    if (!formData.max_replica) newErrors.max_replica = 'Max Replica is required';
    if (!formData.memoryRequest) newErrors.memoryRequest = 'Memory Request is required';
    if (!formData.memoryLimit) newErrors.memoryLimit = 'Memory Limit is required';
    if (!formData.triton_image_tag) newErrors.triton_image_tag = 'Triton Image Tag is required';
    if (!formData.gcs_bucket_path) newErrors.gcs_bucket_path = 'GCS Bucket Path is required';
    if (!isEditMode && !formData.gcs_triton_path) newErrors.gcs_triton_path = 'GCS Triton Path is required';
    if (!formData.machine_type) newErrors.machine_type = 'Machine Type is required';
    if (!isEditMode && !formData.deploymentStrategy) newErrors.deploymentStrategy = 'Deployment Strategy is required';

    // Conditional validations
    if (formData.machine_type === 'CPU') {
      if (!formData.cpuRequest) newErrors.cpuRequest = 'CPU Request is required for CPU machine type';
      if (!formData.cpuLimit) newErrors.cpuLimit = 'CPU Limit is required for CPU machine type';
    } else if (formData.machine_type === 'GPU') {
      if (!formData.cpuRequest) newErrors.cpuRequest = 'CPU Request is required';
      if (!formData.cpuLimit) newErrors.cpuLimit = 'CPU Limit is required';
      if (!formData.gpu_request) newErrors.gpu_request = 'GPU Request is required for GPU machine type';
      if (!formData.gpu_limit) newErrors.gpu_limit = 'GPU Limit is required for GPU machine type';
    }

    // Number validations
    const numFields = ['cpuRequest', 'cpuLimit', 'memoryRequest', 'memoryLimit',
      'gpu_request', 'gpu_limit', 'min_replica', 'max_replica'];

    numFields.forEach(field => {
      if (formData[field] && isNaN(formData[field])) {
        newErrors[field] = `${field.replace(/([A-Z])/g, ' $1').toLowerCase()} must be a number`;
      }
    });

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const validateTuningForm = () => {
    const newErrors = {};

    // Required fields for tuning
    if (!tuningFormData.appName) newErrors.appName = 'App Name is required';
    if (!tuningFormData.machine_type) newErrors.machine_type = 'Machine Type is required';
    if (!tuningFormData.cpu_threshold) newErrors.cpu_threshold = 'CPU Threshold is required';

    // GPU threshold is only required for GPU machine type
    if (tuningFormData.machine_type === 'GPU' && !tuningFormData.gpu_threshold) {
      newErrors.gpu_threshold = 'GPU Threshold is required for GPU machine type';
    }

    // Number validations for thresholds
    if (tuningFormData.cpu_threshold && isNaN(tuningFormData.cpu_threshold)) {
      newErrors.cpu_threshold = 'CPU Threshold must be a number';
    }
    if (tuningFormData.gpu_threshold && isNaN(tuningFormData.gpu_threshold)) {
      newErrors.gpu_threshold = 'GPU Threshold must be a number';
    }

    setTuningErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;

    setIsSubmitting(true);

    try {
      const payload = {
        appName: formData.appName,
        service_name: formData.service_name,
        machine_type: formData.machine_type,
        cpuRequest: formData.cpuRequest,
        cpuRequestUnit: formData.cpuRequestUnit,
        cpuLimit: formData.cpuLimit,
        cpuLimitUnit: formData.cpuLimitUnit,
        memoryRequest: formData.memoryRequest,
        memoryRequestUnit: formData.memoryRequestUnit,
        memoryLimit: formData.memoryLimit,
        memoryLimitUnit: formData.memoryLimitUnit,
        gpu_request: formData.gpu_request,
        gpu_limit: formData.gpu_limit,
        min_replica: formData.min_replica,
        max_replica: formData.max_replica,
        nodeSelectorValue: formData.nodeSelectorValue,
        triton_image_tag: formData.triton_image_tag,
        gcs_bucket_path: formData.gcs_bucket_path,
        gcs_triton_path: formData.gcs_triton_path,
        serviceAccount: formData.serviceAccount,
        deploymentStrategy: formData.deploymentStrategy,
        created_by: user.email,
      };

      let response;
      let result;

      if (isEditMode) {
        response = await fetch(
          `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-registry/deployables`,
          {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${user.token}`,
            },
            body: JSON.stringify(payload)
          }
        );
      } else {
        response = await fetch(
          `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-registry/deployables`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${user.token}`,
            },
            body: JSON.stringify(payload)
          }
        );
      }

      if (!response.ok) {
        throw new Error(`Failed to ${isEditMode ? 'update' : 'create'} deployable`);
      }

      result = await response.json();

      if (result.error) {
        throw new Error(result.error);
      }

      setSnackbar({
        open: true,
        message: result.data?.message || (isEditMode ? 'Deployable successfully updated! Please refresh the deployable to get the latest status.' : 'Deployable successfully registered! Please refresh the deployable to get the latest status.'),
        severity: 'success'
      });
      handleClose();
      // Refresh the table
      fetchDeployables();
    } catch (error) {
      setSnackbar({
        open: true,
        message: error.message || 'An error occurred. Please try again.',
        severity: 'error'
      });
      console.log('Error submitting deployable:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRefreshAction = async (deployable) => {
    try {
      setSnackbar({
        open: true,
        message: `Refreshing ${deployable.deployableName}...`,
        severity: 'info'
      });

      const queryParams = new URLSearchParams({
        app_name: deployable.deployableName,
        service_type: 'predator'
      });

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-registry/deployables/refresh?${queryParams}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${user.token}`,
          },
          body: JSON.stringify({
            updated_by: user.email
          })
        }
      );

      if (!response.ok) {
        throw new Error('Failed to refresh deployable');
      }

      const result = await response.json();

      setSnackbar({
        open: true,
        message: result.data?.message || `${deployable.deployableName} refreshed successfully!`,
        severity: 'success'
      });

      // Refresh the table data
      fetchDeployables();
    } catch (error) {
      setSnackbar({
        open: true,
        message: error.message || 'Failed to refresh deployable. Please try again.',
        severity: 'error'
      });
      console.log('Error refreshing deployable:', error);
    }
  };

  const handleSnackbarClose = () => {
    setSnackbar(prev => ({
      ...prev,
      open: false
    }));
  };

  const handleTuningClick = (deployable) => {
    setTuningFormData({
      service_name: 'predator',
      appName: deployable.deployableName,
      machine_type: deployable.config.machine_type || '',
      cpu_threshold: deployable.config.cpu_threshold || '',
      gpu_threshold: deployable.config.gpu_threshold || '',
    });
    setOpenTuningDialog(true);
  };

  const handleCloseTuningDialog = () => {
    setOpenTuningDialog(false);
    resetTuningForm();
  };

  const handleTuningSubmit = async () => {
    if (!validateTuningForm()) return;

    setIsTuningSubmitting(true);

    try {
      const payload = {
        appName: tuningFormData.appName,
        service_name: tuningFormData.service_name,
        machine_type: tuningFormData.machine_type,
        cpu_threshold: tuningFormData.cpu_threshold,
      };

      // Only include GPU threshold for GPU machine type
      if (tuningFormData.machine_type === 'GPU') {
        payload.gpu_threshold = tuningFormData.gpu_threshold;
      }

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-registry/deployables/tune-thresholds`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${user.token}`,
          },
          body: JSON.stringify(payload)
        }
      );

      if (!response.ok) {
        throw new Error('Failed to update thresholds');
      }

      const result = await response.json();

      if (result.error) {
        throw new Error(result.error);
      }

      setSnackbar({
        open: true,
        message: result.data?.message || 'Thresholds updated successfully!',
        severity: 'success'
      });
      handleCloseTuningDialog();
      // Refresh the table
      fetchDeployables();
    } catch (error) {
      setSnackbar({
        open: true,
        message: error.message || 'Failed to update thresholds. Please try again.',
        severity: 'error'
      });
      console.log('Error updating thresholds:', error);
    } finally {
      setIsTuningSubmitting(false);
    }
  };

  return (
    <Box sx={{ p: 2 }}>
      <GenericDeployableTableComponent
        handleEditAction={handleEditClick}
        handleRefreshAction={handleRefreshAction}
        handleTuningAction={handleTuningClick}
        handleOnboardDeployable={handleOnboardClick}
        deployables={deployables}
        loading={loading}
      />

      {/* Onboard/Edit Deployable Dialog */}
      <Dialog
        open={openDialog}
        onClose={handleClose}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          {isEditMode ? 'Edit Deployable' : 'Onboard Deployable'}
        </DialogTitle>

        <DialogContent dividers>
          <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2, my: 2 }}>
            {/* Service Name - Disabled - Full Width */}
            <FormControl fullWidth>
              <InputLabel>Service</InputLabel>
              <Select
                name="service_name"
                value={formData.service_name}
                label="Service"
                disabled
              >
                <MenuItem value="predator">predator</MenuItem>
              </Select>
            </FormControl>

            {/* App Name - Full Width */}
            <TextField
              label="App Name"
              name="appName"
              value={formData.appName}
              onChange={handleChange}
              error={!!errors.appName}
              helperText={errors.appName}
              disabled={isEditMode}
              required
              fullWidth
            />

            {/* Machine Type - Full Width */}
            <Autocomplete
              options={[...(machineTypes || [])].sort((a, b) => a.localeCompare(b))}
              value={formData.machine_type}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'machine_type',
                    value: newValue || ''
                  }
                };
                handleChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Machine Type"
                  name="machine_type"
                  required
                  fullWidth
                  error={!!errors.machine_type}
                  helperText={errors.machine_type}
                />
              )}
            />

            {/* CPU Request and CPU Limit - Side by Side */}
            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="CPU Request"
                name="cpuRequest"
                value={formData.cpuRequest}
                onChange={handleChange}
                error={!!errors.cpuRequest}
                helperText={errors.cpuRequest}
                required
                sx={{ flex: 1 }}
              />

              <FormControl sx={{ flex: 1 }}>
                <InputLabel>CPU Request Unit</InputLabel>
                <Select
                  name="cpuRequestUnit"
                  value={formData.cpuRequestUnit}
                  label="CPU Request Unit"
                  onChange={handleChange}
                >
                  {cpuRequestUnits.map((unit, index) => (
                    <MenuItem key={unit || `empty-${index}`} value={unit}>
                      {unit || 'cores'}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>

            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="CPU Limit"
                name="cpuLimit"
                value={formData.cpuLimit}
                onChange={handleChange}
                error={!!errors.cpuLimit}
                helperText={errors.cpuLimit}
                required
                sx={{ flex: 1 }}
              />

              <FormControl sx={{ flex: 1 }}>
                <InputLabel>CPU Limit Unit</InputLabel>
                <Select
                  name="cpuLimitUnit"
                  value={formData.cpuLimitUnit}
                  label="CPU Limit Unit"
                  onChange={handleChange}
                >
                  {cpuLimitUnits.map((unit, index) => (
                    <MenuItem key={unit || `empty-${index}`} value={unit}>
                      {unit || 'cores'}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>

            {formData.machine_type === 'GPU' && (
              <Box sx={{ display: 'flex', gap: 2 }}>
                <TextField
                  label="GPU Request"
                  name="gpu_request"
                  value={formData.gpu_request}
                  onChange={handleChange}
                  error={!!errors.gpu_request}
                  helperText={errors.gpu_request}
                  required
                  sx={{ flex: 1 }}
                />

                <TextField
                  label="GPU Limit"
                  name="gpu_limit"
                  value={formData.gpu_limit}
                  onChange={handleChange}
                  error={!!errors.gpu_limit}
                  helperText={errors.gpu_limit}
                  required
                  sx={{ flex: 1 }}
                />
              </Box>
            )}

            {/* Memory Request and Memory Limit - Side by Side */}
            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="Memory Request"
                name="memoryRequest"
                value={formData.memoryRequest}
                onChange={handleChange}
                error={!!errors.memoryRequest}
                helperText={errors.memoryRequest}
                required
                sx={{ flex: 1 }}
              />

              <FormControl sx={{ flex: 1 }} required>
                <InputLabel>Memory Request Unit</InputLabel>
                <Select
                  name="memoryRequestUnit"
                  value={formData.memoryRequestUnit}
                  label="Memory Request Unit"
                  onChange={handleChange}
                >
                  {memoryRequestUnits.map((unit, index) => (
                    <MenuItem key={unit || `empty-${index}`} value={unit}>
                      {unit}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>

            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="Memory Limit"
                name="memoryLimit"
                value={formData.memoryLimit}
                onChange={handleChange}
                error={!!errors.memoryLimit}
                helperText={errors.memoryLimit}
                required
                sx={{ flex: 1 }}
              />

              <FormControl sx={{ flex: 1 }} required>
                <InputLabel>Memory Limit Unit</InputLabel>
                <Select
                  name="memoryLimitUnit"
                  value={formData.memoryLimitUnit}
                  label="Memory Limit Unit"
                  onChange={handleChange}
                >
                  {memoryLimitUnits.map((unit, index) => (
                    <MenuItem key={unit || `empty-${index}`} value={unit}>
                      {unit}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>



            {/* Min Replica and Max Replica - Side by Side */}
            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="Min Replica"
                name="min_replica"
                value={formData.min_replica}
                onChange={handleChange}
                error={!!errors.min_replica}
                helperText={errors.min_replica}
                required
                sx={{ flex: 1 }}
              />

              <TextField
                label="Max Replica"
                name="max_replica"
                value={formData.max_replica}
                onChange={handleChange}
                error={!!errors.max_replica}
                helperText={errors.max_replica}
                required
                sx={{ flex: 1 }}
              />
            </Box>

            {/* Node Selector - Full Width */}
            <Autocomplete
              options={[...(nodeSelectors || [])].sort((a, b) => a.localeCompare(b))}
              value={formData.nodeSelectorValue}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'nodeSelectorValue',
                    value: newValue || ''
                  }
                };
                handleChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Node Selector"
                  name="nodeSelectorValue"
                  required
                  error={!!errors.nodeSelectorValue}
                  helperText={errors.nodeSelectorValue}
                  fullWidth
                />
              )}
            />

            {/* Triton Image Tag - Full Width */}
            <Autocomplete
              options={[...(tritonImageTags || [])].sort((a, b) => a.localeCompare(b))}
              value={formData.triton_image_tag}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'triton_image_tag',
                    value: newValue || ''
                  }
                };
                handleChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Triton Image Tag"
                  name="triton_image_tag"
                  required
                  error={!!errors.triton_image_tag}
                  helperText={errors.triton_image_tag}
                  fullWidth
                />
              )}
            />

            {/* GCS Bucket Path - Full Width */}
            <Autocomplete
              options={[...(gcsBasePaths || [])].sort((a, b) => a.localeCompare(b))}
              value={formData.gcs_bucket_path}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'gcs_bucket_path',
                    value: newValue || ''
                  }
                };
                handleChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="GCS Bucket Path"
                  name="gcs_bucket_path"
                  required
                  error={!!errors.gcs_bucket_path}
                  helperText={errors.gcs_bucket_path}
                  fullWidth
                />
              )}
            />

            {/* GCS Triton Path - Full Width */}
            <Autocomplete
              options={[...(gcsTritonPaths || [])].sort((a, b) => a.localeCompare(b))}
              value={formData.gcs_triton_path}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'gcs_triton_path',
                    value: newValue || ''
                  }
                };
                handleChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="GCS Triton Path"
                  name="gcs_triton_path"
                  required
                  error={!!errors.gcs_triton_path}
                  helperText={errors.gcs_triton_path}
                  fullWidth
                />
              )}
            />

            {/* Service Account - Full Width */}
            <Autocomplete
              options={[...(serviceAccounts || [])].sort((a, b) => a.localeCompare(b))}
              value={formData.serviceAccount}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'serviceAccount',
                    value: newValue || ''
                  }
                };
                handleChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Service Account"
                  name="serviceAccount"
                  error={!!errors.serviceAccount}
                  helperText={errors.serviceAccount}
                  fullWidth
                />
              )}
            />
          </Box>
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleClose}
            variant="outlined"
            disabled={isSubmitting}
            sx={{
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
                color: '#ffffff',
              },
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            disabled={isSubmitting}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              },
            }}
          >
            {isSubmitting ? (
              <CircularProgress size={24} color="inherit" />
            ) : (
              isEditMode ? 'Update' : 'Onboard'
            )}
          </Button>
        </DialogActions>
      </Dialog>

      {/* CPU/GPU Tuning Dialog */}
      <Dialog
        open={openTuningDialog}
        onClose={handleCloseTuningDialog}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          CPU/GPU Tuning
        </DialogTitle>

        <DialogContent dividers>
          <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2, my: 2 }}>
            {/* Service Name - Disabled - Full Width */}
            <Autocomplete
              options={['predator'].sort((a, b) => a.localeCompare(b))}
              value={tuningFormData.service_name}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'service_name',
                    value: newValue || ''
                  }
                };
                handleTuningChange(syntheticEvent);
              }}
              disabled
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Service"
                  name="service_name"
                  fullWidth
                />
              )}
            />

            {/* App Name - Full Width */}
            <TextField
              label="App Name"
              name="appName"
              value={tuningFormData.appName}
              onChange={handleTuningChange}
              error={!!tuningErrors.appName}
              helperText={tuningErrors.appName}
              required
              fullWidth
            />

            {/* Machine Type - Full Width */}
            <Autocomplete
              options={[...(machineTypes || [])].sort((a, b) => a.localeCompare(b))}
              value={tuningFormData.machine_type}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'machine_type',
                    value: newValue || ''
                  }
                };
                handleTuningChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Machine Type"
                  name="machine_type"
                  required
                  error={!!tuningErrors.machine_type}
                  helperText={tuningErrors.machine_type}
                  fullWidth
                />
              )}
            />

            {/* CPU Threshold - Always shown */}
            <TextField
              label="CPU Threshold"
              name="cpu_threshold"
              value={tuningFormData.cpu_threshold}
              onChange={handleTuningChange}
              error={!!tuningErrors.cpu_threshold}
              helperText={tuningErrors.cpu_threshold}
              required
              fullWidth
            />

            {/* GPU Threshold - Only shown for GPU machine type */}
            {tuningFormData.machine_type === 'GPU' && (
              <TextField
                label="GPU Threshold"
                name="gpu_threshold"
                value={tuningFormData.gpu_threshold}
                onChange={handleTuningChange}
                error={!!tuningErrors.gpu_threshold}
                helperText={tuningErrors.gpu_threshold}
                required
                fullWidth
              />
            )}
          </Box>
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleCloseTuningDialog}
            variant="outlined"
            disabled={isTuningSubmitting}
            sx={{
              color: '#450839',
              borderColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
                color: '#ffffff',
              },
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleTuningSubmit}
            variant="contained"
            disabled={isTuningSubmitting}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              },
            }}
          >
            {isTuningSubmitting ? (
              <CircularProgress size={24} color="inherit" />
            ) : (
              'Update Thresholds'
            )}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Success/Error Notification */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={handleSnackbarClose}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default DeployableRegistry;

