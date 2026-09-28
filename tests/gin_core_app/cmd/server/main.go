package main

import (
	"log"
	"os"
	"time"

	gincoreapp "github.com/Zany2/dtoken-go/tests/gin_core_app"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run releases app resources before returning a startup error. run 在返回启动错误前释放应用资源。
func run() error {
	gin.SetMode(gin.ReleaseMode)
	app, err := gincoreapp.NewApp(gincoreapp.Config{
		TokenTimeout:  30 * time.Second,
		ActiveTimeout: -1,
		RedisURL:      redisURL(),
	})
	if err != nil {
		return err
	}
	defer app.Close()

	log.Println("gin core app listening on http://127.0.0.1:8088")
	return app.Engine().Run("127.0.0.1:8088")
}

// redisURL returns the configured Redis URL or empty for in-memory storage. redisURL 返回配置的 Redis 地址，未配置时使用内存存储。
func redisURL() string {
	if value := os.Getenv("DTOKEN_REDIS_URL"); value != "" {
		return value
	}
	return ""
}
