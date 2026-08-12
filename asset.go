package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CreativeType 是素材类型；未知字符串值会原样保留以兼容服务端扩展。
type CreativeType string

const (
	// CreativeTypeImage 表示图片素材。
	CreativeTypeImage CreativeType = "IMAGE"
	// CreativeTypeVideo 表示视频素材。
	CreativeTypeVideo CreativeType = "VIDEO"
	// CreativeTypePlayable 表示试玩素材。
	CreativeTypePlayable CreativeType = "PLAYABLE"
)

// Known 报告素材类型是否为当前 SDK 已知值。
func (creativeType CreativeType) Known() bool {
	switch creativeType {
	case CreativeTypeImage, CreativeTypeVideo, CreativeTypePlayable:
		return true
	default:
		return false
	}
}

// AssetPlatform 是试玩素材适用平台；未知字符串值会原样保留以兼容服务端扩展。
type AssetPlatform string

const (
	// AssetPlatformAndroid 表示 Android 平台。
	AssetPlatformAndroid AssetPlatform = "ANDROID"
	// AssetPlatformIOS 表示 iOS 平台。
	AssetPlatformIOS AssetPlatform = "IOS"
	// AssetPlatformAll 表示所有平台。
	AssetPlatformAll AssetPlatform = "ALL"
)

// Known 报告试玩素材平台是否为当前 SDK 已知值。
func (platform AssetPlatform) Known() bool {
	return platform == AssetPlatformAndroid || platform == AssetPlatformIOS || platform == AssetPlatformAll
}

// AssetListRequest 是素材库列表筛选条件。
type AssetListRequest struct {
	// CreativeMD5 是内容摘要筛选，最多 200 项。
	CreativeMD5 []ContentMD5
	// CreativeName 是素材名称筛选。
	CreativeName string
	// CreativeType 是素材类型筛选。
	CreativeType CreativeType
	// Resolutions 是分辨率筛选，最多 200 项。
	Resolutions []string
	// Page 是从 1 开始的页码。
	Page int
	// Limit 是每页数量，最大 200。
	Limit int
}

// Asset 描述素材库中的一项素材。
type Asset struct {
	// CreativeType 是素材类型。
	CreativeType CreativeType `json:"creative_type"`
	// CreativeMD5 是素材内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
	// CreativeName 是素材名称。
	CreativeName string `json:"creative_name"`
	// Resolution 是素材分辨率。
	Resolution string `json:"resolution"`
	// Size 是以 KB 为单位的素材大小。
	Size DecimalText `json:"size"`
	// Language 是试玩素材语言，非试玩素材可为空。
	Language *string `json:"language"`
	// Platform 是试玩素材平台，非试玩素材可为空。
	Platform *AssetPlatform `json:"platform"`
}

// AssetList 是素材库分页结果。
type AssetList struct {
	// List 是当前页素材。
	List []Asset `json:"list"`
	// Page 是当前页码。
	Page int `json:"page"`
	// Limit 是每页数量。
	Limit int `json:"limit"`
	// Total 是匹配素材总数。
	Total int `json:"total"`
}

// UploadedAsset 是素材上传结果。
type UploadedAsset struct {
	// CreativeMD5 是已上传素材的内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
	// CreativeName 是服务端确认的素材名称。
	CreativeName string `json:"creative_name"`
}

// AssetService 提供素材库查询和上传。
type AssetService struct{ client *Client }

// Assets 返回绑定当前客户端的素材服务。
func (c *Client) Assets() *AssetService { return &AssetService{client: c} }

// List 查询素材库。
func (s *AssetService) List(ctx context.Context, request AssetListRequest, options ...RequestOption) (AssetList, error) { //nolint:gocritic // 公共 API 约定使用值请求，避免调用方共享可变请求。
	if request.Page == 0 {
		request.Page = 1
	}
	if request.Limit == 0 {
		request.Limit = 200
	}
	if request.Page < 1 || request.Limit < 1 || request.Limit > 200 || len(request.CreativeMD5) > 200 || len(request.Resolutions) > 200 || (request.CreativeType != "" && !request.CreativeType.Known()) {
		return AssetList{}, fmt.Errorf("%w: invalid asset list request", ErrInvalidRequest)
	}
	for _, value := range request.CreativeMD5 {
		if _, err := ParseContentMD5(value.String()); err != nil {
			return AssetList{}, err
		}
	}
	spec := requestSpec{operation: "asset.list", method: http.MethodGet, path: "/api/open/v1/creatives/source", authenticated: true, retryable: true, query: func() (url.Values, error) {
		values := make(url.Values)
		values.Set("page", strconv.Itoa(request.Page))
		values.Set("limit", strconv.Itoa(request.Limit))
		if len(request.CreativeMD5) > 0 {
			entries := make([]string, len(request.CreativeMD5))
			for index, value := range request.CreativeMD5 {
				entries[index] = value.String()
			}
			values.Set("creative_md5", strings.Join(entries, ","))
		}
		if request.CreativeName != "" {
			values.Set("creative_name", request.CreativeName)
		}
		if request.CreativeType != "" {
			values.Set("creative_type", string(request.CreativeType))
		}
		if len(request.Resolutions) > 0 {
			values.Set("resolution", strings.Join(request.Resolutions, ","))
		}
		return values, nil
	}}
	return doJSON[AssetList](ctx, s.client, spec, options)
}

// UploadMedia 上传图片或视频素材。
func (s *AssetService) UploadMedia(ctx context.Context, source UploadSource, options ...RequestOption) (UploadedAsset, error) {
	return s.upload(ctx, source, "/api/open/v1/creatives/upload", "asset.upload_media", options)
}

// UploadPlayable 上传试玩素材。
func (s *AssetService) UploadPlayable(ctx context.Context, source UploadSource, options ...RequestOption) (UploadedAsset, error) {
	return s.upload(ctx, source, "/api/open/v1/playable/upload", "asset.upload_playable", options)
}

func (s *AssetService) upload(ctx context.Context, source UploadSource, path, operation string, options []RequestOption) (UploadedAsset, error) {
	body, contentType, err := multipartFileBody(source)
	if err != nil {
		return UploadedAsset{}, err
	}
	spec := requestSpec{operation: operation, method: http.MethodPost, path: path, target: storageTarget, body: body, contentType: contentType, authenticated: true, outcomeRisk: true}
	return doJSON[UploadedAsset](ctx, s.client, spec, options)
}
