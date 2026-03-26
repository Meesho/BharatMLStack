package api

import (
	netHttp "net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/go-sdk/pkg/api/http"
	"github.com/gin-gonic/gin"
)

// Deprecated: Use RequestContext instead of RequestMeta
type RequestMeta struct {
	UserId         string
	UserContext    string
	AppVersionCode int
	ClientId       string
	Pincode        string
	AppSession     string
	FeedSession    string
}

// Deprecated: GetRequestMeta Build RequestMeta from gin context
func GetRequestMeta(context *gin.Context) *RequestMeta {
	rawAppVersionCode := context.Request.Header.Get(http.HeaderAppVersionCode)
	appVersionCode, _ := strconv.Atoi(rawAppVersionCode)
	return &RequestMeta{
		UserId:         context.Request.Header.Get(http.HeaderUserId),
		UserContext:    context.Request.Header.Get(http.HeaderUserContext),
		AppVersionCode: appVersionCode,
		ClientId:       context.Request.Header.Get(http.HeaderClientId),
		Pincode:        context.Request.Header.Get(http.HeaderUserPincode),
	}
}

// Deprecated: UpdateHeaders Update headers from request meta
func UpdateHeaders(header *netHttp.Header, requestMeta *RequestMeta) {
	header.Set(http.HeaderUserId, requestMeta.UserId)
	header.Set(http.HeaderUserContext, requestMeta.UserContext)
	header.Set(http.HeaderAppVersionCode, strconv.Itoa(requestMeta.AppVersionCode))
	header.Set(http.HeaderClientId, requestMeta.ClientId)
	header.Set(http.HeaderUserPincode, requestMeta.Pincode)
}
