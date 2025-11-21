import React, { useState, useEffect, useCallback } from 'react';
import { Modal, ListGroup } from 'react-bootstrap';
import { 
  Button, 
  Dialog, 
  DialogActions, 
  DialogContent, 
  DialogTitle, 
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

const StoreApproval = () => {
  const [showModal, setShowModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [loading, setLoading] = useState(false);
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [responseMessage, setResponseMessage] = useState('');
  const [storeData, setStoreData] = useState({
    "request-id": "",
    "conf-id": "",
    "db-type": "",
    "table": "",
    "table-ttl": "",
    "primary-keys": [],
    "status": "",
    "reject-reason": "",
  });
  const [storeRequests, setStoreRequests] = useState([]);
  const { user } = useAuth();

  const fetchStoreRequests = useCallback(async () => {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-store-requests`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      if (!response.ok) {
        throw new Error('Failed to fetch store requests');
      }
      const data = await response.json();
      setStoreRequests(data);
    } catch (error) {
      console.error('Error fetching store requests:', error);
    }
  }, [user.token]);

  useEffect(() => {
    fetchStoreRequests();
  }, [fetchStoreRequests]);

  const handleOpen = useCallback((store = null) => {
    setStoreData({
      "request-id": "",
      "conf-id": "",
      "db-type": "",
      "table": "",
      "table-ttl": "",
      "primary-keys": [],
      "status": "",
      "reject-reason": "",
    });
    if (store) {
      const parsedData = JSON.parse(store.Payload);
      parsedData["request-id"] = store.RequestId;
      parsedData["status"] = store.Status;
      parsedData["reject-reason"] = store.RejectReason || "";
      setStoreData(parsedData);
    }
    setShowModal(true);
  }, []);

  const handleClose = useCallback(() => {
    setStoreData({
      "request-id": "",
      "conf-id": "",
      "db-type": "",
      "table": "",
      "table-ttl": "",
      "primary-keys": [],
      "status": "",
      "reject-reason": "",
    });
    setShowModal(false);
    setRejectReason('');
  }, []);

  const handleReject = useCallback(() => {
    setShowModal(false);
    setShowRejectModal(true);
  }, []);

  const handleRejectModalClose = useCallback(() => {
    setShowRejectModal(false);
    setRejectReason('');
  }, []);

  const handleSubmit = useCallback(async (event, status) => {
    event.preventDefault();
    setLoading(true);
    try {
      const requestData = {
        "request-id": storeData["request-id"],
        status: status.toUpperCase(),
        "reject-reason": status.toUpperCase() === 'REJECTED' ? rejectReason : '',
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/process-store`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(requestData),
      });

      const result = await response.json();
      if (!response.ok) console.log(result.error || 'Failed to process store');
      
      setResponseMessage(result.message);
      setShowSuccessModal(true);

      fetchStoreRequests();
      
      handleClose();
      handleRejectModalClose();
    } catch (error) {
      setResponseMessage(error.message);
      setShowErrorModal(true);
    } finally {
      setLoading(false);
    }
  }, [storeData, user.token, handleClose, rejectReason, handleRejectModalClose, fetchStoreRequests]);

  // Check if status is APPROVED or REJECTED
  const shouldShowFooter = showModal && storeData.status !== 'APPROVED' && storeData.status !== 'REJECTED';

  return (
    <div>
      <GenericTable
        data={storeRequests}
        excludeColumns={['EntityLabel', 'FeatureGroupLabel']}
        onRowAction={handleOpen}
        loading={false}
        flowType="approval"
      />
      
      <Modal
        show={showModal} 
        onHide={handleClose} 
        size="lg" 
        centered
        dialogClassName="custom-modal-width"
      >
        <Modal.Header closeButton>
          <Modal.Title style={{ fontFamily: "system-ui" }}>Store Details</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <ListGroup>
            <ListGroup.Item>
              <strong>Request ID:</strong> {storeData["request-id"] || 'Not Set'}
            </ListGroup.Item>
            
            <ListGroup.Item>
              <strong>Config ID:</strong> {storeData["conf-id"] || 'Not Set'}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Database Type:</strong> {storeData["db-type"] || 'Not Set'}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Table:</strong> {storeData.table || 'Not Set'}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Table Time to Live:</strong> {storeData["table-ttl"] || 'Not Set'}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Primary Keys:</strong> {storeData["primary-keys"]?.join(", ") || 'Not Set'}
            </ListGroup.Item>

            <ListGroup.Item>
              <strong>Status:</strong> {storeData.status}
            </ListGroup.Item>
            
            {storeData.status === 'REJECTED' && storeData["reject-reason"] && (
              <ListGroup.Item>
                <strong>Reject Reason:</strong> {storeData["reject-reason"]}
              </ListGroup.Item>
            )}
          </ListGroup>
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
            >
              Approve
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
            >
              Reject
            </Button>
          </Modal.Footer>
        )}
      </Modal>

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
              placeholder="Please provide a reason for rejecting this request (max 255 characters)"
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
          >
            Submit
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

      {/* Loading modal */}
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

export default StoreApproval;
