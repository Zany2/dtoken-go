module github.com/Zany2/dtoken-go/examples/sso_deployment

go 1.25.0

require (
	github.com/Zany2/dtoken-go v0.0.1
	github.com/Zany2/dtoken-go/sso v0.0.1
	github.com/Zany2/dtoken-go/storage/redis v0.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.5.1 // indirect
)

replace github.com/Zany2/dtoken-go => ../..

replace github.com/Zany2/dtoken-go/sso => ../../sso

replace github.com/Zany2/dtoken-go/storage/redis => ../../storage/redis
