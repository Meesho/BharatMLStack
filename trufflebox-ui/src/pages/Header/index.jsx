import React, { useState, useCallback } from 'react';
import { Navbar, Offcanvas, Dropdown } from 'react-bootstrap';
import MenuIcon from '@mui/icons-material/Menu';
import StorageIcon from '@mui/icons-material/Storage';
import DashboardIcon from '@mui/icons-material/Dashboard';
import FolderIcon from '@mui/icons-material/Folder';
import SettingsIcon from '@mui/icons-material/Settings';
import ApprovalIcon from '@mui/icons-material/TaskAlt';
import BugReportIcon from '@mui/icons-material/BugReport';
import PersonIcon from '@mui/icons-material/Person';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunch';
import CategoryIcon from '@mui/icons-material/Category';
import ModelTrainingIcon from '@mui/icons-material/ModelTraining';
import ScienceIcon from '@mui/icons-material/Science';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import ScheduleIcon from '@mui/icons-material/Schedule';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import LogoutIcon from '@mui/icons-material/Logout';
import PropTypes from 'prop-types';
import './Header.css';
import { useAuth } from '../Auth/AuthContext';
import { requiresPermissionCheck, getPermissionInfo } from '../../constants/serviceMapping';
import { 
  isOnlineFeatureStoreEnabled, 
  isInferFlowEnabled, 
  isNumerixEnabled, 
  isPredatorEnabled, 
  isEmbeddingPlatformEnabled 
} from '../../config';

function Header({ onMenuItemClick }) {
  const [show, setShow] = useState(false);
  const [expandedItems, setExpandedItems] = useState({});
  const { user, logout, hasScreenAccess } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const isActivePath = (path) => {
    return location.pathname === path;
  };

  const getMenuIcon = (key) => {
    const iconMap = {
      'FeatureStore': <StorageIcon />,
      'InferFlow': <DashboardIcon />,
      'Numerix': <FolderIcon />,
      'Predator': <BugReportIcon />,
      'UserManagement': <PersonIcon />,
      'EmbeddingPlatform': <RocketLaunchIcon />,
      'Discovery': <FolderIcon />,
      'FeatureRegistry': <StorageIcon />,
      'FeatureApproval': <ApprovalIcon />,
      'MPDiscovery': <FolderIcon />,
      'MPApproval': <ApprovalIcon />,
      'MPTesting': <BugReportIcon />,
      'DiscoveryRegistry': <FolderIcon />,
      'NumerixApproval': <ApprovalIcon />,
      'NumerixTesting': <BugReportIcon />,
      'PredatorApproval': <ApprovalIcon />,
      'Testing': <BugReportIcon />,
      'EmbeddingDiscovery': <FolderIcon />,
      'EmbeddingRegistry': <StorageIcon />,
      'EmbeddingApproval': <ApprovalIcon />,
      'EmbeddingOperations': <SettingsIcon />,
      'StoreDiscovery': <StorageIcon />,
      'EntityDiscovery': <CategoryIcon />,
      'ModelDiscovery': <ModelTrainingIcon />,
      'VariantDiscovery': <ScienceIcon />,
      'FilterDiscovery': <FilterAltIcon />,
      'JobFrequencyDiscovery': <ScheduleIcon />,
      'StoreRegistry': <StorageIcon />,
      'EntityRegistry': <CategoryIcon />,
      'ModelRegistry': <ModelTrainingIcon />,
      'VariantRegistry': <ScienceIcon />,
      'FilterRegistry': <FilterAltIcon />,
      'JobFrequencyRegistry': <ScheduleIcon />,
      'EmbeddingStoreApproval': <StorageIcon />,
      'EmbeddingEntityApproval': <CategoryIcon />,
      'EmbeddingModelApproval': <ModelTrainingIcon />,
      'EmbeddingVariantApproval': <ScienceIcon />,
      'EmbeddingFilterApproval': <FilterAltIcon />,
      'EmbeddingJobFrequencyApproval': <ScheduleIcon />,
      'DeploymentOperations': <CloudUploadIcon />,
    };
    return iconMap[key] || <StorageIcon />;
  };

  const allMenuItems = [
    {
      key: 'FeatureStore',
      label: 'Online Feature Store',
      subItems: null,
      roles: ['user', 'admin'],
      children: [
        {
          key: 'Discovery',
          label: 'Discovery',
          subItems: [
            { key: 'FeatureDiscovery', label: 'Feature', path: '/feature-discovery' },
            { key: 'StoreDiscovery', label: 'Store', path: '/store-discovery' },
            { key: 'JobDiscovery', label: 'Jobs', path: '/job-discovery' },
            { key: 'ClientDiscovery', label: 'Clients', path: '/client-discovery' },
          ],
          roles: ['user', 'admin'],
        },
        {
          key: 'FeatureRegistry',
          label: 'Registry',
          subItems: [
            { key: 'StoreRegistry', label: 'Store', path: '/feature-registry/store' },
            { key: 'JobRegistry', label: 'Jobs / Clients', path: '/feature-registry/job' },
            { key: 'EntityRegistry', label: 'Entity', path: '/feature-registry/entity' },
            { key: 'FeatureGroupRegistry', label: 'Feature Group', path: '/feature-registry/feature-group' },
            { key: 'FeatureAddition', label: 'Feature', path: '/feature-registry/feature' },
          ],
          roles: ['user', 'admin'],
        },
        {
          key: 'FeatureApproval',
          label: 'Approval',
          subItems: [
            { key: 'Stores', label: 'Stores', path: '/feature-approval/store' },
            { key: 'Jobs', label: 'Jobs / Clients', path: '/feature-approval/job' },
            { key: 'Entities', label: 'Entities', path: '/feature-approval/entity' },
            { key: 'FeatureGroups', label: 'Feature Groups', path: '/feature-approval/feature-group' },
            { key: 'Features', label: 'Features', path: '/feature-approval/features' },
          ],
          roles: ['admin'],
        },
      ]
    },
    {
      key: 'InferFlow',
      label: 'InferFlow',
      subItems: null,
      roles: null,
      children: [
        {
          key: 'MPDiscovery',
          label: 'Discovery / Registry',
          subItems: [
            { key: 'Deployable', label: 'Deployable', path: '/inferflow/deployable', screenType: 'deployable' },
            { key: 'MPConfig', label: 'Config', path: '/inferflow/config-registry', screenType: 'inferflow-config' },
          ],
          roles: null,
        },
        {
          key: 'MPApproval',
          label: 'Approval',
          subItems: [
            { key: 'MPConfigApproval', label: 'Config', path: '/inferflow/config-approval', screenType: 'inferflow-config-approval' },
          ],
          roles: null,
        },

      ]
    },
    {
      key: 'Numerix',
      label: 'Numerix',
      roles: null,
      subItems: [
        {
          key: 'Config',
          label: 'Config',
          path: '/numerix/config'
        },
        {
          key: 'Approval',
          label: 'Approval',
          path: '/numerix/config-approval'
        },
      ]
    },
    {
      key: 'Predator',
      label: 'Predator',
      subItems: null,
      roles: null,
      children: [
        {
          key: 'DiscoveryRegistry',
          label: 'Discovery/Registry',
          subItems: [
            { key: 'Deployable', label: 'Deployable', path: '/predator/discovery-registry/deployable' },
            { key: 'Model', label: 'Model', path: '/predator/discovery-registry/model' },
          ],
          roles: null,
        },
        {
          key: 'PredatorApproval',
          label: 'Approval',
          subItems: [
            { key: 'ModelApproval', label: 'Model Approval', path: '/predator/approval/model' },
          ],
          roles: null,
        },
      ]
    },
    {
      key: 'EmbeddingPlatform',
      label: 'Embedding Platform',
      subItems: null,
      roles: null,
      children: [
        {
          key: 'EmbeddingDiscovery',
          label: 'Discovery',
          subItems: [
            { key: 'StoreDiscovery', label: 'Store', path: '/embedding-platform/discovery/stores', screenType: 'store-discovery' },
            { key: 'EntityDiscovery', label: 'Entity', path: '/embedding-platform/discovery/entities', screenType: 'entity-discovery' },
            { key: 'ModelDiscovery', label: 'Model', path: '/embedding-platform/discovery/models', screenType: 'model-discovery' },
            { key: 'VariantDiscovery', label: 'Variant', path: '/embedding-platform/discovery/variants', screenType: 'variant-discovery' },
            { key: 'FilterDiscovery', label: 'Filter', path: '/embedding-platform/discovery/filters', screenType: 'filter-discovery' },
            { key: 'JobFrequencyDiscovery', label: 'Job Frequency', path: '/embedding-platform/discovery/job-frequencies', screenType: 'job-frequency-discovery' },
          ],
          roles: null,
        },
        {
          key: 'EmbeddingRegistry',
          label: 'Registry',
          subItems: [
            { key: 'StoreRegistry', label: 'Store', path: '/embedding-platform/registry/store', screenType: 'store-registry' },
            { key: 'EntityRegistry', label: 'Entity', path: '/embedding-platform/registry/entity', screenType: 'entity-registry' },
            { key: 'ModelRegistry', label: 'Model', path: '/embedding-platform/registry/model', screenType: 'model-registry' },
            { key: 'VariantRegistry', label: 'Variant', path: '/embedding-platform/registry/variant', screenType: 'variant-registry' },
            { key: 'FilterRegistry', label: 'Filter', path: '/embedding-platform/registry/filter', screenType: 'filter-registry' },
            { key: 'JobFrequencyRegistry', label: 'Job Frequency', path: '/embedding-platform/registry/job-frequency', screenType: 'job-frequency-registry' },
          ],
          roles: null,
        },
        {
          key: 'EmbeddingApproval',
          label: 'Approval',
          subItems: [
            { key: 'EmbeddingStoreApproval', label: 'Store', path: '/embedding-platform/approval/store', screenType: 'store-approval' },
            { key: 'EmbeddingEntityApproval', label: 'Entity', path: '/embedding-platform/approval/entity', screenType: 'entity-approval' },
            { key: 'EmbeddingModelApproval', label: 'Model', path: '/embedding-platform/approval/model', screenType: 'model-approval' },
            { key: 'EmbeddingVariantApproval', label: 'Variant', path: '/embedding-platform/approval/variant', screenType: 'variant-approval' },
            { key: 'EmbeddingFilterApproval', label: 'Filter', path: '/embedding-platform/approval/filter', screenType: 'filter-approval' },
            { key: 'EmbeddingJobFrequencyApproval', label: 'Job Frequency', path: '/embedding-platform/approval/job-frequency', screenType: 'job-frequency-approval' },
          ],
          roles: ['admin'],
        },
        {
          key: 'EmbeddingOperations',
          label: 'Operations',
          subItems: [
            { key: 'DeploymentOperations', label: 'Deployment', path: '/embedding-platform/deployment-operations', screenType: 'deployment-operations' },
            { key: 'OnboardVariantToDB', label: 'Onboard to DB', path: '/embedding-platform/onboard-variant-to-db', screenType: 'onboard-variant-to-db' },
            { key: 'OnboardVariantApproval', label: 'DB Approvals', path: '/embedding-platform/onboard-variant-approval', screenType: 'onboard-variant-approval', roles: ['admin'] },
          ],
          roles: null,
        },
      ]
    },
    {
      key: 'UserManagement',
      label: 'User Management',
      path: '/user-management',
      roles: ['admin'],
    },
  ];

  // Filter menu items based on feature flags
  const menuItems = allMenuItems.filter(item => {
    // Hide services if feature flags are not enabled
    if (item.key === 'FeatureStore' && !isOnlineFeatureStoreEnabled()) {
      return false;
    }
    if (item.key === 'InferFlow' && !isInferFlowEnabled()) {
      return false;
    }
    if (item.key === 'Numerix' && !isNumerixEnabled()) {
      return false;
    }
    if (item.key === 'Predator' && !isPredatorEnabled()) {
      return false;
    }
    if (item.key === 'EmbeddingPlatform' && !isEmbeddingPlatformEnabled()) {
      return false;
    }
    return true;
  });

  const hasMenuAccess = (menuKey, parentKey = null) => {
    if (!requiresPermissionCheck(menuKey, parentKey)) {
      return true;
    }
    
    let permissionInfo = getPermissionInfo(menuKey, parentKey);
    
    if (menuKey === 'Deployable') {
      if (parentKey === 'InferFlow') {
        permissionInfo = { service: 'inferflow', screenType: 'deployable' };
      } else if (parentKey === 'Predator') {
        permissionInfo = { service: 'predator', screenType: 'deployable' };
      }
    }
    
    if (permissionInfo) {
      return hasScreenAccess(permissionInfo.service, permissionInfo.screenType);
    }
    
    return false;
  };

  const toggleExpanded = (key) => {
    setExpandedItems(prev => ({
      ...prev,
      [key]: !prev[key]
    }));
  };

  const handleNavigation = (path, breadcrumb) => {
    if (path) {
      navigate(path);
      onMenuItemClick && onMenuItemClick(breadcrumb[breadcrumb.length - 1]);
      handleClose();
    }
  };

  const initializeExpandedState = useCallback(() => {
    const currentPath = location.pathname;
    const expanded = {};

    // Find the current active path and set expanded states
    menuItems.forEach((parentItem) => {
      // Check role permissions for parent item
      if (parentItem.roles && !parentItem.roles.includes(user?.role)) {
        return;
      }

      if (parentItem.path && isActivePath(parentItem.path)) {
        return;
      }

      if (parentItem.subItems) {
        const activeSubItem = parentItem.subItems.find(subItem => {
          const hasAccess = hasMenuAccess(subItem.key, parentItem.key);
          return isActivePath(subItem.path) && (!requiresPermissionCheck(subItem.key, parentItem.key) || hasAccess);
        });
        if (activeSubItem) {
          expanded[parentItem.key] = true;
        }
      }

      if (parentItem.children) {
        parentItem.children.forEach((childItem) => {
          // Check role permissions for child item
          if (childItem.roles && !childItem.roles.includes(user?.role)) {
            return;
          }

          if (childItem.subItems) {
            const activeSubItem = childItem.subItems.find(subItem => {
              const hasAccess = hasMenuAccess(subItem.key, parentItem.key);
              return isActivePath(subItem.path) && (!requiresPermissionCheck(subItem.key, parentItem.key) || hasAccess);
            });
            if (activeSubItem) {
              expanded[parentItem.key] = true;
              expanded[`${parentItem.key}-${childItem.key}`] = true;
            }
          }
        });
      }
    });

    setExpandedItems(expanded);
  }, [location.pathname, user?.role, hasScreenAccess]);

  // Initialize expanded state when component mounts or location changes
  React.useEffect(() => {
    initializeExpandedState();
  }, [location.pathname, user?.role, initializeExpandedState]);

  const handleShow = () => {
    setShow(true);
  };
  const handleClose = () => setShow(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const renderBreadcrumb = () => {
    const pathSegments = location.pathname.split('/').filter(segment => segment);
    if (pathSegments.length === 0) return null;
    
    return (
      <div className="breadcrumb-container">
        <div className="breadcrumb-path">
          {pathSegments.map((segment, index) => (
            <span key={index} className="breadcrumb-item">
              {segment}
              {index < pathSegments.length - 1 && <ChevronRightIcon className="breadcrumb-separator" />}
            </span>
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="header-creative">
      {/* Top Navbar */}
      <Navbar className="navbar-creative px-4 py-3" expand="lg">
        <div className="d-flex align-items-center">
          <div className="menu-toggle-creative" onClick={handleShow}>
            <MenuIcon />
          </div>
          
          <Navbar.Brand className="brand-creative ms-3">
            <div className="brand-main-creative">TruffleBox</div>
            <div className="brand-powered-creative">
              Powered by <span className="brand-meesho-creative">Meesho</span>
            </div>
          </Navbar.Brand>
        </div>

        <Navbar.Collapse className="justify-content-end">
          <Dropdown align="end">
            <Dropdown.Toggle className="profile-dropdown-creative">
              <div className="profile-avatar-creative">
                <AccountCircleIcon />
              </div>
              <div className="profile-info-creative">
                <span className="profile-name-creative">{user?.email || 'User'}</span>
                <span className="profile-role-creative">{user?.role || 'Member'}</span>
              </div>
            </Dropdown.Toggle>
            <Dropdown.Menu className="profile-menu-creative">
              <Dropdown.Item onClick={handleLogout} className="logout-item-creative">
                <LogoutIcon className="me-2" />
                Logout
              </Dropdown.Item>
            </Dropdown.Menu>
          </Dropdown>
        </Navbar.Collapse>
      </Navbar>

      {/* Creative Sidebar */}
      <Offcanvas show={show} onHide={handleClose} placement="start" className="sidebar-creative">
        <Offcanvas.Header closeButton className="sidebar-header-creative">
          <Offcanvas.Title className="sidebar-title-creative">
            <DashboardIcon className="me-2" />
            Control Center
          </Offcanvas.Title>
        </Offcanvas.Header>
        
        <Offcanvas.Body className="sidebar-body-creative">
          {renderBreadcrumb()}
          
          <div className="navigation-creative">
            {menuItems?.map((parentItem) => {
              if (parentItem.roles && !parentItem.roles.includes(user?.role)) {
                return null;
              }

              // Check access for parent items
              if (parentItem.subItems && !parentItem.path) {
                const accessibleSubItems = parentItem.subItems.filter(subItem => {
                  const hasAccess = hasMenuAccess(subItem.key, parentItem.key);
                  return !requiresPermissionCheck(subItem.key, parentItem.key) || hasAccess;
                });
                if (accessibleSubItems.length === 0) return null;
              }

              if (parentItem.children && !parentItem.path) {
                const hasAccessibleChildren = parentItem.children.some(childItem => {
                  if (childItem.roles && !childItem.roles.includes(user?.role)) {
                    return false;
                  }
                  if (childItem.subItems) {
                    const accessibleSubItems = childItem.subItems.filter(subItem => {
                      const hasAccess = hasMenuAccess(subItem.key, parentItem.key);
                      return !requiresPermissionCheck(subItem.key, parentItem.key) || hasAccess;
                    });
                    return accessibleSubItems.length > 0;
                  }
                  return true;
                });
                if (!hasAccessibleChildren) return null;
              }

              return (
                <div key={parentItem.key} className="nav-section-creative">
                  {parentItem.path ? (
                    <Link
                      to={parentItem.path}
                      onClick={() => handleNavigation(parentItem.path, [parentItem.label])}
                      className={`nav-main-item-creative ${isActivePath(parentItem.path) ? 'active' : ''}`}
                    >
                      <div className="nav-item-content-creative">
                        <div className="nav-icon-creative">
                          {getMenuIcon(parentItem.key)}
                        </div>
                        <span className="nav-label-creative">{parentItem.label}</span>
                      </div>
                    </Link>
                  ) : (
                    <>
                      <div 
                        className={`nav-main-item-creative expandable ${expandedItems[parentItem.key] ? 'expanded' : ''}`}
                        onClick={() => toggleExpanded(parentItem.key)}
                      >
                        <div className="nav-item-content-creative">
                          <div className="nav-icon-creative">
                            {getMenuIcon(parentItem.key)}
                          </div>
                          <span className="nav-label-creative">{parentItem.label}</span>
                          <div className="expand-icon-creative">
                            {expandedItems[parentItem.key] ? <ExpandMoreIcon /> : <ChevronRightIcon />}
                          </div>
                        </div>
                      </div>
                      
                      {expandedItems[parentItem.key] && (
                        <div className="nav-children-creative">
                          {parentItem.subItems ? (
                            <div className="nav-sub-items-creative">
                              {parentItem.subItems
                                .filter((subItem) => {
                                  const hasAccess = hasMenuAccess(subItem.key, parentItem.key);
                                  return !requiresPermissionCheck(subItem.key, parentItem.key) || hasAccess;
                                })
                                .map((subItem) => (
                                  <Link
                                    key={subItem.key}
                                    to={subItem.path}
                                    onClick={() => handleNavigation(subItem.path, [parentItem.label, subItem.label])}
                                    className={`nav-sub-item-creative ${isActivePath(subItem.path) ? 'active' : ''}`}
                                  >
                                    <div className="nav-connection-line-creative"></div>
                                    <div className="nav-sub-content-creative">
                                      <div className="nav-sub-dot-creative"></div>
                                      <span className="nav-sub-label-creative">{subItem.label}</span>
                                    </div>
                                  </Link>
                                ))}
                            </div>
                          ) : (
                            parentItem.children?.map((childItem) => {
                              if (childItem.roles && !childItem.roles.includes(user?.role)) {
                                return null;
                              }

                              const accessibleSubItems = childItem.subItems?.filter((subItem) => {
                                const hasAccess = hasMenuAccess(subItem.key, parentItem.key);
                                return !requiresPermissionCheck(subItem.key, parentItem.key) || hasAccess;
                              }) || [];

                              if (childItem.subItems && accessibleSubItems.length === 0) {
                                return null;
                              }

                              const childKey = `${parentItem.key}-${childItem.key}`;

                              return (
                                <div key={childItem.key} className="nav-child-section-creative">
                                  <div 
                                    className={`nav-child-item-creative expandable ${expandedItems[childKey] ? 'expanded' : ''}`}
                                    onClick={() => toggleExpanded(childKey)}
                                  >
                                    <div className="nav-connection-line-creative"></div>
                                    <div className="nav-child-content-creative">
                                      <div className="nav-child-icon-creative">
                                        {getMenuIcon(childItem.key)}
                                      </div>
                                      <span className="nav-child-label-creative">{childItem.label}</span>
                                      <div className="expand-icon-creative">
                                        {expandedItems[childKey] ? <ExpandMoreIcon /> : <ChevronRightIcon />}
                                      </div>
                                    </div>
                                  </div>
                                  
                                  {expandedItems[childKey] && childItem.subItems && (
                                    <div className="nav-grandchildren-creative">
                                      {accessibleSubItems.map((subItem) => (
                                        <Link
                                          key={subItem.key}
                                          to={subItem.path}
                                          onClick={() => handleNavigation(subItem.path, [parentItem.label, childItem.label, subItem.label])}
                                          className={`nav-grandchild-item-creative ${isActivePath(subItem.path) ? 'active' : ''}`}
                                        >
                                          <div className="nav-grandchild-connection-creative"></div>
                                          <div className="nav-grandchild-content-creative">
                                            <div className="nav-grandchild-dot-creative"></div>
                                            <span className="nav-grandchild-label-creative">{subItem.label}</span>
                                          </div>
                                        </Link>
                                      ))}
                                    </div>
                                  )}
                                </div>
                              );
                            })
                          )}
                        </div>
                      )}
                    </>
                  )}
                </div>
              );
            })}
          </div>
        </Offcanvas.Body>
      </Offcanvas>
    </div>
  );
}

Header.propTypes = {
  onMenuItemClick: PropTypes.func,
};

export default Header; 