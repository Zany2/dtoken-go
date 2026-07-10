package main

import (
	"log"
	"os"
	"time"

	gincoreapp "github.com/Zany2/dtoken-go/tests/gin_core_app"
)

// defaultRedisURL stores the fallback Redis endpoint for the demo server. defaultRedisURL 保存示例服务端的备用 Redis 地址。
const defaultRedisURL = "redis://:root@192.168.19.104:6379/0"

func main() {
	app := gincoreapp.MustNewApp(gincoreapp.Config{
		TokenTimeout:  30 * time.Second,
		ActiveTimeout: -1,
		RedisURL:      redisURL(),
	})
	defer app.Close()

	log.Println("gin core app listening on :8088")
	if err := app.Engine().Run(":8088"); err != nil {
		log.Fatal(err)
	}
}

// redisURL returns the configured Redis URL or its fallback. redisURL 返回配置的 Redis 地址或备用地址。
func redisURL() string {
	if value := os.Getenv("DTOKEN_REDIS_URL"); value != "" {
		return value
	}
	return defaultRedisURL
}
