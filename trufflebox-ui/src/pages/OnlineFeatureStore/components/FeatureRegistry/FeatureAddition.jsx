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
  Autocomplete,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CheckBoxIcon from '@mui/icons-material/CheckBox';
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank';
import "./styles.scss";
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';
import { removeDataTypePrefix } from '../../../../constants/dataTypes';
import { Search } from '@mui/icons-material';

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

  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

  // Delete features state variables
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteValidationErrors, setDeleteValidationErrors] = useState({});
  const [availableFeatures, setAvailableFeatures] = useState([]);
  const [deleteFeatureGroups, setDeleteFeatureGroups] = useState([]);
  const [featureSearchTerm, setFeatureSearchTerm] = useState('');
  const [deleteFeatureData, setDeleteFeatureData] = useState({
    "entity-label": "",
    "feature-group-label": "",
    "feature-labels": []
  });

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

  const fetchDeleteFeatureGroups = useCallback(async (entityLabel) => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/retrieve-feature-groups?entity=${entityLabel}`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setDeleteFeatureGroups(data);
    } catch (error) {
      console.error('Error fetching delete feature groups:', error);
      setDeleteFeatureGroups([]);
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

  const deleteEntityLabel = deleteFeatureData["entity-label"];
  useEffect(() => {
    if (deleteEntityLabel) {
      fetchDeleteFeatureGroups(deleteEntityLabel);
      // Reset feature group and features when entity changes
      setDeleteFeatureData(prev => ({
        ...prev,
        "feature-group-label": "",
        "feature-labels": []
      }));
      setAvailableFeatures([]);
    }
  }, [deleteEntityLabel, fetchDeleteFeatureGroups]);

  useEffect(() => {
    const selectedDeleteFeatureGroup = deleteFeatureGroups?.find(
      group => group?.['feature-group-label'] === deleteFeatureData["feature-group-label"]
    );
    
    if (selectedDeleteFeatureGroup?.features) {
      const activeVersion = selectedDeleteFeatureGroup['active-version'];
      const versionNumber = parseInt(activeVersion, 10);
      const featureData = selectedDeleteFeatureGroup.features[versionNumber];
      
      if (featureData?.labels) {
        const labels = featureData.labels.split(',').map(label => label.trim());
        setAvailableFeatures(labels);
      } else {
        setAvailableFeatures([]);
      }
    } else {
      setAvailableFeatures([]);
    }
  }, [deleteFeatureData["feature-group-label"], deleteFeatureGroups]);

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
      [name]: typeof value === 'string' ? value.trim() : value,
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

  // Delete features handlers
  const handleDeleteOpen = () => {
    setDeleteFeatureData({
      "entity-label": "",
      "feature-group-label": "",
      "feature-labels": []
    });
    setDeleteValidationErrors({});
    setAvailableFeatures([]);
    setDeleteFeatureGroups([]);
    setFeatureSearchTerm('');
    setShowDeleteModal(true);
  };

  const handleDeleteClose = () => {
    setShowDeleteModal(false);
  };

  const handleDeleteChange = (e) => {
    const { name, value } = e.target;
    setDeleteFeatureData((prevData) => ({
      ...prevData,
      [name]: value,
      ...(name === "entity-label" && { "feature-group-label": "", "feature-labels": [] }),
      ...(name === "feature-group-label" && { "feature-labels": [] }),
    }));
    
    // Clear search term when entity or feature group changes
    if (name === "entity-label" || name === "feature-group-label") {
      setFeatureSearchTerm('');
    }
  };

  const handleFeatureLabelChange = (e) => {
    const { value } = e.target;
    setDeleteFeatureData((prevData) => ({
      ...prevData,
      "feature-labels": typeof value === 'string' ? value.split(',').map(label => label.trim()) : value,
    }));
  };



  const validateDeleteForm = () => {
    const errors = {};
    
    if (!deleteFeatureData["entity-label"]) {
      errors["entity-label"] = "Entity Label is required";
    }

    if (!deleteFeatureData["feature-group-label"]) {
      errors["feature-group-label"] = "Feature Group Label is required";
    }

    if (deleteFeatureData["feature-labels"].length === 0) {
      errors["feature-labels"] = "At least one feature must be selected for deletion";
    }
    
    setDeleteValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleDeleteSubmit = async () => {
    setDeleteValidationErrors({});
    
    if (!validateDeleteForm()) {
      setModalMessage('Please fill in all required fields correctly.');
      setShowErrorModal(true);
      return;
    }

    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/delete-features`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(deleteFeatureData),
      });

      if (response.ok) {
        const result = await response.json();
        setModalMessage(result.message);
        setShowSuccessModal(true);
        setShowDeleteModal(false);
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
            sx: {
              textTransform: 'none',
              fontWeight: 'bold',
              padding: '8px 16px',
              borderRadius: '8px',
              backgroundColor: '#522b4a',
              boxShadow: '0 2px 4px rgba(82, 43, 74, 0.2)',
              '&:hover': {
                backgroundColor: '#613a5c',
                boxShadow: '0 4px 8px rgba(82, 43, 74, 0.3)',
              }
            }
          },
          {
            label: "Delete Features",
            onClick: handleDeleteOpen,
            variant: "outlined",
            sx: {
              textTransform: 'none',
              fontWeight: 'bold',
              padding: '8px 16px',
              borderRadius: '8px',
              border: '2px solid #d32f2f',
              color: '#d32f2f',
              backgroundColor: 'transparent',
              '&:hover': {
                borderColor: '#c62828',
                backgroundColor: 'rgba(211, 47, 47, 0.04)',
                color: '#c62828',
              }
            }
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
          <Autocomplete
            options={[...(entities || [])].sort((a, b) => a.localeCompare(b))}
            value={FeatureAdditionData["entity-label"]}
            onChange={(event, newValue) => {
              const syntheticEvent = {
                target: {
                  name: 'entity-label',
                  value: newValue || ''
                }
              };
              handleChange(syntheticEvent);
            }}
            renderInput={(params) => (
              <TextField
                {...params}
                label="Entity Label *"
                name="entity-label"
                fullWidth
                margin="normal"
                error={!!validationErrors["entity-label"]}
                helperText={validationErrors["entity-label"]}
              />
            )}
          />

          <Autocomplete
            options={[...(featureGroups?.map(group => group?.label) || [])].sort((a, b) => a.localeCompare(b))}
            value={FeatureAdditionData["feature-group-label"]}
            onChange={(event, newValue) => {
              const syntheticEvent = {
                target: {
                  name: 'feature-group-label',
                  value: newValue || ''
                }
              };
              handleChange(syntheticEvent);
            }}
            disabled={!FeatureAdditionData["entity-label"]}
            renderInput={(params) => (
              <TextField
                {...params}
                label="Feature Group Label *"
                name="feature-group-label"
                fullWidth
                margin="normal"
                error={!!validationErrors["feature-group-label"]}
                helperText={validationErrors["feature-group-label"]}
              />
            )}
          />

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
                  <Autocomplete
                    options={['PARQUET_GCS', 'PARQUET_S3', 'PARQUET_ADLS', 'DELTA_GCS', 'DELTA_S3', 'DELTA_ADLS', 'TABLE']}
                    value={feature["storage-provider"]}
                    onChange={(event, newValue) => {
                      const syntheticEvent = {
                        target: {
                          name: 'storage-provider',
                          value: newValue || ''
                        }
                      };
                      handleFeatureChange(index, syntheticEvent);
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Source Type"
                        name="storage-provider"
                        margin="normal"
                        fullWidth
                        InputProps={{
                          ...params.InputProps,
                          endAdornment: (
                            <>
                              {params.InputProps.endAdornment}
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
                                  <InfoIcon style={{ color: '#522b4a', cursor: 'pointer', fontSize: '20px' }} />
                                </Tooltip>
                              </InputAdornment>
                            </>
                          )
                        }}
                      />
                    )}
                  />
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

      {/* Delete Features Modal */}
      <Dialog
        open={showDeleteModal}
        onClose={handleDeleteClose}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>Delete Features</DialogTitle>
        <DialogContent>
          <Autocomplete
            options={[...(entities || [])].sort((a, b) => a.localeCompare(b))}
            value={deleteFeatureData["entity-label"]}
            onChange={(event, newValue) => {
              const syntheticEvent = {
                target: {
                  name: 'entity-label',
                  value: newValue || ''
                }
              };
              handleDeleteChange(syntheticEvent);
            }}
            renderInput={(params) => (
              <TextField
                {...params}
                label="Entity Label *"
                name="entity-label"
                fullWidth
                margin="normal"
                error={!!deleteValidationErrors["entity-label"]}
                helperText={deleteValidationErrors["entity-label"]}
              />
            )}
          />

          <Autocomplete
            options={[...(deleteFeatureGroups?.map(group => group?.['feature-group-label']) || [])].sort((a, b) => a.localeCompare(b))}
            value={deleteFeatureData["feature-group-label"]}
            onChange={(event, newValue) => {
              const syntheticEvent = {
                target: {
                  name: 'feature-group-label',
                  value: newValue || ''
                }
              };
              handleDeleteChange(syntheticEvent);
            }}
            disabled={!deleteFeatureData["entity-label"]}
            renderInput={(params) => (
              <TextField
                {...params}
                label="Feature Group Label *"
                name="feature-group-label"
                fullWidth
                margin="normal"
                error={!!deleteValidationErrors["feature-group-label"]}
                helperText={deleteValidationErrors["feature-group-label"]}
              />
            )}
          />

          <FormControl fullWidth margin="normal" error={!!deleteValidationErrors["feature-labels"]}>
            <InputLabel id="delete-feature-labels-id">Select Features to Delete *</InputLabel>
            <Select
              labelId="delete-feature-labels-id"
              id="delete-feature-labels"
              multiple
              value={deleteFeatureData["feature-labels"]}
              onChange={handleFeatureLabelChange}
              disabled={!deleteFeatureData["feature-group-label"]}
              label="Select Features to Delete *"
              error={!!deleteValidationErrors["feature-labels"]}
              renderValue={(selected) => selected.join(', ')}
              MenuProps={{
                PaperProps: {
                  sx: { 
                    maxHeight: 450,
                    borderRadius: '8px',
                    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.12)',
                    border: '1px solid rgba(211, 47, 47, 0.1)',
                    marginTop: '4px'
                  }
                }
              }}
            >
              <MenuItem 
                disableRipple 
                disableTouchRipple
                value=""
                sx={{ 
                  position: 'sticky', 
                  top: 0, 
                  backgroundColor: '#fafafa', 
                  zIndex: 1000,
                  borderBottom: '2px solid #f0f0f0',
                  padding: '12px 16px',
                  '&:hover': { backgroundColor: '#fafafa' },
                  '&:focus': { backgroundColor: '#fafafa' },
                  '&.Mui-selected': { backgroundColor: '#fafafa' },
                  cursor: 'default'
                }}
                onClick={(e) => e.stopPropagation()}
                onMouseDown={(e) => e.stopPropagation()}
              >
                <Box sx={{ width: '100%' }}>
                  <TextField
                    size="small"
                    placeholder="Type to search features..."
                    value={featureSearchTerm}
                    onChange={(e) => {
                      e.stopPropagation();
                      setFeatureSearchTerm(e.target.value.trim());
                    }}
                    onKeyDown={(e) => e.stopPropagation()}
                    onClick={(e) => e.stopPropagation()}
                    onMouseDown={(e) => e.stopPropagation()}
                    onFocus={(e) => e.stopPropagation()}
                    fullWidth
                    InputProps={{
                      startAdornment: (
                        <InputAdornment position="start">
                          <Search sx={{ color: '#d32f2f', fontSize: '20px' }} />
                        </InputAdornment>
                      ),
                    }}
                    sx={{ 
                      '& .MuiOutlinedInput-root': {
                        backgroundColor: 'white',
                        borderRadius: '8px',
                        '& fieldset': {
                          borderColor: '#e0e0e0',
                        },
                        '&:hover fieldset': {
                          borderColor: '#d32f2f',
                        },
                        '&.Mui-focused fieldset': {
                          borderColor: '#d32f2f',
                          borderWidth: '2px',
                        },
                      }
                    }}
                  />
                  {deleteFeatureData["feature-labels"].length > 0 && (
                    <Typography variant="caption" sx={{ 
                      color: '#d32f2f', 
                      fontWeight: 'bold',
                      marginTop: '4px',
                      display: 'block'
                    }}>
                      {deleteFeatureData["feature-labels"].length} feature{deleteFeatureData["feature-labels"].length !== 1 ? 's' : ''} selected
                    </Typography>
                  )}
                </Box>
              </MenuItem>
              {(() => {
                const filteredFeatures = availableFeatures?.filter(feature => 
                  feature.toLowerCase().includes(featureSearchTerm.toLowerCase())
                ) || [];
                
                if (availableFeatures?.length === 0) {
                  return (
                    <MenuItem disabled sx={{ 
                      justifyContent: 'center', 
                      padding: '24px',
                      fontStyle: 'italic',
                      color: '#9e9e9e'
                    }}>
                      <Typography variant="body2">
                        No features available for the selected feature group
                      </Typography>
                    </MenuItem>
                  );
                }
                
                if (filteredFeatures.length === 0) {
                  return (
                    <MenuItem disabled sx={{ 
                      justifyContent: 'center', 
                      padding: '24px',
                      fontStyle: 'italic'
                    }}>
                      <Typography variant="body2" sx={{ color: '#9e9e9e' }}>
                        🔍 No features found matching "{featureSearchTerm}"
                      </Typography>
                    </MenuItem>
                  );
                }
                
                return filteredFeatures.map((feature) => (
                <MenuItem 
                  key={feature} 
                  value={feature}
                  sx={{ 
                    display: 'flex', 
                    alignItems: 'center',
                    justifyContent: 'flex-start',
                    padding: '12px 16px',
                    minHeight: '56px',
                    borderBottom: '1px solid #f0f0f0',
                    transition: 'all 0.2s ease',
                    cursor: 'pointer',
                    '&:hover': {
                      backgroundColor: 'rgba(211, 47, 47, 0.06)',
                      transform: 'translateX(2px)',
                    },
                    '&.Mui-selected': {
                      backgroundColor: 'rgba(211, 47, 47, 0.12)',
                      borderLeft: '4px solid #d32f2f',
                      '&:hover': {
                        backgroundColor: 'rgba(211, 47, 47, 0.16)',
                      }
                    },
                    '&:last-child': {
                      borderBottom: 'none',
                    }
                  }}
                >
                  <Box sx={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    width: '100%',
                    gap: '12px'
                  }}>
                    {deleteFeatureData["feature-labels"].indexOf(feature) > -1 ? (
                      <CheckBoxIcon sx={{ 
                        color: '#d32f2f', 
                        fontSize: '24px',
                        flexShrink: 0
                      }} />
                    ) : (
                      <CheckBoxOutlineBlankIcon sx={{ 
                        color: '#9e9e9e', 
                        fontSize: '24px',
                        flexShrink: 0,
                        '&:hover': {
                          color: '#d32f2f'
                        }
                      }} />
                    )}
                    
                    <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                      <Typography variant="body1" sx={{ 
                        fontSize: '14px',
                        lineHeight: '1.5',
                        wordBreak: 'break-word',
                        color: deleteFeatureData["feature-labels"].indexOf(feature) > -1 ? '#d32f2f' : '#424242',
                        fontWeight: deleteFeatureData["feature-labels"].indexOf(feature) > -1 ? 600 : 400,
                        transition: 'color 0.2s ease'
                      }}>
                        {feature}
                      </Typography>
                      
                      {deleteFeatureData["feature-labels"].indexOf(feature) > -1 && (
                        <Typography variant="caption" sx={{ 
                          color: '#d32f2f',
                          fontWeight: 500,
                          fontSize: '11px',
                          display: 'block',
                          marginTop: '2px',
                          opacity: 0.8
                        }}>
                          ✓ Selected for deletion
                        </Typography>
                      )}
                    </Box>
                                     </Box>
                 </MenuItem>
                ));
              })()}
            </Select>
            {deleteValidationErrors["feature-labels"] && (
              <Typography variant="caption" color="error" sx={{ mt: 1 }}>
                {deleteValidationErrors["feature-labels"]}
              </Typography>
            )}
          </FormControl>

          {deleteFeatureData["feature-labels"].length > 0 && (
            <Box sx={{ mt: 2, p: 2, bgcolor: '#ffebee', borderRadius: 1, border: '1px solid #ffcdd2' }}>
              <Typography variant="body2" sx={{ color: '#d32f2f', fontWeight: 'bold', mb: 1 }}>
                ⚠️ Warning: You are about to delete the following features:
              </Typography>
              <Typography variant="body2" sx={{ color: '#d32f2f' }}>
                {deleteFeatureData["feature-labels"].join(', ')}
              </Typography>
              <Typography variant="body2" sx={{ color: '#d32f2f', mt: 1, fontStyle: 'italic' }}>
                This action cannot be undone.
              </Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
          <Button
            variant="outlined"
            onClick={handleDeleteClose}
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
            variant="outlined"
            onClick={handleDeleteSubmit}
            sx={{
              textTransform: 'none',
              borderColor: '#d32f2f',
              color: '#d32f2f',
              fontWeight: 'bold',
              borderWidth: '2px',
              '&:hover': {
                borderColor: '#c62828',
                backgroundColor: 'rgba(211, 47, 47, 0.04)',
                borderWidth: '2px',
              },
              '&:disabled': {
                borderColor: '#ccc',
                color: '#ccc',
              },
            }}
            disabled={deleteFeatureData["feature-labels"].length === 0}
          >
            Delete Features
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
};

export default FeatureAddition;