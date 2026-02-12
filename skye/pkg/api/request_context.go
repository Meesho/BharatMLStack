package api

import (
	"context"
	"errors"
	"fmt"
	netHttp "net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/skye/pkg/api/http"
	enum "github.com/Meesho/BharatMLStack/skye/pkg/enums"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RequestContext struct {
	UserId              string
	UserContext         enum.UserContext
	AppVersionCode      int
	ClientId            string
	UserStateCode       string
	UserPinCode         string
	UserCity            string
	AppSession          string
	FeedSession         string
	InstanceId          string
	SessionId           string
	MeeshoUserLongitude string
	MeeshoUserLatitude  string
	MeeshoUserAddressId string
	MeeshoUserCountry   string
	MeeshoUserLanguage  string
	MeeshoUserLocation  string
	MeeshoRealEstate    string
	MeeshoXoToken       string
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
	userId, err := getMetadataValue(mdContext, http.HeaderMeeshoUserId)
	if err != nil {
		return nil, err
	}
	if len(userId) == 0 {
		return nil, errors.New("user id is empty in headers")
	}
	requestContext.UserId = userId

	// get user context (mandatory)
	userContext, err := getMetadataValue(mdContext, http.HeaderMeeshoUserContext)
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
	rawAppVersionCode, _ := getMetadataValue(mdContext, http.HeaderAppVersionCode)
	if len(rawAppVersionCode) != 0 {
		if appVersionCode, err := strconv.Atoi(rawAppVersionCode); err != nil {
			return nil, errors.New("app_version_code should be an integer")
		} else {
			requestContext.AppVersionCode = appVersionCode
		}
	}

	// get client id (optional)
	requestContext.ClientId, _ = getMetadataValue(mdContext, http.HeaderMeeshoClientId)

	// get userStateCode
	requestContext.UserStateCode, _ = getMetadataValue(mdContext, http.HeaderMeeshoUserStateCode)

	// get userPinCode
	requestContext.UserPinCode, _ = getMetadataValue(mdContext, http.HeaderMeeshoUserPincode)

	// get userCity
	requestContext.UserCity, _ = getMetadataValue(mdContext, http.HeaderMeeshoUserCity)
	requestContext.FeedSession, _ = getMetadataValue(mdContext, http.HeaderMeeshoFeedSession)
	requestContext.AppSession, _ = getMetadataValue(mdContext, http.HeaderAppSession)
	requestContext.InstanceId, _ = getMetadataValue(mdContext, http.HeaderMeeshoInstanceId)
	requestContext.SessionId, _ = getMetadataValue(mdContext, http.HeaderMeeshoSessionId)
	requestContext.MeeshoUserLongitude, _ = getMetadataValue(mdContext, http.HeaderMeeshoUserLongitude)
	requestContext.MeeshoUserLatitude, _ = getMetadataValue(mdContext, http.HeaderMeeshoUserLatitude)
	requestContext.MeeshoUserAddressId, _ = getMetadataValue(mdContext, http.HeaderMeeshoUserAddressId)
	requestContext.MeeshoUserCountry, _ = getMetadataValue(mdContext, http.HeaderMeeshoCountry)
	requestContext.MeeshoUserLanguage, _ = getMetadataValue(mdContext, http.HeaderMeeshoLanguage)
	requestContext.MeeshoUserLocation, _ = getMetadataValue(mdContext, http.HeaderAppUserLocation)
	requestContext.MeeshoRealEstate, _ = getMetadataValue(mdContext, http.HeaderRealEstate)
	requestContext.MeeshoXoToken, _ = getMetadataValue(mdContext, http.HeaderMeeshoXoToken)
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
	headers[http.HeaderMeeshoUserId] = context.UserId
	headers[http.HeaderMeeshoUserContext] = context.UserContext.String()
	headers[http.HeaderAppVersionCode] = strconv.Itoa(context.AppVersionCode)
	headers[http.HeaderMeeshoClientId] = context.ClientId
	headers[http.HeaderMeeshoUserStateCode] = context.UserStateCode
	headers[http.HeaderMeeshoUserPincode] = context.UserPinCode
	headers[http.HeaderMeeshoUserCity] = context.UserCity
	headers[http.HeaderMeeshoInstanceId] = context.InstanceId
	headers[http.HeaderMeeshoSessionId] = context.SessionId
	headers[http.HeaderMeeshoUserLongitude] = context.MeeshoUserLongitude
	headers[http.HeaderMeeshoUserLatitude] = context.MeeshoUserLatitude
	headers[http.HeaderMeeshoUserAddressId] = context.MeeshoUserAddressId
	headers[http.HeaderMeeshoCountry] = context.MeeshoUserCountry
	headers[http.HeaderMeeshoLanguage] = context.MeeshoUserLanguage
	headers[http.HeaderRealEstate] = context.MeeshoRealEstate
}

// GetRequestContext Build RequestContext from gin context
func GetRequestContext(context *gin.Context) (*RequestContext, error) {
	requestContext := &RequestContext{}
	// get user id (mandatory)
	userId := context.Request.Header.Get(http.HeaderMeeshoUserId)
	if len(userId) == 0 {
		return nil, errors.New("user id is missing in headers")
	}

	requestContext.UserId = userId
	// get user context (mandatory)
	userContext := context.Request.Header.Get(http.HeaderMeeshoUserContext)
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
	requestContext.ClientId = context.Request.Header.Get(http.HeaderMeeshoClientId)

	// get userStateCode
	requestContext.UserStateCode = context.Request.Header.Get(http.HeaderMeeshoUserStateCode)

	// get userPinCode
	requestContext.UserPinCode = context.Request.Header.Get(http.HeaderMeeshoUserPincode)

	// get userCity
	requestContext.UserCity = context.Request.Header.Get(http.HeaderMeeshoUserCity)

	requestContext.FeedSession = context.Request.Header.Get(http.HeaderMeeshoFeedSession)
	requestContext.AppSession = context.Request.Header.Get(http.HeaderAppSession)
	requestContext.InstanceId = context.Request.Header.Get(http.HeaderMeeshoInstanceId)
	requestContext.SessionId = context.Request.Header.Get(http.HeaderMeeshoSessionId)
	requestContext.MeeshoUserLongitude = context.Request.Header.Get(http.HeaderMeeshoUserLongitude)
	requestContext.MeeshoUserLatitude = context.Request.Header.Get(http.HeaderMeeshoUserLatitude)
	requestContext.MeeshoUserAddressId = context.Request.Header.Get(http.HeaderMeeshoUserAddressId)
	requestContext.MeeshoUserCountry = context.Request.Header.Get(http.HeaderMeeshoCountry)
	requestContext.MeeshoUserLanguage = context.Request.Header.Get(http.HeaderMeeshoLanguage)
	requestContext.MeeshoUserLocation = context.Request.Header.Get(http.HeaderAppUserLocation)
	requestContext.MeeshoXoToken = context.Request.Header.Get(http.HeaderMeeshoXoToken)
	return requestContext, nil
}

// UpdateWithHeaders Update headers from request context
func UpdateWithHeaders(header *netHttp.Header, context *RequestContext) {
	header.Set(http.HeaderMeeshoUserId, context.UserId)
	header.Set(http.HeaderMeeshoUserContext, context.UserContext.Name())
	header.Set(http.HeaderAppVersionCode, strconv.Itoa(context.AppVersionCode))
	header.Set(http.HeaderMeeshoClientId, context.ClientId)
	header.Set(http.HeaderMeeshoUserStateCode, context.UserStateCode)
	header.Set(http.HeaderMeeshoUserPincode, context.UserPinCode)
	header.Set(http.HeaderMeeshoUserCity, context.UserCity)
	header.Set(http.HeaderMeeshoInstanceId, context.InstanceId)
	header.Set(http.HeaderMeeshoSessionId, context.SessionId)
	header.Set(http.HeaderMeeshoUserLongitude, context.MeeshoUserLongitude)
	header.Set(http.HeaderMeeshoUserLatitude, context.MeeshoUserLatitude)
	header.Set(http.HeaderMeeshoUserAddressId, context.MeeshoUserAddressId)
	header.Set(http.HeaderMeeshoCountry, context.MeeshoUserCountry)
	header.Set(http.HeaderMeeshoLanguage, context.MeeshoUserLanguage)
	header.Set(http.HeaderMeeshoXoToken, context.MeeshoXoToken)
}
