package mintegral

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// OfferStatus 是广告单元状态。
type OfferStatus string

// 当前 SDK 已知的广告单元状态。
const (
	OfferStatusRunning             OfferStatus = "RUNNING"
	OfferStatusStopped             OfferStatus = "STOPPED"
	OfferStatusPending             OfferStatus = "PENDING"
	OfferStatusOverCap             OfferStatus = "OVER_CAP"
	OfferStatusOverDailyCap        OfferStatus = "OVER_DAILY_CAP"
	OfferStatusInsufficientBalance OfferStatus = "INSUFFICIENT_ACCOUNT_BALANCE"
	OfferStatusPartiallyOverCap    OfferStatus = "PARTIALLY_OVER_CAP"
)

// Known 报告状态是否为当前 SDK 已知值。
func (value OfferStatus) Known() bool {
	switch value {
	case OfferStatusRunning, OfferStatusStopped, OfferStatusPending, OfferStatusOverCap, OfferStatusOverDailyCap, OfferStatusInsufficientBalance, OfferStatusPartiallyOverCap:
		return true
	default:
		return false
	}
}

// OfferExtendedField 是广告单元列表可选的扩展响应字段。
type OfferExtendedField string

// 广告单元列表支持的扩展响应字段。
const (
	OfferExtendedBidRateByMTGID OfferExtendedField = "bid_rate_by_mtgid"
	OfferExtendedTargetApp      OfferExtendedField = "target_app"
)

// OfferTargetOption 是启用或停用定向项的动作。
type OfferTargetOption string

// 发布商或素材定向支持的操作。
const (
	OfferTargetEnable   OfferTargetOption = "ENABLE"
	OfferTargetDisable  OfferTargetOption = "DISABLE"
	OfferTargetAllowAll OfferTargetOption = "ALLOW_ALL"
)

// AdOutput 是 Mintegral 广告输出格式标识。
type AdOutput int64

// BillingType 是计费模式。
type BillingType string

// 当前 SDK 支持作为请求值的计费模式。
const (
	BillingTypeCPI  BillingType = "CPI"
	BillingTypeCPC  BillingType = "CPC"
	BillingTypeCPM  BillingType = "CPM"
	BillingTypeCPE  BillingType = "CPE"
	BillingTypeOCPI BillingType = "OCPI"
)

// Known 报告计费模式是否为当前 SDK 已知值。
func (value BillingType) Known() bool {
	switch value {
	case BillingTypeCPI, BillingTypeCPC, BillingTypeCPM, BillingTypeCPE, BillingTypeOCPI:
		return true
	default:
		return false
	}
}

// OfferPage 是广告单元列表响应。
type OfferPage struct {
	// List 是当前页广告单元。
	List []Offer `json:"list"`
	// Page 是当前页码。
	Page int `json:"page"`
	// Limit 是每页数量。
	Limit int `json:"limit"`
	// Total 是匹配总数。
	Total int `json:"total"`
}

// Offer 是 Mintegral 广告单元。
type Offer struct {
	// CampaignID 是所属广告活动 ID。
	CampaignID CampaignID `json:"campaign_id"`
	// CampaignName 是所属广告活动名称。
	CampaignName string `json:"campaign_name"`
	// OfferID 是广告单元 ID。
	OfferID OfferID `json:"offer_id"`
	// UUID 是服务端生成的唯一名称。
	UUID string `json:"uuid"`
	// OfferName 是广告单元名称。
	OfferName string `json:"offer_name"`
	// PromoteTimezone 是投放时区。
	PromoteTimezone DecimalText `json:"promote_timezone"`
	// StartTime 是投放开始时间响应文本。
	StartTime string `json:"start_time"`
	// EndTime 是投放结束时间响应文本。
	EndTime string `json:"end_time"`
	// Status 是广告单元状态。
	Status OfferStatus `json:"status"`
	// CountryCode 是列表响应的投放地区。
	CountryCode CountryCode `json:"country_code"`
	// TargetGeo 是写接口响应的投放地区。
	TargetGeo string `json:"target_geo"`
	// Stage 是冷启动阶段的透明字符串。
	Stage string `json:"stage"`
	// BidType 是旧版计费模式的透明字符串。
	BidType string `json:"bid_type"`
	// BillingType 是当前计费模式。
	BillingType BillingType `json:"billing_type"`
	// BidGoal 是优化目标类型。
	BidGoal BidGoal `json:"bid_goal"`
	// BidRate 是默认出价。
	BidRate DecimalText `json:"bid_rate"`
	// BidRateByLocation 是地区出价。
	BidRateByLocation []LocationBid `json:"bid_rate_by_location"`
	// BidRateByMTGID 是发布商出价。
	BidRateByMTGID []MTGIDBid `json:"bid_rate_for_mtgid"`
	// Budget 是地区预算。
	Budget []OfferBudget `json:"budget"`
	// OfferStopReason 是暂停原因。
	OfferStopReason string `json:"offer_stop_reason"`
	// SettlementEvent 是 CPE 结算事件。
	SettlementEvent string `json:"settlement_event"`
	// TrackingMethod 是归因跟踪平台。
	TrackingMethod TrackingMethod `json:"tracking_method"`
	// ClickURL 是点击归因地址。
	ClickURL string `json:"click_url"`
	// ImpressionURL 是展示归因地址。
	ImpressionURL string `json:"impression_url"`
	// OSVersionMin 是最低系统版本。
	OSVersionMin string `json:"os_version_min"`
	// OSVersionMax 是最高系统版本。
	OSVersionMax string `json:"os_version_max"`
	// CustomAdSchedule 是自定义排期响应文本。
	CustomAdSchedule string `json:"custom_ad_schedule"`
	// Network 是网络定向响应值。
	Network NetworkSelection `json:"network"`
	// TargetAdType 是广告展示类型响应值。
	TargetAdType AdTypeSelection `json:"target_ad_type"`
	// Currency 是结算币种透明字符串。
	Currency string `json:"currency"`
	// TargetDevice 是目标设备响应值。
	TargetDevice TargetDeviceSelection `json:"target_device"`
	// ClickURLStatus 是点击地址测试状态。
	ClickURLStatus string `json:"click_url_status"`
	// TargetApps 是应用流量定向详情。
	TargetApps []OfferTargetApp `json:"target_app"`
	// MaintainBy 是设置维护方。
	MaintainBy string `json:"maintain_by"`
	// IncludeAudienceIDs 是包含的人群 ID。
	IncludeAudienceIDs []AudienceID `json:"include_ta_id"`
	// ExcludeAudienceIDs 是排除的人群 ID。
	ExcludeAudienceIDs []AudienceID `json:"exclude_ta_id"`
	// TargetMTGEvent 是标准优化事件。
	TargetMTGEvent []string `json:"target_mtg_event"`
	// TargetOriginalEvent 是原始优化事件。
	TargetOriginalEvent string `json:"target_original_event"`
	// TargetGoalWindow 是优化时间窗。
	TargetGoalWindow string `json:"target_goal_window"`
	// TargetGoal 是广告单元优化目标值。
	TargetGoal DecimalText `json:"target_goal"`
	// TargetGoalByGeo 是地区优化目标值。
	TargetGoalByGeo []GeoTargetGoal `json:"target_goal_by_geo"`
}

// UnmarshalJSON 兼容广告单元接口把 campaign_id 返回为 number 或十进制字符串。
func (value *Offer) UnmarshalJSON(data []byte) error {
	type offerAlias Offer
	var wire struct {
		*offerAlias
		CampaignID json.RawMessage `json:"campaign_id"`
	}
	wire.offerAlias = (*offerAlias)(value)
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if len(wire.CampaignID) == 0 {
		return nil
	}
	raw := string(wire.CampaignID)
	if len(raw) >= 2 && raw[0] == '"' {
		var text string
		if err := json.Unmarshal(wire.CampaignID, &text); err != nil {
			return err
		}
		raw = text
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: offer campaign_id is invalid", ErrUnexpectedResponse)
	}
	value.CampaignID = CampaignID(id)
	return nil
}

// LocationBid 是国家或地区维度的出价。
type LocationBid struct {
	// CountryCode 是出价地区。
	CountryCode CountryCode `json:"country_code"`
	// BidRate 是地区出价。
	BidRate DecimalText `json:"bid_rate"`
}

// MTGIDBid 是发布商维度的出价。
type MTGIDBid struct {
	// CountryCode 是出价地区。
	CountryCode CountryCode `json:"country_code"`
	// MTGID 是发布商标识。
	MTGID string `json:"mtgid"`
	// BidRate 是发布商出价。
	BidRate DecimalText `json:"bid_rate"`
}

// OfferBudget 是一个国家或地区范围的预算设置。
type OfferBudget struct {
	// CountryCode 是预算地区或 ALL。
	CountryCode CountryCode `json:"country_code"`
	// DailyCapType 是每日限额类型。
	DailyCapType DailyCapType `json:"daily_cap_type"`
	// DailyCap 是每日限额或 OPEN。
	DailyCap BudgetAmount `json:"daily_cap"`
	// TotalBudget 是总预算或 OPEN。
	TotalBudget BudgetAmount `json:"total_budget"`
}

// GeoTargetGoal 是国家或地区维度的优化目标值。
type GeoTargetGoal struct {
	// Geo 是目标地区。
	Geo CountryCode `json:"geo"`
	// TargetGoal 是地区优化目标值。
	TargetGoal DecimalText `json:"target_goal"`
}

// OfferTargetApp 是应用流量定向详情。
type OfferTargetApp struct {
	// Switch 是应用定向模式。
	Switch int `json:"switch"`
	// MTGIDs 是发布商标识。
	MTGIDs []string `json:"mtg_id"`
	// Packages 是应用包名。
	Packages []string `json:"package"`
}

// OfferCreative 是旧版广告单元素材设置。
type OfferCreative struct {
	// CreativeMD5 是素材内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
	// CreativeName 是素材名称。
	CreativeName string `json:"creative_name"`
	// ApplyInArea 是素材投放地区。
	ApplyInArea string `json:"apply_in_area"`
	// CreativeSetName 是可选创意组名称。
	CreativeSetName string `json:"creative_set_name,omitempty"`
	// Option 是素材启停操作。
	Option OfferTargetOption `json:"option"`
}

// OfferCreativeSet 是创建广告单元时使用的创意组。
type OfferCreativeSet struct {
	// CreativeSetName 是创意组名称。
	CreativeSetName string `json:"creative_set_name"`
	// Geos 是创意组投放地区。
	Geos []CountryCode `json:"geos"`
	// AdOutputs 是广告输出格式标识。
	AdOutputs []AdOutput `json:"ad_outputs"`
	// Creatives 是创意组素材。
	Creatives []OfferCreative `json:"creatives"`
}

// BudgetAmount 是预算金额，可为精确十进制数或 OPEN。
type BudgetAmount string

// OpenBudget 表示不限制预算。
const OpenBudget BudgetAmount = "OPEN"

// DecimalBudget 创建精确保留文本的预算金额。
func DecimalBudget(value DecimalText) BudgetAmount { return BudgetAmount(value) }

// MarshalJSON 将 OPEN 编码为字符串，将金额编码为 JSON number。
func (value BudgetAmount) MarshalJSON() ([]byte, error) {
	if value == OpenBudget {
		return []byte(`"OPEN"`), nil
	}
	return DecimalText(value).MarshalJSON()
}

// UnmarshalJSON 接受 OPEN、JSON number 或十进制字符串。
func (value *BudgetAmount) UnmarshalJSON(data []byte) error {
	if string(data) == `"OPEN"` {
		*value = OpenBudget
		return nil
	}
	var decimal DecimalText
	if err := decimal.UnmarshalJSON(data); err != nil {
		return err
	}
	*value = BudgetAmount(decimal)
	return nil
}

// AdSchedule 是一周七天的小时投放计划，值为逗号分隔的 0 至 23 时。
type AdSchedule struct {
	// Day1 是星期一投放小时。
	Day1 string `json:"1,omitempty"`
	// Day2 是星期二投放小时。
	Day2 string `json:"2,omitempty"`
	// Day3 是星期三投放小时。
	Day3 string `json:"3,omitempty"`
	// Day4 是星期四投放小时。
	Day4 string `json:"4,omitempty"`
	// Day5 是星期五投放小时。
	Day5 string `json:"5,omitempty"`
	// Day6 是星期六投放小时。
	Day6 string `json:"6,omitempty"`
	// Day7 是星期日投放小时。
	Day7 string `json:"7,omitempty"`
}
