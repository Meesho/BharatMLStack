import React, { useState, useEffect, useCallback } from 'react';
import { Modal, ListGroup } from 'react-bootstrap';
import { 
  Button, 
  Dialog, 
  DialogTitle, 
  DialogContent, 
  DialogActions, 
  Box, 
  Typography, 
  CircularProgress,
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

const JobApproval = () => {
  const [showModal, setShowModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [jobData, setJobData] = useState({
    "request-id": "",
    "job-type": "",
    "job-id": "",
    "token": "",
    "status": "",
    "reject-reason": "",
  });
  const [jobRequests, setJobRequests] = useState([]);
  const { user } = useAuth();
  
  // Added states for modals and loading
  const [isLoading, setIsLoading] = useState(false);
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [responseMessage, setResponseMessage] = useState('');

  const fetchJobRequests = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-job-requests`, {
        headers: {
          'Authorization': `Bearer ${user.token}`,
        },
      });
      if (!response.ok) throw new Error('Failed to fetch job requests');
      const data = await response.json();
      setJobRequests(data);
    } catch (error) {
      console.error('Error fetching job requests:', error);
    } finally {
      setIsLoading(false);
    }
  }, [user.token]);

  useEffect(() => {
    fetchJobRequests();
  }, [fetchJobRequests]);

  const handleOpen = (job = null) => {
    setJobData({
      "request-id": "",
      "job-type": "",
      "job-id": "",
      "token": "",
      "status": "",
      "reject-reason": "",
    });
    if (job) {
      const parsedData = JSON.parse(job.Payload);
      parsedData["request-id"] = job.RequestId;
      parsedData["status"] = job.Status;
      parsedData["reject-reason"] = job.RejectReason || "N/A";
      setJobData(parsedData);
    }
    setShowModal(true);
  };

  const handleClose = () => {
    setJobData({
      "request-id": "",
      "job-type": "",
      "job-id": "",
      "token": "",
      "status": "",
      "reject-reason": "",
    });
    setShowModal(false);
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
        "request-id": jobData["request-id"],
        status: status.toUpperCase(),
        "reject-reason": status.toUpperCase() === 'REJECTED' ? rejectReason : '',
      };

      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/process-job`, {
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
        await fetchJobRequests();
      } else {
        setResponseMessage(result.error || 'Failed to process job');
        setShowErrorModal(true);
      }
      
      handleClose();
      setShowRejectModal(false);
    } catch (error) {
      setResponseMessage('Network error. Please try again.');
      setShowErrorModal(true);
    } finally {
      setIsLoading(false);
    }
  };

  // Check if status is APPROVED or REJECTED
  const shouldShowFooter = showModal && jobData.status !== 'APPROVED' && jobData.status !== 'REJECTED';

  return (
    <div>
      <GenericTable 
        data={jobRequests} 
        excludeColumns={['EntityLabel','FeatureGroupLabel']} 
        onRowAction={handleOpen} 
        loading={isLoading} 
      />
      
      <Modal
        show={showModal} 
        onHide={handleClose} 
        size="lg" 
        centered
        dialogClassName="custom-modal-width"
      >
        <Modal.Header closeButton>
          <Modal.Title style={{ fontFamily: "system-ui" }}>Job Details</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <ListGroup>
            <ListGroup.Item>
              <strong>Request ID:</strong> {jobData["request-id"]}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Job Type:</strong> {jobData["job-type"]}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Job Name:</strong> {jobData["job-id"]}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Job Token:</strong> {jobData.token}
            </ListGroup.Item>
            <ListGroup.Item>
              <strong>Status:</strong> {jobData.status}
            </ListGroup.Item>
            {jobData.status === 'REJECTED' && jobData["reject-reason"] && (
              <ListGroup.Item>
                <strong>Reject Reason:</strong> {jobData["reject-reason"]}
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
              disabled={isLoading}
              sx={{
                backgroundColor: '#66bb6a',  
                textTransform: 'none',  
                '&:hover': {
                  backgroundColor: '#4caf50', 
                }
              }}
              onClick={(e) => handleSubmit(e, 'APPROVED')}
            >
              {isLoading ? <CircularProgress size={24} color="inherit" /> : 'Approve'}
            </Button>
            <Button 
              variant="contained" 
              color="error"
              startIcon={isLoading ? null : <CancelIcon />}
              disabled={isLoading}
              sx={{
                backgroundColor: '#ef5350',  
                textTransform: 'none',  
                '&:hover': {
                  backgroundColor: '#e53935', 
                }
              }}
              onClick={handleReject}
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
            startIcon={isLoading ? null : null}
            disabled={isLoading}
            sx={{
              backgroundColor: '#ef5350',
              textTransform: 'none',
              '&:hover': {
                backgroundColor: '#e53935',
              }
            }}
            onClick={(e) => handleSubmit(e, 'REJECTED')}
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

export default JobApproval;
