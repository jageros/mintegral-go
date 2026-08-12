package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// OfferService 提供广告单元管理接口。
type OfferService struct{ client *Client }

// Offers 返回与 Client 共享传输和凭据配置的广告单元服务。
func (c *Client) Offers() *OfferService { return &OfferService{client: c} }

// List 查询广告单元列表。
func (s OfferService) List(ctx context.Context, request OfferListRequest, options ...RequestOption) (OfferPage, error) { //nolint:gocritic // 保持已发布的值参数签名兼容性。
	if request.Page < 0 || request.Limit < 0 || request.Limit > 50 {
		return OfferPage{}, fmt.Errorf("%w: offer page and limit are invalid", ErrInvalidRequest)
	}
	for _, campaignID := range request.CampaignIDs {
		if campaignID <= 0 {
			return OfferPage{}, fmt.Errorf("%w: campaign IDs must be positive", ErrInvalidRequest)
		}
	}
	for _, offerID := range request.OfferIDs {
		if offerID <= 0 {
			return OfferPage{}, fmt.Errorf("%w: offer IDs must be positive", ErrInvalidRequest)
		}
	}
	for _, status := range request.Statuses {
		if !status.Known() {
			return OfferPage{}, fmt.Errorf("%w: unsupported offer status", ErrInvalidRequest)
		}
	}
	for _, field := range request.ExtendedFields {
		if field != OfferExtendedBidRateByMTGID && field != OfferExtendedTargetApp {
			return OfferPage{}, fmt.Errorf("%w: unsupported extended field", ErrInvalidRequest)
		}
	}
	return doJSON[OfferPage](ctx, s.client, requestSpec{
		operation: "offer.list", method: http.MethodGet, path: "/api/open/v1/offers",
		query:         func() (url.Values, error) { return offerListQuery(request), nil },
		authenticated: true, retryable: true,
	}, options)
}

// Create 创建广告单元。
func (s OfferService) Create(ctx context.Context, request CreateOfferRequest, options ...RequestOption) (Offer, error) { //nolint:gocritic // 保持已发布的值参数签名兼容性。
	if request.CampaignID <= 0 || strings.TrimSpace(request.OfferName) == "" || request.StartTime <= 0 || strings.TrimSpace(request.TargetGeo) == "" {
		return Offer{}, fmt.Errorf("%w: required create-offer field is missing", ErrInvalidRequest)
	}
	if !request.BillingType.Known() {
		return Offer{}, fmt.Errorf("%w: unsupported billing type", ErrInvalidRequest)
	}
	if !request.TargetAdType.valid() || (request.DailyCapType != "" && !request.DailyCapType.Known()) ||
		(request.Network != "" && !request.Network.valid()) || (request.TargetDevice != "" && !request.TargetDevice.valid()) {
		return Offer{}, fmt.Errorf("%w: unsupported offer targeting value", ErrInvalidRequest)
	}
	return doJSON[Offer](ctx, s.client, offerWriteSpec("offer.create", http.MethodPost, "/api/open/v1/offer", request), options)
}

// Update 部分更新广告单元基础信息。
func (s OfferService) Update(ctx context.Context, request UpdateOfferRequest, options ...RequestOption) (Offer, error) { //nolint:gocritic // 保持已发布的值参数签名兼容性。
	if err := requireOfferID(request.OfferID); err != nil {
		return Offer{}, err
	}
	if (request.Network != nil && !request.Network.valid()) || (request.TargetDevice != nil && !request.TargetDevice.valid()) {
		return Offer{}, fmt.Errorf("%w: unsupported offer targeting value", ErrInvalidRequest)
	}
	return doJSON[Offer](ctx, s.client, offerWriteSpec("offer.update", http.MethodPut, "/api/open/v1/offer", request), options)
}

// UpdateBids 全量更新默认、地区和发布商出价；空数组表示清除对应设置。
func (s OfferService) UpdateBids(ctx context.Context, request UpdateOfferBidsRequest, options ...RequestOption) (OfferMutationResult, error) {
	if err := requireOfferID(request.OfferID); err != nil {
		return OfferMutationResult{}, err
	}
	if request.BidRate == nil && request.BidRateByLocation == nil && request.BidRateByMTGID == nil {
		return OfferMutationResult{}, fmt.Errorf("%w: at least one bid setting is required", ErrInvalidRequest)
	}
	return doJSON[OfferMutationResult](ctx, s.client, offerWriteSpec("offer.update_bids", http.MethodPut, "/api/open/v1/offer/bid_rate", request), options)
}

// UpdateBudget 全量更新广告单元预算。
func (s OfferService) UpdateBudget(ctx context.Context, request UpdateOfferBudgetRequest, options ...RequestOption) error {
	if err := requireOfferID(request.OfferID); err != nil {
		return err
	}
	if request.Budget == nil {
		return fmt.Errorf("%w: budget is required", ErrInvalidRequest)
	}
	spec := offerWriteSpec("offer.update_budget", http.MethodPut, "/api/open/v1/offer/budget", request)
	spec.allowEmptyData = true
	_, err := doJSON[struct{}](ctx, s.client, spec, options)
	return err
}

// SetStatus 启动或暂停广告单元。
func (s OfferService) SetStatus(ctx context.Context, request SetOfferStatusRequest, options ...RequestOption) error {
	if err := requireOfferID(request.OfferID); err != nil {
		return err
	}
	if request.Status != OfferStatusRunning && request.Status != OfferStatusStopped {
		return fmt.Errorf("%w: status must be RUNNING or STOPPED", ErrInvalidRequest)
	}
	spec := offerWriteSpec("offer.set_status", http.MethodPut, "/api/open/v1/offer/status", request)
	spec.allowEmptyData = true
	_, err := doJSON[struct{}](ctx, s.client, spec, options)
	return err
}

// UpdateTrafficDelivery 更新发布商流量投放状态。
func (s OfferService) UpdateTrafficDelivery(ctx context.Context, request UpdateTrafficDeliveryRequest, options ...RequestOption) error {
	if err := requireOfferID(request.OfferID); err != nil {
		return err
	}
	if !validTargetOption(request.Option) || (request.Option != OfferTargetAllowAll && (len(request.MTGIDs) == 0 || len(request.MTGIDs) > 3000)) {
		return fmt.Errorf("%w: traffic option and mtgid are required", ErrInvalidRequest)
	}
	body := struct {
		OfferID OfferID           `json:"offer_id"`
		Option  OfferTargetOption `json:"option"`
		MTGID   string            `json:"mtgid"`
	}{request.OfferID, request.Option, strings.Join(request.MTGIDs, ",")}
	spec := offerWriteSpec("offer.update_traffic", http.MethodPut, "/api/open/v1/offer/target", body)
	spec.allowEmptyData = true
	_, err := doJSON[struct{}](ctx, s.client, spec, options)
	return err
}

// UpdateTracking 更新归因跟踪配置。
func (s OfferService) UpdateTracking(ctx context.Context, request UpdateOfferTrackingRequest, options ...RequestOption) (OfferTracking, error) {
	if err := requireOfferID(request.OfferID); err != nil {
		return OfferTracking{}, err
	}
	if !request.TrackingMethod.Known() || strings.TrimSpace(request.ClickURL) == "" ||
		(request.SupportServerClick != "" && !request.SupportServerClick.Known()) {
		return OfferTracking{}, fmt.Errorf("%w: tracking method and click URL are required", ErrInvalidRequest)
	}
	return doJSON[OfferTracking](ctx, s.client, offerWriteSpec("offer.update_tracking", http.MethodPut, "/api/open/v1/tracking", request), options)
}

// SetAudiences 全量更新包含和排除的人群；空数组表示清除对应设置。
func (s OfferService) SetAudiences(ctx context.Context, request SetOfferAudiencesRequest, options ...RequestOption) error {
	if err := requireOfferID(request.OfferID); err != nil {
		return err
	}
	if request.IncludeAudienceIDs == nil && request.ExcludeAudienceIDs == nil {
		return fmt.Errorf("%w: at least one audience setting is required", ErrInvalidRequest)
	}
	spec := offerWriteSpec("offer.set_audiences", http.MethodPut, "/api/open/v1/offer/target-audience", request)
	spec.allowEmptyData = true
	_, err := doJSON[struct{}](ctx, s.client, spec, options)
	return err
}

// UpdateTargetGoal 更新广告单元和地区优化目标；空数组会清除地区目标。
func (s OfferService) UpdateTargetGoal(ctx context.Context, request UpdateOfferTargetGoalRequest, options ...RequestOption) error {
	if err := requireOfferID(request.OfferID); err != nil {
		return err
	}
	if request.TargetGoal == nil && request.TargetGoalByGeo == nil {
		return fmt.Errorf("%w: at least one target goal is required", ErrInvalidRequest)
	}
	spec := offerWriteSpec("offer.update_target_goal", http.MethodPut, "/api/open/v3/offer/target_goal", request)
	spec.allowEmptyData = true
	_, err := doJSON[struct{}](ctx, s.client, spec, options)
	return err
}

// ApplyCreatives 使用即将废弃的旧版接口更新素材。
//
// Deprecated: 请改用创意组接口。
func (s OfferService) ApplyCreatives(ctx context.Context, request ApplyOfferCreativesRequest, options ...RequestOption) ([]string, error) {
	if err := requireOfferID(request.OfferID); err != nil {
		return nil, err
	}
	if !request.AdType.valid() || len(request.Creatives) == 0 {
		return nil, fmt.Errorf("%w: ad type and creatives are required", ErrInvalidRequest)
	}
	for _, creative := range request.Creatives {
		if _, err := ParseContentMD5(string(creative.CreativeMD5)); err != nil {
			return nil, err
		}
		if strings.TrimSpace(creative.CreativeName) == "" || strings.TrimSpace(creative.ApplyInArea) == "" ||
			(creative.Option != OfferTargetEnable && creative.Option != OfferTargetDisable) {
			return nil, fmt.Errorf("%w: invalid creative apply field", ErrInvalidRequest)
		}
	}
	return doJSON[[]string](ctx, s.client, offerWriteSpec("offer.apply_creatives", http.MethodPut, "/api/open/v1/offer/apply_creative", request), options)
}

func offerWriteSpec[T any](operation, method, path string, request T) requestSpec {
	return requestSpec{operation: operation, method: method, path: path, body: jsonBody(request), contentType: "application/json", authenticated: true, outcomeRisk: true}
}

func requireOfferID(offerID OfferID) error {
	if offerID <= 0 {
		return fmt.Errorf("%w: offer ID must be positive", ErrInvalidRequest)
	}
	return nil
}

func validTargetOption(option OfferTargetOption) bool {
	return option == OfferTargetEnable || option == OfferTargetDisable || option == OfferTargetAllowAll
}

func offerListQuery(request OfferListRequest) url.Values { //nolint:gocritic // 与公开 List 保持同一值语义且只读。
	query := make(url.Values)
	setInt64List(query, "campaign_id", request.CampaignIDs)
	setInt64List(query, "offer_id", request.OfferIDs)
	setString(query, "campaign_name", request.CampaignName)
	setString(query, "offer_name", request.OfferName)
	setString(query, "offer_uuid", request.OfferUUID)
	setString(query, "package_name", request.PackageName)
	setTypedStringList(query, "ext_fields", request.ExtendedFields)
	setTypedStringList(query, "status", request.Statuses)
	if request.Page > 0 {
		query.Set("page", strconv.Itoa(request.Page))
	}
	if request.Limit > 0 {
		query.Set("limit", strconv.Itoa(request.Limit))
	}
	return query
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setInt64List[T ~int64](query url.Values, key string, values []T) {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.FormatInt(int64(value), 10)
	}
	if len(parts) > 0 {
		query.Set(key, strings.Join(parts, ","))
	}
}

func setTypedStringList[T ~string](query url.Values, key string, values []T) {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = string(value)
	}
	if len(parts) > 0 {
		query.Set(key, strings.Join(parts, ","))
	}
}
