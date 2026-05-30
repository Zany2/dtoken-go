module github.com/Zany2/dtoken-go/examples/sso_gin_server

go 1.25.0

require (
	github.com/Zany2/dtoken-go/sso v0.0.0
	github.com/gin-gonic/gin v1.10.0
)

require github.com/Zany2/dtoken-go/core v0.0.0 // indirect

replace github.com/Zany2/dtoken-go/sso => ../../sso

replace github.com/Zany2/dtoken-go/core => ../../core
