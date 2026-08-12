# Mintegral Open API 契约

检查日期：**2026-08-12**。本文是 SDK 的线路契约清单，不替代 Mintegral 官方文档。所有 `/api/open/*` 管理接口使用 Mintegral token 鉴权；预签名 S3/OSS 上传只使用计划给出的凭据/表单字段，绝不附带管理接口 token。

## 统计口径

下表记录 **31 个 Mintegral Open API 契约**：同一路径的不同 HTTP 方法分别计数；`/api/v2/reports/data` 是一个端点，`type=1/2` 是其固定状态机模式。S3 与 OSS 是该 Open API 返回的外部预签名上传分支，随后单列说明，不混入这 31 个管理 API。

所有 JSON 管理接口的成功业务码为 `code=200`；HTTP 200 但其他业务码仍是失败。响应可能使用 `msg` 或 `message`：只有一个非空时采用；二者都非空且不相同则视为契约错误。

## JSON 响应与 `null` 语义

- 对自定义解码器值，Provider 响应中的已知字段仍按严格类型解析；唯一例外是字段显式为 JSON 字面量 `null`：此时解码为对应 Go 零值，并清空复用中的目标。
- 成功业务包（`code=200`）若显式返回 `data: null`，返回零值结果 `T`。`data` 字段缺失仍是契约错误，唯一例外是这六个已记录的 mutation：`UpdateOfferBudget`、`UpdateOfferStatus`、`UpdateOfferTarget`（traffic）、`UpdateOfferAudience`、`UpdateOfferTargetGoal`、`DeleteAudience`。
- HTTP 错误或业务错误即使带有 `data: null`，仍返回对应的 typed error；不会按成功零值处理。
- 字符串 `"null"`、字段缺失以及非 `null` 的无效类型都不属于 JSON `null`，仍按原有严格规则处理。
- 请求侧必填校验和零值 Marshal 行为保持严格；报表 TSV 契约不变。

| # | SDK 操作 | 方法 / 路径 | 编码 | 限制、权限或生命周期 | 弃用 | 官方文档 |
| --- | --- | --- | --- | --- | --- |
| 1 | AccountBalance | `GET /api/open/v1/account/balance` | 无 body | token；读取账户结算余额 | 否 | [账户余额](https://adv.mintegral.com/doc/en/guide/account/getAccountBalance.html) |
| 2 | ListCampaigns | `GET /api/open/v1/campaign` | JSON body（Provider 示例） | token；`campaign_id` 一次最多 50 个 | 否 | [Campaign List](https://adv.mintegral.com/doc/en/guide/campaign/getCampaign.html) |
| 3 | CreateCampaign | `POST /api/open/v1/campaign` | JSON | token；写操作不可盲重试 | 否 | [Create Campaign](https://adv.mintegral.com/doc/en/guide/campaign/createCampaign.html) |
| 4 | UpdateCampaign | `PUT /api/open/v1/campaign` | JSON | token；写操作不可盲重试 | 否 | [Update Campaign](https://adv.mintegral.com/doc/en/guide/campaign/updateCampaign.html) |
| 5 | ListOffers | `GET /api/open/v1/offers` | query、无 body | token；`limit≤50` | 否 | [Offer List](https://helpcenter.mintegral.com/en/docs/retreive-ad-offer-list-api) |
| 6 | CreateOffer | `POST /api/open/v1/offer` | JSON | token；写操作不可盲重试 | 否 | [Create Offer](https://helpcenter.mintegral.com/en/docs/create-ad-unit-api) |
| 7 | UpdateOffer | `PUT /api/open/v1/offer` | JSON | token；写操作不可盲重试 | 否 | [Update Offer](https://helpcenter.mintegral.com/en/docs/manage-offer-api) |
| 8 | UpdateOfferBidRate | `PUT /api/open/v1/offer/bid_rate` | JSON | token；完整替换；空地区/MTGID数组清除对应出价；高级 MTGID 权限 | 否 | [Update Bids](https://helpcenter.mintegral.com/en/docs/update-bid-api) |
| 9 | UpdateOfferBudget | `PUT /api/open/v1/offer/budget` | JSON | token；完整替换预算；写操作不可盲重试 | 否 | [Update Budget](https://helpcenter.mintegral.com/cn/docs/update-budget-api) |
| 10 | UpdateOfferStatus | `PUT /api/open/v1/offer/status` | JSON | token；状态仅 `RUNNING`/`STOPPED` | 否 | [Update Offer Status](https://helpcenter.mintegral.com/cn/docs/update-ad-unit-status-api) |
| 11 | UpdateOfferTarget | `PUT /api/open/v1/offer/target` | JSON | token；高级流量投放权限；写操作不可盲重试 | 否 | [Update Offer Target](https://helpcenter.mintegral.com/cn/docs/update-traffic-delivery-status-api) |
| 12 | UpdateTracking | `PUT /api/open/v1/tracking` | JSON | token；写操作不可盲重试 | 否 | [Update Tracking](https://helpcenter.mintegral.com/cn/docs/update-tracking-url) |
| 13 | ApplyOfferCreative | `PUT /api/open/v1/offer/apply_creative` | JSON | token；写操作不可盲重试 | **是**，改用 Creative Set | [Update Creatives](https://helpcenter.mintegral.com/cn/docs/update-creative-asset-api) |
| 14 | UpdateOfferAudience | `PUT /api/open/v1/offer/target-audience` | JSON | token；写操作不可盲重试 | 否；当前帮助页路径有差异，见下文 | [Update Audience Target](https://helpcenter.mintegral.com/cn/docs/update-ad-unit-audience-targeting-api) |
| 15 | UpdateOfferTargetGoal | `PUT /api/open/v3/offer/target_goal` | JSON | token；空地区数组清除对应目标；写操作不可盲重试 | 否 | [Update Target Goal](https://helpcenter.mintegral.com/cn/docs/update-optimization-objective-api) |
| 16 | ListBidGoalSupports | `GET /api/open/v3/event/bid_goal_supports` | query、无 body | token；必须指定 `campaign_id` 或 `package_name`；目标仅 `Target-CPE`/`Target-ROAS` | 否 | [Bid Goal Supports](https://helpcenter.mintegral.com/cn/docs/get-event-api) |
| 17 | ListCreativeSets | `GET /api/open/v1/creative_sets` | query、无 body | token；`limit` 最大 50，默认 10 | 否 | [Creative Set List](https://adv.mintegral.com/doc/en/guide/creative_set/getCreativeSet.html) |
| 18 | CreateCreativeSet | `POST /api/open/v1/creative_set` | JSON | token；写操作不可盲重试 | 否 | [Create Creative Set](https://adv.mintegral.com/doc/en/guide/creative_set/createCreativeSet.html) |
| 19 | UpdateCreativeSet | `PUT /api/open/v1/creative_set` | JSON | token；写操作不可盲重试 | 否 | [Update Creative Set](https://adv.mintegral.com/doc/en/guide/creative_set/updateCreativeSet.html) |
| 20 | DeleteCreativeSet | `DELETE /api/open/v1/creative_set` | JSON | token；写操作不可盲重试 | 否 | [Delete Creative Set](https://adv.mintegral.com/doc/en/guide/creative_set/deleteCreativeSet.html) |
| 21 | ListCreativeAds | `GET /api/open/v1/creative-ad/list` | **JSON body** | token；`ad_ids` 必填；页从 1 起，默认 20，`limit≤100` | 否 | [Get Ads](https://adv.mintegral.com/doc/en/guide/creative_set/getAdList.html) |
| 22 | ListCreativeSources | `GET /api/open/v1/creatives/source` | query、无 body | token；`limit` 最大 200；获取素材来源/规格 | 否 | [Creative List](https://adv.mintegral.com/doc/en/guide/creative/getCreative.html) |
| 23 | UploadCreative | `POST /api/open/v1/creatives/upload` | storage multipart | token；文件上传，重试需可重放 body | 否 | [Upload Creatives](https://adv.mintegral.com/doc/en/guide/creative/uploadCreative.html) |
| 24 | UploadPlayable | `POST /api/open/v1/playable/upload` | storage multipart | token；文件上传，重试需可重放 body | 否 | [Upload Playable](https://adv.mintegral.com/doc/en/guide/creative/uploadPlayable.html) |
| 25 | AdvancedReport | `GET /api/v2/reports/data` | query、无 body；`type=1` 状态/生成，`type=2` TSV 下载 | token；`start_time/end_time` 为 `YYYY-MM-DD` 日期文本；闭区间最多 7 天；默认 daily/+8；type=1 与 type=2 必须按 v2 状态机配合 | 否 | [Advanced Report](https://adv-new.mintegral.com/doc/en/guide/report/advancedPerformanceReport.html) |
| 26 | LookupTargetAppName | `POST /api/open/v1/target-apps/app-name` | JSON | token；按目标 app 查询名称 | 否 | [App Name](https://adv.mintegral.com/doc/en/guide/report/getAppName.html) |
| 27 | ListAudiences | `GET /api/open/v1/audience` | query、无 body | token；`limit` 默认 10、最大 500，页从 1 起 | 否 | [Audience List](https://helpcenter.mintegral.com/en/docs/get-audience-package-list) |
| 28 | GetAudiencePresign | `GET /api/open/v1/audience/presigned-upload-data` | query | token；返回短期 bearer URL/表单，不记录日志；文件最大 5 GB，支持 txt/csv 及 gzip 变体 | 否 | [Upload Audience File](https://helpcenter.mintegral.com/en/docs/upload-audience-package-file) |
| 29 | CreateAudience | `POST /api/open/v1/audience` | JSON | token；创建后最迟约 12 小时才能用于定向 | 否 | [Create Audience](https://helpcenter.mintegral.com/en/docs/create-audience-package) |
| 30 | UpdateAudience | `PUT /api/open/v1/audience` | JSON | token；服务端禁止在创建或上次修改后 12 小时内再次更新；不可盲重试 | 否 | [Update Audience](https://helpcenter.mintegral.com/en/docs/update-audience-package) |
| 31 | DeleteAudience | `DELETE /api/open/v1/audience` | JSON | token；写操作不可盲重试 | 否 | [Delete Audience](https://helpcenter.mintegral.com/en/docs/delete-audience-package) |

### Audience 外部预签名上传分支（不属于上述 31 个 Open API）

| 分支 | 方法 / 目标 | 编码 | 权限与限制 | 来源 |
| --- | --- | --- | --- | --- |
| S3 | `PUT` 到 `Audiences().PresignUpload` 返回的 URL | 原始文件字节，绝非 multipart | `area_type=1`；不带 Mintegral token；同一文件快照且未过期时最多发送两次 | [Upload Audience File](https://helpcenter.mintegral.com/en/docs/upload-audience-package-file) |
| OSS | `POST` 到 `Audiences().PresignUpload` 返回的 host | `multipart/form-data`，字段 `key`、`OSSAccessKeyId`、`policy`、小写 `signature`、`success_action_status=200`、`file` | `area_type=2`；不带 Mintegral token；预签名 URL/字段均为 bearer secret；同一文件快照且签名有效时最多发送两次 | [Upload Audience File](https://helpcenter.mintegral.com/en/docs/upload-audience-package-file) |

## 已裁决的文档矛盾

| 主题 | SDK 决策 |
| --- | --- |
| Report v1/v2 | 仅实现 v2 Advanced Report。`start_time`、`end_time`、`dimension_option` 与 `type=1/2` 固定走 `/api/v2/reports/data`；不得因失败回退 v1。 |
| GET 的 body | 不能根据 HTTP 方法推断编码。Campaign 与 Creative Ad 按 Provider 文档使用 GET + JSON body；Offer、Creative Set、素材来源和 Audience 按 SDK 的固定线路使用 query、无 body；报表 v2 也使用 query、无 body。 |
| Creative Ad | `ad_ids` 是必填；未提供时 Provider 可能返回 HTTP 200、业务码 10000。页号默认 1、limit 默认 20、最大 100。 |
| 时间 | 当前官方 v2 文档契约使用 `YYYY-MM-DD` 的 `start_time/end_time`。历史运行记录曾对另一/旧契约观察到“必须为整数”的业务错误；SDK 以当前官方日期文本契约为准，不根据错误自动切换为 Unix 秒或回退其他版本。 |
| `app_size` | Create Campaign 的请求类型为 JSON number，而响应示例为 JSON string。请求/响应使用不同 DTO，不能做全局 string/number 宽松转换。 |
| `msg` / `message` | Provider 页面与示例并存两种字段。SDK 接受任一字段；两者非空且不一致即拒绝响应，防止隐藏冲突。 |
| Audience IDFV | 旧页面使用 device type 10/11，Help Center 的现行页面使用 12/13。SDK 保留线路原值：当前 profile 创建请求使用 12/13；legacy profile 仅显式选择时使用 10/11；读取时不静默改写。 |
| Offer audience 路径 | 已批准 SDK 固定使用 `/api/open/v1/offer/target-audience`；当前帮助页仍展示 `/api/open/v1/offer/audience`。该差异不可自动探测或回退，升级前需重新核对 Provider 契约。 |
| 报表数值 | v2 `type=2` 是 TSV，金额/比率以文本单元格接收并保留十进制精度，不能先转 `float64`。 |

## 实现守则

- 每个 operation 固定 method、path、编码、成功状态和重试分类；不得自动探测 GET body/query，也不得自动切换 API 版本。
- 响应新增字段、未知 enum 与 TSV 新列应保持兼容；已知必填字段缺失或类型改变则为契约错误。
- API key、token、签名、预签名 query、上传内容与 URL 永远不可出现在错误和日志中。
- 所有分页基于响应元数据前进；调用方应为逐页遍历设置记录数或页数预算。
