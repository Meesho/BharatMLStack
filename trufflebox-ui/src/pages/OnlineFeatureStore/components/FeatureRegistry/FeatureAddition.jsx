import React, { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Divider,
  Table as MuiTable,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
  Box,
  IconButton,
} from '@mui/material';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import "./styles.scss";
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';

const FeatureAddition = () => {
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [entities, setEntities] = useState([]);
  const [featureGroups, setFeatureGroups] = useState([]);
  const [featuresRequests, setFeaturesRequests] = useState([]);
  const [selectedFeatureInfo, setSelectedFeatureInfo] = useState(null);
  const { user } = useAuth();

  const cellStyle = {
    maxWidth: "150px", 
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "normal",
    wordBreak: "break-word",
    borderRight: "1px solid rgba(224, 224, 224, 1)"
  };

  const headerCellStyle = {
    ...cellStyle,
    backgroundColor: '#f5f5f5',
    fontWeight: 'bold',
    wordBreak: "break-word",
    borderRight: "1px solid rgba(224, 224, 224, 1)"
  };

  const [FeatureAdditionData, setFeatureAdditionData] = useState({
    "entity-label": "",
    "feature-group-label": "",
    features: [{ 
      labels: "", 
      "default-values": "",
      "source-base-path": "",
      "source-data-column": "",
      "storage-provider": ""
    }],
  });

  // Add these state variables
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');

  // Memoize fetch functions with useCallback
  const fetchEntities = useCallback(async () => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-entites`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setEntities(data);
    } catch (error) {
      console.error('Error fetching entities:', error);
    }
  }, [user.token]);

  const fetchFeatureGroups = useCallback(async (entityLabel) => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-feature-groups?entityLabel=${entityLabel}`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setFeatureGroups(data);
    } catch (error) {
      console.error('Error fetching feature groups:', error);
    }
  }, [user.token]);

  const fetchFeaturesRequests = useCallback(async () => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-add-features-requests`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setFeaturesRequests(data);
    } catch (error) {
      console.error('Error fetching features requests:', error);
    }
  }, [user.token]);

  useEffect(() => {
    fetchEntities();
    fetchFeaturesRequests();
  }, [fetchEntities, fetchFeaturesRequests]);

  // Extract the entity label to a variable to avoid complex expression
  const entityLabel = FeatureAdditionData["entity-label"];
  useEffect(() => {
    if (entityLabel) {
      fetchFeatureGroups(entityLabel);
    }
  }, [entityLabel, fetchFeatureGroups]);

  // Existing handle functions for creation modal
  const handleChange = (e) => {
    const { name, value } = e.target;
    setFeatureAdditionData((prevData) => ({
      ...prevData,
      [name]: value,
      ...(name === "entity-label" && { "feature-group-label": "" }),
    }));
  };

  const handleFeatureChange = (index, e) => {
    const { name, value } = e.target;
    const updatedFeatures = [...FeatureAdditionData.features];
    updatedFeatures[index] = {
      ...updatedFeatures[index],
      [name]: value,
    };
    setFeatureAdditionData((prevData) => ({
      ...prevData,
      features: updatedFeatures,
    }));
  };

  const addFeatureRow = () => {
    setFeatureAdditionData((prevData) => ({
      ...prevData,
      features: [...prevData.features, { 
        labels: "", 
        "default-values": "",
        "source-base-path": "",
        "source-data-column": "",
        "storage-provider": "",
        "string-length": "",
        "vector-length": ""
      }],
    }));
  };

  const removeFeatureRow = (index) => {
    const updatedFeatures = [...FeatureAdditionData.features];
    updatedFeatures.splice(index, 1);
    setFeatureAdditionData((prevData) => ({
      ...prevData,
      features: updatedFeatures,
    }));
  };

  // New handlers for view modal
  const handleViewOpen = (featureAddition) => {
    setSelectedFeatureInfo({
      ...featureAddition.Payload,
      Status: featureAddition.Status,
      RejectReason: featureAddition.RejectReason
    });
    setShowViewModal(true);
  };

  const handleViewClose = () => {
    setShowViewModal(false);
    setSelectedFeatureInfo(null);
  };

  // Modified handlers for create modal
  const handleCreateOpen = () => {
    setFeatureAdditionData({
      "entity-label": "",
      "feature-group-label": "",
      features: [{ 
        labels: "", 
        "default-values": "",
        "source-base-path": "",
        "source-data-column": "",
        "storage-provider": ""
      }],
    });
    setOpen(true);
  };

  const handleCreateClose = () => {
    setOpen(false);
  };

  const handleSubmit = async () => {
    // Validate required fields
    const hasEmptyLengthFields = FeatureAdditionData.features.some(
      feature => !feature["string-length"] || !feature["vector-length"]
    );
    
    if (hasEmptyLengthFields) {
      setModalMessage("String Length and Vector Length are required for all features");
      setShowErrorModal(true);
      return;
    }
    
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/add-features`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(FeatureAdditionData),
      });

      if (response.ok) {
        const result = await response.json();
        setModalMessage(result.message);
        setShowSuccessModal(true);
      } else {
        const errorData = await response.json();
        setModalMessage(errorData.error);
        setShowErrorModal(true);
      }
    } catch (error) {
      setModalMessage('Network error. Please try again.');
      setShowErrorModal(true);
    }
  };

  const handleSuccessModalClose = () => {
    setShowSuccessModal(false);
    window.location.reload();
  };

  const handleErrorModalClose = () => {
    setShowErrorModal(false);
  };

  return (
    <div className="p-4">
      <GenericTable
        data={featuresRequests}
        onRowAction={handleViewOpen}
        loading={false}
        actionButtons={[
          {
            label: "Add Features",
            onClick: handleCreateOpen,
            variant: "contained",
            color: "#522b4a",
            hoverColor: "#2c3e50"
          }
        ]}
      />

      {/* Create Modal */}
      <Dialog 
        open={open} 
        onClose={handleCreateClose}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>Add Features</DialogTitle>
        <DialogContent>
          <FormControl fullWidth margin="normal">
            <InputLabel id="entity-label-id">Entity Label</InputLabel>
            <Select
              labelId="entity-label-id"
              id="entity-label"
              name="entity-label"
              value={FeatureAdditionData["entity-label"]}
              onChange={handleChange}
              label="Entity Label"
            >
              {entities.map((entity) => (
                <MenuItem key={entity} value={entity}>
                  {entity}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <FormControl fullWidth margin="normal">
            <InputLabel id="feature-group-label-id">Feature Group Label</InputLabel>
            <Select
              labelId="feature-group-label-id"
              id="feature-group-label"
              name="feature-group-label"
              value={FeatureAdditionData["feature-group-label"]}
              onChange={handleChange}
              disabled={!FeatureAdditionData["entity-label"]}
              label="Feature Group Label"
            >
              {featureGroups.map((group) => (
                <MenuItem key={group} value={group}>
                  {group}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <h5>Features</h5>
          {FeatureAdditionData.features.map((feature, index) => (
            <React.Fragment key={index}>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(4, 1fr)',
                  gap: '0px 16px'
                }}
              >
                <TextField
                  label="Label"
                  name="labels"
                  value={feature.labels}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                />
                <TextField
                  label="Default Value"
                  name="default-values"
                  value={feature["default-values"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                />
                <TextField
                  label="Source Base Path"
                  name="source-base-path"
                  value={feature["source-base-path"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                />
                <TextField
                  label="Source Data Column"
                  name="source-data-column"
                  value={feature["source-data-column"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                />
                <FormControl margin="normal" fullWidth>
                  <InputLabel id={`storage-provider-label-${index}`}>Storage Provider</InputLabel>
                  <Select
                    labelId={`storage-provider-label-${index}`}
                    id={`storage-provider-${index}`}
                    name="storage-provider"
                    value={feature["storage-provider"]}
                    onChange={(e) => handleFeatureChange(index, e)}
                    label="Storage Provider"
                  >
                    <MenuItem value="PARQUET_GCS">PARQUET_GCS</MenuItem>
                    <MenuItem value="PARQUET_S3">PARQUET_S3</MenuItem>
                    <MenuItem value="PARQUET_ADLS">PARQUET_ADLS</MenuItem>
                    <MenuItem value="DELTA_GCS">DELTA_GCS</MenuItem>
                    <MenuItem value="DELTA_S3">DELTA_S3</MenuItem>
                    <MenuItem value="DELTA_ADLS">DELTA_ADLS</MenuItem>
                    <MenuItem value="TABLE">TABLE</MenuItem>
                  </Select>
                </FormControl>
                <TextField
                  label="String Length"
                  name="string-length"
                  value={feature["string-length"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                  required
                  error={feature["string-length"] === ""}
                  helperText={feature["string-length"] === "" ? "String Length is required" : ""}
                />
                <TextField
                  label="Vector Length"
                  name="vector-length"
                  value={feature["vector-length"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                  required
                  error={feature["vector-length"] === ""}
                  helperText={feature["vector-length"] === "" ? "Vector Length is required" : ""}
                />
                <div style={{
                  marginTop: '16px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '0 14px'
                  }}
                >
                  <Button
                    onClick={() => removeFeatureRow(index)}
                    startIcon={<RemoveCircleOutlineIcon />}
                    color='error'
                    sx={{
                      color: '#d32f2f',
                      '&:hover': {
                        backgroundColor: 'transparent'
                      }
                    }}
                  >
                    Remove
                  </Button>
                </div>
              </div>
              {index < FeatureAdditionData.features.length - 1 && (
                <Divider sx={{ my: 2, borderColor: '#522b4a' }} />
              )}
            </React.Fragment>
          ))}
          <Button
            startIcon={<AddCircleOutlineIcon />}
            onClick={addFeatureRow}
            style={{ marginTop: '10px', color: 'green'}}
            sx={{
              '&:hover': {
                backgroundColor: '#fff',
              },
            }}
          >
            Add Row
          </Button>
        </DialogContent>
        <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
          <Button
            variant="outlined"
            onClick={handleCreateClose}
            sx={{
              textTransform: 'none',
              borderColor: '#522b4a',
              color: '#522b4a',
              '&:hover': {
                borderColor: '#613a5c',
                backgroundColor: 'rgba(61, 86, 114, 0.04)',
              },
            }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            sx={{
              textTransform: 'none',
              backgroundColor: '#522b4a',
              '&:hover': {
                backgroundColor: '#613a5c',
              },
            }}
          >
            Submit
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Modal using MUI Dialog instead of React Bootstrap */}
      <Dialog 
        open={showViewModal} 
        onClose={handleViewClose}
        fullWidth
        maxWidth="lg"
        PaperProps={{
          style: {
            minWidth: '90%',
          },
        }}
      >
        <DialogTitle>
          Feature Details
          <IconButton
            aria-label="close"
            onClick={handleViewClose}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              color: (theme) => theme.palette.grey[500],
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider sx={{ borderColor: '#522b4a' }} />
        <DialogContent>
          {selectedFeatureInfo && (
            <Box>
              <Box sx={{ mb: 2, mt: 1 }}>
                <Typography variant="body1">
                  <strong>Entity Label:</strong> {selectedFeatureInfo["entity-label"]}
                </Typography>
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="body1">
                  <strong>Feature Group Label:</strong> {selectedFeatureInfo["feature-group-label"]}
                </Typography>
              </Box>
              <Box>
                <Typography variant="body1" sx={{ mb: 1 }}>
                  <strong>Features:</strong>
                </Typography>
                <TableContainer component={Paper}>
                  <MuiTable size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={headerCellStyle}><strong>Label</strong></TableCell>
                        <TableCell sx={headerCellStyle}><strong>Default Value</strong></TableCell>
                        <TableCell sx={headerCellStyle}><strong>Source Base Path</strong></TableCell>
                        <TableCell sx={headerCellStyle}><strong>Source Data Column</strong></TableCell>
                        <TableCell sx={headerCellStyle} style={{ borderRight: 'none' }}><strong>Storage Provider</strong></TableCell>
                        <TableCell sx={headerCellStyle}><strong>String Length</strong></TableCell>
                        <TableCell sx={headerCellStyle}><strong>Vector Length</strong></TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {selectedFeatureInfo.features.map((feature, index) => (
                        <TableRow key={index}>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature.labels}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["default-values"]}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["source-base-path"] || "-"}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["source-data-column"] || "-"}</TableCell>
                          <TableCell sx={{...cellStyle, borderRight: 'none'}} style={cellStyle}>{feature["storage-provider"] || "-"}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["string-length"] || "-"}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["vector-length"] || "-"}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </MuiTable>
                </TableContainer>
              </Box>
              {selectedFeatureInfo.Status === "REJECTED" && (
                <Box sx={{ mb: 2, mt: 3 }}>
                  <Typography variant="body1">
                    <strong>Reject Reason:</strong> {selectedFeatureInfo.RejectReason}
                  </Typography>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
      </Dialog>

      {/* Success Modal */}
      <Dialog
        open={showSuccessModal}
        onClose={handleSuccessModalClose}
        maxWidth="sm"
      >
        <DialogTitle>
          Success
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CheckCircleOutlineIcon sx={{ color: 'green' }} />
            <Typography>
              {modalMessage}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={handleSuccessModalClose}
            sx={{
              backgroundColor: '#522b4a',
              color: 'white',
              '&:hover': {
                backgroundColor: '#613a5c',
              },
            }}
          >
            OK
          </Button>
        </DialogActions>
      </Dialog>

      {/* Error Modal */}
      <Dialog
        open={showErrorModal}
        onClose={handleErrorModalClose}
        maxWidth="sm"
      >
        <DialogTitle>
          Error
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ErrorOutlineIcon sx={{ color: 'red' }} />
            <Typography>
              {modalMessage}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={handleErrorModalClose}
            sx={{
              backgroundColor: '#522b4a',
              color: 'white',
              '&:hover': {
                backgroundColor: '#613a5c',
              },
            }}
          >
            OK
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
};

export default FeatureAddition;