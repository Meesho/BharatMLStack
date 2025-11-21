import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  CircularProgress,
  Snackbar,
  Alert,
  Typography
} from '@mui/material';
import { useSearchParams } from 'react-router-dom';
import GenericNumerixTable from '../shared/GenericNumerixTable';
import { useAuth } from '../../Auth/AuthContext';
import { SERVICES, SCREEN_TYPES, ACTIONS } from '../../../constants/permissions';
import useFormatDate from '../../../hooks/useFormatDate';
import InfixExpressionEditor from '../../../components/InfixExpressionEditor';
import ProductionCredentialModal from '../../../common/ProductionCredentialModal';
import { TestConfigModal } from '../shared/components/TestConfigModal';
import axios from 'axios';

import * as URL_CONSTANTS from '../../../config';

const NumerixConfigDiscoveryRegistry = () => {
  const { user, hasPermission, permissions } = useAuth();
  const { formatDateToIST } = useFormatDate();
  const [searchParams, setSearchParams] = useSearchParams();
  
  const service = SERVICES.NUMERIX;
  const screenType = SCREEN_TYPES.NUMERIX.CONFIG;
  const isPermissionsLoaded = permissions !== null;
  const [loading, setLoading] = useState(true);
  const [configData, setConfigData] = useState([]);
  const [openOnboardModal, setOpenOnboardModal] = useState(false);
  const [openEditModal, setOpenEditModal] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [showProdCredentialModal, setShowProdCredentialModal] = useState(false);
  const [selectedRowForPromotion, setSelectedRowForPromotion] = useState(null);
  const [selectedRowForEdit, setSelectedRowForEdit] = useState(null);
  const [openTestModal, setOpenTestModal] = useState(false);
  const [selectedRowForTest, setSelectedRowForTest] = useState(null);

  // Initialize state from URL params
  const [page, setPage] = useState(() => {
    const pageParam = searchParams.get('page');
    return pageParam ? parseInt(pageParam, 10) : 1;
  });
  const [limit] = useState(25);
  const [searchQuery, setSearchQuery] = useState(() => {
    return searchParams.get('config_id') || '';
  });
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState(() => {
    return searchParams.get('config_id') || '';
  });
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 25,
    total_count: 0,
    total_pages: 0
  });
  
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success'
  });

  const fetchConfigs = useCallback(async (currentPage, configId) => {
    try {
      setLoading(true);
      const params = new URLSearchParams({
        page: currentPage.toString(),
        limit: limit.toString()
      });
      
      // Add config_id filter if search query exists
      if (configId && configId.trim() !== '') {
        params.append('config_id', configId.trim());
      }

      const response = await axios.get(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-discovery/configs?${params.toString()}`,
        {
          headers: {
            'Authorization': `Bearer ${user.token}`
          }
        }
      );
      
      if (response.data && response.data.error === "") {
        // Transform data to match the table format
        const transformedData = response.data.data.map(item => ({
          ComputeId: item.config_id,
          InfixExpression: item.infix_expression,
          PostfixExpression: item.postfix_expression,
          Created_by: item.created_by,
          Created_at: formatDateToIST(item.created_at),
          Updated_by: item.updated_by,
          Updated_at: formatDateToIST(item.updated_at),
          dashboard_link: item.monitoring_url,
          test_results: {
            is_functionally_tested: item.test_results?.is_functionally_tested || false
          },
          deployable_running_status: item.deployable_running_status === "true"
        }));
        setConfigData(transformedData);

        // Update pagination metadata
        if (response.data.pagination) {
          setPagination(response.data.pagination);
        }
      } else {
        showNotification(response.data.error || 'Config Data Loading Error', 'error');
      }
    } catch (error) {
      showNotification('Config Data Loading Error', 'error');
      console.log('Error fetching configs:', error);
    } finally {
      setLoading(false);
    }
  }, [user.token, formatDateToIST, limit]);

  // Track if we're syncing from URL to prevent update loops
  const isSyncingFromUrl = React.useRef(false);

  // Sync state from URL params (for browser back/forward)
  useEffect(() => {
    const pageParam = searchParams.get('page');
    const configIdParam = searchParams.get('config_id');

    const urlPage = pageParam ? parseInt(pageParam, 10) : 1;
    const urlConfigId = configIdParam || '';

    // Only update if URL params differ from current state
    if (urlPage !== page || urlConfigId !== searchQuery) {
      isSyncingFromUrl.current = true;

      if (urlPage !== page) {
        setPage(urlPage);
      }

      if (urlConfigId !== searchQuery) {
        setSearchQuery(urlConfigId);
      }

      // Reset sync flag after state updates
      setTimeout(() => {
        isSyncingFromUrl.current = false;
      }, 0);
    }
  }, [searchParams]);

  // Update URL when page or search changes (but not on initial mount or during URL sync)
  const isInitialMount = React.useRef(true);

  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }

    // Skip if syncing from URL to prevent loops
    if (isSyncingFromUrl.current) {
      return;
    }

    const params = new URLSearchParams();
    if (page > 1) {
      params.set('page', page.toString());
    }

    if (searchQuery && searchQuery.trim() !== '') {
      params.set('config_id', searchQuery.trim());
    }

    // Check if URL already matches what we want to set
    const currentPage = searchParams.get('page');
    const currentConfigId = searchParams.get('config_id') || '';
    const expectedPage = page > 1 ? page.toString() : null;
    const expectedConfigId = searchQuery.trim() || null;

    // Only update if different
    if (currentPage !== expectedPage || currentConfigId !== expectedConfigId) {
      // Use push (not replace) to create proper browser history entries
      setSearchParams(params, { replace: false });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, searchQuery, setSearchParams]); // Note: searchParams NOT in dependencies

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  useEffect(() => {
    if (isPermissionsLoaded && hasPermission(service, screenType, ACTIONS.VIEW)) {
      fetchConfigs(page, debouncedSearchQuery);
    } else if (isPermissionsLoaded) {
      setLoading(false);
    }
  }, [isPermissionsLoaded, hasPermission, service, screenType, page, debouncedSearchQuery, fetchConfigs]);

  const handlePageChange = (event, newPage) => {
    setPage(newPage);
  };

  const handleSearchChange = (value) => {
    setSearchQuery(value);
    setPage(1); // Reset to first page when searching
  };

  const handleOnboardClick = () => {
    if (!hasPermission(service, screenType, ACTIONS.ONBOARD)) {
      showNotification('You do not have permission to onboard configurations', 'error');
      return;
    }
    
    setOpenOnboardModal(true);
  };

  const handleCloseOnboardModal = () => {
    setOpenOnboardModal(false);
  };

  const handleOnboardSubmit = async (infixExpression, postfixExpression) => {
    try {
      setSubmitting(true);
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-registry/onboard`,
        {
          payload: {
            config_value: {
              infix_expression: infixExpression,
              postfix_expression: postfixExpression
            }
          },
          created_by: user.email
        },
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data && response.data.error === "") {
        showNotification(response.data.data.message || 'Config Successfully Onboarded', 'success');
        handleCloseOnboardModal();
        fetchConfigs(page, debouncedSearchQuery); // Refresh data
      } else {
        showNotification(response.data.error || 'Config Onboarding Error', 'error');
      }
    } catch (error) {
      showNotification(
        error.response?.data?.error || 'Config Onboarding Error', 
        'error'
      ); 
      console.log('Error onboarding config:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const handlePromote = async (row) => {
    if (!hasPermission(service, screenType, ACTIONS.PROMOTE)) {
      showNotification('You do not have permission to promote configurations', 'error');
      return;
    }
    
    setSelectedRowForPromotion(row);
    setShowProdCredentialModal(true);
  };

  const handleProductionCredentialSuccess = async (prodCredentials) => {
    setShowProdCredentialModal(false);
    
    if (!selectedRowForPromotion) {
      showNotification('No configuration selected for promotion', 'error');
      return;
    }

    try {
      // Use the production token to promote the configuration
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/horizon/numerix-config-registry/promote`,
        {
          payload: {
            config_id: selectedRowForPromotion.ComputeId,
            config_value: {
              infix_expression: selectedRowForPromotion.InfixExpression,
              postfix_expression: selectedRowForPromotion.PostfixExpression
            }
          },
          updated_by: prodCredentials.email
        },
        {
          headers: {
            'Authorization': `Bearer ${prodCredentials.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data && response.data.error === "") {
        showNotification(response.data.data.message || 'Config Successfully Promoted', 'success');
        fetchConfigs(page, debouncedSearchQuery); // Refresh data
      } else {
        showNotification(response.data.error || 'Config Promotion Error', 'error');
      }
    } catch (error) {
      showNotification(
        error.response?.data?.error || 'Config Promotion Error', 
        'error'
      );
      console.log('Error promoting config:', error);
    } finally {
      setSelectedRowForPromotion(null);
    }
  };

  const handleProductionCredentialClose = () => {
    setShowProdCredentialModal(false);
    setSelectedRowForPromotion(null);
  };

  // Edit handlers
  const handleEditClick = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.EDIT)) {
      showNotification('You do not have permission to edit configurations', 'error');
      return;
    }
    
    setSelectedRowForEdit(row);
    setOpenEditModal(true);
  };

  const handleCloseEditModal = () => {
    setOpenEditModal(false);
    setSelectedRowForEdit(null);
  };

  const handleEditSubmit = async (infixExpression, postfixExpression) => {
    if (!selectedRowForEdit) {
      showNotification('No configuration selected for editing', 'error');
      return;
    }

    try {
      setSubmitting(true);
      const response = await axios.post(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/numerix-config-registry/edit`,
        {
          payload: {
            config_id: selectedRowForEdit.ComputeId,
            config_value: {
              infix_expression: infixExpression,
              postfix_expression: postfixExpression
            }
          },
          created_by: user.email
        },
        {
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data && response.data.error === "") {
        showNotification(response.data.data.message || 'Config Successfully Updated', 'success');
        handleCloseEditModal();
        fetchConfigs(page, debouncedSearchQuery); // Refresh data
      } else {
        showNotification(response.data.error || 'Config Update Error', 'error');
      }
    } catch (error) {
      showNotification(
        error.response?.data?.error || 'Config Update Error', 
        'error'
      ); 
      console.log('Error updating config:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleTestClick = (row) => {
    if (!hasPermission(service, screenType, ACTIONS.TEST)) {
      showNotification('You do not have permission to test configurations', 'error');
      return;
    }
    setSelectedRowForTest(row);
    setOpenTestModal(true);
  };

  const handleCloseTestModal = () => {
    setOpenTestModal(false);
    setSelectedRowForTest(null);
  };

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
          <Typography>You do not have permission to view Numerix configurations.</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ padding: '20px' }}>
      <GenericNumerixTable
        data={configData}
        excludeColumns={[]}
        onRowAction={handlePromote}
        onEditAction={handleEditClick}
        onTestAction={handleTestClick}
        loading={loading}
        pageName="numerix_registry"
        searchPlaceholder="Search by compute ID"
        buttonName="Onboard Compute Config"
        searchQuery={searchQuery}
        onSearchChange={handleSearchChange}
        pagination={{
          page: pagination.page,
          totalPages: pagination.total_pages,
          totalCount: pagination.total_count,
          limit: pagination.limit
        }}
        onPageChange={handlePageChange}
        actionButtons={[
          {
            label: "Onboard Compute Config",
            onClick: handleOnboardClick,
            variant: "contained",
            color: "#522b4a",
            hoverColor: "#2c3e50"
          }
        ]}
      />

      {/* Onboard Expression Editor */}
      <InfixExpressionEditor
        open={openOnboardModal}
        onClose={handleCloseOnboardModal}
        onSubmit={handleOnboardSubmit}
        title="Onboard Compute Config"
        submitting={submitting}
      />

      {/* Edit Expression Editor */}
      <InfixExpressionEditor
        open={openEditModal}
        onClose={handleCloseEditModal}
        onSubmit={handleEditSubmit}
        initialExpression={selectedRowForEdit?.InfixExpression || ''}
        initialPostfixExpression={selectedRowForEdit?.PostfixExpression || ''}
        title="Edit Compute Config"
        submitting={submitting}
      />

      {/* Test Modal */}
      <TestConfigModal
        open={openTestModal}
        onClose={handleCloseTestModal}
        selectedConfig={selectedRowForTest}
        user={user}
        showNotification={showNotification}
      />

      {/* Production Credential Modal */}
      <ProductionCredentialModal
        open={showProdCredentialModal}
        onClose={handleProductionCredentialClose}
        onSuccess={handleProductionCredentialSuccess}
        title="Production Credential Verification"
        description="Please enter your production credentials to proceed with the Numerix configuration promotion."
      />

      {/* Notification Toast */}
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleCloseNotification}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
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

export default NumerixConfigDiscoveryRegistry;
