import React from 'react';
import { Box, Typography, Button, Paper } from '@mui/material';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HomeIcon from '@mui/icons-material/Home';
import RefreshIcon from '@mui/icons-material/Refresh';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { 
      hasError: false, 
      error: null,
      errorInfo: null 
    };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({
      error: error,
      errorInfo: errorInfo
    });
  }

  handleRefresh = () => {
    this.setState({ 
      hasError: false, 
      error: null, 
      errorInfo: null 
    });
    if (this.props.onRefresh) {
      this.props.onRefresh();
    } else {
      window.location.reload();
    }
  };

  handleGoHome = () => {
    // Navigate to the home/landing page
    window.location.href = this.props.homeUrl || '/feature-discovery';
  };

  render() {
    if (this.state.hasError) {
      // Custom error UI
      return (
        <Paper 
          elevation={3} 
          sx={{ 
            padding: 4, 
            margin: 2, 
            textAlign: 'center',
            backgroundColor: '#fafafa'
          }}
        >
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
            <ErrorOutlineIcon sx={{ fontSize: 60, color: '#ef5350' }} />
            <Typography variant="h5" color="error">
              Something went wrong
            </Typography>
            <Typography variant="body1" color="textSecondary">
              {this.props.fallbackMessage || 'The application encountered an unexpected error.'}
            </Typography>
            {process.env.NODE_ENV === 'development' && this.state.error && (
              <Box sx={{ 
                backgroundColor: '#f5f5f5', 
                padding: 2, 
                borderRadius: 1, 
                maxWidth: '100%',
                overflow: 'auto'
              }}>
                <Typography variant="caption" component="pre">
                  {this.state.error.toString()}
                </Typography>
              </Box>
            )}
            
            <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', justifyContent: 'center' }}>
              <Button 
                variant="contained"
                startIcon={<RefreshIcon />}
                onClick={this.handleRefresh}
                sx={{
                  backgroundColor: '#450839',
                  '&:hover': {
                    backgroundColor: '#5A0A4B',
                  },
                }}
              >
                Try Again
              </Button>
              
              <Button 
                variant="outlined"
                startIcon={<HomeIcon />}
                onClick={this.handleGoHome}
                sx={{
                  borderColor: '#450839',
                  color: '#450839',
                  '&:hover': {
                    backgroundColor: 'rgba(69, 8, 57, 0.1)',
                    borderColor: '#5A0A4B',
                  },
                }}
              >
                Go to Home
              </Button>
            </Box>
          </Box>
        </Paper>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;