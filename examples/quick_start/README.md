# DToken Quick Start - 完整测试示例

这是一个完整的 DToken 框架测试示例，包含了框架所有功能的 API 接口实现。

## 功能概览

本示例实现了 DToken 框架的所有核心功能：

### 1. 认证管理 (Authentication)
- ✅ 用户登录（支持设备和设备ID）
- ✅ Token 续期登录
- ✅ 用户登出（支持按 Token、设备、设备ID）
- ✅ 登录状态检查
- ✅ Token 信息获取（登录ID、设备、创建时间、TTL等）
- ✅ 在线终端数量统计

### 2. 在线状态管理 (Online Status)
- ✅ 踢人下线（Kickout）- 按 Token、设备、设备ID
- ✅ 顶人下线（Replace）- 按 Token、设备、设备ID

### 3. 权限管理 (Permission)
- ✅ 添加/删除权限（支持按 LoginID 和 Token）
- ✅ 获取权限列表
- ✅ 权限检查（单个权限、AND逻辑、OR逻辑）

### 4. 角色管理 (Role)
- ✅ 添加/删除角色（支持按 LoginID 和 Token）
- ✅ 获取角色列表
- ✅ 角色检查（单个角色、AND逻辑、OR逻辑）

### 5. Session 管理
- ✅ 获取会话信息
- ✅ 获取 Token 列表（支持按设备筛选）

### 6. 账号封禁管理 (Disable)
- ✅ 封禁账号（指定时长和原因）
- ✅ 解封账号
- ✅ 检查封禁状态
- ✅ 获取封禁信息和剩余时间

## 快速开始

### 1. 前置条件

确保已安装：
- Go 1.25+
- Redis（默认配置：localhost:6379）

### 2. 安装依赖

```bash
cd examples/quick_start
go mod tidy

# 安装 Swagger 文档生成工具
go install github.com/swaggo/swag/cmd/swag@latest
```

### 3. 生成 Swagger 文档

```bash
# 在 quick_start 目录下执行
swag init

# 这将生成 docs/ 目录，包含 Swagger 文档文件
```

### 4. 启动 Redis

```bash
# 使用 Docker 启动 Redis
docker run -d -p 6379:6379 redis:latest

# 或使用本地 Redis
redis-server
```

### 5. 运行示例

```bash
go run main.go
```

服务器将在 `http://localhost:8080` 启动。

### 6. 访问 Swagger UI

打开浏览器访问：
```
http://localhost:8080/swagger/index.html
```

在 Swagger UI 中，您可以：
- 📖 查看所有 API 接口的详细文档
- 🧪 直接在浏览器中测试 API 接口
- 📝 查看请求/响应的数据结构
- 🔍 按标签（Tags）浏览不同功能模块的接口

## API 接口文档

### 认证接口 (`/api/auth`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 用户登录 |
| POST | `/api/auth/login-by-token` | Token 续期登录 |
| POST | `/api/auth/logout` | 用户登出 |
| POST | `/api/auth/logout-by-device` | 按设备类型登出 |
| POST | `/api/auth/logout-by-device-id` | 按设备和设备ID登出 |
| POST | `/api/auth/is-login` | 检查是否登录 |
| POST | `/api/auth/check-login` | 验证登录状态 |
| POST | `/api/auth/get-login-id` | 获取登录ID |
| POST | `/api/auth/get-token-info` | 获取Token信息 |
| POST | `/api/auth/get-device` | 获取设备类型 |
| POST | `/api/auth/get-device-id` | 获取设备ID |
| POST | `/api/auth/get-token-create-time` | 获取Token创建时间 |
| POST | `/api/auth/get-token-ttl` | 获取Token TTL |
| GET | `/api/auth/online-count/:loginId` | 获取在线终端总数 |
| GET | `/api/auth/online-count/:loginId/:device` | 获取指定设备在线终端数 |

### 在线状态管理接口 (`/api/online`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/online/kickout` | 根据Token踢人下线 |
| POST | `/api/online/kickout-by-device` | 根据设备类型踢人下线 |
| POST | `/api/online/kickout-by-device-id` | 根据设备和设备ID踢人下线 |
| POST | `/api/online/replace` | 根据Token顶人下线 |
| POST | `/api/online/replace-by-device` | 根据设备类型顶人下线 |
| POST | `/api/online/replace-by-device-id` | 根据设备和设备ID顶人下线 |

### 权限管理接口 (`/api/permission`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/permission/add` | 添加权限 |
| POST | `/api/permission/add-by-token` | 根据Token添加权限 |
| POST | `/api/permission/remove` | 删除权限 |
| POST | `/api/permission/remove-by-token` | 根据Token删除权限 |
| GET | `/api/permission/list/:loginId` | 获取权限列表 |
| POST | `/api/permission/list-by-token` | 根据Token获取权限列表 |
| POST | `/api/permission/has` | 检查是否拥有指定权限 |
| POST | `/api/permission/has-by-token` | 根据Token检查权限 |
| POST | `/api/permission/has-and` | 检查是否拥有所有权限（AND） |
| POST | `/api/permission/has-and-by-token` | 根据Token检查所有权限（AND） |
| POST | `/api/permission/has-or` | 检查是否拥有任一权限（OR） |
| POST | `/api/permission/has-or-by-token` | 根据Token检查任一权限（OR） |

### 角色管理接口 (`/api/role`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/role/add` | 添加角色 |
| POST | `/api/role/add-by-token` | 根据Token添加角色 |
| POST | `/api/role/remove` | 删除角色 |
| POST | `/api/role/remove-by-token` | 根据Token删除角色 |
| GET | `/api/role/list/:loginId` | 获取角色列表 |
| POST | `/api/role/list-by-token` | 根据Token获取角色列表 |
| POST | `/api/role/has` | 检查是否拥有指定角色 |
| POST | `/api/role/has-by-token` | 根据Token检查角色 |
| POST | `/api/role/has-and` | 检查是否拥有所有角色（AND） |
| POST | `/api/role/has-and-by-token` | 根据Token检查所有角色（AND） |
| POST | `/api/role/has-or` | 检查是否拥有任一角色（OR） |
| POST | `/api/role/has-or-by-token` | 根据Token检查任一角色（OR） |

### Session 管理接口 (`/api/session`)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/session/:loginId` | 获取会话 |
| POST | `/api/session/by-token` | 根据Token获取会话 |
| GET | `/api/session/tokens/:loginId` | 获取所有Token |
| GET | `/api/session/tokens/:loginId/:device` | 获取指定设备的所有Token |

### 账号封禁管理接口 (`/api/disable`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/disable/ban` | 封禁账号 |
| POST | `/api/disable/unban` | 解封账号 |
| GET | `/api/disable/is-disabled/:loginId` | 检查是否被封禁 |
| GET | `/api/disable/info/:loginId` | 获取封禁信息 |
| GET | `/api/disable/ttl/:loginId` | 获取封禁TTL |

## 使用示例

### 1. 用户登录

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "loginId": "user001",
    "device": "PC",
    "deviceId": "device-001"
  }'
```

响应：
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "xxx-xxx-xxx",
    "loginId": "user001"
  }
}
```

### 2. 检查登录状态

```bash
curl -X POST http://localhost:8080/api/auth/is-login \
  -H "Content-Type: application/json" \
  -d '{
    "token": "xxx-xxx-xxx"
  }'
```

### 3. 添加权限

```bash
curl -X POST http://localhost:8080/api/permission/add \
  -H "Content-Type: application/json" \
  -d '{
    "loginId": "user001",
    "permissions": ["user:read", "user:write"]
  }'
```

### 4. 检查权限

```bash
curl -X POST http://localhost:8080/api/permission/has \
  -H "Content-Type: application/json" \
  -d '{
    "loginId": "user001",
    "permission": "user:read"
  }'
```

### 5. 封禁账号

```bash
curl -X POST http://localhost:8080/api/disable/ban \
  -H "Content-Type: application/json" \
  -d '{
    "loginId": "user001",
    "duration": 3600,
    "reason": "违规操作"
  }'
```

## 测试建议

### 完整测试流程

1. **认证测试**
   - 测试登录功能
   - 测试 Token 续期
   - 测试登出功能
   - 测试多设备登录

2. **权限测试**
   - 添加权限
   - 检查权限（单个、AND、OR）
   - 删除权限
   - 验证权限失效

3. **角色测试**
   - 添加角色
   - 检查角色（单个、AND、OR）
   - 删除角色
   - 验证角色失效

4. **在线状态测试**
   - 测试踢人下线
   - 测试顶人下线
   - 验证在线终端数量

5. **封禁测试**
   - 封禁账号
   - 验证封禁状态
   - 解封账号
   - 验证解封后可正常登录

6. **Session 测试**
   - 获取会话信息
   - 获取 Token 列表
   - 验证会话数据完整性

## 配置说明

### Redis 配置

在 `initDToken()` 函数中修改 Redis 配置：

```go
storage := redis.NewRedisStorage(&redis.Config{
    Addr:     "localhost:6379",  // Redis 地址
    Password: "",                 // Redis 密码
    DB:       0,                  // Redis 数据库
})
```

### DToken 配置

在 `initDToken()` 函数中修改 DToken 配置：

```go
mgr, err := builder.NewBuilder().
    AuthType("login").           // 认证体系类型
    KeyPrefix("dtoken:").        // 存储键前缀
    TokenName("token").          // Token 名称
    Timeout(7200).               // 超时时间（秒）
    AutoRenew(true).             // 启用自动续期
    RenewMaxRefresh(1800).       // 续期触发阈值（秒）
    IsConcurrent(true).          // 允许并发登录
    MaxLoginCount(5).            // 最大并发登录数
    IsLog(true).                 // 开启日志
    Storage(storage).            // 设置存储适配器
    Build()
```

## 注意事项

1. **Redis 连接**：确保 Redis 服务正常运行
2. **端口占用**：默认使用 8080 端口，如需修改请在 `main()` 函数中更改
3. **错误处理**：所有接口都包含完整的错误处理
4. **日志输出**：框架会输出详细的日志信息，便于调试

## 项目结构

```
quick_start/
├── main.go          # 主程序文件（包含所有接口实现）
├── README.md        # 本文档
└── go.mod           # Go 模块文件
```

## 代码统计

- **总行数**: 1429 行
- **接口数量**: 60+ 个完整的 API 接口
- **功能模块**: 6 大功能模块
- **测试覆盖**: 覆盖 DToken 框架所有核心功能

## 贡献

如果发现任何问题或有改进建议，欢迎提交 Issue 或 Pull Request。

## 许可证

本示例代码遵循 DToken 框架的许可证。
