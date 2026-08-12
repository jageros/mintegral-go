package mintegral

import (
	"encoding/json"
	"time"
)

// AudienceDeviceType 是人群包文件内设备标识的类型。
type AudienceDeviceType int

const (
	// AudienceDeviceIMEI 表示原始 IMEI。
	AudienceDeviceIMEI AudienceDeviceType = 1
	// AudienceDeviceIDFA 表示原始 IDFA。
	AudienceDeviceIDFA AudienceDeviceType = 2
	// AudienceDeviceGAID 表示原始 GAID。
	AudienceDeviceGAID AudienceDeviceType = 3
	// AudienceDeviceOAID 表示原始 OAID。
	AudienceDeviceOAID AudienceDeviceType = 4
	// AudienceDeviceIMEIMD5 表示 MD5 后的 IMEI。
	AudienceDeviceIMEIMD5 AudienceDeviceType = 6
	// AudienceDeviceIDFAMD5 表示 MD5 后的 IDFA。
	AudienceDeviceIDFAMD5 AudienceDeviceType = 7
	// AudienceDeviceGAIDMD5 表示 MD5 后的 GAID。
	AudienceDeviceGAIDMD5 AudienceDeviceType = 8
	// AudienceDeviceOAIDMD5 表示 MD5 后的 OAID。
	AudienceDeviceOAIDMD5 AudienceDeviceType = 9
	// AudienceDeviceIDFV 表示原始 IDFV。
	AudienceDeviceIDFV AudienceDeviceType = 12
	// AudienceDeviceIDFVMD5 表示 MD5 后的 IDFV。
	AudienceDeviceIDFVMD5 AudienceDeviceType = 13
)

// AudienceListRequest 筛选并分页查询人群包；Limit 和 Page 为零时分别使用 10 和 1。
type AudienceListRequest struct {
	TAIDs    []AudienceID // TAIDs 按人群包标识筛选。
	TAName   string       // TAName 按人群包名称模糊筛选。
	Platform int          // Platform 按投放平台筛选：1 Android、2 iOS、3 混合。
	Limit    int          // Limit 是每页数量，零值表示 10，最大为 500。
	Page     int          // Page 是页码，零值表示 1。
}

// AudienceList 是人群包分页结果。
type AudienceList struct {
	List         []Audience `json:"list"`          // List 是当前页的人群包。
	Page         int        `json:"page"`          // Page 是当前页码。
	Limit        int        `json:"limit"`         // Limit 是当前页容量。
	CurrentTotal int        `json:"current_total"` // CurrentTotal 是当前页实际数量。
	Total        int        `json:"total"`         // Total 是符合条件的总数量。
}

// Audience 是 Mintegral 返回的人群包信息。
type Audience struct {
	TAName     string      `json:"ta_name"`     // TAName 是人群包名称。
	DeviceType string      `json:"device_type"` // DeviceType 是服务端返回的逗号分隔设备类型。
	TAID       AudienceID  `json:"ta_id"`       // TAID 是人群包标识。
	Platform   int         `json:"platform"`    // Platform 是投放平台。
	TAType     int         `json:"ta_type"`     // TAType 是人群包类型。
	AreaType   int         `json:"area_type"`   // AreaType 是数据集群。
	CreatedAt  UnixSeconds `json:"ctime"`       // CreatedAt 是创建时间。
	UpdatedAt  UnixSeconds `json:"utime"`       // UpdatedAt 是更新时间。
}

// AudienceDataPath 关联一种设备标识类型与已上传文件路径。
type AudienceDataPath struct {
	DeviceType AudienceDeviceType `json:"device_type"` // DeviceType 是文件内设备标识的类型。
	DataPath   string             `json:"data_path"`   // DataPath 是预签名上传返回的完整文件路径。
}

// CreateAudienceRequest 创建上传型人群包。
type CreateAudienceRequest struct {
	TAName   string             `json:"ta_name"`   // TAName 是新建人群包名称。
	AreaType int                `json:"area_type"` // AreaType 是数据集群。
	Platform int                `json:"platform"`  // Platform 是投放平台。
	DataPath []AudienceDataPath `json:"data_path"` // DataPath 是已上传的人群文件列表。
}

// UpdateAudienceRequest 替换已有上传型人群包的数据文件。
type UpdateAudienceRequest struct {
	TAID     AudienceID         `json:"ta_id"`     // TAID 是要更新的人群包标识。
	DataPath []AudienceDataPath `json:"data_path"` // DataPath 是替换后的人群文件列表。
}

// DeleteAudienceRequest 批量删除人群包。
type DeleteAudienceRequest struct {
	TAIDs []AudienceID `json:"ta_id"` // TAIDs 是要删除的人群包标识。
}

// AudienceMutationResult 是创建或更新成功后返回的人群包标识。
type AudienceMutationResult struct {
	TAID AudienceID `json:"ta_id"` // TAID 是创建或更新后的人群包标识。
}

// AudiencePresignRequest 请求一个与文件元数据绑定的上传计划。
type AudiencePresignRequest struct {
	FileName string     // FileName 是不含目录的上传文件名。
	FileMD5  ContentMD5 // FileMD5 是文件内容摘要。
	FileSize int64      // FileSize 是文件字节数，最大为 5 GiB。
	AreaType int        // AreaType 是数据集群：1 非中国大陆、2 中国大陆。
}

// AudienceUploadPlan 是只能用于同一文件的限时上传计划。
type AudienceUploadPlan struct {
	S3        *AudienceS3Upload  // S3 是非中国大陆集群的上传参数。
	OSS       *AudienceOSSUpload // OSS 是中国大陆集群的上传参数。
	ExpiresAt time.Time          // ExpiresAt 是计划失效时刻。
	FileName  string             // FileName 是计划绑定的文件名。
	FileMD5   ContentMD5         // FileMD5 是计划绑定的内容摘要。
	FileSize  int64              // FileSize 是计划绑定的字节数。
	AreaType  int                // AreaType 是计划所属的数据集群。
}

// AudienceS3Upload 描述 S3 预签名原始 PUT。
type AudienceS3Upload struct {
	Method   string `json:"method"`    // Method 是上传方法，当前为 PUT。
	URL      string `json:"url"`       // URL 是带签名的上传地址，禁止记录或输出。
	DataPath string `json:"data_path"` // DataPath 是上传后用于人群包请求的路径。
}

// AudienceOSSUpload 描述 OSS 预签名 multipart/form-data POST。
type AudienceOSSUpload struct {
	Method    string `json:"method"`    // Method 是上传方法，当前为 POST。
	AccessID  string `json:"accessid"`  // AccessID 是 OSS 临时访问标识，禁止记录或输出。
	Host      string `json:"host"`      // Host 是 OSS 上传地址。
	Expire    string `json:"expire"`    // Expire 是 OSS 签名的 Unix 秒级失效时间。
	Signature string `json:"signature"` // Signature 是 OSS 临时签名，禁止记录或输出。
	Policy    string `json:"policy"`    // Policy 是 OSS 临时策略，禁止记录或输出。
	Directory string `json:"dir"`       // Directory 是 OSS 对象键前缀。
	DataPath  string `json:"data_path"` // DataPath 是上传后用于人群包请求的路径。
}

// AudienceUploadResult 是上传后供创建或更新请求使用的文件路径。
type AudienceUploadResult struct {
	DataPath string // DataPath 是上传后用于创建或更新人群包的路径。
}

// String 返回不包含预签名地址或 OSS 凭据的摘要。
func (AudienceUploadPlan) String() string { return "mintegral.AudienceUploadPlan(<redacted>)" }

// GoString 返回不包含预签名地址或 OSS 凭据的摘要。
func (AudienceUploadPlan) GoString() string { return "mintegral.AudienceUploadPlan(<redacted>)" }

// MarshalJSON 将上传计划编码为固定脱敏文本。
func (AudienceUploadPlan) MarshalJSON() ([]byte, error) { return json.Marshal("<redacted>") }

// String 返回不包含预签名地址的摘要。
func (AudienceS3Upload) String() string { return "mintegral.AudienceS3Upload(<redacted>)" }

// GoString 返回不包含预签名地址的摘要。
func (AudienceS3Upload) GoString() string { return "mintegral.AudienceS3Upload(<redacted>)" }

// MarshalJSON 将 S3 上传参数编码为固定脱敏文本。
func (AudienceS3Upload) MarshalJSON() ([]byte, error) { return json.Marshal("<redacted>") }

// String 返回不包含 OSS 凭据的摘要。
func (AudienceOSSUpload) String() string { return "mintegral.AudienceOSSUpload(<redacted>)" }

// GoString 返回不包含 OSS 凭据的摘要。
func (AudienceOSSUpload) GoString() string { return "mintegral.AudienceOSSUpload(<redacted>)" }

// MarshalJSON 将 OSS 上传参数编码为固定脱敏文本。
func (AudienceOSSUpload) MarshalJSON() ([]byte, error) { return json.Marshal("<redacted>") }
