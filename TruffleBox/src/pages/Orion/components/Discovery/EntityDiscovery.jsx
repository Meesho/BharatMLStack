import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { 
    Modal, 
    Box, 
    Typography, 
    Table, 
    TableBody, 
    TableCell, 
    TableContainer, 
    TableHead, 
    TableRow, 
    Paper, 
    TextField, 
    Button, 
    Select, 
    MenuItem, 
    InputLabel, 
    FormControl, 
    CircularProgress, 
    InputAdornment,
    IconButton,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import SearchIcon from '@mui/icons-material/Search';
import InfoIcon from '@mui/icons-material/Info';
import ViewInArIcon from '@mui/icons-material/ViewInAr';
import EditIcon from '@mui/icons-material/Edit';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import axios from 'axios';
import { useAuth } from '../../../Auth/AuthContext';

import * as URL_CONSTANTS from '../../../../config';

function EntityDiscovery({ onEntityClick }) {
    const [entities, setEntities] = useState([]);
    const [selectedEntityInfo, setSelectedEntityInfo] = useState(null);
    const [clickedEntityLabel, setClickedEntityLabel] = useState(null);
    const [showModal, setShowModal] = useState(false);
    const [showEditModal, setShowEditModal] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);
    const [searchTerm, setSearchTerm] = useState('');
    const { user } = useAuth();
    const [showSuccessNotification, setShowSuccessNotification] = useState(false);

    const [editFormData, setEditFormData] = useState({
        inMemoryCache: {
            enabled: '',
            ttlInSeconds: '',
            jitterPercentage: '',
            confId: ''
        },
        distributedCache: {
            enabled: '',
            ttlInSeconds: '',
            jitterPercentage: '',
            confId: ''
        }
    });

    // Fetch entities from API
    const fetchEntities = useCallback(() => {
        setLoading(true);
        setError(null);
        const token = user.token;
        axios
            .get(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/retrieve-entities`, {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            })
            .then((response) => {
                const formattedEntities = response.data.map((entity) => ({
                    label: entity['entity-label'],
                    keys: entity.keys,
                    inMemoryCache: entity['in-memory-cache'],
                    distributedCache: entity['distributed-cache'],
                }));
                setEntities(formattedEntities);
            })
            .catch((error) => {
                console.error('Error fetching entities:', error);
                setError('Failed to fetch entities. Please try again later.');
            })
            .finally(() => setLoading(false));
    }, [user]);

    const handleInfoClick = (entity) => {
        setSelectedEntityInfo(entity);
        setShowModal(true);
    };

    const handleEditClick = (entity) => {
        setSelectedEntityInfo(entity);
        setEditFormData({
            inMemoryCache: {
                enabled: entity.inMemoryCache.enabled,
                ttlInSeconds: entity.inMemoryCache['ttl-in-seconds'],
                jitterPercentage: entity.inMemoryCache['jitter-percentage'],
                confId: entity.inMemoryCache['conf-id'] !== undefined && entity.inMemoryCache['conf-id'] !== null 
                    ? entity.inMemoryCache['conf-id'] 
                    : ''
            },
            distributedCache: {
                enabled: entity.distributedCache.enabled,
                ttlInSeconds: entity.distributedCache['ttl-in-seconds'],
                jitterPercentage: entity.distributedCache['jitter-percentage'],
                confId: entity.distributedCache['conf-id'] !== undefined && entity.distributedCache['conf-id'] !== null 
                    ? entity.distributedCache['conf-id'] 
                    : ''
            }
        });
        setShowEditModal(true);
    };

    const handleCloseSuccessNotification = () => {
        setShowSuccessNotification(false);
    };

    const handleEditSubmit = () => {
        const token = user.token;
        const payload = {
            "entity-label": selectedEntityInfo.label,
            "in-memory-cache": {
                "enabled": editFormData.inMemoryCache.enabled,
                "ttl-in-seconds": parseInt(editFormData.inMemoryCache.ttlInSeconds),
                "jitter-percentage": parseInt(editFormData.inMemoryCache.jitterPercentage),
                "conf-id": editFormData.inMemoryCache.confId
            },
            "distributed-cache": {
                "enabled": editFormData.distributedCache.enabled,
                "ttl-in-seconds": parseInt(editFormData.distributedCache.ttlInSeconds),
                "jitter-percentage": parseInt(editFormData.distributedCache.jitterPercentage),
                "conf-id": editFormData.distributedCache.confId
            }
        };

        axios
            .post(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/api/v1/orion/edit-entity`, payload, {
                headers: {
                    Authorization: `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
            })
            .then(() => {
                fetchEntities();
                setShowEditModal(false);
                setShowSuccessNotification(true);
            })
            .catch((error) => {
                console.error('Error editing entity:', error);
                setError('Failed to edit entity. Please try again later.');
            });
    };

    const closeModal = () => setShowModal(false);
    const closeEditModal = () => setShowEditModal(false);

    const handleFeatureGroupClick = useCallback((entity) => {
        setClickedEntityLabel(entity.label);
        onEntityClick(entity.label);
    }, [onEntityClick]);

    useEffect(() => {
        fetchEntities();
    }, [fetchEntities]);

    // Memoize entity list rendering to optimize performance
    const renderedEntities = useMemo(() => {
        const filteredEntities = entities.filter((entity) =>
            entity.label.toLowerCase().includes(searchTerm.toLowerCase())
        );
    
        return filteredEntities.map((entity, index) => (
            <Box
                key={index}
                sx={{
                    backgroundColor: clickedEntityLabel === entity.label ? '#e6f7ff' : '#ffffff',
                    margin: '5px 0',
                    padding: '10px',
                    border: clickedEntityLabel === entity.label ? '2px solid #24031e' : '1px solid #ddd',
                    borderRadius: '5px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    cursor: 'pointer',
                    transition: 'box-shadow 0.3s ease, transform 0.2s ease',
                    boxShadow: clickedEntityLabel === entity.label ? '0 0 10px rgba(0, 0, 0, 0.2)' : 'none',
                }}
                onClick={() => handleFeatureGroupClick(entity)}
            >
                <Box sx={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    flexGrow: 1,
                    marginRight: '10px'
                }}>
                    <ViewInArIcon sx={{ marginRight: '10px', color: '#24031e' }} />
                    <Typography sx={{ 
                        fontWeight: '500', 
                        fontSize: '16px',
                        whiteSpace: 'normal',
                        wordBreak: 'break-all',
                        wordWrap: 'break-word',
                        flexGrow: 1,
                        color: '#24031e'
                    }}>
                        {entity.label}
                    </Typography>
                </Box>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center'
                }}>
                    <InfoIcon
                        onClick={(e) => {
                            e.stopPropagation();
                            handleInfoClick(entity);
                        }}
                        sx={{
                            color: '#24031e',
                            cursor: 'pointer',
                            marginRight: '10px',
                            transition: 'color 0.2s ease',
                        }}
                    />
                    <EditIcon
                        onClick={(e) => {
                            e.stopPropagation();
                            handleEditClick(entity);
                        }}
                        sx={{
                            color: '#24031e',
                            cursor: 'pointer',
                            transition: 'color 0.2s ease',
                        }}
                    />
                </Box>
            </Box>
        ));
    }, [entities, clickedEntityLabel, searchTerm, handleFeatureGroupClick]);

    return (
        <Box sx={{ position: 'relative', height: '100%' }}>
            {/* Sticky Header */}
            <Box
                sx={{
                    position: 'sticky',
                    top: 0,
                    backgroundColor: '#35072c',
                    width: '340px',
                    height: '50px',
                    margin: '5px',
                    padding: '10px',
                    fontSize: '20px',
                    fontWeight: 'bold',
                    color: '#ffffff',
                    zIndex: 1000,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    borderRadius: '5px',
                }}
            >
                Entities
            </Box>

            {/* Search Input */}
            <Box sx={{ padding: '10px' }}>
                <TextField
                    fullWidth
                    variant="outlined"
                    placeholder="Search Entities"
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    InputProps={{
                        startAdornment: (
                            <InputAdornment position="start">
                                <SearchIcon />
                            </InputAdornment>
                        ),
                    }}
                />
            </Box>

            {/* Entity List */}
            <Box
                sx={{
                    display: 'flex',
                    flexDirection: 'column',
                    overflowY: 'scroll',
                    width: '350px',
                    height: 'calc(100vh - 120px)',
                    backgroundColor: '#f5f5f5',
                    borderRadius: '5px',
                    padding: '5px',
                }}
            >
                {loading ? (
                    <Box sx={{ textAlign: 'center', padding: '20px' }}>
                        <CircularProgress />
                    </Box>
                ) : error ? (
                    <Box sx={{ textAlign: 'center', padding: '20px', color: 'red' }}>
                        {error}
                    </Box>
                ) : (
                    renderedEntities
                )}
            </Box>

            {/* Modal for Entity Details */}
            {selectedEntityInfo && (
                <Modal 
                    open={showModal} 
                    onClose={closeModal} 
                    aria-labelledby="entity-details-modal"
                >
                    <Box sx={{
                        position: 'absolute',
                        top: '50%',
                        left: '50%',
                        transform: 'translate(-50%, -50%)',
                        width: '80%',
                        maxWidth: 600,
                        bgcolor: 'background.paper',
                        boxShadow: 24,
                        p: 4,
                        borderRadius: 2
                    }}>
                        <Box sx={{ 
                            display: 'flex', 
                            justifyContent: 'space-between', 
                            alignItems: 'center',
                            mb: 2 
                        }}>
                            <Typography id="entity-details-modal" variant="h6" component="h2">
                                Entity Details
                            </Typography>
                            <IconButton 
                                onClick={closeModal}
                                sx={{
                                    position: 'absolute',
                                    top: 8,
                                    right: 8,
                                }}
                            >
                                <CloseIcon />
                            </IconButton>
                        </Box>
                        <Box sx={{ mt: 2 }}>
                            <Typography><strong>Label:</strong> {selectedEntityInfo.label}</Typography>
                            
                            <Typography sx={{ mt: 2 }}><strong>Keys:</strong></Typography>
                            <TableContainer component={Paper}>
                                <Table size="small">
                                    <TableHead>
                                        <TableRow>
                                            <TableCell>Sequence</TableCell>
                                            <TableCell>Entity Label</TableCell>
                                            <TableCell>Column Label</TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {Object.values(selectedEntityInfo.keys).map((key, index) => (
                                            <TableRow key={index}>
                                                <TableCell>{key.sequence}</TableCell>
                                                <TableCell>{key['entity-label']}</TableCell>
                                                <TableCell>{key['column-label']}</TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </TableContainer>

                            <Typography sx={{ mt: 2 }}><strong>In-Memory Cache:</strong></Typography>
                            <TableContainer component={Paper}>
                                <Table size="small">
                                    <TableBody>
                                        {Object.entries(selectedEntityInfo.inMemoryCache).map(([key, value], index) => (
                                            <TableRow key={index}>
                                                <TableCell><strong>{key}</strong></TableCell>
                                                <TableCell>{value.toString()}</TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </TableContainer>

                            <Typography sx={{ mt: 2 }}><strong>Distributed Cache:</strong></Typography>
                            <TableContainer component={Paper}>
                                <Table size="small">
                                    <TableBody>
                                        {Object.entries(selectedEntityInfo.distributedCache).map(([key, value], index) => (
                                            <TableRow key={index}>
                                                <TableCell><strong>{key}</strong></TableCell>
                                                <TableCell>{value.toString()}</TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </TableContainer>
                        </Box>
                    </Box>
                </Modal>
            )}

            {/* Edit Modal */}
            {selectedEntityInfo && (
                <Modal 
                    open={showEditModal} 
                    onClose={closeEditModal}
                >
                    <Box sx={{
                        position: 'absolute',
                        top: '50%',
                        left: '50%',
                        transform: 'translate(-50%, -50%)',
                        width: '80%',
                        maxWidth: 600,
                        bgcolor: 'background.paper',
                        boxShadow: 24,
                        p: 4,
                        borderRadius: 2
                    }}>
                        <Box sx={{ 
                            display: 'flex', 
                            justifyContent: 'space-between', 
                            alignItems: 'center',
                            mb: 2 
                        }}>
                            <Typography variant="h6">
                                Edit Entity: {selectedEntityInfo.label}
                            </Typography>
                            <IconButton 
                                onClick={closeEditModal}
                                sx={{
                                    position: 'absolute',
                                    top: 8,
                                    right: 8,
                                }}
                            >
                                <CloseIcon />
                            </IconButton>
                        </Box>
                        <Box component="form" sx={{ mt: 2 }}>
                            <Typography variant="h6" sx={{ mb: 2 }}>In-Memory Cache</Typography>
                            <TextField
                                fullWidth
                                label="Config ID"
                                value={editFormData.inMemoryCache.confId}
                                disabled
                                sx={{ mb: 2 }}
                            />
                            <FormControl fullWidth sx={{ mb: 2 }}>
                                <InputLabel>Enabled</InputLabel>
                                <Select
                                    value={editFormData.inMemoryCache.enabled}
                                    label="Enabled"
                                    onChange={(e) => setEditFormData(prev => ({
                                        ...prev,
                                        inMemoryCache: {
                                            ...prev.inMemoryCache,
                                            enabled: e.target.value
                                        }
                                    }))}
                                >
                                    <MenuItem value="true">True</MenuItem>
                                    <MenuItem value="false">False</MenuItem>
                                </Select>
                            </FormControl>
                            <TextField
                                fullWidth
                                type="number"
                                label="TTL (Seconds)"
                                value={editFormData.inMemoryCache.ttlInSeconds}
                                onChange={(e) => setEditFormData(prev => ({
                                    ...prev,
                                    inMemoryCache: {
                                        ...prev.inMemoryCache,
                                        ttlInSeconds: e.target.value
                                    }
                                }))}
                                sx={{ mb: 2 }}
                            />
                            <TextField
                                fullWidth
                                type="number"
                                label="Jitter Percentage"
                                value={editFormData.inMemoryCache.jitterPercentage}
                                onChange={(e) => setEditFormData(prev => ({
                                    ...prev,
                                    inMemoryCache: {
                                        ...prev.inMemoryCache,
                                        jitterPercentage: e.target.value
                                    }
                                }))}
                                sx={{ mb: 2 }}
                            />

                            <Typography variant="h6" sx={{ mb: 2 }}>Distributed Cache</Typography>
                            <TextField
                                fullWidth
                                label="Config ID"
                                value={editFormData.distributedCache.confId}
                                disabled
                                sx={{ mb: 2 }}
                            />
                            <FormControl fullWidth sx={{ mb: 2 }}>
                                <InputLabel>Enabled</InputLabel>
                                <Select
                                    value={editFormData.distributedCache.enabled}
                                    label="Enabled"
                                    onChange={(e) => setEditFormData(prev => ({
                                        ...prev,
                                        distributedCache: {
                                            ...prev.distributedCache,
                                            enabled: e.target.value
                                        }
                                    }))}
                                >
                                    <MenuItem value="true">True</MenuItem>
                                    <MenuItem value="false">False</MenuItem>
                                </Select>
                            </FormControl>
                            <TextField
                                fullWidth
                                type="number"
                                label="TTL (Seconds)"
                                value={editFormData.distributedCache.ttlInSeconds}
                                onChange={(e) => setEditFormData(prev => ({
                                    ...prev,
                                    distributedCache: {
                                        ...prev.distributedCache,
                                        ttlInSeconds: e.target.value
                                    }
                                }))}
                                sx={{ mb: 2 }}
                            />
                            <TextField
                                fullWidth
                                type="number"
                                label="Jitter Percentage"
                                value={editFormData.distributedCache.jitterPercentage}
                                onChange={(e) => setEditFormData(prev => ({
                                    ...prev,
                                    distributedCache: {
                                        ...prev.distributedCache,
                                        jitterPercentage: e.target.value
                                    }
                                }))}
                                sx={{ mb: 2 }}
                            />
                        </Box>
                        <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 2 }}>
                            <Button 
                                variant="contained" 
                                onClick={handleEditSubmit}
                                sx={{
                                    backgroundColor: '#35072c',
                                    '&:hover': {
                                      backgroundColor: '#35072c',
                                    },
                                  }}
                            >
                                Save Changes
                            </Button>
                        </Box>
                    </Box>
                </Modal>
            )}

            {/* Success Notification Dialog */}
            <Dialog
                open={showSuccessNotification}
                onClose={handleCloseSuccessNotification}
                maxWidth="sm"
            >
                <DialogTitle>
                    Success
                </DialogTitle>
                <DialogContent sx={{ pt: 2, pb: 2, minWidth: '300px' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <CheckCircleOutlineIcon sx={{ color: 'green' }} />
                        <Typography>
                            Your entity edit request has been successfully processed.
                        </Typography>
                    </Box>
                </DialogContent>
                <DialogActions>
                    <Button 
                        onClick={handleCloseSuccessNotification}
                        sx={{
                            backgroundColor: '#35072c',
                            color: 'white',
                            '&:hover': {
                                backgroundColor: '#24031e',
                            },
                        }}
                    >
                        OK
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}

export default EntityDiscovery;
