import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, CircularProgress, Alert,
  Card, CardContent, CardActions, Button,
  Select, MenuItem, FormControl, InputLabel,
  Pagination, IconButton, Tooltip,
} from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import RefreshIcon from '@mui/icons-material/Refresh';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../Auth/AuthContext';
import * as URL_CONSTANTS from '../../config';
import StatusBadge from './components/StatusBadge';

const ReviewList = () => {
  const [reviews, setReviews] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [repoFilter, setRepoFilter] = useState('');
  const [page, setPage] = useState(1);
  const perPage = 20;

  const { user } = useAuth();
  const navigate = useNavigate();

  const fetchReviews = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (statusFilter) params.set('status', statusFilter);
      if (repoFilter) params.set('repo', repoFilter);
      params.set('page', page.toString());
      params.set('per_page', perPage.toString());

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews?${params}`,
        {
          headers: { 'Authorization': `Bearer ${user.token}` },
        }
      );
      if (!response.ok) {
        throw new Error(`Failed to fetch reviews: ${response.status}`);
      }
      const data = await response.json();
      setReviews(data.reviews || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [user.token, statusFilter, repoFilter, page]);

  useEffect(() => {
    fetchReviews();
  }, [fetchReviews]);

  const repos = [...new Set(reviews.map(r => r.repo))];
  const totalPages = Math.ceil(total / perPage);

  const formatAge = (dateStr) => {
    const diff = Date.now() - new Date(dateStr).getTime();
    const hours = Math.floor(diff / 3600000);
    if (hours < 1) return 'just now';
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ago`;
  };

  if (loading && reviews.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h5">Feature Reviews</Typography>
        <Tooltip title="Refresh">
          <IconButton onClick={fetchReviews} disabled={loading}>
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
        <FormControl size="small" sx={{ minWidth: 180 }}>
          <InputLabel>Status</InputLabel>
          <Select
            value={statusFilter}
            label="Status"
            onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
          >
            <MenuItem value="">All</MenuItem>
            <MenuItem value="pending">Pending</MenuItem>
            <MenuItem value="plan_computed">Plan Computed</MenuItem>
            <MenuItem value="approved">Approved</MenuItem>
            <MenuItem value="rejected">Rejected</MenuItem>
            <MenuItem value="merged">Merged</MenuItem>
            <MenuItem value="closed">Closed</MenuItem>
          </Select>
        </FormControl>

        {repos.length > 1 && (
          <FormControl size="small" sx={{ minWidth: 250 }}>
            <InputLabel>Repository</InputLabel>
            <Select
              value={repoFilter}
              label="Repository"
              onChange={(e) => { setRepoFilter(e.target.value); setPage(1); }}
            >
              <MenuItem value="">All Repos</MenuItem>
              {repos.map(r => <MenuItem key={r} value={r}>{r}</MenuItem>)}
            </Select>
          </FormControl>
        )}
      </Box>

      {reviews.length === 0 && !loading && (
        <Alert severity="info">No feature reviews found.</Alert>
      )}

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {reviews.map(review => (
          <Card key={review.id} variant="outlined" sx={{ cursor: 'pointer', '&:hover': { boxShadow: 2 } }}>
            <CardContent
              onClick={() => navigate(`/feature-reviews/${review.id}`)}
              sx={{ pb: 1 }}
            >
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
                <Box>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                    PR #{review.pr_number} — {review.pr_title}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {review.repo} · {review.pr_author} · {formatAge(review.created_at)}
                  </Typography>
                </Box>
                <StatusBadge status={review.status} />
              </Box>
              {review.asset_summary && (
                <Typography variant="body2" color="text.secondary">
                  {review.asset_summary.total} asset(s)
                  {review.asset_summary.new > 0 && ` · ${review.asset_summary.new} new`}
                  {review.asset_summary.modified > 0 && ` · ${review.asset_summary.modified} modified`}
                  {review.asset_summary.removed > 0 && ` · ${review.asset_summary.removed} removed`}
                </Typography>
              )}
            </CardContent>
            <CardActions sx={{ pt: 0, px: 2, pb: 1 }}>
              <Button
                size="small"
                onClick={() => navigate(`/feature-reviews/${review.id}`)}
              >
                {review.status === 'pending' ? 'Show Plan' : 'View Details'}
              </Button>
              {review.pr_url && (
                <Button
                  size="small"
                  endIcon={<OpenInNewIcon fontSize="small" />}
                  onClick={(e) => { e.stopPropagation(); window.open(review.pr_url, '_blank', 'noopener'); }}
                >
                  View PR
                </Button>
              )}
            </CardActions>
          </Card>
        ))}
      </Box>

      {totalPages > 1 && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 3 }}>
          <Pagination
            count={totalPages}
            page={page}
            onChange={(_, v) => setPage(v)}
            color="primary"
          />
        </Box>
      )}
    </Box>
  );
};

export default ReviewList;
