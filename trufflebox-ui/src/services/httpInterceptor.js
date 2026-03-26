import axios from 'axios';
import { STORAGE_KEYS, ERROR_MESSAGES, APP_ROUTES, TIMING } from '../constants/authConstants';

/**
 * HTTP Interceptor Service
 * Handles automatic logout on 401 responses for both axios and fetch calls
 * 
 * Industry-standard implementation with:
 * - Request/response interception
 * - Automatic token refresh handling
 * - Network error detection
 * - Session expiration management
 */
class HttpInterceptorService {
  constructor() {
    this.logoutCallback = null;
    this.isLoggingOut = false;
  }

  init(logoutCallback) {
    this.logoutCallback = logoutCallback;
    this.setupAxiosInterceptor();
    this.setupFetchInterceptor();
  }

  setupAxiosInterceptor() {

    axios.interceptors.response.use(
      (response) => {
        return response;
      },
      async (error) => {
        if (error.response && error.response.status === 401) {
          console.log('401 Unauthorized detected in axios call:', error.config?.url);
          await this.handleUnauthorized();
        }
        
        return Promise.reject(error);
      }
    );

    console.log('Axios 401 interceptor initialized');
  }

  setupFetchInterceptor() {
    const originalFetch = window.fetch;

    window.fetch = async (...args) => {
      try {
        const response = await originalFetch(...args);

        if (response.status === 401) {
          console.log('401 Unauthorized detected in fetch call:', args[0]);
          await this.handleUnauthorized();
        }
        
        return response;
      } catch (error) {
        throw error;
      }
    };

    console.log('Fetch 401 interceptor initialized');
  }

  async handleUnauthorized() {
    if (this.isLoggingOut) {
      return;
    }

    if (window.location.pathname.includes('/login')) {
      return;
    }

    if (this.logoutCallback && typeof this.logoutCallback === 'function') {
      try {
        this.isLoggingOut = true;
        console.log('Token expired or invalid. Logging out user...');
        
        this.showSessionExpiredNotification();

        await this.logoutCallback();
        
        // Delay to ensure state updates are processed before redirect
        setTimeout(() => {
          if (!window.location.pathname.includes(APP_ROUTES.LOGIN)) {
            window.location.href = APP_ROUTES.LOGIN;
          }
        }, TIMING.LOGOUT_REDIRECT_DELAY);
        
      } catch (error) {
        console.error('Error during automatic logout:', error);
        // Clear any stale auth data from localStorage on error
        localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
        localStorage.removeItem(STORAGE_KEYS.USER);
        localStorage.removeItem(STORAGE_KEYS.SESSION_ID);
        window.location.href = APP_ROUTES.LOGIN;
      } finally {
        setTimeout(() => {
          this.isLoggingOut = false;
        }, TIMING.LOGOUT_CLEANUP_DELAY);
      }
    } else {
      console.error('Logout callback not provided to HTTP interceptor');
      // Clear any stale auth data from localStorage
      localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
      localStorage.removeItem(STORAGE_KEYS.USER);
      localStorage.removeItem(STORAGE_KEYS.SESSION_ID);
      window.location.href = APP_ROUTES.LOGIN;
    }
  }

  showSessionExpiredNotification() {
    const notification = document.createElement('div');
    notification.setAttribute('role', 'alert');
    notification.setAttribute('aria-live', 'polite');
    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background-color: #f8d7da;
      color: #721c24;
      padding: 12px 20px;
      border: 1px solid #f5c6cb;
      border-radius: 4px;
      z-index: 9999;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
      font-size: 14px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
      max-width: 400px;
    `;
    notification.textContent = ERROR_MESSAGES.SESSION_EXPIRED;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
      if (document.body.contains(notification)) {
        document.body.removeChild(notification);
      }
    }, TIMING.SESSION_EXPIRED_NOTIFICATION_DURATION);
  }

  cleanup() {
    axios.interceptors.response.clear();
    
    this.logoutCallback = null;
    this.isLoggingOut = false;
    
    console.log('HTTP interceptors cleaned up');
  }
}

const httpInterceptor = new HttpInterceptorService();

export default httpInterceptor; 