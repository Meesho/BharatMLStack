import React from 'react';
import {
  Box, Typography, Chip,
  Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, Alert,
} from '@mui/material';

const changeTypeColor = {
  added: 'success',
  modified: 'info',
  removed: 'error',
};

const necessityColor = {
  active: 'success',
  transient: 'warning',
  skipped: 'default',
};

const PlanView = ({ plan }) => {
  if (!plan) return null;

  const changes = plan.changes || [];
  const necessityChanges = plan.necessity_changes || {};
  const directImpact = plan.direct_impact || [];
  const cascadeImpact = plan.cascade_impact || [];
  const allImpact = [...directImpact, ...cascadeImpact];
  const warnings = plan.warnings || [];
  const airflowDags = plan.airflow_dags || [];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      {changes.length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>Changes</Typography>
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Type</TableCell>
                  <TableCell>Asset Name</TableCell>
                  <TableCell>Details</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {changes.map((change, idx) => (
                  <TableRow key={idx}>
                    <TableCell>
                      <Chip
                        label={change.change_type?.toUpperCase()}
                        color={changeTypeColor[change.change_type] || 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{change.asset_name}</TableCell>
                    <TableCell>
                      {change.code_changed && <Chip label="Code Changed" size="small" variant="outlined" sx={{ mr: 0.5 }} />}
                      {change.inputs_added?.map((inp, i) => (
                        <Chip key={`add-${i}`} label={`+${inp}`} size="small" color="success" variant="outlined" sx={{ mr: 0.5, mb: 0.5 }} />
                      ))}
                      {change.inputs_removed?.map((inp, i) => (
                        <Chip key={`rm-${i}`} label={`-${inp}`} size="small" color="error" variant="outlined" sx={{ mr: 0.5, mb: 0.5 }} />
                      ))}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}

      {Object.keys(necessityChanges).length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>Necessity Changes</Typography>
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Asset Name</TableCell>
                  <TableCell>Before</TableCell>
                  <TableCell>After</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {Object.entries(necessityChanges).map(([asset, [before, after]]) => (
                  <TableRow key={asset}>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{asset}</TableCell>
                    <TableCell>
                      {before ? (
                        <Chip label={before.toUpperCase()} color={necessityColor[before] || 'default'} size="small" />
                      ) : (
                        <Typography variant="body2" color="text.secondary">(new)</Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      <Chip label={after.toUpperCase()} color={necessityColor[after] || 'default'} size="small" />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}

      {allImpact.length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>Recomputation Impact</Typography>
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Asset Name</TableCell>
                  <TableCell>Partitions</TableCell>
                  <TableCell>Reason</TableCell>
                  <TableCell>Est. Cost</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {allImpact.map((impact, idx) => (
                  <TableRow key={idx}>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{impact.asset_name}</TableCell>
                    <TableCell>{impact.partitions_affected}</TableCell>
                    <TableCell>{impact.reason}</TableCell>
                    <TableCell>
                      {impact.estimated_cost_usd != null
                        ? `$${impact.estimated_cost_usd.toFixed(2)}`
                        : '-'}
                    </TableCell>
                  </TableRow>
                ))}
                {plan.total_partitions > 0 && (
                  <TableRow>
                    <TableCell sx={{ fontWeight: 'bold' }}>TOTAL</TableCell>
                    <TableCell sx={{ fontWeight: 'bold' }}>{plan.total_partitions}</TableCell>
                    <TableCell />
                    <TableCell sx={{ fontWeight: 'bold' }}>
                      {plan.total_estimated_cost_usd != null
                        ? `$${plan.total_estimated_cost_usd.toFixed(2)}`
                        : '-'}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}

      {airflowDags.length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>Airflow DAGs (to be generated)</Typography>
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
            {airflowDags.map((dag, idx) => (
              <Chip key={idx} label={dag} variant="outlined" sx={{ fontFamily: 'monospace' }} />
            ))}
          </Box>
        </Box>
      )}

      {warnings.length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>Warnings</Typography>
          {warnings.map((warning, idx) => (
            <Alert key={idx} severity="warning" sx={{ mb: 1 }}>{warning}</Alert>
          ))}
        </Box>
      )}
    </Box>
  );
};

export default PlanView;
