import React, { useState, useEffect } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Box,
  Typography
} from '@mui/material';
import { Modal, ListGroup } from 'react-bootstrap';
import "./styles.scss";
import GenericTable from '../../common/GenericTable';
import { useAuth } from '../../../Auth/AuthContext';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

import * as URL_CONSTANTS from '../../../../config';

const JobRegistry = () => {
  const [open, setOpen] = useState(false);
  const [showViewModal, setShowViewModal] = useState(false);
  const [selectedJob, setSelectedJob] = useState(null);
  const [creationType, setCreationType] = useState('job'); // 'job' or 'client'
  const [jobData, setJobData] = useState({
    "job-type": "writer",
    "job-id": "",
    "token": "",
  });
  const [jobRequests, setJobRequests] = useState([]);
  const { user } = useAuth();
  
  // Modal states
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [modalMessage, setModalMessage] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

  // Configuration for different creation types
  const creationConfig = {
    job: {
      title: 'Create Job',
      nameLabel: 'Job Name',
      tokenLabel: 'Job Token',
      jobType: 'writer'
    },
    client: {
      title: 'Create Client',
      nameLabel: 'Client Name',
      tokenLabel: 'Client Token',
      jobType: 'reader'
    }
  };

  useEffect(() => {
    const fetchJobRequests = async () => {
      try {
        const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/get-job-requests`, {
          headers: {
            Authorization: `Bearer ${user.token}`,
          },
        });

        if (!response.ok) {
          console.error('Failed to fetch job requests:', response.statusText);
          return;
        }

        const data = await response.json();
        setJobRequests(data);
      } catch (error) {
        console.error('Error fetching job requests:', error);
      }
    };

    fetchJobRequests();
  }, [user.token]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    const trimmedValue = typeof value === 'string' ? value.trim() : value;
    setJobData((prevData) => ({
      ...prevData,
      [name]: trimmedValue,
    }));
  };

  const handleOpen = (type = 'job') => {
    const config = creationConfig[type];
    setCreationType(type);
    setJobData({
      "job-type": config.jobType,
      "job-id": "",
      "token": "",
    });
    setValidationErrors({});
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
    setCreationType('job');
  };

  const handleViewJob = (job) => {
    const payload = JSON.parse(job.Payload);
    setSelectedJob({
      ...payload,
      Status: job.Status,
      RejectReason: job.RejectReason
    });
    setShowViewModal(true);
  };

  const closeViewModal = () => {
    setShowViewModal(false);
    setSelectedJob(null);
  };

  const validateForm = () => {
    const errors = {};
    
    if (!jobData["job-id"] || jobData["job-id"].trim() === "") {
      errors["job-id"] = `${currentConfig.nameLabel} is required`;
    }
    
    if (!jobData.token || jobData.token.trim() === "") {
      errors.token = `${currentConfig.tokenLabel} is required`;
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

    const apiEndpoint = `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/online-feature-store/register-job`;
    try {
      const response = await fetch(apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify(jobData),
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

  const currentConfig = creationConfig[creationType];

  return (
    <div style={{ padding: '20px' }}>
      <GenericTable
        data={jobRequests}
        excludeColumns={['EntityLabel', 'FeatureGroupLabel']}
        onRowAction={(row) => handleViewJob(row)}
        loading={false}
        actionButtons={[
          {
            label: "Create Job",
            onClick: () => handleOpen('job'),
            variant: "contained",
            color: "#522b4a",
            hoverColor: "#613a5c"
          },
          {
            label: "Create Client",
            onClick: () => handleOpen('client'),
            variant: "contained",
            color: "#522b4a",
            hoverColor: "#613a5c"
          }
        ]}
      />

      {/* Create Job/Client Modal */}
      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle>{currentConfig.title}</DialogTitle>
        <DialogContent>
          <TextField
            label={`${currentConfig.nameLabel} *`}
            name="job-id"
            value={jobData["job-id"]}
            onChange={handleChange}
            fullWidth
            margin="normal"
            error={!!validationErrors["job-id"]}
            helperText={validationErrors["job-id"]}
          />
          <TextField
            label={`${currentConfig.tokenLabel} *`}
            name="token"
            value={jobData.token}
            onChange={handleChange}
            fullWidth
            margin="normal"
            error={!!validationErrors.token}
            helperText={validationErrors.token}
          />
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

      {/* View Job Modal */}
      {selectedJob && (
        <Modal show={showViewModal} onHide={closeViewModal} size="lg" centered>
          <Modal.Header closeButton>
            <Modal.Title>Job Details</Modal.Title>
          </Modal.Header>
          <Modal.Body>
            <ListGroup>
              <ListGroup.Item>
                <strong>Job Type:</strong> {selectedJob["job-type"]}
              </ListGroup.Item>
              <ListGroup.Item>
                <strong>Job Name:</strong> {selectedJob["job-id"]}
              </ListGroup.Item>
              <ListGroup.Item>
                <strong>Job Token:</strong> {selectedJob.token}
              </ListGroup.Item>
              {selectedJob.Status === "REJECTED" && (
                <ListGroup.Item>
                  <strong>Reject Reason:</strong> {selectedJob.RejectReason}
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

export default JobRegistry;
