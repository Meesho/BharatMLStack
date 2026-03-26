# Authentication & Authorization Security Documentation

## Overview

This document outlines the security architecture, best practices, and considerations for the authentication and authorization system in TruffleBox.

## Security Architecture

### Token Management

#### Current Implementation
- **Access Tokens**: Stored in `localStorage` (JWT format)
- **Refresh Tokens**: Stored in `localStorage` (JWT format)
- **Token Type**: JWT (JSON Web Tokens)
- **Token Refresh**: Automatic refresh 10 minutes before expiry

#### Security Considerations

⚠️ **localStorage Security Risk**: 
- Tokens stored in `localStorage` are vulnerable to XSS (Cross-Site Scripting) attacks
- If an attacker can inject JavaScript, they can access tokens
- **Mitigation**: 
  - All user input is sanitized
  - Content Security Policy (CSP) headers should be implemented
  - Consider migrating to httpOnly cookies for production (requires backend changes)

✅ **Best Practices Implemented**:
- Tokens are automatically refreshed before expiry
- Failed refresh attempts trigger logout
- Network errors during refresh don't cause immediate logout
- Token validation on every request
- Automatic cleanup on logout

### CSRF Protection

#### OAuth State Token
- CSRF state tokens are stored in `sessionStorage` (more secure than `localStorage`)
- State tokens are validated on OAuth callback
- State tokens are single-use (removed after validation)

### Error Handling

#### Industry-Standard Error Messages
- Generic error messages prevent information leakage
- Network errors are distinguished from authentication errors
- User-friendly error messages improve UX

#### Error Categories
1. **Network Errors**: Connection issues, timeouts
2. **Authentication Errors**: Invalid credentials, expired tokens
3. **Authorization Errors**: Insufficient permissions
4. **Validation Errors**: Invalid input data

## Permission System

### Role-Based Access Control (RBAC)

#### Roles
- **user**: Standard user with limited permissions
- **admin**: Administrative user with elevated permissions
- **super_admin**: Full system access (bypasses all permission checks)

#### Permission Structure
- **Service**: Application service (e.g., `predator`, `inferflow`)
- **Screen Type**: UI screen/component (e.g., `deployable`, `model`)
- **Actions**: Specific operations (e.g., `view`, `edit`, `delete`)

### Permission Checks

#### Frontend Checks
- Route-level protection via `ProtectedRoute` component
- Component-level checks via `hasPermission` hook
- Screen-level checks via `hasScreenAccess` hook

#### Backend Validation
- All permission checks are validated on the backend
- Frontend checks are for UX only (hiding/showing UI elements)
- Backend is the source of truth for authorization

## Security Best Practices

### 1. Token Storage

**Current**: localStorage
**Recommendation for Production**:
- Consider httpOnly cookies for better XSS protection
- Implement SameSite cookie attribute
- Use Secure flag in production (HTTPS only)

### 2. Token Refresh

**Current Implementation**:
- Automatic refresh 10 minutes before expiry
- Retry logic for network errors
- Prevents multiple simultaneous refresh attempts

**Best Practices**:
- ✅ Implemented: Token refresh queue
- ✅ Implemented: Network error handling
- ✅ Implemented: Automatic retry on network failure

### 3. Session Management

**Current Implementation**:
- Session tracking via `/track-session` endpoint
- Session ID stored in localStorage
- Automatic cleanup on logout

### 4. Error Handling

**Best Practices**:
- ✅ Generic error messages (no information leakage)
- ✅ Network error detection
- ✅ User-friendly error messages
- ✅ Proper error logging (console.warn/error)

### 5. Input Validation

**Recommendations**:
- All user input should be validated on both frontend and backend
- Sanitize all inputs before processing
- Use parameterized queries (backend)
- Validate OAuth state tokens

## Security Headers (Backend)

The following security headers should be implemented on the backend:

```
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

## OAuth 2.0 Implementation

### Google OAuth Flow

1. **Initiation**: User clicks "Sign in with Google"
2. **Redirect**: User redirected to Google OAuth consent screen
3. **Callback**: Google redirects back with authorization code
4. **Token Exchange**: Backend exchanges code for access token
5. **User Creation/Login**: User account created or logged in
6. **Token Storage**: Access and refresh tokens stored

### Security Measures
- ✅ CSRF state token validation
- ✅ Single-use state tokens
- ✅ Secure redirect URI validation
- ✅ Token expiration handling

## Audit Logging

### Recommended Logging Events
- User login/logout
- Permission changes
- Role changes
- Failed authentication attempts
- Token refresh events
- OAuth flow events

## Migration Path for Enhanced Security

### Phase 1: Current (localStorage)
- ✅ Implemented
- Suitable for development and staging

### Phase 2: Enhanced (httpOnly Cookies)
- Migrate tokens to httpOnly cookies
- Implement SameSite attribute
- Add Secure flag for HTTPS
- Update backend to set cookies

### Phase 3: Advanced (Token Encryption)
- Encrypt tokens before storage
- Implement token rotation
- Add device fingerprinting
- Implement session management

## Compliance Considerations

### GDPR
- User consent for data processing
- Right to access/delete data
- Data minimization
- Secure data storage

### SOC 2
- Access controls
- Audit logging
- Security monitoring
- Incident response

## Testing Security

### Recommended Tests
1. **XSS Testing**: Verify tokens cannot be accessed via XSS
2. **CSRF Testing**: Verify state token validation
3. **Token Expiry**: Test automatic refresh and logout
4. **Permission Bypass**: Verify super_admin checks
5. **Network Errors**: Test behavior during network failures

## Known Limitations

1. **localStorage XSS Risk**: Tokens accessible via XSS (mitigated by input sanitization)
2. **No Token Encryption**: Tokens stored in plain text (acceptable for JWT)
3. **No Rate Limiting**: Frontend doesn't implement rate limiting (backend should)

## Future Enhancements

1. **Multi-Factor Authentication (MFA)**
2. **Device Management**: Track and manage devices
3. **Session Management UI**: View and revoke active sessions
4. **Password Policies**: Enforce strong passwords
5. **Account Lockout**: Lock accounts after failed attempts

## References

- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [OAuth 2.0 Security Best Practices](https://tools.ietf.org/html/draft-ietf-oauth-security-topics)



