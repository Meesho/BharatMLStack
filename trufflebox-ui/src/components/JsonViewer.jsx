import React, { useState } from 'react';
import {
  Box,
  Typography,
  IconButton,
  Button,
  Paper,
  Chip,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  Collapse,
  Snackbar,
  Alert,
  TextField,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Error as ErrorIcon,
  Close as CloseIcon,
  Fullscreen as FullscreenIcon,
  Edit as EditIcon,
  Save as SaveIcon,
  Cancel as CancelIcon,
  CompareArrows as CompareIcon,
  ContentCopy as ContentCopyIcon,
} from '@mui/icons-material';
import JsonView from '@uiw/react-json-view';
import { lightTheme } from '@uiw/react-json-view/light';
import { darkTheme } from '@uiw/react-json-view/dark';
import JsonDiffView from './JsonDiffView';

const JsonViewer = ({ 
  data, 
  title = "JSON Viewer", 
  defaultExpanded = true,
  editable = false,
  maxHeight = 400,
  enableCopy = true,
  enableDiff = true,
  originalData = null,
  onDataChange = null,
  sx = {},
  displayDataTypes = false,
  theme = 'light'
}) => {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const [showFullscreenDialog, setShowFullscreenDialog] = useState(false);
  const [showDiffDialog, setShowDiffDialog] = useState(false);
  const [localData, setLocalData] = useState(data);
  const [isEditing, setIsEditing] = useState(false);
  const [editedData, setEditedData] = useState(data);
  const [editedText, setEditedText] = useState('');
  const [jsonError, setJsonError] = useState('');
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });
  const [showManualCopyDialog, setShowManualCopyDialog] = useState(false);

  // Update local data when prop changes
  React.useEffect(() => {
    setLocalData(data);
    setEditedData(data);
  }, [data]);

  const formattedJson = React.useMemo(() => {
    try {
      return JSON.stringify(localData, null, 2);
    } catch (error) {
      return 'Invalid JSON';
    }
  }, [localData]);

  const isJsonValid = React.useMemo(() => {
    try {
      JSON.stringify(localData);
      return true;
    } catch (error) {
      return false;
    }
  }, [localData]);

  const hasChanges = React.useMemo(() => {
    if (!originalData) return false;
    try {
      return JSON.stringify(originalData) !== JSON.stringify(localData);
    } catch (error) {
      return false;
    }
  }, [originalData, localData]);

  const handleToggleExpanded = () => {
    setExpanded(!expanded);
  };

  const handleShowFullscreen = () => {
    setShowFullscreenDialog(true);
  };

  const handleShowDiff = () => {
    setShowDiffDialog(true);
  };

  const handleStartEdit = () => {
    setIsEditing(true);
    setEditedData(localData);
    setEditedText(JSON.stringify(localData, null, 2));
    setJsonError('');
  };

  const handleSaveEdit = () => {
    try {
      const parsed = JSON.parse(editedText);
      setLocalData(parsed);
      setEditedData(parsed);
      setIsEditing(false);
      setJsonError('');
      if (onDataChange) {
        onDataChange(parsed);
      }
    } catch (error) {
      setJsonError(error.message);
      setSnackbar({
        open: true,
        message: 'Invalid JSON: ' + error.message,
        severity: 'error'
      });
    }
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditedData(localData);
    setEditedText('');
    setJsonError('');
  };

  const handleEdit = (edit) => {
    if (edit && edit.src !== undefined) {
      setEditedData(edit.src);
    } else if (edit && edit.updated_src !== undefined) {
      setEditedData(edit.updated_src);
    } else if (edit && typeof edit === 'object') {
      // Sometimes the library passes the updated object directly
      setEditedData(edit);
    }
  };

  // Open manual copy dialog for user to copy JSON
  const handleCopy = () => {
    setShowManualCopyDialog(true);
  };
  
  const handleManualCopy = () => {
    const textArea = document.getElementById('manual-copy-textarea');
    if (textArea) {
      textArea.select();
      try {
        document.execCommand('copy');
        setSnackbar({
          open: true,
          message: 'JSON copied to clipboard!',
          severity: 'success'
        });
        setShowManualCopyDialog(false);
      } catch (error) {
        console.error('Manual copy also failed:', error);
      }
    }
  };
  
  const handleCloseSnackbar = () => {
    setSnackbar({ ...snackbar, open: false });
  };

  const customTheme = theme === 'light' ? {
    ...lightTheme,
    '--w-rjv-font-family': 'Monaco, Menlo, "Ubuntu Mono", monospace',
    '--w-rjv-color': '#2c3e50',
    '--w-rjv-key-string': '#2c3e50',
    '--w-rjv-background-color': '#ffffff',
    '--w-rjv-line-color': '#f5f5f5',
    '--w-rjv-arrow-color': '#7f8c8d',
    '--w-rjv-edit-color': '#450839',
    '--w-rjv-info-color': '#95a5a6',
    '--w-rjv-update-color': '#27ae60',
    '--w-rjv-copied-color': '#27ae60',
    '--w-rjv-copied-success-color': '#27ae60',
    '--w-rjv-curlybraces-color': '#2c3e50',
    '--w-rjv-colon-color': '#2c3e50',
    '--w-rjv-brackets-color': '#2c3e50',
    '--w-rjv-ellipsis-color': '#95a5a6',
    '--w-rjv-quotes-color': '#27ae60',
    '--w-rjv-quotes-string-color': '#27ae60',
    '--w-rjv-type-string-color': '#27ae60',
    '--w-rjv-type-int-color': '#e74c3c',
    '--w-rjv-type-float-color': '#e74c3c',
    '--w-rjv-type-bigint-color': '#e74c3c',
    '--w-rjv-type-boolean-color': '#9b59b6',
    '--w-rjv-type-date-color': '#3498db',
    '--w-rjv-type-url-color': '#3498db',
    '--w-rjv-type-null-color': '#95a5a6',
    '--w-rjv-type-nan-color': '#95a5a6',
    '--w-rjv-type-undefined-color': '#95a5a6',
  } : darkTheme;

  return (
    <Box sx={{ width: '100%', ...sx }}>
      <Paper 
        elevation={2}
        sx={{ 
          border: '1px solid #e0e0e0',
          borderRadius: '8px',
          overflow: 'hidden'
        }}
      >
        {/* Header */}
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 16px',
            backgroundColor: '#f8f9fa',
            borderBottom: '1px solid #e0e0e0'
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, color: '#2c3e50' }}>
              {title}
            </Typography>
            
            {!isJsonValid && (
              <Chip
                icon={<ErrorIcon />}
                label="Invalid JSON"
                size="small"
                color="error"
                variant="outlined"
              />
            )}
            
            {isEditing && (
              <Chip
                label="Editing"
                size="small"
                color="primary"
                variant="outlined"
              />
            )}
            
            {hasChanges && enableDiff && (
              <Chip
                label="Modified"
                size="small"
                color="warning"
                variant="outlined"
              />
            )}
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {enableCopy && (
              <Tooltip title="Copy JSON">
                <IconButton 
                  size="small" 
                  onClick={handleCopy}
                  sx={{ 
                    backgroundColor: 'transparent',
                    '&:hover': { backgroundColor: '#f0f0f0' }
                  }}
                >
                  <ContentCopyIcon />
                </IconButton>
              </Tooltip>
            )}
            
            <Tooltip title="View Fullscreen">
              <IconButton 
                size="small" 
                onClick={handleShowFullscreen}
                sx={{ 
                  backgroundColor: 'transparent',
                  '&:hover': { backgroundColor: '#f0f0f0' }
                }}
              >
                <FullscreenIcon />
              </IconButton>
            </Tooltip>
            
            {enableDiff && originalData && (
              <Tooltip title="Show Diff">
                <IconButton 
                  size="small" 
                  onClick={handleShowDiff}
                  sx={{ 
                    backgroundColor: hasChanges ? '#fff3e0' : 'transparent',
                    '&:hover': { backgroundColor: '#f0f0f0' }
                  }}
                >
                  <CompareIcon />
                </IconButton>
              </Tooltip>
            )}
            
            {editable && !isEditing && (
              <Tooltip title="Edit JSON">
                <IconButton 
                  size="small" 
                  onClick={handleStartEdit}
                  sx={{ 
                    backgroundColor: 'transparent',
                    '&:hover': { backgroundColor: '#f0f0f0' }
                  }}
                >
                  <EditIcon />
                </IconButton>
              </Tooltip>
            )}
            
            {isEditing && (
              <>
                <Tooltip title="Save Changes">
                  <IconButton 
                    size="small" 
                    onClick={handleSaveEdit}
                    sx={{ 
                      backgroundColor: '#e3f2fd',
                      '&:hover': { backgroundColor: '#bbdefb' }
                    }}
                  >
                    <SaveIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Cancel Edit">
                  <IconButton 
                    size="small" 
                    onClick={handleCancelEdit}
                    sx={{ 
                      backgroundColor: '#ffebee',
                      '&:hover': { backgroundColor: '#ffcdd2' }
                    }}
                  >
                    <CancelIcon />
                  </IconButton>
                </Tooltip>
              </>
            )}
            
            <Tooltip title={expanded ? "Collapse" : "Expand"}>
              <IconButton 
                size="small" 
                onClick={handleToggleExpanded}
                sx={{ 
                  backgroundColor: 'transparent',
                  '&:hover': { backgroundColor: '#f0f0f0' }
                }}
              >
                {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        {/* JSON Content */}
        <Collapse in={expanded}>
          <Box
            sx={{
              maxHeight: maxHeight,
              overflow: 'auto',
              padding: '16px',
              backgroundColor: '#ffffff',
              '& .w-json-view-container': {
                fontSize: '14px',
                lineHeight: '1.6'
              }
            }}
          >
            {isEditing && editable ? (
              <Box>
                <TextField
                  fullWidth
                  multiline
                  value={editedText}
                  onChange={(e) => {
                    setEditedText(e.target.value);
                    // Try to parse to show real-time validation
                    try {
                      JSON.parse(e.target.value);
                      setJsonError('');
                    } catch (error) {
                      setJsonError(error.message);
                    }
                  }}
                  error={!!jsonError}
                  helperText={jsonError || 'Edit JSON and click Save'}
                  sx={{
                    '& .MuiInputBase-root': {
                      fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
                      fontSize: '13px',
                      lineHeight: '1.6',
                      padding: 0
                    },
                    '& .MuiInputBase-input': {
                      padding: '8px',
                      minHeight: `${maxHeight - 32}px`
                    }
                  }}
                  InputProps={{
                    style: {
                      backgroundColor: '#fafafa'
                    }
                  }}
                />
              </Box>
            ) : (
              <JsonView
                value={isEditing ? editedData : localData}
                collapsed={false}
                displayDataTypes={displayDataTypes}
                displayObjectSize={true}
                enableClipboard={false}
                editable={false}
                style={customTheme}
                keyName="root"
              />
            )}
          </Box>
        </Collapse>
      </Paper>

      {/* Fullscreen JSON View Dialog */}
      <Dialog
        open={showFullscreenDialog}
        onClose={() => setShowFullscreenDialog(false)}
        maxWidth="xl"
        fullWidth
        PaperProps={{
          sx: {
            maxHeight: '95vh',
            height: '95vh'
          }
        }}
      >
        <DialogTitle sx={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          padding: '16px 24px',
          borderBottom: '1px solid #e0e0e0',
          bgcolor: '#450839',
          color: 'white'
        }}>
          <Typography variant="h6" component="div">
            {title} - Fullscreen View
          </Typography>
          <IconButton
            onClick={() => setShowFullscreenDialog(false)}
            size="small"
            sx={{ 
              color: 'white',
              '&:hover': {
                backgroundColor: 'rgba(255, 255, 255, 0.1)'
              }
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ p: 3, height: 'calc(100% - 64px)', overflow: 'auto', backgroundColor: '#fafafa' }}>
          {isEditing && editable ? (
            <TextField
              fullWidth
              multiline
              value={editedText}
              onChange={(e) => {
                setEditedText(e.target.value);
                // Try to parse to show real-time validation
                try {
                  JSON.parse(e.target.value);
                  setJsonError('');
                } catch (error) {
                  setJsonError(error.message);
                }
              }}
              error={!!jsonError}
              helperText={jsonError || 'Edit JSON and click Save'}
              sx={{
                height: '100%',
                '& .MuiInputBase-root': {
                  fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
                  fontSize: '14px',
                  lineHeight: '1.8',
                  height: '100%'
                },
                '& .MuiInputBase-input': {
                  padding: '16px',
                  height: '100% !important',
                  overflow: 'auto !important'
                }
              }}
              InputProps={{
                style: {
                  backgroundColor: '#ffffff',
                  height: '100%'
                }
              }}
            />
          ) : (
            <JsonView
              value={isEditing ? editedData : localData}
              expanded={true}
              displayDataTypes={displayDataTypes}
              displayObjectSize={true}
              enableClipboard={false}
              editable={false}
              style={{
                ...customTheme,
                '--w-rjv-font-family': 'Monaco, Menlo, "Ubuntu Mono", monospace',
                fontSize: '15px',
                lineHeight: '1.8'
              }}
              keyName="root"
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Diff View Dialog */}
      <Dialog
        open={showDiffDialog}
        onClose={() => setShowDiffDialog(false)}
        maxWidth="xl"
        fullWidth
        PaperProps={{
          sx: {
            maxHeight: '90vh',
            height: '90vh'
          }
        }}
      >
        <DialogTitle sx={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          padding: '16px 24px',
          borderBottom: '1px solid #e0e0e0'
        }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CompareIcon />
            <Typography variant="h6" component="div">
              JSON Configuration Diff
            </Typography>
          </Box>
          <IconButton
            onClick={() => setShowDiffDialog(false)}
            size="small"
            sx={{ 
              color: 'text.secondary',
              '&:hover': {
                backgroundColor: 'action.hover'
              }
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ p: 0, height: 'calc(100% - 64px)' }}>
          <JsonDiffView
            originalText={originalData ? JSON.stringify(originalData, null, 2) : ''}
            changedText={formattedJson}
            title=""
          />
        </DialogContent>
      </Dialog>

      {/* Manual Copy Dialog - Fallback for HTTP contexts */}
      <Dialog
        open={showManualCopyDialog}
        onClose={() => setShowManualCopyDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle sx={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          backgroundColor: '#f5f5f5',
          borderBottom: '1px solid #ddd'
        }}>
          <Typography variant="h6">Manual Copy</Typography>
          <IconButton
            onClick={() => setShowManualCopyDialog(false)}
            size="small"
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ p: 3 }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            Click "Copy to Clipboard" or manually select the text and press Ctrl+C (Windows/Linux) or Cmd+C (Mac) to copy.
          </Alert>
          <Box sx={{ mb: 2 }}>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              JSON Content:
            </Typography>
            <Box
              component="textarea"
              id="manual-copy-textarea"
              value={formattedJson}
              readOnly
              onClick={(e) => e.target.select()}
              sx={{
                width: '100%',
                minHeight: '400px',
                maxHeight: '60vh',
                fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
                fontSize: '13px',
                padding: '12px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                backgroundColor: '#f9f9f9',
                resize: 'vertical',
                '&:focus': {
                  outline: '2px solid #1976d2',
                  outlineOffset: '2px'
                }
              }}
            />
          </Box>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1 }}>
            <Button
              onClick={handleManualCopy}
              variant="contained"
              color="primary"
              startIcon={<ContentCopyIcon />}
            >
              Copy to Clipboard
            </Button>
          </Box>
        </DialogContent>
      </Dialog>

      {/* Snackbar for copy notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={4000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert 
          onClose={handleCloseSnackbar} 
          severity={snackbar.severity}
          sx={{ width: '100%' }}
          variant="filled"
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default JsonViewer;

