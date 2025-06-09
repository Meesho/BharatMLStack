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
  Tooltip,
  Divider,
  InputAdornment,
  Box,
  Typography
} from '@mui/material';
import { Modal, ListGroup, Table } from 'react-bootstrap';
import "./styles.scss";
import GenericTable from '../../common/GenericTable';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import InfoIcon from '@mui/icons-material/Info';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { useAuth } from '../../../Auth/AuthContext';
import { dataTypes, addDataTypePrefix, removeDataTypePrefix } from '../../../../constants/dataTypes';

import * as URL_CONSTANTS from '../../../../config';

const FeatureGroupRegistry = () => {
  const [open, setOpen] = useState(false);
  const [viewMode] = useState(false);
  const [entities, setEntities] = useState([]);
  const [stores, setStores] = useState({});
  const [jobs, setJobs] = useState([]);
  const [selectedStore, setSelectedStore] = useState(null);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedFeatureGroup, setSelectedFeatureGroup] = useState(null);
  const [requestType, setRequestType] = useState("");
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');

  
  const [featureGroupData, setFeatureGroupData] = useState({
    "entity-label": "",
    "fg-label": "",
    "job-id": "",
    "store-id": 0,
    "ttl-in-seconds": 0,
    "in-memory-cache-enabled": true,
    "distributed-cache-enabled": true,
    "data-type": "",
    "layout-version": 1,
    features: [{ 
      labels: "", 
      "default-values": "", 
      "source-base-path": "", 
      "source-data-column": "", 
      "storage-provider": "", 
      "string-length": "", 
      "vector-length": "" 
    }],
  });
  
  const { user } = useAuth();
  const [featureGroupRequests, setFeatureGroupRequests] = useState([]);

  const cellStyle = {
    maxWidth: "150px", 
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "normal",
    wordBreak: "break-word"
  };

  const fetchStores = async () => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-store`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setStores(data);
    } catch (error) {
      console.error('Error fetching stores:', error);
    }
  };

  const fetchEntities = async () => {
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
  };

  const fetchJobs = useCallback(async () => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-jobs?jobType=writer`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setJobs(data);
    } catch (error) {
      console.error('Error fetching jobs:', error);
    }
  }, [user.token]);

  // Fetch jobs once on component load
  useEffect(() => {
    fetchJobs();
  }, [fetchJobs]); 

  useEffect(() => {
    const fetchFeatureGroupRequests = async () => {
      try {
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-feature-group-requests`, {
          headers: {
            'Authorization': `Bearer ${user.token}`,
          },
        });
        const data = await response.json();
        setFeatureGroupRequests(data);
      } catch (error) {
        console.error('Error fetching feature group requests:', error);
      }
    };

    fetchFeatureGroupRequests();
  }, [user.token]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    
    if (name === "store-id") {
      setSelectedStore(stores[value]);
    }
    
    setFeatureGroupData((prevData) => ({
      ...prevData,
      [name]: name === "store-id" || name === "ttl-in-seconds" 
        ? parseInt(value, 10) 
        : name === "distributed-cache-enabled" || name === "in-memory-cache-enabled"
        ? value === 'true'
        : name === "data-type"
        ? addDataTypePrefix(value) 
        : value,
    }));
  };

  // Update feature change handler to handle string and vector length
  const handleFeatureChange = (index, e) => {
    const { name, value } = e.target;
    const updatedFeatures = [...featureGroupData.features];
    updatedFeatures[index] = {
      ...updatedFeatures[index],
      [name]: value,
    };
    setFeatureGroupData((prevData) => ({
      ...prevData,
      features: updatedFeatures,
    }));
  };

  const addFeatureRow = () => {
    // Copy the string and vector length values from the last feature values
    const lastFeatureValue = featureGroupData.features[featureGroupData.features.length - 1];
    setFeatureGroupData((prevData) => ({
      ...prevData,
      features: [...prevData.features, { 
        labels: "", 
        "default-values": "", 
        "source-base-path": "", 
        "source-data-column": "", 
        "storage-provider": "",
        "string-length": lastFeatureValue["string-length"] || "",
        "vector-length": lastFeatureValue["vector-length"] || ""
      }],
    }));
  };

  const removeFeatureRow = (index) => {
    const updatedFeatures = [...featureGroupData.features];
    updatedFeatures.splice(index, 1);
    setFeatureGroupData((prevData) => ({
      ...prevData,
      features: updatedFeatures,
    }));
  };

  const handleOpen = async (featureGroup = null) => {
    if (featureGroup) {
      setSelectedFeatureGroup({
        ...featureGroup.Payload,
        Status: featureGroup.Status,
        RejectReason: featureGroup.RejectReason
      });
      setRequestType(featureGroup.RequestType || "");
      setShowViewModal(true);
    } else {
      await Promise.all([fetchStores(), fetchEntities()]);
      setFeatureGroupData({
        "entity-label": "",
        "fg-label": "",
        "job-id": "",
        "store-id": 0,
        "ttl-in-seconds": 0,
        "in-memory-cache-enabled": true,
        "distributed-cache-enabled": true,
        "data-type": "",
        "layout-version": 1,
        features: [{ 
          labels: "", 
          "default-values": "", 
          "source-base-path": "", 
          "source-data-column": "", 
          "storage-provider": "", 
          "string-length": "", 
          "vector-length": "" 
        }],
      });
      setOpen(true); // Open form modal
    }
  };  

  const handleClose = () => {
    setOpen(false);
    setSelectedStore(null);
  };

  const handleSubmit = async () => {
    // Check if any feature has empty string-length or vector-length
    const hasEmptyLengthFields = featureGroupData.features.some(
      feature => !feature["string-length"] || !feature["vector-length"]
    );
    
    if (hasEmptyLengthFields) {
      setModalMessage("String Length and Vector Length are required for all features");
      setShowErrorModal(true);
      return; // Prevent form submission
    }

    const apiEndpoint = `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/register-feature-group`;
    try {
      const response = await fetch(apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(featureGroupData),
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

  const renderFeatureGroupModal = () => (
    <Dialog open={open} onClose={handleClose} maxWidth="md">
      <DialogTitle>{viewMode ? 'Feature Group' : 'Create Feature Group'}</DialogTitle>
      <DialogContent>
        <div style={{margin: '1.5rem'}}>
          <FormControl fullWidth margin="normal">
            <InputLabel id="entity-label">Entity Label</InputLabel>
            <Select
              labelId="entity-label"
              name="entity-label"
              value={featureGroupData["entity-label"]}
              onChange={handleChange}
              fullWidth
              disabled={viewMode}
              label="Entity Label"
            >
              {entities.map((entity) => (
                <MenuItem key={entity} value={entity}>
                  {entity}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <TextField
            label="Feature Group Label"
            name="fg-label"
            value={featureGroupData["fg-label"]}
            onChange={handleChange}
            fullWidth
            margin="normal"
            disabled={viewMode}
          />

          <TextField
            label="Layout Version"
            name="layout-version"
            value={featureGroupData["layout-version"]}
            fullWidth
            margin="normal"
            disabled={true}
          />

          <FormControl fullWidth margin="normal">
            <InputLabel id="job-name-label">Job Name</InputLabel>
            <Select
              labelId="job-name-label"
              name="job-id"
              value={featureGroupData["job-id"]}
              onChange={handleChange}
              fullWidth
              disabled={viewMode}
              label="Job Name"
            >
              {jobs.map((job) => (
                <MenuItem key={job} value={job}>
                  {job}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <FormControl fullWidth margin="normal">
            <InputLabel id="store-id-label">Store ID</InputLabel>
            <Select
              labelId="store-id-label"
              name="store-id"
              value={featureGroupData["store-id"]}
              onChange={handleChange}
              fullWidth
              disabled={viewMode}
              label="Store ID"
              endAdornment={
                selectedStore && (
                  <InputAdornment position="end">
                    <Tooltip
                      title={
                        <div>
                          <p>DB Type: {selectedStore["db-type"]}</p>
                          <p>Conf ID: {selectedStore["conf-id"]}</p>
                          <p>Table: {selectedStore["table"]}</p>
                          <p>Table TTL(in Seconds): {selectedStore["table-ttl"]}</p>
                          <p>Primary keys: {selectedStore["primary-keys"]}</p>
                        </div>
                      }
                      placement="bottom-end"
                      componentsProps={{
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
                      <InfoIcon style={{ color: '#522b4a', cursor: 'pointer' }} />
                    </Tooltip>
                  </InputAdornment>
                )
              }
            >
              {Object.keys(stores).map((storeId) => (
                <MenuItem key={storeId} value={parseInt(storeId)}>
                  {storeId}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <TextField
            label="TTL in Seconds"
            name="ttl-in-seconds"
            type="number"
            value={featureGroupData["ttl-in-seconds"]}
            onChange={handleChange}
            fullWidth
            margin="normal"
            disabled={viewMode}
          />

          <FormControl fullWidth margin="normal">
            <InputLabel id="distributed-cache-enabled-label">Distributed Caching Enabled</InputLabel>
            <Select
              labelId="distributed-cache-enabled-label"
              name="distributed-cache-enabled"
              value={featureGroupData["distributed-cache-enabled"].toString()}
              onChange={handleChange}
              fullWidth
              disabled={viewMode}
              label="Distributed Caching Enabled"
            >
              <MenuItem value="true">True</MenuItem>
              <MenuItem value="false">False</MenuItem>
            </Select>
          </FormControl>

          <FormControl fullWidth margin="normal">
            <InputLabel id="in-memory-cache-enabled-label">In Memory Caching Enabled</InputLabel>
            <Select
              labelId="in-memory-cache-enabled-label"
              name="in-memory-cache-enabled"
              value={featureGroupData["in-memory-cache-enabled"].toString()}
              onChange={handleChange}
              fullWidth
              disabled={viewMode}
              label="In Memory Caching Enabled"
            >
              <MenuItem value="true">True</MenuItem>
              <MenuItem value="false">False</MenuItem>
            </Select>
          </FormControl>

          <FormControl fullWidth margin="normal">
            <InputLabel id="data-type-label">Data Type</InputLabel>
            <Select
              labelId="data-type-label"
              name="data-type"
              value={removeDataTypePrefix(featureGroupData["data-type"])}
              onChange={handleChange}
              fullWidth
              disabled={viewMode}
              label="Data Type"
            >
              {dataTypes.map((type) => (
                <MenuItem key={type} value={type}>
                  {type}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <h5>Features</h5>
          {featureGroupData.features.map((feature, index) => (
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
                  disabled={viewMode}
                  fullWidth
                />
                <TextField
                  label="Default Value"
                  name="default-values"
                  value={feature["default-values"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                  disabled={viewMode}
                />
                <TextField
                  label="Source Base Path"
                  name="source-base-path"
                  value={feature["source-base-path"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                  disabled={viewMode}
                />
                <TextField
                  label="Source Data Column"
                  name="source-data-column"
                  value={feature["source-data-column"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                  disabled={viewMode}
                />
                <FormControl margin="normal" fullWidth>
                  <InputLabel>Storage Provider</InputLabel>
                  <Select
                    name="storage-provider"
                    value={feature["storage-provider"]}
                    onChange={(e) => handleFeatureChange(index, e)}
                    label="Storage Provider"
                    disabled={viewMode}
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
                  disabled={viewMode}
                  required
                />
                <TextField
                  label="Vector Length"
                  name="vector-length"
                  value={feature["vector-length"]}
                  onChange={(e) => handleFeatureChange(index, e)}
                  margin="normal"
                  fullWidth
                  disabled={viewMode}
                  required
                />
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
                  // disabled={featureGroupData.features.length <= 1}
                >
                Remove
                </Button>
                </div>
              </div>
              {index < featureGroupData.features.length - 1 && (
                <Divider sx={{ my: 2, borderColor: '#522b4a' }} />
              )}
            </React.Fragment>
          ))}
          {!viewMode && (
            <Button
            startIcon={<AddCircleOutlineIcon />}
            onClick={addFeatureRow}
            sx={{ marginTop: 2,
              '&:hover': {
                backgroundColor: '#fff',
              },
             }}
            style={{
              marginTop: '10px',
              color: 'green',
            }}
            >
              Add Feature Row
          </Button>
          )}
        </div>
      </DialogContent>
      {!viewMode && (
        <DialogActions sx={{ padding: '16px 24px', gap: '12px' }}>
                  <Button
                    variant="outlined"
                    onClick={handleClose}
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
        )}
        </Dialog>
      );
      const FeatureGroupDetailsModal = () => (
        <Modal show={showViewModal} onHide={() => setShowViewModal(false)} size="xl" centered>
          <Modal.Header closeButton>
            <Modal.Title>{requestType === "EDIT" ? "Edited Feature Group Request" : "Feature Group Details"}</Modal.Title>
          </Modal.Header>
          <Modal.Body>
            {selectedFeatureGroup && (
              <ListGroup>
                {requestType && (
                  <ListGroup.Item><strong>Request Type:</strong> {requestType}</ListGroup.Item>
                )}
                <ListGroup.Item><strong>Entity Label:</strong> {selectedFeatureGroup["entity-label"]}</ListGroup.Item>
                <ListGroup.Item><strong>Feature Group Label:</strong> {selectedFeatureGroup["fg-label"]}</ListGroup.Item>
                
                {requestType !== "EDIT" && (
                  <>
                    <ListGroup.Item><strong>Job Name:</strong> {selectedFeatureGroup["job-id"]}</ListGroup.Item>
                    <ListGroup.Item><strong>Store ID:</strong> {selectedFeatureGroup["store-id"]}</ListGroup.Item>
                  </>
                )}
                
                <ListGroup.Item><strong>TTL (Seconds):</strong> {selectedFeatureGroup["ttl-in-seconds"]}</ListGroup.Item>
                <ListGroup.Item><strong>In-Memory Cache Enabled:</strong> {selectedFeatureGroup["in-memory-cache-enabled"].toString()}</ListGroup.Item>
                <ListGroup.Item><strong>Distributed Cache Enabled:</strong> {selectedFeatureGroup["distributed-cache-enabled"].toString()}</ListGroup.Item>
                
                {requestType !== "EDIT" && (
                  <ListGroup.Item><strong>Data Type:</strong> {selectedFeatureGroup["data-type"]}</ListGroup.Item>
                )}
                
                {selectedFeatureGroup["layout-version"] !== undefined && (
                  <ListGroup.Item><strong>Layout Version:</strong> {selectedFeatureGroup["layout-version"]}</ListGroup.Item>
                )}
                
                {requestType !== "EDIT" && selectedFeatureGroup.features && (
                  <ListGroup.Item>
                    <strong>Features:</strong>
                    <Table bordered size="sm" className="mt-2">
                      <thead>
                        <tr>
                          <th>Label</th>
                          <th>Default Value</th>
                          <th>Source Base Path</th>
                          <th>Source Data Column</th>
                          <th>Storage Provider</th>
                          <th>String Length</th>
                          <th>Vector Length</th>
                        </tr>
                      </thead>
                      <tbody>
                        {selectedFeatureGroup.features.map((feature, index) => (
                          <tr key={index}>
                            <td style={cellStyle}>{feature.labels}</td>
                            <td style={cellStyle}>{feature["default-values"]}</td>
                            <td style={cellStyle}>{feature["source-base-path"]}</td>
                            <td style={cellStyle}>{feature["source-data-column"]}</td>
                            <td style={cellStyle}>{feature["storage-provider"]}</td>
                            <td style={cellStyle}>{feature["string-length"]}</td>
                            <td style={cellStyle}>{feature["vector-length"]}</td>
                          </tr>
                        ))}
                      </tbody>
                    </Table>
                  </ListGroup.Item>
                )}
                {selectedFeatureGroup.Status === "REJECTED" && (
                  <ListGroup.Item>
                    <strong>Reject Reason:</strong> {selectedFeatureGroup.RejectReason}
                  </ListGroup.Item>
                )}
              </ListGroup>
            )}
          </Modal.Body>
        </Modal>
      );
      
    
      return (
        <div style={{ padding: '20px' }}>
          {renderFeatureGroupModal()}
          <FeatureGroupDetailsModal />
          
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
          
          <GenericTable
            data={featureGroupRequests}
            onRowAction={handleOpen}
            loading={false}
            actionButtons={[
              {
                label: "Create Feature Group",
                onClick: () => handleOpen(),
                variant: "contained",
                color: "#522b4a",
                hoverColor: "#2c3e50"
              }
            ]}
          />
        </div>
      );
    };

export default FeatureGroupRegistry;
