package main

import (
	"log"
	"os"
	"time"

	gincoreapp "github.com/Zany2/dtoken-go/tests/gin_core_app"
)

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

// redisURL returns the configured Redis URL or empty for in-memory storage. redisURL 返回配置的 Redis 地址，未配置时使用内存存储。
func redisURL() string {
	if value := os.Getenv("DTOKEN_REDIS_URL"); value != "" {
		return value
	}
	return ""
}
