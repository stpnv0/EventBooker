package main

import (
	"log"

	"github.com/stpnv0/EventBooker/internal/app"
	"github.com/stpnv0/EventBooker/internal/config"
)

func main() {
	cfg := config.MustLoad()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("app init: %v", err)
	}

	if err = application.Run(); err != nil {
		log.Fatalf("app run: %v", err)
	}
}
