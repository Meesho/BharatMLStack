import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  CircularProgress,
  Alert,
  IconButton,
  TextField,
  Button,
  Chip,
  Grid,
  List,
  ListItem,
  Collapse,
  Divider,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import CategoryIcon from '@mui/icons-material/Category';
import ModelTrainingIcon from '@mui/icons-material/ModelTraining';
import ScienceIcon from '@mui/icons-material/Science';
import InfoIcon from '@mui/icons-material/Info';
import SettingsIcon from '@mui/icons-material/Settings';
import embeddingPlatformAPI from '../../../services/embeddingPlatform/api';

const HierarchicalDiscovery = () => {
  const [entities, setEntities] = useState([]);
  const [allModelsData, setAllModelsData] = useState({}); // { entityName: { StoreId, Models: {...} } }
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedEntities, setExpandedEntities] = useState(new Set());
  const [expandedModels, setExpandedModels] = useState(new Set());
  const [selectedItem, setSelectedItem] = useState(null);
  const [selectedType, setSelectedType] = useState(null); // 'entity', 'model', 'variant'

  // Fetch all entities and models
  useEffect(() => {
    fetchAllData();
  }, []);

  // Auto-expand entities that contain matching models when searching
  useEffect(() => {
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const entitiesToExpand = new Set();
      
      // Check all entities for matching models
      entities.forEach(entity => {
        const entityData = allModelsData[entity.name];
        const modelsObj = entityData?.Models ?? entityData?.models;
        if (entityData && modelsObj) {
          const hasMatchingModel = Object.entries(modelsObj).some(([modelName, modelData]) => {
            const mt = modelData.model_type ?? modelData.ModelType;
            const jf = modelData.job_frequency ?? modelData.JobFrequency;
            return modelName.toLowerCase().includes(query) ||
              mt?.toLowerCase().includes(query) ||
              jf?.toLowerCase().includes(query);
          });
          if (hasMatchingModel) {
            entitiesToExpand.add(entity.name);
          }
        }
      });
      
      if (entitiesToExpand.size > 0) {
        setExpandedEntities(prev => {
          const newSet = new Set(prev);
          entitiesToExpand.forEach(entityName => newSet.add(entityName));
          return newSet;
        });
      }
    }
  }, [searchQuery, entities, allModelsData]);

  const fetchAllData = async () => {
    try {
      setLoading(true);
      setError('');
      
      // Fetch entities
      const entitiesResponse = await embeddingPlatformAPI.getEntities();
      const entitiesList = entitiesResponse.entities || [];
      setEntities(entitiesList);
      
      // Fetch all models
      const modelsResponse = await embeddingPlatformAPI.getModels();
      if (modelsResponse.models && typeof modelsResponse.models === 'object' && !Array.isArray(modelsResponse.models)) {
        setAllModelsData(modelsResponse.models);
      }
    } catch (error) {
      console.error('Error fetching data:', error);
      setError('Failed to load data. Please refresh the page.');
    } finally {
      setLoading(false);
    }
  };

  const handleEntityToggle = (entityName) => {
    setExpandedEntities(prev => {
      const newSet = new Set(prev);
      if (newSet.has(entityName)) {
        newSet.delete(entityName);
      } else {
        newSet.add(entityName);
      }
      return newSet;
    });
  };

  const handleModelToggle = (entityName, modelName) => {
    const modelKey = `${entityName}_${modelName}`;
    setExpandedModels(prev => {
      const newSet = new Set(prev);
      if (newSet.has(modelKey)) {
        newSet.delete(modelKey);
      } else {
        newSet.add(modelKey);
      }
      return newSet;
    });
  };

  const handleItemClick = (type, data) => {
    setSelectedItem(data);
    setSelectedType(type);
  };

  // Filter entities, models, and variants based on search query
  const filteredEntities = useMemo(() => {
    if (!searchQuery) return entities;
    
    const query = searchQuery.toLowerCase();
    return entities.filter(entity => {
      // Check if entity name or store_id matches
      const entityMatches = entity.name?.toLowerCase().includes(query) ||
        entity.store_id?.toString().includes(query);
      
      // Check if any model in this entity matches
      const entityData = allModelsData[entity.name];
      const modelsObj = entityData?.Models ?? entityData?.models;
      if (entityData && modelsObj) {
        const hasMatchingModel = Object.entries(modelsObj).some(([modelName, modelData]) => {
          const mt = modelData.model_type ?? modelData.ModelType;
          const jf = modelData.job_frequency ?? modelData.JobFrequency;
          return modelName.toLowerCase().includes(query) ||
            mt?.toLowerCase().includes(query) ||
            jf?.toLowerCase().includes(query);
        });
        if (hasMatchingModel) return true;
      }
      
      return entityMatches;
    });
  }, [entities, searchQuery, allModelsData]);

  // Track which entities match the search (for showing all their children)
  const matchingEntityNames = useMemo(() => {
    if (!searchQuery) return new Set();
    const query = searchQuery.toLowerCase();
    const matching = new Set();
    
    entities.forEach(entity => {
      const entityMatches = entity.name?.toLowerCase().includes(query) ||
        entity.store_id?.toString().includes(query);
      if (entityMatches) {
        matching.add(entity.name);
      }
    });
    
    return matching;
  }, [entities, searchQuery]);

  // Track which models match the search (for showing all their children)
  const matchingModelKeys = useMemo(() => {
    if (!searchQuery) return new Set();
    const query = searchQuery.toLowerCase();
    const matching = new Set();
    
    entities.forEach(entity => {
      const entityData = allModelsData[entity.name];
      const modelsObj = entityData?.Models ?? entityData?.models;
      if (entityData && modelsObj) {
        Object.entries(modelsObj).forEach(([modelName, modelData]) => {
          const mt = modelData.model_type ?? modelData.ModelType;
          const jf = modelData.job_frequency ?? modelData.JobFrequency;
          const modelMatches = modelName.toLowerCase().includes(query) ||
            mt?.toLowerCase().includes(query) ||
            jf?.toLowerCase().includes(query);
          if (modelMatches) {
            matching.add(`${entity.name}_${modelName}`);
          }
        });
      }
    });
    
    return matching;
  }, [entities, searchQuery, allModelsData]);

  const getModelsForEntity = (entityName) => {
    const entityData = allModelsData[entityName];
    const modelsObj = entityData?.Models ?? entityData?.models;
    if (!entityData || !modelsObj) return [];
    
    return Object.entries(modelsObj).map(([modelName, modelData]) => {
      const mt = modelData.model_type ?? modelData.ModelType;
      const mc = modelData.model_config ?? modelData.ModelConfig;
      const jf = modelData.job_frequency ?? modelData.JobFrequency;
      return {
        name: modelName,
        ...modelData,
        entity: entityName,
        ModelType: mt,
        ModelConfig: mc,
        JobFrequency: jf,
      };
    });
  };

  const getFilteredModels = (entityName) => {
    const models = getModelsForEntity(entityName);
    if (!searchQuery) return models;
    
    // If the entity matches the search, show all its models
    if (matchingEntityNames.has(entityName)) {
      return models;
    }
    
    // Otherwise, filter models based on search query
    const query = searchQuery.toLowerCase();
    return models.filter(model => {
      const mt = model.model_type ?? model.ModelType;
      const jf = model.job_frequency ?? model.JobFrequency;
      return model.name?.toLowerCase().includes(query) ||
        mt?.toLowerCase().includes(query) ||
        jf?.toLowerCase().includes(query);
    });
  };

  const getVariantsForModel = (entityName, modelName) => {
    const entityData = allModelsData[entityName];
    const modelsObj = entityData?.Models ?? entityData?.models;
    if (!entityData || !modelsObj || !modelsObj[modelName]) return [];
    
    const model = modelsObj[modelName];
    const variantsObj = model?.variants ?? model?.Variants;
    if (!variantsObj) return [];
    
    const modelType = model?.model_type ?? model?.ModelType;
    const otdPath = model?.otd_training_data_path ?? model?.OtdTrainingDataPath;
    return Object.entries(variantsObj).map(([variantName, variantData]) => ({
      name: variantName,
      ...variantData,
      entity: entityName,
      model: modelName,
      Type: variantData.type ?? variantData.Type,
      VariantState: variantData.variant_state ?? variantData.VariantState,
      model_type: modelType,
      ModelType: modelType,
      otd_training_data_path: variantData.otd_training_data_path ?? variantData.OtdTrainingDataPath ?? otdPath,
    }));
  };

  const getFilteredVariants = (entityName, modelName) => {
    const variants = getVariantsForModel(entityName, modelName);
    if (!searchQuery) return variants;
    
    const modelKey = `${entityName}_${modelName}`;
    // If the model matches the search, show all its variants
    if (matchingModelKeys.has(modelKey)) {
      return variants;
    }
    
    // Otherwise, filter variants based on search query
    const query = searchQuery.toLowerCase();
    return variants.filter(variant => {
      const typeVal = variant.type ?? variant.Type;
      const stateVal = variant.variant_state ?? variant.VariantState;
      return variant.name?.toLowerCase().includes(query) ||
        typeVal?.toLowerCase().includes(query) ||
        stateVal?.toLowerCase().includes(query);
    });
  };

  const renderEntityItem = (entity) => {
    const isExpanded = expandedEntities.has(entity.name);
    const models = getFilteredModels(entity.name);
    const hasModels = models.length > 0;
    const isSelected = selectedType === 'entity' && selectedItem?.name === entity.name;

    return (
      <Box key={entity.name}>
        <ListItem
          sx={{
            px: 2,
            py: 1,
            cursor: 'pointer',
            backgroundColor: isSelected ? 'rgba(82, 43, 74, 0.08)' : 'transparent',
            '&:hover': {
              backgroundColor: isSelected ? 'rgba(82, 43, 74, 0.12)' : 'rgba(82, 43, 74, 0.04)',
            },
          }}
          onClick={() => {
            handleEntityToggle(entity.name);
            handleItemClick('entity', { name: entity.name, store_id: entity.store_id, ...allModelsData[entity.name] });
          }}
        >
          <IconButton
            size="small"
            sx={{ mr: 1, color: 'text.secondary' }}
            onClick={(e) => {
              e.stopPropagation();
              handleEntityToggle(entity.name);
            }}
          >
            {isExpanded ? <ExpandMoreIcon /> : <ChevronRightIcon />}
          </IconButton>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: 1 }}>
            <CategoryIcon sx={{ color: '#1976d2', fontSize: '16px' }} />
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              {entity.name}
            </Typography>
          </Box>
        </ListItem>
        <Collapse in={isExpanded} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            {hasModels ? (
              models.map(model => renderModelItem(entity.name, model))
            ) : (
              <ListItem sx={{ pl: 6, py: 1 }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontSize: '0.875rem' }}>
                  No models available
                </Typography>
              </ListItem>
            )}
          </List>
        </Collapse>
      </Box>
    );
  };

  const renderModelItem = (entityName, model) => {
    const modelKey = `${entityName}_${model.name}`;
    const isExpanded = expandedModels.has(modelKey);
    const variants = getFilteredVariants(entityName, model.name);
    const hasVariants = variants.length > 0;
    const isSelected = selectedType === 'model' && selectedItem?.name === model.name && selectedItem?.entity === entityName;

    return (
      <Box key={modelKey}>
        <ListItem
          sx={{
            pl: 6,
            pr: 2,
            py: 0.75,
            cursor: 'pointer',
            backgroundColor: isSelected ? 'rgba(82, 43, 74, 0.08)' : 'transparent',
            '&:hover': {
              backgroundColor: isSelected ? 'rgba(82, 43, 74, 0.12)' : 'rgba(82, 43, 74, 0.04)',
            },
          }}
          onClick={() => {
            handleModelToggle(entityName, model.name);
            handleItemClick('model', { ...model, entity: entityName });
          }}
        >
          <IconButton
            size="small"
            sx={{ mr: 1, color: 'text.secondary' }}
            onClick={(e) => {
              e.stopPropagation();
              handleModelToggle(entityName, model.name);
            }}
          >
            {isExpanded ? <ExpandMoreIcon /> : <ChevronRightIcon />}
          </IconButton>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: 1 }}>
            <ModelTrainingIcon sx={{ color: '#f57c00', fontSize: '16px' }} />
            <Typography variant="body2">
              {model.name}
            </Typography>
          </Box>
        </ListItem>
        <Collapse in={isExpanded} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            {hasVariants ? (
              variants.map(variant => renderVariantItem(entityName, model.name, variant))
            ) : (
              <ListItem sx={{ pl: '90px', py: 1 }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontSize: '0.875rem' }}>
                  No variants available
                </Typography>
              </ListItem>
            )}
          </List>
        </Collapse>
      </Box>
    );
  };

  const renderVariantItem = (entityName, modelName, variant) => {
    const variantKey = `${entityName}_${modelName}_${variant.name}`;
    const isSelected = selectedType === 'variant' && selectedItem?.name === variant.name && selectedItem?.entity === entityName && selectedItem?.model === modelName;

    return (
      <ListItem
        key={variantKey}
        sx={{
          pl: '90px',
          pr: 2,
          py: 0.5,
          cursor: 'pointer',
          backgroundColor: isSelected ? 'rgba(82, 43, 74, 0.08)' : 'transparent',
          '&:hover': {
            backgroundColor: isSelected ? 'rgba(82, 43, 74, 0.12)' : 'rgba(82, 43, 74, 0.04)',
          },
        }}
        onClick={() => handleItemClick('variant', { ...variant, entity: entityName, model: modelName })}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
          <ScienceIcon sx={{ color: '#ff9800', fontSize: '16px' }} />
          <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
            {variant.name}
          </Typography>
        </Box>
      </ListItem>
    );
  };

  const renderDetailsPanel = () => {
    if (!selectedItem || !selectedType) {
      return (
        <Box sx={{ p: 3, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body1">Select an item to view details</Typography>
        </Box>
      );
    }

    return (
      <Box sx={{ p: 3, height: '100%', overflow: 'auto' }}>
        {/* Entity Details */}
        {selectedType === 'entity' && (
          <>
            <Box sx={{ mb: 3 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <CategoryIcon sx={{ color: '#1976d2', fontSize: 28 }} />
                <Typography variant="h6">Entity: {selectedItem.name}</Typography>
              </Box>
              <Divider sx={{ mb: 2 }} />
            </Box>
            <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)', mb: 2 }}>
              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                Entity Information
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Entity Name</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.name || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Store ID</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.StoreId || selectedItem.store_id || 'N/A'}
                  </Typography>
                </Grid>
              </Grid>
            </Box>
          </>
        )}

        {/* Model Details */}
        {selectedType === 'model' && (
          <>
            <Box sx={{ mb: 3 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <ModelTrainingIcon sx={{ color: '#f57c00', fontSize: 28 }} />
                <Typography variant="h6">Model: {selectedItem.name}</Typography>
              </Box>
              <Divider sx={{ mb: 2 }} />
            </Box>
            <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)', mb: 2 }}>
              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                Model Information
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Model Name</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.name || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Entity</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.entity || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Model Type</Typography>
                  <Chip 
                    label={selectedItem.ModelType || 'N/A'}
                    size="small"
                    sx={{ 
                      backgroundColor: selectedItem.ModelType?.toLowerCase() === 'delta' ? '#e8f5e8' : '#fff3e0',
                      color: selectedItem.ModelType?.toLowerCase() === 'delta' ? '#2e7d32' : '#f57c00',
                      fontWeight: 600,
                      mt: 0.5
                    }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Job Frequency</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.JobFrequency || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Vector Dimension</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.ModelConfig?.vector_dimension || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Topic Name</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.TopicName || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Distance Function</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.ModelConfig?.distance_function || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Number of Partitions</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.NumberOfPartitions || 'N/A'}
                  </Typography>
                </Grid>
              </Grid>
            </Box>
            <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)' }}>
              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                Model Configuration
              </Typography>
              <Box sx={{ backgroundColor: '#f5f5f5', p: 2, borderRadius: 1 }}>
                <pre style={{ margin: 0, fontSize: '0.875rem', whiteSpace: 'pre-wrap' }}>
                  {JSON.stringify(selectedItem.ModelConfig || {}, null, 2)}
                </pre>
              </Box>
            </Box>
          </>
        )}

        {/* Variant Details */}
        {selectedType === 'variant' && (
          <>
            <Box sx={{ mb: 3 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <ScienceIcon sx={{ color: '#ff9800', fontSize: 28 }} />
                <Typography variant="h6">Variant: {selectedItem.name}</Typography>
              </Box>
              <Divider sx={{ mb: 2 }} />
            </Box>
            <Box sx={{ p: 2, backgroundColor: 'rgba(82, 43, 74, 0.02)', borderRadius: 1, border: '1px solid rgba(82, 43, 74, 0.1)', mb: 2 }}>
              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                <InfoIcon fontSize="small" sx={{ color: '#522b4a' }} />
                Variant Information
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Variant Name</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.name || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Entity</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.entity || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Model</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a' }}>
                    {selectedItem.model || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Type</Typography>
                  <Chip 
                    label={selectedItem.Type || 'EXPERIMENT'}
                    size="small"
                    sx={{ 
                      backgroundColor: '#e3f2fd',
                      color: '#1976d2',
                      fontWeight: 600,
                      mt: 0.5
                    }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Vector DB Type</Typography>
                  <Chip 
                    label={selectedItem.VectorDbType || 'QDRANT'}
                    size="small"
                    sx={{ 
                      backgroundColor: '#f3e5f5',
                      color: '#7b1fa2',
                      fontWeight: 600,
                      mt: 0.5
                    }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>State</Typography>
                  <Chip 
                    label={(selectedItem.VariantState || 'ACTIVE').toUpperCase()}
                    size="small"
                    sx={{ 
                      backgroundColor: (selectedItem.VariantState || 'active').toLowerCase() === 'completed' ? '#E7F6E7' : '#EEEEEE',
                      color: (selectedItem.VariantState || 'active').toLowerCase() === 'completed' ? '#2E7D32' : '#616161',
                      fontWeight: 600,
                      mt: 0.5
                    }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Rate Limit</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.RateLimiter?.RateLimit || 0}/s
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Burst Limit</Typography>
                  <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5 }}>
                    {selectedItem.RateLimiter?.BurstLimit || 0}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Enabled</Typography>
                  <Chip 
                    label={selectedItem.Enabled ? 'Yes' : 'No'}
                    size="small"
                    sx={{ 
                      backgroundColor: selectedItem.Enabled ? '#E7F6E7' : '#EEEEEE',
                      color: selectedItem.Enabled ? '#2E7D32' : '#616161',
                      fontWeight: 600,
                      mt: 0.5
                    }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>Onboarded</Typography>
                  <Chip 
                    label={selectedItem.Onboarded ? 'Yes' : 'No'}
                    size="small"
                    sx={{ 
                      backgroundColor: selectedItem.Onboarded ? '#E7F6E7' : '#EEEEEE',
                      color: selectedItem.Onboarded ? '#2E7D32' : '#616161',
                      fontWeight: 600,
                      mt: 0.5
                    }}
                  />
                </Grid>
                {(selectedItem.model_type ?? selectedItem.ModelType ?? '').toString().toUpperCase() === 'DELTA' && (
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>OTD Training Data Path</Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace', color: '#522b4a', mt: 0.5, wordBreak: 'break-all' }}>
                      {selectedItem.otd_training_data_path || selectedItem.OtdTrainingDataPath || '—'}
                    </Typography>
                  </Grid>
                )}
              </Grid>
            </Box>
            {selectedItem.Filter && (
              <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)', mb: 2 }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                  Filter Configuration
                </Typography>
                {selectedItem.Filter.criteria && Array.isArray(selectedItem.Filter.criteria) && selectedItem.Filter.criteria.length > 0 ? (
                  <Box>
                    {selectedItem.Filter.criteria.map((criterion, index) => (
                      <Box 
                        key={index} 
                        sx={{ 
                          p: 2, 
                          mb: 1.5, 
                          backgroundColor: '#ffffff', 
                          borderRadius: 1, 
                          border: '1px solid #e0e0e0' 
                        }}
                      >
                        <Grid container spacing={2}>
                          <Grid item xs={12} sm={4}>
                            <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 0.5 }}>
                              Column Name
                            </Typography>
                            <Typography variant="body1" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                              {criterion.column_name || 'N/A'}
                            </Typography>
                          </Grid>
                          <Grid item xs={12} sm={4}>
                            <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 0.5 }}>
                              Filter Value
                            </Typography>
                            <Typography variant="body1" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                              {criterion.filter_value || 'N/A'}
                            </Typography>
                          </Grid>
                          <Grid item xs={12} sm={4}>
                            <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 0.5 }}>
                              Default Value
                            </Typography>
                            <Typography variant="body1" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                              {criterion.default_value || 'N/A'}
                            </Typography>
                          </Grid>
                          {criterion.operator && (
                            <Grid item xs={12} sm={4}>
                              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 0.5 }}>
                                Operator
                              </Typography>
                              <Typography variant="body1" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                                {criterion.operator}
                              </Typography>
                            </Grid>
                          )}
                        </Grid>
                      </Box>
                    ))}
                  </Box>
                ) : (
                  <Box sx={{ p: 2, backgroundColor: '#f5f5f5', borderRadius: 1 }}>
                    <Typography variant="body2" color="text.secondary">
                      No filter criteria configured
                    </Typography>
                  </Box>
                )}
              </Box>
            )}
            {selectedItem.VectorDbConfig && (
              <Box sx={{ p: 2, backgroundColor: 'rgba(25, 118, 210, 0.02)', borderRadius: 1, border: '1px solid rgba(25, 118, 210, 0.1)' }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <SettingsIcon fontSize="small" sx={{ color: '#1976d2' }} />
                  Vector DB Configuration
                </Typography>
                <Box sx={{ backgroundColor: '#f5f5f5', p: 2, borderRadius: 1 }}>
                  <pre style={{ margin: 0, fontSize: '0.875rem', whiteSpace: 'pre-wrap' }}>
                    {JSON.stringify(selectedItem.VectorDbConfig, null, 2)}
                  </pre>
                </Box>
              </Box>
            )}
          </>
        )}
      </Box>
    );
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Paper elevation={0} sx={{ width: '100%', height: 'calc(100vh - 100px)', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ p: 2, borderBottom: '1px solid #e0e0e0' }}>
        <Typography variant="h6" sx={{ mb: 1 }}>Hierarchical Discovery</Typography>
        <TextField
          label="Search Entities or Models"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <SearchIcon sx={{ color: 'action.active', mr: 1 }} />
            ),
          }}
          size="small"
          fullWidth
        />
      </Box>

      {/* Split View */}
      <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Left Panel - Tree View */}
        <Box sx={{ width: '20%', borderRight: '1px solid #e0e0e0', overflow: 'auto' }}>
          {filteredEntities.length === 0 ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '200px', p: 3 }}>
              <Typography color="text.secondary">
                {entities.length === 0 ? 'No entities available' : 'No entities match your search'}
              </Typography>
            </Box>
          ) : (
            <List sx={{ width: '100%' }}>
              {filteredEntities.map(entity => renderEntityItem(entity))}
            </List>
          )}
        </Box>

        {/* Right Panel - Details */}
        <Box sx={{ width: '80%', overflow: 'auto', backgroundColor: '#fafafa' }}>
          {renderDetailsPanel()}
        </Box>
      </Box>
    </Paper>
  );
};

export default HierarchicalDiscovery;