import React, { useState, useEffect } from 'react';
import {
  Paper,
  Box,
  Typography,
  Button,
  Tabs,
  Tab,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Grid,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Snackbar,
  Alert,
  FormHelperText,
  Divider,
  Chip
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import CloudIcon from '@mui/icons-material/Cloud';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunch';
import LaunchIcon from '@mui/icons-material/Launch';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';

const DeploymentRegistry = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [open, setOpen] = useState(false);
  const [operationType, setOperationType] = useState('cluster'); // 'cluster', 'promotion', 'onboarding'
  const { user } = useAuth();
  const [notification, setNotification] = useState({ open: false, message: "", severity: "success" });
  const [entities, setEntities] = useState([]);
  const [models, setModels] = useState([]);
  const [variants, setVariants] = useState([]);
  const [clusters, setClusters] = useState([]);
  const [validationErrors, setValidationErrors] = useState({});

  // Cluster Creation Form State
  const [clusterData, setClusterData] = useState({
    requestor: '',
    reason: '',
    node_count: 3,
    instance_type: 'm5.large',
    storage: '100GB',
    qdrant_version: 'v1.7.0',
    dns_subdomain: '',
    project: 'embedding-platform'
  });

  // Variant Promotion Form State
  const [promotionData, setPromotionData] = useState({
    requestor: '',
    reason: '',
    entity: '',
    model: '',
    variant: '',
    vector_db_type: 'QDRANT',
    traffic_percentage: 50,
    rollout_strategy: 'CANARY',
    monitoring_duration: '24h',
    error_rate_threshold: 5,
    latency_threshold: 500,
    target_host: ''
  });

  // Variant Onboarding Form State
  const [onboardingData, setOnboardingData] = useState({
    requestor: '',
    reason: '',
    entity: '',
    model: '',
    variant: '',
    vector_db_type: 'QDRANT',
    cluster_endpoint: '',
    collection_name: '',
    vector_size: 128,
    distance: 'EUCLIDEAN',
    index_type: 'HNSW',
    index_m: 16,
    index_ef_construct: 100,
    alerts_enabled: true,
    metrics_dashboard: true,
    log_level: 'INFO'
  });

  const instanceTypes = ['m5.large', 'm5.xlarge', 'm5.2xlarge', 'm5.4xlarge'];
  const qdrantVersions = ['v1.7.0', 'v1.6.1', 'v1.5.0'];
  const rolloutStrategies = ['CANARY', 'BLUE_GREEN', 'IMMEDIATE'];
  const distanceTypes = ['EUCLIDEAN', 'COSINE', 'DOT'];
  const logLevels = ['DEBUG', 'INFO', 'WARN', 'ERROR'];

  useEffect(() => {
    fetchDependencies();
  }, []);

  const fetchDependencies = async () => {
    try {
      // Fetch entities, models, variants, and clusters for form dropdowns
      const [entitiesRes, modelsRes, variantsRes, clustersRes] = await Promise.all([
        embeddingPlatformAPI.getEntities(),
        embeddingPlatformAPI.getModels(),
        embeddingPlatformAPI.getVariants(),
        embeddingPlatformAPI.getQdrantClusters()
      ]);

      if (entitiesRes.entities) setEntities(entitiesRes.entities);
      if (modelsRes.models) setModels(modelsRes.models);
      if (variantsRes.variants) setVariants(variantsRes.variants);
      if (clustersRes.data?.clusters) setClusters(clustersRes.data.clusters);
    } catch (error) {
      console.error('Error fetching dependencies:', error);
    }
  };

  const handleTabChange = (event, newValue) => {
    setActiveTab(newValue);
  };

  const handleOpen = (type) => {
    setOperationType(type);
    setValidationErrors({});
    
    // Set requestor based on user
    const requestor = user?.email || '';
    
    if (type === 'cluster') {
      setClusterData(prev => ({ ...prev, requestor }));
    } else if (type === 'promotion') {
      setPromotionData(prev => ({ ...prev, requestor }));
    } else if (type === 'onboarding') {
      setOnboardingData(prev => ({ ...prev, requestor }));
    }
    
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
    setValidationErrors({});
  };

  const handleClusterChange = (e) => {
    const { name, value } = e.target;
    setClusterData(prev => ({
      ...prev,
      [name]: value
    }));
    if (validationErrors[name]) {
      setValidationErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const handlePromotionChange = (e) => {
    const { name, value } = e.target;
    setPromotionData(prev => ({
      ...prev,
      [name]: value
    }));
    if (validationErrors[name]) {
      setValidationErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const handleOnboardingChange = (e) => {
    const { name, value, type, checked } = e.target;
    setOnboardingData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
    if (validationErrors[name]) {
      setValidationErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const validateClusterForm = () => {
    const errors = {};
    if (!clusterData.reason) errors.reason = 'Reason is required';
    if (!clusterData.dns_subdomain) errors.dns_subdomain = 'DNS subdomain is required';
    if (clusterData.node_count < 1) errors.node_count = 'Node count must be at least 1';
    return errors;
  };

  const validatePromotionForm = () => {
    const errors = {};
    if (!promotionData.reason) errors.reason = 'Reason is required';
    if (!promotionData.entity) errors.entity = 'Entity is required';
    if (!promotionData.model) errors.model = 'Model is required';
    if (!promotionData.variant) errors.variant = 'Variant is required';
    if (!promotionData.target_host) errors.target_host = 'Target host is required';
    if (promotionData.traffic_percentage < 1 || promotionData.traffic_percentage > 100) {
      errors.traffic_percentage = 'Traffic percentage must be between 1 and 100';
    }
    return errors;
  };

  const validateOnboardingForm = () => {
    const errors = {};
    if (!onboardingData.reason) errors.reason = 'Reason is required';
    if (!onboardingData.entity) errors.entity = 'Entity is required';
    if (!onboardingData.model) errors.model = 'Model is required';
    if (!onboardingData.variant) errors.variant = 'Variant is required';
    if (!onboardingData.cluster_endpoint) errors.cluster_endpoint = 'Cluster endpoint is required';
    if (!onboardingData.collection_name) errors.collection_name = 'Collection name is required';
    if (onboardingData.vector_size < 1) errors.vector_size = 'Vector size must be at least 1';
    return errors;
  };

  const handleSubmit = async () => {
    let errors = {};
    let apiCall = null;
    let payload = {};

    if (operationType === 'cluster') {
      errors = validateClusterForm();
      if (Object.keys(errors).length === 0) {
        payload = {
          requestor: clusterData.requestor,
          reason: clusterData.reason,
          payload: {
            node_conf: {
              count: parseInt(clusterData.node_count),
              instance_type: clusterData.instance_type,
              storage: clusterData.storage
            },
            qdrant_version: clusterData.qdrant_version,
            dns_subdomain: clusterData.dns_subdomain,
            project: clusterData.project
          }
        };
        apiCall = () => embeddingPlatformAPI.createQdrantCluster(payload);
      }
    } else if (operationType === 'promotion') {
      errors = validatePromotionForm();
      if (Object.keys(errors).length === 0) {
        payload = {
          requestor: promotionData.requestor,
          reason: promotionData.reason,
          payload: {
            entity: promotionData.entity,
            model: promotionData.model,
            variant: promotionData.variant,
            vector_db_type: promotionData.vector_db_type,
            promotion_config: {
              traffic_percentage: parseInt(promotionData.traffic_percentage),
              rollout_strategy: promotionData.rollout_strategy,
              monitoring_duration: promotionData.monitoring_duration,
              rollback_threshold: {
                error_rate: parseInt(promotionData.error_rate_threshold),
                latency_p99: parseInt(promotionData.latency_threshold)
              }
            },
            target_host: promotionData.target_host
          }
        };
        apiCall = () => embeddingPlatformAPI.promoteVariant(payload);
      }
    } else if (operationType === 'onboarding') {
      errors = validateOnboardingForm();
      if (Object.keys(errors).length === 0) {
        payload = {
          requestor: onboardingData.requestor,
          reason: onboardingData.reason,
          payload: {
            entity: onboardingData.entity,
            model: onboardingData.model,
            variant: onboardingData.variant,
            vector_db_type: onboardingData.vector_db_type,
            onboarding_config: {
              cluster_endpoint: onboardingData.cluster_endpoint,
              collection_setup: {
                collection_name: onboardingData.collection_name,
                vector_size: parseInt(onboardingData.vector_size),
                distance: onboardingData.distance,
                index_config: {
                  type: onboardingData.index_type,
                  m: parseInt(onboardingData.index_m),
                  ef_construct: parseInt(onboardingData.index_ef_construct)
                }
              },
              monitoring: {
                alerts_enabled: onboardingData.alerts_enabled,
                metrics_dashboard: onboardingData.metrics_dashboard,
                log_level: onboardingData.log_level
              }
            }
          }
        };
        apiCall = () => embeddingPlatformAPI.onboardVariant(payload);
      }
    }

    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors);
      return;
    }

    try {
      await apiCall();
      setNotification({
        open: true,
        message: `${operationType === 'cluster' ? 'Cluster creation' : operationType === 'promotion' ? 'Variant promotion' : 'Variant onboarding'} request submitted successfully!`,
        severity: 'success'
      });
      handleClose();
      
      // Reset form
      if (operationType === 'cluster') {
        setClusterData({
          requestor: user?.email || '',
          reason: '',
          node_count: 3,
          instance_type: 'm5.large',
          storage: '100GB',
          qdrant_version: 'v1.7.0',
          dns_subdomain: '',
          project: 'embedding-platform'
        });
      }
      // Reset other forms similarly...
      
    } catch (error) {
      console.error(`Error submitting ${operationType} request:`, error);
      setNotification({
        open: true,
        message: error.message || `Failed to submit ${operationType} request. Please try again.`,
        severity: 'error'
      });
    }
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({ ...prev, open: false }));
  };

  const getDialogTitle = () => {
    switch (operationType) {
      case 'cluster': return 'Create Qdrant Cluster';
      case 'promotion': return 'Promote Variant';
      case 'onboarding': return 'Onboard Variant';
      default: return 'Deployment Operation';
    }
  };

  const getDialogIcon = () => {
    switch (operationType) {
      case 'cluster': return <CloudIcon />;
      case 'promotion': return <RocketLaunchIcon />;
      case 'onboarding': return <LaunchIcon />;
      default: return <AddIcon />;
    }
  };

  const renderClusterForm = () => (
    <Grid container spacing={2}>
      <Grid item xs={12}>
        <TextField
          fullWidth
          required
          multiline
          rows={3}
          size="small"
          name="reason"
          label="Reason for Cluster Creation"
          value={clusterData.reason}
          onChange={handleClusterChange}
          error={!!validationErrors.reason}
          helperText={validationErrors.reason || "Explain why this cluster is needed"}
          placeholder="Setting up new Qdrant cluster for production workloads with high availability"
        />
      </Grid>
      <Grid item xs={6}>
        <TextField
          fullWidth
          required
          size="small"
          name="dns_subdomain"
          label="DNS Subdomain"
          value={clusterData.dns_subdomain}
          onChange={handleClusterChange}
          error={!!validationErrors.dns_subdomain}
          helperText={validationErrors.dns_subdomain || "Will create subdomain.meesho.int"}
          placeholder="qdrant-prod"
        />
      </Grid>
      <Grid item xs={6}>
        <FormControl fullWidth size="small">
          <InputLabel>Qdrant Version</InputLabel>
          <Select
            name="qdrant_version"
            value={clusterData.qdrant_version}
            onChange={handleClusterChange}
            label="Qdrant Version"
          >
            {qdrantVersions.map(version => (
              <MenuItem key={version} value={version}>{version}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <TextField
          fullWidth
          required
          size="small"
          type="number"
          name="node_count"
          label="Node Count"
          value={clusterData.node_count}
          onChange={handleClusterChange}
          error={!!validationErrors.node_count}
          helperText={validationErrors.node_count}
          inputProps={{ min: 1, max: 10 }}
        />
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small">
          <InputLabel>Instance Type</InputLabel>
          <Select
            name="instance_type"
            value={clusterData.instance_type}
            onChange={handleClusterChange}
            label="Instance Type"
          >
            {instanceTypes.map(type => (
              <MenuItem key={type} value={type}>{type}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <TextField
          fullWidth
          size="small"
          name="storage"
          label="Storage per Node"
          value={clusterData.storage}
          onChange={handleClusterChange}
          placeholder="100GB"
        />
      </Grid>
    </Grid>
  );

  const renderPromotionForm = () => (
    <Grid container spacing={2}>
      <Grid item xs={12}>
        <TextField
          fullWidth
          required
          multiline
          rows={3}
          size="small"
          name="reason"
          label="Reason for Promotion"
          value={promotionData.reason}
          onChange={handlePromotionChange}
          error={!!validationErrors.reason}
          helperText={validationErrors.reason || "Explain why this variant should be promoted"}
          placeholder="Experiment variant showing 15% improvement in CTR. Ready for production traffic."
        />
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small" required error={!!validationErrors.entity}>
          <InputLabel>Entity</InputLabel>
          <Select
            name="entity"
            value={promotionData.entity}
            onChange={handlePromotionChange}
            label="Entity"
          >
            {entities.map(entity => (
              <MenuItem key={entity.entity} value={entity.entity}>{entity.entity}</MenuItem>
            ))}
          </Select>
          {validationErrors.entity && <FormHelperText>{validationErrors.entity}</FormHelperText>}
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small" required error={!!validationErrors.model}>
          <InputLabel>Model</InputLabel>
          <Select
            name="model"
            value={promotionData.model}
            onChange={handlePromotionChange}
            label="Model"
          >
            {models.filter(m => m.entity === promotionData.entity).map(model => (
              <MenuItem key={model.model} value={model.model}>{model.model}</MenuItem>
            ))}
          </Select>
          {validationErrors.model && <FormHelperText>{validationErrors.model}</FormHelperText>}
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small" required error={!!validationErrors.variant}>
          <InputLabel>Variant</InputLabel>
          <Select
            name="variant"
            value={promotionData.variant}
            onChange={handlePromotionChange}
            label="Variant"
          >
            {variants.filter(v => v.entity === promotionData.entity && v.model === promotionData.model).map(variant => (
              <MenuItem key={variant.variant} value={variant.variant}>{variant.variant}</MenuItem>
            ))}
          </Select>
          {validationErrors.variant && <FormHelperText>{validationErrors.variant}</FormHelperText>}
        </FormControl>
      </Grid>
      <Grid item xs={6}>
        <TextField
          fullWidth
          required
          size="small"
          name="target_host"
          label="Target Host"
          value={promotionData.target_host}
          onChange={handlePromotionChange}
          error={!!validationErrors.target_host}
          helperText={validationErrors.target_host}
          placeholder="qdrant-prod.meesho.int"
        />
      </Grid>
      <Grid item xs={6}>
        <FormControl fullWidth size="small">
          <InputLabel>Rollout Strategy</InputLabel>
          <Select
            name="rollout_strategy"
            value={promotionData.rollout_strategy}
            onChange={handlePromotionChange}
            label="Rollout Strategy"
          >
            {rolloutStrategies.map(strategy => (
              <MenuItem key={strategy} value={strategy}>{strategy}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <TextField
          fullWidth
          size="small"
          type="number"
          name="traffic_percentage"
          label="Traffic Percentage"
          value={promotionData.traffic_percentage}
          onChange={handlePromotionChange}
          error={!!validationErrors.traffic_percentage}
          helperText={validationErrors.traffic_percentage}
          inputProps={{ min: 1, max: 100 }}
        />
      </Grid>
      <Grid item xs={4}>
        <TextField
          fullWidth
          size="small"
          type="number"
          name="error_rate_threshold"
          label="Error Rate Threshold (%)"
          value={promotionData.error_rate_threshold}
          onChange={handlePromotionChange}
          inputProps={{ min: 0, max: 100 }}
        />
      </Grid>
      <Grid item xs={4}>
        <TextField
          fullWidth
          size="small"
          type="number"
          name="latency_threshold"
          label="Latency Threshold (ms)"
          value={promotionData.latency_threshold}
          onChange={handlePromotionChange}
          inputProps={{ min: 0 }}
        />
      </Grid>
    </Grid>
  );

  const renderOnboardingForm = () => (
    <Grid container spacing={2}>
      <Grid item xs={12}>
        <TextField
          fullWidth
          required
          multiline
          rows={3}
          size="small"
          name="reason"
          label="Reason for Onboarding"
          value={onboardingData.reason}
          onChange={handleOnboardingChange}
          error={!!validationErrors.reason}
          helperText={validationErrors.reason || "Explain why this variant should be onboarded"}
          placeholder="Onboarding approved variant to production Qdrant cluster with monitoring setup"
        />
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small" required error={!!validationErrors.entity}>
          <InputLabel>Entity</InputLabel>
          <Select
            name="entity"
            value={onboardingData.entity}
            onChange={handleOnboardingChange}
            label="Entity"
          >
            {entities.map(entity => (
              <MenuItem key={entity.entity} value={entity.entity}>{entity.entity}</MenuItem>
            ))}
          </Select>
          {validationErrors.entity && <FormHelperText>{validationErrors.entity}</FormHelperText>}
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small" required error={!!validationErrors.model}>
          <InputLabel>Model</InputLabel>
          <Select
            name="model"
            value={onboardingData.model}
            onChange={handleOnboardingChange}
            label="Model"
          >
            {models.filter(m => m.entity === onboardingData.entity).map(model => (
              <MenuItem key={model.model} value={model.model}>{model.model}</MenuItem>
            ))}
          </Select>
          {validationErrors.model && <FormHelperText>{validationErrors.model}</FormHelperText>}
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small" required error={!!validationErrors.variant}>
          <InputLabel>Variant</InputLabel>
          <Select
            name="variant"
            value={onboardingData.variant}
            onChange={handleOnboardingChange}
            label="Variant"
          >
            {variants.filter(v => v.entity === onboardingData.entity && v.model === onboardingData.model).map(variant => (
              <MenuItem key={variant.variant} value={variant.variant}>{variant.variant}</MenuItem>
            ))}
          </Select>
          {validationErrors.variant && <FormHelperText>{validationErrors.variant}</FormHelperText>}
        </FormControl>
      </Grid>
      <Grid item xs={6}>
        <TextField
          fullWidth
          required
          size="small"
          name="cluster_endpoint"
          label="Cluster Endpoint"
          value={onboardingData.cluster_endpoint}
          onChange={handleOnboardingChange}
          error={!!validationErrors.cluster_endpoint}
          helperText={validationErrors.cluster_endpoint}
          placeholder="qdrant-prod.meesho.int:6333"
        />
      </Grid>
      <Grid item xs={6}>
        <TextField
          fullWidth
          required
          size="small"
          name="collection_name"
          label="Collection Name"
          value={onboardingData.collection_name}
          onChange={handleOnboardingChange}
          error={!!validationErrors.collection_name}
          helperText={validationErrors.collection_name}
          placeholder="product_embeddings_v1"
        />
      </Grid>
      <Grid item xs={4}>
        <TextField
          fullWidth
          size="small"
          type="number"
          name="vector_size"
          label="Vector Size"
          value={onboardingData.vector_size}
          onChange={handleOnboardingChange}
          error={!!validationErrors.vector_size}
          helperText={validationErrors.vector_size}
          inputProps={{ min: 1 }}
        />
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small">
          <InputLabel>Distance</InputLabel>
          <Select
            name="distance"
            value={onboardingData.distance}
            onChange={handleOnboardingChange}
            label="Distance"
          >
            {distanceTypes.map(type => (
              <MenuItem key={type} value={type}>{type}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
      <Grid item xs={4}>
        <FormControl fullWidth size="small">
          <InputLabel>Log Level</InputLabel>
          <Select
            name="log_level"
            value={onboardingData.log_level}
            onChange={handleOnboardingChange}
            label="Log Level"
          >
            {logLevels.map(level => (
              <MenuItem key={level} value={level}>{level}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
    </Grid>
  );

  return (
    <Paper elevation={0} sx={{ width: '100%', height: '100vh', padding: '2rem', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h6">Deployment Registry</Typography>
          <Typography variant="body1" color="text.secondary">
            Manage cluster creation, variant promotion, and onboarding requests
          </Typography>
        </Box>
      </Box>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={activeTab} onChange={handleTabChange}>
          <Tab 
            icon={<CloudIcon />} 
            iconPosition="start"
            label="Cluster Creation" 
          />
          <Tab 
            icon={<RocketLaunchIcon />} 
            iconPosition="start"
            label="Variant Promotion" 
          />
          <Tab 
            icon={<LaunchIcon />} 
            iconPosition="start"
            label="Variant Onboarding" 
          />
        </Tabs>
      </Box>

      {/* Tab Content */}
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
        {activeTab === 0 && (
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6">Create Qdrant Cluster</Typography>
              <Button
                variant="contained"
                startIcon={<CloudIcon />}
                onClick={() => handleOpen('cluster')}
                sx={{
                  backgroundColor: '#522b4a',
                  '&:hover': { backgroundColor: '#613a5c' },
                }}
              >
                Create Cluster
              </Button>
            </Box>
            <Typography variant="body2" color="text.secondary">
              Deploy new Qdrant clusters for vector database operations with configurable node setup and resource allocation.
            </Typography>
          </Box>
        )}

        {activeTab === 1 && (
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6">Promote Variant</Typography>
              <Button
                variant="contained"
                startIcon={<RocketLaunchIcon />}
                onClick={() => handleOpen('promotion')}
                sx={{
                  backgroundColor: '#522b4a',
                  '&:hover': { backgroundColor: '#613a5c' },
                }}
              >
                Promote Variant
              </Button>
            </Box>
            <Typography variant="body2" color="text.secondary">
              Promote experiment variants to production with traffic routing and monitoring configuration.
            </Typography>
          </Box>
        )}

        {activeTab === 2 && (
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6">Onboard Variant</Typography>
              <Button
                variant="contained"
                startIcon={<LaunchIcon />}
                onClick={() => handleOpen('onboarding')}
                sx={{
                  backgroundColor: '#522b4a',
                  '&:hover': { backgroundColor: '#613a5c' },
                }}
              >
                Onboard Variant
              </Button>
            </Box>
            <Typography variant="body2" color="text.secondary">
              Onboard approved variants to production infrastructure with collection setup and monitoring.
            </Typography>
          </Box>
        )}

        {/* Info Cards */}
        <Grid container spacing={2} sx={{ mt: 2 }}>
          <Grid item xs={4}>
            <Paper sx={{ p: 2, backgroundColor: '#f8f9fa' }}>
              <Typography variant="h6" sx={{ color: '#522b4a', mb: 1 }}>Active Clusters</Typography>
              <Chip label={clusters.length} sx={{ backgroundColor: '#E7F6E7', color: '#2E7D32', fontWeight: 600 }} />
            </Paper>
          </Grid>
          <Grid item xs={4}>
            <Paper sx={{ p: 2, backgroundColor: '#f8f9fa' }}>
              <Typography variant="h6" sx={{ color: '#522b4a', mb: 1 }}>Available Variants</Typography>
              <Chip label={variants.length} sx={{ backgroundColor: '#fff8e1', color: '#f57c00', fontWeight: 600 }} />
            </Paper>
          </Grid>
          <Grid item xs={4}>
            <Paper sx={{ p: 2, backgroundColor: '#f8f9fa' }}>
              <Typography variant="h6" sx={{ color: '#522b4a', mb: 1 }}>Registered Models</Typography>
              <Chip label={models.length} sx={{ backgroundColor: '#e3f2fd', color: '#1976d2', fontWeight: 600 }} />
            </Paper>
          </Grid>
        </Grid>
      </Box>

      {/* Operation Modal */}
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle sx={{ color: '#522b4a', fontWeight: 600 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {getDialogIcon()}
            {getDialogTitle()}
          </Box>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            {operationType === 'cluster' && renderClusterForm()}
            {operationType === 'promotion' && renderPromotionForm()}
            {operationType === 'onboarding' && renderOnboardingForm()}
          </Box>
        </DialogContent>
        <DialogActions sx={{ backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
          <Button
            onClick={handleClose}
            sx={{
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            sx={{
              backgroundColor: '#522b4a',
              '&:hover': { backgroundColor: '#613a5c' }
            }}
          >
            Submit Request
          </Button>
        </DialogActions>
      </Dialog>

      {/* Toast Notification */}
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleCloseNotification}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          onClose={handleCloseNotification}
          severity={notification.severity}
          sx={{ width: '100%' }}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </Paper>
  );
};

export default DeploymentRegistry;