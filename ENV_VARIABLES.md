# Environment Variables Documentation

This document lists all environment variables required for the authentication and permission system.

## Backend (Horizon) Environment Variables

### Required Variables

#### JWT Configuration
- **`JWT_SECRET_KEY`** (Required)
  - Description: Secret key for signing JWT tokens
  - Example: `JWT_SECRET_KEY=your-very-secure-random-string-here`
  - **Security Note**: Use a strong, randomly generated string in production. Never commit this to version control.
  - Default: `horizon-admin-secret` (development only - will log a warning)

### Optional Variables (SSO Configuration)

#### SSO Enable/Disable
- **`SSO_ENABLED`** (Optional, default: `false`)
  - Description: Enable or disable SSO authentication
  - Values: `true` or `false`
  - Example: `SSO_ENABLED=true`

#### SSO Provider Mode
- **`SSO_PROVIDER`** (Optional, default: `password`)
  - Description: Authentication mode configuration
  - Values:
    - `password` - Only username/password login available
    - `google` - Only Google SSO available
    - `both` - Both username/password and Google SSO available
  - Example: `SSO_PROVIDER=both`

#### Google OAuth Credentials (Required if SSO_ENABLED=true and SSO_PROVIDER includes 'google')
- **`GOOGLE_OAUTH_CLIENT_ID`** (Required for Google SSO)
  - Description: Google OAuth 2.0 Client ID
  - How to get: 
    1. Go to [Google Cloud Console](https://console.cloud.google.com/)
    2. Create a new project or select existing
    3. Enable Google+ API
    4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
    5. Application type: Web application
    6. Copy the Client ID
  - Example: `GOOGLE_OAUTH_CLIENT_ID=123456789-abcdefghijklmnop.apps.googleusercontent.com`

- **`GOOGLE_OAUTH_CLIENT_SECRET`** (Required for Google SSO)
  - Description: Google OAuth 2.0 Client Secret
  - How to get: Same as above, copy the Client Secret
  - Example: `GOOGLE_OAUTH_CLIENT_SECRET=GOCSPX-abcdefghijklmnopqrstuvwxyz`
  - **Security Note**: Keep this secret secure. Never commit to version control.

- **`GOOGLE_OAUTH_REDIRECT_URI`** (Required for Google SSO)
  - Description: OAuth callback URL that Google will redirect to after authentication
  - Must match exactly with the redirect URI configured in Google Cloud Console
  - Format: `http://your-frontend-domain/auth/google/callback`
  - Examples:
    - Local development: `http://localhost:3000/auth/google/callback`
    - Production: `https://yourdomain.com/auth/google/callback`
  - Example: `GOOGLE_OAUTH_REDIRECT_URI=http://localhost:3000/auth/google/callback`

#### Token Expiry Configuration (Optional)
- **`ACCESS_TOKEN_EXPIRY`** (Optional, default: `24`)
  - Description: Access token expiry time in hours
  - Example: `ACCESS_TOKEN_EXPIRY=24`

- **`REFRESH_TOKEN_EXPIRY`** (Optional, default: `7`)
  - Description: Refresh token expiry time in days
  - Example: `REFRESH_TOKEN_EXPIRY=7`

## Frontend (TruffleBox UI) Environment Variables

### Optional Variables (SSO Configuration)

#### SSO Enable/Disable
- **`REACT_APP_SSO_ENABLED`** (Optional, default: `false`)
  - Description: Enable or disable SSO authentication in frontend
  - Should match backend `SSO_ENABLED` setting
  - Values: `true` or `false`
  - Example: `REACT_APP_SSO_ENABLED=true`

#### SSO Provider Mode
- **`REACT_APP_SSO_PROVIDER`** (Optional, default: `password`)
  - Description: Authentication mode configuration for frontend
  - Should match backend `SSO_PROVIDER` setting
  - Values:
    - `password` - Only username/password login available
    - `google` - Only Google SSO available
    - `both` - Both username/password and Google SSO available
  - Example: `REACT_APP_SSO_PROVIDER=both`

## Setup Instructions

### For Local Development

1. **Backend Setup** (`horizon/env.example` or `.env`):
   ```bash
   # Required
   JWT_SECRET_KEY=your-secure-random-key-here
   
   # Optional - for SSO
   SSO_ENABLED=true
   SSO_PROVIDER=both
   GOOGLE_OAUTH_CLIENT_ID=your-client-id
   GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
   GOOGLE_OAUTH_REDIRECT_URI=http://localhost:3000/auth/google/callback
   ```

2. **Frontend Setup** (`trufflebox-ui/env.example` or `.env`):
   ```bash
   # Optional - for SSO
   REACT_APP_SSO_ENABLED=true
   REACT_APP_SSO_PROVIDER=both
   ```

### For Production

1. **Generate a secure JWT secret**:
   ```bash
   # Generate a random 32-byte key (base64 encoded)
   openssl rand -base64 32
   ```

2. **Set all environment variables** in your deployment configuration (Docker, Kubernetes, etc.)

3. **Ensure Google OAuth redirect URI** matches your production frontend URL

## Google OAuth Setup Steps

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable "Google+ API" (or "Google Identity API")
4. Navigate to "APIs & Services" → "Credentials"
5. Click "Create Credentials" → "OAuth 2.0 Client ID"
6. Configure:
   - Application type: Web application
   - Name: Your application name
   - Authorized JavaScript origins: `http://localhost:3000` (dev) or `https://yourdomain.com` (prod)
   - Authorized redirect URIs: `http://localhost:3000/auth/google/callback` (dev) or `https://yourdomain.com/auth/google/callback` (prod)
7. Copy the Client ID and Client Secret
8. Set them in your environment variables

## Security Best Practices

1. **Never commit secrets to version control**
   - Use `.env` files (already in `.gitignore`)
   - Use secret management systems in production (AWS Secrets Manager, HashiCorp Vault, etc.)

2. **Use strong JWT secrets**
   - Minimum 32 characters
   - Randomly generated
   - Different for each environment (dev/staging/prod)

3. **Rotate secrets regularly**
   - Change JWT_SECRET_KEY periodically
   - Rotate Google OAuth credentials if compromised

4. **Use HTTPS in production**
   - OAuth redirects must use HTTPS
   - Protects tokens in transit

## Example Configuration Files

### Backend `.env` (horizon/.env)
```bash
# ... existing variables ...

# JWT Configuration
JWT_SECRET_KEY=your-production-secret-key-here-min-32-chars

# SSO Configuration
SSO_ENABLED=true
SSO_PROVIDER=both
GOOGLE_OAUTH_CLIENT_ID=123456789-abc.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=GOCSPX-xyz123
GOOGLE_OAUTH_REDIRECT_URI=https://yourdomain.com/auth/google/callback

# Token Expiry (optional)
ACCESS_TOKEN_EXPIRY=24
REFRESH_TOKEN_EXPIRY=7
```

### Frontend `.env` (trufflebox-ui/.env)
```bash
# ... existing variables ...

# SSO Configuration
REACT_APP_SSO_ENABLED=true
REACT_APP_SSO_PROVIDER=both
```

## Verification

After setting up environment variables:

1. **Backend**: Check logs for warnings about missing OAuth credentials
2. **Frontend**: Check browser console for SSO status
3. **Test SSO**: Try the "Sign in with Google" button (if enabled)

## Troubleshooting

- **SSO button not showing**: Check `REACT_APP_SSO_ENABLED=true` and backend `SSO_ENABLED=true`
- **OAuth redirect fails**: Verify `GOOGLE_OAUTH_REDIRECT_URI` matches Google Cloud Console configuration
- **Token refresh not working**: Check `JWT_SECRET_KEY` is set and consistent across restarts
- **Permission denied errors**: Verify permissions are set up in the database for your role


