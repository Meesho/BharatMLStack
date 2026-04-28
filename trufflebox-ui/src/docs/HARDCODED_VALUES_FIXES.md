# Hardcoded Values & Configuration Fixes

## Overview

This document outlines all hardcoded values that were identified and fixed, making them configurable and maintainable.

## Issues Fixed

### 1. ✅ Hardcoded Routes

**Before:**
```javascript
navigate('/feature-discovery');
navigate('/login');
navigate('/unauthorized');
```

**After:**
```javascript
import { APP_ROUTES } from '../../constants/authConstants';
navigate(APP_ROUTES.HOME);
navigate(APP_ROUTES.LOGIN);
navigate(APP_ROUTES.UNAUTHORIZED);
```

**Files Fixed:**
- `src/pages/Auth/Login.jsx`
- `src/pages/Auth/Unauthorized.jsx`
- `src/pages/Auth/Register.jsx`
- `src/pages/Auth/ProtectedRoute.jsx`
- `src/services/httpInterceptor.js`

### 2. ✅ Hardcoded Timeout Values

**Before:**
```javascript
setTimeout(() => {...}, 200);
setTimeout(() => {...}, 1000);
setTimeout(() => {...}, 3000);
const expiryTime = payload.exp * 1000;
```

**After:**
```javascript
import { TIMING } from '../../constants/authConstants';
setTimeout(() => {...}, TIMING.LOGOUT_REDIRECT_DELAY); // 200ms
setTimeout(() => {...}, TIMING.LOGOUT_CLEANUP_DELAY); // 1000ms
setTimeout(() => {...}, TIMING.SESSION_EXPIRED_NOTIFICATION_DURATION); // 3000ms
const expiryTime = payload.exp * TIMING.TOKEN_EXPIRY_MULTIPLIER; // 1000
```

**Files Fixed:**
- `src/pages/Auth/AuthContext.jsx`
- `src/services/httpInterceptor.js`

### 3. ✅ Hardcoded JWT Token Field Names

**Before:**
```javascript
userId: decodedToken.sub || decodedToken.user_id
role: decodedToken.role || role
const expiryTime = payload.exp * 1000;
```

**After:**
```javascript
import { JWT_CLAIMS } from '../../constants/authConstants';
userId: decodedToken[JWT_CLAIMS.SUBJECT] || decodedToken[JWT_CLAIMS.USER_ID]
role: decodedToken[JWT_CLAIMS.ROLE] || role
const expiryTime = payload[JWT_CLAIMS.EXPIRY] * TIMING.TOKEN_EXPIRY_MULTIPLIER;
```

**Files Fixed:**
- `src/pages/Auth/Login.jsx`

### 4. ✅ Hardcoded Default Auth Provider

**Before:**
```javascript
const login = useCallback(async (email, role, token, refreshToken = null, authProvider = 'password') => {
```

**After:**
```javascript
import { DEFAULTS } from '../../constants/authConstants';
const login = useCallback(async (email, role, token, refreshToken = null, authProvider = DEFAULTS.AUTH_PROVIDER) => {
```

**Files Fixed:**
- `src/pages/Auth/AuthContext.jsx`

### 5. ✅ Hacky Staging Environment Bypass

**Before:**
```javascript
// In staging environment, skip API call and return mock permissions
const isStaging = REACT_APP_ENVIRONMENT.toLowerCase() === 'staging';
if (isStaging) {
  const mockPermissions = { role: 'admin', permissions: [] };
  // ... bypass logic
}
```

**After:**
```javascript
// Configurable staging bypass (disabled by default, must be explicitly enabled)
import { ENV_CONFIG } from '../../constants/authConstants';
const isStaging = REACT_APP_ENVIRONMENT.toLowerCase() === 'staging';
const shouldBypass = ENV_CONFIG.ENABLE_STAGING_BYPASS && isStaging;
if (shouldBypass) {
  // ... bypass logic with warning comments
}
```

**Configuration:**
- Set `REACT_APP_ENABLE_STAGING_BYPASS=true` to enable (only for staging)
- Default: `false` (disabled for security)

**Files Fixed:**
- `src/pages/Auth/AuthContext.jsx`
- `src/pages/Auth/ProtectedRoute.jsx`

### 6. ✅ Hardcoded Default SSO Status

**Before:**
```javascript
setSsoStatus({
  sso_enabled: false,
  providers: [],
  allow_password: true,
  allow_both: false,
});
```

**After:**
```javascript
import { ENV_CONFIG } from '../../constants/authConstants';
setSsoStatus(ENV_CONFIG.DEFAULT_SSO_STATUS);
```

**Files Fixed:**
- `src/pages/Auth/Login.jsx`

### 7. ✅ Hardcoded API Endpoints

**Before:**
```javascript
fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/register`, {...})
fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}/login`, {...})
```

**After:**
```javascript
import { API_ENDPOINTS } from '../../constants/authConstants';
fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.REGISTER}`, {...})
fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_BASE_URL}${API_ENDPOINTS.LOGIN}`, {...})
```

**Files Fixed:**
- `src/pages/Auth/Register.jsx`

## New Configuration Constants

### `src/constants/authConstants.js`

Added the following configuration sections:

#### 1. Application Routes
```javascript
export const APP_ROUTES = {
  LOGIN: '/login',
  REGISTER: '/register',
  UNAUTHORIZED: '/unauthorized',
  HOME: '/feature-discovery', // Default landing page after login
  ROOT: '/',
};
```

#### 2. JWT Token Claims
```javascript
export const JWT_CLAIMS = {
  SUBJECT: 'sub', // Standard JWT subject claim
  USER_ID: 'user_id', // Custom claim (fallback)
  ROLE: 'role', // Custom claim
  EXPIRY: 'exp', // Standard JWT expiry claim
};
```

#### 3. Timing Constants
```javascript
export const TIMING = {
  LOGOUT_REDIRECT_DELAY: 200, // Delay before redirecting to login after logout
  LOGOUT_CLEANUP_DELAY: 1000, // Delay before resetting logout flag
  SESSION_EXPIRED_NOTIFICATION_DURATION: 3000, // How long to show session expired notification
  TOKEN_EXPIRY_MULTIPLIER: 1000, // Convert JWT exp (seconds) to milliseconds
};
```

#### 4. Environment Configuration
```javascript
export const ENV_CONFIG = {
  // Staging environment bypass (should be false in production)
  // This is a temporary workaround - should be removed in production
  ENABLE_STAGING_BYPASS: process.env.REACT_APP_ENABLE_STAGING_BYPASS === 'true',
  
  // Default SSO fallback values (used when SSO status fetch fails)
  DEFAULT_SSO_STATUS: {
    sso_enabled: false,
    providers: [],
    allow_password: true,
    allow_both: false,
  },
};
```

#### 5. Default Values
```javascript
export const DEFAULTS = {
  AUTH_PROVIDER: AUTH_PROVIDERS.PASSWORD,
  DEFAULT_ROLE: USER_ROLES.USER,
};
```

## Environment Variables

### New Environment Variable

**`REACT_APP_ENABLE_STAGING_BYPASS`**
- **Purpose**: Enable staging environment permission bypass (for testing only)
- **Default**: `false` (disabled)
- **Usage**: Set to `true` only in staging environment
- **Security**: Should NEVER be `true` in production

**Example:**
```bash
# .env.staging
REACT_APP_ENABLE_STAGING_BYPASS=true

# .env.production
REACT_APP_ENABLE_STAGING_BYPASS=false
```

## Benefits

### 1. Maintainability
- ✅ All hardcoded values centralized in one file
- ✅ Easy to update routes, timeouts, and configurations
- ✅ Single source of truth for constants

### 2. Configuration
- ✅ Environment-specific settings via env vars
- ✅ Easy to customize for different deployments
- ✅ Clear documentation of configurable values

### 3. Security
- ✅ Staging bypass is now opt-in (disabled by default)
- ✅ No accidental security bypasses in production
- ✅ Clear warnings in code about temporary workarounds

### 4. Code Quality
- ✅ No magic numbers or strings
- ✅ Consistent naming conventions
- ✅ Better code readability
- ✅ Easier to test and mock

## Migration Guide

### For Developers

1. **Use constants instead of hardcoded values:**
   ```javascript
   // ❌ Bad
   navigate('/login');
   
   // ✅ Good
   import { APP_ROUTES } from '../../constants/authConstants';
   navigate(APP_ROUTES.LOGIN);
   ```

2. **Use timing constants:**
   ```javascript
   // ❌ Bad
   setTimeout(() => {...}, 1000);
   
   // ✅ Good
   import { TIMING } from '../../constants/authConstants';
   setTimeout(() => {...}, TIMING.LOGOUT_CLEANUP_DELAY);
   ```

3. **Use JWT claim constants:**
   ```javascript
   // ❌ Bad
   const userId = decodedToken.sub;
   
   // ✅ Good
   import { JWT_CLAIMS } from '../../constants/authConstants';
   const userId = decodedToken[JWT_CLAIMS.SUBJECT];
   ```

### For DevOps

1. **Set environment variables:**
   ```bash
   # Staging
   REACT_APP_ENABLE_STAGING_BYPASS=true
   
   # Production
   REACT_APP_ENABLE_STAGING_BYPASS=false
   ```

2. **Update deployment configs:**
   - Add `REACT_APP_ENABLE_STAGING_BYPASS` to staging environment
   - Ensure it's `false` or unset in production

## Remaining Hardcoded Values (Acceptable)

These values are intentionally hardcoded as they are:
- Standard values that shouldn't change
- Part of the application logic
- Not configuration-related

1. **JWT token structure** - Standard JWT format
2. **HTTP status codes** - Standard HTTP codes
3. **Permission action names** - Business logic constants

## Future Improvements

1. **Make home route configurable:**
   - Add `REACT_APP_DEFAULT_HOME_ROUTE` env var
   - Allow customization of default landing page

2. **Make timing values configurable:**
   - Add env vars for timing constants
   - Allow runtime configuration

3. **Remove staging bypass:**
   - Once proper staging environment is set up
   - Remove the bypass logic entirely

## Summary

All hardcoded values have been moved to a centralized configuration file (`authConstants.js`), making the codebase:
- ✅ More maintainable
- ✅ More configurable
- ✅ More secure (staging bypass is opt-in)
- ✅ More testable
- ✅ Industry-standard compliant



