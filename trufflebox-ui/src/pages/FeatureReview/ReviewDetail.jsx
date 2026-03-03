import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, CircularProgress, Alert, Button, TextField,
  Card, CardContent, Divider, Dialog, DialogTitle, DialogContent,
  DialogActions,
} from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import MergeIcon from '@mui/icons-material/CallMerge';
import RefreshIcon from '@mui/icons-material/Refresh';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../Auth/AuthContext';
import * as URL_CONSTANTS from '../../config';
import StatusBadge from './components/StatusBadge';
import PlanView from './components/PlanView';

const ReviewDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [review, setReview] = useState(null);
  const [plan, setPlan] = useState(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState(null);
  const [successMsg, setSuccessMsg] = useState(null);

  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectComment, setRejectComment] = useState('');
  const [approveComment, setApproveComment] = useState('');

  const fetchReview = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews/${id}`,
        { headers: { 'Authorization': `Bearer ${user.token}` } }
      );
      if (!response.ok) throw new Error(`Failed to fetch review: ${response.status}`);
      const data = await response.json();
      setReview(data);

      if (data.plan_json) {
        try { setPlan(JSON.parse(data.plan_json)); } catch { setPlan(null); }
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [id, user.token]);

  useEffect(() => {
    fetchReview();
  }, [fetchReview]);

  const handleComputePlan = async () => {
    setActionLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews/${id}/compute-plan`,
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.error || `Failed: ${response.status}`);
      }
      const data = await response.json();
      setPlan(data.plan);
      setReview(prev => ({ ...prev, status: 'plan_computed' }));
      setSuccessMsg('Plan computed successfully');
    } catch (err) {
      setError(err.message);
    } finally {
      setActionLoading(false);
    }
  };

  const handleApprove = async () => {
    setActionLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews/${id}/approve`,
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
            'X-User-Email': user.email || '',
          },
          body: JSON.stringify({ comment: approveComment }),
        }
      );
      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.error || `Failed: ${response.status}`);
      }
      setSuccessMsg('Review approved and PR approved on GitHub');
      fetchReview();
    } catch (err) {
      setError(err.message);
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async () => {
    if (!rejectComment.trim()) {
      setError('Comment is required for rejection');
      return;
    }
    setActionLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews/${id}/reject`,
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
            'X-User-Email': user.email || '',
          },
          body: JSON.stringify({ comment: rejectComment }),
        }
      );
      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.error || `Failed: ${response.status}`);
      }
      setRejectDialogOpen(false);
      setRejectComment('');
      setSuccessMsg('Review rejected');
      fetchReview();
    } catch (err) {
      setError(err.message);
    } finally {
      setActionLoading(false);
    }
  };

  const handleMerge = async () => {
    setActionLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews/${id}/merge`,
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
            'X-User-Email': user.email || '',
          },
        }
      );
      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.error || `Failed: ${response.status}`);
      }
      setSuccessMsg('PR merge initiated');
      fetchReview();
    } catch (err) {
      setError(err.message);
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!review) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">Review not found</Alert>
        <Button sx={{ mt: 2 }} onClick={() => navigate('/feature-reviews')}>Back to Reviews</Button>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Button variant="text" onClick={() => navigate('/feature-reviews')} sx={{ mb: 2 }}>
        ← Back to Reviews
      </Button>

      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      {successMsg && <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccessMsg(null)}>{successMsg}</Alert>}

      {/* Section A: PR Info */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <Box>
              <Typography variant="h6">
                PR #{review.pr_number} — {review.pr_title}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                repo: {review.repo} &nbsp;·&nbsp; author: {review.pr_author} &nbsp;·&nbsp;
                sha: {review.head_sha?.substring(0, 8)}
              </Typography>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <StatusBadge status={review.status} />
              {review.pr_url && (
                <Button
                  size="small"
                  endIcon={<OpenInNewIcon fontSize="small" />}
                  onClick={() => window.open(review.pr_url, '_blank', 'noopener')}
                >
                  View PR
                </Button>
              )}
            </Box>
          </Box>
        </CardContent>
      </Card>

      {/* Section B: Plan */}
      {plan && (
        <Card variant="outlined" sx={{ mb: 3 }}>
          <CardContent>
            <Typography variant="h6" gutterBottom>Impact Analysis</Typography>
            <PlanView plan={plan} />
          </CardContent>
        </Card>
      )}

      {/* Section C: Actions */}
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" gutterBottom>Actions</Typography>
          <Divider sx={{ mb: 2 }} />

          {review.status === 'pending' && (
            <Button
              variant="contained"
              onClick={handleComputePlan}
              disabled={actionLoading}
              startIcon={actionLoading ? <CircularProgress size={16} /> : <RefreshIcon />}
            >
              Show Plan
            </Button>
          )}

          {review.status === 'plan_computed' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField
                label="Comment (optional)"
                size="small"
                value={approveComment}
                onChange={(e) => setApproveComment(e.target.value)}
                fullWidth
                multiline
                minRows={2}
              />
              <Box sx={{ display: 'flex', gap: 2 }}>
                <Button
                  variant="contained"
                  color="success"
                  onClick={handleApprove}
                  disabled={actionLoading}
                  startIcon={<CheckCircleIcon />}
                >
                  Approve
                </Button>
                <Button
                  variant="contained"
                  color="error"
                  onClick={() => setRejectDialogOpen(true)}
                  disabled={actionLoading}
                  startIcon={<CancelIcon />}
                >
                  Reject
                </Button>
                <Button
                  variant="outlined"
                  onClick={handleComputePlan}
                  disabled={actionLoading}
                  startIcon={<RefreshIcon />}
                >
                  Recompute Plan
                </Button>
              </Box>
            </Box>
          )}

          {review.status === 'approved' && (
            <Box>
              <Typography variant="body2" sx={{ mb: 2 }}>
                Approved by {review.reviewed_by}
                {review.reviewed_at && ` on ${new Date(review.reviewed_at).toLocaleString()}`}
              </Typography>
              <Button
                variant="contained"
                onClick={handleMerge}
                disabled={actionLoading}
                startIcon={<MergeIcon />}
              >
                Merge
              </Button>
            </Box>
          )}

          {review.status === 'rejected' && (
            <Box>
              <Alert severity="error" sx={{ mb: 2 }}>
                Rejected by {review.reviewed_by}: {review.review_comment}
              </Alert>
              <Button
                variant="outlined"
                onClick={handleComputePlan}
                disabled={actionLoading}
                startIcon={<RefreshIcon />}
              >
                Recompute Plan
              </Button>
            </Box>
          )}

          {review.status === 'merged' && (
            <Box>
              <Alert severity="success" icon={<CheckCircleIcon />}>
                Merged by {review.merged_by}
                {review.merged_at && ` on ${new Date(review.merged_at).toLocaleString()}`}
              </Alert>
            </Box>
          )}

          {review.status === 'closed' && (
            <Alert severity="info">PR was closed without merging.</Alert>
          )}
        </CardContent>
      </Card>

      {/* Reject Dialog */}
      <Dialog open={rejectDialogOpen} onClose={() => setRejectDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Reject Feature Review</DialogTitle>
        <DialogContent>
          <TextField
            label="Rejection reason (required)"
            value={rejectComment}
            onChange={(e) => setRejectComment(e.target.value)}
            fullWidth
            multiline
            minRows={3}
            sx={{ mt: 1 }}
            required
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRejectDialogOpen(false)}>Cancel</Button>
          <Button
            onClick={handleReject}
            color="error"
            variant="contained"
            disabled={!rejectComment.trim() || actionLoading}
          >
            Reject
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ReviewDetail;
