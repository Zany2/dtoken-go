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
- **Refresh Token** - Login returns access and refresh tokens, and refresh rotates token pairs
- **Token Introspection** - Inspect the current token from the Gin request context
- **Annotation-Based Routes** - Various check middlewares (login, role, permission, all)

## Prerequisites

- Go 1.25 or higher

## Installation

```bash
cd examples/gin
go mod tidy
```

## Configuration

The example uses the bundled memory storage through `gindt.NewBuilder()`. Replace it with Redis storage in `initDToken()` when you need persistence.

## Running the Example

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## Available Endpoints

### Public Endpoints

- `POST /login` - User login
  - Request: `{"username": "admin", "password": "123456"}`
  - Response: Returns access and refresh tokens

- `POST /refresh` - Refresh access token
  - Request: `{"refreshToken": "<refresh-token>"}`

### Protected Endpoints (Requires Login)

- `GET /me` - Get current user information
  - Headers: `Authorization: Bearer <token>`

- `GET /introspect` - Inspect current token status
  - Headers: `Authorization: Bearer <token>`

- `POST /logout` - Logout current user
  - Headers: `Authorization: Bearer <token>`

### Admin Endpoints (Requires Admin Role)

- `GET /admin` - Admin data

### Resource Endpoints (Requires Permissions)

- `GET /articles` - List articles (requires `article:read` permission)

## Testing with cURL

### 1. Login

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

Response:
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "accessToken": "your-access-token-here",
    "refreshToken": "your-refresh-token-here",
    "tokenType": "Bearer"
  }
}
```

### 2. Get User Info

```bash
curl http://localhost:8080/me \
  -H "Authorization: Bearer your-access-token-here"
```

### 3. Refresh Token

```bash
curl -X POST http://localhost:8080/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken":"your-refresh-token-here"}'
```

### 4. Inspect Current Token

```bash
curl http://localhost:8080/introspect \
  -H "Authorization: Bearer your-access-token-here"
```

## Adding Roles and Permissions

To test role and permission-based endpoints, uncomment the following lines in `handleLogin` function:

The demo seeds `admin` and `article:read` during login so role and permission routes can be called immediately.

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
mgr := defaults.NewBuilder().
    Timeout(3600).       // 1 hour
    RefreshTokenTimeout(30 * 24 * 60 * 60).
    ActiveTimeout(1800). // 30 minutes
    MaxLoginCount(3).
    Build()
dtoken.SetManager(mgr)
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
