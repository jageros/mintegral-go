package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// AppService 提供按包名查询应用名称的接口。
type AppService struct{ client *Client }

// AppNameRequest 是应用名称查询请求。
type AppNameRequest struct {
	// PackageName 是逗号分隔的 Android 包名或 iOS App Store ID。
	PackageName string `json:"package_name"`
}

// AppName 是包名匹配到的应用名称。
type AppName struct {
	// PackageName 是请求对应的包名或 App Store ID。
	PackageName string `json:"package_name"`
	// AppName 是匹配到的应用名称。
	AppName string `json:"app_name"`
}

// Names 按一个或多个逗号分隔包名查询应用名称。
func (s *AppService) Names(ctx context.Context, request AppNameRequest, options ...RequestOption) ([]AppName, error) {
	if strings.TrimSpace(request.PackageName) == "" {
		return nil, fmt.Errorf("%w: package name is required", ErrInvalidRequest)
	}
	return doJSON[[]AppName](ctx, s.client, requestSpec{
		operation: "app.names", method: http.MethodPost, path: "/api/open/v1/target-apps/app-name",
		body: jsonBody(request), contentType: "application/json", authenticated: true, outcomeRisk: true,
	}, options)
}
