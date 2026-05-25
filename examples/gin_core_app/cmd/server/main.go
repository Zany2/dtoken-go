package main

import (
	"log"
	"time"

	gincoreapp "github.com/Zany2/dtoken-go/examples/gin_core_app"
)

func main() {
	app := gincoreapp.MustNewApp(gincoreapp.Config{
		TokenTimeout:  30 * time.Second,
		ActiveTimeout: -1,
	})
	defer app.Close()

	log.Println("gin core app listening on :8088")
	if err := app.Engine().Run(":8088"); err != nil {
		log.Fatal(err)
	}
}
