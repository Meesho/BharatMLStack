import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  LinearProgress,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Collapse,
  Snackbar,
  Alert,
  useTheme,
  alpha,
  Tooltip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import DeploymentIcon from '@mui/icons-material/RocketLaunch';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import PendingIcon from '@mui/icons-material/Pending';
import AssignmentIcon from '@mui/icons-material/Assignment';
import PromoteIcon from '@mui/icons-material/KeyboardDoubleArrowUp';
import { useAuth } from '../../../Auth/AuthContext';
import usePromoteWithProdCredentials from '../../../../common/PromoteWithProdCredentials';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import * as URL_CONSTANTS from '../../../../config';

const ACCENT = '#522b4a';

const PAYLOAD_DETAIL_FIELDS = [
  { key: 'collection_status', label: 'Collection Status' },
  { key: 'indexed_vector_count', label: 'Indexed Vector Count' },
  { key: 'points_count', label: 'Points Count' },
  { key: 'segments_count', label: 'Segments Count' },
];

const parsePayload = (payload) => {
  if (payload == null || payload === '') return {};
  if (typeof payload === 'object') return payload;
  try {
    return typeof payload === 'string' ? JSON.parse(payload) : {};
  } catch {
    return {};
  }
};

const getStatusIcon = (status) => {
  const s = (status || '').toUpperCase();
  if (s === 'DEPLOYED' || s === 'SUCCESS' || s === 'COMPLETED') return <CheckCircleIcon sx={{ color: 'success.main' }} />;
  if (s === 'FAILED' || s === 'STOPPED') return <ErrorIcon sx={{ color: 'error.main' }} />;
  return <PendingIcon sx={{ color: 'warning.main' }} />;
};

const getStatusColor = (status) => {
  const s = (status || '').toUpperCase();
  if (s === 'DEPLOYED' || s === 'SUCCESS' || s === 'COMPLETED') return 'success';
  if (s === 'FAILED' || s === 'STOPPED') return 'error';
  return 'warning';
};

const isCompleted = (status) => (status || '').toUpperCase() === 'COMPLETED';

const isModelPresentOnProd = (modelsResponse, entity, model) => {
  const models = modelsResponse?.models ?? modelsResponse?.Models ?? {};
  const entityData = models[entity];
  const modelsObj = entityData?.Models ?? entityData?.models;
  if (!modelsObj || typeof modelsObj !== 'object') return false;
  return Object.prototype.hasOwnProperty.call(modelsObj, model);
};

const isVariantPresentOnProd = (variantsResponse, variant) => {
  const variants = variantsResponse?.variants ?? variantsResponse?.Variants ?? {};
  if (typeof variants !== 'object') return false;
  return Object.prototype.hasOwnProperty.call(variants, variant);
};

const buildModelRegisterPayloadFromDev = (devModelData, entity, model) => {
  const mt = devModelData?.model_type ?? devModelData?.ModelType ?? 'RESET';
  const mc = devModelData?.model_config ?? devModelData?.ModelConfig ?? {};
  const jf = devModelData?.job_frequency ?? devModelData?.JobFrequency ?? '';
  return {
    entity: String(entity),
    model: String(model),
    model_type: mt,
    model_config: mc,
    job_frequency: jf,
    embedding_store_enabled: devModelData?.embedding_store_enabled ?? devModelData?.EmbeddingStoreEnabled ?? false,
    embedding_store_ttl: devModelData?.embedding_store_ttl ?? devModelData?.EmbeddingStoreTTL ?? 0,
    mq_id: devModelData?.mq_id ?? devModelData?.MQID ?? 0,
    training_data_path: devModelData?.training_data_path ?? devModelData?.TrainingDataPath ?? '',
    number_of_partitions: devModelData?.number_of_partitions ?? devModelData?.NumberOfPartitions ?? 24,
    topic_name: devModelData?.topic_name ?? devModelData?.TopicName ?? '',
    metadata: devModelData?.metadata ?? devModelData?.Metadata ?? {},
    failure_producer_mq_id: devModelData?.failure_producer_mq_id ?? devModelData?.FailureProducerMqId ?? 0,
  };
};

const DeploymentDashboard = () => {
  const { user } = useAuth();
  const { initiatePromotion, ProductionCredentialModalComponent } = usePromoteWithProdCredentials();
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [expandedTaskId, setExpandedTaskId] = useState(null);
  const [promotingTaskId, setPromotingTaskId] = useState(null);
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success',
  });
  const theme = useTheme();

  const fetchTasks = async (showRefreshIndicator = false) => {
    try {
      if (showRefreshIndicator) setRefreshing(true);
      else setLoading(true);

      const response = await embeddingPlatformAPI.getVariantOnboardingTasks().catch(() => ({
        variant_onboarding_tasks: [],
        total_count: 0,
      }));

      setTasks(response.variant_onboarding_tasks || []);
    } catch (error) {
      console.error('Error fetching tasks:', error);
      showNotification('Error fetching tasks', 'error');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchTasks();
    const interval = setInterval(fetchTasks, 300000);
    return () => clearInterval(interval);
  }, []);

  const handleRefresh = () => fetchTasks(true);

  const toggleExpand = (taskId) => {
    setExpandedTaskId((prev) => (prev === taskId ? null : taskId));
  };

  const handlePromote = (task) => {
    if (!task.entity || !task.model || !task.variant) {
      showNotification('Task missing entity, model or variant', 'error');
      return;
    }
    const prodBaseUrl = (URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL || '').replace(/\/$/, '');
    if (!prodBaseUrl) {
      showNotification('Production Horizon URL is not configured', 'error');
      return;
    }

    initiatePromotion(task, async (item, prodCredentials) => {
      const { entity, model, variant } = item;
      const token = prodCredentials?.token;
      const requestor = prodCredentials?.email || '';

      setPromotingTaskId(item.task_id);
      try {
        // 1. Get prod models
        const prodModelsResponse = await embeddingPlatformAPI.getModelsWithAuth(prodBaseUrl, token);
        const modelExistsOnProd = isModelPresentOnProd(prodModelsResponse, entity, model);

        if (!modelExistsOnProd) {
          // Branch A: model not on prod → promote model first, then variant
          const devModelsResponse = await embeddingPlatformAPI.getModels().catch(() => ({}));
          const devModels = devModelsResponse?.models ?? devModelsResponse?.Models ?? {};
          const devEntityData = devModels[entity];
          const devModelsObj = devEntityData?.Models ?? devEntityData?.models;
          const devModelData = devModelsObj?.[model];

          if (!devModelData) {
            showNotification(
              `Model "${entity}/${model}" not found on dev. Cannot promote model to prod.`,
              'error'
            );
            return;
          }

          const modelPayload = buildModelRegisterPayloadFromDev(devModelData, entity, model);
          await embeddingPlatformAPI.registerModelWithAuth(prodBaseUrl, token, {
            requestor,
            reason: 'Promote model to prod (from variant promotion)',
            payload: modelPayload,
          });
        } else {
          // Branch B: model on prod → check if variant already exists
          const prodVariantsResponse = await embeddingPlatformAPI.getVariantsWithAuth(
            prodBaseUrl,
            token,
            entity,
            model
          );
          if (isVariantPresentOnProd(prodVariantsResponse, variant)) {
            showNotification('This variant already exists on prod.', 'info');
            return;
          }
        }

        const variantResult = await embeddingPlatformAPI.registerVariantWithAuth(prodBaseUrl, token, {
          requestor,
          reason: 'Promoted from completed onboarding task',
          payload: { entity, model, variant },
          request_type: 'PROMOTE',
        });
        showNotification(
          variantResult?.message || 'Variant promotion request submitted successfully.',
          'success'
        );
        fetchTasks(true);
      } catch (error) {
        console.error('Error during promotion:', error);
        showNotification(error.message || 'Failed to complete promotion', 'error');
      } finally {
        setPromotingTaskId(null);
      }
    });
  };

  const showNotification = (message, severity) => {
    setNotification({ open: true, message, severity });
  };

  const handleCloseNotification = () => {
    setNotification((prev) => ({ ...prev, open: false }));
  };

  return (
    <Box sx={{ maxWidth: 1400, mx: 'auto', px: { xs: 1, sm: 2 }, py: 2 }}>
      {/* Header card */}
      <Card
        elevation={0}
        sx={{
          mb: 3,
          borderRadius: 2,
          border: '1px solid',
          borderColor: 'divider',
          backgroundColor: alpha(ACCENT, 0.02),
        }}
      >
        <CardContent sx={{ py: 2.5, '&:last-child': { pb: 2.5 } }}>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'space-between', alignItems: 'center', gap: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box
                sx={{
                  width: 48,
                  height: 48,
                  borderRadius: 2,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  backgroundColor: alpha(ACCENT, 0.1),
                }}
              >
                <DeploymentIcon sx={{ fontSize: 28, color: ACCENT }} />
              </Box>
              <Box>
                <Typography variant="h5" fontWeight={600} color="text.primary">
                  Variant Onboarding Tasks
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {tasks.length} task{tasks.length !== 1 ? 's' : ''} • Auto-refresh every 5m
                </Typography>
              </Box>
            </Box>
            <Button
              variant="contained"
              startIcon={<RefreshIcon />}
              onClick={handleRefresh}
              disabled={refreshing}
              sx={{
                backgroundColor: ACCENT,
                textTransform: 'none',
                fontWeight: 600,
                px: 2,
                '&:hover': { backgroundColor: alpha(ACCENT, 0.85) },
              }}
            >
              {refreshing ? 'Refreshing…' : 'Refresh'}
            </Button>
          </Box>
        </CardContent>
      </Card>

      {/* Tasks table card */}
      <Card
        elevation={0}
        sx={{
          borderRadius: 2,
          border: '1px solid',
          borderColor: 'divider',
          overflow: 'hidden',
        }}
      >
        <TableContainer>
          <Table size="medium" aria-label="Variant onboarding tasks" stickyHeader>
            <TableHead>
              <TableRow
                sx={{
                  backgroundColor: alpha(theme.palette.primary.main, 0.04),
                  '& th': {
                    fontWeight: 600,
                    fontSize: '0.8125rem',
                    color: 'text.secondary',
                    borderBottom: '1px solid',
                    borderColor: 'divider',
                    py: 1.5,
                  },
                }}
              >
                <TableCell padding="checkbox" sx={{ width: 48 }} />
                <TableCell>Task ID</TableCell>
                <TableCell>Entity</TableCell>
                <TableCell>Model</TableCell>
                <TableCell>Variant</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Created At</TableCell>
                <TableCell align="right" sx={{ minWidth: 100 }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading && (
                <TableRow>
                  <TableCell colSpan={8} align="center" sx={{ py: 6 }}>
                    <LinearProgress sx={{ maxWidth: 280, mx: 'auto', borderRadius: 1 }} />
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                      Loading tasks…
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading && tasks.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} align="center" sx={{ py: 6 }}>
                    <AssignmentIcon sx={{ fontSize: 48, color: 'action.disabled', mb: 1 }} />
                    <Typography variant="body1" color="text.secondary" display="block">
                      No tasks found
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      Tasks will appear here when variant onboarding runs.
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                tasks.map((task, index) => {
                  const taskId = task.task_id;
                  const isExpanded = expandedTaskId === taskId;
                  const payloadObj = parsePayload(task.payload);

                  return (
                    <React.Fragment key={taskId}>
                      <TableRow
                        hover
                        sx={{
                          backgroundColor: index % 2 === 1 ? alpha(theme.palette.grey[500], 0.02) : undefined,
                          '& > *': { borderBottom: '1px solid', borderColor: 'divider' },
                          '&:hover': { backgroundColor: alpha(ACCENT, 0.03) },
                        }}
                      >
                        <TableCell padding="checkbox" sx={{ width: 48 }}>
                          <IconButton
                            size="small"
                            onClick={() => toggleExpand(taskId)}
                            aria-label={isExpanded ? 'Collapse row' : 'Expand row'}
                            sx={{
                              color: isExpanded ? ACCENT : 'text.secondary',
                              '&:hover': { backgroundColor: alpha(ACCENT, 0.08) },
                            }}
                          >
                            {isExpanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                          </IconButton>
                        </TableCell>
                        <TableCell sx={{ fontWeight: 500 }}>{task.task_id}</TableCell>
                        <TableCell>{task.entity ?? '—'}</TableCell>
                        <TableCell>{task.model ?? '—'}</TableCell>
                        <TableCell>{task.variant ?? '—'}</TableCell>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            {getStatusIcon(task.status)}
                            <Chip
                              label={task.status || '—'}
                              color={getStatusColor(task.status)}
                              size="small"
                              sx={{ fontWeight: 500, textTransform: 'uppercase' }}
                            />
                          </Box>
                        </TableCell>
                        <TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
                          {task.created_at ?? '—'}
                        </TableCell>
                        <TableCell align="right">
                          <Tooltip title={isCompleted(task.status) ? 'Promote this variant' : 'Complete the task to promote'}>
                            <span>
                              <Button
                                size="small"
                                variant="outlined"
                                startIcon={<PromoteIcon />}
                                onClick={() => handlePromote(task)}
                                disabled={!isCompleted(task.status) || promotingTaskId !== null}
                                sx={{
                                  textTransform: 'none',
                                  fontWeight: 600,
                                  borderColor: ACCENT,
                                  color: ACCENT,
                                  '&:hover': { borderColor: ACCENT, backgroundColor: alpha(ACCENT, 0.08) },
                                }}
                              >
                                {promotingTaskId === taskId ? 'Promoting…' : 'Promote'}
                              </Button>
                            </span>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell colSpan={8} sx={{ py: 0, borderBottom: isExpanded ? '1px solid' : 0, borderColor: 'divider' }}>
                          <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                            <Box
                              sx={{
                                py: 2,
                                px: 3,
                                backgroundColor: alpha(theme.palette.grey[500], 0.04),
                                borderTop: '1px solid',
                                borderColor: 'divider',
                              }}
                            >
                              <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1.5, fontWeight: 600 }}>
                                Payload details
                              </Typography>
                              <Box
                                sx={{
                                  display: 'grid',
                                  gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' },
                                  gap: 2,
                                }}
                              >
                                {PAYLOAD_DETAIL_FIELDS.map(({ key, label }) => {
                                  const value = payloadObj[key];
                                  const display = value !== undefined && value !== null ? String(value) : '—';
                                  return (
                                    <Box
                                      key={key}
                                      sx={{
                                        p: 1.5,
                                        borderRadius: 1,
                                        backgroundColor: 'background.paper',
                                        border: '1px solid',
                                        borderColor: 'divider',
                                      }}
                                    >
                                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 500 }}>
                                        {label}
                                      </Typography>
                                      <Typography variant="body2" sx={{ mt: 0.25, fontFamily: 'monospace' }}>
                                        {display}
                                      </Typography>
                                    </Box>
                                  );
                                })}
                              </Box>
                              {PAYLOAD_DETAIL_FIELDS.every(({ key }) => payloadObj[key] === undefined) ? (
                                <Typography variant="body2" color="text.secondary" sx={{ mt: 1.5 }}>
                                  No payload detail fields available for this task.
                                </Typography>
                              ) : null}
                            </Box>
                          </Collapse>
                        </TableCell>
                      </TableRow>
                    </React.Fragment>
                  );
                })}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      <ProductionCredentialModalComponent
        title="Promote Variant"
        description="Enter your production credentials to promote this variant."
      />

      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleCloseNotification}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseNotification} severity={notification.severity} sx={{ width: '100%' }}>
          {notification.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default DeploymentDashboard;
