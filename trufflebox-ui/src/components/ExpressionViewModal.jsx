import React, { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Grid,
  Paper,
  Box,
  IconButton
} from '@mui/material';
import {
  Close as CloseIcon
} from '@mui/icons-material';
import { convertInfixToLatex } from '../utils/infixToPostfix';

export const ExpressionViewModal = ({ open, onClose, infixExpression, postfixExpression }) => {
  const [showLatex, setShowLatex] = useState(true);
  const [latexRenderError, setLatexRenderError] = useState(false);
  const [isRendering, setIsRendering] = useState(false);
  const latexRef = useRef(null);

  // Re-render MathJax when the content changes
  useEffect(() => {
    if (open && showLatex && infixExpression) {
      setLatexRenderError(false);
      setIsRendering(true);
      
      const renderLatex = async () => {
        try {
          if (window.MathJax && latexRef.current) {
            // Clear previous content and add new content
            const latexExpression = convertInfixToLatex(infixExpression);
            if (latexRef.current) {
              latexRef.current.innerHTML = `\\(${latexExpression}\\)`;
              
              // Use MathJax to render the expression
              await window.MathJax.typesetPromise([latexRef.current]);
              setIsRendering(false);
            }
          } else {
            // MathJax not available, show error
            console.warn('MathJax not available');
            setLatexRenderError(true);
            setIsRendering(false);
          }
        } catch (err) {
          console.error('MathJax rendering failed:', err);
          setLatexRenderError(true);
          setIsRendering(false);
        }
      };

      // Add a longer delay to ensure DOM is ready
      const timer = setTimeout(renderLatex, 200);
      return () => clearTimeout(timer);
    } else if (!showLatex) {
      setIsRendering(false);
      setLatexRenderError(false);
    }
  }, [open, showLatex, infixExpression]);

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: { 
          minHeight: '400px',
          maxHeight: '90vh',
          height: 'auto'
        }
      }}
    >
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', bgcolor: '#450839', color: 'white' }}>
        <Typography variant="h6">Expression Details</Typography>
        <IconButton onClick={onClose} size="small" sx={{ color: 'white' }}>
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      
      <DialogContent dividers sx={{ p: 3, maxHeight: '70vh', overflow: 'auto' }}>
        <Grid container spacing={3}>
          {/* Infix Expression */}
          <Grid item xs={12}>
            <Paper sx={{ p: 2, bgcolor: '#f8f9fa' }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6" color="primary">
                  Infix Expression
                </Typography>
                <Button
                  size="small"
                  onClick={() => {
                    setShowLatex(!showLatex);
                    setLatexRenderError(false);
                    setIsRendering(false);
                  }}
                  sx={{ 
                    color: '#450839',
                    '&:hover': {
                      backgroundColor: 'rgba(69, 8, 57, 0.04)'
                    }
                  }}
                >
                  {showLatex ? 'Show Raw Text' : 'Show LATEX Notation'}
                </Button>
              </Box>
              <Box sx={{ 
                p: 2, 
                bgcolor: 'white', 
                borderRadius: 1, 
                border: '1px solid #e0e0e0',
                minHeight: '80px',
                maxHeight: '300px',
                overflow: 'auto',
                display: 'flex',
                alignItems: showLatex && !latexRenderError ? 'center' : 'flex-start',
                fontFamily: showLatex ? 'inherit' : 'monospace',
                fontSize: showLatex ? '1.1rem' : '0.9rem',
                wordBreak: 'break-word',
                overflowWrap: 'break-word',
                position: 'relative'
              }}>
                {showLatex ? (
                  <>
                    {isRendering && (
                      <Box sx={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: 1, 
                        color: '#666',
                        width: '100%',
                        justifyContent: 'center'
                      }}>
                        <Typography variant="body2">Rendering LaTeX...</Typography>
                      </Box>
                    )}
                    
                    {latexRenderError ? (
                      <Box sx={{ width: '100%' }}>
                        <Typography variant="body2" color="error" sx={{ mb: 1 }}>
                          LaTeX rendering failed. Showing raw expression:
                        </Typography>
                        <Typography 
                          variant="body1" 
                          component="div" 
                          sx={{ 
                            fontFamily: 'monospace',
                            wordBreak: 'break-all',
                            whiteSpace: 'pre-wrap',
                            bgcolor: '#f8f8f8',
                            p: 1,
                            borderRadius: 1,
                            border: '1px solid #ddd'
                          }}
                        >
                          {infixExpression}
                        </Typography>
                      </Box>
                    ) : (
                      <Box
                        ref={latexRef}
                        sx={{ 
                          width: '100%',
                          minHeight: '40px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          '& .MathJax': {
                            maxWidth: '100% !important',
                            overflow: 'auto !important',
                            fontSize: '1rem !important'
                          },
                          '& .MathJax_Display': {
                            width: '100% !important',
                            textAlign: 'center !important',
                            margin: '0 !important'
                          },
                          '& mjx-container': {
                            maxWidth: '100% !important',
                            overflow: 'auto !important'
                          }
                        }}
                        style={{ visibility: isRendering ? 'hidden' : 'visible' }}
                      />
                    )}
                  </>
                ) : (
                  <Typography 
                    variant="body1" 
                    component="div" 
                    sx={{ 
                      fontFamily: 'monospace',
                      wordBreak: 'break-all',
                      whiteSpace: 'pre-wrap',
                      width: '100%',
                      lineHeight: 1.5
                    }}
                  >
                    {infixExpression}
                  </Typography>
                )}
              </Box>
            </Paper>
          </Grid>

          {/* Postfix Expression */}
          <Grid item xs={12}>
            <Paper sx={{ p: 2, bgcolor: '#f8f9fa' }}>
              <Typography variant="h6" color="secondary" sx={{ mb: 2 }}>
                Postfix Expression
              </Typography>
              <Box sx={{ 
                p: 2, 
                bgcolor: 'white', 
                borderRadius: 1, 
                border: '1px solid #e0e0e0',
                minHeight: '60px',
                display: 'flex',
                alignItems: 'flex-start',
                wordBreak: 'break-word',
                overflowWrap: 'break-word'
              }}>
                <Typography 
                  variant="body1" 
                  component="div" 
                  sx={{ 
                    fontFamily: 'monospace', 
                    color: '#666',
                    wordBreak: 'break-all',
                    whiteSpace: 'pre-wrap',
                    width: '100%'
                  }}
                >
                  {postfixExpression}
                </Typography>
              </Box>
            </Paper>
          </Grid>
        </Grid>
      </DialogContent>
      
      <DialogActions sx={{ p: 2 }}>
        <Button 
          onClick={onClose}
          variant="contained"
          sx={{
            backgroundColor: '#450839',
            '&:hover': {
              backgroundColor: '#380730'
            },
          }}
        >
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ExpressionViewModal; 