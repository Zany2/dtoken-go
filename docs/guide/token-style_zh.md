# Token 风格

[English](token-style.md) | 中文文档

DToken-Go 的 `TokenStyle` 控制 Token 字符串的生成方式。无论使用哪种风格，登录态、Session、权限、封禁、踢下线和顶下线仍然由服务端存储参与管理。

## 支持的风格

| 风格 | 常量 | 特点 | 适用场景 |
| --- | --- | --- | --- |
| UUID | `adapter.TokenStyleUUID` | 默认风格，长度固定，通用性好 | 大多数业务 |
| Simple | `adapter.TokenStyleSimple` | 简单随机字符串 | 内部系统、测试 |
| Random32 | `adapter.TokenStyleRandom32` | 32 位随机字符串 | 需要较短随机 Token |
| Random64 | `adapter.TokenStyleRandom64` | 64 位随机字符串 | 更高随机强度 |
| Random128 | `adapter.TokenStyleRandom128` | 128 位随机字符串 | 高安全要求 |
| Hash | `adapter.TokenStyleHash` | SHA256 哈希风格 | 希望 Token 表现为哈希串 |
| Timestamp | `adapter.TokenStyleTimestamp` | 包含时间戳特征 | 调试、排查、内部链路 |
| Tik | `adapter.TokenStyleTik` | 短 ID 风格 | 希望 Token 较短的场景 |
| JWT | `adapter.TokenStyleJWT` | 可解析 Claim，长度较长 | 网关协作、调试、需要 Claim 的场景 |

## 基本使用

```go
mgr, err := defaults.NewBuilder().
    TokenStyle(adapter.TokenStyleRandom64).
    SetStorage(storage).
    Build()
```

JWT 风格需要设置密钥：

```go
mgr, err := defaults.NewBuilder().
    TokenStyle(adapter.TokenStyleJWT).
    JwtSecretKey("your-very-strong-secret").
    SetStorage(storage).
    Build()
```

也可以使用快捷方法：

```go
mgr, err := defaults.NewBuilder().
    JwtSecret("your-very-strong-secret").
    SetStorage(storage).
    Build()
```

## JWT 不是纯无状态

`TokenStyleJWT` 只表示 Token 字符串采用 JWT 格式，不表示 DToken-Go 变成纯无状态认证。

以下能力仍然依赖服务端存储：

- 登录态校验
- Session 与终端管理
- 自动续期与活跃超时
- 权限和角色
- 账号、服务、设备封禁
- `Logout`、`Kickout`、`Replace`

完整 JWT 说明见 [JWT 指南](jwt_zh.md)。

## 自定义生成器

如果内置风格不满足要求，可以实现 `adapter.Generator`：

```go
type MyGenerator struct{}

func (MyGenerator) Generate(loginID, device, deviceID string) (string, error) {
    return "custom-token-value", nil
}

mgr, err := defaults.NewBuilder().
    SetGenerator(MyGenerator{}).
    SetStorage(storage).
    Build()
```

自定义生成器应保证：

- Token 不容易被猜测。
- 同一时间大量生成时冲突概率足够低。
- 不直接泄露敏感业务信息。
- 生成失败时返回明确错误。

## 选择建议

| 需求 | 推荐 |
| --- | --- |
| 默认通用场景 | UUID |
| 更高随机强度 | Random64 或 Random128 |
| 需要 Claim 可读性 | JWT |
| 需要较短 Token | Tik 或 Simple |
| 内部排查需要时间特征 | Timestamp |
| 有统一企业 Token 规范 | 自定义 Generator |

## 相关文档

- [JWT 指南](jwt_zh.md)
- [配置指南](configuration_zh.md)
- [组件生态](component-ecosystem_zh.md)
