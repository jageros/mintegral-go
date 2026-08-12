package mintegral

import (
	"context"
	"slices"
	"time"
)

// Dimension 表示高级报表的聚合维度。
type Dimension string

const (
	// DimensionOffer 按 Offer 聚合。
	DimensionOffer Dimension = "Offer"
	// DimensionCampaign 按 Campaign 聚合。
	DimensionCampaign Dimension = "Campaign"
	// DimensionCampaignPackage 按 Campaign Package 聚合。
	DimensionCampaignPackage Dimension = "CampaignPackage"
	// DimensionCreative 按 Creative 聚合。
	DimensionCreative Dimension = "Creative"
	// DimensionAdType 按 Ad Type 聚合。
	DimensionAdType Dimension = "AdType"
	// DimensionSub 按 Sub 聚合。
	DimensionSub Dimension = "Sub"
	// DimensionPackage 按 Package 聚合。
	DimensionPackage Dimension = "Package"
	// DimensionLocation 按 Location 聚合。
	DimensionLocation Dimension = "Location"
	// DimensionEndcard 按 Endcard 聚合。
	DimensionEndcard Dimension = "Endcard"
	// DimensionAdOutputType 按 Ad Output Type 聚合。
	DimensionAdOutputType Dimension = "AdOutputType"
	// DimensionDma 按 DMA 聚合。
	DimensionDma Dimension = "Dma"
	// DimensionState 按 State 聚合。
	DimensionState Dimension = "State"
)

// Granularity 表示高级报表的时间粒度。
type Granularity string

const (
	// GranularityDaily 按日聚合。
	GranularityDaily Granularity = "daily"
	// GranularityHourly 按小时聚合。
	GranularityHourly Granularity = "hourly"
)

// ReportQuery 描述高级报表查询条件。
type ReportQuery struct {
	// Timezone 是 -11 到 +11 的整点 UTC 偏移，默认 +8。
	Timezone string
	// StartDate 是闭区间开始日期。
	StartDate Date
	// EndDate 是闭区间结束日期。
	EndDate Date
	// Dimensions 是至少一个且不重复的聚合维度。
	Dimensions []Dimension
	// Granularity 是 daily 或 hourly，默认 daily。
	Granularity Granularity
}

// ReportOpenRequest 配置报表轮询和下载。
type ReportOpenRequest struct {
	// Query 是报表查询条件。
	Query ReportQuery
	// AllowIncomplete 允许下载 is_complete=false 的可用报表。
	AllowIncomplete bool
	// PollInterval 是状态轮询间隔，默认 5 秒。
	PollInterval time.Duration
	// MaxWait 是状态轮询的最长等待，默认 30 分钟。
	MaxWait time.Duration
}

// ReportConsumeRequest 配置同步批量消费。
type ReportConsumeRequest struct {
	// Open 是报表打开参数。
	Open ReportOpenRequest
	// BatchSize 是每批最大行数，默认 500。
	BatchSize int
}

// ReportStatus 是 type=1 返回的生成状态。
type ReportStatus struct {
	// Code 是 Mintegral 业务状态码。
	Code int
	// Hours 是服务端报告的已生成小时信息。
	Hours []int
	// IsComplete 表示报表是否完整生成。
	IsComplete bool
}

// ReportExtras 保存服务端新增的未知列，且不暴露可变映射。
type ReportExtras struct{ values map[string]string }

// Get 返回未知列值。
func (e ReportExtras) Get(column string) (string, bool) {
	value, ok := e.values[column]
	return value, ok
}

// Columns 返回未知列名的副本。
func (e ReportExtras) Columns() []string {
	columns := make([]string, 0, len(e.values))
	for column := range e.values {
		columns = append(columns, column)
	}
	slices.Sort(columns)
	return columns
}

// ReportRow 表示高级报表中的一行。
type ReportRow struct {
	// Date 是 YYYY-MM-DD 报表日期。
	Date Date
	// Timestamp 是小时粒度的 Unix 秒。
	Timestamp UnixSeconds
	// OfferID 是 Offer 标识。
	OfferID OfferID
	// OfferUUID 是 Offer UUID。
	OfferUUID string
	// OfferName 是 Offer 名称。
	OfferName string
	// CampaignID 是 Campaign 标识。
	CampaignID CampaignID
	// CampaignPackage 是 Campaign 包名。
	CampaignPackage string
	// CreativeID 是 Creative 标识。
	CreativeID CreativeID
	// CreativeName 是 Creative 名称。
	CreativeName string
	// AdType 是广告类型。
	AdType string
	// SubID 是子渠道标识。
	SubID string
	// PackageName 是推广包名。
	PackageName string
	// Location 是国家或地区。
	Location string
	// EndcardID 是 Endcard 标识。
	EndcardID int64
	// EndcardName 是 Endcard 名称。
	EndcardName string
	// AdOutputType 是广告输出类型。
	AdOutputType string
	// DmaCode 是美国 DMA 代码。
	DmaCode int64
	// StateCode 是地区代码。
	StateCode string
	// Currency 是金额币种。
	Currency string
	// Impressions 是展示次数。
	Impressions int64
	// Clicks 是点击次数。
	Clicks int64
	// Conversions 是转化次数。
	Conversions int64
	// ECPM 是每千次展示成本。
	ECPM DecimalText
	// CPC 是单次点击成本。
	CPC DecimalText
	// CTR 是点击率。
	CTR DecimalText
	// CVR 是点击转化率。
	CVR DecimalText
	// IVR 是展示转化率。
	IVR DecimalText
	// Spend 是消耗金额。
	Spend DecimalText
	// Extras 保存未知列。
	Extras ReportExtras
}

// ReportDelivery 描述已经得到处理器确认的交付量。
type ReportDelivery struct {
	// ParsedRows 是已经从 TSV 成功解析的总行数，包括处理器未确认的行。
	ParsedRows int64
	// AcknowledgedRows 是处理器成功返回后确认的行数。
	AcknowledgedRows int64
	// Status 是打开下载流前最后一次成功获得的报表状态。
	Status ReportStatus
}

// ReportHandler 同步处理一批报表行。
type ReportHandler func(context.Context, []ReportRow) error
