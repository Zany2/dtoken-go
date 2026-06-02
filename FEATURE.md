# DToken-Go Feature Notes

[中文](#中文)

This file tracks feature status and near-term roadmap items. It is intentionally lightweight; detailed usage stays in `docs/`.

## Available

| Feature | Status |
| --- | --- |
| Login authentication | Available |
| Token lifecycle management | Available |
| Session and terminal management | Available |
| Role and permission checks | Available |
| Account, service, and device disable | Available |
| Nonce anti-replay | Available |
| OAuth2 server primitives | Available |
| SSO primitives | Available |
| Temporary Ticket | Available |
| Short-key access credential | Available |
| Token Introspection | Available |
| Refresh Token for normal login | Available |
| Framework middleware tests | Available |
| Framework examples for Refresh Token, Introspection, and route access | Available |

## Roadmap

| Feature | Notes |
| --- | --- |
| Full HTTP SSO service | Build a complete service layer on top of existing SSO primitives |
| OAuth2 PKCE | Add authorization-code proof-key support for public clients |

## 中文

本文件用于记录功能状态与近期规划。详细使用方式仍放在 `docs/` 目录下。

## 已可用

| 能力 | 状态 |
| --- | --- |
| 登录认证 | 已可用 |
| Token 生命周期管理 | 已可用 |
| Session 与终端管理 | 已可用 |
| 角色与权限校验 | 已可用 |
| 账号、服务、设备封禁 | 已可用 |
| Nonce 防重放 | 已可用 |
| OAuth2 服务端基础能力 | 已可用 |
| SSO 基础能力 | 已可用 |
| Ticket 临时凭证 | 已可用 |
| 短 Key 访问凭证 | 已可用 |
| Token Introspection | 已可用 |
| 普通登录 Refresh Token | 已可用 |
| 框架中间件测试 | 已可用 |
| Refresh Token、Introspection、路由访问规则框架示例 | 已可用 |

## 规划中

| 能力 | 说明 |
| --- | --- |
| 完整 HTTP SSO 服务 | 基于现有 SSO 基础能力补充完整服务层 |
| OAuth2 PKCE | 为公开客户端补充授权码 proof-key 支持 |
