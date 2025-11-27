package api

import (
	"context"
	"errors"
	"fmt"
	netHttp "net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/api/http"
	enum "github.com/Meesho/BharatMLStack/helix-client/pkg/enums"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RequestContext struct {
	UserId         string
	UserContext    enum.UserContext
	AppVersionCode int
	ClientId       string
	UserStateCode  string
	UserPinCode    string
	UserCity       string
	AppSession     string
	FeedSession    string
	InstanceId     string
	SessionId      string
	UserLongitude  string
	UserLatitude   string
	UserAddressId  string
	UserCountry    string
	UserLanguage   string
	UserLocation   string
}

const (
	RequestContextValue = "REQUEST_CONTEXT"
)

func GetRequestContextForGRPC(ctx context.Context) (*RequestContext, error) {
	requestContext := RequestContext{}
	mdContext, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "metadata is not provided")
	}

	// get user id (mandatory)
	userId, err := getMetadataValue(mdContext, http.HeaderUserId)
	if err != nil {
		return nil, err
	}
	if len(userId) == 0 {
		return nil, errors.New("user id is empty in headers")
	}
	requestContext.UserId = userId

	// get user context (mandatory)
	userContext, err := getMetadataValue(mdContext, http.HeaderUserContext)
	if err != nil {
		return nil, err
	}
	if len(userContext) == 0 {
		return nil, errors.New("user context is empty in headers")
	} else {
		if userCtx, err := enum.ParseUserContext(userContext); err != nil {
			return nil, err
		} else {
			requestContext.UserContext = userCtx
		}
	}

	// get app version code (optional)
	rawAppVersionCode, err := getMetadataValue(mdContext, http.HeaderAppVersionCode)
	if err != nil {
		return nil, err
	}
	if len(rawAppVersionCode) != 0 {
		if appVersionCode, err := strconv.Atoi(rawAppVersionCode); err != nil {
			return nil, errors.New("app_version_code should be an integer")
		} else {
			requestContext.AppVersionCode = appVersionCode
		}
	}

	// get client id (optional)
	requestContext.ClientId, _ = getMetadataValue(mdContext, http.HeaderClientId)

	// get userStateCode
	requestContext.UserStateCode, _ = getMetadataValue(mdContext, http.HeaderUserStateCode)

	// get userPinCode
	requestContext.UserPinCode, _ = getMetadataValue(mdContext, http.HeaderUserPincode)

	// get userCity
	requestContext.UserCity, _ = getMetadataValue(mdContext, http.HeaderUserCity)
	requestContext.FeedSession, _ = getMetadataValue(mdContext, http.HeaderFeedSession)
	requestContext.AppSession, _ = getMetadataValue(mdContext, http.HeaderAppSession)
	requestContext.InstanceId, _ = getMetadataValue(mdContext, http.HeaderInstanceId)
	requestContext.SessionId, _ = getMetadataValue(mdContext, http.HeaderSessionId)
	requestContext.UserLongitude, _ = getMetadataValue(mdContext, http.HeaderUserLongitude)
	requestContext.UserLatitude, _ = getMetadataValue(mdContext, http.HeaderUserLatitude)
	requestContext.UserAddressId, _ = getMetadataValue(mdContext, http.HeaderUserAddressId)
	requestContext.UserCountry, _ = getMetadataValue(mdContext, http.HeaderCountry)
	requestContext.UserLanguage, _ = getMetadataValue(mdContext, http.HeaderLanguage)
	requestContext.UserLocation, _ = getMetadataValue(mdContext, http.HeaderAppUserLocation)
	return &requestContext, nil
}

func getMetadataValue(md metadata.MD, key string) (string, error) {
	values := md.Get(key)
	if len(values) == 0 {
		return "", fmt.Errorf("metadata key '%s' is missing ", key)
	}
	return values[0], nil
}

// UpdateWithHeaders Update headers from request context
func UpdateWithGRPCHeaders(headers map[string]string, context *RequestContext) {
	headers[http.HeaderUserId] = context.UserId
	headers[http.HeaderUserContext] = context.UserContext.String()
	headers[http.HeaderAppVersionCode] = strconv.Itoa(context.AppVersionCode)
	headers[http.HeaderClientId] = context.ClientId
	headers[http.HeaderUserStateCode] = context.UserStateCode
	headers[http.HeaderUserPincode] = context.UserPinCode
	headers[http.HeaderUserCity] = context.UserCity
	headers[http.HeaderInstanceId] = context.InstanceId
	headers[http.HeaderSessionId] = context.SessionId
	headers[http.HeaderUserLongitude] = context.UserLongitude
	headers[http.HeaderUserLatitude] = context.UserLatitude
	headers[http.HeaderUserAddressId] = context.UserAddressId
	headers[http.HeaderCountry] = context.UserCountry
	headers[http.HeaderLanguage] = context.UserLanguage
}

// GetRequestContext Build RequestContext from gin context
func GetRequestContext(context *gin.Context) (*RequestContext, error) {
	requestContext := &RequestContext{}
	// get user id (mandatory)
	userId := context.Request.Header.Get(http.HeaderUserId)
	if len(userId) == 0 {
		return nil, errors.New("user id is missing in headers")
	}

	requestContext.UserId = userId
	// get user context (mandatory)
	userContext := context.Request.Header.Get(http.HeaderUserContext)
	if len(userContext) == 0 {
		return nil, errors.New("user context is missing in headers")
	} else {
		if userCtx, err := enum.ParseUserContext(userContext); err != nil {
			return nil, err
		} else {
			requestContext.UserContext = userCtx
		}
	}

	// get app version code (optional)
	rawAppVersionCode := context.Request.Header.Get(http.HeaderAppVersionCode)
	if len(rawAppVersionCode) != 0 {
		if appVersionCode, err := strconv.Atoi(rawAppVersionCode); err != nil {
			return nil, errors.New("app_version_code should be an integer")
		} else {
			requestContext.AppVersionCode = appVersionCode
		}
	}

	// get client id (optional)
	requestContext.ClientId = context.Request.Header.Get(http.HeaderClientId)

	// get userStateCode
	requestContext.UserStateCode = context.Request.Header.Get(http.HeaderUserStateCode)

	// get userPinCode
	requestContext.UserPinCode = context.Request.Header.Get(http.HeaderUserPincode)

	// get userCity
	requestContext.UserCity = context.Request.Header.Get(http.HeaderUserCity)

	requestContext.FeedSession = context.Request.Header.Get(http.HeaderFeedSession)
	requestContext.AppSession = context.Request.Header.Get(http.HeaderAppSession)
	requestContext.InstanceId = context.Request.Header.Get(http.HeaderInstanceId)
	requestContext.SessionId = context.Request.Header.Get(http.HeaderSessionId)
	requestContext.UserLongitude = context.Request.Header.Get(http.HeaderUserLongitude)
	requestContext.UserLatitude = context.Request.Header.Get(http.HeaderUserLatitude)
	requestContext.UserAddressId = context.Request.Header.Get(http.HeaderUserAddressId)
	requestContext.UserCountry = context.Request.Header.Get(http.HeaderCountry)
	requestContext.UserLanguage = context.Request.Header.Get(http.HeaderLanguage)
	requestContext.UserLocation = context.Request.Header.Get(http.HeaderAppUserLocation)
	return requestContext, nil
}

// UpdateWithHeaders Update headers from request context
func UpdateWithHeaders(header *netHttp.Header, context *RequestContext) {
	header.Set(http.HeaderUserId, context.UserId)
	header.Set(http.HeaderUserContext, context.UserContext.Name())
	header.Set(http.HeaderAppVersionCode, strconv.Itoa(context.AppVersionCode))
	header.Set(http.HeaderClientId, context.ClientId)
	header.Set(http.HeaderUserStateCode, context.UserStateCode)
	header.Set(http.HeaderUserPincode, context.UserPinCode)
	header.Set(http.HeaderUserCity, context.UserCity)
	header.Set(http.HeaderInstanceId, context.InstanceId)
	header.Set(http.HeaderSessionId, context.SessionId)
	header.Set(http.HeaderUserLongitude, context.UserLongitude)
	header.Set(http.HeaderUserLatitude, context.UserLatitude)
	header.Set(http.HeaderUserAddressId, context.UserAddressId)
	header.Set(http.HeaderCountry, context.UserCountry)
	header.Set(http.HeaderLanguage, context.UserLanguage)
}
