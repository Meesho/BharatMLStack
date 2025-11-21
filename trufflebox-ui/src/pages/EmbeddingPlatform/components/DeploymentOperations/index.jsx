import React, { useState } from 'react';
import { Box, Tab, Tabs } from '@mui/material';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunch';
import TaskAltIcon from '@mui/icons-material/TaskAlt';
import DashboardIcon from '@mui/icons-material/Dashboard';
import DeploymentRegistry from './DeploymentRegistry';
import DeploymentApproval from './DeploymentApproval';
import DeploymentDashboard from './DeploymentDashboard';
import { useAuth } from '../../../Auth/AuthContext';

const TabPanel = ({ children, value, index, ...other }) => {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`deployment-tabpanel-${index}`}
      aria-labelledby={`deployment-tab-${index}`}
      {...other}
    >
      {value === index && <Box>{children}</Box>}
    </div>
  );
};

const DeploymentOperations = () => {
  const [value, setValue] = useState(0);
  const { user, permissions } = useAuth();
  const isAdmin = permissions?.role === 'admin';

  const handleChange = (event, newValue) => {
    setValue(newValue);
  };

  return (
    <div>
      <Box sx={{ width: '100%' }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs 
            value={value} 
            onChange={handleChange} 
            aria-label="deployment operations tabs"
            sx={{
              '& .MuiTab-root': {
                textTransform: 'none',
                fontSize: '16px',
                fontWeight: 500,
                color: '#666',
                '&.Mui-selected': {
                  color: '#522b4a',
                },
              },
              '& .MuiTabs-indicator': {
                backgroundColor: '#522b4a',
              },
            }}
          >
            <Tab 
              icon={<DashboardIcon />} 
              iconPosition="start"
              label="Dashboard" 
              id="deployment-tab-0"
              aria-controls="deployment-tabpanel-0"
            />
            <Tab 
              icon={<RocketLaunchIcon />} 
              iconPosition="start"
              label="Registry" 
              id="deployment-tab-1"
              aria-controls="deployment-tabpanel-1"
            />
            {isAdmin && (
              <Tab 
                icon={<TaskAltIcon />} 
                iconPosition="start"
                label="Approvals" 
                id="deployment-tab-2"
                aria-controls="deployment-tabpanel-2"
              />
            )}
          </Tabs>
        </Box>

        <TabPanel value={value} index={0}>
          <DeploymentDashboard />
        </TabPanel>
        
        <TabPanel value={value} index={1}>
          <DeploymentRegistry />
        </TabPanel>
        
        {isAdmin && (
          <TabPanel value={value} index={2}>
            <DeploymentApproval />
          </TabPanel>
        )}
      </Box>
    </div>
  );
};

export default DeploymentOperations;

