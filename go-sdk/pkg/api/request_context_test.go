package api

import (
	"context"
	"fmt"

	httpHeaders "github.com/Meesho/BharatMLStack/go-sdk/pkg/api/http"

	"net/http"
	"net/http/httptest"
	"testing"

	enum "github.com/Meesho/BharatMLStack/go-sdk/pkg/enums"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

// TestGetRequestContext tests the GetRequestContext function
func TestGetRequestContextForGRPC(t *testing.T) {

	userContext, _ := enum.ParseUserContext("anonymous")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedResult *RequestContext
		expectedError  string
	}{
		{
			name: "Valid headers with all fields",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserStateCode:  "state_1",
				UserPinCode:    "123456",
			},
			expectedError: "",
		},
		{
			name: "Missing user id",
			headers: map[string]string{
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "metadata key 'USER-ID' is missing ",
		},
		{
			name: "Empty user id",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "user id is empty in headers",
		},
		{
			name: "Empty user context",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "123",
				httpHeaders.HeaderUserContext:    "",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "user context is empty in headers",
		},
		{
			name: "Missing user context",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "metadata key 'USER-CONTEXT' is missing ",
		},
		{
			name: "Invalid user context",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "invalid_context",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  fmt.Sprintf("%q is not a valid UserContext", "invalid_context"),
		},
		{
			name: "Non-integer app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "non_integer",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "app_version_code should be an integer",
		},
		{
			name: "Missing user state code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserStateCode:  "",
				UserPinCode:    "123456",
			},
			expectedError: "",
		},
		{
			name: "Missing user pin code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserStateCode:  "state_1",
				UserPinCode:    "",
			},
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			headers := make(map[string]string)
			for key, value := range test.headers {
				headers[key] = value
			}
			ctx = metadata.NewIncomingContext(ctx, metadata.New(headers))

			result, err := GetRequestContextForGRPC(ctx)
			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

func TestGetRequestContextForGrpc_error(t *testing.T) {

	t.Run("test.name", func(t *testing.T) {
		ctx := context.Background()
		result, err := GetRequestContextForGRPC(ctx)
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = metadata is not provided")
		assert.Nil(t, result)

	})

}

// TestUpdateWithHeaders tests the UpdateWithHeaders function
func TestUpdateWithGRPCHeaders(t *testing.T) {
	userContext, _ := enum.ParseUserContext("anonymous")

	header := make(map[string]string)
	context := &RequestContext{
		UserId:         "12345",
		UserContext:    userContext,
		AppVersionCode: 2,
		ClientId:       "client_1",
		UserStateCode:  "state_1",
		UserPinCode:    "123456",
		UserCity:       "city_1",
	}

	UpdateWithGRPCHeaders(header, context)

	assert.Equal(t, "12345", header[httpHeaders.HeaderUserId])
	assert.Equal(t, "anonymous", header[httpHeaders.HeaderUserContext])
	assert.Equal(t, "2", header[httpHeaders.HeaderAppVersionCode])
	assert.Equal(t, "client_1", header[httpHeaders.HeaderClientId])
	assert.Equal(t, "state_1", header[httpHeaders.HeaderUserStateCode])
	assert.Equal(t, "123456", header[httpHeaders.HeaderUserPincode])
	assert.Equal(t, "city_1", header[httpHeaders.HeaderUserCity])
}

// TestGetRequestContext tests the GetRequestContext function
func TestGetRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userContext, _ := enum.ParseUserContext("anonymous")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedResult *RequestContext
		expectedError  string
	}{
		{
			name: "Valid headers with all fields",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserStateCode:  "state_1",
				UserPinCode:    "123456",
			},
			expectedError: "",
		},
		{
			name: "Missing user id",
			headers: map[string]string{
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "user id is missing in headers",
		},
		{
			name: "Missing user context",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "user context is missing in headers",
		},
		{
			name: "Invalid user context",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "invalid_context",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  fmt.Sprintf("%q is not a valid UserContext", "invalid_context"),
		},
		{
			name: "Non-integer app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "non_integer",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: nil,
			expectedError:  "app_version_code should be an integer",
		},
		{
			name: "Missing user state code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserPincode:    "123456",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserStateCode:  "",
				UserPinCode:    "123456",
			},
			expectedError: "",
		},
		{
			name: "Missing user pin code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserStateCode:  "state_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserStateCode:  "state_1",
				UserPinCode:    "",
			},
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			request, _ := http.NewRequest(http.MethodGet, "/", nil)
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			context.Request = request

			result, err := GetRequestContext(context)
			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

// TestUpdateWithHeaders tests the UpdateWithHeaders function
func TestUpdateWithHTTPHeaders(t *testing.T) {
	userContext, _ := enum.ParseUserContext("anonymous")

	header := &http.Header{}
	context := &RequestContext{
		UserId:         "12345",
		UserContext:    userContext,
		AppVersionCode: 2,
		ClientId:       "client_1",
		UserStateCode:  "state_1",
		UserPinCode:    "123456",
		UserCity:       "city_1",
	}

	UpdateWithHeaders(header, context)

	assert.Equal(t, "12345", header.Get(httpHeaders.HeaderUserId))
	assert.Equal(t, "anonymous", header.Get(httpHeaders.HeaderUserContext))
	assert.Equal(t, "2", header.Get(httpHeaders.HeaderAppVersionCode))
	assert.Equal(t, "client_1", header.Get(httpHeaders.HeaderClientId))
	assert.Equal(t, "state_1", header.Get(httpHeaders.HeaderUserStateCode))
	assert.Equal(t, "123456", header.Get(httpHeaders.HeaderUserPincode))
	assert.Equal(t, "city_1", header.Get(httpHeaders.HeaderUserCity))
}

// TestGetRequestContextForGRPC_WithFeedAndAppSession tests FeedSession and AppSession fields
func TestGetRequestContextForGRPC_WithFeedAndAppSession(t *testing.T) {
	userContext, _ := enum.ParseUserContext("anonymous")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedResult *RequestContext
		expectedError  string
	}{
		{
			name: "Valid headers with FeedSession and AppSession",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserCity:       "city_1",
				httpHeaders.HeaderFeedSession:    "feed_session_123",
				httpHeaders.HeaderAppSession:     "app_session_456",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserCity:       "city_1",
				FeedSession:    "feed_session_123",
				AppSession:     "app_session_456",
			},
			expectedError: "",
		},
		{
			name: "Valid headers without FeedSession and AppSession",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserCity:       "city_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserCity:       "city_1",
				FeedSession:    "",
				AppSession:     "",
			},
			expectedError: "",
		},
		{
			name: "Valid headers with empty FeedSession and AppSession",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserCity:       "city_1",
				httpHeaders.HeaderFeedSession:    "",
				httpHeaders.HeaderAppSession:     "",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserCity:       "city_1",
				FeedSession:    "",
				AppSession:     "",
			},
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			headers := make(map[string]string)
			for key, value := range test.headers {
				headers[key] = value
			}
			ctx = metadata.NewIncomingContext(ctx, metadata.New(headers))

			result, err := GetRequestContextForGRPC(ctx)
			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

// TestGetRequestContext_WithFeedAndAppSession tests FeedSession and AppSession fields for HTTP requests
func TestGetRequestContext_WithFeedAndAppSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userContext, _ := enum.ParseUserContext("anonymous")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedResult *RequestContext
		expectedError  string
	}{
		{
			name: "Valid headers with FeedSession and AppSession",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserCity:       "city_1",
				httpHeaders.HeaderFeedSession:    "feed_session_123",
				httpHeaders.HeaderAppSession:     "app_session_456",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserCity:       "city_1",
				FeedSession:    "feed_session_123",
				AppSession:     "app_session_456",
			},
			expectedError: "",
		},
		{
			name: "Valid headers without FeedSession and AppSession",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "2",
				httpHeaders.HeaderClientId:       "client_1",
				httpHeaders.HeaderUserCity:       "city_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 2,
				ClientId:       "client_1",
				UserCity:       "city_1",
				FeedSession:    "",
				AppSession:     "",
			},
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			request, _ := http.NewRequest(http.MethodGet, "/", nil)
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			context.Request = request

			result, err := GetRequestContext(context)
			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

// TestUpdateWithGRPCHeaders_WithAllFields tests UpdateWithGRPCHeaders with all fields including FeedSession and AppSession
func TestUpdateWithGRPCHeaders_WithAllFields(t *testing.T) {
	userContext, _ := enum.ParseUserContext("anonymous")

	header := make(map[string]string)
	context := &RequestContext{
		UserId:         "12345",
		UserContext:    userContext,
		AppVersionCode: 2,
		ClientId:       "client_1",
		UserStateCode:  "state_1",
		UserPinCode:    "123456",
		UserCity:       "city_1",
		FeedSession:    "feed_session_123",
		AppSession:     "app_session_456",
	}

	UpdateWithGRPCHeaders(header, context)

	assert.Equal(t, "12345", header[httpHeaders.HeaderUserId])
	assert.Equal(t, "anonymous", header[httpHeaders.HeaderUserContext])
	assert.Equal(t, "2", header[httpHeaders.HeaderAppVersionCode])
	assert.Equal(t, "client_1", header[httpHeaders.HeaderClientId])
	assert.Equal(t, "state_1", header[httpHeaders.HeaderUserStateCode])
	assert.Equal(t, "123456", header[httpHeaders.HeaderUserPincode])
	assert.Equal(t, "city_1", header[httpHeaders.HeaderUserCity])

	// Note: FeedSession and AppSession are not included in UpdateWithGRPCHeaders function
	// This is intentional as per the current implementation
}

// TestUpdateWithHTTPHeaders_WithAllFields tests UpdateWithHeaders with all fields including FeedSession and AppSession
func TestUpdateWithHTTPHeaders_WithAllFields(t *testing.T) {
	userContext, _ := enum.ParseUserContext("anonymous")

	header := &http.Header{}
	context := &RequestContext{
		UserId:         "12345",
		UserContext:    userContext,
		AppVersionCode: 2,
		ClientId:       "client_1",
		UserStateCode:  "state_1",
		UserPinCode:    "123456",
		UserCity:       "city_1",
		FeedSession:    "feed_session_123",
		AppSession:     "app_session_456",
	}

	UpdateWithHeaders(header, context)

	assert.Equal(t, "12345", header.Get(httpHeaders.HeaderUserId))
	assert.Equal(t, "anonymous", header.Get(httpHeaders.HeaderUserContext))
	assert.Equal(t, "2", header.Get(httpHeaders.HeaderAppVersionCode))
	assert.Equal(t, "client_1", header.Get(httpHeaders.HeaderClientId))
	assert.Equal(t, "state_1", header.Get(httpHeaders.HeaderUserStateCode))
	assert.Equal(t, "123456", header.Get(httpHeaders.HeaderUserPincode))
	assert.Equal(t, "city_1", header.Get(httpHeaders.HeaderUserCity))

	// Note: FeedSession and AppSession are not included in UpdateWithHeaders function
	// This is intentional as per the current implementation
}

// TestGetMetadataValue tests the getMetadataValue helper function
func TestGetMetadataValue(t *testing.T) {
	tests := []struct {
		name          string
		metadata      map[string]string
		key           string
		expectedValue string
		expectedError string
	}{
		{
			name: "Valid metadata with key",
			metadata: map[string]string{
				"test-key": "test-value",
			},
			key:           "test-key",
			expectedValue: "test-value",
			expectedError: "",
		},
		{
			name: "Missing key in metadata",
			metadata: map[string]string{
				"other-key": "other-value",
			},
			key:           "test-key",
			expectedValue: "",
			expectedError: "metadata key 'test-key' is missing ",
		},
		{
			name:          "Empty metadata",
			metadata:      map[string]string{},
			key:           "test-key",
			expectedValue: "",
			expectedError: "metadata key 'test-key' is missing ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			md := metadata.New(test.metadata)
			value, err := getMetadataValue(md, test.key)

			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedValue, value)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Equal(t, test.expectedValue, value)
			}
		})
	}
}

// TestGetRequestContextForGRPC_EdgeCases tests additional edge cases
func TestGetRequestContextForGRPC_EdgeCases(t *testing.T) {
	userContext, _ := enum.ParseUserContext("anonymous")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedResult *RequestContext
		expectedError  string
	}{
		{
			name: "Zero app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "0",
				httpHeaders.HeaderClientId:       "client_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 0,
				ClientId:       "client_1",
			},
			expectedError: "",
		},
		{
			name: "Negative app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "-1",
				httpHeaders.HeaderClientId:       "client_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: -1,
				ClientId:       "client_1",
			},
			expectedError: "",
		},
		{
			name: "Empty string app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "",
				httpHeaders.HeaderClientId:       "client_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 0,
				ClientId:       "client_1",
			},
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			headers := make(map[string]string)
			for key, value := range test.headers {
				headers[key] = value
			}
			ctx = metadata.NewIncomingContext(ctx, metadata.New(headers))

			result, err := GetRequestContextForGRPC(ctx)
			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

// TestGetRequestContext_EdgeCases tests additional edge cases for HTTP requests
func TestGetRequestContext_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userContext, _ := enum.ParseUserContext("anonymous")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedResult *RequestContext
		expectedError  string
	}{
		{
			name: "Zero app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "0",
				httpHeaders.HeaderClientId:       "client_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 0,
				ClientId:       "client_1",
			},
			expectedError: "",
		},
		{
			name: "Empty string app version code",
			headers: map[string]string{
				httpHeaders.HeaderUserId:         "12345",
				httpHeaders.HeaderUserContext:    "anonymous",
				httpHeaders.HeaderAppVersionCode: "",
				httpHeaders.HeaderClientId:       "client_1",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 0,
				ClientId:       "client_1",
			},
			expectedError: "",
		},
		{
			name: "Minimum required headers only",
			headers: map[string]string{
				httpHeaders.HeaderUserId:      "12345",
				httpHeaders.HeaderUserContext: "anonymous",
			},
			expectedResult: &RequestContext{
				UserId:         "12345",
				UserContext:    userContext,
				AppVersionCode: 0,
				ClientId:       "",
				UserStateCode:  "",
				UserPinCode:    "",
				UserCity:       "",
				FeedSession:    "",
				AppSession:     "",
			},
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			request, _ := http.NewRequest(http.MethodGet, "/", nil)
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			context.Request = request

			result, err := GetRequestContext(context)
			if test.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			} else {
				assert.EqualError(t, err, test.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}
