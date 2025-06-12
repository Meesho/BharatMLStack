import React, { useState, useEffect } from 'react';
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
  Typography,
  Box,
} from '@mui/material';
import { Modal, ListGroup } from 'react-bootstrap';
import "./styles.scss";
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

import * as URL_CONSTANTS from '../../../../config';

const StoreRegistry = () => {
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedStore, setSelectedStore] = useState(null);
  const [configOptions, setConfigOptions] = useState([]);
  const [configDbTypeMap, setConfigDbTypeMap] = useState({});
  const [storeData, setStoreData] = useState({
    "conf-id": "",
    "db-type": "",
    table: "",
    "table-ttl": "",
    "primary-keys": [],
    "RejectReason": ""
  });
  const [storeRequests, setStoreRequests] = useState([]);
  const { user } = useAuth();
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

  useEffect(() => {
    const fetchStoreRequests = async () => {
      try {
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-store-requests`, {
          headers: {
            Authorization: `Bearer ${user.token}`,
          },
        });

        if (!response.ok) {
          console.error('Failed to fetch store requests:', response.statusText);
          return;
        }

        const data = await response.json();
        setStoreRequests(data);
      } catch (error) {
        console.error('Error fetching store requests:', error);
      }
    };

    const fetchConfigOptions = async () => {
      try {
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-config`, {
          headers: {
            Authorization: `Bearer ${user.token}`,
          },
        });

        if (!response.ok) {
          console.error('Failed to fetch config options:', response.statusText);
          return;
        }

        const data = await response.json();
        const configIds = Object.keys(data).map(id => parseInt(id, 10));
        setConfigOptions(configIds);
        setConfigDbTypeMap(data);
      } catch (error) {
        console.error('Error fetching config options:', error);
      }
    };

    fetchStoreRequests();
    fetchConfigOptions();
  }, [user.token]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setStoreData((prevData) => {
      const updates = {
        [name]: name === "conf-id" ? parseInt(value, 10) : 
                name === "table-ttl" ? (value === "" ? "" : parseInt(value, 10)) : 
                value,
      };
      
      // Auto-fill db-type when conf-id changes
      if (name === "conf-id" && value) {
        updates["db-type"] = configDbTypeMap[value] || "";
      }
      
      return {
        ...prevData,
        ...updates
      };
    });
  };

  const handleOpen = () => {
    setStoreData({
      "conf-id": "",
      "db-type": "",
      table: "",
      "table-ttl": "",
      "primary-keys": [""],
      "RejectReason": ""
    });
    setValidationErrors({});
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleViewStore = (store) => {
    const parsedPayload = JSON.parse(store.Payload);
    setSelectedStore({
      ...parsedPayload,
      RejectReason: store.RejectReason || "-",
      Status: store.Status
    });
    setShowViewModal(true);
  };

  const closeViewModal = () => {
    setShowViewModal(false);
    setSelectedStore(null);
  };

  const validateForm = () => {
    const errors = {};
    
    if (!storeData["conf-id"]) {
      errors["conf-id"] = "Config ID is required";
    }
    
    if (!storeData["db-type"]) {
      errors["db-type"] = "Database Type is required";
    }
    
    if (!storeData.table || storeData.table.trim() === "") {
      errors.table = "Table name is required";
    }
    
    if (!storeData["table-ttl"] || storeData["table-ttl"] === "") {
      errors["table-ttl"] = "Table Time to Live is required";
    } else if (storeData["table-ttl"] <= 0) {
      errors["table-ttl"] = "Table Time to Live must be greater than 0";
    }
    
    const nonEmptyKeys = storeData["primary-keys"].filter(key => key && key.trim() !== "");
    if (nonEmptyKeys.length === 0) {
      errors["primary-keys"] = "At least one primary key is required";
    }
    
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

    const apiEndpoint = `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/register-store`;
    try {
      const response = await fetch(apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(storeData),
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

  const addPrimaryKey = () => {
    setStoreData(prevData => ({
      ...prevData,
      "primary-keys": [...prevData["primary-keys"], ""]
    }));
  };

  const removePrimaryKey = (index) => {
    setStoreData(prevData => {
      const newKeys = [...prevData["primary-keys"]];
      newKeys.splice(index, 1);
      return {
        ...prevData,
        "primary-keys": newKeys
      };
    });
  };

  const updatePrimaryKey = (index, value) => {
    setStoreData(prevData => {
      const newKeys = [...prevData["primary-keys"]];
      newKeys[index] = value;
      return {
        ...prevData,
        "primary-keys": newKeys
      };
    });
  };

  return (
    <div style={{ padding: '20px' }}>
      <GenericTable
        data={storeRequests}
        excludeColumns={['EntityLabel', 'FeatureGroupLabel']}
        onRowAction={(row) => handleViewStore(row)}
        loading={false}
        actionButtons={[
          {
            label: "Register Store",
            onClick: handleOpen,
            variant: "contained",
            color: "#522b4a",
            hoverColor: "#613a5c"
          }
        ]}
      />

      {/* Create/Register Modal */}
      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle>Register Store</DialogTitle>
        <DialogContent>
          <FormControl fullWidth margin="normal" error={!!validationErrors["conf-id"]}>
            <InputLabel id="conf-id-label">Config ID *</InputLabel>
            <Select
              labelId="conf-id-label"
              name="conf-id"
              value={storeData["conf-id"]}
              onChange={handleChange}
              fullWidth
              label="Config ID *"
              error={!!validationErrors["conf-id"]}
            >
              {configOptions.map((id) => (
                <MenuItem key={id} value={id}>
                  {id}
                </MenuItem>
              ))}
            </Select>
            {validationErrors["conf-id"] && (
              <Typography variant="caption" color="error" sx={{ mt: 1 }}>
                {validationErrors["conf-id"]}
              </Typography>
            )}
          </FormControl>
          <FormControl fullWidth margin="normal" error={!!validationErrors["db-type"]}>
            <InputLabel id="db-type-label">Database Type *</InputLabel>
            <Select
              labelId="db-type-label"
              name="db-type"
              value={storeData["db-type"]}
              onChange={handleChange}
              fullWidth
              label="Database Type *"
              disabled
              error={!!validationErrors["db-type"]}
            >
              <MenuItem value={storeData["db-type"]}>{storeData["db-type"]}</MenuItem>
            </Select>
            {validationErrors["db-type"] && (
              <Typography variant="caption" color="error" sx={{ mt: 1 }}>
                {validationErrors["db-type"]}
              </Typography>
            )}
          </FormControl>
          <TextField
            label="Table *"
            name="table"
            value={storeData.table}
            onChange={handleChange}
            fullWidth
            margin="normal"
            error={!!validationErrors.table}
            helperText={validationErrors.table}
          />
          <TextField
            label="Table Time to Live (in seconds) *"
            name="table-ttl"
            type="number"
            value={storeData["table-ttl"]}
            onChange={handleChange}
            fullWidth
            margin="normal"
            error={!!validationErrors["table-ttl"]}
            helperText={validationErrors["table-ttl"]}
          />
          
          <div style={{ marginTop: '20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography variant="subtitle1" color={validationErrors["primary-keys"] ? "error" : "inherit"}>
                Primary Keys *
              </Typography>
              <Button 
                variant="text" 
                size="small" 
                onClick={addPrimaryKey}
                startIcon={<AddCircleOutlineIcon />}
                sx={{
                  color: 'green',
                  '&:hover': {
                    backgroundColor: '#fff',
                  },
                  textTransform: 'none',
                }}
              >
                Add Primary Key
              </Button>
            </div>
            
            {validationErrors["primary-keys"] && (
              <Typography variant="caption" color="error" sx={{ mt: 1, display: 'block' }}>
                {validationErrors["primary-keys"]}
              </Typography>
            )}
            
            {storeData["primary-keys"].length === 0 && (
              <Typography variant="body2" color="textSecondary" style={{ marginTop: '10px' }}>
                No primary keys added yet
              </Typography>
            )}
            
            {storeData["primary-keys"].map((key, index) => (
              <div key={index} style={{ display: 'flex', gap: '10px', marginTop: '10px', alignItems: 'center' }}>
                <TextField
                  fullWidth
                  label={`Primary Key ${index + 1} *`}
                  value={key}
                  onChange={(e) => updatePrimaryKey(index, e.target.value)}
                  error={validationErrors["primary-keys"] && (!key || key.trim() === "")}
                />
                <Button 
                  variant="text" 
                  color="error" 
                  size="small" 
                  onClick={() => removePrimaryKey(index)}
                  startIcon={<RemoveCircleOutlineIcon />}
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
            ))}
          </div>
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

      {/* View Store Modal */}
      {selectedStore && (
        <Modal
          show={showViewModal}
          onHide={closeViewModal}
          size="lg" 
          centered
          dialogClassName="custom-modal-width"
        >
          <Modal.Header closeButton>
            <Modal.Title>Store Details</Modal.Title>
          </Modal.Header>
          <Modal.Body>
            <ListGroup>
              <ListGroup.Item>
                <strong>Config ID:</strong> {selectedStore["conf-id"]}
              </ListGroup.Item>
              <ListGroup.Item>
                <strong>Database Type:</strong> {selectedStore["db-type"]}
              </ListGroup.Item>
              <ListGroup.Item>
                <strong>Table:</strong> {selectedStore.table}
              </ListGroup.Item>
              <ListGroup.Item>
                <strong>Table Time to Live (in seconds):</strong> {selectedStore["table-ttl"] || "-"}
              </ListGroup.Item>
              <ListGroup.Item>
                <strong>Primary Keys:</strong>{" "}
                {selectedStore["primary-keys"] && selectedStore["primary-keys"].length > 0
                  ? selectedStore["primary-keys"].join(", ")
                  : "-"}
              </ListGroup.Item>
              {selectedStore.Status === "REJECTED" && (
                <ListGroup.Item>
                  <strong>Reject Reason:</strong> {selectedStore.RejectReason}
                </ListGroup.Item>
              )}
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

export default StoreRegistry;
