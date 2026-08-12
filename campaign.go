package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const campaignPath = "/api/open/v1/campaign"

// CampaignService 提供 Mintegral 广告活动接口。
type CampaignService struct{ client *Client }

// List 查询广告活动列表。当前 Mintegral 文档要求 GET 请求携带 JSON 请求体。
//
//nolint:gocritic // 公共 SDK 签名使用值请求，保持与既有服务调用方式一致。
func (s *CampaignService) List(ctx context.Context, request CampaignListRequest, options ...RequestOption) (CampaignPage, error) {
	if err := validateCampaignListRequest(&request); err != nil {
		return CampaignPage{}, err
	}
	page, err := doJSON[campaignPageWire](ctx, s.client, campaignListSpec(campaignListBody(&request)), options)
	if err != nil {
		return CampaignPage{}, err
	}
	return parseCampaignPage(&page)
}

// Create 创建广告活动。
//
//nolint:gocritic // 公共 SDK 签名使用值请求，保持与既有服务调用方式一致。
func (s *CampaignService) Create(ctx context.Context, request CreateCampaignRequest, options ...RequestOption) (Campaign, error) {
	if err := validateCreateCampaignRequest(&request); err != nil {
		return Campaign{}, err
	}
	campaign, err := doJSON[campaignWire](ctx, s.client, campaignCreateSpec(&request), options)
	if err != nil {
		return Campaign{}, err
	}
	return parseCampaign(&campaign)
}

// Update 更新广告活动；请求必须包含 CampaignID。
//
//nolint:gocritic // 公共 SDK 签名使用值请求，保持与既有服务调用方式一致。
func (s *CampaignService) Update(ctx context.Context, request UpdateCampaignRequest, options ...RequestOption) (Campaign, error) {
	if err := validateUpdateCampaignRequest(&request); err != nil {
		return Campaign{}, err
	}
	campaign, err := doJSON[campaignWire](ctx, s.client, campaignUpdateSpec(&request), options)
	if err != nil {
		return Campaign{}, err
	}
	return parseCampaign(&campaign)
}

func validateCampaignListRequest(request *CampaignListRequest) error {
	if len(request.CampaignIDs) > 50 || request.Page < 0 || request.Limit < 0 || request.Limit > 50 {
		return fmt.Errorf("%w: campaign list bounds are invalid", ErrInvalidRequest)
	}
	for _, campaignID := range request.CampaignIDs {
		if campaignID <= 0 {
			return fmt.Errorf("%w: campaign ID must be positive", ErrInvalidRequest)
		}
	}
	if request.OfferID < 0 {
		return fmt.Errorf("%w: offer ID must not be negative", ErrInvalidRequest)
	}
	return nil
}

func validateCreateCampaignRequest(request *CreateCampaignRequest) error {
	if strings.TrimSpace(request.CampaignName) == "" || !validCampaignPromotionType(request.PromotionType) ||
		len(request.CampaignName) > 100 || len(request.ProductName) > 100 || len(request.Description) > 4000 {
		return fmt.Errorf("%w: campaign request bounds are invalid", ErrInvalidRequest)
	}
	if !validOptionalCampaignYesNo(request.IsCOPPA) || !validOptionalCampaignYesNo(request.AliveInStore) || !validOptionalCampaignPlatform(request.Platform) {
		return fmt.Errorf("%w: campaign enum is invalid", ErrInvalidRequest)
	}
	return validateCampaignAppSize(request.AppSize)
}

func validateUpdateCampaignRequest(request *UpdateCampaignRequest) error {
	if request.CampaignID <= 0 || stringTooLong(request.CampaignName, 100) || stringTooLong(request.ProductName, 100) || stringTooLong(request.Description, 4000) {
		return fmt.Errorf("%w: campaign request bounds are invalid", ErrInvalidRequest)
	}
	if !validCampaignYesNoPointer(request.IsCOPPA) || !validCampaignPromotionTypePointer(request.PromotionType) {
		return fmt.Errorf("%w: campaign enum is invalid", ErrInvalidRequest)
	}
	return validateCampaignAppSize(request.AppSize)
}

func validCampaignYesNo(value CampaignYesNo) bool { return value.Known() }

func validOptionalCampaignYesNo(value CampaignYesNo) bool {
	return value == "" || validCampaignYesNo(value)
}

func validCampaignPromotionType(value CampaignPromotionType) bool {
	return value.Known()
}

func validCampaignPlatform(value CampaignPlatform) bool {
	return value.Known()
}

func validOptionalCampaignPlatform(value CampaignPlatform) bool {
	return value == "" || validCampaignPlatform(value)
}

func validCampaignYesNoPointer(value *CampaignYesNo) bool {
	return value == nil || validCampaignYesNo(*value)
}

func validCampaignPromotionTypePointer(value *CampaignPromotionType) bool {
	return value == nil || validCampaignPromotionType(*value)
}

func validateCampaignAppSize(value *DecimalText) error {
	if value != nil && strings.HasPrefix(value.String(), "-") {
		return fmt.Errorf("%w: application size must not be negative", ErrInvalidRequest)
	}
	return nil
}

func stringTooLong(value *string, maximum int) bool { return value != nil && len(*value) > maximum }

func campaignListSpec(body *campaignListJSON) requestSpec {
	return requestSpec{operation: "campaign.list", method: http.MethodGet, path: campaignPath, body: jsonBody(body), contentType: "application/json", authenticated: true, retryable: true}
}

func campaignCreateSpec(body *CreateCampaignRequest) requestSpec {
	return requestSpec{operation: "campaign.create", method: http.MethodPost, path: campaignPath, body: jsonBody(body), contentType: "application/json", authenticated: true, outcomeRisk: true}
}

func campaignUpdateSpec(body *UpdateCampaignRequest) requestSpec {
	return requestSpec{operation: "campaign.update", method: http.MethodPut, path: campaignPath, body: jsonBody(body), contentType: "application/json", authenticated: true, outcomeRisk: true}
}

type campaignListJSON struct {
	CampaignID   string  `json:"campaign_id,omitempty"`
	CampaignName string  `json:"campaign_name,omitempty"`
	PackageName  string  `json:"package_name,omitempty"`
	OfferID      OfferID `json:"offer_id,omitempty"`
	OfferName    string  `json:"offer_name,omitempty"`
	OfferUUID    string  `json:"offer_uuid,omitempty"`
	Page         int     `json:"page,omitempty"`
	Limit        int     `json:"limit,omitempty"`
}

func campaignListBody(request *CampaignListRequest) *campaignListJSON {
	return &campaignListJSON{
		CampaignID:   strings.Join(campaignIDsText(request.CampaignIDs), ","),
		CampaignName: request.CampaignName,
		PackageName:  request.PackageName,
		OfferID:      request.OfferID,
		OfferName:    request.OfferName,
		OfferUUID:    request.OfferUUID,
		Page:         request.Page,
		Limit:        request.Limit,
	}
}

func campaignIDsText(ids []CampaignID) []string {
	values := make([]string, len(ids))
	for index, id := range ids {
		values[index] = fmt.Sprintf("%d", id)
	}
	return values
}

type campaignPageWire struct {
	List  []campaignWire `json:"list"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int            `json:"total"`
}

type campaignWire struct {
	CampaignID    CampaignID         `json:"campaign_id"`
	CampaignName  string             `json:"campaign_name"`
	IsCOPPA       string             `json:"is_coppa"`
	PromotionType string             `json:"promotion_type"`
	AliveInStore  string             `json:"alive_in_store"`
	PreviewURL    string             `json:"preview_url"`
	ProductName   string             `json:"product_name"`
	PackageName   string             `json:"package_name"`
	Description   string             `json:"description"`
	Icon          string             `json:"icon"`
	Platform      string             `json:"platform"`
	Category      string             `json:"category"`
	AppSize       string             `json:"app_size"`
	MinVersion    string             `json:"min_version"`
	MaintainBy    CampaignMaintainer `json:"maintain_by"`
	Status        CampaignStatus     `json:"status"`
}

func parseCampaignPage(page *campaignPageWire) (CampaignPage, error) {
	list := make([]Campaign, len(page.List))
	for index := range page.List {
		campaign, err := parseCampaign(&page.List[index])
		if err != nil {
			return CampaignPage{}, err
		}
		list[index] = campaign
	}
	return CampaignPage{List: list, Page: page.Page, Limit: page.Limit, Total: page.Total}, nil
}

func parseCampaign(wire *campaignWire) (Campaign, error) {
	appSize := DecimalText("")
	if wire.AppSize != "" {
		parsed, err := ParseDecimalText(wire.AppSize)
		if err != nil {
			return Campaign{}, fmt.Errorf("%w: campaign app_size must be a decimal string", ErrUnexpectedResponse)
		}
		appSize = parsed
	}
	return Campaign{
		CampaignID: wire.CampaignID, CampaignName: wire.CampaignName, IsCOPPA: CampaignYesNo(wire.IsCOPPA),
		PromotionType: CampaignPromotionType(wire.PromotionType), AliveInStore: CampaignYesNo(wire.AliveInStore), PreviewURL: wire.PreviewURL,
		ProductName: wire.ProductName, PackageName: wire.PackageName, Description: wire.Description,
		Icon: wire.Icon, Platform: CampaignPlatform(wire.Platform), Category: wire.Category, AppSize: appSize,
		MinVersion: wire.MinVersion, MaintainBy: wire.MaintainBy, Status: wire.Status,
	}, nil
}
