import React, { useState, useEffect, useCallback } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  CircularProgress,
  Snackbar,
  Alert,
  Typography,
  IconButton,
  Grid,
  Card,
  CardContent,
  Chip,
  Autocomplete,
  Backdrop,
} from '@mui/material';
import JsonViewer from '../../../../components/JsonViewer';

import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { useAuth } from '../../../Auth/AuthContext';
import GenericModelTableComponent from './Shared/GenericModelTableComponent';
import ModelTestingModal from './components/ModelTestingModal';
import EditModelModal from './components/EditModelModal';
import UploadModelModal from './components/UploadModelModal';
import EditModelModalForUpload from './components/EditModelModalForUpload';
import ProductionCredentialModal from '../../../../common/ProductionCredentialModal';
import * as URL_CONSTANTS from '../../../../config';
import { Add, Delete } from '@mui/icons-material';

const ModelRegistry = () => {
  const { user } = useAuth();
  const [openDialog, setOpenDialog] = useState(false);
  const [openScaleUpDialog, setOpenScaleUpDialog] = useState(false);
  const [openPromoteDialog, setOpenPromoteDialog] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoadingModelParams, setIsLoadingModelParams] = useState(false);
  const [models, setModels] = useState([]);
  const [modelsLoading, setModelsLoading] = useState(true);
  const [hosts, setHosts] = useState([]);
  const [promoteHosts, setPromoteHosts] = useState([]);
  const [sourceModels, setSourceModels] = useState([]);
  const [sourceModelsLoading, setSourceModelsLoading] = useState(false);
  const [modelParamsLoaded, setModelParamsLoaded] = useState(false);
  const [selectedModelsForAction, setSelectedModelsForAction] = useState([]);
  const [modelNameError, setModelNameError] = useState('');
  const [existingModelNames, setExistingModelNames] = useState([]);
  const [scaleUpFormData, setScaleUpFormData] = useState({
    selectedHost: ''
  });
  const [promoteFormData, setPromoteFormData] = useState({
    selectedHost: ''
  });
  const [onboardFormData, setOnboardFormData] = useState({
    selectedHost: ''
  });
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [selectedModelForTest, setSelectedModelForTest] = useState(null);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [selectedModelForEdit, setSelectedModelForEdit] = useState(null);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [editModelForUploadModalOpen, setEditModelForUploadModalOpen] = useState(false);
  const [productionCredentialModalOpen, setProductionCredentialModalOpen] = useState(false);
  const [productionCredentialError, setProductionCredentialError] = useState('');
  const [isLoadingPromoteSetup, setIsLoadingPromoteSetup] = useState(false);

  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  const [modelForms, setModelForms] = useState([]);

  const getPathFromModelName = (modelName) => {
    const folder = sourceModels.find(folder => folder.name === modelName);
    return folder ? folder.transformedPath : '';
  };

  const getAvailableModelOptions = (currentIndex) => {
    const selectedModelNames = modelForms
      .map((form, index) => index !== currentIndex ? form.selectedModelName : null)
      .filter(name => name && name.trim() !== '');

    return sourceModels.filter(folder => !selectedModelNames.includes(folder.name));
  };

  const formatDate = useCallback((dateString) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString();
  }, []);

  const fetchModels = useCallback(async () => {
    try {
      setModelsLoading(true);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/models`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
      });

      if (!response.ok) {
        console.log('Failed to fetch models');
      }

      const result = await response.json();
      const data = result.data || [];

      const formattedData = data.map(item => ({
        id: item.id,
        modelName: item.model_name,
        metaData: item.meta_data,
        host: item.host,
        machineType: item.machine_type,
        deploymentConfig: item.deployment_config,
        testResults: item.test_results,
        appToken: item.app_token,
        connectionConfig: item.connection_config,
        monitoringUrl: item.monitoring_url,
        gcsPath: item.gcs_path,
        createdBy: item.created_by,
        createdAt: formatDate(item.created_at),
        updatedBy: item.updated_by,
        updatedAt: formatDate(item.updated_at),
        active: item.active,
        deployableRunningStatus: item.deployable_running_status,
        originalData: item
      }));

      setModels(formattedData);

    } catch (error) {
      console.error('Error fetching models:', error);
      setSnackbar({
        open: true,
        message: 'Failed to fetch models',
        severity: 'error'
      });
    } finally {
      setModelsLoading(false);
    }
  }, [user.token, formatDate]);

  const fetchHosts = useCallback(async () => {
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=PREDATOR`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        }
      );

      if (!response.ok) {
        console.log('Failed to fetch hosts');
      }

      const result = await response.json();

      const data = Array.isArray(result) ? result : result.data;

      const hostValues = data
        .filter(item => item.host)
        .map(item => ({
          id: item.id,
          host: item.host,
          name: item.name,
          active: item.active,
          config: item.config || {}
        }));

      setHosts(hostValues);
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to fetch hosts',
        severity: 'error'
      });
      console.log('Error fetching hosts:', error);
    }
  }, [user.token]);

  const fetchExistingModelNames = useCallback(async () => {
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/models`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
          },
        }
      );

      const result = await response.json();

      if (result.error) {
        console.log(result.error);
        return;
      }

      const modelNames = [];

      if (result.data && result.data.length > 0) {
        result.data.forEach(model => {
          if (model.model_name) {
            modelNames.push(model.model_name);
          }
        });
      }

      setExistingModelNames(modelNames);
    } catch (error) {
      console.log('Error fetching existing model names:', error);
    }
  }, [user.token]);

  const fetchPromoteHosts = async (prodToken) => {
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/horizon/deployable-discovery/deployables?service_name=PREDATOR`,
        {
          headers: {
            'Authorization': `Bearer ${prodToken}`,
          },
        }
      );

      if (!response.ok) {
        console.log('Failed to fetch promote hosts');
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();

      const data = Array.isArray(result) ? result : result.data;

      const hostValues = data
        .filter(item => item.host)
        .map(item => ({
          id: item.id,
          host: item.host,
          name: item.name,
          active: item.active,
          config: item.config || {}
        }));

      setPromoteHosts(hostValues);
      return true;
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to fetch promote hosts',
        severity: 'error'
      });
      console.log('Error fetching promote hosts:', error);
      return false;
    }
  };

  const fetchSourceModels = useCallback(async () => {
    setSourceModelsLoading(true);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-discovery/source-models`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        }
      );

      if (!response.ok) {
        console.log('Failed to fetch source models');
      }

      const data = await response.json();
      const transformedFolders = (data.data?.folders || []).map(folder => ({
        ...folder,
        transformedPath: folder.path.includes('://') ?
          '/' + folder.path.split('://')[1] :
          folder.path
      }));
      setSourceModels(transformedFolders);
    } catch (error) {
      console.error('Error fetching source models:', error);
      setSnackbar({
        open: true,
        message: 'Failed to fetch source models',
        severity: 'error'
      });
    } finally {
      setSourceModelsLoading(false);
    }
  }, [user.token]);

  const fetchModelParams = async (modelPath, formIndex) => {
    setIsLoadingModelParams(true);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/model-params?model_path=${modelPath}`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        }
      );

      if (!response.ok) {
        console.log('Failed to fetch model parameters');
      }

      const data = await response.json();

      setModelForms(prev => {
        const updated = [...prev];
        updated[formIndex] = {
          ...updated[formIndex],
          instanceCount: data.instance_count?.toString() || '',
          instanceType: data.instance_type || '',
          batchSize: data.batch_size?.toString() || '',
          backend: data.backend || '',
          inputs: data.inputs || [],
          outputs: data.outputs || [],
          dynamicBatchingEnabled: data.dynamic_batching_enabled !== undefined ? data.dynamic_batching_enabled : false,
          platform: data.platform || '',
          ensembleScheduling: data.ensemble_scheduling || {},
          deploymentConfig: {
            cpu_request: data.deployment_config?.cpu_request || '',
            cpu_limit: data.deployment_config?.cpu_limit || '',
            mem_request: data.deployment_config?.mem_request || '',
            mem_limit: data.deployment_config?.mem_limit || '',
            gpu_request: data.deployment_config?.gpu_request || '',
            gpu_limit: data.deployment_config?.gpu_limit || '',
            min_replica: data.deployment_config?.min_replica || '',
            max_replica: data.deployment_config?.max_replica || '',
            node_selector: data.deployment_config?.node_selector || ''
          },
          appToken: data.app_token || '',
          isLoadingModelParams: false,
          modelParamsLoaded: true,
          errors: {}
        };

        if (updated[formIndex].platform === "ensemble") {
          console.log("Ensemble model detected:", updated[formIndex].ensembleScheduling);
        }

        return updated;
      });

      setModelParamsLoaded(true);
      return true;
    } catch (error) {
      console.error('Error in fetchModelParams:', error);
      setSnackbar({
        open: true,
        message: 'Failed to fetch model parameters',
        severity: 'error'
      });
      return false;
    } finally {
      setIsLoadingModelParams(false);
    }
  };

  const handlePublishClick = async (index) => {
    const modelForm = modelForms[index];
    
    if (!modelForm.selectedModelName.trim()) {
      return;
    }

    const modelPath = getPathFromModelName(modelForm.selectedModelName);
    if (!modelPath) {
      setSnackbar({
        open: true,
        message: 'Could not find path for selected model',
        severity: 'error'
      });
      return;
    }

    const success = await fetchModelParams(modelPath, index);
    if (success) {
      setSnackbar({
        open: true,
        message: 'Model parameters loaded successfully',
        severity: 'success'
      });
    }
  };

  const handlePublishClickWithModelName = async (index, modelName) => {
    if (!modelName || !modelName.trim()) {
      return;
    }

    const modelPath = getPathFromModelName(modelName);
    if (!modelPath) {
      setSnackbar({
        open: true,
        message: 'Could not find path for selected model',
        severity: 'error'
      });
      return;
    }

    const success = await fetchModelParams(modelPath, index);
    if (success) {
      setSnackbar({
        open: true,
        message: 'Model parameters loaded successfully',
        severity: 'success'
      });
    }
  };

  const handleOnboardClick = async () => {
    resetForm();
    await fetchSourceModels();
    // Add one default blank model form
    setModelForms([{
      id: Date.now(),
      selectedModelName: '',
      instanceCount: '',
      instanceType: '',
      batchSize: '',
      backend: '',
      inputs: [],
      outputs: [],
      dynamicBatchingEnabled: false,
      platform: '',
      ensembleScheduling: {},
      deploymentConfig: {
        cpu_request: '',
        cpu_limit: '',
        mem_request: '',
        mem_limit: '',
        gpu_request: '',
        gpu_limit: '',
        min_replica: '',
        max_replica: '',
        node_selector: ''
      },
      appToken: '',
      isLoadingModelParams: false,
      modelParamsLoaded: false,
      errors: {}
    }]);
    setOpenDialog(true);
  };

  const handleCloneAction = async (models) => {
    const modelsData = Array.isArray(models) ? models : [models];
    setSelectedModelsForAction(modelsData.map(model => ({
      ...model,
      originalModelName: model.modelName,
      selectedModelName: model.modelName,
      isNameEdited: false
    })));
    setScaleUpFormData({
      selectedHost: modelsData[0]?.host || ''
    });

    await fetchExistingModelNames();
    await fetchSourceModels();
    setOpenScaleUpDialog(true);
  };

  const handlePromoteAction = async (models) => {
    const modelsData = Array.isArray(models) ? models : [models];
    setSelectedModelsForAction(modelsData);
    setPromoteFormData({
      selectedHost: ''
    });

    setProductionCredentialModalOpen(true);
  };

  const handlePromoteFormChange = (event) => {
    const { name, value } = event.target;
    setPromoteFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleProductionCredentialSuccess = async (prodCredentials) => {
    setProductionCredentialModalOpen(false);
    setIsLoadingPromoteSetup(true);

    if (!selectedModelsForAction.length) {
      setSnackbar({
        open: true,
        message: 'No models selected for promotion',
        severity: 'error'
      });
      setIsLoadingPromoteSetup(false);
      return;
    }

    try {
      const hostsFetched = await fetchPromoteHosts(prodCredentials.token);
      if (!hostsFetched) {
        setIsLoadingPromoteSetup(false);
        return;
      }

      window.prodCredentials = prodCredentials;

      setOpenPromoteDialog(true);
    } catch (error) {
      console.error('Error setting up promote dialog:', error);
      setSnackbar({
        open: true,
        message: 'Failed to load promote hosts',
        severity: 'error'
      });
    } finally {
      setIsLoadingPromoteSetup(false);
    }
  };

  const handleProductionCredentialClose = () => {
    setProductionCredentialModalOpen(false);
    setProductionCredentialError('');
    setIsLoadingPromoteSetup(false);
    delete window.prodCredentials;
  };

  const processPromote = async () => {
    if (!promoteFormData.selectedHost) {
      setSnackbar({
        open: true,
        message: 'Please select a host',
        severity: 'error'
      });
      return;
    }

    const selectedHost = promoteHosts.find(host => host.id === parseInt(promoteFormData.selectedHost));

    const prodCredentials = window.prodCredentials;
    if (!prodCredentials) {
      setSnackbar({
        open: true,
        message: 'Production credentials not found. Please try again.',
        severity: 'error'
      });
      return;
    }

    setIsSubmitting(true);

    try {
      const payload = {
        payload: selectedModelsForAction.map(model => ({
          model_name: model.modelName,
          model_source_path: `${model.gcsPath}/${model.modelName}` || model.originalData?.gcs_path || `${selectedHost.config.base_path}/${model.modelName}`,
          meta_data: normalizeModelMetadata(model.metaData),
          config_mapping: {
            service_deployable_id: parseInt(promoteFormData.selectedHost)
          }
        })),
        created_by: prodCredentials.email
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/horizon/predator-config-registry/models/promote`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${prodCredentials.token}`,
        },
        body: JSON.stringify(payload),
      });

      const result = await response.json();

      if (result.error) {
        setSnackbar({
          open: true,
          message: result.error.message || result.error || 'Promotion failed',
          severity: 'error'
        });
        return;
      }

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      setSnackbar({
        open: true,
        message: result.data?.message || 'Models promoted successfully',
        severity: 'success'
      });

      setOpenPromoteDialog(false);

      // Refresh models list
      await fetchModels();

      delete window.prodCredentials;

    } catch (error) {
      console.error('Error during promote operation:', error);
      setSnackbar({
        open: true,
        message: `Promotion failed: ${error.message}`,
        severity: 'error'
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleScaleUpFormChange = (event) => {
    const { name, value } = event.target;
    setScaleUpFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleOnboardFormChange = (event) => {
    const { name, value } = event.target;
    setOnboardFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleModelNameChange = (index, newName) => {
    setSelectedModelsForAction(prev => {
      const updated = [...prev];
      updated[index] = {
        ...updated[index],
        selectedModelName: newName.trim(),
        isNameEdited: true
      };
      return updated;
    });

    if (modelNameError && !existingModelNames.includes(newName)) {
      setModelNameError('');
    }
  };

  const normalizeModelMetadata = (metaData) => {
    const metadata = {
      inputs: metaData.inputs || [],
      outputs: metaData.outputs || []
    };

    if (metaData.platform) {
      metadata.platform = metaData.platform;
    }

    const batchSize = metaData.batchSize || metaData.batch_size;
    if (batchSize) {
      metadata.batch_size = parseInt(batchSize);
    }

    const instanceCount = metaData.instanceCount || metaData.instance_count;
    if (instanceCount) {
      metadata.instance_count = parseInt(instanceCount);
    }

    const instanceType = metaData.instanceType || metaData.instance_type;
    if (instanceType) {
      metadata.instance_type = instanceType;
    }

    if (metaData.backend) {
      metadata.backend = metaData.backend;
    }

    const dynamicBatching = metaData.dynamicBatchingEnabled !== undefined ? metaData.dynamicBatchingEnabled : metaData.dynamic_batching_enabled;
    if (dynamicBatching !== undefined) {
      metadata.dynamic_batching_enabled = dynamicBatching;
    }

    if (metaData.platform === 'ensemble' && (metaData.ensembleScheduling || metaData.ensemble_scheduling)) {
      metadata.ensemble_scheduling = metaData.ensembleScheduling || metaData.ensemble_scheduling;
    }

    const appToken = metaData.appToken || metaData.app_token;
    if (appToken) {
      metadata.app_token = appToken;
    }

    return metadata;
  };

  const processScaleUp = async () => {
    const invalidNames = selectedModelsForAction.filter(model =>
      existingModelNames.includes(model.selectedModelName)
    );

    if (invalidNames.length > 0) {
      setModelNameError('One or more model names already exist. Please ensure all names are unique.');
      return;
    }

    // Check if all models have selected model names
    const emptyNames = selectedModelsForAction.filter(model => !model.selectedModelName);
    if (emptyNames.length > 0) {
      setModelNameError('Please select a model name for all models.');
      return;
    }

    if (!scaleUpFormData.selectedHost) {
      setSnackbar({
        open: true,
        message: 'Please select a host',
        severity: 'error'
      });
      return;
    }

    // Find the selected host to get its base_path
    const selectedHost = hosts.find(host => host.id === parseInt(scaleUpFormData.selectedHost));
    // if (!selectedHost || !selectedHost.config || !selectedHost.config.base_path) {
    //   setSnackbar({
    //     open: true,
    //     message: 'Selected host does not have a valid base path configuration',
    //     severity: 'error'
    //   });
    //   return;
    // }

    setIsSubmitting(true);

    try {
      const payload = {
        payload: selectedModelsForAction.map(model => {
          const originalHost = hosts.find(host => host.host === model.host);
          const originalBasePath = originalHost?.config?.base_path || '';

          // Use original model name for source path construction only
          const originalModelName = model.originalModelName || model.selectedModelName;

          const modelSourcePath = originalBasePath ? `${originalBasePath}/${originalModelName}` :
            (`${model.gcsPath}/${originalModelName}` || model.originalData?.gcs_path || originalModelName);

          return {
            model_name: model.selectedModelName,
            model_source_path: modelSourcePath,
            meta_data: normalizeModelMetadata(model.metaData),
            config_mapping: {
              service_deployable_id: parseInt(scaleUpFormData.selectedHost)
            }
          };
        }),
        created_by: user.email
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/models/scale-up`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
        body: JSON.stringify(payload),
      });

      const result = await response.json();

      if (result.error) {
        setSnackbar({
          open: true,
          message: result.error.message || result.error || 'Scale up failed',
          severity: 'error'
        });
        return;
      }

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      setSnackbar({
        open: true,
        message: result.data?.message || 'Models scaled up successfully',
        severity: 'success'
      });

      setOpenScaleUpDialog(false);

      // Refresh models list
      await fetchModels();

    } catch (error) {
      console.error('Error during scale up operation:', error);
      setSnackbar({
        open: true,
        message: `Scale up failed: ${error.message}`,
        severity: 'error'
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCloseScaleUpDialog = () => {
    setOpenScaleUpDialog(false);
    setSelectedModelsForAction([]);
    setScaleUpFormData({
      selectedHost: ''
    });
    setModelNameError('');
  };

  const handleClosePromoteDialog = () => {
    setOpenPromoteDialog(false);
    setSelectedModelsForAction([]);
    setPromoteFormData({
      selectedHost: ''
    });
    setIsLoadingPromoteSetup(false);
    delete window.prodCredentials;
  };

  const handleRemoveAction = async (modelIds) => {
    try {
      setIsSubmitting(true);

      const payload = {
        ids: Array.isArray(modelIds) ? modelIds : [modelIds]
      };

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/models/delete`,
        {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${user.token}`,
          },
          body: JSON.stringify(payload),
        });

      const result = await response.json();

      if (result.error) {
        console.log(result.error);
        setSnackbar({
          open: true,
          message: result.error.message || result.error || 'Delete operation failed',
          severity: 'error'
        });
        return;
      }

      setSnackbar({
        open: true,
        message: result.data?.message || 'Models deleted successfully',
        severity: 'success'
      });

      // Refresh models list
      await fetchModels();

    } catch (error) {
      console.error('Error during delete operation:', error);
      setSnackbar({
        open: true,
        message: `Delete failed: ${error.message}`,
        severity: 'error'
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    setOpenDialog(false);
    resetForm();
  };

  const resetForm = () => {
    setModelForms([]);
    setModelParamsLoaded(false);
    setOnboardFormData({
      selectedHost: ''
    });
  };

  const handleChange = (event, index) => {
    const { name, value } = event.target;

    setModelForms(prev => {
      const updated = [...prev];
      
      if (name.includes('deploymentConfig.')) {
        const configField = name.split('.')[1];
        updated[index] = {
          ...updated[index],
          deploymentConfig: {
            ...updated[index].deploymentConfig,
            [configField]: value
          }
        };
      } else {
        updated[index] = {
          ...updated[index],
          [name]: value
        };
      }

      if (value && value.trim()) {
        updated[index] = {
          ...updated[index],
          errors: {
            ...updated[index].errors,
            [name]: ''
          }
        };
      }

      return updated;
    });
  };

  const validateForm = () => {
    const newErrors = {};

    if (!modelForms.length) newErrors.modelForms = 'No models to onboard';
    if (!modelForms.every(m => m.selectedModelName.trim())) newErrors.modelNames = 'All models require a model name selection';
    if (!onboardFormData.selectedHost) newErrors.host = 'Host is required';
    if (!modelForms.every(m => m.batchSize.trim())) newErrors.batchSizes = 'All models require a batch size';

    setModelForms(prev => prev.map((m, index) => {
      const hasSelectedModelNameError = !m.selectedModelName.trim();
      const hasBatchSizeError = !m.batchSize.trim();
      
      return {
        ...m,
        errors: {
          ...m.errors,
          selectedModelName: hasSelectedModelNameError ? 'Model name selection is required' : '',
          batchSize: hasBatchSizeError ? 'Batch size is required' : '',
        }
      };
    }));

    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      const selectedHost = hosts.find(host => host.id === parseInt(onboardFormData.selectedHost));
      if (!selectedHost || !selectedHost.config || !selectedHost.config.base_path) {
        console.warn(`Host ${onboardFormData.selectedHost} does not have a valid base path configuration`);
      }

      const payload = {
        payload: modelForms.map(m => {
          const basePath = selectedHost?.config?.base_path || '';
          // Get the actual path from the selected model name
          const originalModelPath = getPathFromModelName(m.selectedModelName);
          // Combine base_path with selected model name if basePath exists, otherwise use original path
          const modelSourcePath = basePath ? `${basePath}/${m.selectedModelName}` : originalModelPath;

          return {
            model_name: m.selectedModelName,
            model_source_path: `gs:/${modelSourcePath}`,
            meta_data: normalizeModelMetadata(m),
            config_mapping: {
              service_deployable_id: parseInt(onboardFormData.selectedHost)
            }
          };
        }),
        created_by: user.email || ""
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/predator-config-registry/models/onboard`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${user.token}`,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        console.log(`Failed to onboard models: ${response.status} ${response.statusText}`);
      }

      const result = await response.json();

      if (result.error) {
        console.log(result.error);
        setSnackbar({
          open: true,
          message: result.error.message || result.error || 'Onboard operation failed',
          severity: 'error'
        });
        return;
      }

      setSnackbar({
        open: true,
        message: result.data?.message || "Models onboarding request raised successfully",
        severity: 'success'
      });

      setOpenDialog(false);
      resetForm();

      // Refresh the models list
      await fetchModels();

    } catch (error) {
      console.error('Error submitting models:', error);
      setSnackbar({
        open: true,
        message: `Failed to onboard models: ${error.message}`,
        severity: 'error'
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSnackbarClose = (event, reason) => {
    if (reason === 'clickaway') {
      return;
    }
    setSnackbar(prev => ({ ...prev, open: false }));
  };

  const getDims = (item) => {
    if (item.dims !== undefined) {
      return Array.isArray(item.dims) ? JSON.stringify(item.dims) : item.dims;
    }
    return '';
  };

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

  const handleAddModel = () => {
    setModelForms(prev => [
      ...prev,
      {
        id: Date.now(),
        selectedModelName: '',
        instanceCount: '',
        instanceType: '',
        batchSize: '',
        backend: '',
        inputs: [],
        outputs: [],
        dynamicBatchingEnabled: false,
        platform: '',
        ensembleScheduling: {},
        deploymentConfig: {
          cpu_request: '',
          cpu_limit: '',
          mem_request: '',
          mem_limit: '',
          gpu_request: '',
          gpu_limit: '',
          min_replica: '',
          max_replica: '',
          node_selector: ''
        },
        appToken: '',
        isLoadingModelParams: false,
        modelParamsLoaded: false,
        errors: {}
      }
    ]);
  };

  const handleRemoveModel = (index) => {
    setModelForms(prev => prev.filter((_, i) => i !== index));
  };

  const handleTestModel = (model) => {
    setSelectedModelForTest(model);
    setTestModalOpen(true);
  };

  const handleTestComplete = (message, severity) => {
    setSnackbar({
      open: true,
      message,
      severity
    });
    setTestModalOpen(false);
    setSelectedModelForTest(null);
    
    // Refresh models list if test was successful
    if (severity === 'success') {
      fetchModels();
    }
  };

  const handleEditModel = (model) => {
    setSelectedModelForEdit(model);
    setEditModalOpen(true);
  };

  const handleEditComplete = (message, severity) => {
    setSnackbar({
      open: true,
      message,
      severity
    });
    setEditModalOpen(false);
    setSelectedModelForEdit(null);
    
    // Refresh models list if edit was successful
    if (severity === 'success') {
      fetchModels();
    }
  };

  const handleUploadModel = () => {
    setUploadModalOpen(true);
  };

  const handleUploadComplete = (message, severity) => {
    setSnackbar({
      open: true,
      message,
      severity
    });
    setUploadModalOpen(false);
    
    // Refresh models list if upload was successful
    if (severity === 'success') {
      fetchModels();
    }
  };

  const handleEditModelForUpload = () => {
    setEditModelForUploadModalOpen(true);
  };

  const handleEditModelForUploadComplete = (message, severity) => {
    setSnackbar({
      open: true,
      message,
      severity
    });
    setEditModelForUploadModalOpen(false);
    
    // Refresh models list if edit was successful
    if (severity === 'success') {
      fetchModels();
    }
  };

  useEffect(() => {
    fetchHosts();
    fetchExistingModelNames();
    fetchModels();
    fetchSourceModels();
  }, [fetchHosts, fetchExistingModelNames, fetchModels, fetchSourceModels]);

  return (
    <Box sx={{ p: 3 }}>
      <GenericModelTableComponent
        models={models}
        loading={modelsLoading}
        handleCloneAction={handleCloneAction}
        handlePromoteAction={handlePromoteAction}
        handleRemoveAction={handleRemoveAction}
        handleOnboardModel={handleOnboardClick}
        handleTestModel={handleTestModel}
        handleEditModel={handleEditModel}
        handleUploadModel={handleUploadModel}
        handleEditModelForUpload={handleEditModelForUpload}
      />

      {/* Onboard Model Dialog */}
      <Dialog
        open={openDialog}
        onClose={handleClose}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="h6">
              Onboard Models ({modelForms.length})
            </Typography>
            <Button
              startIcon={<Add />}
              onClick={handleAddModel}
              variant="outlined"
              size="small"
              sx={{
                color: '#450839',
                borderColor: '#450839',
                '&:hover': {
                  backgroundColor: '#450839',
                  color: 'white',
                  borderColor: '#450839',
                },
              }}
            >
              Add Model
            </Button>
          </Box>
        </DialogTitle>

        <DialogContent dividers>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, my: 2 }}>
            {modelForms.map((modelForm, modelIndex) => (
              <Card key={modelForm.id} variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                    <Typography variant="h6">
                      Model {modelIndex + 1}
                    </Typography>
                    {modelForms.length > 1 && (
                      <IconButton
                        onClick={() => handleRemoveModel(modelIndex)}
                        color="error"
                        size="small"
                      >
                        <Delete />
                      </IconButton>
                    )}
                  </Box>

                  <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2, boxShadow: 'none'}}>
                    {/* Model Name Dropdown */}
                    <Box sx={{ position: 'relative' }}>
                      <Autocomplete
                        options={getAvailableModelOptions(modelIndex).sort((a, b) => (a.name || '').localeCompare(b.name || ''))}
                        getOptionLabel={(option) => option.name}
                        value={getAvailableModelOptions(modelIndex).find(option => option.name === modelForm.selectedModelName) || null}
                        onChange={(event, newValue) => {
                          if (newValue && newValue.name) {
                            setModelForms(prev => {
                              const updated = [...prev];
                              updated[modelIndex] = {
                                ...updated[modelIndex],
                                selectedModelName: newValue.name,
                                errors: {
                                  ...updated[modelIndex].errors,
                                  selectedModelName: ''
                                }
                              };
                              return updated;
                            });
                            
                            handlePublishClickWithModelName(modelIndex, newValue.name);
                          } else {
                            const syntheticEvent = {
                              target: {
                                name: 'selectedModelName',
                                value: ''
                              }
                            };
                            handleChange(syntheticEvent, modelIndex);
                          }
                        }}
                        disabled={sourceModelsLoading}
                        disableClearable={true}
                        renderInput={(params) => (
                          <TextField
                            {...params}
                            label="Model Name"
                            name="selectedModelName"
                            required
                            error={!!modelForm.errors.selectedModelName}
                            helperText={modelForm.errors.selectedModelName}
                            InputProps={{
                              ...params.InputProps,
                              endAdornment: (
                                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                                  {modelForm.isLoadingModelParams && (
                                    <CircularProgress size={24} sx={{ mr: 1 }} />
                                  )}
                                  {params.InputProps.endAdornment}
                                </Box>
                              ),
                            }}
                          />
                        )}
                        isOptionEqualToValue={(option, value) => option.name === value.name}
                        noOptionsText="No available models (all selected or loading...)"
                        filterOptions={(options, { inputValue }) =>
                          options.filter(option =>
                            option.name.toLowerCase().includes(inputValue.toLowerCase())
                          )
                        }
                      />
                    </Box>



                    {/* Only show fields when model is selected and params are loaded */}
                    {modelForm.selectedModelName && modelForm.modelParamsLoaded && (
                      <>
                        {/* Only show these fields for non-ensemble models and when they have values */}
                        {(!modelForm.platform || modelForm.platform !== 'ensemble') && (
                          <>
                            {modelForm.instanceCount && (
                              <TextField
                                label="Instance Count"
                                name="instanceCount"
                                value={modelForm.instanceCount}
                                InputProps={{ readOnly: true }}
                                required
                                fullWidth
                                type="number"
                                disabled
                              />
                            )}

                            {modelForm.instanceType && (
                              <TextField
                                label="Instance Type"
                                name="instanceType"
                                value={modelForm.instanceType}
                                InputProps={{ readOnly: true }}
                                required
                                fullWidth
                                disabled
                              />
                            )}

                            {modelForm.backend && (
                              <TextField
                                label="Backend"
                                name="backend"
                                value={modelForm.backend}
                                InputProps={{ readOnly: true }}
                                required
                                fullWidth
                                disabled
                              />
                            )}

                            {modelForm.dynamicBatchingEnabled !== undefined && (
                              <TextField
                                label="Dynamic Batching Enabled"
                                name="dynamicBatchingEnabled"
                                value={String(modelForm.dynamicBatchingEnabled)}
                                InputProps={{ readOnly: true }}
                                fullWidth
                                disabled
                              />
                            )}
                          </>
                        )}

                        {/* Platform for ensemble models - only show when available */}
                        {modelForm.platform === 'ensemble' && (
                          <TextField
                            label="Platform"
                            name="platform"
                            value={modelForm.platform}
                            InputProps={{ readOnly: true }}
                            fullWidth
                            disabled
                          />
                        )}

                        {/* Batch Size - only show when available */}
                        {modelForm.batchSize && (
                          <TextField
                            label="Batch Size"
                            name="batchSize"
                            value={modelForm.batchSize}
                            InputProps={{ readOnly: true }}
                            required
                            fullWidth
                            type="number"
                            disabled
                          />
                        )}

                        {/* Inputs Display with fields in single line */}
                        {modelForm.inputs && modelForm.inputs.length > 0 && (
                          <>
                            <Typography variant="subtitle1" sx={{ mt: 2 }}>
                              Input Parameters
                            </Typography>
                            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                              {modelForm.inputs.map((input, index) => (
                                <Box key={index} sx={{ border: '1px solid #e0e0e0', borderRadius: 1, p: 2, backgroundColor: '#f9f9f9' }}>
                                  <Grid container spacing={2}>
                                    <Grid item xs={4}>
                                      <TextField
                                        label="Name"
                                        value={input.name}
                                        InputProps={{ readOnly: true }}
                                        fullWidth
                                        size="small"
                                      />
                                    </Grid>
                                    <Grid item xs={4}>
                                      <TextField
                                        label="Dims"
                                        value={getDims(input)}
                                        InputProps={{ readOnly: true }}
                                        fullWidth
                                        size="small"
                                      />
                                    </Grid>
                                    <Grid item xs={4}>
                                      <TextField
                                        label="Data Type"
                                        value={input.data_type}
                                        InputProps={{ readOnly: true }}
                                        fullWidth
                                        size="small"
                                      />
                                    </Grid>
                                    {input.features && input.features.length > 0 && (
                                      <Grid item xs={12}>
                                        <Box sx={{ mt: 1 }}>
                                          <FeaturesDisplay features={input.features} />
                                        </Box>
                                      </Grid>
                                    )}
                                  </Grid>
                                </Box>
                              ))}
                            </Box>
                          </>
                        )}

                        {/* Outputs Display with fields in single line */}
                        {modelForm.outputs && modelForm.outputs.length > 0 && (
                          <>
                            <Typography variant="subtitle1" sx={{ mt: 2 }}>
                              Output Parameters
                            </Typography>
                            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                              {modelForm.outputs.map((output, index) => (
                                <Box key={index} sx={{ border: '1px solid #e0e0e0', borderRadius: 1, p: 2, backgroundColor: '#f9f9f9' }}>
                                  <Grid container spacing={2}>
                                    <Grid item xs={4}>
                                      <TextField
                                        label="Name"
                                        value={output.name}
                                        InputProps={{ readOnly: true }}
                                        fullWidth
                                        size="small"
                                      />
                                    </Grid>
                                    <Grid item xs={4}>
                                      <TextField
                                        label="Dims"
                                        value={getDims(output)}
                                        InputProps={{ readOnly: true }}
                                        fullWidth
                                        size="small"
                                      />
                                    </Grid>
                                    <Grid item xs={4}>
                                      <TextField
                                        label="Data Type"
                                        value={output.data_type}
                                        InputProps={{ readOnly: true }}
                                        fullWidth
                                        size="small"
                                      />
                                    </Grid>
                                  </Grid>
                                </Box>
                              ))}
                            </Box>
                          </>
                        )}

                        {/* Ensemble Scheduling section - only show when not empty */}
                        {modelForm.ensembleScheduling && Object.keys(modelForm.ensembleScheduling).length > 0 && (
                          <>
                            <Typography variant="subtitle1" sx={{ mt: 2 }}>
                              Ensemble Scheduling
                            </Typography>
                            <JsonViewer
                              data={modelForm.ensembleScheduling}
                              title="Ensemble Scheduling Configuration"
                              defaultExpanded={true}
                              maxHeight={400}
                              enableCopy={true}
                              enableDiff={false}
                              editable={false}
                              sx={{ mb: 2 }}
                            />
                          </>
                        )}
                      </>
                    )}


                  </Box>
                </CardContent>
              </Card>
            ))}

            {/* Host - Editable for all models, placed at the end */}
            <Box sx={{ mt: 3 }}>
              <Autocomplete
                options={[...(hosts || [])].sort((a, b) => (a.host || '').localeCompare(b.host || ''))}
                getOptionLabel={(option) => `${option.host} (${option.name})`}
                value={hosts.find(host => host.id === onboardFormData.selectedHost) || null}
                onChange={(event, newValue) => {
                  const syntheticEvent = {
                    target: {
                      name: 'selectedHost',
                      value: newValue?.id || ''
                    }
                  };
                  handleOnboardFormChange(syntheticEvent);
                }}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Host"
                    name="selectedHost"
                    required
                    fullWidth
                  />
                )}
              />
            </Box>
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
                backgroundColor: '#450839',
                color: 'white',
                borderColor: '#450839',
              },
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            disabled={isSubmitting || modelForms.length === 0}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              },
            }}
          >
            {isSubmitting ? (
              <CircularProgress size={24} color="inherit" />
            ) : `Onboard (${modelForms.length})`}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Scale Up Dialog */}
      <Dialog
        open={openScaleUpDialog}
        onClose={handleCloseScaleUpDialog}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Scale Up {selectedModelsForAction.length > 1 ? `Models (${selectedModelsForAction.length})` : 'Model'}
        </DialogTitle>

        <DialogContent dividers>
          <Box sx={{ mb: 3 }}>
            {/* Global error for model names */}
            {modelNameError && (
              <Typography variant="body2" color="error" sx={{ mb: 2 }}>
                {modelNameError}
              </Typography>
            )}
          </Box>

          {/* Model List with details */}
          {selectedModelsForAction.map((model, modelIndex) => (
            <Box
              key={modelIndex}
              sx={{
                border: '1px solid #e0e0e0',
                borderRadius: 1,
                p: 2,
                mb: 3,
                backgroundColor: modelIndex % 2 === 0 ? '#f9f9f9' : 'white'
              }}
            >
              <Typography variant="h6" sx={{ mb: 2 }}>
                Model {modelIndex + 1}
              </Typography>

              <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {/* Model Name - Textbox */}
                <TextField
                  fullWidth
                  required
                  label="Model Name"
                  value={model.selectedModelName}
                  onChange={(e) => handleModelNameChange(modelIndex, e.target.value)}
                  error={!!modelNameError && existingModelNames.includes(model.selectedModelName)}
                  helperText={modelNameError && existingModelNames.includes(model.selectedModelName) ? "This model name already exists" : ""}
                  disabled={sourceModelsLoading}
                />

                {/* For non-ensemble models */}
                {(!model.metaData?.platform || model.metaData?.platform !== 'ensemble') && (
                  <>
                    <TextField
                      label="Instance Count"
                      value={model.metaData?.instance_count || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Instance Type"
                      value={model.metaData?.instance_type || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Batch Size"
                      value={model.metaData?.batch_size || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Backend"
                      value={model.metaData?.backend || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Dynamic Batching Enabled"
                      value={model.metaData?.dynamic_batching_enabled === undefined ? 'false' :
                        String(model.metaData.dynamic_batching_enabled)}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />
                  </>
                )}

                {/* For ensemble models */}
                {model.metaData?.platform === 'ensemble' && (
                  <>
                    <TextField
                      label="Batch Size"
                      value={model.metaData?.batch_size || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Platform"
                      value={model.metaData?.platform || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />
                  </>
                )}

                {/* Input Parameters */}
                <Typography variant="subtitle1" sx={{ mt: 2 }}>
                  Input Parameters
                </Typography>

                {model.metaData?.inputs && model.metaData.inputs.length > 0 ? (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    {model.metaData.inputs.map((input, index) => (
                      <Box key={index} sx={{ border: '1px solid #e0e0e0', borderRadius: 1, p: 2, backgroundColor: '#f9f9f9' }}>
                        <Grid container spacing={2}>
                          <Grid item xs={4}>
                            <TextField
                              label="Name"
                              value={input.name}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Dims"
                              value={getDims(input)}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Data Type"
                              value={input.data_type}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={12}>
                            <Box sx={{ mt: 1 }}>
                              <FeaturesDisplay features={input.features} />
                            </Box>
                          </Grid>
                        </Grid>
                      </Box>
                    ))}
                  </Box>
                ) : (
                  <Typography variant="body2" color="textSecondary">
                    No input parameters available.
                  </Typography>
                )}

                {/* Output Parameters */}
                <Typography variant="subtitle1" sx={{ mt: 2 }}>
                  Output Parameters
                </Typography>

                {model.metaData?.outputs && model.metaData.outputs.length > 0 ? (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    {model.metaData.outputs.map((output, index) => (
                      <Box key={index} sx={{ border: '1px solid #e0e0e0', borderRadius: 1, p: 2, backgroundColor: '#f9f9f9' }}>
                        <Grid container spacing={2}>
                          <Grid item xs={4}>
                            <TextField
                              label="Name"
                              value={output.name}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Dims"
                              value={getDims(output)}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Data Type"
                              value={output.data_type}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                        </Grid>
                      </Box>
                    ))}
                  </Box>
                ) : (
                  <Typography variant="body2" color="textSecondary">
                    No output parameters available.
                  </Typography>
                )}

                {/* Ensemble Scheduling */}
                {model.metaData?.platform === 'ensemble' && model.metaData?.ensemble_scheduling && (
                  <>
                    <Typography variant="subtitle1" sx={{ mt: 2 }}>
                      Ensemble Scheduling
                    </Typography>

                    <JsonViewer
                      data={model.metaData.ensemble_scheduling}
                      title="Ensemble Scheduling Configuration"
                      defaultExpanded={false}
                      maxHeight={200}
                      enableCopy={true}
                      enableDiff={false}
                      editable={false}
                      sx={{ mb: 2 }}
                    />
                  </>
                )}
              </Box>
            </Box>
          ))}

          {/* Host - Editable for all models, placed at the end */}
          <Box sx={{ mt: 3 }}>
            <Autocomplete
              options={[...(hosts || [])].sort((a, b) => (a.host || '').localeCompare(b.host || ''))}
              getOptionLabel={(option) => `${option.host} (${option.name})`}
              value={hosts.find(host => host.id === scaleUpFormData.selectedHost) || null}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'selectedHost',
                    value: newValue?.id || ''
                  }
                };
                handleScaleUpFormChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Host"
                  name="selectedHost"
                  required
                  fullWidth
                />
              )}
            />
          </Box>
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleCloseScaleUpDialog}
            variant="outlined"
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={processScaleUp}
            variant="contained"
            disabled={isSubmitting || selectedModelsForAction.length === 0}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              },
            }}
          >
            {isSubmitting ? (
              <CircularProgress size={24} color="inherit" />
            ) : `Scale Up (${selectedModelsForAction.length})`}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Promote Dialog */}
      <Dialog
        open={openPromoteDialog}
        onClose={handleClosePromoteDialog}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Promote {selectedModelsForAction.length > 1 ? `Models (${selectedModelsForAction.length})` : 'Model'}
        </DialogTitle>

        <DialogContent dividers>
          <Box sx={{ mb: 3 }}>
            {/* Global error for model names */}
            {modelNameError && (
              <Typography variant="body2" color="error" sx={{ mb: 2 }}>
                {modelNameError}
              </Typography>
            )}
          </Box>

          {/* Model List with details */}
          {selectedModelsForAction.map((model, modelIndex) => (
            <Box
              key={modelIndex}
              sx={{
                border: '1px solid #e0e0e0',
                borderRadius: 1,
                p: 2,
                mb: 3,
                backgroundColor: modelIndex % 2 === 0 ? '#f9f9f9' : 'white'
              }}
            >
              <Typography variant="h6" sx={{ mb: 2 }}>
                Model {modelIndex + 1}
              </Typography>

              <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {/* Model Name - Not Editable */}
                <TextField
                  label="Model Name"
                  value={model.modelName}
                  InputProps={{ readOnly: true }}
                  fullWidth
                />

                {/* For non-ensemble models */}
                {(!model.metaData?.platform || model.metaData?.platform !== 'ensemble') && (
                  <>
                    <TextField
                      label="Instance Count"
                      value={model.metaData?.instance_count || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Instance Type"
                      value={model.metaData?.instance_type || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Batch Size"
                      value={model.metaData?.batch_size || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Backend"
                      value={model.metaData?.backend || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Dynamic Batching Enabled"
                      value={model.metaData?.dynamic_batching_enabled === undefined ? 'false' :
                        String(model.metaData.dynamic_batching_enabled)}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />
                  </>
                )}

                {/* For ensemble models */}
                {model.metaData?.platform === 'ensemble' && (
                  <>
                    <TextField
                      label="Batch Size"
                      value={model.metaData?.batch_size || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />

                    <TextField
                      label="Platform"
                      value={model.metaData?.platform || ''}
                      InputProps={{ readOnly: true }}
                      fullWidth
                    />
                  </>
                )}

                {/* Input Parameters */}
                <Typography variant="subtitle1" sx={{ mt: 2 }}>
                  Input Parameters
                </Typography>

                {model.metaData?.inputs && model.metaData.inputs.length > 0 ? (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    {model.metaData.inputs.map((input, index) => (
                      <Box key={index} sx={{ border: '1px solid #e0e0e0', borderRadius: 1, p: 2, backgroundColor: '#f9f9f9' }}>
                        <Grid container spacing={2}>
                          <Grid item xs={4}>
                            <TextField
                              label="Name"
                              value={input.name}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Dims"
                              value={getDims(input)}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Data Type"
                              value={input.data_type}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={12}>
                            <Box sx={{ mt: 1 }}>
                              <FeaturesDisplay features={input.features} />
                            </Box>
                          </Grid>
                        </Grid>
                      </Box>
                    ))}
                  </Box>
                ) : (
                  <Typography variant="body2" color="textSecondary">
                    No input parameters available.
                  </Typography>
                )}

                {/* Output Parameters */}
                <Typography variant="subtitle1" sx={{ mt: 2 }}>
                  Output Parameters
                </Typography>

                {model.metaData?.outputs && model.metaData.outputs.length > 0 ? (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    {model.metaData.outputs.map((output, index) => (
                      <Box key={index} sx={{ border: '1px solid #e0e0e0', borderRadius: 1, p: 2, backgroundColor: '#f9f9f9' }}>
                        <Grid container spacing={2}>
                          <Grid item xs={4}>
                            <TextField
                              label="Name"
                              value={output.name}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Dims"
                              value={getDims(output)}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                          <Grid item xs={4}>
                            <TextField
                              label="Data Type"
                              value={output.data_type}
                              InputProps={{ readOnly: true }}
                              fullWidth
                              size="small"
                            />
                          </Grid>
                        </Grid>
                      </Box>
                    ))}
                  </Box>
                ) : (
                  <Typography variant="body2" color="textSecondary">
                    No output parameters available.
                  </Typography>
                )}

                {/* Ensemble Scheduling */}
                {model.metaData?.platform === 'ensemble' && model.metaData?.ensemble_scheduling && (
                  <>
                    <Typography variant="subtitle1" sx={{ mt: 2 }}>
                      Ensemble Scheduling
                    </Typography>

                    <JsonViewer
                      data={model.metaData.ensemble_scheduling}
                      title="Ensemble Scheduling Configuration"
                      defaultExpanded={false}
                      maxHeight={200}
                      enableCopy={true}
                      enableDiff={false}
                      editable={false}
                      sx={{ mb: 2 }}
                    />
                  </>
                )}
              </Box>
            </Box>
          ))}

          {/* Host - Editable for all models, placed at the end */}
          <Box sx={{ mt: 3 }}>
            <Autocomplete
              options={[...(promoteHosts || [])].sort((a, b) => (a.host || '').localeCompare(b.host || ''))}
              getOptionLabel={(option) => `${option.host} (${option.name})`}
              value={promoteHosts.find(host => host.id === promoteFormData.selectedHost) || null}
              onChange={(event, newValue) => {
                const syntheticEvent = {
                  target: {
                    name: 'selectedHost',
                    value: newValue?.id || ''
                  }
                };
                handlePromoteFormChange(syntheticEvent);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Host"
                  name="selectedHost"
                  required
                  fullWidth
                />
              )}
            />
          </Box>
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleClosePromoteDialog}
            variant="outlined"
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={processPromote}
            variant="contained"
            disabled={isSubmitting || selectedModelsForAction.length === 0}
            sx={{
              backgroundColor: '#450839',
              '&:hover': {
                backgroundColor: '#380730',
              },
            }}
          >
            {isSubmitting ? (
              <CircularProgress size={24} color="inherit" />
            ) : `Promote (${selectedModelsForAction.length})`}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Test Modal */}
      <ModelTestingModal
        open={testModalOpen}
        onClose={() => setTestModalOpen(false)}
        model={selectedModelForTest}
        onTestComplete={handleTestComplete}
      />

      {/* Edit Modal */}
      <EditModelModal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        model={selectedModelForEdit}
        onEditComplete={handleEditComplete}
      />

      {/* Upload Modal */}
      <UploadModelModal
        open={uploadModalOpen}
        onClose={() => setUploadModalOpen(false)}
        onUploadComplete={handleUploadComplete}
      />

      {/* Edit Model For Upload Modal */}
      <EditModelModalForUpload
        open={editModelForUploadModalOpen}
        onClose={() => setEditModelForUploadModalOpen(false)}
        onUploadComplete={handleEditModelForUploadComplete}
      />

      {/* Production Credential Modal */}
      <ProductionCredentialModal
        open={productionCredentialModalOpen}
        onClose={handleProductionCredentialClose}
        onSuccess={handleProductionCredentialSuccess}
        title="Production Credential Verification"
        description="Please enter your production credentials to proceed with the model promotion."
      />

      {/* Success/Error Notification */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={4000}
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

      {/* Loading Backdrop for Promote Setup */}
      <Backdrop
        sx={{
          color: '#fff',
          zIndex: (theme) => theme.zIndex.drawer + 1,
          backgroundColor: 'rgba(0, 0, 0, 0.7)',
          display: 'flex',
          flexDirection: 'column',
          gap: 2
        }}
        open={isLoadingPromoteSetup}
      >
        <CircularProgress color="inherit" size={60} thickness={4} />
        <Typography variant="h6" sx={{ fontWeight: 500 }}>
          Preparing Promotion...
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.8 }}>
          Fetching production hosts and validating credentials
        </Typography>
      </Backdrop>
    </Box>
  );
};

export default ModelRegistry;
