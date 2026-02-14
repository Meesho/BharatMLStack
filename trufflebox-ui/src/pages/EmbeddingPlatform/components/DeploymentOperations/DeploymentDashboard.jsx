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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
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
import ScienceIcon from '@mui/icons-material/Science';
import { useAuth } from '../../../Auth/AuthContext';
import usePromoteWithProdCredentials from '../../../../common/PromoteWithProdCredentials';
import embeddingPlatformAPI from '../../../../services/embeddingPlatform/api';
import * as URL_CONSTANTS from '../../../../config';
import { useNotification } from '../shared/hooks/useNotification';

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

const normalizeTaskStatus = (status) => (status || '').toString().toUpperCase().replace(/\s+/g, '_').trim();

const getStatusIcon = (status) => {
  const s = normalizeTaskStatus(status);
  if (s === 'DEPLOYED' || s === 'SUCCESS' || s === 'COMPLETED' || s === 'TESTED_SUCCESSFULLY') return <CheckCircleIcon sx={{ color: 'success.main' }} />;
  if (s === 'FAILED' || s === 'STOPPED') return <ErrorIcon sx={{ color: 'error.main' }} />;
  return <PendingIcon sx={{ color: 'warning.main' }} />;
};

const getStatusColor = (status) => {
  const s = normalizeTaskStatus(status);
  if (s === 'DEPLOYED' || s === 'SUCCESS' || s === 'COMPLETED' || s === 'TESTED_SUCCESSFULLY') return 'success';
  if (s === 'FAILED' || s === 'STOPPED') return 'error';
  return 'warning';
};

const isCompleted = (status) => normalizeTaskStatus(status) === 'COMPLETED';
const isTestedSuccessfully = (status) => normalizeTaskStatus(status) === 'TESTED_SUCCESSFULLY';

/** Returns color for collection_status dot: green, red, yellow; gray for empty/unknown */
const getCollectionStatusDotColor = (value) => {
  const v = (value ?? '').toString().toLowerCase().trim();
  if (v === 'green') return '#4caf50';
  if (v === 'red') return '#f44336';
  if (v === 'yellow') return '#ffc107';
  return '#9e9e9e';
};

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
  const { notification, showNotification, closeNotification } = useNotification();
  const theme = useTheme();

  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testTask, setTestTask] = useState(null);
  const [testGeneratedRequestJson, setTestGeneratedRequestJson] = useState('');
  const [testStep, setTestStep] = useState('idle');
  const [testGenerateLoading, setTestGenerateLoading] = useState(false);
  const [testExecuteLoading, setTestExecuteLoading] = useState(false);
  const [testError, setTestError] = useState('');
  const [testSuccessMsg, setTestSuccessMsg] = useState('');
  const [testExecuteResponse, setTestExecuteResponse] = useState(null);

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
    const interval = setInterval(fetchTasks, 60000);
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
        const prodModelsResponse = await embeddingPlatformAPI.getModelsWithAuth(prodBaseUrl, token);
        const modelExistsOnProd = isModelPresentOnProd(prodModelsResponse, entity, model);

        if (!modelExistsOnProd) {
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

  const handleOpenTestModal = (task) => {
    setTestTask(task);
    setTestModalOpen(true);
    setTestStep('idle');
    setTestGeneratedRequestJson('');
    setTestError('');
    setTestSuccessMsg('');
    setTestExecuteResponse(null);
  };

  const handleCloseTestModal = () => {
    setTestModalOpen(false);
    setTestTask(null);
    setTestStep('idle');
    setTestGeneratedRequestJson('');
    setTestError('');
    setTestSuccessMsg('');
    setTestExecuteResponse(null);
  };

  const handleGenerateRequest = async () => {
    if (!testTask) return;
    setTestGenerateLoading(true);
    setTestError('');
    try {
      const res = await embeddingPlatformAPI.generateVariantTestRequest({
        entity: testTask.entity,
        model: testTask.model,
        variant: testTask.variant,
      });
      const requestObj = res?.request;
      if (!requestObj) throw new Error('No request in response');
      setTestGeneratedRequestJson(JSON.stringify(requestObj, null, 2));
      setTestStep('generated');
    } catch (err) {
      setTestError(err?.message || 'Failed to generate request');
    } finally {
      setTestGenerateLoading(false);
    }
  };

  const handleExecuteRequest = async () => {
    setTestExecuteLoading(true);
    setTestError('');
    setTestSuccessMsg('');
    setTestExecuteResponse(null);
    try {
      let payload;
      try {
        payload = JSON.parse(testGeneratedRequestJson);
      } catch (parseErr) {
        setTestError('Invalid JSON in request payload. Please fix the editable payload and try again.');
        return;
      }
      const res = await embeddingPlatformAPI.executeVariantTestRequest(payload);
      setTestExecuteResponse(res?.response ?? res ?? null);
      setTestSuccessMsg('Successfully tested / executed the request.');
      fetchTasks(true);
    } catch (err) {
      setTestError(err?.message || 'Failed to execute request');
    } finally {
      setTestExecuteLoading(false);
    }
  };

  return (
    <Box sx={{ maxWidth: 1400, mx: 'auto', px: { xs: 1, sm: 2 }, py: 2 }}>
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
                  {tasks.length} task{tasks.length !== 1 ? 's' : ''} • Auto-refresh every 1m
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
                <TableCell align="right" sx={{ minWidth: 140 }}>Actions</TableCell>
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
                          <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                            <Tooltip title={isCompleted(task.status) ? 'Test this variant' : 'Complete the task to test'}>
                              <span>
                                <Button
                                  size="small"
                                  variant="outlined"
                                  startIcon={<ScienceIcon />}
                                  onClick={() => handleOpenTestModal(task)}
                                  disabled={!isCompleted(task.status)}
                                  sx={{
                                    textTransform: 'none',
                                    fontWeight: 600,
                                    borderColor: '#1976d2',
                                    color: '#1976d2',
                                    '&:hover': { borderColor: '#1976d2', backgroundColor: 'rgba(25, 118, 210, 0.08)' },
                                  }}
                                >
                                  Test
                                </Button>
                              </span>
                            </Tooltip>
                            <Tooltip title={isTestedSuccessfully(task.status) ? 'Promote this variant' : 'Test the variant first to enable promote'}>
                              <span>
                                <Button
                                  size="small"
                                  variant="outlined"
                                  startIcon={<PromoteIcon />}
                                  onClick={() => handlePromote(task)}
                                  disabled={!isTestedSuccessfully(task.status) || promotingTaskId !== null}
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
                          </Box>
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
                                  const isCollectionStatus = key === 'collection_status';
                                  const dotColor = isCollectionStatus ? getCollectionStatusDotColor(value) : null;
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
                                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.25 }}>
                                        {dotColor != null && (
                                          <Box
                                            sx={{
                                              width: 10,
                                              height: 10,
                                              borderRadius: '50%',
                                              backgroundColor: dotColor,
                                              flexShrink: 0,
                                            }}
                                            aria-hidden
                                          />
                                        )}
                                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                                          {display}
                                        </Typography>
                                      </Box>
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

      {/* Test Variant Modal */}
      <Dialog open={testModalOpen} onClose={handleCloseTestModal} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <ScienceIcon sx={{ color: '#1976d2', fontSize: 28 }} />
            <Box>
              <Typography variant="h6" component="span" sx={{ fontWeight: 600 }}>
                Test Variant
              </Typography>
              {testTask && (
                <Typography display="block" variant="body2" color="text.secondary" sx={{ fontWeight: 500, mt: 0.25 }}>
                  {testTask.entity} → {testTask.model} → {testTask.variant}
                </Typography>
              )}
            </Box>
          </Box>
        </DialogTitle>
        <DialogContent sx={{ pt: 2.5, display: 'flex', flexDirection: 'column', gap: 2.5 }}>
          {testError && (
            <Alert severity="error" onClose={() => setTestError('')}>
              {testError}
            </Alert>
          )}
          {testSuccessMsg && (
            <Alert severity="success">{testSuccessMsg}</Alert>
          )}

          <Paper variant="outlined" sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderColor: 'rgba(25, 118, 210, 0.2)' }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1.5, color: 'text.primary' }}>
              1. Generate request
            </Typography>
            <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>
              Payload sent to generate-request (read-only)
            </Typography>
            <Box
              sx={{
                p: 1.5,
                borderRadius: 1,
                backgroundColor: 'grey.50',
                border: '1px solid',
                borderColor: 'divider',
                mb: 1.5,
              }}
            >
              <Typography
                component="pre"
                variant="body2"
                sx={{
                  fontFamily: 'monospace',
                  fontSize: '0.8125rem',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  m: 0,
                }}
              >
                {testTask
                  ? JSON.stringify(
                      { entity: testTask.entity, model: testTask.model, variant: testTask.variant },
                      null,
                      2
                    )
                  : '—'}
              </Typography>
            </Box>
            <Button
              variant="contained"
              size="small"
              onClick={handleGenerateRequest}
              disabled={testGenerateLoading}
              sx={{ backgroundColor: '#1976d2', '&:hover': { backgroundColor: '#1565c0' } }}
            >
              {testGenerateLoading ? 'Generating…' : 'Generate Request'}
            </Button>
          </Paper>

          {testStep === 'generated' && (
            <Paper variant="outlined" sx={{ p: 2, backgroundColor: 'rgba(46, 125, 50, 0.02)', borderColor: 'rgba(46, 125, 50, 0.2)' }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: 'text.primary' }}>
                2. Execute request
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>
                Edit payload if needed, then run execute-request
              </Typography>
              <TextField
                multiline
                minRows={6}
                maxRows={14}
                fullWidth
                value={testGeneratedRequestJson}
                onChange={(e) => setTestGeneratedRequestJson(e.target.value)}
                variant="outlined"
                size="small"
                InputProps={{ style: { fontFamily: 'monospace', fontSize: '0.8125rem' } }}
                sx={{ mb: 1.5, backgroundColor: '#fafafa', '& .MuiOutlinedInput-root': { alignItems: 'flex-start' } }}
              />
              <Button
                variant="contained"
                size="small"
                onClick={handleExecuteRequest}
                disabled={testExecuteLoading}
                sx={{ backgroundColor: '#2e7d32', '&:hover': { backgroundColor: '#1b5e20' } }}
              >
                {testExecuteLoading ? 'Executing…' : 'Execute Request'}
              </Button>
            </Paper>
          )}

          {testExecuteResponse != null && (
            <Paper variant="outlined" sx={{ p: 2, backgroundColor: 'rgba(102, 187, 106, 0.04)', borderColor: 'rgba(46, 125, 50, 0.3)' }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: 'text.primary' }}>
                Execute response
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>
                Response from execute-request API
              </Typography>
              <Box
                sx={{
                  p: 1.5,
                  borderRadius: 1,
                  backgroundColor: '#fafafa',
                  border: '1px solid',
                  borderColor: 'divider',
                  maxHeight: 280,
                  overflow: 'auto',
                }}
              >
                <Typography
                  component="pre"
                  variant="body2"
                  sx={{
                    fontFamily: 'monospace',
                    fontSize: '0.8125rem',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    m: 0,
                  }}
                >
                  {typeof testExecuteResponse === 'object'
                    ? JSON.stringify(testExecuteResponse, null, 2)
                    : String(testExecuteResponse)}
                </Typography>
              </Box>
            </Paper>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2, borderTop: '1px solid', borderColor: 'divider' }}>
          <Button onClick={handleCloseTestModal} sx={{ color: ACCENT, fontWeight: 600 }}>
            Close
          </Button>
        </DialogActions>
      </Dialog>

      <ProductionCredentialModalComponent
        title="Promote Variant"
        description="Enter your production credentials to promote this variant."
      />

      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={closeNotification}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert onClose={closeNotification} severity={notification.severity} sx={{ width: '100%' }}>
          {notification.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default DeploymentDashboard;
