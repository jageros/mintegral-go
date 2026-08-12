package mintegral

// OfferListRequest 是广告单元列表筛选条件。
type OfferListRequest struct {
	// CampaignIDs 按广告活动 ID 筛选。
	CampaignIDs []CampaignID
	// CampaignName 按广告活动名称模糊筛选。
	CampaignName string
	// OfferIDs 按广告单元 ID 筛选。
	OfferIDs []OfferID
	// OfferName 按广告单元名称模糊筛选。
	OfferName string
	// OfferUUID 按广告单元 UUID 筛选。
	OfferUUID string
	// PackageName 按应用包名模糊筛选。
	PackageName string
	// ExtendedFields 指定扩展响应字段。
	ExtendedFields []OfferExtendedField
	// Statuses 按广告单元状态筛选。
	Statuses []OfferStatus
	// Page 是从 1 开始的页码；零使用服务端默认值。
	Page int
	// Limit 是每页数量，最大 50；零使用服务端默认值。
	Limit int
}

// CreateOfferRequest 是创建广告单元的请求。
// TargetGeo 使用官方接口的逗号分隔国家代码文本，也可为 ALL。
type CreateOfferRequest struct {
	// CampaignID 是所属广告活动 ID。
	CampaignID CampaignID `json:"campaign_id"`
	// OfferName 是广告单元名称。
	OfferName string `json:"offer_name"`
	// PromoteTimezone 是投放时区。
	PromoteTimezone DecimalText `json:"promote_timezone"`
	// StartTime 是投放开始时间。
	StartTime UnixSeconds `json:"start_time"`
	// EndTime 是可选结束时间。
	EndTime *UnixSeconds `json:"end_time,omitempty"`
	// TargetGeo 是投放地区或 ALL。
	TargetGeo string `json:"target_geo"`
	// BillingType 是计费模式。
	BillingType BillingType `json:"billing_type"`
	// TargetAdType 是广告展示类型。
	TargetAdType AdTypeSelection `json:"target_ad_type"`
	// BidGoal 是优化目标类型透明请求文本。
	BidGoal string `json:"bid_goal,omitempty"`
	// TargetMTGEvent 是标准优化事件。
	TargetMTGEvent []string `json:"target_mtg_event,omitempty"`
	// OriginalEvent 是原始优化事件。
	OriginalEvent string `json:"original_event,omitempty"`
	// TargetGoalWindow 是优化时间窗。
	TargetGoalWindow string `json:"target_goal_window,omitempty"`
	// TargetGoal 是广告单元优化目标。
	TargetGoal *DecimalText `json:"target_goal,omitempty"`
	// BidRate 是默认出价。
	BidRate DecimalText `json:"bid_rate"`
	// DailyCapType 是每日限额类型。
	DailyCapType DailyCapType `json:"daily_cap_type"`
	// DailyCap 是每日限额或 OPEN。
	DailyCap *BudgetAmount `json:"daily_cap,omitempty"`
	// TotalBudget 是总预算或 OPEN。
	TotalBudget *BudgetAmount `json:"total_budget,omitempty"`
	// SettlementEvent 是 CPE 结算事件。
	SettlementEvent string `json:"settlement_event,omitempty"`
	// OSVersionMin 是最低系统版本。
	OSVersionMin string `json:"os_version_min"`
	// OSVersionMax 是最高系统版本。
	OSVersionMax string `json:"os_version_max,omitempty"`
	// CustomAdSchedule 是投放排期。
	CustomAdSchedule *AdSchedule `json:"custom_ad_schedule,omitempty"`
	// Network 是网络定向。
	Network NetworkSelection `json:"network,omitempty"`
	// TargetDevice 是设备定向。
	TargetDevice TargetDeviceSelection `json:"target_device,omitempty"`
	// TargetGoalByGeo 是地区优化目标。
	TargetGoalByGeo *[]GeoTargetGoal `json:"target_goal_by_geo,omitempty"`
	// CreativeSets 是初始创意组。
	CreativeSets []OfferCreativeSet `json:"creative_sets,omitempty"`
}

// UpdateOfferRequest 是广告单元基础信息的部分更新请求。
type UpdateOfferRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// OfferName 是可选广告单元名称。
	OfferName *string `json:"offer_name,omitempty"`
	// PromoteTimezone 是可选投放时区。
	PromoteTimezone *DecimalText `json:"promote_timezone,omitempty"`
	// StartTime 是可选开始时间。
	StartTime *UnixSeconds `json:"start_time,omitempty"`
	// EndTime 是可选结束时间。
	EndTime *UnixSeconds `json:"end_time,omitempty"`
	// TargetGeo 是可选投放地区。
	TargetGeo *string `json:"target_geo,omitempty"`
	// OSVersionMin 是可选最低系统版本。
	OSVersionMin *string `json:"os_version_min,omitempty"`
	// CustomAdSchedule 是可选投放排期。
	CustomAdSchedule *AdSchedule `json:"custom_ad_schedule,omitempty"`
	// Network 是可选网络定向。
	Network *NetworkSelection `json:"network,omitempty"`
	// TargetDevice 是可选设备定向。
	TargetDevice *TargetDeviceSelection `json:"target_device,omitempty"`
}

// UpdateOfferBidsRequest 是广告单元出价的全量设置。
type UpdateOfferBidsRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// BidRate 是可选默认出价。
	BidRate *DecimalText `json:"bid_rate,omitempty"`
	// BidRateByLocation 是地区出价；空数组表示清除。
	BidRateByLocation *[]LocationBid `json:"bid_rate_by_location,omitempty"`
	// BidRateByMTGID 是发布商出价；空数组表示清除。
	BidRateByMTGID *[]MTGIDBid `json:"bid_rate_by_mtgid,omitempty"`
}

// UpdateOfferBudgetRequest 是广告单元预算的全量设置。
type UpdateOfferBudgetRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// Budget 是完整预算设置。
	Budget []OfferBudget `json:"budget"`
}

// SetOfferStatusRequest 是广告单元启停请求。
type SetOfferStatusRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// Status 只能为 RUNNING 或 STOPPED。
	Status OfferStatus `json:"status"`
}

// UpdateTrafficDeliveryRequest 是发布商流量投放更新请求。
type UpdateTrafficDeliveryRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// Option 是流量投放操作。
	Option OfferTargetOption `json:"option"`
	// MTGIDs 是发布商标识；ALLOW_ALL 时可为空。
	MTGIDs []string `json:"-"`
}

// UpdateOfferTrackingRequest 是归因跟踪配置更新请求。
type UpdateOfferTrackingRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// TrackingMethod 是归因跟踪平台。
	TrackingMethod TrackingMethod `json:"tracking_method"`
	// ClickURL 是点击归因地址。
	ClickURL string `json:"click_url"`
	// ImpressionURL 是可选展示归因地址。
	ImpressionURL string `json:"impression_url,omitempty"`
	// SupportServerClick 是可选服务端点击开关。
	SupportServerClick YesNo `json:"support_server_click,omitempty"`
}

// SetOfferAudiencesRequest 是广告单元人群定向的全量设置。
type SetOfferAudiencesRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// IncludeAudienceIDs 是完整包含列表；空数组表示清除。
	IncludeAudienceIDs *[]AudienceID `json:"include_ta_id,omitempty"`
	// ExcludeAudienceIDs 是完整排除列表；空数组表示清除。
	ExcludeAudienceIDs *[]AudienceID `json:"exclude_ta_id,omitempty"`
}

// UpdateOfferTargetGoalRequest 是广告单元优化目标更新请求。
type UpdateOfferTargetGoalRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// TargetGoal 是广告单元优化目标。
	TargetGoal *DecimalText `json:"target_goal,omitempty"`
	// TargetGoalByGeo 是地区目标；空数组表示清除。
	TargetGoalByGeo *[]GeoTargetGoal `json:"target_goal_by_geo,omitempty"`
}

// ApplyOfferCreativesRequest 是旧版广告单元素材应用请求。
type ApplyOfferCreativesRequest struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// AdType 是素材适用广告展示类型。
	AdType AdTypeSelection `json:"ad_type"`
	// Creatives 是要启停的素材。
	Creatives []OfferCreative `json:"creative"`
}

// OfferMutationResult 是创建或更新广告单元的标识响应。
type OfferMutationResult struct {
	// OfferID 是服务端返回的广告单元 ID。
	OfferID OfferID `json:"id"`
}

// OfferTracking 是更新后的归因跟踪配置。
type OfferTracking struct {
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// TrackingMethod 是归因跟踪平台。
	TrackingMethod TrackingMethod `json:"tracking_method"`
	// ImpressionURL 是展示归因地址。
	ImpressionURL string `json:"impression_url"`
	// ClickURL 是点击归因地址。
	ClickURL string `json:"click_url"`
	// ClickURLStatus 是点击地址测试状态。
	ClickURLStatus string `json:"click_url_status"`
}
