# mintegral-go

`mintegral-go` 是 Mintegral Open API 的强类型 Go SDK。它负责鉴权、请求/响应契约、分页、报表流和素材/人群包上传；不负责数据库、任务调度、断点或业务日志。

> Go 版本：`1.26.0`（见 `go.mod`）。所有网络方法都接受 `context.Context`，请始终把调用方的超时和取消信号传入。

## 安装

```bash
go get github.com/jageros/mintegral-go
```

## 初始化与凭据

凭据只在内存中保存，`Credentials` 的格式化输出会脱敏。不要把 Access Key/API Key 写入源码、测试夹具或日志。

默认凭据适合单租户客户端：

```go
credentials, err := mintegral.NewCredentials(os.Getenv("MINTEGRAL_ACCESS_KEY"), os.Getenv("MINTEGRAL_API_KEY"))
if err != nil {
	return err
}
client, err := mintegral.NewClient(mintegral.WithDefaultCredentials(credentials))
if err != nil {
	return err
}
```

多租户场景可创建不带默认凭据的客户端，并只在当前调用覆盖凭据；该选项不会修改 `Client`，可安全并发使用：

```go
client, err := mintegral.NewClient()
if err != nil {
	return err
}
credentials, err := mintegral.NewCredentials(accessKey, apiKey)
if err != nil {
	return err
}

// 例：所有鉴权方法的最后一个可选参数都可传入。
balance, err := client.Accounts().Balance(ctx, mintegral.WithRequestCredentials(credentials))
```

显式提供无效凭据会返回 `ErrInvalidCredentials`，不会回退到默认凭据；未提供凭据的鉴权调用在发送 HTTP 请求前返回 `ErrCredentialsRequired`。

## 服务总览

所有服务均由同一个可并发复用的 `Client` 创建，并接受 `context.Context` 与可选的 `RequestOption`。写操作出现传输中断时必须按 `ErrOutcomeUnknown` 处理，不会自动重发。

| 服务 | 公开方法 |
| --- | --- |
| `Accounts()` | `Balance` |
| `Campaigns()` | `List`、`Create`、`Update` |
| `Apps()` | `Names` |
| `Offers()` | `List`、`Create`、`Update`、`UpdateBids`、`UpdateBudget`、`SetStatus`、`UpdateTrafficDelivery`、`UpdateTracking`、`SetAudiences`、`UpdateTargetGoal`、`ApplyCreatives`（已弃用） |
| `Events()` | `BidGoalSupports` |
| `CreativeSets()` | `List`、`Create`、`Update`、`Delete` |
| `CreativeAds()` | `List`（`ad_ids` 必填） |
| `Assets()` | `List`、`UploadMedia`、`UploadPlayable` |
| `Reports()` | `Status`、`Open`、`Consume` |
| `Audiences()` | `List`、`PresignUpload`、`Upload`、`UploadFile`、`Create`、`Update`、`Delete` |

## Context、错误与重试

为每次业务操作设置 deadline；不要用 `context.Background()` 替代调用链的 context。

```go
ctx, cancel := context.WithTimeout(parentCtx, 90*time.Second)
defer cancel()

result, err := client.Campaigns().List(ctx, request)
if err != nil {
	var apiErr *mintegral.APIError
	switch {
	case errors.Is(err, mintegral.ErrRateLimited):
		// 可结合 apiErr.RetryAfter 安排下一次工作。
	case errors.As(err, &apiErr):
		// 读取 Operation、HTTPStatus、Code；不要按 Error() 文本分支。
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// 调用方已取消或超时。
	}
	return err
}
```

SDK 仅对已标为可重放的只读请求做有限重试（默认最多 3 次，包含首次）。创建、更新、删除和素材上传不会因传输中断而盲目重放；Audience 预签名上传只执行下文定义的一次精确重放。其他写请求结果未知时，使用 `errors.Is(err, mintegral.ErrOutcomeUnknown)` 处理。错误不会携带 URL、签名、token、凭据或文件内容。

## 分页

列表请求使用 `PageRequest{Number, Limit}`，响应返回 `PageInfo{Number, Limit, Total, Returned}`。使用 `Next()` 仅在元数据表明存在非空下一页时继续，避免根据 `len(list) == limit` 猜测。

```go
page := mintegral.PageRequest{Number: 1, Limit: 50}
for {
	result, err := client.CreativeAds().List(ctx, mintegral.CreativeAdListRequest{
		AdIDs: []mintegral.AdID{123456789},
		Page: page.Number, Limit: page.Limit,
	})
	if err != nil {
		return err
	}
	consume(result.List)
	next, ok := mintegral.PageInfo{
		Number: result.Page, Limit: result.Limit,
		Total: result.Total, Returned: len(result.List),
	}.Next()
	if !ok {
		break
	}
	page = next
}
```

调用方应始终依据 `PageInfo.Next()` 显式分页，并为总页数或总记录数设置自己的业务预算；不要根据 `len(list) == limit` 猜测是否还有下一页。

## Advanced Report v2：流式消费

报表只使用 Advanced Performance Report **v2**：`GET /api/v2/reports/data`。`type=1` 请求/轮询报表，`type=2` 下载 TSV；SDK 将其封装为一次流式读取，不会回退到 v1。

```go
delivery, err := client.Reports().Consume(ctx, mintegral.ReportConsumeRequest{
	Open: mintegral.ReportOpenRequest{
		Query: mintegral.ReportQuery{
			Timezone: "+8", StartDate: startDate, EndDate: endDate,
			Dimensions: []mintegral.Dimension{mintegral.DimensionCampaign, mintegral.DimensionCreative},
		},
	},
	BatchSize: 500,
}, func(ctx context.Context, rows []mintegral.ReportRow) error {
	for _, row := range rows { consume(row) }
	return nil
})
if err != nil { return err }
fmt.Println(delivery.AcknowledgedRows)
```

回调在单次报表内同步、按行顺序调用。若异步保存 `rows`，先复制；回调返回后 SDK 可以复用其内存。任一批已成功交付、后续读取失败时返回 `ErrPartialDelivery`，不能假设可从头透明重试。

## 素材与人群包

素材上传使用 storage API；人群包需先取得预签名计划，再按 `area_type` 上传，最后创建 audience：

1. `Audiences().PresignUpload` 调用 `GET /api/open/v1/audience/presigned-upload-data`，创建绑定 `file_name`、MD5、字节数和 `area_type` 的 `AudienceUploadPlan`。
2. `Audiences().Upload` 执行已取得的计划：`area_type=1` 以 **S3 PUT** 上传原始文件，绝不包装为 multipart；`area_type=2` 以 **OSS multipart/form-data POST** 上传，字段为 `key`、`OSSAccessKeyId`、`policy`、小写 `signature`、`success_action_status=200` 和 `file`。
3. `Audiences().UploadFile` 仅组合“预签名 + 上传”；它不会创建人群包。使用返回的 `DataPath` 调用 `Audiences().Create` 或 `Audiences().Update`。

预签名 URL、OSS policy 和 signature 都是 bearer secret：不能写日志、不能带 Mintegral 鉴权头、不能跨主机重定向。`Upload` 会先流式校验源内容并创建权限受限的本地临时快照，因此需预留约等于源文件大小的临时磁盘空间；网络发送不把文件完整读入内存。仅当计划仍有效时，SDK 才从这份已验证快照对可重试网络/HTTP 状态作**一次精确重放**（最多两次物理发送）。任何不匹配、过期或不确定的结果都不会改用另一存储方式、重新预签名或自动创建 audience。

## 离线验证

SDK 测试不需要真实凭据、数据库或访问 Mintegral。注入 `http.Client`/`http.RoundTripper` 和 `WithAPIBaseURL`、`WithStorageBaseURL` 进行契约测试。

```bash
task fmt-check
task test
task race
task examples
task vet
task lint
task nilaway
task build
```

CI 使用 Go 1.26。仅改文档或配置时，至少运行 `task fmt-check` 和相应文件的 diff/文本检查；不要以真实 Provider 调用作为测试替代品。

## 契约与兼容性

完整端点、编码、权限、限制、弃用项和文档矛盾见 [中文 API 契约](docs/api-contract.zh-CN.md)。Provider 文档会变化；升级 SDK 或新增端点前先核对该文档的检查日期与原始官方页面。
