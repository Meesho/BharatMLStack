import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  IconButton,
  Typography,
  Box,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Divider
} from '@mui/material';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import CloseIcon from '@mui/icons-material/Close';

const InfixExpressionGuidelines = ({ open, onClose }) => {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
    >
      <DialogTitle sx={{ 
        bgcolor: '#450839', 
        color: 'white',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        pb: 1
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <HelpOutlineIcon />
          <Typography variant="h6">Guidelines for Writing Infix Expressions</Typography>
        </Box>
        <IconButton 
          onClick={onClose} 
          sx={{ color: 'white' }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers>
        <List>
          <ListItem>
            <ListItemIcon>
              <ErrorOutlineIcon color="warning" />
            </ListItemIcon>
            <ListItemText 
              primary="Handling Unary Minus"
              secondary={
                <Box component="div" sx={{ mt: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    The parser does not support implicit unary minus. Use explicit multiplication:
                  </Typography>
                  <Box sx={{ mt: 1, mb: 1, pl: 2 }}>
                    <Typography variant="body2" color="success.main" sx={{ fontFamily: 'monospace' }}>
                      ✓ -1 * exp(x)<br />
                      ✓ -1 * pcvr
                    </Typography>
                    <Typography variant="body2" color="error.main" sx={{ fontFamily: 'monospace' }}>
                      ✗ -exp(x)<br />
                      ✗ -pcvr
                    </Typography>
                  </Box>
                </Box>
              }
            />
          </ListItem>
          
          <Divider variant="inset" component="li" />
          
          <ListItem>
            <ListItemIcon>
              <CheckCircleOutlineIcon color="success" />
            </ListItemIcon>
            <ListItemText 
              primary="Supported Operators"
              secondary={
                <Box component="div" sx={{ mt: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    Only the following operators are supported (listed by precedence):
                  </Typography>
                  <Box sx={{ mt: 1, pl: 2 }}>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                      • ^ (power)<br />
                      • * (multiply), / (divide)<br />
                      • + (add), - (subtract)<br />
                      • {'>'}, {'<'}, {'>='}, {'<='}, == (comparisons)<br />
                      • & (AND), | (OR)
                    </Typography>
                  </Box>
                </Box>
              }
            />
          </ListItem>
          
          <Divider variant="inset" component="li" />
          
          <ListItem>
            <ListItemIcon>
              <CheckCircleOutlineIcon color="success" />
            </ListItemIcon>
            <ListItemText 
              primary="Supported Functions"
              secondary={
                <Box component="div" sx={{ mt: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    The following functions are supported (case-sensitive):
                  </Typography>
                  <Box sx={{ mt: 1, pl: 2 }}>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                      Single argument:<br />
                      • exp(x), log(x), abs(x)<br />
                      • norm_min_max(x)<br />
                      • percentile_rank(x)<br />
                      • norm_percentile_0_99(x)<br />
                      • norm_percentile_5_95(x)<br />
                      <br />
                      Two arguments:<br />
                      • min(x, y), max(x, y)
                    </Typography>
                  </Box>
                </Box>
              }
            />
          </ListItem>
          
          <Divider variant="inset" component="li" />
          
          <ListItem>
            <ListItemIcon>
              <ErrorOutlineIcon color="warning" />
            </ListItemIcon>
            <ListItemText 
              primary="Common Mistakes to Avoid"
              secondary={
                <Box component="div" sx={{ mt: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    • Use explicit multiplication (no implicit)<br />
                    • Use & and | instead of && and ||<br />
                    • Add spaces around negative numbers<br />
                    • Avoid scientific notation (e.g., 1.5e-3)<br />
                    • Don't use variable names that match function names
                  </Typography>
                  <Box sx={{ mt: 1, mb: 1, pl: 2 }}>
                    <Typography variant="body2" color="success.main" sx={{ fontFamily: 'monospace' }}>
                      ✓ 2 * (x + y)<br />
                      ✓ (a & b) == c
                    </Typography>
                    <Typography variant="body2" color="error.main" sx={{ fontFamily: 'monospace' }}>
                      ✗ 2(x + y)<br />
                      ✗ a && b == c
                    </Typography>
                  </Box>
                </Box>
              }
            />
          </ListItem>
        </List>
      </DialogContent>
      <DialogActions>
        <Button 
          onClick={onClose}
          variant="contained"
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730'
            },
          }}
        >
          Got it
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default InfixExpressionGuidelines; 