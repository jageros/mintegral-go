# mintegral-go 开发约定

## 范围

这是一个独立的 Go SDK（模块：`github.com/jageros/mintegral-go`），只负责 Mintegral Open API 的强类型客户端。不得向本仓库引入数据库、调度器、断点、业务日志、HTTP 服务或调用方业务模型。

## 变更规则

- 开始前检查工作区现状；保留其他人的未提交改动，不回退、不格式化无关文件。
- 所有网络公开方法首参为 `context.Context`；不要丢弃取消/超时原因。
- 公开 API 使用强类型请求、响应和语义化 ID/时间/金额；金额和比例保留十进制文本，不用 `float64` 作为线路类型。
- 凭据、token、签名、预签名 URL 和文件内容不可写入错误、日志、测试夹具或文档示例。错误应支持 `errors.Is/As`。
- 每个 endpoint 显式定义 method、path、编码、认证、可重试性和副作用；不得按 HTTP 方法猜测 body，也不得将 Report v1/v2 自动互相回退。
- 读操作的有限重试必须尊重 context、`Retry-After` 与请求 body 的可重放性。创建、更新和删除不得盲重试；素材上传默认不重试。Audience 的 S3/OSS 仅在预签名仍有效、对象标识与文件快照完全一致且 body 可重新打开时，最多额外精确重放一次；其他结果不确定的写请求返回 `ErrOutcomeUnknown`。
- 流式回调的行/切片所有权必须在类型和文档中说明。跨账户并发调用回调时，调用方必须自行同步。
- 更新 API 契约时同步维护 `docs/api-contract.zh-CN.md`，并记录 Provider 文档检查日期。

## 验证

修改 Go 代码后，按影响范围执行：

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

测试必须使用内存 `http.RoundTripper`/`httptest`、固定 clock 和虚拟凭据；禁止以真实 Provider 调用、真实凭据或外部存储作为 CI 测试。文档/配置改动至少运行对应文本检查和 `task fmt-check`。
