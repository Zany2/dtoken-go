# FEATURE.md

## 下一版本计划

下一版本主要围绕跨系统认证、临时凭证、统一登录态查询、运维清理和网关集成能力进行迭代。本文档记录规划方向，不代表所有能力已经完成。

## 核心方向

### 1. Ticket 临时凭证

- 设计统一的临时凭证模型，作为 SSO、短 Key 访问凭证、扫码确认、临时授权等能力的底座。
- 支持创建 Ticket，并绑定 loginID、authType、device、deviceId、业务来源、目标应用和扩展数据。
- 支持 Ticket 校验、一次性消费、撤销、过期时间查询和状态查询。
- 明确 Ticket 状态：有效、已消费、已撤销、已过期、无效。
- 支持 Ticket 换取正式 Token，或换取业务侧自定义结果。
- 复用当前 Storage、Codec、Event、Manager 设计，不单独引入新的存储体系。

### 2. 短 Key 访问凭证

- 增加短 Key 访问凭证能力，用于短链接访问、扫码确认、一次性访问、临时授权和系统间换票。
- 支持生成带有效期的短 Key。
- 支持短 Key 绑定 loginID、authType、device、deviceId、来源应用、目标应用和扩展元数据。
- 支持短 Key 一次性消费，消费成功后可换取正式 Token、Ticket 或业务侧自定义结果。
- 支持查询短 Key 状态和剩余有效时间。
- 明确短 Key 过期、已消费、已撤销、无效、信息不匹配时的错误返回。
- 补充短 Key 创建、消费、撤销、过期等事件。

### 3. SSO 单点登录

- 增加 SSO 单点登录能力，支持多个业务系统之间共享登录状态。
- 设计 SSO 票据生成、校验、续期、撤销和交换流程。
- 支持 SSO 登录回调，方便业务系统接入统一登录中心。
- 支持 SSO 登出回调，方便业务系统同步清理本地登录态。
- 支持 SSO 票据换取当前系统 Token。
- 支持统一登出、指定应用登出、指定终端登出和全部应用登出。
- 明确 SSO 与现有 Token、Session、Terminal、权限、角色、事件系统之间的关系。

### 4. Token Introspection

- 增加标准化 Token 状态查询接口。
- 返回 Token 是否有效、loginID、authType、device、deviceId、创建时间、剩余 TTL、状态原因等信息。
- 支持区分普通无效、过期、被踢下线、被顶下线、活跃超时、账号封禁、设备封禁等状态。
- 支持普通 Token 和 OAuth2 Access Token 的查询边界说明。
- 用于网关、SSO、第三方系统鉴权和调试排查。

### 5. 登录态清理能力

- 当前 Terminal 存在懒清理策略，下一版本可补充主动清理能力。
- 支持清理指定账号下已失效 Terminal。
- 支持按 authType、loginID、device、deviceId 清理过期登录态。
- 支持可选的后台清理任务。
- 清理动作需要和 Token 状态、metadata 清理、Session 销毁事件保持一致。

### 6. 事件系统补充

- 补充 Ticket 创建、校验、消费、撤销事件。
- 补充短 Key 创建、消费、撤销事件。
- 补充 SSO 登录、SSO 登出、SSO 票据校验事件。
- 评估是否需要补充 Token Introspection 事件。
- 统一事件 Extra 字段命名，例如 action、reason、sourceApp、targetApp、ticketType、result 等。

### 7. 多应用维度

- 评估是否需要在 SSO、Ticket、短 Key 访问凭证中引入 appId、clientId 或 systemId。
- 明确 appId 与 authType、device、deviceId 的边界。
- 支持按应用维度查询、登出、清理登录态。
- 支持限制某个 Ticket 或短 Key 只能被指定应用消费。

### 8. 独立 Refresh Token 能力

- 评估是否需要引入独立于 OAuth2 的业务登录 Refresh Token。
- 明确 Refresh Token 与当前 LoginByToken、RenewTimeout、AutoRenew 的区别。
- 如果引入，需要支持签发、刷新、撤销、过期、轮换和安全校验。
- 保持现有 Token 续期逻辑兼容，不影响已有使用方式。

### 9. 错误码补充

- 增加 Ticket 相关错误：无效、过期、已消费、已撤销、信息不匹配。
- 增加短 Key 相关错误：无效、过期、已消费、已撤销、信息不匹配。
- 增加 SSO 相关错误：票据无效、应用未授权、回调地址不匹配、SSO 会话不存在。
- 增加 Token Introspection 场景下的状态原因说明。

### 10. 框架集成和示例

- 补充 Gin、Echo、Fiber、GoFrame、Hertz、Chi、Kratos 的下一版本使用示例。
- 示例覆盖普通登录、权限校验、SSO、短 Key 访问凭证、统一登出等场景。
- 确认 integrations 中导出的 API 与 dtoken 核心 API 保持一致。

## 兼容性要求

- 尽量保持现有登录、登出、Session、权限、角色、封禁、Nonce、OAuth2、事件和框架集成 API 兼容。
- 优先新增 API，不轻易修改现有方法签名。
- 新功能需要兼容当前多认证类型 authType、设备维度 device/deviceId、终端管理逻辑。
- 新功能优先复用现有 Manager、Session、TokenInfo、Storage、Codec、Event 设计。
