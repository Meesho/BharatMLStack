import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Chip,
} from '@mui/material';
import StorageIcon from '@mui/icons-material/Storage';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import PersonIcon from '@mui/icons-material/Person';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import StatusChip from './StatusChip';

/**
 * Reusable View Detail Modal Component
 * Displays detailed information about a record in a structured modal
 * 
 * @param {boolean} open - Whether the modal is open
 * @param {function} onClose - Callback when modal is closed
 * @param {object} data - The data object to display
 * @param {object} config - Configuration object for customizing the modal
 * @param {array} config.sections - Array of section configurations (each may have fields or render(data))
 * @param {string} config.title - Modal title
 * @param {object} config.actions - Custom action buttons
 */
const ViewDetailModal = ({ 
  open, 
  onClose, 
  data,
  config = {}
}) => {
  if (!data) return null;

  const {
    title = 'Details',
    icon: Icon = StorageIcon,
    sections = [],
    actions = null
  } = config;

  // Default sections if none provided
  const defaultSections = sections.length > 0 ? sections : [
    {
      title: 'Information',
      icon: InfoIcon,
      fields: [
        { label: 'Request ID', key: 'request_id' },
        { label: 'Status', key: 'status', type: 'status' }
      ]
    },
    {
      title: 'Configuration',
      icon: SettingsIcon,
      fields: [
        { label: 'Configuration ID', key: 'payload.conf_id' },
        { label: 'Database Type', key: 'payload.db', type: 'chip' }
      ]
    },
    {
      title: 'Metadata',
      icon: PersonIcon,
      fields: [
        { label: 'Created By', key: 'created_by' },
        { label: 'Created At', key: 'created_at', type: 'date' }
      ]
    }
  ];

  const getNestedValue = (obj, path) => {
    return path.split('.').reduce((current, key) => current?.[key], obj);
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
  };

  const renderFieldValue = (field, value) => {
    switch (field.type) {
      case 'status':
        return <StatusChip status={value} />;
      case 'chip':
        return (
          <Chip 
            label={value || 'N/A'}
            size="small"
            sx={{ 
              backgroundColor: '#e3f2fd',
              color: '#1976d2',
              fontWeight: 600,
              mt: 0.5
            }}
          />
        );
      case 'date':
        return (
          <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
            <AccessTimeIcon fontSize="small" sx={{ color: '#1976d2' }} />
            {formatDate(value)}
          </Typography>
        );
      case 'monospace':
        return (
          <Typography variant="body1" sx={{ fontFamily: 'monospace', backgroundColor: '#f5f5f5', p: 1, borderRadius: 0.5, mt: 0.5 }}>
            {value || 'N/A'}
          </Typography>
        );
      default:
        return (
          <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
            {value || 'N/A'}
          </Typography>
        );
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ pb: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Icon sx={{ color: '#522b4a' }} />
          {title}
        </Box>
      </DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 2 }}>
          {defaultSections.map((section, sectionIndex) => {
            const sectionContent = section.render ? section.render(data) : null;
            if (section.render && (sectionContent == null || sectionContent === false)) return null;

            const SectionIcon = section.icon || InfoIcon;
            const sectionBgColor = section.backgroundColor || 'rgba(82, 43, 74, 0.02)';
            const sectionBorderColor = section.borderColor || 'rgba(82, 43, 74, 0.1)';

            return (
              <Box 
                key={sectionIndex}
                sx={{ 
                  p: 2, 
                  backgroundColor: sectionBgColor, 
                  borderRadius: 1, 
                  border: `1px solid ${sectionBorderColor}` 
                }}
              >
                {section.title ? (
                  <Typography 
                    variant="body2" 
                    color="text.secondary" 
                    sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}
                  >
                    <SectionIcon fontSize="small" sx={{ color: '#522b4a' }} />
                    {section.title}
                  </Typography>
                ) : null}

                {section.render ? (
                  sectionContent
                ) : (
                  <Box sx={section.layout === 'grid' ? { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 } : { display: 'flex', flexDirection: 'column', gap: 2 }}>
                    {section.fields.map((field, fieldIndex) => {
                      const value = getNestedValue(data, field.key);

                      return (
                        <Box key={fieldIndex}>
                          <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                            {field.label}
                          </Typography>
                          {renderFieldValue(field, value)}
                        </Box>
                      );
                    })}
                  </Box>
                )}
              </Box>
            );
          })}
        </Box>
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 2, backgroundColor: '#fafafa', borderTop: '1px solid #e0e0e0' }}>
        {actions ? (
          typeof actions === 'function' ? actions(data, onClose) : actions
        ) : (
          <Button 
            onClick={onClose}
            sx={{ 
              color: '#522b4a',
              '&:hover': { backgroundColor: 'rgba(82, 43, 74, 0.04)' }
            }}
          >
            Close
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

export default ViewDetailModal;
