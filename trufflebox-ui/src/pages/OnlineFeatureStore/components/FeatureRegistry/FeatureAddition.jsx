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
  InputAdornment,
  Tooltip,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import "./styles.scss";
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';
import { removeDataTypePrefix } from '../../../../constants/dataTypes';

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
      "storage-provider": "",
      "string-length": "0",
      "vector-length": "0"
    }],
  });

  // Add these state variables
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

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
    const selectedFeatureGroup = featureGroups?.find(
      group => group?.label === FeatureAdditionData["feature-group-label"]
    );
    const dataType = selectedFeatureGroup?.["data-type"];
    const showStringLength = shouldShowField(dataType, 'string');
    const showVectorLength = shouldShowField(dataType, 'vector');
    
    setFeatureAdditionData((prevData) => ({
      ...prevData,
      features: [...prevData.features, {
        labels: "",
        "default-values": "",
        "source-base-path": "",
        "source-data-column": "",
        "storage-provider": "",
        "string-length": showStringLength ? "" : "0",
        "vector-length": showVectorLength ? "" : "0"
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
        "storage-provider": "",
        "string-length": "0",
        "vector-length": "0"
      }],
    });
    setValidationErrors({});
    setOpen(true);
  };

  const handleCreateClose = () => {
    setOpen(false);
  };

  // Helper function to determine which length fields to show
  const shouldShowField = (dataType, fieldType) => {
    if (!dataType) return false;

    try {
      const cleanDataType = removeDataTypePrefix(dataType);
      if (!cleanDataType) return false;

      const lowerDataType = cleanDataType.toLowerCase();

      if (fieldType === 'string') {
        return lowerDataType.includes('string');
      }
      if (fieldType === 'vector') {
        return lowerDataType.includes('vector');
      }
      return false;
    } catch (error) {
      return false;
    }
  };

  const validateForm = () => {
    const errors = {};
    
    if (!FeatureAdditionData["entity-label"]) {
      errors["entity-label"] = "Entity Label is required";
    }

    if (!FeatureAdditionData["feature-group-label"]) {
      errors["feature-group-label"] = "Feature Group Label is required";
    }
    
    const selectedFeatureGroup = featureGroups?.find(
      group => group?.label === FeatureAdditionData["feature-group-label"]
    );
    const dataType = selectedFeatureGroup?.["data-type"];
    const showStringLength = shouldShowField(dataType, 'string');
    const showVectorLength = shouldShowField(dataType, 'vector');
    
    FeatureAdditionData.features.forEach((feature, index) => {
      if (!feature.labels || feature.labels.trim() === "") {
        errors[`features.${index}.labels`] = "Label is required";
      }
      
      if (!feature["default-values"] || feature["default-values"].trim() === "") {
        errors[`features.${index}.default-values`] = "Default Value is required";
      }
      
      if (showStringLength && (!feature["string-length"] || parseFloat(feature["string-length"]) <= 0)) {
        errors[`features.${index}.string-length`] = "String Length must be greater than 0";
      }
      
      if (showVectorLength && (!feature["vector-length"] || parseFloat(feature["vector-length"]) <= 0)) {
        errors[`features.${index}.vector-length`] = "Vector Length must be greater than 0";
      }
    });
    
    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    setValidationErrors({});
    
    if (!validateForm()) {
      setModalMessage('Please fill in all required fields correctly.');
      setShowErrorModal(true);
      return;
    }
    
    const selectedFeatureGroup = featureGroups.find(
      group => group?.label === FeatureAdditionData["feature-group-label"]
    );
    const dataType = selectedFeatureGroup?.["data-type"];
    const showStringLength = shouldShowField(dataType, 'string');
    const showVectorLength = shouldShowField(dataType, 'vector');

    const updatedFeatures = FeatureAdditionData.features.map(feature => ({
      ...feature,
      "string-length": showStringLength ? feature["string-length"] : "0",
      "vector-length": showVectorLength ? feature["vector-length"] : "0"
    }));

    const finalFeatureAdditionData = {
      ...FeatureAdditionData,
      features: updatedFeatures
    };

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/add-features`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(finalFeatureAdditionData),
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
            hoverColor: "#613a5c"
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
          <FormControl fullWidth margin="normal" error={!!validationErrors["entity-label"]}>
            <InputLabel id="entity-label-id">Entity Label *</InputLabel>
            <Select
              labelId="entity-label-id"
              id="entity-label"
              name="entity-label"
              value={FeatureAdditionData["entity-label"]}
              onChange={handleChange}
              label="Entity Label *"
              error={!!validationErrors["entity-label"]}
            >
              {entities?.map((entity) => (
                <MenuItem key={entity} value={entity}>
                  {entity}
                </MenuItem>
              ))}
            </Select>
            {validationErrors["entity-label"] && (
              <Typography variant="caption" color="error" sx={{ mt: 1 }}>
                {validationErrors["entity-label"]}
              </Typography>
            )}
          </FormControl>

          <FormControl fullWidth margin="normal" error={!!validationErrors["feature-group-label"]}>
            <InputLabel id="feature-group-label-id">Feature Group Label *</InputLabel>
            <Select
              labelId="feature-group-label-id"
              id="feature-group-label"
              name="feature-group-label"
              value={FeatureAdditionData["feature-group-label"]}
              onChange={handleChange}
              disabled={!FeatureAdditionData["entity-label"]}
              label="Feature Group Label *"
              error={!!validationErrors["feature-group-label"]}
            >
              {featureGroups?.map((group) => (
                <MenuItem key={group?.label} value={group?.label}>
                  {group?.label}
                </MenuItem>
              ))}
            </Select>
            {validationErrors["feature-group-label"] && (
              <Typography variant="caption" color="error" sx={{ mt: 1 }}>
                {validationErrors["feature-group-label"]}
              </Typography>
            )}
          </FormControl>

          <h5>Features</h5>
          {FeatureAdditionData?.features?.map((feature, index) => (
            <React.Fragment key={index}>
              <div>
                {/* Row 1: Label and Default Value */}
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(3, 1fr)',
                    gap: '0px 16px'
                  }}
                >
                  <TextField
                    label="Label *"
                    name="labels"
                    value={feature.labels}
                    onChange={(e) => handleFeatureChange(index, e)}
                    margin="normal"
                    error={!!validationErrors[`features.${index}.labels`]}
                    helperText={validationErrors[`features.${index}.labels`]}
                  />
                  <TextField
                    label="Default Value *"
                    name="default-values"
                    value={feature["default-values"]}
                    onChange={(e) => handleFeatureChange(index, e)}
                    margin="normal"
                    error={!!validationErrors[`features.${index}.default-values`]}
                    helperText={validationErrors[`features.${index}.default-values`]}
                  />
                </div>

                {/* Row 2: Source Type, Source Base Path, Source Data Column */}
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(3, 1fr)',
                    gap: '0px 16px'
                  }}
                >
                  <FormControl margin="normal" fullWidth>
                    <InputLabel id={`source-type-label-${index}`}>
                      Source Type
                    </InputLabel>
                    <Select
                      labelId={`source-type-label-${index}`}
                      id={`source-type-${index}`}
                      name="storage-provider"
                      value={feature["storage-provider"]}
                      onChange={(e) => handleFeatureChange(index, e)}
                      label="Source Type"
                      endAdornment={
                        <InputAdornment position="end">
                          <Tooltip
                            title="Cloud storage or table"
                            placement="bottom-end"
                            slotProps={{
                              tooltip: {
                                sx: {
                                  bgcolor: 'white',
                                  color: 'black',
                                  border: '1px solid #cccccc',
                                  boxShadow: '0px 2px 8px rgba(0, 0, 0, 0.15)',
                                  p: 1,
                                  width: '280px',
                                  maxWidth: '300px',
                                  '& p': {
                                    my: 0.5,
                                  }
                                }
                              }
                            }}
                          >
                            <InfoIcon style={{ color: '#522b4a', cursor: 'pointer', fontSize: '20px', position: 'absolute', right: '24px' }} />
                          </Tooltip>
                        </InputAdornment>
                      }
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
                    label="Source Base Path"
                    name="source-base-path"
                    value={feature["source-base-path"]}
                    onChange={(e) => handleFeatureChange(index, e)}
                    margin="normal"
                    fullWidth
                    slotProps={{
                      input: {
                        endAdornment: (
                          <InputAdornment position="end">
                            <Tooltip
                              title="Offline cloud storage path or table name source for the feature"
                              placement="bottom-end"
                              slotProps={{
                                tooltip: {
                                  sx: {
                                    bgcolor: 'white',
                                    color: 'black',
                                    border: '1px solid #cccccc',
                                    boxShadow: '0px 2px 8px rgba(0, 0, 0, 0.15)',
                                    p: 1,
                                    maxWidth: '250px',
                                  }
                                }
                              }}
                            >
                              <InfoIcon style={{ color: '#522b4a', cursor: 'pointer', fontSize: '20px' }} />
                            </Tooltip>
                          </InputAdornment>
                        )
                      }
                    }}
                  />
                  <TextField
                    label="Source Data Column"
                    name="source-data-column"
                    value={feature["source-data-column"]}
                    onChange={(e) => handleFeatureChange(index, e)}
                    margin="normal"
                    fullWidth
                    slotProps={{
                      input: {
                        endAdornment: (
                          <InputAdornment position="end">
                            <Tooltip
                              title="Name of the column in offline source"
                              placement="bottom-end"
                              slotProps={{
                                tooltip: {
                                  sx: {
                                    bgcolor: 'white',
                                    color: 'black',
                                    border: '1px solid #cccccc',
                                    boxShadow: '0px 2px 8px rgba(0, 0, 0, 0.15)',
                                    p: 1,
                                    maxWidth: '250px',
                                  }
                                }
                              }}
                            >
                              <InfoIcon style={{ color: '#522b4a', cursor: 'pointer', fontSize: '20px' }} />
                            </Tooltip>
                          </InputAdornment>
                        )
                      }
                    }}
                  />
                </div>

                {/* Row 3: String Length, Vector Length, and Remove Button */}
                {(() => {
                  const selectedFeatureGroup = featureGroups?.find(
                    group => group?.label === FeatureAdditionData["feature-group-label"]
                  );
                  const dataType = selectedFeatureGroup?.["data-type"];

                  const showStringLength = shouldShowField(dataType, 'string');
                  const showVectorLength = shouldShowField(dataType, 'vector');

                  return (
                    <div
                      style={{
                        display: 'grid',
                        gridTemplateColumns: 'repeat(3, 1fr)',
                        gap: '0px 16px'
                      }}
                    >
                      {showStringLength ? (
                        <TextField
                          label="String Length *"
                          name="string-length"
                          value={feature["string-length"]}
                          onChange={(e) => handleFeatureChange(index, e)}
                          margin="normal"
                          fullWidth
                          error={!!validationErrors[`features.${index}.string-length`]}
                          helperText={validationErrors[`features.${index}.string-length`]}
                        />
                      ) : showVectorLength ? (
                        <TextField
                          label="Vector Length *"
                          name="vector-length"
                          value={feature["vector-length"]}
                          onChange={(e) => handleFeatureChange(index, e)}
                          margin="normal"
                          fullWidth
                          error={!!validationErrors[`features.${index}.vector-length`]}
                          helperText={validationErrors[`features.${index}.vector-length`]}
                        />
                      ) : (
                        <div></div>
                      )}

                      {showStringLength && showVectorLength ? (
                        <TextField
                          label="Vector Length *"
                          name="vector-length"
                          value={feature["vector-length"]}
                          onChange={(e) => handleFeatureChange(index, e)}
                          margin="normal"
                          fullWidth
                          error={!!validationErrors[`features.${index}.vector-length`]}
                          helperText={validationErrors[`features.${index}.vector-length`]}
                        />
                      ) : (
                        <div></div>
                      )}

                      <div style={{
                        marginTop: '16px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '0 14px'
                      }}>
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
                  );
                })()}
              </div>
              {index < FeatureAdditionData.features.length - 1 && (
                <Divider sx={{ my: 2, borderColor: '#522b4a' }} />
              )}
            </React.Fragment>
          ))}
          <Button
            startIcon={<AddCircleOutlineIcon />}
            onClick={addFeatureRow}
            style={{ marginTop: '10px', color: 'green' }}
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
                      {selectedFeatureInfo?.features?.map((feature, index) => (
                        <TableRow key={index}>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature.labels}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["default-values"]}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["source-base-path"] || "-"}</TableCell>
                          <TableCell sx={cellStyle} style={cellStyle}>{feature["source-data-column"] || "-"}</TableCell>
                          <TableCell sx={{ ...cellStyle, borderRight: 'none' }} style={cellStyle}>{feature["storage-provider"] || "-"}</TableCell>
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