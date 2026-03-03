import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, CircularProgress, Alert, TextField,
  Table, TableBody, TableCell, TableContainer, TableHead,
  TableRow, Paper, TableSortLabel, Select, MenuItem,
  FormControl, InputLabel, Pagination, IconButton, Tooltip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../Auth/AuthContext';
import * as URL_CONSTANTS from '../../config';
import StatusBadge from './components/StatusBadge';

const ReviewHistory = () => {
  const [reviews, setReviews] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [page, setPage] = useState(1);
  const [orderBy, setOrderBy] = useState('created_at');
  const [orderDir, setOrderDir] = useState('desc');
  const perPage = 20;

  const { user } = useAuth();
  const navigate = useNavigate();

  const fetchReviews = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (statusFilter) params.set('status', statusFilter);
      params.set('page', page.toString());
      params.set('per_page', perPage.toString());

      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/feature-reviews?${params}`,
        {
          headers: { 'Authorization': `Bearer ${user.token}` },
        }
      );
      if (!response.ok) throw new Error(`Failed to fetch: ${response.status}`);
      const data = await response.json();
      setReviews(data.reviews || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [user.token, statusFilter, page]);

  useEffect(() => {
    fetchReviews();
  }, [fetchReviews]);

  const handleSort = (column) => {
    if (orderBy === column) {
      setOrderDir(prev => prev === 'asc' ? 'desc' : 'asc');
    } else {
      setOrderBy(column);
      setOrderDir('desc');
    }
  };

  const filteredReviews = searchTerm
    ? reviews.filter(r =>
        (r.pr_title || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        (r.pr_author || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        (r.repo || '').toLowerCase().includes(searchTerm.toLowerCase())
      )
    : reviews;

  const sortedReviews = [...filteredReviews].sort((a, b) => {
    let aVal = a[orderBy];
    let bVal = b[orderBy];
    if (orderBy === 'created_at' || orderBy === 'reviewed_at') {
      aVal = new Date(aVal || 0).getTime();
      bVal = new Date(bVal || 0).getTime();
    }
    if (aVal < bVal) return orderDir === 'asc' ? -1 : 1;
    if (aVal > bVal) return orderDir === 'asc' ? 1 : -1;
    return 0;
  });

  const totalPages = Math.ceil(total / perPage);

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
        <Typography variant="h5">Review History</Typography>
        <Tooltip title="Refresh">
          <IconButton onClick={fetchReviews} disabled={loading}>
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
        <TextField
          label="Search"
          size="small"
          placeholder="PR title, author, repo..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ minWidth: 250 }}
        />
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
      </Box>

      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>
                <TableSortLabel
                  active={orderBy === 'pr_number'}
                  direction={orderBy === 'pr_number' ? orderDir : 'asc'}
                  onClick={() => handleSort('pr_number')}
                >
                  PR
                </TableSortLabel>
              </TableCell>
              <TableCell>Title</TableCell>
              <TableCell>
                <TableSortLabel
                  active={orderBy === 'pr_author'}
                  direction={orderBy === 'pr_author' ? orderDir : 'asc'}
                  onClick={() => handleSort('pr_author')}
                >
                  Author
                </TableSortLabel>
              </TableCell>
              <TableCell>Repo</TableCell>
              <TableCell>
                <TableSortLabel
                  active={orderBy === 'status'}
                  direction={orderBy === 'status' ? orderDir : 'asc'}
                  onClick={() => handleSort('status')}
                >
                  Status
                </TableSortLabel>
              </TableCell>
              <TableCell>Reviewed By</TableCell>
              <TableCell>
                <TableSortLabel
                  active={orderBy === 'created_at'}
                  direction={orderBy === 'created_at' ? orderDir : 'asc'}
                  onClick={() => handleSort('created_at')}
                >
                  Date
                </TableSortLabel>
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {sortedReviews.map(review => (
              <TableRow
                key={review.id}
                hover
                sx={{ cursor: 'pointer' }}
                onClick={() => navigate(`/feature-reviews/${review.id}`)}
              >
                <TableCell>#{review.pr_number}</TableCell>
                <TableCell sx={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {review.pr_title}
                </TableCell>
                <TableCell>{review.pr_author}</TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.85em' }}>{review.repo}</TableCell>
                <TableCell><StatusBadge status={review.status} /></TableCell>
                <TableCell>{review.reviewed_by || '-'}</TableCell>
                <TableCell>{new Date(review.created_at).toLocaleDateString()}</TableCell>
              </TableRow>
            ))}
            {sortedReviews.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  <Typography color="text.secondary" sx={{ py: 2 }}>No reviews found</Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

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

export default ReviewHistory;
