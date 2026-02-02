# DToken Quick Start Example

这是一个使用 Gin 框架和 DToken 的完整示例项目，展示了所有功能的使用方法。

## 快速开始

### 1. 准备 Redis

确保 Redis 服务已启动并可访问：
- 地址：192.168.19.105:6379
- 密码：root
- 数据库：0

如需修改 Redis 配置，请编辑 `main.go` 中的 `initDToken()` 函数。

### 2. 安装依赖

```bash
cd examples/quick_start
go mod tidy
```

### 3. 运行服务

```bash
go run main.go
```

服务将在 `http://localhost:8080` 启动。

## API 接口文档

### 公开接口

#### 测试接口
```bash
GET /api/test
```

#### 登录
```bash
POST /api/login
Content-Type: application/json

{
  "loginId": "user001",
  "device": "PC",
  "deviceId": "device001"
}
```

### 认证接口（需要在 Header 或 Query 中携带 token）

#### Token 管理

##### 登出
```bash
POST /api/logout
Header: token: <your-token>
```

##### 根据设备类型登出
```bash
POST /api/logout/device?loginId=user001&device=PC
Header: token: <your-token>
```

##### 获取 Token 信息
```bash
GET /api/token/info
Header: token: <your-token>
```

##### 获取 Token 列表
```bash
GET /api/token/list?loginId=user001&checkAlive=true
Header: token: <your-token>
```

##### 获取 Token 剩余时间
```bash
GET /api/token/ttl
Header: token: <your-token>
```

##### 获取在线终端数量
```bash
GET /api/online/count?loginId=user001
GET /api/online/count?loginId=user001&device=PC
Header: token: <your-token>
```

#### 踢人/顶人下线

##### 踢人下线
```bash
POST /api/kickout?targetToken=<target-token>
Header: token: <your-token>
```

##### 根据设备类型踢人下线
```bash
POST /api/kickout/device?loginId=user001&device=PC
Header: token: <your-token>
```

##### 顶人下线
```bash
POST /api/replace?targetToken=<target-token>
Header: token: <your-token>
```

##### 根据设备类型顶人下线
```bash
POST /api/replace/device?loginId=user001&device=PC
Header: token: <your-token>
```

#### 账号封禁

##### 封禁账号
```bash
POST /api/disable
Header: token: <your-token>
Content-Type: application/json

{
  "loginId": "user001",
  "duration": 3600,
  "reason": "违规操作"
}
```

##### 解封账号
```bash
POST /api/undisable?loginId=user001
Header: token: <your-token>
```

##### 获取封禁信息
```bash
GET /api/disable/info?loginId=user001
Header: token: <your-token>
```

#### 权限管理

##### 添加权限
```bash
POST /api/permission/add
Header: token: <your-token>
Content-Type: application/json

{
  "loginId": "user001",
  "permissions": ["user:add", "user:delete", "user:*"]
}
```

##### 移除权限
```bash
POST /api/permission/remove
Header: token: <your-token>
Content-Type: application/json

{
  "loginId": "user001",
  "permissions": ["user:delete"]
}
```

##### 检查权限
```bash
GET /api/permission/check?permission=user:add
Header: token: <your-token>
```

##### 获取权限列表
```bash
GET /api/permission/list?loginId=user001
Header: token: <your-token>
```

#### 角色管理

##### 添加角色
```bash
POST /api/role/add
Header: token: <your-token>
Content-Type: application/json

{
  "loginId": "user001",
  "roles": ["admin", "user"]
}
```

##### 移除角色
```bash
POST /api/role/remove
Header: token: <your-token>
Content-Type: application/json

{
  "loginId": "user001",
  "roles": ["user"]
}
```

##### 检查角色
```bash
GET /api/role/check?role=admin
Header: token: <your-token>
```

##### 获取角色列表
```bash
GET /api/role/list?loginId=user001
Header: token: <your-token>
```

#### Session 管理

##### 获取 Session 信息
```bash
GET /api/session
Header: token: <your-token>
```

## 完整测试流程

### 1. 登录获取 Token
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"loginId":"user001","device":"PC","deviceId":"device001"}'
```

响应：
```json
{
  "code": 0,
  "msg": "登录成功",
  "data": {
    "token": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "loginId": "user001"
  }
}
```

### 2. 使用 Token 访问认证接口
```bash
curl -X GET http://localhost:8080/api/token/info \
  -H "token: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

### 3. 添加权限
```bash
curl -X POST http://localhost:8080/api/permission/add \
  -H "token: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"loginId":"user001","permissions":["user:add","user:delete"]}'
```

### 4. 检查权限
```bash
curl -X GET "http://localhost:8080/api/permission/check?permission=user:add" \
  -H "token: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

### 5. 获取 Session 信息
```bash
curl -X GET http://localhost:8080/api/session \
  -H "token: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

### 6. 登出
```bash
curl -X POST http://localhost:8080/api/logout \
  -H "token: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

## 配置说明

在 `main.go` 的 `initDToken()` 函数中可以修改配置：

```go
cfg := &config.Config{
    AuthType:         "login",        // 认证类型
    TokenName:        "token",        // Token 名称
    TokenStyle:       adapter.TokenStyleUUID, // Token 风格
    Timeout:          86400,          // Token 超时时间（秒）
    AutoRenew:        true,           // 是否自动续期
    RenewMaxRefresh:  604800,         // 最大续期时间（秒）
    RenewInterval:    3600,           // 续期间隔（秒）
    ActiveTimeout:    1800,           // 最大不活跃时长（秒）
    IsConcurrent:     true,           // 是否允许并发登录
    ConcurrencyScope: config.ConcurrencyScopeAccount, // 并发范围
    IsShare:          false,          // 是否共享 Token
    MaxLoginCount:    5,              // 最大登录数量
    IsPrintBanner:    true,           // 是否打印 Banner
}
```

## 功能特性

- ✅ 用户登录/登出
- ✅ Token 管理（获取信息、列表、TTL）
- ✅ 在线状态管理（踢人、顶人下线）
- ✅ 账号封禁/解封
- ✅ 权限管理（添加、移除、检查）
- ✅ 角色管理（添加、移除、检查）
- ✅ Session 管理
- ✅ 多设备支持
- ✅ 认证中间件
- ✅ 统一响应格式

## 注意事项

1. 本示例使用内存存储，重启后数据会丢失
2. 生产环境建议使用 Redis 等持久化存储
3. Token 可以通过 Header 或 Query 参数传递
4. 所有认证接口都需要有效的 Token
