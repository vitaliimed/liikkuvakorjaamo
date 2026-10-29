package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := loadConfig()
	if len(os.Args) == 2 && os.Args[1] == "admin-password" {
		cfg.AdminStore = firestoreAdminAuthStore{cfg: cfg, operatorToken: os.Getenv("LIIKKUVA_OPERATOR_TOKEN")}
		if err := resetAdminPasswordCLI(cfg, os.Stdin); err != nil {
			log.Fatal(err)
		}
		log.Print("Пароль сохранён. Все предыдущие сеансы завершены.")
		return
	}
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
