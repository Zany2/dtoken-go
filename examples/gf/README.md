# GoFrame DToken Example

This example demonstrates how to use DToken-Go with the GoFrame web framework.

## Features

This example showcases:

- **Manager Initialization** - Using Redis storage with builder pattern
- **Middleware Registration** - DToken context middleware for all routes
- **Public Routes** - Login and public endpoints without authentication
- **Protected Routes** - User info and logout requiring authentication
- **Role-Based Access Control** - Admin endpoints requiring admin role
- **Permission-Based Access Control** - Resource endpoints requiring specific permissions
- **Refresh Token** - Login returns access and refresh tokens, and refresh rotates token pairs
- **Token Introspection** - Inspect the current token from the GoFrame request context
- **Route Access Rules** - Use one middleware to decide login, permission, and role rules by URL
- **Annotation-Based Routes** - Various check middlewares (login, role, permission, all)

## Prerequisites

- Go 1.25 or higher

## Installation

```bash
cd examples/gf
go mod tidy
```

## Configuration

The example uses the bundled memory storage through `gfdt.NewBuilder()`. Replace it with Redis storage in `initDToken()` when you need persistence.

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

### Route Access Endpoints

- `GET /access/public` - Public endpoint, skips auth
- `GET /access/me` - Requires login only
- `GET /access/articles` - Requires `article:read`
- `GET /access/admin` - Requires `admin`

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

### 4. Route Access Example

```bash
curl http://localhost:8080/access/articles \
  -H "Authorization: Bearer your-access-token-here"
```

## Adding Roles and Permissions

To test role and permission-based endpoints, uncomment the following lines in `handleLogin` function:

The demo seeds `admin` and `article:read` during login so role and permission routes can be called immediately.

## Project Structure

```
examples/gf/
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
s.Use(gfdt.RegisterDTokenContextMiddleware(ctx))

// Authentication middleware
group.Middleware(gfdt.AuthMiddleware(ctx))

// Role middleware
group.Middleware(gfdt.RoleMiddleware(ctx, []string{"admin"}))

// Permission middleware
group.Middleware(gfdt.PermissionMiddleware(ctx, []string{"resource:read"}))
```

### Route Access Rules

```go
group.Middleware(gfdt.AccessMiddleware(ctx,
    gfdt.WithRouteAccessHandler(resolveRouteAccess),
    gfdt.WithFailFunc(handleAuthFail),
))
```

`resolveRouteAccess` can call `SkipAuth()`, `SkipPermission()`, `RequirePermissions(...)`, or `RequireRoles(...)` according to the current URL.

### Annotation-Based Checks

```go
// Check login only
group.GET("/profile", gfdt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))

// Check role
group.GET("/admin-data", gfdt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdminData, handleAuthFail))

// Check permission
group.GET("/sensitive", gfdt.CheckPermissionMiddleware(ctx, []string{"data:read"}, handleSensitiveData, handleAuthFail))
```

## License

This example is part of the DToken-Go project.
