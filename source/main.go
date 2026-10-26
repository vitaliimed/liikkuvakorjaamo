package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := loadConfig()
	if err := validateLiikkuva(cfg); err != nil {
		log.Fatal(err)
	}

	renderer, err := newTemplateRenderer()
	if err != nil {
		log.Fatal(err)
	}

	server := newServer(cfg, renderer)

	log.Printf("Starting server on url %s", cfg.SiteURL)
	log.Fatal(server.ListenAndServe())
}
