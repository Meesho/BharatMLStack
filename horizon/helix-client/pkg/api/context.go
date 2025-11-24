package api

import (
	netHttp "net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/api/http"
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
		UserId:         context.Request.Header.Get(http.HeaderMeeshoUserId),
		UserContext:    context.Request.Header.Get(http.HeaderMeeshoUserContext),
		AppVersionCode: appVersionCode,
		ClientId:       context.Request.Header.Get(http.HeaderMeeshoClientId),
		Pincode:        context.Request.Header.Get(http.HeaderMeeshoUserPincode),
	}
}

// Deprecated: UpdateHeaders Update headers from request meta
func UpdateHeaders(header *netHttp.Header, requestMeta *RequestMeta) {
	header.Set(http.HeaderMeeshoUserId, requestMeta.UserId)
	header.Set(http.HeaderMeeshoUserContext, requestMeta.UserContext)
	header.Set(http.HeaderAppVersionCode, strconv.Itoa(requestMeta.AppVersionCode))
	header.Set(http.HeaderMeeshoClientId, requestMeta.ClientId)
	header.Set(http.HeaderMeeshoUserPincode, requestMeta.Pincode)
}
