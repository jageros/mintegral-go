package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// CreativeAdListRequest 是广告创意列表请求。
type CreativeAdListRequest struct {
	// AdIDs 是必填广告 ID 列表。
	AdIDs []AdID `json:"ad_ids"`
	// DemandPackageName 是可选推广包名筛选条件。
	DemandPackageName string `json:"demand_package_name,omitempty"`
	// Page 是从 1 开始的页码。
	Page int `json:"page,omitempty"`
	// Limit 是每页数量，最大 100。
	Limit int `json:"limit,omitempty"`
}

// CreativeAdAsset 描述广告所引用的素材。
type CreativeAdAsset struct {
	// CreativeID 是素材 ID。
	CreativeID CreativeID `json:"creative_id"`
	// CreativeName 是素材名称。
	CreativeName string `json:"creative_name"`
	// CreativeMD5 是素材内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
	// CreativeType 是素材类型。
	CreativeType CreativeType `json:"creative_type"`
	// CreativeURL 是素材访问地址。
	CreativeURL string `json:"creative_url"`
}

// CreativeAd 描述一个由创意组产出的广告。
type CreativeAd struct {
	// AdID 是广告 ID。
	AdID AdID `json:"ad_id"`
	// AdName 是广告名称。
	AdName string `json:"ad_name"`
	// OfferID 是推广商品 ID。
	OfferID OfferID `json:"offer_id"`
	// OfferUUID 是推广商品 UUID。
	OfferUUID string `json:"offer_uuid"`
	// CreativeSetID 是创意组 ID。
	CreativeSetID CreativeSetID `json:"creative_set_id"`
	// CreativeSetName 是创意组名称。
	CreativeSetName string `json:"creative_set_name"`
	// CombinationMethod 是创意组合方式。
	CombinationMethod CombinationMethod `json:"combination_method"`
	// AdOutput 是广告输出类型编号。
	AdOutput AdOutput `json:"ad_output"`
	// Creatives 是广告引用的素材列表。
	Creatives []CreativeAdAsset `json:"creatives"`
}

// CreativeAdList 是广告创意分页结果。
type CreativeAdList struct {
	// List 是当前页广告创意。
	List []CreativeAd `json:"list"`
	// Page 是当前页码。
	Page int `json:"page"`
	// Limit 是每页数量。
	Limit int `json:"limit"`
	// Total 是匹配记录总数。
	Total int `json:"total"`
}

// CreativeAdService 提供广告创意查询。
type CreativeAdService struct{ client *Client }

// CreativeAds 返回绑定当前客户端的广告创意服务。
func (c *Client) CreativeAds() *CreativeAdService { return &CreativeAdService{client: c} }

// List 使用 GET JSON body 查询广告创意。
func (s *CreativeAdService) List(ctx context.Context, request CreativeAdListRequest, options ...RequestOption) (CreativeAdList, error) {
	if request.Page == 0 {
		request.Page = 1
	}
	if request.Limit == 0 {
		request.Limit = 20
	}
	if len(request.AdIDs) == 0 || request.Page < 1 || request.Limit < 1 || request.Limit > 100 || strings.TrimSpace(request.DemandPackageName) != request.DemandPackageName {
		return CreativeAdList{}, fmt.Errorf("%w: invalid creative ad list request", ErrInvalidRequest)
	}
	for _, id := range request.AdIDs {
		if id < 1 {
			return CreativeAdList{}, fmt.Errorf("%w: ad IDs must be positive", ErrInvalidRequest)
		}
	}
	spec := requestSpec{operation: "creative_ad.list", method: http.MethodGet, path: "/api/open/v1/creative-ad/list", body: jsonBody(request), contentType: "application/json", authenticated: true, retryable: true}
	return doJSON[CreativeAdList](ctx, s.client, spec, options)
}
