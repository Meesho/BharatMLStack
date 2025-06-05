import React, { useState } from 'react';
import { Navbar, Accordion, Offcanvas, ListGroup, Dropdown } from 'react-bootstrap';
import MenuIcon from '@mui/icons-material/Menu';
import StorageIcon from '@mui/icons-material/Storage';
import { Link, useNavigate } from 'react-router-dom';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import PropTypes from 'prop-types';
import './Header.css';
import { useAuth } from '../Auth/AuthContext';

function Header({ onMenuItemClick }) {
  const [show, setShow] = useState(false);
  const { user, logout } = useAuth(); // Get user and logout function from AuthContext
  const navigate = useNavigate();

  const handleShow = () => setShow(true);
  const handleClose = () => setShow(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const menuItems = [
    // Parent: Online Feature Store
    {
      key: 'FeatureStore',
      label: 'Online Feature Store',
      subItems: null,
      roles: ['user', 'admin'],
      children: [
        // Feature Store Discovery
        {
          key: 'Discovery',
          label: 'Discovery',
          subItems: [
            { key: 'FeatureDiscovery', label: 'Feature', path: '/feature-discovery' },
            { key: 'StoreDiscovery', label: 'Store', path: '/store-discovery' },
            { key: 'JobDiscovery', label: 'Jobs', path: '/job-discovery' },
          ],
          roles: ['user', 'admin'], // Accessible by both roles
        },
        // Feature Registry
        {
          key: 'FeatureRegistry',
          label: 'Registry',
          subItems: [
            { key: 'StoreRegistry', label: 'Store', path: '/feature-registry/store' },
            { key: 'JobRegistry', label: 'Jobs', path: '/feature-registry/job' },
            { key: 'EntityRegistry', label: 'Entity', path: '/feature-registry/entity' },
            { key: 'FeatureGroupRegistry', label: 'Feature Group', path: '/feature-registry/feature-group' },
            { key: 'FeatureAddition', label: 'Feature', path: '/feature-registry/feature' },
          ],
          roles: ['user', 'admin'],
        },
        // Feature Approval
        {
          key: 'FeatureApproval',
          label: 'Approval',
          subItems: [
            { key: 'Stores', label: 'Stores', path: '/feature-approval/store' },
            { key: 'Jobs', label: 'Jobs', path: '/feature-approval/job' },
            { key: 'Entities', label: 'Entities', path: '/feature-approval/entity' },
            { key: 'FeatureGroups', label: 'Feature Groups', path: '/feature-approval/feature-group' },
            { key: 'Features', label: 'Features', path: '/feature-approval/features' },
          ],
          roles: ['admin'], // Accessible by admins only
        },
      ]
    },
  ];

  return (
    <div>
      {/* Top Navbar */}
      <Navbar bg="primary" expand="lg" className="px-3">
        <MenuIcon
          onClick={handleShow}
          style={{ cursor: 'pointer', color: '#ffffff', marginRight: '10px' }}
        />
        <Navbar.Brand href="#home" className="fw-bold text-white d-flex flex-column">
          <div className="fs-5">TruffleBox</div>
          <div className="small fw-bold" style={{ fontSize: '0.7rem' }}>(Powered by <span style={{ fontSize: '0.9rem' }}>Meesho</span>)</div>
        </Navbar.Brand>
        <Navbar.Collapse className="justify-content-end">
          <Dropdown align="end">
            <Dropdown.Toggle
              variant="link"
              id="dropdown-profile"
              className="text-white d-flex align-items-center"
              style={{ textDecoration: 'none', cursor: 'pointer' }}
            >
              <AccountCircleIcon style={{ marginRight: '8px' }} />
              {user?.email || 'User'}
            </Dropdown.Toggle>
            <Dropdown.Menu>
              <Dropdown.Item onClick={handleLogout}>Logout</Dropdown.Item>
            </Dropdown.Menu>
          </Dropdown>
        </Navbar.Collapse>
      </Navbar>

      {/* Offcanvas Sidebar */}
      <Offcanvas show={show} onHide={handleClose} placement="start" className="offcanvas-custom">
        <Offcanvas.Header closeButton>
          <Offcanvas.Title className="fw-bold text-dark">
            Control Center
          </Offcanvas.Title>
        </Offcanvas.Header>
        <Offcanvas.Body>
          <Accordion defaultActiveKey="0" flush>
            {menuItems.map((parentItem) =>
              // Check if the current user has access to the parent item
              parentItem.roles.includes(user?.role) ? (
                <Accordion.Item key={parentItem.key} eventKey={parentItem.key}>
                  <Accordion.Header>
                    <StorageIcon style={{ marginRight: '10px', color: '#000' }} />
                    {parentItem.label}
                  </Accordion.Header>
                  <Accordion.Body>
                    <Accordion flush className="nested-accordion">
                      {parentItem.children.map((childItem) =>
                        // Check if user has access to this child item
                        childItem.roles.includes(user?.role) ? (
                          <Accordion.Item key={childItem.key} eventKey={`${parentItem.key}-${childItem.key}`}>
                            <Accordion.Header>
                              {childItem.label}
                            </Accordion.Header>
                            <Accordion.Body>
                              {childItem.subItems && (
                                <ListGroup variant="flush">
                                  {childItem.subItems.map((subItem) => (
                                    <ListGroup.Item
                                      key={subItem.key}
                                      as={Link}
                                      to={subItem.path}
                                      onClick={() => {
                                        onMenuItemClick && onMenuItemClick(subItem.key);
                                        handleClose();
                                      }}
                                      className="offcanvas-link fw-bold fs-6"
                                      style={{ paddingLeft: '20px' }}
                                    >
                                      {subItem.label}
                                    </ListGroup.Item>
                                  ))}
                                </ListGroup>
                              )}
                            </Accordion.Body>
                          </Accordion.Item>
                        ) : null
                      )}
                    </Accordion>
                  </Accordion.Body>
                </Accordion.Item>
              ) : null
            )}
          </Accordion>
        </Offcanvas.Body>
      </Offcanvas>
    </div>
  );
}

Header.propTypes = {
  onMenuItemClick: PropTypes.func,
};

export default Header;
