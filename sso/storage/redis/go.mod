module github.com/Zany2/dtoken-go/sso/storage/redis

go 1.25.0

require (
	github.com/Zany2/dtoken-go/com/storage/redis v0.0.0
	github.com/Zany2/dtoken-go/sso v0.0.0
)

require github.com/Zany2/dtoken-go/core v0.0.0 // indirect

replace github.com/Zany2/dtoken-go/com/storage/redis => ../../../com/storage/redis

replace github.com/Zany2/dtoken-go/sso => ../..

replace github.com/Zany2/dtoken-go/core => ../../../core
