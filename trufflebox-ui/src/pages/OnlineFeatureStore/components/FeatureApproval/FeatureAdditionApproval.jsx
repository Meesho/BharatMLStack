import React, { useState, useEffect, useCallback } from 'react';
import { Modal, ListGroup, Table } from 'react-bootstrap';
import { 
  Button, 
  CircularProgress, 
  Dialog, 
  DialogTitle, 
  DialogContent, 
  DialogActions, 
  Box, 
  Typography,
  Divider,
  Chip
} from '@mui/material';
import './styles.scss';
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

import * as URL_CONSTANTS from '../../../../config';

const FeatureAdditionApproval = () => {
  const [showModal, setShowModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [featureAdditionData, setFeatureAdditionData] = useState({
    "request-id": "",
    "entity-label": "",
    "feature-group-label": "",
    "features": [{ 
      "labels": "", 
      "default-values": "",
      "source-base-path": "",
      "source-data-column": "",
      "storage-provider": "",
      "string-length": 0,
      "vector-length": 0
    }],
    "status": "",
    "request-type": "",
    "reject-reason": "",
  });
  const [featuresRequests, setFeaturesRequests] = useState([]);
  const { user } = useAuth();
  
  // Add new state variables
  const [isLoading, setIsLoading] = useState(false);
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [responseMessage, setResponseMessage] = useState('');

  const cellStyle = {
    maxWidth: "150px", 
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "normal",
    wordBreak: "break-word"
  };

  const fetchFeaturesRequests = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-add-features-requests`, {
        headers: { Authorization: `Bearer ${user.token}` },
      });
      const data = await response.json();

      if (Array.isArray(data)) {
        setFeaturesRequests(data);
      } else {
        setFeaturesRequests([]);
      }
    } catch (error) {
      console.error('Error fetching features requests:', error);
      setFeaturesRequests([]);
    } finally {
      setIsLoading(false);
    }
  }, [user.token]);

  useEffect(() => {
    fetchFeaturesRequests();
  }, [fetchFeaturesRequests]);

  const handleOpen = (featureAddition = null) => {
    try {
      const data = featureAddition?.Payload || {};
      const requestType = featureAddition?.RequestType || "CREATE";
      
      let features = [];
      if (requestType === 'DELETE') {
        // For DELETE requests, feature-labels is an array of strings
        features = data["feature-labels"]?.map(label => ({ labels: label })) || [];
      } else {
        // For CREATE/other requests, features is an array of objects with detailed info
        features = data["features"]?.map(item => ({
          "labels": item.labels || "",
          "default-values": item["default-values"] || "",
          "source-base-path": item["source-base-path"] || "",
          "source-data-column": item["source-data-column"] || "",
          "storage-provider": item["storage-provider"] || "",
          "string-length": item["string-length"] || "",
          "vector-length": item["vector-length"] || ""
        })) || [];
      }
      
      setFeatureAdditionData({
        "request-id": featureAddition?.RequestId || "",
        "entity-label": data["entity-label"] || "",
        "feature-group-label": data["feature-group-label"] || "",
        "status": featureAddition?.Status || "",
        "request-type": requestType,
        "reject-reason": featureAddition?.RejectReason || "",
        "features": features,
      });
      setShowModal(true);
    } catch (error) {
      console.error('Error parsing feature payload:', error);
    }
  };

  const handleClose = () => {
    setShowModal(false);
    // Reset the feature addition data when closing the modal
    setFeatureAdditionData({
      "request-id": "",
      "entity-label": "",
      "feature-group-label": "",
      "features": [{ 
        "labels": "", 
        "default-values": "",
        "source-base-path": "",
        "source-data-column": "",
        "storage-provider": "",
        "string-length": "",
        "vector-length": ""
      }],
      "status": "",
      "request-type": "",
      "reject-reason": "",
    });
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
    setIsLoading(true);
    try {
      const requestData = {
        "request-id": featureAdditionData["request-id"],
        status: status.toUpperCase(),
        "reject-reason": status.toUpperCase() === 'REJECTED' ? rejectReason : '',
      };

      const isDeleteRequest = featureAdditionData["request-type"] === 'DELETE';
      const apiEndpoint = isDeleteRequest 
        ? `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/process-delete-features`
        : `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/process-add-features`;

      const response = await fetch(apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(requestData),
      });

      const result = await response.json();
      if (response.ok) {
        setResponseMessage(result.message);
        setShowSuccessModal(true);
      } else {
        setResponseMessage(result.error);
        setShowErrorModal(true);
      }
      handleClose();
      setShowRejectModal(false);
      fetchFeaturesRequests(); // Refresh data instead of page reload
    } catch (error) {
      setResponseMessage('Network error. Please try again.');
      setShowErrorModal(true);
    } finally {
      setIsLoading(false);
    }
  };

  // Check if status is APPROVED or REJECTED
  const shouldShowFooter = showModal && featureAdditionData.status !== 'APPROVED' && featureAdditionData.status !== 'REJECTED';

  return (
    <div>
      <GenericTable 
        data={featuresRequests} 
        onRowAction={handleOpen} 
        loading={isLoading} 
        flowType="approval"
      />
      
      <Modal
        show={showModal} 
        onHide={handleClose} 
        size="xl" 
        centered
      >
        <Modal.Header closeButton>
          <Modal.Title style={{ fontFamily: "system-ui" }}>
            {featureAdditionData["request-type"] === 'DELETE' ? 'Feature Deletion Request' : 'Feature Addition Request'}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <ListGroup>
            <ListGroup.Item>
              <strong>Request ID:</strong> {featureAdditionData["request-id"]}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Entity Label:</strong> {featureAdditionData["entity-label"]}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Feature Group Label:</strong> {featureAdditionData["feature-group-label"]}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Status:</strong> {featureAdditionData.status}
            </ListGroup.Item>
            {featureAdditionData.status === 'REJECTED' && featureAdditionData["reject-reason"] && (
              <ListGroup.Item>
                <strong>Reject Reason:</strong> {featureAdditionData["reject-reason"]}
              </ListGroup.Item>
            )}
            <ListGroup.Item>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <strong>Request Type:</strong> 
                <Chip 
                  label={featureAdditionData["request-type"]}
                  variant="filled"
                  size="small"
                  sx={{
                    backgroundColor: featureAdditionData["request-type"] === 'DELETE' 
                      ? '#ffebee' 
                      : featureAdditionData["request-type"] === 'CREATE'
                      ? '#e8f5e8'
                      : '#f3e5f5',
                    color: featureAdditionData["request-type"] === 'DELETE' 
                      ? '#d32f2f' 
                      : featureAdditionData["request-type"] === 'CREATE'
                      ? '#2e7d32'
                      : '#7b1fa2',
                    fontWeight: 'bold',
                    fontSize: '12px',
                    textTransform: 'uppercase',
                    letterSpacing: '0.5px'
                  }}
                />
              </div>
            </ListGroup.Item>
            {featureAdditionData["request-type"] === 'DELETE' ? (
              <ListGroup.Item>
                <div style={{ marginBottom: '12px' }}>
                  <strong>{featureAdditionData["status"] === 'APPROVED' ? 'Deleted Features:' : 'Features to Delete:'}</strong>
                </div>
                <Box sx={{ 
                  backgroundColor: '#ffebee', 
                  border: '1px solid #ffcdd2', 
                  borderRadius: '8px', 
                  padding: '16px',
                  maxHeight: '300px',
                  overflowY: 'auto'
                }}>
                  {featureAdditionData.features?.length > 0 ? (
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                      {featureAdditionData.features.map((feature, index) => (
                        <Box 
                          key={index}
                          sx={{ 
                            display: 'flex', 
                            alignItems: 'center',
                            backgroundColor: 'white',
                            padding: '12px 16px',
                            borderRadius: '6px',
                            border: '1px solid #ffcdd2',
                            boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                          }}
                        >
                          <Typography variant="body2" sx={{ 
                            color: '#d32f2f',
                            fontWeight: 500,
                            fontSize: '14px',
                            wordBreak: 'break-word',
                            flexGrow: 1
                          }}>
                            • {feature.labels || feature}
                          </Typography>
                        </Box>
                      ))}
                    </Box>
                  ) : (
                    <Typography variant="body2" sx={{ 
                      color: '#9e9e9e',
                      fontStyle: 'italic',
                      textAlign: 'center'
                    }}>
                      No features specified for deletion
                    </Typography>
                  )}
                  
                  <Box sx={{ 
                    mt: 2, 
                    pt: 2, 
                    borderTop: '1px solid #ffcdd2',
                    textAlign: 'center'
                  }}>
                    <Typography variant="caption" sx={{ 
                      color: '#d32f2f',
                      fontWeight: 'bold',
                      fontSize: '12px'
                    }}>
                      {featureAdditionData["status"] === 'APPROVED' ? 'These features have been permanently deleted' : '⚠️ These features will be permanently deleted'}
                    </Typography>
                  </Box>
                </Box>
              </ListGroup.Item>
            ) : (
              <ListGroup.Item>
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
                    {featureAdditionData.features.map((feature, index) => (
                      <tr key={index}>
                      <td style={cellStyle}>{feature.labels}</td>
                      <td style={cellStyle}>{feature["default-values"]}</td>
                      <td style={cellStyle}>{feature["source-base-path"]}</td>
                      <td style={cellStyle}>{feature["source-data-column"]}</td>
                      <td style={cellStyle}>{feature["storage-provider"]}</td>
                      <td>{feature["string-length"]}</td>
                      <td>{feature["vector-length"]}</td>
                    </tr>
                    ))}
                  </tbody>
                </Table>
              </ListGroup.Item>
            )}
          </ListGroup>
        </Modal.Body>
        {shouldShowFooter && (
          <Modal.Footer style={{ gap: '12px', justifyContent: 'flex-end' }}>
            <Button 
              variant="contained" 
              color="success"
              startIcon={isLoading ? null : <CheckCircleIcon />}
              sx={{
                backgroundColor: '#66bb6a',  
                textTransform: 'none',  
                '&:hover': {
                  backgroundColor: '#4caf50', 
                }
              }}
              onClick={(e) => handleSubmit(e, 'APPROVED')}
              disabled={isLoading}
            >
              {isLoading ? <CircularProgress size={24} color="inherit" /> : 'Approve'}
            </Button>
            <Button 
              variant="contained" 
              color="error"
              startIcon={isLoading ? null : <CancelIcon />}
              sx={{
                backgroundColor: '#ef5350',  
                textTransform: 'none',  
                '&:hover': {
                  backgroundColor: '#e53935', 
                }
              }}
              onClick={handleReject}
              disabled={isLoading}
            >
              {isLoading ? <CircularProgress size={24} color="inherit" /> : 'Reject'}
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
            disabled={isLoading}
          >
            {isLoading ? <CircularProgress size={24} color="inherit" /> : 'Submit'}
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
              {responseMessage}
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
              {responseMessage}
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
        open={isLoading}
        maxWidth="sm"
      >
        <DialogContent sx={{ pt: 3, pb: 3, minWidth: '250px' }}>
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 2 }}>
            <CircularProgress size={30} style={{ color: '#450839' }} />
            <Typography>
              Processing your request...
            </Typography>
          </Box>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default FeatureAdditionApproval;
