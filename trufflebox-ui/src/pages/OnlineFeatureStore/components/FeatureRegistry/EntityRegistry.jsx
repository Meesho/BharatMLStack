import React, { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Grid,
  Box,
  Divider,
  Tooltip,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Typography,
  CircularProgress,
} from '@mui/material';
import { Modal, ListGroup, Table } from 'react-bootstrap';
import { useForm } from 'react-cool-form';
import './styles.scss';
import GenericTable from '../../common/GenericTable';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';

const EntityRegistry = () => {
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState(null);
  const [entityData, setEntityData] = useState(getInitialEntityData());
  const [entityRequests, setEntityRequests] = useState([]);
  const [isEditRequest, setIsEditRequest] = useState(false);
  const { user } = useAuth();
  
  // New states for success/error modals and loading
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = (e) => {
    const { name, value } = e.target;
    updateNestedState(setEntityData, name, value);
  };

  const handleKeyMapChange = (index, e) => {
    const { name, value } = e.target;
    setEntityData((prevData) => {
      const updatedKeyMap = [...prevData['key-map']];
      updatedKeyMap[index] = {
        ...updatedKeyMap[index],
        [name.split('.')[2]]: value,
      };
      return { ...prevData, 'key-map': updatedKeyMap };
    });
  };

  const modifyKeyMapRow = (index, action) => {
    setEntityData((prevData) => {
      const updatedKeyMap = [...prevData['key-map']];
      if (action === 'add') {
        updatedKeyMap.push({ sequence: '', 'entity-label': '', 'column-label': '' });
      } else if (action === 'remove') {
        updatedKeyMap.splice(index, 1);
      }
      return { ...prevData, 'key-map': updatedKeyMap };
    });
  };

  const handleOpen = () => {
    setEntityData(getInitialEntityData());
    setOpen(true);
  };

  const handleViewEntity = (entity) => {
    // Check if this is an edit request, use "CREATE" as default if RequestType is not specified
    const requestType = entity.RequestType || "CREATE";
    const isEdit = requestType === "EDIT";
    setIsEditRequest(isEdit);
    
    // Parse and transform the entity data
    const parsedPayload = JSON.parse(entity.Payload);
    
    // Add RequestType to the transformed data
    const transformedData = transformEntityData(parsedPayload, isEdit);
    transformedData.requestType = requestType;
    transformedData.status = entity.Status || "";
    transformedData.rejectReason = entity.RejectReason || "";
    
    setSelectedEntity(transformedData);
    setShowViewModal(true);
  };

  const closeViewModal = () => {
    setShowViewModal(false);
    setSelectedEntity(null);
  };

  const transformEntityData = (data, isEdit) => {
    if (isEdit) {
      return {
        "entity-label": data["entity-label"] || "",
        "distributed-cache": {
          enabled: data["distributed-cache"]?.enabled || "true",
          "ttl-in-seconds": data["distributed-cache"]["ttl-in-seconds"],
          "jitter-percentage": data["distributed-cache"]["jitter-percentage"],
          "conf-id": parseInt(data["distributed-cache"]["conf-id"] || "0", 10),
        },
        "in-memory-cache": {
          enabled: data["in-memory-cache"]?.enabled || "false",
          "ttl-in-seconds": data["in-memory-cache"]["ttl-in-seconds"],
          "jitter-percentage": data["in-memory-cache"]["jitter-percentage"],
          "conf-id": parseInt(data["in-memory-cache"]["conf-id"] || "0", 10),
        },
      };
    }
    
    return {
      "entity-label": data["entity-label"] || "",
      "key-map": Object.values(data["key-map"] || {}).map((item) => ({
        sequence: item.sequence || "",
        "entity-label": item["entity-label"] || "",
        "column-label": item["column-label"] || "",
      })),
      "distributed-cache": {
        enabled: data["distributed-cache"]?.enabled || "true",
        "ttl-in-seconds": data["distributed-cache"]["ttl-in-seconds"],
        "jitter-percentage": data["distributed-cache"]["jitter-percentage"],
        "conf-id": parseInt(data["distributed-cache"]["conf-id"] || "0", 10),
      },
      "in-memory-cache": {
        enabled: data["in-memory-cache"]?.enabled || "false",
        "ttl-in-seconds": data["in-memory-cache"]["ttl-in-seconds"],
        "jitter-percentage": data["in-memory-cache"]["jitter-percentage"],
        "conf-id": parseInt(data["in-memory-cache"]["conf-id"] || "0", 10),
      },
    };
  };

  const handleClose = () => setOpen(false);

  const { form, reset } = useForm({ defaultValues: getInitialEntityData() });

  const handleSubmit = async (event) => {
    event.preventDefault();
    setIsSubmitting(true);
    try {
      const requestData = transformRequestData(entityData);
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/register-entity`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(requestData),
      });
      const result = await response.json();
      if (response.ok) {
        setModalMessage(result.message || 'Request processed successfully');
        setShowSuccessModal(true);
        reset(getInitialEntityData());
        setEntityData(getInitialEntityData());
        setOpen(false);
        // Fetch fresh data instead of page reload
        fetchEntityRequests();
      } else {
        setModalMessage(result.error || 'Error processing request');
        setShowErrorModal(true);
      }
    } catch(error) {
      console.error('Submission error:', error);
      setModalMessage(error.message || 'Network error. Please try again.');
      setShowErrorModal(true);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSuccessModalClose = () => {
    setShowSuccessModal(false);
    // Don't reload the page, just refresh data
    fetchEntityRequests();
  };

  const handleErrorModalClose = () => {
    setShowErrorModal(false);
  };

  const fetchEntityRequests = useCallback(async () => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-entity-requests`, {
        headers: {
          Authorization: `Bearer ${user.token}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setEntityRequests(data);
      } else {
        console.error('Error fetching data:', response.status);
      }
    } catch (error) {
      console.error('Error fetching data:', error);
    }
  }, [user.token]);

  useEffect(() => {
    fetchEntityRequests();
  }, [fetchEntityRequests]);

  const renderEntityModal = () => (
    <Dialog open={open} onClose={handleClose} maxWidth="lg">
      <DialogTitle>Create Entity</DialogTitle>
      <DialogContent>
        <form ref={form} onSubmit={handleSubmit} noValidate>
          <Box sx={{ marginBottom: 3 }}>
            <TextField
              label="Entity Label"
              name="entity-label"
              value={entityData['entity-label']}
              onChange={handleChange}
              fullWidth
              className="custom-textfield"
              placeholder='Enter Entity Label'
            />
          </Box>
          
          <Box sx={{ marginBottom: 3 }}>
            <h5>Keys</h5>
            {entityData['key-map'].map((keyMap, index) => (
              <Box key={index} sx={{ marginBottom: 2 }}>
                <Grid container spacing={2} alignItems="center">
                  <Grid item xs={4}>
                    <TextField
                      label="Entity Key"
                      name={`key-map.${index}.entity-label`}
                      value={keyMap['entity-label']}
                      onChange={(e) => handleKeyMapChange(index, e)}
                      fullWidth
                      className="custom-textfield"
                      placeholder="Enter Entity Key"
                    />
                  </Grid>
                  <Grid item xs={4}>
                    <TextField
                      label="Column Key"
                      name={`key-map.${index}.column-label`}
                      value={keyMap['column-label']}
                      onChange={(e) => handleKeyMapChange(index, e)}
                      fullWidth
                      className="custom-textfield"
                      placeholder="Enter Column Key"
                    />
                  </Grid>
                  <Grid item>
                    <Tooltip title="Remove Row">
                      <Button
                        onClick={() => modifyKeyMapRow(index, 'remove')}
                        color="error"
                        sx={{
                          textTransform: 'none',
                          '&:hover': {
                            backgroundColor: 'rgba(244, 67, 54, 0.04)',
                          },
                        }}
                        startIcon={<RemoveCircleOutlineIcon style={{ fontSize: '1.5rem' }} />}
                      >
                        Remove
                      </Button>
                    </Tooltip>
                  </Grid>
                </Grid>
              </Box>
            ))}
            <Button
              startIcon={<AddCircleOutlineIcon />}
              onClick={() => modifyKeyMapRow(null, 'add')}
              color="#446e9b"
              sx={{
                marginTop: 2,
                '&:hover': {
                  backgroundColor: '#fff',
                },
              }}
            >
              Add Row
            </Button>
          </Box>

          <Divider sx={{ marginY: 2 }} />
          {renderCacheSection('Distributed Cache', 'distributed-cache')}
          <Divider sx={{ marginY: 2 }} />
          {renderCacheSection('In-Memory Cache', 'in-memory-cache')}
        </form>
      </DialogContent>
      <DialogActions sx={{ padding: '0px 24px 16px 24px', gap: '12px' }}>
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
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          variant="contained"
          onClick={handleSubmit}
          sx={{
            textTransform: 'none',
            backgroundColor: '#522b4a',
            '&:hover': {
              backgroundColor: '#613a5c',
            },
          }}
          disabled={isSubmitting}
        >
          {isSubmitting ? (
            <CircularProgress size={24} color="inherit" />
          ) : (
            'Submit'
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );

  const renderCacheSection = (title, cacheKey) => (
    <div>
      <h5>{title}</h5>
      {['enabled', 'conf-id', 'ttl-in-seconds', 'jitter-percentage'].map((field) => (
        field === 'enabled' ? (
          <FormControl key={field} fullWidth style={{ marginTop: '1.5rem', display: 'flex', justifyContent: 'flex-start' }}>
            <InputLabel id={`${cacheKey}-${field}-label`} sx={{ left: '0px', '&.Mui-focused': { left: '0px' }}}>
              Enabled
            </InputLabel>
            <Select
              labelId={`${cacheKey}-${field}-label`}
              name={`${cacheKey}.${field}`}
              value={entityData[cacheKey][field]}
              onChange={handleChange}
              label="Enabled"
              className="custom-textfield"
              sx={{ '& .MuiSelect-select': { textAlign: 'left' }}}
            >
              <MenuItem value="true">True</MenuItem>
              <MenuItem value="false">False</MenuItem>
            </Select>
          </FormControl>
        ) : field === 'conf-id' ? (
          <FormControl key={field} fullWidth style={{ marginTop: '1.5rem', display: 'flex', justifyContent: 'flex-start' }}>
            <InputLabel id={`${cacheKey}-${field}-label`} sx={{ left: '0px', '&.Mui-focused': { left: '0px' }}}>
              Config ID
            </InputLabel>
            <Select
              labelId={`${cacheKey}-${field}-label`}
              name={`${cacheKey}.${field}`}
              value={entityData[cacheKey][field]}
              onChange={handleChange}
              label="Config ID"
              className="custom-textfield"
              sx={{ '& .MuiSelect-select': { textAlign: 'left' }}}
            >
              {cacheKey === 'distributed-cache' ? (
                <MenuItem value="2">2</MenuItem>
              ) : (
                <MenuItem value="3">3</MenuItem>
              )}
            </Select>
          </FormControl>
        ) : (
          <TextField
            key={field}
            label={field.charAt(0).toUpperCase() + field.slice(1).replace(/-/g, ' ')}
            name={`${cacheKey}.${field}`}
            value={entityData[cacheKey][field]}
            onChange={handleChange}
            fullWidth
            placeholder={`Enter ${field.charAt(0).toUpperCase() + field.slice(1).replace(/-/g, ' ')}`}
            style={{ display: 'flex', justifyContent: 'flex-start', marginTop: '1.5rem' }}
            className="custom-textfield"
          />
        )
      ))}
    </div>
  );

  return (
    <div style={{ padding: '20px' }}>
      <GenericTable
        data={entityRequests}
        excludeColumns={['FeatureGroupLabel']}
        onRowAction={(row) => handleViewEntity(row)}
        loading={false}
        actionButtons={[
          {
            label: "Create Entity",
            onClick: handleOpen,
            variant: "contained",
            color: "#522b4a",
            hoverColor: "#613a5c"
          }
        ]}
      />
      {renderEntityModal()}

      {/* View Entity Modal */}
      {selectedEntity && (
        <Modal show={showViewModal} onHide={closeViewModal} size="lg" centered>
          <Modal.Header closeButton>
            <Modal.Title>Entity Details</Modal.Title>
          </Modal.Header>
          <Modal.Body>
            <ListGroup>
              <ListGroup.Item>
                <strong>Request Type:</strong> {selectedEntity.requestType}
              </ListGroup.Item>
              
              <ListGroup.Item>
                <strong>Entity Label:</strong> {selectedEntity["entity-label"]}
              </ListGroup.Item>
              
              {selectedEntity.status === "REJECTED" && (
                <ListGroup.Item>
                  <strong>Reject Reason:</strong> {selectedEntity.rejectReason}
                </ListGroup.Item>
              )}
              
              {!isEditRequest && (
                <ListGroup.Item>
                  <strong>Keys:</strong>
                  <Table bordered hover size="sm" className="mt-2">
                    <thead>
                      <tr>
                        <th>Sequence</th>
                        <th>Entity Key</th>
                        <th>Column Key</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedEntity["key-map"].map((key, index) => (
                        <tr key={index}>
                          <td>{index + 1}</td>
                          <td>{key["entity-label"]}</td>
                          <td>{key["column-label"]}</td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                </ListGroup.Item>
              )}
              <ListGroup.Item>
                <strong>Distributed Cache:</strong>
                <Table bordered hover size="sm" className="mt-2">
                  <tbody>
                    <tr>
                      <td>Enabled</td>
                      <td>{selectedEntity["distributed-cache"].enabled}</td>
                    </tr>
                    <tr>
                      <td>Config ID</td>
                      <td>{selectedEntity["distributed-cache"]["conf-id"]}</td>
                    </tr>
                    <tr>
                      <td>TTL (seconds)</td>
                      <td>{selectedEntity["distributed-cache"]["ttl-in-seconds"]}</td>
                    </tr>
                    <tr>
                      <td>Jitter Percentage</td>
                      <td>{selectedEntity["distributed-cache"]["jitter-percentage"]}</td>
                    </tr>
                  </tbody>
                </Table>
              </ListGroup.Item>

              <ListGroup.Item>
                <strong>In-Memory Cache:</strong>
                <Table bordered hover size="sm" className="mt-2">
                  <tbody>
                    <tr>
                      <td>Enabled</td>
                      <td>{selectedEntity["in-memory-cache"].enabled}</td>
                    </tr>
                    <tr>
                      <td>Config ID</td>
                      <td>{selectedEntity["in-memory-cache"]["conf-id"]}</td>
                    </tr>
                    <tr>
                      <td>TTL (seconds)</td>
                      <td>{selectedEntity["in-memory-cache"]["ttl-in-seconds"]}</td>
                    </tr>
                    <tr>
                      <td>Jitter Percentage</td>
                      <td>{selectedEntity["in-memory-cache"]["jitter-percentage"]}</td>
                    </tr>
                  </tbody>
                </Table>
              </ListGroup.Item>
            </ListGroup>
          </Modal.Body>
        </Modal>
      )}

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

const getInitialEntityData = () => ({
  'entity-label': '',
  'key-map': [{ sequence: '', 'entity-label': '', 'column-label': '' }],
  'distributed-cache': { enabled: '', 'conf-id': '', 'ttl-in-seconds': '', 'jitter-percentage': '' },
  'in-memory-cache': { enabled: '', 'conf-id': '', 'ttl-in-seconds': '', 'jitter-percentage': '' },
});

const updateNestedState = (setStateFn, name, value) => {
  const keys = name.split('.');
  setStateFn((prevData) => {
    const updatedData = { ...prevData };
    let ref = updatedData;
    for (let i = 0; i < keys.length - 1; i++) {
      ref = ref[keys[i]];
    }
    ref[keys[keys.length - 1]] = value;
    return updatedData;
  });
};

const transformRequestData = (entityData) => ({
  'entity-label': entityData['entity-label'],
  'key-map': entityData['key-map'].reduce((acc, keyMap, index) => {
    acc[index.toString()] = {
      sequence: index,
      'entity-label': keyMap['entity-label'],
      'column-label': keyMap['column-label'],
    };
    return acc;
  }, {}),
  'distributed-cache': {
    ...entityData['distributed-cache'],
    'ttl-in-seconds': parseInt(entityData['distributed-cache']['ttl-in-seconds'], 10) || 0,
    'jitter-percentage': parseInt(entityData['distributed-cache']['jitter-percentage'], 10) || 0,
    'conf-id': parseInt(entityData['distributed-cache']['conf-id'], 10) || 0,
  },
  'in-memory-cache': {
    ...entityData['in-memory-cache'],
    'ttl-in-seconds': parseInt(entityData['in-memory-cache']['ttl-in-seconds'], 10) || 0,
    'jitter-percentage': parseInt(entityData['in-memory-cache']['jitter-percentage'], 10) || 0,
    'conf-id': parseInt(entityData['in-memory-cache']['conf-id'], 10) || 0,
  },
});

export default EntityRegistry;
