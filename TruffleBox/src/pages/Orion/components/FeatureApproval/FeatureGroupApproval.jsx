import React, { useState, useEffect, useCallback } from 'react';
import { Modal, ListGroup, Table } from 'react-bootstrap';
import { 
  Button, 
  Dialog,
  DialogTitle,
  DialogContent, 
  DialogActions,
  Box, 
  Typography,
  Divider
} from '@mui/material';
import './styles.scss';
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

import * as URL_CONSTANTS from '../../../../config';

const FeatureGroupApproval = () => {
  const initialFeatureGroupData = {
    "user-id": "",
    "request-id": "",
    "entity-label": "",
    "fg-label": "",
    "job-id": "",
    "store-id": 0,
    "ttl-in-seconds": 0,
    "in-memory-cache-enabled": false,
    "distributed-cache-enabled": false,
    "data-type": "",
    "status": "",
    "request-type": "",
    features: [{ 
      labels: "", 
      "default-values": "",
      "source-base-path": "",
      "source-data-column": "",
      "storage-provider": "",
      "string-length": "",
      "vector-length": ""
    }],
  };

  const [showModal, setShowModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [featureGroupData, setFeatureGroupData] = useState(initialFeatureGroupData);
  const [featureGroupRequests, setFeatureGroupRequests] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [actionMessage, setActionMessage] = useState('');
  const { user } = useAuth();

  const fetchFeatureGroupRequests = useCallback(async () => {
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/get-feature-group-requests`,
        {
          headers: {
            Authorization: `Bearer ${user.token}`,
          },
        }
      );
      const data = await response.json();
      setFeatureGroupRequests(data);
    } catch (error) {
      console.error('Error fetching feature group requests:', error);
    }
  }, [user.token]);

  useEffect(() => {
    fetchFeatureGroupRequests();
  }, [fetchFeatureGroupRequests]);

  const transformPayloadData = (payload, requestId, status, requestType, rejectReason) => {
    return {
      ...payload,
      "request-id": requestId,
      "status": status,
      "request-type": requestType,
      "reject-reason": rejectReason || "",
    };
  };

  const handleOpen = (featureGroup = null) => {
    try {
      if (featureGroup) {
          const transformedData = transformPayloadData(
            featureGroup.Payload,
            featureGroup.RequestId,
            featureGroup.Status,
            featureGroup.RequestType,
            featureGroup.RejectReason
          );
          setFeatureGroupData(transformedData);
      }
      setShowModal(true);
    } catch (error) {
      console.error('Error in handleOpen:', error);
      setFeatureGroupData(initialFeatureGroupData);
      setShowModal(true);
    }
  };

  const handleClose = () => {
    setShowModal(false);
    // Reset the feature group data when closing the modal
    setFeatureGroupData(initialFeatureGroupData);
    setRejectReason('');
  };

  const handleReject = () => {
    setShowModal(false);
    setShowRejectModal(true);
  };

  const handleRejectModalClose = () => {
    setShowRejectModal(false);
    setRejectReason('');
    setShowModal(true);
  };

  const handleSubmit = async (event, status) => {
    event.preventDefault();
    setLoading(true);
    try {
      const requestData = {
        "request-id": featureGroupData["request-id"],
        status: status.toUpperCase(),
        "reject-reason": status.toUpperCase() === 'REJECTED' ? rejectReason : '',
      };

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/process-feature-group`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${user.token}`,
          },
          body: JSON.stringify(requestData),
        }
      );

      const result = await response.json();
      if (response.ok) {
        setActionMessage(result.message || 'Feature group request processed successfully');
        setShowSuccessModal(true);
        // Fetch updated data after action
        await fetchFeatureGroupRequests();
        // Close the modal after successful action
        handleClose();
        setShowRejectModal(false);
      } else {
        setActionMessage(result.error || 'Unknown error occurred');
        setShowErrorModal(true);
      }
    } catch (error) {
      setActionMessage('Network error. Please try again.');
      setShowErrorModal(true);
    } finally {
      setLoading(false);
    }
  };

  // Check if status is APPROVED or REJECTED
  const shouldShowFooter = showModal && featureGroupData.status !== 'APPROVED' && featureGroupData.status !== 'REJECTED';

  const cellStyle = {
    maxWidth: "150px", 
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "normal",
    wordBreak: "break-word"
  };

  const renderModalBody = () => {
    if (featureGroupData["request-type"] === "EDIT") {
      return (
        <ListGroup>
          <ListGroup.Item>
            <strong>Request ID:</strong> {featureGroupData["request-id"]}
          </ListGroup.Item>
          <ListGroup.Item>
            <strong>Request Type:</strong> {featureGroupData["request-type"]}
          </ListGroup.Item>
          <ListGroup.Item>
            <strong>Entity Label:</strong> {featureGroupData["entity-label"]}
          </ListGroup.Item>
          <ListGroup.Item>
            <strong>Feature Group Label:</strong> {featureGroupData["fg-label"]}
          </ListGroup.Item>
          <ListGroup.Item>
            <strong>Status:</strong> {featureGroupData.status}
          </ListGroup.Item>
          {featureGroupData.status === 'REJECTED' && featureGroupData["reject-reason"] && (
            <ListGroup.Item>
              <strong>Reject Reason:</strong> {featureGroupData["reject-reason"]}
            </ListGroup.Item>
          )}
          <ListGroup.Item>
            <strong>TTL in Seconds:</strong> {featureGroupData["ttl-in-seconds"]}
          </ListGroup.Item>
          <ListGroup.Item>
            <strong>In-Memory Cache:</strong> {featureGroupData["in-memory-cache-enabled"] ? "Enabled" : "Disabled"}
          </ListGroup.Item>
          <ListGroup.Item>
            <strong>Distributed Cache:</strong> {featureGroupData["distributed-cache-enabled"] ? "Enabled" : "Disabled"}
          </ListGroup.Item>
          {featureGroupData["layout-version"] !== undefined && (
            <ListGroup.Item>
              <strong>Layout Version:</strong> {featureGroupData["layout-version"]}
            </ListGroup.Item>
          )}
        </ListGroup>
      );
    }
    
    // For CREATE requests, show the original detailed view
    return (
      <ListGroup>
        <ListGroup.Item>
          <strong>Request ID:</strong> {featureGroupData["request-id"]}
        </ListGroup.Item>
        {featureGroupData["request-type"] && (
          <ListGroup.Item>
            <strong>Request Type:</strong> {featureGroupData["request-type"]}
          </ListGroup.Item>
        )}
        <ListGroup.Item>
          <strong>Entity Label:</strong> {featureGroupData["entity-label"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Feature Group Label:</strong> {featureGroupData["fg-label"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Status:</strong> {featureGroupData.status}
        </ListGroup.Item>
        {featureGroupData.status === 'REJECTED' && featureGroupData["reject-reason"] && (
          <ListGroup.Item>
            <strong>Reject Reason:</strong> {featureGroupData["reject-reason"]}
          </ListGroup.Item>
        )}
        <ListGroup.Item>
          <strong>Job Name:</strong> {featureGroupData["job-id"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Store ID:</strong> {featureGroupData["store-id"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>TTL in Seconds:</strong> {featureGroupData["ttl-in-seconds"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>In-Memory Cache:</strong> {featureGroupData["in-memory-cache-enabled"] ? "Enabled" : "Disabled"}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Distributed Cache:</strong> {featureGroupData["distributed-cache-enabled"] ? "Enabled" : "Disabled"}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Data Type:</strong> {featureGroupData["data-type"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Layout Version:</strong> {featureGroupData["layout-version"]}
        </ListGroup.Item>
        <ListGroup.Item>
          <strong>Features</strong>
          <Table bordered hover size="sm" className="mt-2">
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
              {featureGroupData.features?.map((feature, index) => (
                <tr key={index}>
                  <td style={cellStyle}>{feature.labels}</td>
                  <td style={cellStyle}>{feature["default-values"]}</td>
                  <td style={cellStyle}>{feature["source-base-path"] || "N/A"}</td>
                  <td style={cellStyle}>{feature["source-data-column"] || "N/A"}</td>
                  <td style={cellStyle}>{feature["storage-provider"] || "N/A"}</td>
                  <td>{feature["string-length"]}</td>
                  <td>{feature["vector-length"]}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        </ListGroup.Item>
      </ListGroup>
    );
  };

  return (
    <div>
      <GenericTable
        data={featureGroupRequests}
        onRowAction={handleOpen}
        loading={loading}
      />
      
      <Modal show={showModal} onHide={handleClose} size="xl" centered>
        <Modal.Header closeButton>
          <Modal.Title style={{ fontFamily: "system-ui" }}>
            {featureGroupData["request-type"] === "EDIT" ? "Edit Feature Group" : "Feature Groups"}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {renderModalBody()}
        </Modal.Body>
        {shouldShowFooter && (
          <Modal.Footer style={{ gap: '12px', justifyContent: 'flex-end' }}>
            <Button 
              variant="contained" 
              color="success"
              startIcon={<CheckCircleIcon />}
              sx={{
                backgroundColor: '#66bb6a',  
                textTransform: 'none',  
                '&:hover': {
                  backgroundColor: '#4caf50', 
                }
              }}
              onClick={(e) => handleSubmit(e, 'APPROVED')}
              disabled={loading}
            >
              {'Approve'}
            </Button>
            <Button 
              variant="contained" 
              color="error"
              startIcon={<CancelIcon />}
              sx={{
                backgroundColor: '#ef5350',  
                textTransform: 'none',  
                '&:hover': {
                  backgroundColor: '#e53935', 
                }
              }}
              onClick={handleReject}
              disabled={loading}
            >
              {'Reject'}
            </Button>
          </Modal.Footer>
        )}
      </Modal>

      {/* Reject Reason Modal */}
      <Modal
        show={showRejectModal}
        onHide={handleRejectModalClose}
        size="md"
        centered
      >
        <Modal.Header closeButton>
          <Modal.Title style={{ fontFamily: "system-ui" }}>Reject Reason</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <div className="form-group">
            <textarea
              className="form-control"
              placeholder="Please provide a reason for rejection (max 255 characters)"
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value.slice(0, 255))}
              rows={4}
              maxLength={255}
              style={{ resize: 'none' }}
            />
            <small className="text-muted">{rejectReason.length}/255 characters</small>
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button 
            variant="contained" 
            color="error"
            sx={{
              backgroundColor: '#ef5350',
              textTransform: 'none',
              '&:hover': {
                backgroundColor: '#e53935',
              }
            }}
            onClick={(e) => handleSubmit(e, 'REJECTED')}
            disabled={loading}
          >
            {loading ? 'Processing...' : 'Submit'}
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Success Modal */}
      <Dialog
        open={showSuccessModal}
        onClose={() => setShowSuccessModal(false)}
        maxWidth="sm"
      >
        <DialogTitle>
          Success
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CheckCircleOutlineIcon sx={{ color: 'green' }} />
            <Typography>
              {actionMessage}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => setShowSuccessModal(false)}
            sx={{
              backgroundColor: '#450839',
              color: 'white',
              '&:hover': {
                backgroundColor: '#5A0A4B',
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
        onClose={() => setShowErrorModal(false)}
        maxWidth="sm"
      >
        <DialogTitle>
          Error
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ErrorOutlineIcon sx={{ color: 'red' }} />
            <Typography>
              {actionMessage}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => setShowErrorModal(false)}
            sx={{
              backgroundColor: '#450839',
              color: 'white',
              '&:hover': {
                backgroundColor: '#5A0A4B',
              },
            }}
          >
            OK
          </Button>
        </DialogActions>
      </Dialog>

      {/* Loading Modal */}
      <Dialog
        open={loading}
        maxWidth="sm"
      >
        <DialogContent sx={{ pt: 3, pb: 3, minWidth: '250px' }}>
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 2 }}>
            <div className="spinner-border" style={{ color: '#450839' }} role="status">
              <span className="visually-hidden">Loading...</span>
            </div>
            <Typography>
              Processing your request...
            </Typography>
          </Box>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default FeatureGroupApproval;
