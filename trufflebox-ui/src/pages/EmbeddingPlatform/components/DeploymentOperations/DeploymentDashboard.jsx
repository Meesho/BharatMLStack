import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  LinearProgress,
  Chip,
  Alert,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Snackbar,
  IconButton,
  Tooltip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import SearchIcon from '@mui/icons-material/Search';
import DeploymentIcon from '@mui/icons-material/RocketLaunch';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import PendingIcon from '@mui/icons-material/Pending';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import StopIcon from '@mui/icons-material/Stop';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import GenericTable from '../../../OnlineFeatureStore/common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import { 
  transformToTableData,
} from '../../../../services/embeddingPlatform/utils';

const DEPLOYMENT_STATUS = {
  PENDING: 'pending',
  DEPLOYING: 'deploying',
  DEPLOYED: 'deployed',
  FAILED: 'failed',
  STOPPED: 'stopped',
  RESTARTING: 'restarting'
};

const DeploymentDashboard = () => {
  const [deployments, setDeployments] = useState([]);
  const [searchDeploymentId, setSearchDeploymentId] = useState('');
  const [selectedDeployment, setSelectedDeployment] = useState(null);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [filterStatus, setFilterStatus] = useState('');
  const [filterType, setFilterType] = useState('');
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });
  const { user } = useAuth();

  useEffect(() => {
    fetchDeployments();
    // Set up polling for real-time updates every 30 seconds
    const interval = setInterval(fetchDeployments, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchDeployments = async (showRefreshIndicator = false) => {
    try {
      if (showRefreshIndicator) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }

      // Fetch available deployment data (currently only Qdrant clusters)
      const [
        qdrantClusters,
      ] = await Promise.all([
        embeddingPlatformAPI.getQdrantClusters().catch(() => ({ data: { clusters: [] } })),
      ]);

      // Transform Qdrant clusters into deployment format
      const combinedDeployments = [
        ...(qdrantClusters.data?.clusters || []).map(cluster => ({
          ...cluster,
          type: 'Qdrant Cluster',
          deployment_id: cluster.cluster_id,
          status: cluster.status || DEPLOYMENT_STATUS.DEPLOYED,
          created_by: cluster.created_by || 'System',
          created_at: cluster.created_at || new Date().toISOString(),
          updated_at: cluster.updated_at || new Date().toISOString(),
          environment: cluster.environment || 'production'
        })),
      ];

      const transformedDeployments = transformToTableData(combinedDeployments, 'deployments')
        .map(dep => ({
          ...dep,
          Type: dep.type,
          EntityLabel: dep.Type,
          FeatureGroupLabel: dep.environment || '-',
          Actions: 'manage'
        }))
        .sort((a, b) => new Date(b.CreatedAt) - new Date(a.CreatedAt));

      setDeployments(transformedDeployments);
    } catch (error) {
      console.error('Error fetching deployments:', error);
      showNotification('Error fetching deployments', 'error');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = () => {
    fetchDeployments(true);
  };

  const handleViewDeployment = async (deployment) => {
    setSelectedDeployment(deployment);
    setShowDetailModal(true);
  };

  const closeDetailModal = () => {
    setShowDetailModal(false);
    setSelectedDeployment(null);
  };

  const handleSearch = () => {
    if (searchDeploymentId.trim()) {
      const foundDeployment = deployments.find(dep => 
        dep.deployment_id?.toString() === searchDeploymentId.trim() ||
        dep.cluster_id?.toString() === searchDeploymentId.trim() ||
        dep.RequestId?.toString() === searchDeploymentId.trim()
      );
      if (foundDeployment) {
        handleViewDeployment(foundDeployment);
      } else {
        showNotification('Deployment ID not found', 'warning');
      }
    }
  };

  const handleDeploymentAction = async (deploymentId, action) => {
    try {
      // Placeholder implementation - these methods would need to be added to the API
      switch (action) {
        case 'start':
          showNotification('Start deployment functionality not yet implemented', 'info');
          break;
        case 'stop':
          showNotification('Stop deployment functionality not yet implemented', 'info');
          break;
        case 'restart':
          showNotification('Restart deployment functionality not yet implemented', 'info');
          break;
        default:
          return;
      }
      
      // Refresh deployments after action (when implemented)
      // fetchDeployments();
    } catch (error) {
      console.error(`Error ${action}ing deployment:`, error);
      showNotification(`Error ${action}ing deployment`, 'error');
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case DEPLOYMENT_STATUS.DEPLOYED:
        return <CheckCircleIcon sx={{ color: 'success.main' }} />;
      case DEPLOYMENT_STATUS.FAILED:
      case DEPLOYMENT_STATUS.STOPPED:
        return <ErrorIcon sx={{ color: 'error.main' }} />;
      case DEPLOYMENT_STATUS.DEPLOYING:
      case DEPLOYMENT_STATUS.RESTARTING:
        return <PendingIcon sx={{ color: 'warning.main' }} />;
      default:
        return <PendingIcon sx={{ color: 'warning.main' }} />;
    }
  };

  const getProgressValue = (status) => {
    switch (status) {
      case DEPLOYMENT_STATUS.PENDING:
        return 25;
      case DEPLOYMENT_STATUS.DEPLOYING:
        return 50;
      case DEPLOYMENT_STATUS.DEPLOYED:
        return 100;
      case DEPLOYMENT_STATUS.RESTARTING:
        return 75;
      case DEPLOYMENT_STATUS.FAILED:
      case DEPLOYMENT_STATUS.STOPPED:
        return 100;
      default:
        return 0;
    }
  };

  const getProgressColor = (status) => {
    switch (status) {
      case DEPLOYMENT_STATUS.DEPLOYED:
        return 'success';
      case DEPLOYMENT_STATUS.FAILED:
      case DEPLOYMENT_STATUS.STOPPED:
        return 'error';
      case DEPLOYMENT_STATUS.DEPLOYING:
      case DEPLOYMENT_STATUS.RESTARTING:
        return 'info';
      default:
        return 'primary';
    }
  };

  const getDeploymentSummary = () => {
    const total = deployments.length;
    const deployed = deployments.filter(dep => dep.Status === DEPLOYMENT_STATUS.DEPLOYED).length;
    const deploying = deployments.filter(dep => 
      dep.Status === DEPLOYMENT_STATUS.DEPLOYING || dep.Status === DEPLOYMENT_STATUS.RESTARTING
    ).length;
    const failed = deployments.filter(dep => 
      dep.Status === DEPLOYMENT_STATUS.FAILED || dep.Status === DEPLOYMENT_STATUS.STOPPED
    ).length;
    const pending = deployments.filter(dep => dep.Status === DEPLOYMENT_STATUS.PENDING).length;

    return { total, deployed, deploying, failed, pending };
  };

  const filteredDeployments = deployments.filter(dep => {
    if (filterStatus && dep.Status !== filterStatus) return false;
    if (filterType && dep.Type !== filterType) return false;
    return true;
  });

  const summary = getDeploymentSummary();
  const deploymentTypes = [...new Set(deployments.map(dep => dep.Type))];

  const showNotification = (message, severity) => {
    setNotification({
      open: true,
      message,
      severity
    });
  };

  const handleCloseNotification = () => {
    setNotification(prev => ({
      ...prev,
      open: false
    }));
  };

  const renderActionButtons = (deployment) => {
    const canStart = deployment.Status === DEPLOYMENT_STATUS.STOPPED || deployment.Status === DEPLOYMENT_STATUS.FAILED;
    const canStop = deployment.Status === DEPLOYMENT_STATUS.DEPLOYED || deployment.Status === DEPLOYMENT_STATUS.DEPLOYING;
    const canRestart = deployment.Status === DEPLOYMENT_STATUS.DEPLOYED;

    return (
      <Box sx={{ display: 'flex', gap: 1 }}>
        {canStart && (
          <Tooltip title="Start Deployment">
            <IconButton 
              size="small" 
              color="success"
              onClick={() => handleDeploymentAction(deployment.deployment_id || deployment.cluster_id, 'start')}
            >
              <PlayArrowIcon />
            </IconButton>
          </Tooltip>
        )}
        {canStop && (
          <Tooltip title="Stop Deployment">
            <IconButton 
              size="small" 
              color="error"
              onClick={() => handleDeploymentAction(deployment.deployment_id || deployment.cluster_id, 'stop')}
            >
              <StopIcon />
            </IconButton>
          </Tooltip>
        )}
        {canRestart && (
          <Tooltip title="Restart Deployment">
            <IconButton 
              size="small" 
              color="primary"
              onClick={() => handleDeploymentAction(deployment.deployment_id || deployment.cluster_id, 'restart')}
            >
              <RestartAltIcon />
            </IconButton>
          </Tooltip>
        )}
      </Box>
    );
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <DeploymentIcon sx={{ mr: 2, fontSize: 32, color: '#522b4a' }} />
          <Typography variant="h4">Deployment Operations Dashboard</Typography>
        </Box>
        <Button
          startIcon={refreshing ? <LinearProgress size={20} /> : <RefreshIcon />}
          onClick={handleRefresh}
          disabled={refreshing}
          variant="outlined"
          sx={{ borderColor: '#522b4a', color: '#522b4a' }}
        >
          {refreshing ? 'Refreshing...' : 'Refresh'}
        </Button>
      </Box>

      {/* Search Bar */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <TextField
              label="Search by Deployment ID"
              value={searchDeploymentId}
              onChange={(e) => setSearchDeploymentId(e.target.value)}
              variant="outlined"
              size="small"
              sx={{ minWidth: 200 }}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
            <Button
              startIcon={<SearchIcon />}
              onClick={handleSearch}
              variant="contained"
              sx={{ backgroundColor: '#522b4a', '&:hover': { backgroundColor: '#613a5c' } }}
            >
              Search
            </Button>
          </Box>
        </CardContent>
      </Card>

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography variant="h4" color="primary">
                {summary.total}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Total Deployments
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography variant="h4" color="success.main">
                {summary.deployed}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Active Deployments
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography variant="h4" color="warning.main">
                {summary.deploying + summary.pending}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                In Progress
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography variant="h4" color="error.main">
                {summary.failed}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Failed/Stopped
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Filters */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel>Filter by Status</InputLabel>
              <Select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                label="Filter by Status"
              >
                <MenuItem value="">All Status</MenuItem>
                <MenuItem value={DEPLOYMENT_STATUS.PENDING}>Pending</MenuItem>
                <MenuItem value={DEPLOYMENT_STATUS.DEPLOYING}>Deploying</MenuItem>
                <MenuItem value={DEPLOYMENT_STATUS.DEPLOYED}>Deployed</MenuItem>
                <MenuItem value={DEPLOYMENT_STATUS.RESTARTING}>Restarting</MenuItem>
                <MenuItem value={DEPLOYMENT_STATUS.FAILED}>Failed</MenuItem>
                <MenuItem value={DEPLOYMENT_STATUS.STOPPED}>Stopped</MenuItem>
              </Select>
            </FormControl>

            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel>Filter by Type</InputLabel>
              <Select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value)}
                label="Filter by Type"
              >
                <MenuItem value="">All Types</MenuItem>
                {deploymentTypes.map((type) => (
                  <MenuItem key={type} value={type}>
                    {type}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Box>
        </CardContent>
      </Card>

      {/* Deployments Table */}
      <GenericTable
        data={filteredDeployments.map(dep => ({
          ...dep,
          Actions: renderActionButtons(dep)
        }))}
        excludeColumns={[]}
        onRowAction={handleViewDeployment}
        loading={loading}
        flowType="default"
      />

        {/* Deployment Detail Modal */}
        <Dialog open={showDetailModal} onClose={closeDetailModal} maxWidth="md" fullWidth>
          <DialogTitle>Deployment Details</DialogTitle>
          <DialogContent>
            {selectedDeployment && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Deployment ID
                  </Typography>
                  <Typography variant="body1">
                    {selectedDeployment.deployment_id || selectedDeployment.cluster_id || selectedDeployment.RequestId || 'N/A'}
                  </Typography>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Type
                  </Typography>
                  <Typography variant="body1">
                    {selectedDeployment.Type || 'N/A'}
                  </Typography>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Status
                  </Typography>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 1 }}>
                    {getStatusIcon(selectedDeployment.Status)}
                    <Chip 
                      label={selectedDeployment.Status}
                      color={getProgressColor(selectedDeployment.Status)}
                      size="small"
                    />
                  </Box>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Progress
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={getProgressValue(selectedDeployment.Status)}
                    color={getProgressColor(selectedDeployment.Status)}
                    sx={{ mt: 1, height: 8, borderRadius: 4 }}
                  />
                  <Typography variant="caption" color="text.secondary">
                    {getProgressValue(selectedDeployment.Status)}% Complete
                  </Typography>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Environment
                  </Typography>
                  <Typography variant="body1">
                    {selectedDeployment.FeatureGroupLabel || 'N/A'}
                  </Typography>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Created By
                  </Typography>
                  <Typography variant="body1">
                    {selectedDeployment.CreatedBy || 'N/A'}
                  </Typography>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Created At
                  </Typography>
                  <Typography variant="body1">
                    {selectedDeployment.CreatedAt || 'N/A'}
                  </Typography>
                </Box>

                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Last Updated
                  </Typography>
                  <Typography variant="body1">
                    {selectedDeployment.UpdatedAt || 'N/A'}
                  </Typography>
                </Box>

                {/* Deployment Actions */}
                <Box sx={{ pt: 2, borderTop: '1px solid', borderColor: 'divider' }}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2 }}>
                    Available Actions
                  </Typography>
                  {renderActionButtons(selectedDeployment)}
                </Box>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button 
              onClick={closeDetailModal}
              sx={{ 
                color: '#522b4a',
                '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
              }}
            >
              Close
            </Button>
          </DialogActions>
        </Dialog>

      {/* Toast Notifications */}
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
    </Box>
  );
};

export default DeploymentDashboard;