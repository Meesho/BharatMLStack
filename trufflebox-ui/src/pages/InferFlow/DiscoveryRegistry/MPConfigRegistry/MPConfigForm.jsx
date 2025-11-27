import React, { useState } from 'react';
import {
  TextField,
  Button,
  Box,
  IconButton,
  Typography,
  Grid,
  Divider,
  Chip,
  Alert,
  Autocomplete,
  Tooltip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import RepeatIcon from '@mui/icons-material/Repeat';

// Repeat Dimension Input Component
const RepeatDimensionInput = ({ onAddRepeated }) => {
  const [dimensionValue, setDimensionValue] = useState('');
  const [count, setCount] = useState('');

  const handleAdd = () => {
    if (dimensionValue.trim() && count && Number(count) > 0) {
      onAddRepeated(dimensionValue.trim(), Number(count));
      setDimensionValue('');
      setCount('');
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && dimensionValue.trim() && count && Number(count) > 0) {
      handleAdd();
    }
  };

  return (
    <Box 
      sx={{ 
        mb: 1,
        p: 1.5,
        border: '1px solid #e0e0e0',
        borderRadius: 1,
        bgcolor: '#f9f9f9'
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        <RepeatIcon sx={{ fontSize: '18px', color: '#f57c00' }} />
        <Typography variant="caption" sx={{ color: '#666', fontWeight: 500 }}>
          Quick Add: Repeat Dimension
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
        <TextField
          size="small"
          placeholder="Dimension value"
          value={dimensionValue}
          onChange={(e) => setDimensionValue(e.target.value)}
          onKeyDown={handleKeyDown}
          sx={{ flex: 1, bgcolor: 'white' }}
          inputProps={{ 
            style: { fontSize: '0.875rem' }
          }}
        />
        <Typography variant="body2" sx={{ color: '#666', whiteSpace: 'nowrap' }}>
          ×
        </Typography>
        <TextField
          size="small"
          type="number"
          placeholder="Count"
          value={count}
          onChange={(e) => {
            const val = e.target.value;
            if (val === '' || (Number(val) > 0 && Number(val) <= 10000)) {
              setCount(val);
            }
          }}
          onKeyDown={handleKeyDown}
          sx={{ width: '100px', bgcolor: 'white' }}
          inputProps={{ 
            min: 1,
            max: 10000,
            style: { fontSize: '0.875rem' }
          }}
        />
        <Button
          size="small"
          variant="contained"
          onClick={handleAdd}
          disabled={!dimensionValue.trim() || !count || Number(count) <= 0}
          startIcon={<RepeatIcon />}
          sx={{
            bgcolor: '#f57c00',
            color: 'white',
            textTransform: 'capitalize',
            minWidth: 'auto',
            px: 2,
            '&:hover': {
              bgcolor: '#e65100',
            },
            '&.Mui-disabled': {
              bgcolor: '#ccc',
              color: '#999'
            }
          }}
        >
          Add
        </Button>
      </Box>
      <Typography variant="caption" sx={{ color: '#999', mt: 0.5, display: 'block', fontSize: '0.7rem' }}>
        Example: Enter "1" and "35" to add dimension "1" thirty-five times
      </Typography>
    </Box>
  );
};

const MPConfigForm = ({
  formData,
  isEditMode = false,
  isCloneMode = false,
  isAdmin = false,
  
  // Handlers
  handleBasicInfoChange,
  handleRankerChange,
  handleInputChange,
  handleOutputChange,
  handleReRankerChange,
  handleResponseChange,
  handleConfigMappingChange,
  
  // CRUD operations
  addRanker,
  removeRanker,
  addReRanker,
  removeReRanker,
  
  // Accordion state
  expandedRankers,
  handleRankerAccordionChange,
  expandedReRankers,
  handleReRankerAccordionChange,
  expandedFeatures,
  setExpandedFeatures,
  
  // Data lists
  modelsList,
  computeConfigs,
  expressionVariables,
  mpHosts,
  featureTypes,
  featureTypesLoading,
  
  // Helper functions
  parseDimensionsFromString,
  convertDimensionsToString,
  
  // Constants
  CALIBRATION_OPTIONS,
  DATA_TYPE_OPTIONS,
}) => {
  
  const configId = `${formData.real_estate}-${formData.tenant}-${formData.config_identifier}`;

  return (
    <Box>
      {/* Basic Information */}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Grid container spacing={2}>
          <Grid item xs={6}>
            <TextField
              fullWidth
              label="Real Estate"
              size="small"
              value={formData.real_estate}
              onChange={handleBasicInfoChange('real_estate')}
              disabled={isEditMode}
              InputProps={isEditMode ? {
                sx: {
                  bgcolor: '#f5f5f5',
                  '& .Mui-disabled': {
                    WebkitTextFillColor: '#666',
                  }
                }
              } : {}}
            />
          </Grid>
          <Grid item xs={6}>
            <TextField
              fullWidth
              label="Tenant"
              size="small"
              value={formData.tenant}
              onChange={handleBasicInfoChange('tenant')}
              disabled={isEditMode}
              InputProps={isEditMode ? {
                sx: {
                  bgcolor: '#f5f5f5',
                  '& .Mui-disabled': {
                    WebkitTextFillColor: '#666',
                  }
                }
              } : {}}
            />
          </Grid>
        </Grid>
        <TextField
          fullWidth
          label="Config Identifier"
          size="small"
          value={formData.config_identifier}
          onChange={handleBasicInfoChange('config_identifier')}
          disabled={isEditMode}
          InputProps={isEditMode ? {
            sx: {
              bgcolor: '#f5f5f5',
              '& .Mui-disabled': {
                WebkitTextFillColor: '#666',
              }
            }
          } : {}}
        />

        {/* MP Config ID Info */}
        <Alert 
          severity="info" 
          sx={{ 
            mt: 2,
            backgroundColor: '#e3f2fd',
            border: '1px solid #2196f3',
            '& .MuiAlert-icon': {
              color: '#1976d2'
            }
          }}
        >
          {isEditMode ? (
            <Typography variant="caption">
              InferFlow Config ID: <strong>{configId}</strong> (cannot be changed)
            </Typography>
          ) : (
            <>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                <strong>💡 InferFlow Config ID Generation:</strong> The above three fields (Real Estate, Tenant, Config Identifier) will be automatically combined using hyphens (-) to create your unique InferFlow Config ID.
              </Typography>
              <Typography variant="caption" sx={{ color: '#666', mt: 0.5, display: 'block' }}>
                Example: <code>fy-organic-nqd</code>
              </Typography>
            </>
          )}
        </Alert>
      </Box>

      <Divider sx={{ my: 3 }} />

      {/* Rankers Section */}
      <Box sx={{ mb: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6" sx={{ color: '#1976d2', fontWeight: 600 }}>Rankers (Predator)</Typography>
          <Button
            startIcon={<AddIcon />}
            onClick={addRanker}
            variant="contained"
            size="small"
            sx={{
              bgcolor: '#1976d2',
              color: 'white',
              textTransform: 'capitalize',
              '&:hover': {
                bgcolor: '#1565c0',
                boxShadow: '0 4px 8px rgba(25, 118, 210, 0.3)'
              }
            }}
          >
            Add Ranker
          </Button>
        </Box>

        {formData.rankers?.map((ranker, rankerIndex) => (
          <Accordion
            key={rankerIndex}
            expanded={expandedRankers.includes(rankerIndex)}
            onChange={() => handleRankerAccordionChange(rankerIndex)}
            sx={{
              mb: 2,
              border: '1px solid #e0e0e0',
              borderRadius: '8px !important',
              '&:before': { display: 'none' },
              boxShadow: expandedRankers.includes(rankerIndex) ? '0 2px 8px rgba(0,0,0,0.1)' : 'none'
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon />}
              sx={{
                bgcolor: '#f5f9ff',
                borderRadius: '8px',
                '&.Mui-expanded': {
                  borderBottomLeftRadius: 0,
                  borderBottomRightRadius: 0,
                },
                '& .MuiAccordionSummary-content': {
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  my: 1
                }
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, color: '#1976d2' }}>
                  Ranker {rankerIndex + 1}
                </Typography>
                {ranker.model_name && (
                  <Chip
                    label={ranker.model_name}
                    size="small"
                    sx={{ bgcolor: '#e3f2fd', color: '#1976d2', fontWeight: 500 }}
                  />
                )}
                {ranker.entity_id && ranker.entity_id.length > 0 && (
                  <Chip
                    label={`${ranker.entity_id.length} Entities`}
                    size="small"
                    variant="outlined"
                    sx={{ borderColor: '#1976d2', color: '#1976d2' }}
                  />
                )}
              </Box>
              <IconButton
                size="small"
                color="error"
                onClick={(e) => {
                  e.stopPropagation();
                  removeRanker(rankerIndex);
                }}
                sx={{
                  '&:hover': {
                    bgcolor: 'rgba(211, 47, 47, 0.1)',
                    transform: 'scale(1.1)'
                  }
                }}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 3, bgcolor: '#fafafa' }}>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Autocomplete
                    options={(modelsList || []).sort((a, b) => (a.model_name || '').localeCompare(b.model_name || ''))}
                    getOptionLabel={(option) => option.model_name || ''}
                    value={modelsList?.find(model => model.model_name === ranker.model_name) || null}
                    onChange={(event, newValue) => {
                      handleRankerChange(rankerIndex, 'model_name', newValue?.model_name || '');
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Model Name"
                        size="small"
                        fullWidth
                      />
                    )}
                  />
                </Grid>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    size="small"
                    label="End Point"
                    value={ranker.end_point || ''}
                    onChange={(e) => handleRankerChange(rankerIndex, 'end_point', e.target.value)}
                    placeholder="Auto-populated from selected model"
                    disabled
                  />
                </Grid>
                <Grid item xs={4}>
                  <Autocomplete
                    options={[...(CALIBRATION_OPTIONS || [])].sort((a, b) => a.localeCompare(b))}
                    value={ranker.calibration}
                    onChange={(event, newValue) => {
                      handleRankerChange(rankerIndex, 'calibration', newValue || '');
                    }}
                    freeSolo
                    onInputChange={(event, newInputValue) => {
                      if (event && event.type === 'change') {
                        handleRankerChange(rankerIndex, 'calibration', newInputValue || '');
                      }
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Calibration"
                        size="small"
                        fullWidth
                        placeholder="Select or type custom value"
                      />
                    )}
                  />
                </Grid>
                <Grid item xs={4}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Batch Size"
                    type="number"
                    value={ranker.batch_size}
                    onChange={(e) => handleRankerChange(rankerIndex, 'batch_size', e.target.value)}
                    InputProps={{
                      inputProps: { 
                        min: 1,
                        max: ranker.max_batch_size || undefined
                      }
                    }}
                    helperText={ranker.max_batch_size ? `Max: ${ranker.max_batch_size}` : ''}
                    error={ranker.max_batch_size && Number(ranker.batch_size) > ranker.max_batch_size}
                  />
                </Grid>
                <Grid item xs={4}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Deadline"
                    type="number"
                    value={ranker.deadline}
                    onChange={(e) => handleRankerChange(rankerIndex, 'deadline', e.target.value)}
                  />
                </Grid>
              </Grid>

              {/* Route Config Section - Admin Only */}
              {isAdmin && (
                <Box sx={{ mt: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    <Typography variant="subtitle2">Route Config</Typography>
                    <Tooltip 
                      title='Enter valid JSON object. Example: [{"secondary_model_name": "model_name", "secondary_model_endpoint": "endpoint", "routing_percentage": "10"}]'
                      placement="top"
                    >
                      <HelpOutlineIcon sx={{ color: '#1976d2', cursor: 'help', fontSize: '18px' }} />
                    </Tooltip>
                  </Box>
                  <TextField
                    fullWidth
                    multiline
                    rows={4}
                    size="small"
                    label="Route Config"
                    value={typeof ranker.route_config === 'string' 
                      ? ranker.route_config 
                      : ranker.route_config 
                        ? JSON.stringify(ranker.route_config, null, 2)
                        : ''}
                    onChange={(e) => {
                      const value = e.target.value;
                      handleRankerChange(rankerIndex, 'route_config', value);
                    }}
                    placeholder='{"secondary_model_name": "model_name", "secondary_model_endpoint": "endpoint", "routing_percentage": "10"}'
                    error={(() => {
                      if (!ranker.route_config || ranker.route_config.trim() === '') return false;
                      try {
                        JSON.parse(ranker.route_config);
                        return false;
                      } catch (e) {
                        return true;
                      }
                    })()}
                    helperText={(() => {
                      if (!ranker.route_config || ranker.route_config.trim() === '') {
                        return 'Enter valid JSON object for route config';
                      }
                      try {
                        JSON.parse(ranker.route_config);
                        return 'Valid JSON';
                      } catch (e) {
                        return `Invalid JSON: ${e.message}`;
                      }
                    })()}
                    sx={{
                      '& .MuiInputBase-input': {
                        fontFamily: 'monospace',
                        fontSize: '0.875rem'
                      }
                    }}
                  />
                </Box>
              )}

              {/* Entity IDs */}
              <Box sx={{ mt: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>Entity IDs</Typography>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 1 }}>
                  {ranker.entity_id?.map((entityId, index) => (
                    <Chip
                      key={index}
                      label={entityId}
                      onDelete={() => {
                        const newEntityIds = ranker.entity_id.filter((_, i) => i !== index);
                        handleRankerChange(rankerIndex, 'entity_id', newEntityIds);
                      }}
                    />
                  ))}
                </Box>
                <TextField
                  fullWidth
                  size="small"
                  placeholder="Add entity ID(s) and press Enter (comma-separated for multiple)"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && e.target.value.trim()) {
                      const inputValues = e.target.value.split(',').map(v => v.trim()).filter(v => v !== '');
                      const newEntityIds = [...ranker.entity_id, ...inputValues];
                      handleRankerChange(rankerIndex, 'entity_id', newEntityIds);
                      e.target.value = '';
                    }
                  }}
                />
              </Box>

              {/* Inputs Section */}
              <Typography variant="subtitle1" sx={{ mt: 3, mb: 2, fontWeight: 600, color: '#555' }}>Inputs</Typography>
              {ranker.inputs?.map((input, inputIndex) => (
                <Box key={inputIndex} sx={{ mb: 3, p: 2, bgcolor: 'white', borderRadius: 1, border: '1px solid #f0f0f0' }}>
                  <Grid container spacing={2}>
                    <Grid item xs={4}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Input Name"
                        value={input.name}
                        onChange={(e) => handleInputChange(rankerIndex, inputIndex, 'name', e.target.value)}
                        disabled
                      />
                    </Grid>
                    <Grid item xs={4}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Data Type"
                        value={input.data_type}
                        onChange={(e) => handleInputChange(rankerIndex, inputIndex, 'data_type', e.target.value)}
                        disabled
                      />
                    </Grid>
                    <Grid item xs={4}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Dimensions"
                        value={Array.isArray(input.dims) ? input.dims.join(',') : input.dims}
                        onChange={(e) => handleInputChange(rankerIndex, inputIndex, 'dims', e.target.value.split(',').map(Number))}
                        disabled
                      />
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="body2" sx={{ mb: 1 }}>Features (Read-only)</Typography>
                      {(!input.features || input.features.length === 0) ? (
                        <Typography variant="body2" sx={{ color: '#666', fontStyle: 'italic', mb: 1 }}>
                          Select a model to see features
                        </Typography>
                      ) : (
                        <Box>
                          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 1 }}>
                            {input.features
                              ?.filter(feature => feature.trim() !== '')
                              .slice(0, expandedFeatures[`${rankerIndex}-${inputIndex}`] ? undefined : 4)
                              .map((feature, featureIndex) => (
                                <Chip
                                  key={featureIndex}
                                  label={feature}
                                  variant="outlined"
                                  size="small"
                                  sx={{ 
                                    bgcolor: '#f5f5f5',
                                    '&:hover': { bgcolor: '#f0f0f0' }
                                  }}
                                />
                              ))}
                          </Box>
                          {input.features?.filter(feature => feature.trim() !== '').length > 4 && (
                            <Button
                              size="small"
                              variant="text"
                              startIcon={expandedFeatures[`${rankerIndex}-${inputIndex}`] ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                              onClick={() => {
                                setExpandedFeatures(prev => ({
                                  ...prev,
                                  [`${rankerIndex}-${inputIndex}`]: !prev[`${rankerIndex}-${inputIndex}`]
                                }));
                              }}
                              sx={{ 
                                textTransform: 'capitalize',
                                color: '#1976d2',
                                fontSize: '0.75rem',
                                minHeight: 'auto',
                                padding: '2px 8px'
                              }}
                            >
                              {expandedFeatures[`${rankerIndex}-${inputIndex}`] ? 
                                'Show less' : 
                                `Show ${input.features.filter(feature => feature.trim() !== '').length - 4} more`
                              }
                            </Button>
                          )}
                        </Box>
                      )}
                    </Grid>
                  </Grid>
                </Box>
              ))}

              {/* Outputs Section */}
              <Typography variant="subtitle1" sx={{ mt: 3, mb: 2, fontWeight: 600, color: '#555' }}>Outputs</Typography>
              {ranker.outputs?.map((output, outputIndex) => (
                <Box key={outputIndex} sx={{ mb: 3, p: 2, bgcolor: 'white', borderRadius: 1, border: '1px solid #f0f0f0' }}>
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Output Name"
                        value={output.name}
                        onChange={(e) => handleOutputChange(rankerIndex, outputIndex, 'name', e.target.value)}
                        disabled
                      />
                    </Grid>
                    <Grid item xs={6}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Data Type"
                        value={output.data_type}
                        onChange={(e) => handleOutputChange(rankerIndex, outputIndex, 'data_type', e.target.value)}
                        disabled
                      />
                    </Grid>
                    
                    {/* Model Scores */}
                    <Grid item xs={12}>
                      <Box sx={{ 
                        border: '1px solid #e0e0e0', 
                        borderRadius: 1, 
                        p: 2, 
                        bgcolor: '#fafafa' 
                      }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500, color: '#1976d2' }}>
                            Model Scores
                          </Typography>
                          <Tooltip 
                            title='Add score names one by one. Press Enter to add each score.'
                            placement="top"
                          >
                            <HelpOutlineIcon sx={{ color: '#1976d2', cursor: 'help', fontSize: '18px' }} />
                          </Tooltip>
                        </Box>
                        
                        <Grid container spacing={2} alignItems="flex-start">
                          <Grid item xs={12} md={6}>
                            <Box sx={{ mb: 1 }}>
                              <Typography variant="caption" sx={{ color: '#666', mb: 1, display: 'block' }}>
                                Score Names ({output.model_scores?.length || 0} scores)
                              </Typography>
                              <Box sx={{ 
                                display: 'flex', 
                                gap: 1, 
                                flexWrap: 'wrap', 
                                mb: 1, 
                                minHeight: '36px',
                                p: 1,
                                border: '1px dashed #ccc',
                                borderRadius: 1,
                                bgcolor: 'white'
                              }}>
                                {output.model_scores?.length > 0 ? (
                                  output.model_scores.map((score, scoreIndex) => (
                                    <Chip
                                      key={scoreIndex}
                                      label={score}
                                      size="small"
                                      onDelete={() => {
                                        const newScores = output.model_scores.filter((_, i) => i !== scoreIndex);
                                        handleOutputChange(rankerIndex, outputIndex, 'model_scores', newScores);
                                      }}
                                      sx={{ 
                                        bgcolor: '#e3f2fd',
                                        color: '#1976d2',
                                        '& .MuiChip-deleteIcon': { color: '#1976d2' }
                                      }}
                                    />
                                  ))
                                ) : (
                                  <Typography variant="caption" sx={{ color: '#999', fontStyle: 'italic' }}>
                                    No scores added yet
                                  </Typography>
                                )}
                              </Box>
                              <TextField
                                fullWidth
                                size="small"
                                placeholder="Type score name(s) and press Enter (comma-separated for multiple)"
                                variant="outlined"
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    if (e.target.value.trim()) {
                                      const inputScores = e.target.value.split(',').map(v => {
                                        let trimmed = v.trim();
                                        // Remove surrounding double quotes
                                        if (trimmed.startsWith('"') && trimmed.endsWith('"')) {
                                          trimmed = trimmed.slice(1, -1);
                                        }
                                        // Remove surrounding single quotes
                                        else if (trimmed.startsWith("'") && trimmed.endsWith("'")) {
                                          trimmed = trimmed.slice(1, -1);
                                        }
                                        return trimmed;
                                      }).filter(v => v !== '');
                                      const uniqueNewScores = inputScores.filter(score => !output.model_scores.includes(score));
                                      if (uniqueNewScores.length > 0) {
                                        const newScores = [...output.model_scores, ...uniqueNewScores];
                                        handleOutputChange(rankerIndex, outputIndex, 'model_scores', newScores);
                                      }
                                      e.target.value = '';
                                    }
                                  }
                                }}
                                sx={{ bgcolor: 'white' }}
                              />
                            </Box>
                          </Grid>
                          
                          <Grid item xs={12} md={6}>
                            <Box sx={{ mb: 1 }}>
                              <Typography variant="caption" sx={{ color: '#666', mb: 1, display: 'block' }}>
                                Dimensions ({parseDimensionsFromString(output.model_scores_dims)?.length || 0} dimensions)
                              </Typography>
                              <Box sx={{ 
                                display: 'flex', 
                                gap: 1, 
                                flexWrap: 'wrap', 
                                mb: 1, 
                                minHeight: '36px',
                                p: 1,
                                border: '1px dashed #ccc',
                                borderRadius: 1,
                                bgcolor: 'white'
                              }}>
                                {parseDimensionsFromString(output.model_scores_dims)?.length > 0 ? (
                                  parseDimensionsFromString(output.model_scores_dims).map((dim, dimIndex) => (
                                    <Chip
                                      key={dimIndex}
                                      label={dim}
                                      size="small"
                                      onDelete={() => {
                                        const currentDims = parseDimensionsFromString(output.model_scores_dims);
                                        const newDims = currentDims.filter((_, i) => i !== dimIndex);
                                        const newDimsString = convertDimensionsToString(newDims);
                                        handleOutputChange(rankerIndex, outputIndex, 'model_scores_dims', newDimsString);
                                      }}
                                      sx={{ 
                                        bgcolor: '#fff3e0',
                                        color: '#f57c00',
                                        '& .MuiChip-deleteIcon': { color: '#f57c00' }
                                      }}
                                    />
                                  ))
                                ) : (
                                  <Typography variant="caption" sx={{ color: '#999', fontStyle: 'italic' }}>
                                    No dimensions added yet
                                  </Typography>
                                )}
                              </Box>
                              
                              {/* Repeat Dimension Feature */}
                              <RepeatDimensionInput
                                rankerIndex={rankerIndex}
                                outputIndex={outputIndex}
                                onAddRepeated={(dimensionValue, count) => {
                                  const currentDims = parseDimensionsFromString(output.model_scores_dims);
                                  const repeatedDims = Array(Number(count)).fill(dimensionValue);
                                  const newDims = [...currentDims, ...repeatedDims];
                                  const newDimsString = convertDimensionsToString(newDims);
                                  handleOutputChange(rankerIndex, outputIndex, 'model_scores_dims', newDimsString);
                                }}
                              />
                              
                              <TextField
                                fullWidth
                                size="small"
                                placeholder="Type dimension(s) and press Enter (comma-separated for multiple)"
                                variant="outlined"
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    if (e.target.value.trim()) {
                                      const inputDims = e.target.value.split(',').map(v => {
                                        let trimmed = v.trim();
                                        // Remove surrounding double quotes
                                        if (trimmed.startsWith('"') && trimmed.endsWith('"')) {
                                          trimmed = trimmed.slice(1, -1);
                                        }
                                        // Remove surrounding single quotes
                                        else if (trimmed.startsWith("'") && trimmed.endsWith("'")) {
                                          trimmed = trimmed.slice(1, -1);
                                        }
                                        return trimmed;
                                      }).filter(v => v !== '');
                                      const currentDims = parseDimensionsFromString(output.model_scores_dims);
                                      const newDims = [...currentDims, ...inputDims];
                                      const newDimsString = convertDimensionsToString(newDims);
                                      handleOutputChange(rankerIndex, outputIndex, 'model_scores_dims', newDimsString);
                                      e.target.value = '';
                                    }
                                  }
                                }}
                                sx={{ bgcolor: 'white', mt: 1 }}
                              />
                              {output.model_scores?.length > 0 && parseDimensionsFromString(output.model_scores_dims)?.length !== output.model_scores.length && (
                                <Typography variant="caption" sx={{ color: '#f57c00', mt: 1, fontSize: '0.7rem' }}>
                                  💡 Add {output.model_scores.length} dimension(s) to match {output.model_scores.length} score(s)
                                </Typography>
                              )}
                            </Box>
                          </Grid>
                        </Grid>
                      </Box>
                    </Grid>
                  </Grid>
                </Box>
              ))}
            </AccordionDetails>
          </Accordion>
        ))}
      </Box>

      <Divider sx={{ my: 3 }} />

      {/* Re-Rankers Section */}
      <Box sx={{ mb: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6" sx={{ color: '#9c27b0', fontWeight: 600 }}>Re-Rankers (Numerix)</Typography>
          <Button
            startIcon={<AddIcon />}
            onClick={addReRanker}
            variant="contained"
            size="small"
            sx={{
              bgcolor: '#9c27b0',
              color: 'white',
              textTransform: 'capitalize',
              '&:hover': {
                bgcolor: '#7b1fa2',
                boxShadow: '0 4px 8px rgba(156, 39, 176, 0.3)'
              }
            }}
          >
            Add Re-Ranker
          </Button>
        </Box>

        {formData.re_rankers?.map((reRanker, index) => (
          <Accordion
            key={index}
            expanded={expandedReRankers.includes(index)}
            onChange={() => handleReRankerAccordionChange(index)}
            sx={{
              mb: 2,
              border: '1px solid #e0e0e0',
              borderRadius: '8px !important',
              '&:before': { display: 'none' },
              boxShadow: expandedReRankers.includes(index) ? '0 2px 8px rgba(0,0,0,0.1)' : 'none'
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon />}
              sx={{
                bgcolor: '#f3e5f5',
                borderRadius: '8px',
                '&.Mui-expanded': {
                  borderBottomLeftRadius: 0,
                  borderBottomRightRadius: 0,
                },
                '& .MuiAccordionSummary-content': {
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  my: 1
                }
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, color: '#9c27b0' }}>
                  Re-Ranker {index + 1}
                </Typography>
                {reRanker.eq_id && (
                  <Chip
                    label={reRanker.eq_id}
                    size="small"
                    sx={{ bgcolor: '#f3e5f5', color: '#9c27b0', fontWeight: 500 }}
                  />
                )}
                {reRanker.score && (
                  <Chip
                    label={`Score: ${reRanker.score}`}
                    size="small"
                    variant="outlined"
                    sx={{ borderColor: '#9c27b0', color: '#9c27b0' }}
                  />
                )}
              </Box>
              <IconButton
                size="small"
                color="error"
                onClick={(e) => {
                  e.stopPropagation();
                  removeReRanker(index);
                }}
                sx={{
                  '&:hover': {
                    bgcolor: 'rgba(211, 47, 47, 0.1)',
                    transform: 'scale(1.1)'
                  }
                }}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 3, bgcolor: '#fafafa' }}>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <Autocomplete
                    options={computeConfigs || []}
                    getOptionLabel={(option) => option.compute_id || ''}
                    value={computeConfigs?.find(config => config.compute_id === reRanker.eq_id) || null}
                    onChange={(event, newValue) => {
                      const value = newValue?.compute_id || '';
                      handleReRankerChange(index, 'eq_id', value);
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Compute ID"
                        size="small"
                        fullWidth
                      />
                    )}
                  />
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>Variables</Typography>
                  {expressionVariables[index]?.map((variable, varIndex) => {
                    const currentValue = reRanker.eq_variables[variable] || '';
                    const [leftPart, rightPart] = currentValue.includes('|') 
                      ? currentValue.split('|') 
                      : ['', currentValue];
                    
                    return (
                      <Box 
                        key={varIndex} 
                        sx={{ 
                          display: 'flex', 
                          gap: 2, 
                          mb: 2,
                          alignItems: 'center' 
                        }}
                      >
                        <TextField
                          sx={{ flex: 0.8 }}
                          label="Variable Name"
                          size="small"
                          value={variable}
                          disabled
                        />
                        <Autocomplete
                          options={[...(featureTypes || [])].sort((a, b) => a.localeCompare(b))}
                          value={leftPart || null}
                          onChange={(event, newValue) => {
                            const newFeature = `${newValue || ''}|${rightPart}`;
                            handleReRankerChange(index, 'variable', {
                              varName: variable,
                              varValue: newFeature
                            });
                          }}
                          size="small"
                          sx={{ flex: 1 }}
                          loading={featureTypesLoading}
                          disableClearable={true}
                          renderInput={(params) => (
                            <TextField
                              {...params}
                              label="Feature Type"
                              size="small"
                              placeholder="Search and select feature type"
                            />
                          )}
                        />
                        <Typography variant="body2" sx={{ minWidth: 'fit-content' }}>
                          |
                        </Typography>
                        <TextField
                          sx={{ flex: 1 }}
                          label="Feature"
                          size="small"
                          value={rightPart || ''}
                          onChange={(e) => {
                            const newFeature = `${leftPart}|${e.target.value}`;
                            handleReRankerChange(index, 'variable', {
                              varName: variable,
                              varValue: newFeature
                            });
                          }}
                          placeholder="Enter feature name"
                        />
                      </Box>
                    );
                  })}
                </Grid>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Score Name"
                    value={reRanker.score}
                    onChange={(e) => handleReRankerChange(index, 'score', e.target.value)}
                  />
                </Grid>
                <Grid item xs={6}>
                  <Autocomplete
                    size="small"
                    options={[...(DATA_TYPE_OPTIONS || [])].sort((a, b) => a.localeCompare(b))}
                    value={reRanker.data_type || ''}
                    onChange={(event, newValue) => {
                      handleReRankerChange(index, 'data_type', newValue || '');
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Score Data Type"
                        placeholder="Search data types..."
                      />
                    )}
                    renderOption={(props, option) => (
                      <Box component="li" {...props} key={option}>
                        {option}
                      </Box>
                    )}
                    filterOptions={(options, { inputValue }) => {
                      return options.filter(option =>
                        option.toLowerCase().includes(inputValue.toLowerCase())
                      );
                    }}
                    noOptionsText="No matching data types"
                  />
                </Grid>
                
                {/* Entity IDs for re-ranker */}
                <Grid item xs={12}>
                  <Box sx={{ mt: 2 }}>
                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Entity IDs</Typography>
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 1 }}>
                      {reRanker.entity_id?.map((entityId, idx) => (
                        <Chip
                          key={idx}
                          label={entityId}
                          onDelete={() => {
                            const newEntityIds = reRanker.entity_id.filter((_, i) => i !== idx);
                            handleReRankerChange(index, 'entity_id', newEntityIds);
                          }}
                        />
                      ))}
                    </Box>
                    <TextField
                      fullWidth
                      size="small"
                      placeholder="Add entity ID(s) and press Enter (comma-separated for multiple)"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && e.target.value.trim()) {
                          const inputValues = e.target.value.split(',').map(v => v.trim()).filter(v => v !== '');
                          const newEntityIds = [...reRanker.entity_id, ...inputValues];
                          handleReRankerChange(index, 'entity_id', newEntityIds);
                          e.target.value = '';
                        }
                      }}
                    />
                  </Box>
                </Grid>
              </Grid>
            </AccordionDetails>
          </Accordion>
        ))}
      </Box>

      <Divider sx={{ my: 3 }} />

      {/* Response Section */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h6" sx={{ mb: 2, color: '#2e7d32', fontWeight: 600 }}>Response Config</Typography>
        <Grid container spacing={2}>
          <Grid item xs={6}>
            <TextField
              fullWidth
              size="small"
              label="Prism Logging Percentage"
              type="number"
              value={formData.response.prism_logging_perc}
              onChange={(e) => handleResponseChange('prism_logging_perc', Number(e.target.value))}
              InputProps={{ inputProps: { min: 0, max: 100 } }}
            />
          </Grid>
          <Grid item xs={6}>
            <TextField
              fullWidth
              size="small"
              label="Ranker Schema Features Percentage"
              type="number"
              value={formData.response.ranker_schema_features_in_response_perc}
              onChange={(e) => handleResponseChange('ranker_schema_features_in_response_perc', Number(e.target.value))}
              InputProps={{ inputProps: { min: 0, max: 100 } }}
            />
          </Grid>
          <Grid item xs={6}>
            <TextField
              fullWidth
              size="small"
              label="Log Batch Size"
              type="number"
              value={formData.response.log_batch_size}
              onChange={(e) => handleResponseChange('log_batch_size', Number(e.target.value))}
              InputProps={{ inputProps: { min: 0 } }}
            />
          </Grid>
          <Grid item xs={6}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 1 }}>
              <Typography variant="body2" sx={{ color: '#666' }}>Log Selective Features:</Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button
                  size="small"
                  variant={formData.response.log_features ? "contained" : "outlined"}
                  onClick={() => handleResponseChange('log_features', true)}
                  sx={{
                    minWidth: '60px',
                    bgcolor: formData.response.log_features ? '#2e7d32' : 'transparent',
                    color: formData.response.log_features ? 'white' : '#2e7d32',
                    borderColor: '#2e7d32',
                    '&:hover': {
                      bgcolor: formData.response.log_features ? '#1b5e20' : 'rgba(46, 125, 50, 0.1)',
                      borderColor: '#2e7d32'
                    }
                  }}
                >
                  Yes
                </Button>
                <Button
                  size="small"
                  variant={!formData.response.log_features ? "contained" : "outlined"}
                  onClick={() => handleResponseChange('log_features', false)}
                  sx={{
                    minWidth: '60px',
                    bgcolor: !formData.response.log_features ? '#d32f2f' : 'transparent',
                    color: !formData.response.log_features ? 'white' : '#d32f2f',
                    borderColor: '#d32f2f',
                    '&:hover': {
                      bgcolor: !formData.response.log_features ? '#b71c1c' : 'rgba(211, 47, 47, 0.1)',
                      borderColor: '#d32f2f'
                    }
                  }}
                >
                  No
                </Button>
              </Box>
            </Box>
          </Grid>
          <Grid item xs={12}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>Selective Features</Typography>
            
            {/* Entity ID Field - Required, always at 0th position */}
            <Box sx={{ mb: 2 }}>
              <TextField
                fullWidth
                size="small"
                label="Entity ID"
                required
                value={formData.response.response_features?.[0] || ''}
                onChange={(e) => {
                  const entityId = e.target.value.trim();
                  const otherFeatures = formData.response.response_features?.slice(1) || [];
                  handleResponseChange('response_features', [entityId, ...otherFeatures]);
                }}
                placeholder="Enter entity ID (required)"
                sx={{ bgcolor: 'white' }}
              />
            </Box>

            {/* Other Features */}
            <Box sx={{ mb: 1 }}>
              <Typography variant="body2" sx={{ mb: 1, color: '#666' }}>
                Other Features
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 1 }}>
                {(formData.response.response_features?.slice(1) || []).filter(feature => feature.trim() !== '').map((feature, index) => (
                  <Chip
                    key={index}
                    label={feature}
                    onDelete={() => {
                      const entityId = formData.response.response_features?.[0] || '';
                      const otherFeatures = formData.response.response_features?.slice(1) || [];
                      const newOtherFeatures = otherFeatures.filter((_, i) => i !== index);
                      handleResponseChange('response_features', [entityId, ...newOtherFeatures]);
                    }}
                  />
                ))}
              </Box>
              <TextField
                fullWidth
                size="small"
                placeholder="Add feature(s) and press Enter (comma-separated for multiple)"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && e.target.value.trim()) {
                    const inputFeatures = e.target.value.split(',').map(v => {
                      let trimmed = v.trim();
                      // Remove surrounding double quotes
                      if (trimmed.startsWith('"') && trimmed.endsWith('"')) {
                        trimmed = trimmed.slice(1, -1);
                      }
                      // Remove surrounding single quotes
                      else if (trimmed.startsWith("'") && trimmed.endsWith("'")) {
                        trimmed = trimmed.slice(1, -1);
                      }
                      return trimmed;
                    }).filter(v => v !== '');
                    const entityId = formData.response.response_features?.[0] || '';
                    const existingOtherFeatures = formData.response.response_features?.slice(1) || [];
                    const uniqueNewFeatures = inputFeatures.filter(feature => !existingOtherFeatures.includes(feature));
                    if (uniqueNewFeatures.length > 0) {
                      handleResponseChange(
                        'response_features',
                        [entityId, ...existingOtherFeatures, ...uniqueNewFeatures]
                      );
                    }
                    e.target.value = '';
                  }
                }}
                sx={{ bgcolor: 'white' }}
              />
            </Box>
          </Grid>
        </Grid>
      </Box>

      <Divider sx={{ my: 3 }} />

      {/* Host Mapping Section */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h6" sx={{ mb: 2, color: '#1976d2', fontWeight: 600 }}>Host Mapping</Typography>
        <Grid container spacing={2}>
          <Grid item xs={12}>
            <Autocomplete
              size="small"
              options={[...(mpHosts || [])].sort((a, b) => (a.name || '').localeCompare(b.name || ''))}
              getOptionLabel={(option) => `${option.name} (${option.host})`}
              value={mpHosts.find(host => host.id === formData.config_mapping.deployable_id) || null}
              onChange={(event, newValue) => {
                handleConfigMappingChange('deployable_id', newValue ? newValue.id : '');
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="InferFlow Host"
                  placeholder="Search and select InferFlow host..."
                />
              )}
              renderOption={(props, option) => (
                <Box component="li" {...props} key={option.id}>
                  <Box>
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>
                      {option.name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {option.host}
                    </Typography>
                  </Box>
                </Box>
              )}
              filterOptions={(options, { inputValue }) => {
                return options.filter(option =>
                  option.name.toLowerCase().includes(inputValue.toLowerCase()) ||
                  option.host.toLowerCase().includes(inputValue.toLowerCase())
                );
              }}
              noOptionsText="No InferFlow hosts found"
              isOptionEqualToValue={(option, value) => option.id === value.id}
            />
          </Grid>
        </Grid>
      </Box>
    </Box>
  );
};

export default MPConfigForm;

