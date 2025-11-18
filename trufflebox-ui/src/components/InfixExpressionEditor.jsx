import React, { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Chip,
  Typography,
  Grid,
  Paper,
  Alert,
  IconButton,
  Tooltip,
  Divider,
  ToggleButton,
  ToggleButtonGroup
} from '@mui/material';
import {
  Close as CloseIcon,
  Functions as FunctionsIcon,
  Calculate as CalculateIcon,
  Help as HelpIcon,
  SwapHoriz as SwapHorizIcon
} from '@mui/icons-material';
import {
  validateInfixExpression,
  convertInfixToPostfix,
  validatePostfixExpression,
  convertPostfixToInfix,
  getAvailableFunctions,
  getAvailableOperators,
  getFunctionArity
} from '../utils/infixToPostfix';

const InfixExpressionEditor = ({ 
  open, 
  onClose, 
  onSubmit, 
  initialExpression = '', 
  initialPostfixExpression = '', 
  title = "Expression Editor",
  submitting = false 
}) => {
  const [mode, setMode] = useState('infixToPostfix'); // 'infixToPostfix' or 'postfixToInfix'
  const [infixExpression, setInfixExpression] = useState(initialExpression);
  const [postfixExpression, setPostfixExpression] = useState(initialPostfixExpression);
  const [inputExpression, setInputExpression] = useState(initialExpression);
  const [outputExpression, setOutputExpression] = useState('');
  const [validation, setValidation] = useState({ isValid: true, errors: [] });
  const [, setCursorPosition] = useState(0);
  const [showHelp, setShowHelp] = useState(false);
  
  const textFieldRef = useRef(null);

  const functions = getAvailableFunctions();
  const operators = getAvailableOperators();

  // Update validation and conversion when input expression or mode changes
  useEffect(() => {
    if (inputExpression.trim()) {
      if (mode === 'infixToPostfix') {
        const validationResult = validateInfixExpression(inputExpression);
        setValidation(validationResult);
        
        if (validationResult.isValid) {
          const conversionResult = convertInfixToPostfix(inputExpression);
          if (conversionResult.success) {
            setOutputExpression(conversionResult.postfix);
            // Update stored expressions
            setInfixExpression(inputExpression);
            setPostfixExpression(conversionResult.postfix);
          } else {
            setOutputExpression('');
            setValidation({ isValid: false, errors: conversionResult.errors });
          }
        } else {
          setOutputExpression('');
        }
      } else { // postfixToInfix
        const validationResult = validatePostfixExpression(inputExpression);
        setValidation(validationResult);
        
        if (validationResult.isValid) {
          const conversionResult = convertPostfixToInfix(inputExpression);
          if (conversionResult.success) {
            setOutputExpression(conversionResult.infix);
            // Update stored expressions
            setPostfixExpression(inputExpression);
            setInfixExpression(conversionResult.infix);
          } else {
            setOutputExpression('');
            setValidation({ isValid: false, errors: conversionResult.errors });
          }
        } else {
          setOutputExpression('');
        }
      }
    } else {
      setValidation({ isValid: true, errors: [] });
      setOutputExpression('');
    }
  }, [inputExpression, mode]);

  // Reset when dialog opens/closes
  useEffect(() => {
    if (open) {
      setInfixExpression(initialExpression);
      setPostfixExpression(initialPostfixExpression);
      setMode('infixToPostfix');
      setInputExpression(initialExpression);
      
      // If we have both expressions, show the postfix as output
      if (initialExpression && initialPostfixExpression) {
        setOutputExpression(initialPostfixExpression);
      } else {
        setOutputExpression('');
      }
      
      setCursorPosition(0);
    }
  }, [open, initialExpression, initialPostfixExpression]);

  const handleModeChange = (event, newMode) => {
    if (newMode !== null) {
      const prevMode = mode;
      setMode(newMode);
      setValidation({ isValid: true, errors: [] });
      
      if (newMode === 'infixToPostfix') {
        // Show infix as input, postfix as output
        if (infixExpression) {
          setInputExpression(infixExpression);
          if (postfixExpression) {
            setOutputExpression(postfixExpression);
            setValidation({ isValid: true, errors: [] });
          } else {
            setOutputExpression('');
          }
        } else if (prevMode === 'postfixToInfix' && inputExpression) {
          // Try to convert current postfix input to infix for display
          const conversionResult = convertPostfixToInfix(inputExpression);
          if (conversionResult.success) {
            setInfixExpression(conversionResult.infix);
            setInputExpression(conversionResult.infix);
            setOutputExpression(inputExpression); // current postfix becomes output
          } else {
            setInputExpression('');
            setOutputExpression('');
          }
        } else {
          setInputExpression('');
          setOutputExpression('');
        }
      } else {
        // Show postfix as input, infix as output  
        if (postfixExpression) {
          setInputExpression(postfixExpression);
          if (infixExpression) {
            setOutputExpression(infixExpression);
            setValidation({ isValid: true, errors: [] });
          } else {
            setOutputExpression('');
          }
        } else if (prevMode === 'infixToPostfix' && inputExpression) {
          // Try to convert current infix input to postfix for display
          const conversionResult = convertInfixToPostfix(inputExpression);
          if (conversionResult.success) {
            setPostfixExpression(conversionResult.postfix);
            setInputExpression(conversionResult.postfix);
            setOutputExpression(inputExpression); // current infix becomes output
          } else {
            setInputExpression('');
            setOutputExpression('');
          }
        } else {
          setInputExpression('');
          setOutputExpression('');
        }
      }
    }
  };

  const insertAtCursor = (text) => {
    const textarea = textFieldRef.current?.querySelector('textarea');
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const newExpression = inputExpression.substring(0, start) + text + inputExpression.substring(end);
    
    setInputExpression(newExpression);
    setCursorPosition(start + text.length);
    
    // Focus back to textarea and set cursor position
    setTimeout(() => {
      textarea.focus();
      textarea.setSelectionRange(start + text.length, start + text.length);
    }, 0);
  };

  const handleFunctionClick = (functionName) => {
    if (mode === 'infixToPostfix') {
      const arity = getFunctionArity(functionName);
      let functionText = `${functionName}(`;
      
      if (arity === 1) {
        functionText += ')';
      } else if (arity === 2) {
        functionText += ', )';
      }
      
      insertAtCursor(functionText);
    } else {
      // For postfix, just insert the function name
      insertAtCursor(`${functionName} `);
    }
  };

  const handleOperatorClick = (operator) => {
    if (mode === 'infixToPostfix') {
      insertAtCursor(` ${operator} `);
    } else {
      // For postfix, just insert the operator
      insertAtCursor(`${operator} `);
    }
  };

  const handleSubmit = () => {
    if (validation.isValid && outputExpression) {
      onSubmit(infixExpression, postfixExpression);
    }
  };

  const handleClose = () => {
    setInfixExpression('');
    setPostfixExpression('');
    setInputExpression('');
    setOutputExpression('');
    setValidation({ isValid: true, errors: [] });
    setMode('infixToPostfix');
    onClose();
  };

  const commonVariables = ['pctr', 'pcvr', 'x', 'y', 'z'];

  const getInputLabel = () => {
    return mode === 'infixToPostfix' ? 'Infix Expression' : 'Postfix Expression';
  };

  const getOutputLabel = () => {
    return mode === 'infixToPostfix' ? 'Postfix Expression' : 'Infix Expression';
  };

  const getInputPlaceholder = () => {
    return mode === 'infixToPostfix' 
      ? 'Enter your infix expression here or use the buttons below'
      : 'Enter your postfix expression here (space-separated tokens)';
  };

  const getHelpText = () => {
    if (mode === 'infixToPostfix') {
      return (
        <ul style={{ margin: '8px 0', paddingLeft: '20px' }}>
          <li>Use explicit multiplication: 2 * (x + y), not 2(x + y)</li>
          <li>Use & and | for logical operations, not && and ||</li>
          <li>Functions are case-sensitive: log(x) ✓, LOG(x) ✗</li>
          <li>No scientific notation: use 0.0015 instead of 1.5e-3</li>
          <li>Negative numbers: -5 ✓, -(x + y) ✓</li>
        </ul>
      );
    } else {
      return (
        <ul style={{ margin: '8px 0', paddingLeft: '20px' }}>
          <li>Separate tokens with spaces: x y + not x y+</li>
          <li>Operands come before operators: x y + not + x y</li>
          <li>Functions come after their operands: x log not log x</li>
          <li>For two operands: x y max not max x y</li>
          <li>Example: x 2 * y 3 * + (equivalent to x*2 + y*3)</li>
        </ul>
      );
    }
  };

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: { minHeight: '600px' }
      }}
    >
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6">{title}</Typography>
        <Box>
          <Tooltip title="Show Help">
            <IconButton onClick={() => setShowHelp(!showHelp)} size="small">
              <HelpIcon />
            </IconButton>
          </Tooltip>
          <IconButton onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>
      </DialogTitle>
      
      <DialogContent dividers>
        {/* Mode Toggle */}
        <Box sx={{ mb: 2, display: 'flex', justifyContent: 'center' }}>
          <ToggleButtonGroup
            value={mode}
            exclusive
            onChange={handleModeChange}
            aria-label="conversion mode"
            sx={{
              '& .MuiToggleButton-root': {
                borderColor: '#450839',
                color: '#450839',
                '&.Mui-selected': {
                  backgroundColor: '#450839',
                  color: 'white',
                  '&:hover': {
                    backgroundColor: '#380730'
                  }
                },
                '&:hover': {
                  backgroundColor: 'rgba(69, 8, 57, 0.04)'
                }
              }
            }}
          >
            <ToggleButton value="infixToPostfix" aria-label="infix to postfix">
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography variant="body2">Infix → Postfix</Typography>
              </Box>
            </ToggleButton>
            <ToggleButton value="postfixToInfix" aria-label="postfix to infix">
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <SwapHorizIcon />
                <Typography variant="body2">Postfix → Infix</Typography>
              </Box>
            </ToggleButton>
          </ToggleButtonGroup>
        </Box>

        {showHelp && (
          <Alert severity="info" sx={{ mb: 2 }}>
            <Typography variant="body2">
              <strong>Quick Tips for {mode === 'infixToPostfix' ? 'Infix' : 'Postfix'} expressions:</strong>
              {getHelpText()}
            </Typography>
          </Alert>
        )}

        <Box sx={{ mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            {getInputLabel()}:
          </Typography>
          <TextField
            ref={textFieldRef}
            fullWidth
            multiline
            rows={3}
            value={inputExpression}
            onChange={(e) => setInputExpression(e.target.value)}
            placeholder={getInputPlaceholder()}
            error={!validation.isValid}
            helperText={validation.errors.length > 0 ? validation.errors.join(', ') : ''}
            sx={{ mb: 1 }}
          />
        </Box>

        <Box sx={{ mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            {getOutputLabel()}:
          </Typography>
          <TextField
            fullWidth
            multiline
            minRows={2}
            maxRows={4}
            value={outputExpression}
            disabled
            placeholder="Will be generated automatically"
            sx={{ 
              '& .MuiInputBase-input.Mui-disabled': {
                WebkitTextFillColor: '#000',
              },
              '& .MuiInputBase-root': {
                wordBreak: 'break-word',
                overflowWrap: 'break-word'
              },
              backgroundColor: '#f5f5f5'
            }}
          />
        </Box>

        <Grid container spacing={2}>
          {/* Functions Section */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2, height: 'fit-content' }}>
              <Typography variant="subtitle2" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <FunctionsIcon fontSize="small" />
                Functions
              </Typography>
              <Divider sx={{ mb: 1 }} />
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {functions.map((func) => (
                  <Chip
                    key={func}
                    label={mode === 'infixToPostfix' 
                      ? `${func}(${getFunctionArity(func) === 2 ? 'x,y' : 'x'})`
                      : func
                    }
                    onClick={() => handleFunctionClick(func)}
                    clickable
                    color="primary"
                    variant="outlined"
                    size="small"
                  />
                ))}
              </Box>
            </Paper>
          </Grid>

          {/* Operators Section */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2, height: 'fit-content' }}>
              <Typography variant="subtitle2" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <CalculateIcon fontSize="small" />
                Operators
              </Typography>
              <Divider sx={{ mb: 1 }} />
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {operators.map((op) => (
                  <Chip
                    key={op}
                    label={op}
                    onClick={() => handleOperatorClick(op)}
                    clickable
                    color="secondary"
                    variant="outlined"
                    size="small"
                  />
                ))}
              </Box>
            </Paper>
          </Grid>

          {/* Common Elements Section */}
          <Grid item xs={12}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="subtitle2" gutterBottom>
                Common Elements
              </Typography>
              <Divider sx={{ mb: 1 }} />
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {/* Parentheses - only for infix mode */}
                {mode === 'infixToPostfix' && (
                  <Chip
                    label="( )"
                    onClick={() => insertAtCursor('()')}
                    clickable
                    variant="outlined"
                    size="small"
                  />
                )}
                
                {/* Numbers */}
                <Chip
                  label="0"
                  onClick={() => insertAtCursor(mode === 'infixToPostfix' ? '0' : '0 ')}
                  clickable
                  variant="outlined"
                  size="small"
                />
                <Chip
                  label="1"
                  onClick={() => insertAtCursor(mode === 'infixToPostfix' ? '1' : '1 ')}
                  clickable
                  variant="outlined"
                  size="small"
                />
                <Chip
                  label="-1"
                  onClick={() => insertAtCursor(mode === 'infixToPostfix' ? '-1' : '-1 ')}
                  clickable
                  variant="outlined"
                  size="small"
                />
                
                {/* Common Variables */}
                {commonVariables.map((variable) => (
                  <Chip
                    key={variable}
                    label={variable}
                    onClick={() => insertAtCursor(mode === 'infixToPostfix' ? variable : `${variable} `)}
                    clickable
                    color="info"
                    variant="outlined"
                    size="small"
                  />
                ))}
                
                {/* Comma for multi-argument functions - only for infix mode */}
                {mode === 'infixToPostfix' && (
                  <Chip
                    label=","
                    onClick={() => insertAtCursor(', ')}
                    clickable
                    variant="outlined"
                    size="small"
                  />
                )}
                
                {/* Space for postfix mode */}
                {mode === 'postfixToInfix' && (
                  <Chip
                    label="Space"
                    onClick={() => insertAtCursor(' ')}
                    clickable
                    variant="outlined"
                    size="small"
                  />
                )}
              </Box>
            </Paper>
          </Grid>
        </Grid>
      </DialogContent>
      
      <DialogActions sx={{ p: 2 }}>
        <Button 
          onClick={handleClose}
          sx={{
            color: '#450839',
            borderColor: '#450839',
            '&:hover': {
              backgroundColor: 'rgba(69, 8, 57, 0.04)',
              borderColor: '#380730'
            },
          }}
        >
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={!validation.isValid || !outputExpression || submitting}
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730'
            },
          }}
        >
          {submitting ? 'Processing...' : 'Submit'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default InfixExpressionEditor; 