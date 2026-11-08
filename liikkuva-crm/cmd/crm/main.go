package main

import (
	"context"
	"liikkuvakorjaamo.fi/landing/crm"
	"log"
	"net/http"
	"os"
	"time"
)

func env(k, f string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return f
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	s, e := crm.Open(ctx, env("GOOGLE_CLOUD_PROJECT", "liikkuva-korjaamo"), env("CRM_DATABASE", "liikkuvakorjaamo"), os.Getenv("CRM_PHOTO_BUCKET"), env("SITE_URL", "http://localhost:8080"))
	if e != nil {
		log.Fatal(e)
	}
	defer s.DB.Close()
	if e = s.Bootstrap(ctx); e != nil {
		log.Fatal(e)
	}
	mux := http.NewServeMux()
	mux.Handle("/crm", s)
	mux.Handle("/crm/", s)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	server := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: mux, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	log.Fatal(server.ListenAndServe())
}
