# Gin DToken Example

This example demonstrates how to use DToken-Go with the Gin web framework.

## Features

This example showcases:

- **Manager Initialization** - Using Redis storage with builder pattern
- **Middleware Registration** - DToken context middleware for all routes
- **Public Routes** - Login and public endpoints without authentication
- **Protected Routes** - User info and logout requiring authentication
- **Role-Based Access Control** - Admin endpoints requiring admin role
- **Permission-Based Access Control** - Resource endpoints requiring specific permissions
- **Annotation-Based Routes** - Various check middlewares (login, role, permission, all)

## Prerequisites

- Go 1.21 or higher
- Redis server running (default: `192.168.19.104:6379`)

## Installation

```bash
cd examples/gin
go mod tidy
```

## Configuration

Update the Redis connection string in `main.go` if needed:

```go
storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
```

## Running the Example

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## Available Endpoints

### Public Endpoints

- `POST /api/login` - User login
  - Request: `{"username": "admin", "password": "123456"}`
  - Response: Returns token for authentication

- `GET /api/public` - Public endpoint (no authentication required)

### Protected Endpoints (Requires Login)

- `GET /api/user/info` - Get current user information
  - Headers: `Authorization: Bearer <token>`

- `POST /api/user/logout` - Logout current user
  - Headers: `Authorization: Bearer <token>`

### Admin Endpoints (Requires Admin Role)

- `GET /api/admin/users` - List all users
- `POST /api/admin/disable` - Disable a user account
  - Request: `{"username": "user1"}`
- `POST /api/admin/enable` - Enable a user account
  - Request: `{"username": "user1"}`

### Resource Endpoints (Requires Permissions)

- `GET /api/resource/list` - List resources (requires `resource:read` permission)

### Annotation-Based Endpoints

- `GET /api/annotation/profile` - User profile (login check only)
- `GET /api/annotation/admin-data` - Admin data (requires admin role)
- `GET /api/annotation/sensitive` - Sensitive data (requires `data:read` permission)
- `GET /api/annotation/super` - Super admin data (requires `super-admin` role and `all:access` permission)

## Testing with cURL

### 1. Login

```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

Response:
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "token": "your-token-here",
    "username": "admin"
  }
}
```

### 2. Get User Info

```bash
curl http://localhost:8080/api/user/info \
  -H "Authorization: Bearer your-token-here"
```

### 3. Access Admin Endpoint

```bash
curl http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer your-token-here"
```

## Adding Roles and Permissions

To test role and permission-based endpoints, uncomment the following lines in `handleLogin` function:

```go
_ = gindt.AddRoles(c.Request.Context(), req.Username, []string{"admin", "super-admin"})
_ = gindt.AddPermissions(c.Request.Context(), req.Username, []string{"resource:read", "resource:write", "data:read", "all:access"})
```

## Project Structure

```
examples/gin/
├── main.go          # Main application file
├── go.mod           # Go module file
└── README.md        # This file
```

## Key Concepts

### Manager Initialization

```go
builder := gindt.NewDefaultBuilder()
mgr := builder.
    SetStorage(storage).
    Timeout(3600).       // 1 hour
    ActiveTimeout(1800). // 30 minutes
    MaxLoginCount(3).
    Build()
gindt.SetManager(mgr)
```

### Middleware Usage

```go
// Register DToken context middleware
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

// Authentication middleware
user := r.Group("/api/user")
user.Use(gindt.AuthMiddleware(ctx))

// Role middleware
admin := r.Group("/api/admin")
admin.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))

// Permission middleware
resource := r.Group("/api/resource")
resource.Use(gindt.PermissionMiddleware(ctx, []string{"resource:read"}))
```

### Annotation-Based Checks

```go
// Check login only
annotation.GET("/profile", gindt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))

// Check role
annotation.GET("/admin-data", gindt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdminData, handleAuthFail))

// Check permission
annotation.GET("/sensitive", gindt.CheckPermissionMiddleware(ctx, []string{"data:read"}, handleSensitiveData, handleAuthFail))
```

### Getting DToken Context

```go
// Get DToken context from Gin context
dCtx, ok := gindt.GetDTokenContext(c)
if !ok {
    // Handle error
}
tokenValue := dCtx.GetTokenValue()
```

## Differences from GoFrame Example

- Uses `gin.Default()` instead of `g.Server()`
- Uses `c.JSON()` for responses instead of `r.Response.WriteJson()`
- Uses `c.ShouldBind()` for request binding
- Uses `c.Request.Context()` for context access
- Uses `gindt.GetDTokenContext(c)` to get DToken context

## License

This example is part of the DToken-Go project.
