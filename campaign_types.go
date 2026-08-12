package mintegral

// CampaignListRequest 是广告活动列表筛选条件。
type CampaignListRequest struct {
	// CampaignIDs 是最多 50 个广告活动唯一标识。
	CampaignIDs []CampaignID
	// CampaignName 是广告活动名称的筛选文本。
	CampaignName string
	// PackageName 是应用包名的筛选文本。
	PackageName string
	// OfferID 是广告单元唯一标识。
	OfferID OfferID
	// OfferName 是广告单元名称的筛选文本。
	OfferName string
	// OfferUUID 是广告单元 UUID 的筛选文本。
	OfferUUID string
	// Page 是从 1 开始的页码；0 使用服务端默认值。
	Page int
	// Limit 是每页数量，范围为 1 至 50；0 使用服务端默认值。
	Limit int
}

// CampaignYesNo 是请求中的二值设置。
type CampaignYesNo string

const (
	// CampaignYes 表示启用或肯定设置。
	CampaignYes CampaignYesNo = "YES"
	// CampaignNo 表示禁用或否定设置。
	CampaignNo CampaignYesNo = "NO"
)

// Known 报告该二值设置是否为文档列出的已知值。
func (value CampaignYesNo) Known() bool { return value == CampaignYes || value == CampaignNo }

// CampaignPromotionType 是广告活动推广目标类型。
type CampaignPromotionType string

const (
	// CampaignPromotionApp 表示推广移动应用。
	CampaignPromotionApp CampaignPromotionType = "APP"
	// CampaignPromotionWebsite 表示推广网站。
	CampaignPromotionWebsite CampaignPromotionType = "WEBSITE"
)

// Known 报告该推广目标类型是否为文档列出的已知值。
func (value CampaignPromotionType) Known() bool {
	return value == CampaignPromotionApp || value == CampaignPromotionWebsite
}

// CampaignPlatform 是移动应用推广平台。
type CampaignPlatform string

const (
	// CampaignPlatformIOS 表示 Apple iOS 平台。
	CampaignPlatformIOS CampaignPlatform = "IOS"
	// CampaignPlatformAndroid 表示 Android 平台。
	CampaignPlatformAndroid CampaignPlatform = "ANDROID"
)

// Known 报告该推广平台是否为文档列出的已知值。
func (value CampaignPlatform) Known() bool {
	return value == CampaignPlatformIOS || value == CampaignPlatformAndroid
}

// CreateCampaignRequest 是创建广告活动的请求。
type CreateCampaignRequest struct {
	// CampaignName 是最长 100 个字符的广告活动名称。
	CampaignName string `json:"campaign_name,omitempty"`
	// IsCOPPA 是 COPPA 设置，例如 YES 或 NO。
	IsCOPPA CampaignYesNo `json:"is_coppa,omitempty"`
	// PromotionType 是推广目标类型，例如 APP 或 WEBSITE。
	PromotionType CampaignPromotionType `json:"promotion_type,omitempty"`
	// AliveInStore 表示应用是否已在应用商店上架，例如 YES 或 NO。
	AliveInStore CampaignYesNo `json:"alive_in_store,omitempty"`
	// PreviewURL 是推广对象的预览链接。
	PreviewURL string `json:"preview_url,omitempty"`
	// ProductName 是最长 100 个字符的应用或网站名称。
	ProductName string `json:"product_name,omitempty"`
	// PackageName 是 Android 包名或 iOS App Store ID。
	PackageName string `json:"package_name,omitempty"`
	// Description 是最长 4000 个字符的推广对象说明。
	Description string `json:"description,omitempty"`
	// Icon 是 512x512 图标文件的 MD5 文本。
	Icon string `json:"icon,omitempty"`
	// Platform 是推广平台，例如 IOS 或 ANDROID。
	Platform CampaignPlatform `json:"platform,omitempty"`
	// Category 是推广对象类别。
	Category string `json:"category,omitempty"`
	// AppSize 是应用包体大小，单位为 MB；网站推广时省略。
	AppSize *DecimalText `json:"app_size,omitempty"`
	// MinVersion 是应用最低系统版本。
	MinVersion string `json:"min_version,omitempty"`
}

// UpdateCampaignRequest 是广告活动的部分更新请求。
type UpdateCampaignRequest struct {
	// CampaignID 是更新目标的唯一标识。
	CampaignID CampaignID `json:"campaign_id"`
	// CampaignName 是最长 100 个字符的广告活动名称；nil 表示不更新。
	CampaignName *string `json:"campaign_name,omitempty"`
	// IsCOPPA 是 COPPA 设置；nil 表示不更新。
	IsCOPPA *CampaignYesNo `json:"is_coppa,omitempty"`
	// PromotionType 是推广目标类型；nil 表示不更新。
	PromotionType *CampaignPromotionType `json:"promotion_type,omitempty"`
	// ProductName 是最长 100 个字符的应用或网站名称；nil 表示不更新。
	ProductName *string `json:"product_name,omitempty"`
	// PackageName 是 Android 包名或 iOS App Store ID；nil 表示不更新。
	PackageName *string `json:"package_name,omitempty"`
	// Description 是最长 4000 个字符的推广对象说明；nil 表示不更新。
	Description *string `json:"description,omitempty"`
	// Icon 是 512x512 图标文件的 MD5 文本；nil 表示不更新。
	Icon *string `json:"icon,omitempty"`
	// Category 是推广对象类别；nil 表示不更新。
	Category *string `json:"category,omitempty"`
	// AppSize 是应用包体大小，单位为 MB；nil 表示不更新。
	AppSize *DecimalText `json:"app_size,omitempty"`
	// MinVersion 是应用最低系统版本；nil 表示不更新。
	MinVersion *string `json:"min_version,omitempty"`
}

// CampaignPage 是广告活动列表的分页结果。
type CampaignPage struct {
	// Page 是当前页码。
	Page int `json:"page"`
	// Limit 是每页数量。
	Limit int `json:"limit"`
	// Total 是符合筛选条件的总数。
	Total int `json:"total"`
	// List 是当前页广告活动。
	List []Campaign `json:"list"`
}

// CampaignStatus 是广告活动状态；未知文本值会原样保留。
type CampaignStatus string

const (
	// CampaignStatusRunning 表示广告活动正在投放。
	CampaignStatusRunning CampaignStatus = "RUNNING"
	// CampaignStatusStopped 表示广告活动已停止。
	CampaignStatusStopped CampaignStatus = "STOPPED"
	// CampaignStatusOverDailyCap 表示广告活动已用尽日预算。
	CampaignStatusOverDailyCap CampaignStatus = "OVER_DAILY_CAP"
	// CampaignStatusInsufficientBalance 表示账户余额不足。
	CampaignStatusInsufficientBalance CampaignStatus = "INSUFFICIENT_ACCOUNT_BALANCE"
)

// Known 报告该广告活动状态是否为文档列出的已知值。
func (value CampaignStatus) Known() bool {
	return value == CampaignStatusRunning || value == CampaignStatusStopped || value == CampaignStatusOverDailyCap || value == CampaignStatusInsufficientBalance
}

// CampaignMaintainer 是广告活动维护权限主体；未知文本值会原样保留。
type CampaignMaintainer string

const (
	// CampaignMaintainerAM 表示仅可查看广告活动。
	CampaignMaintainerAM CampaignMaintainer = "AM"
	// CampaignMaintainerADV 表示可自由维护广告活动。
	CampaignMaintainerADV CampaignMaintainer = "ADV"
)

// Known 报告该维护权限主体是否为文档列出的已知值。
func (value CampaignMaintainer) Known() bool {
	return value == CampaignMaintainerAM || value == CampaignMaintainerADV
}

// Campaign 是 Mintegral 广告活动。
type Campaign struct {
	// CampaignID 是广告活动唯一标识。
	CampaignID CampaignID `json:"campaign_id"`
	// CampaignName 是广告活动名称。
	CampaignName string `json:"campaign_name"`
	// IsCOPPA 是 COPPA 设置。
	IsCOPPA CampaignYesNo `json:"is_coppa"`
	// PromotionType 是推广目标类型。
	PromotionType CampaignPromotionType `json:"promotion_type"`
	// AliveInStore 表示应用是否已上架。
	AliveInStore CampaignYesNo `json:"alive_in_store"`
	// PreviewURL 是推广对象的预览链接。
	PreviewURL string `json:"preview_url"`
	// ProductName 是推广对象名称。
	ProductName string `json:"product_name"`
	// PackageName 是应用包名或 App Store ID。
	PackageName string `json:"package_name"`
	// Description 是推广对象说明。
	Description string `json:"description"`
	// Icon 是图标 MD5 文本。
	Icon string `json:"icon"`
	// Platform 是推广平台。
	Platform CampaignPlatform `json:"platform"`
	// Category 是推广对象类别。
	Category string `json:"category"`
	// AppSize 是服务端返回的应用包体大小十进制文本。
	AppSize DecimalText `json:"app_size"`
	// MinVersion 是最低系统版本。
	MinVersion string `json:"min_version"`
	// MaintainBy 是允许维护该活动的主体。
	MaintainBy CampaignMaintainer `json:"maintain_by"`
	// Status 是广告活动状态。
	Status CampaignStatus `json:"status"`
}
