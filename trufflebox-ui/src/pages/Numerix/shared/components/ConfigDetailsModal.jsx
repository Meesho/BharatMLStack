import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  IconButton,
  Typography,
  Box,
  Paper,
  Button,
  Chip,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import PersonIcon from '@mui/icons-material/Person';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import { OpenInNew } from '@mui/icons-material';

export const ConfigDetailsModal = ({ open, onClose, config }) => {
  if (!config) return null;

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth="md"
      fullWidth
    >
      <DialogTitle sx={{ bgcolor: '#450839', color: 'white', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6">Config Details</Typography>
        <IconButton onClick={onClose} sx={{ color: 'white' }}>
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent sx={{ mt: 2 }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {/* Config ID */}
          <Box>
            <Typography variant="subtitle2" color="textSecondary">Config ID</Typography>
            <Typography variant="body1">{config.ComputeId}</Typography>
          </Box>

          {/* Expressions */}
          <Box>
            <Typography variant="subtitle2" color="textSecondary">Infix Expression</Typography>
            <Paper elevation={0} sx={{ p: 2, bgcolor: '#f5f5f5', fontFamily: 'monospace' }}>
              {config.InfixExpression}
            </Paper>
          </Box>
          <Box>
            <Typography variant="subtitle2" color="textSecondary">Postfix Expression</Typography>
            <Paper elevation={0} sx={{ p: 2, bgcolor: '#f5f5f5', fontFamily: 'monospace' }}>
              {config.PostfixExpression}
            </Paper>
          </Box>

          {/* Creation Info */}
          <Box sx={{ display: 'flex', gap: 4 }}>
            <Box>
              <Typography variant="subtitle2" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <PersonIcon fontSize="small" />
                Created By
              </Typography>
              <Typography variant="body1">{config.Created_by}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <CalendarTodayIcon fontSize="small" />
                Created At
              </Typography>
              <Typography variant="body1">{config.Created_at}</Typography>
            </Box>
          </Box>

          {/* Update Info */}
          {config.Updated_by && (
            <Box sx={{ display: 'flex', gap: 4 }}>
              <Box>
                <Typography variant="subtitle2" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <PersonIcon fontSize="small" />
                  Updated By
                </Typography>
                <Typography variant="body1">{config.Updated_by}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <CalendarTodayIcon fontSize="small" />
                  Updated At
                </Typography>
                <Typography variant="body1">{config.Updated_at}</Typography>
              </Box>
            </Box>
          )}

          {/* Status Section */}
          <Box sx={{ display: 'flex', gap: 4 }}>
            {/* Test Status */}
            <Box>
              <Typography variant="subtitle2" color="textSecondary" gutterBottom>
                Test Status
              </Typography>
              <Chip 
                label={config.test_results?.is_functionally_tested ? "Functionally Tested" : "Not Tested"}
                color={config.test_results?.is_functionally_tested ? "success" : "warning"}
                size="small"
              />
            </Box>
          </Box>

          {/* Monitoring URL */}
          {config.dashboard_link && (
            <Box>
              <Typography variant="subtitle2" color="textSecondary">Monitoring Dashboard</Typography>
              <Button
                startIcon={<OpenInNew />}
                onClick={() => window.open(config.dashboard_link, '_blank')}
                sx={{ color: '#2196f3', textTransform: 'none' }}
              >
                Open Dashboard
              </Button>
            </Box>
          )}
        </Box>
      </DialogContent>
    </Dialog>
  );
}; 