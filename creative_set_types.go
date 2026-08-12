package mintegral

// CreativeAuditStatus 是素材审核状态；未知整数值会原样保留以兼容服务端扩展。
type CreativeAuditStatus int

const (
	// CreativeAuditInitialized 表示素材尚未进入审核。
	CreativeAuditInitialized CreativeAuditStatus = iota
	// CreativeAuditApproved 表示素材审核通过。
	CreativeAuditApproved
	// CreativeAuditRejected 表示素材审核拒绝。
	CreativeAuditRejected
	// CreativeAuditReviewing 表示素材正在审核。
	CreativeAuditReviewing
)

// Known 报告审核状态是否为当前 SDK 已知值。
func (status CreativeAuditStatus) Known() bool {
	return status >= CreativeAuditInitialized && status <= CreativeAuditReviewing
}

// CombinationMethod 是创意组组合方式；未知整数值会原样保留以兼容服务端扩展。
type CombinationMethod int

const (
	// CombinationProgrammatic 表示程序化组合。
	CombinationProgrammatic CombinationMethod = 1
	// CombinationCustomized 表示自定义组合。
	CombinationCustomized CombinationMethod = 2
)

// Known 报告组合方式是否为当前 SDK 已知值。
func (method CombinationMethod) Known() bool {
	return method == CombinationProgrammatic || method == CombinationCustomized
}

const (
	// AdOutputFullScreenImage 表示全屏图片组合。
	AdOutputFullScreenImage AdOutput = 111
	// AdOutputNativeImageBanner 表示原生图片横幅组合。
	AdOutputNativeImageBanner AdOutput = 121
	// AdOutputIconBanner 表示图标横幅组合。
	AdOutputIconBanner AdOutput = 122
	// AdOutputBasicBanner 表示基础横幅组合。
	AdOutputBasicBanner AdOutput = 131
	// AdOutputImageBanner 表示图片横幅组合。
	AdOutputImageBanner AdOutput = 132
	// AdOutputVideoEndcard 表示视频和图片尾卡组合。
	AdOutputVideoEndcard AdOutput = 211
	// AdOutputVideoPlayable 表示视频和试玩组合。
	AdOutputVideoPlayable AdOutput = 212
	// AdOutputFullScreenVideo 表示全屏视频组合。
	AdOutputFullScreenVideo AdOutput = 213
	// AdOutputNativeVideoBanner 表示原生视频横幅组合。
	AdOutputNativeVideoBanner AdOutput = 221
	// AdOutputVideoBanner 表示视频横幅组合。
	AdOutputVideoBanner AdOutput = 231
	// AdOutputPlayable 表示试玩组合。
	AdOutputPlayable AdOutput = 311
)

// Known 报告广告输出类型是否为当前 SDK 已知值。
func (output AdOutput) Known() bool {
	switch output {
	case AdOutputFullScreenImage, AdOutputNativeImageBanner, AdOutputIconBanner,
		AdOutputBasicBanner, AdOutputImageBanner, AdOutputVideoEndcard,
		AdOutputVideoPlayable, AdOutputFullScreenVideo, AdOutputNativeVideoBanner,
		AdOutputVideoBanner, AdOutputPlayable:
		return true
	default:
		return false
	}
}

// CreativeSetListRequest 是创意组列表筛选条件。
type CreativeSetListRequest struct {
	// CreativeSetName 是创意组名称筛选。
	CreativeSetName string
	// OfferID 是推广商品 ID 筛选。
	OfferID OfferID
	// CreativeSetID 是创意组 ID 筛选。
	CreativeSetID CreativeSetID
	// CombinationMethod 是创意组合方式筛选。
	CombinationMethod CombinationMethod
	// Page 是从 1 开始的页码。
	Page int
	// Limit 是每页数量，最大 50。
	Limit int
}

// CreativeSetCreative 描述创意组绑定的一个素材。
type CreativeSetCreative struct {
	// CreativeName 是素材名称。
	CreativeName string `json:"creative_name"`
	// CreativeMD5 是素材内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
	// CreativeType 是素材类型。
	CreativeType CreativeType `json:"creative_type"`
	// Dimension 是素材尺寸。
	Dimension string `json:"dimension"`
	// CreatedAt 是绑定素材的 Unix 秒时间戳。
	CreatedAt UnixSeconds `json:"created_at"`
	// AuditStatus 是素材审核状态。
	AuditStatus CreativeAuditStatus `json:"audit_status"`
	// AuditReason 是素材拒绝原因。
	AuditReason string `json:"audit_reason"`
}

// CreativeSet 描述一个创意组。
type CreativeSet struct {
	// CreativeSetName 是创意组名称。
	CreativeSetName string `json:"creative_set_name"`
	// OfferID 是绑定的推广商品 ID。
	OfferID OfferID `json:"offer_id"`
	// CreativeSetID 是创意组 ID。
	CreativeSetID CreativeSetID `json:"creative_set_id"`
	// CombinationMethod 是创意组合方式。
	CombinationMethod CombinationMethod `json:"combination_method"`
	// AdOutputs 是广告输出类型编号列表。
	AdOutputs []AdOutput `json:"ad_outputs"`
	// Geos 是适用国家或地区列表。
	Geos []string `json:"geos"`
	// Creatives 是已绑定素材列表。
	Creatives []CreativeSetCreative `json:"creatives"`
}

// CreativeSetList 是创意组分页结果。
type CreativeSetList struct {
	// List 是当前页创意组。
	List []CreativeSet `json:"list"`
	// Page 是当前页码。
	Page int `json:"page"`
	// Limit 是每页数量。
	Limit int `json:"limit"`
	// Total 是匹配创意组总数。
	Total int `json:"total"`
}

// CreativeSetInput 是创建创意组时绑定的素材。
type CreativeSetInput struct {
	// CreativeName 是素材名称。
	CreativeName string `json:"creative_name"`
	// CreativeMD5 是素材内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
}

// CreateCreativeSetRequest 是创建创意组请求。
type CreateCreativeSetRequest struct {
	// CreativeSetName 是新创意组名称。
	CreativeSetName string `json:"creative_set_name"`
	// CombinationMethod 是创意组合方式；零值使用服务端默认值。
	CombinationMethod CombinationMethod `json:"combination_method,omitempty"`
	// OfferID 是可选绑定推广商品 ID。
	OfferID OfferID `json:"offer_id,omitempty"`
	// Geos 是可选适用国家或地区列表。
	Geos []string `json:"geos,omitempty"`
	// AdOutputs 是必填广告输出类型编号列表。
	AdOutputs []AdOutput `json:"ad_outputs"`
	// Creatives 是必填绑定素材列表。
	Creatives []CreativeSetInput `json:"creatives"`
}

// CreativeSetMutationResult 是创建创意组结果。
type CreativeSetMutationResult struct {
	// CreativeSetName 是已创建创意组名称。
	CreativeSetName string `json:"creative_set_name"`
	// CreativeSetID 是已创建创意组 ID。
	CreativeSetID CreativeSetID `json:"creative_set_id"`
	// OfferID 是已绑定推广商品 ID。
	OfferID OfferID `json:"offer_id"`
}

// CreativeSetEditOption 是创意组素材的添加或删除方式。
type CreativeSetEditOption string

const (
	// CreativeSetEnable 表示向创意组添加素材。
	CreativeSetEnable CreativeSetEditOption = "ENABLE"
	// CreativeSetDisable 表示从创意组删除素材。
	CreativeSetDisable CreativeSetEditOption = "DISABLE"
)

func (option CreativeSetEditOption) valid() bool {
	return option == CreativeSetEnable || option == CreativeSetDisable
}

// CreativeSetEdit 描述更新创意组时的一项素材变更。
type CreativeSetEdit struct {
	// CreativeName 是素材名称。
	CreativeName string `json:"creative_name"`
	// CreativeMD5 是素材内容摘要。
	CreativeMD5 ContentMD5 `json:"creative_md5"`
	// Option 是添加或删除操作。
	Option CreativeSetEditOption `json:"option"`
}

// UpdateCreativeSetRequest 是更新创意组请求。
type UpdateCreativeSetRequest struct {
	// OfferID 是必填推广商品 ID。
	OfferID OfferID `json:"offer_id"`
	// CreativeSetName 是必填创意组名称。
	CreativeSetName string `json:"creative_set_name"`
	// Geos 为 nil 时不更新，非 nil 空切片清空国家或地区。
	Geos *[]string `json:"geos,omitempty"`
	// AdOutputs 为 nil 时不更新，非 nil 空切片清空输出类型。
	AdOutputs *[]AdOutput `json:"ad_outputs,omitempty"`
	// Creatives 为 nil 时不更新，非 nil 空切片发送空变更列表。
	Creatives *[]CreativeSetEdit `json:"creatives,omitempty"`
}

// DeleteCreativeSetRequest 是删除创意组请求。
type DeleteCreativeSetRequest struct {
	// OfferID 是必填推广商品 ID。
	OfferID OfferID `json:"offer_id"`
	// CreativeSetName 是必填创意组名称。
	CreativeSetName string `json:"creative_set_name"`
}
