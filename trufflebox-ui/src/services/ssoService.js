import * as URL_CONSTANTS from '../config';

/**
 * SSO Service - Handles all SSO-related API calls
 */
class SSOService {
  /**
   * Get SSO status and configuration
   * @returns {Promise<Object>} SSO status response
   */
  async getSSOStatus() {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/auth/sso/status`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch SSO status');
      }

      return await response.json();
    } catch (error) {
      console.error('Error fetching SSO status:', error);
      throw error;
    }
  }

  /**
   * Initiate Google OAuth flow
   * @returns {Promise<Object>} OAuth initiation response with redirect URL and state
   */
  async initiateGoogleOAuth() {
    try {
      const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/auth/google/initiate`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to initiate Google OAuth');
      }

      return await response.json();
    } catch (error) {
      console.error('Error initiating Google OAuth:', error);
      throw error;
    }
  }

  /**
   * Handle Google OAuth callback
   * @param {string} code - Authorization code from Google
   * @param {string} state - CSRF state token
   * @returns {Promise<Object>} Login response with tokens
   */
  async handleGoogleCallback(code, state) {
    try {
      const response = await fetch(
        `${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/auth/google/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to complete Google OAuth');
      }

      return await response.json();
    } catch (error) {
      console.error('Error handling Google callback:', error);
      throw error;
    }
  }

}

export default new SSOService();


