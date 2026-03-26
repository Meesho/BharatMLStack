package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
)

// CSRFStateStore is a simple in-memory store for CSRF state tokens
// In production, consider using Redis or database for distributed systems
var csrfStateStore = make(map[string]time.Time)
var csrfStateMutex sync.RWMutex

// GenerateCSRFState generates a secure random state token for OAuth
func GenerateCSRFState() (string, error) {
	b := make([]byte, constants.CSRFStateSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)
	
	// Store state with expiration
	csrfStateMutex.Lock()
	csrfStateStore[state] = time.Now().Add(time.Duration(constants.CSRFStateExpiryMinutes) * time.Minute)
	csrfStateMutex.Unlock()
	
	// Cleanup expired states
	go cleanupExpiredStates()
	
	return state, nil
}

// ValidateCSRFState validates and removes a CSRF state token
func ValidateCSRFState(state string) bool {
	csrfStateMutex.Lock()
	defer csrfStateMutex.Unlock()
	
	expiry, exists := csrfStateStore[state]
	if !exists {
		return false
	}
	
	if time.Now().After(expiry) {
		delete(csrfStateStore, state)
		return false
	}
	
	// Remove used state
	delete(csrfStateStore, state)
	return true
}

func cleanupExpiredStates() {
	csrfStateMutex.Lock()
	defer csrfStateMutex.Unlock()
	
	now := time.Now()
	for state, expiry := range csrfStateStore {
		if now.After(expiry) {
			delete(csrfStateStore, state)
		}
	}
}

// InitiateGoogleOAuth generates the Google OAuth URL
func InitiateGoogleOAuth() (string, string, error) {
	cfg := config.GetOAuthConfig()
	
	if !cfg.SSOEnabled {
		return "", "", fmt.Errorf(constants.ErrSSONotEnabled)
	}
	
	if cfg.GoogleClientID == "" || cfg.RedirectURI == "" {
		return "", "", fmt.Errorf(constants.ErrOAuthConfigIncomplete)
	}
	
	state, err := GenerateCSRFState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate CSRF state: %w", err)
	}
	
	params := url.Values{}
	params.Set("client_id", cfg.GoogleClientID)
	params.Set("redirect_uri", cfg.RedirectURI)
	params.Set("response_type", constants.OAuthResponseType)
	params.Set("scope", constants.GoogleOAuthScopes)
	params.Set("state", state)
	params.Set("access_type", constants.OAuthAccessType)
	params.Set("prompt", constants.OAuthPrompt)
	
	authURL := fmt.Sprintf("%s?%s", constants.GoogleAuthURL, params.Encode())
	return authURL, state, nil
}

// ExchangeGoogleCode exchanges authorization code for access token
func ExchangeGoogleCode(code string) (*GoogleTokenResponse, error) {
	cfg := config.GetOAuthConfig()
	
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", cfg.GoogleClientID)
	data.Set("client_secret", cfg.GoogleClientSecret)
	data.Set("redirect_uri", cfg.RedirectURI)
	data.Set("grant_type", "authorization_code")
	
	resp, err := http.PostForm(constants.GoogleTokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}
	
	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	return &tokenResp, nil
}

// GetGoogleUserInfo fetches user information from Google
func GetGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", constants.GoogleUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	
	client := &http.Client{Timeout: time.Duration(constants.GoogleAPITimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}
	
	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	
	return &userInfo, nil
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

