import React, { useState, useEffect } from 'react';
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

const EntityApproval = () => {
  const [showModal, setShowModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [entityData, setEntityData] = useState(getInitialEntityData());
  const [entityRequests, setEntityRequests] = useState([]);
  const { user } = useAuth();
  
  // Add new state variables
  const [isLoading, setIsLoading] = useState(false);
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [resultMessage, setResultMessage] = useState('');

  const handleEntityOpen = (entity = null) => {
    setEntityData(transformEntityData(entity));
    setShowModal(true);
  };

  const transformEntityData = (entity) => {
    if (!entity) return getInitialEntityData();
    
    var data = JSON.parse(entity.Payload)
    console.log("Entity Data:", entity.RequestId);
    console.log("Original Data:", data);
    return {
      "request-id": entity.RequestId || "",
      "entity-label": data["entity-label"] || "",
      "status": entity.Status || "",
      "request-type": entity.RequestType || "CREATE",
      "reject-reason": entity.RejectReason || "N/A",
      "key-map": Object.values(data["key-map"] || {}).map((item) => ({
        sequence: item.sequence || "",
        "entity-label": item["entity-label"] || "",
        "column-label": item["column-label"] || "",
      })),
      "distributed-cache": {
        enabled: data["distributed-cache"]?.enabled || "true",
        "conf-id": data["distributed-cache"]["conf-id"] !== undefined 
        ? data["distributed-cache"]["conf-id"] 
        : "",
        "ttl-in-seconds": data["distributed-cache"]["ttl-in-seconds"],
        "jitter-percentage": data["distributed-cache"]["jitter-percentage"],
      },
      "in-memory-cache": {
        enabled: data["in-memory-cache"]?.enabled || "false",
        "conf-id": data["in-memory-cache"]["conf-id"] !== undefined 
        ? data["in-memory-cache"]["conf-id"] 
        : "",
        "ttl-in-seconds": data["in-memory-cache"]["ttl-in-seconds"],
        "jitter-percentage": data["in-memory-cache"]["jitter-percentage"],
      },
    };
  };

  const handleClose = () => {
    setShowModal(false);
    // Reset the entity data when closing the modal
    setEntityData(getInitialEntityData());
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
        "request-id": entityData["request-id"],
        status: status.toUpperCase(),
        "reject-reason": status.toUpperCase() === 'REJECTED' ? rejectReason : '',
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/process-entity`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(requestData),
      });

      const result = await response.json();
      if (response.ok) {
        setResultMessage(result.message);
        setShowSuccessModal(true);
      } else {
        setResultMessage(result.error);
        setShowErrorModal(true);
      }
      handleClose();
      setShowRejectModal(false);
      fetchEntityRequests(); // Refresh data
    } catch (error) {
      setResultMessage('Network error. Please try again.');
      setShowErrorModal(true);
    } finally {
      setIsLoading(false);
    }
  };

  // Extract fetching logic to reusable function
  const fetchEntityRequests = async () => {
    setIsLoading(true);
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/get-entity-requests`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      const data = await response.json();
      setEntityRequests(data);
    } catch (error) {
      console.error('Error fetching data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchEntityRequests();
  }, [user.token]);

  // Check if status is APPROVED or REJECTED
  const shouldShowFooter = showModal && entityData.status !== 'APPROVED' && entityData.status !== 'REJECTED';

  return (
    <div>
      <GenericTable 
        data={entityRequests}
        excludeColumns={['FeatureGroupLabel']}
        onRowAction={handleEntityOpen} 
        loading={isLoading} 
      />
      
      <Modal show={showModal} onHide={handleClose} size="lg" centered>
        <Modal.Header closeButton>
          <Modal.Title style={{ fontFamily: "system-ui" }}>Entity Details</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <ListGroup>
            <ListGroup.Item>
              <strong>Request ID:</strong> {entityData["request-id"]}
            </ListGroup.Item>
            
            <ListGroup.Item>
              <strong>Request Type:</strong> {entityData["request-type"]}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Entity Label:</strong> {entityData["entity-label"]}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Status:</strong> {entityData.status}
            </ListGroup.Item>
            
            {entityData.status === 'REJECTED' && entityData["reject-reason"] && (
              <ListGroup.Item>
                <strong>Reject Reason:</strong> {entityData["reject-reason"]}
              </ListGroup.Item>
            )}

            {entityData["request-type"] !== "EDIT" && (
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
                    {entityData["key-map"].map((key, index) => (
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
                    <td>{entityData["distributed-cache"].enabled}</td>
                  </tr>
                  <tr>
                    <td>Config ID</td>
                    <td>{entityData["distributed-cache"]["conf-id"]}</td>
                  </tr>
                  <tr>
                    <td>TTL (seconds)</td>
                    <td>{entityData["distributed-cache"]["ttl-in-seconds"]}</td>
                  </tr>
                  <tr>
                    <td>Jitter Percentage</td>
                    <td>{entityData["distributed-cache"]["jitter-percentage"]}</td>
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
                    <td>{entityData["in-memory-cache"].enabled}</td>
                  </tr>
                  <tr>
                    <td>Config ID</td>
                    <td>{entityData["in-memory-cache"]["conf-id"]}</td>
                  </tr>
                  <tr>
                    <td>TTL (seconds)</td>
                    <td>{entityData["in-memory-cache"]["ttl-in-seconds"]}</td>
                  </tr>
                  <tr>
                    <td>Jitter Percentage</td>
                    <td>{entityData["in-memory-cache"]["jitter-percentage"]}</td>
                  </tr>
                </tbody>
              </Table>
            </ListGroup.Item>
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
              {resultMessage}
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
              {resultMessage}
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

const getInitialEntityData = () => ({
  'request-id': '',
  'entity-label': '',
  'status': '',
  'request-type': 'CREATE',
  'reject-reason': '',
  'key-map': [{ sequence: '', 'entity-label': '', 'column-label': '' }],
  'distributed-cache': { enabled: '', 'conf-id': '', 'ttl-in-seconds': '', 'jitter-percentage': '' },
  'in-memory-cache': { enabled: '', 'conf-id': '', 'ttl-in-seconds': '', 'jitter-percentage': '' },
});

export default EntityApproval;
