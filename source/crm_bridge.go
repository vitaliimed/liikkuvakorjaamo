package main

import (
	"context"
	"liikkuvakorjaamo.fi/landing/crm"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

func registerCRM(mux *http.ServeMux, cfg Config) {
	var mu sync.Mutex
	var server *crm.Server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if server == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			url := os.Getenv("CRM_PUBLIC_URL")
			if url == "" {
				url = cfg.SiteURL
			}
			candidate, err := crm.Open(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo", os.Getenv("CRM_PHOTO_BUCKET"), url)
			if err == nil {
				err = candidate.Bootstrap(ctx)
			}
			cancel()
			if err != nil {
				if candidate != nil {
					candidate.DB.Close()
				}
				log.Printf("CRM initialization failed: %v", err)
				mu.Unlock()
				http.Error(w, "CRM temporarily unavailable", 503)
				return
			}
			server = candidate
		}
		current := server
		mu.Unlock()
		current.ServeHTTP(w, r)
	})
	mux.Handle("/crm", handler)
	mux.Handle("/crm/", handler)
}
