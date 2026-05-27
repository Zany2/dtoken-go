module github.com/Zany2/dtoken-go/tests/gin_core_app

go 1.25.0

require (
	github.com/Zany2/dtoken-go/core v0.0.0
	github.com/Zany2/dtoken-go/dtoken v0.0.0
	github.com/gin-gonic/gin v1.10.0
)

require (
	github.com/Zany2/dtoken-go/com/codec/json v0.0.0-20260522165342-e2983ea02616 // indirect
	github.com/Zany2/dtoken-go/com/generator/dgenerator v0.0.0-20260522165342-e2983ea02616 // indirect
	github.com/Zany2/dtoken-go/com/log/dlog v0.0.0-20260522165342-e2983ea02616 // indirect
	github.com/Zany2/dtoken-go/com/pool/ants v0.0.0-20260522165342-e2983ea02616 // indirect
	github.com/Zany2/dtoken-go/com/storage/memory v0.0.0-20260522165342-e2983ea02616 // indirect
	github.com/Zany2/dtoken-go/com/storage/redis v0.0.0
	github.com/Zany2/dtoken-go/defaults v0.0.0-20260522165342-e2983ea02616 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/panjf2000/ants/v2 v2.11.3 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/redis/go-redis/v9 v9.5.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/Zany2/dtoken-go/core => ../../core

replace github.com/Zany2/dtoken-go/dtoken => ../../dtoken

replace github.com/Zany2/dtoken-go/defaults => ../../defaults

replace github.com/Zany2/dtoken-go/com/codec/json => ../../com/codec/json

replace github.com/Zany2/dtoken-go/com/generator/dgenerator => ../../com/generator/dgenerator

replace github.com/Zany2/dtoken-go/com/log/dlog => ../../com/log/dlog

replace github.com/Zany2/dtoken-go/com/pool/ants => ../../com/pool/ants

replace github.com/Zany2/dtoken-go/com/storage/memory => ../../com/storage/memory

replace github.com/Zany2/dtoken-go/com/storage/redis => ../../com/storage/redis
