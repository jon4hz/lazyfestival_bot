package main

import (
	"log"
	"os"

	"github.com/jon4hz/lazyfestival_bot/db"
)

func main() {
	bands, err := load()
	if err != nil {
		log.Fatal(err)
	}

	var bandsByDay = make([][]Band, 1)
	currDay := bands[0].Date.Day()
	i := 0
	for _, band := range bands {
		if band.Date.Day() != currDay {
			i++
			currDay = band.Date.Day()
			bandsByDay = append(bandsByDay, make([]Band, 0))
		}
		bandsByDay[i] = append(bandsByDay[i], band)
	}

	db := db.New(os.Getenv("DB_FILE"))
	if err := db.Connect(); err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}

	b, err := NewClient(os.Getenv("BOTTOKEN"), bandsByDay, db, getWebhookOpts())
	if err != nil {
		log.Fatal(err)
	}
	if err := b.Run(); err != nil {
		log.Fatal(err)
	}
}

func getWebhookOpts() *WebhookOpts {
	domain := os.Getenv("WEBHOOK_DOMAIN")
	secret := os.Getenv("WEBHOOK_SECRET")
	path := os.Getenv("WEBHOOK_PATH")
	if path == "" {
		path = "/webhook"
	}
	listenAddr := os.Getenv("WEBHOOK_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "localhost:8080"
	}

	if domain != "" && secret != "" {
		return &WebhookOpts{
			Domain:     domain,
			Secret:     secret,
			Path:       path,
			ListenAddr: listenAddr,
		}
	}
	return nil
}
